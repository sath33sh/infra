package wapi

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/nbio/httpcontext"
	"github.com/sath33sh/infra/log"
	"github.com/sath33sh/infra/push"
	"github.com/sath33sh/infra/util"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

const (
	WS = "ws"

	// Time allowed to write a message to client.
	WriteWait = 10 * time.Second

	// Send pings to client with this interval.
	PingInterval = 20 * time.Second

	// Wait for ping timeout before closing connection.
	PingTimeout = 3 * PingInterval

	// Command response timeout.
	ResponseTimeout = 5 * time.Second

	// Maximum message size allowed.
	MaxMessageSize = 32 * 1024
)

// Websocket upgrader.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  2 * MaxMessageSize,
	WriteBufferSize: 2 * MaxMessageSize,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Websocket message envelope.
type Envelope struct {
	Rid       string          `json:"rid,omitempty"`   // Resource identifier.
	Timestamp int64           `json:"timestamp"`       // UTC timestamp in milliseconds.
	Method    string          `json:"method"`          // Method: "GET", "POST" or "PUSH".
	Uri       string          `json:"uri"`             // URI endpoint.
	Push      bool            `json:"push"`            // Message pushed from server.
	Data      json.RawMessage `json:"data,omitempty"`  // Data.
	Error     json.RawMessage `json:"error,omitempty"` // Error.
}

// Websocket connection.
type Conn struct {
	ws        *websocket.Conn // Websocket connection.
	envelope  Envelope        // Message envelope.
	LogPrefix string          // Log prefix.
}

func (c *Conn) Errorf(format string, v ...interface{}) {
	log.ErrorfOutput(3, c.LogPrefix+format, v...)
}

func (c *Conn) Debugf(format string, v ...interface{}) {
	log.DebugfOutput(3, MODULE, c.LogPrefix+format, v...)
}

// Get JSON data from envelope.
func (c *Conn) wsGetData(v interface{}) error {
	return json.Unmarshal(c.envelope.Data, v)
}

// Return success.
func (c *Conn) wsReturnOk(v interface{}) {
	var err error

	// Encode data.
	if c.envelope.Data, err = json.Marshal(v); err != nil {
		c.Errorf("JSON data encode failed: %s", err)
		c.envelope.Data = nil
		c.envelope.Error, _ = util.ErrInternal.MarshalJSON()
	} else {
		c.envelope.Error = nil
	}

	// Set timestamp.
	c.envelope.Timestamp = util.NowMilli()

	// Write response.
	c.ws.SetWriteDeadline(time.Now().Add(WriteWait))
	if err = c.ws.WriteJSON(&c.envelope); err != nil {
		c.Errorf("OK: write envelope error: %s", err)
		return
	}

	return
}

// Return error.
func (c *Conn) wsReturnError(err error) {
	c.envelope.Error, _ = err.(util.Err).MarshalJSON()
	c.envelope.Data = nil

	// Set timestamp.
	c.envelope.Timestamp = util.NowMilli()

	// Write response.
	c.ws.SetWriteDeadline(time.Now().Add(WriteWait))
	if err = c.ws.WriteJSON(&c.envelope); err != nil {
		c.Errorf("Error: write envelope error: %s", err)
		return
	}
}

func (c *Conn) apiLoop(w http.ResponseWriter, r *http.Request) {
	var err error

	defer func() {
		httpcontext.Clear(r)
		c.ws.Close()
	}()

	// Configure websocket connection.
	c.ws.SetReadLimit(MaxMessageSize)
	c.ws.SetPongHandler(func(string) error {
		//c.Debugf("Pong")
		c.ws.SetReadDeadline(time.Now().Add(PingTimeout))
		return nil
	})

	for {
		// Read API request from client.
		c.envelope.Data = nil
		c.ws.SetReadDeadline(time.Now().Add(PingTimeout))
		if err := c.ws.ReadJSON(&c.envelope); err != nil {
			if err == io.EOF {
				// Connection closed.
				break
			}

			if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
				// Read timed out. Client is not responding to ping.
				// Close the connection and move on.
				c.Debugf("Read envelope timed out: %s", err)
				break
			}

			// Read error, possibly due to wrong JSON format.
			// c.Errorf("Read envelope error: %s", err)
			c.wsReturnError(util.ErrJsonDecode)
			break
		}

		c.Debugf("Method %s, URI %s, Data %s", c.envelope.Method, c.envelope.Uri, string(c.envelope.Data))

		if r.URL, err = url.ParseRequestURI(c.envelope.Uri); err != nil {
			c.Errorf("Invalid URI %s: %v", c.envelope.Uri, err)
			c.wsReturnError(util.ErrInvalidMethod)
			continue
		}

		if handler, params, _ := router.mux.Lookup(c.envelope.Method, r.URL.Path); handler != nil {
			handler(w, r, params)
		} else {
			c.Errorf("Handler not found: %s %s", c.envelope.Method, r.URL.Path)
			c.wsReturnError(util.ErrInvalidMethod)
		}
	}
}

func (c *Conn) pushLoop(userId, sessionId string) {
	var err error
	pe := Envelope{
		Push: true,
	}

	// Open push session.
	duct := push.OpenSession(userId, sessionId, true)

	// Create ticker for sending ping messages.
	ticker := time.NewTicker(PingInterval)

	defer func() {
		ticker.Stop()
		push.CloseSession(userId, sessionId, duct)
		c.ws.Close()
	}()

	for {
		select {
		case payload := <-duct:
			if payload == nil {
				continue
			}

			c.Debugf("Kind %s, Op %s, URI %s, Data %s", payload.Kind, payload.Op, payload.Uri, string(payload.Data))

			// Copy payload content.
			pe.Rid = payload.Kind
			pe.Method = string(payload.Op)
			pe.Uri = payload.Uri
			pe.Data = payload.Data

			// Set timestamp.
			pe.Timestamp = util.NowMilli()

			// Push.
			c.ws.SetWriteDeadline(time.Now().Add(WriteWait))
			if err = c.ws.WriteJSON(&pe); err != nil {
				if err == io.EOF {
					// Connection closed.
					return
				}
				c.Errorf("Push: write envelope error: %v", err)
				return
			}

		case <-ticker.C:
			//c.Debugf("Ping")
			c.ws.SetWriteDeadline(time.Now().Add(WriteWait))
			if err = c.ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				if err == io.EOF {
					// Connection closed.
					return
				}
				c.Errorf("Ping send error: %s", err)
				return
			}
		}
	}
}

func NewConn(w http.ResponseWriter, r *http.Request, logPrefix string) (c *Conn, err error) {
	c = &Conn{LogPrefix: logPrefix}

	// Upgrade to websocket.
	c.ws, err = upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorln("Websocket upgrade error:", err)
		return c, util.ErrInternal
	}

	// Save context in request.
	httpcontext.Set(r, WS, c)

	return c, nil
}

func (c *Conn) StartLoop(w http.ResponseWriter, r *http.Request, userId, sessionId string) {
	// Start the websocket loop.
	go c.pushLoop(userId, sessionId)
	c.apiLoop(w, r)
}

func init() {
	log.EnableDebug(MODULE)
}
