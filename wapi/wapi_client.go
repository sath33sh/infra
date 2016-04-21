// Web API.
package wapi

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/sath33sh/infra/util"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// Connection error handler.
type ConnErrorHandler func(c *Client, err error)

// Client context.
type Client struct {
	ws           *websocket.Conn  // Websocket connection.
	envelope     Envelope         // Message envelope.
	readLoopSync chan Envelope    // Read loop synchronizer.
	connErrorCb  ConnErrorHandler // Connection error handler.
	debug        bool             // Enable debug.
}

// Global variables.
var (
	httpUrl string // HTTP server URL.
	wsUrl   string // Websocket server URL.
	secure  bool   // Server connection is secured.
)

func validateHost(host string) (string, error) {
	if host == "" {
		// Read host from env.
		host = os.Getenv("WAPI_HOST")
	}

	if host == "" {
		return host, util.ErrInvalidInput
	}

	// By default security is enabled unless ${WAPI_SECURE} is set to "false".
	if strings.EqualFold(os.Getenv("WAPI_SECURE"), "false") {
		secure = false
	} else {
		secure = true
	}

	return host, nil
}

func GetHttpUrl(host string) (string, error) {
	if httpUrl == "" {
		host, err := validateHost(host)
		if err != nil {
			return httpUrl, err
		}

		if secure {
			httpUrl = fmt.Sprintf("https://%s", host)
		} else {
			httpUrl = fmt.Sprintf("http://%s", host)
		}
	}

	return httpUrl, nil
}

func GetWebsocketUrl(host string) (string, error) {
	if wsUrl == "" {
		host, err := validateHost(host)
		if err != nil {
			return wsUrl, err
		}

		if secure {
			wsUrl = fmt.Sprintf("wss://%s/ws", host)
		} else {
			wsUrl = fmt.Sprintf("ws://%s/ws", host)
		}
	}

	return wsUrl, nil
}

func NopOnConnError(c *Client, err error) {
	c.Debugf("NOP: %v\n", err)
}

func ExitOnConnError(c *Client, err error) {
	c.Debugf("Exit: %v\n", err)
	os.Exit(-4)
}

var wsDialer = websocket.Dialer{
	ReadBufferSize:  2 * MaxMessageSize,
	WriteBufferSize: 2 * MaxMessageSize,
}

var wsTlsDialer = websocket.Dialer{
	ReadBufferSize:  2 * MaxMessageSize,
	WriteBufferSize: 2 * MaxMessageSize,
	TLSClientConfig: &tls.Config{
		InsecureSkipVerify: true,
	},
}

func NewClient(host, userId, sessionId, accessToken string,
	once, debug bool,
	connErrorCb ConnErrorHandler) (*Client, error) {

	c := &Client{debug: debug}
	var err error

	// Construct header.
	hdr := http.Header{
		"X-UserId":                 {userId},
		"X-SessionId":              {sessionId},
		"X-AccessToken":            {accessToken},
		"Sec-WebSocket-Extensions": {"permessage-deflate; client_max_window_bits, x-webkit-deflate-frame"},
	}

	// Construct websocket url.
	url, err := GetWebsocketUrl(host)
	if err != nil {
		return c, err
	}

	// Connect to server.
	if secure {
		c.ws, _, err = wsTlsDialer.Dial(url, hdr)
		if err != nil {
			return c, err
		}
	} else {
		c.ws, _, err = wsDialer.Dial(url, hdr)
		if err != nil {
			return c, err
		}
	}

	// Create sync channel.
	c.readLoopSync = make(chan Envelope)

	// Save handlers.
	c.connErrorCb = connErrorCb

	// Start read loop.
	go c.readLoop(once)

	return c, err
}

func (c *Client) Debugf(format string, v ...interface{}) {
	if c.debug {
		fmt.Printf(format+"\n", v...)
	}
}

func (c *Client) Close() {
	c.Debugf("Closing connection")
	c.ws.Close()
	close(c.readLoopSync)
}

func (c *Client) readLoop(once bool) {
	var resp Envelope

	defer func() {
		c.ws.Close()
		close(c.readLoopSync)
	}()

	// Set message size limit.
	c.ws.SetReadLimit(MaxMessageSize)

	// Set read deadline to ping timeout interval.
	c.ws.SetReadDeadline(time.Now().Add(PingTimeout))

	// Set ping handler for refreshing read deadline.
	c.ws.SetPingHandler(func(string) error {
		// fmt.Printf("Ping\n")
		c.ws.SetWriteDeadline(time.Now().Add(WriteWait))
		if err := c.ws.WriteMessage(websocket.PongMessage, []byte{}); err != nil {
			if err == io.EOF {
				// Connection closed.
				return err
			}
			c.Debugf("Pong send error: %v\n", err)
			return err
		}

		// Reset read deadline.
		c.ws.SetReadDeadline(time.Now().Add(PingTimeout))
		return nil
	})

	for {
		// Reset response envelope.
		resp.Data = nil
		resp.Error = nil
		resp.Rid = ""
		resp.Method = ""

		// Read from server.
		if err := c.ws.ReadJSON(&resp); err != nil {
			if err == io.EOF {
				// Connection closed.
				return
			}

			if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
				// Read timed out. Server is not responding.
				// Close the connection and move on.
				fmt.Printf("Connection timed out\n")
				c.connErrorCb(c, util.ErrNetAccess)
				return
			}

			// Read error.
			fmt.Printf("Read error: %v\n", err)
			c.connErrorCb(c, util.ErrNetAccess)
			return
		}

		if resp.Push {
			// Received a push message. Not a response.
			fmt.Printf("PUSH: Rid %s, Uri %s\n", resp.Rid, resp.Uri)
			continue
		} else {
			// Received a response.
			c.readLoopSync <- resp
		}

		if once {
			return
		}
	}
}

func (c *Client) RestExec(rid, method, uri string, reqData, respData, respErr interface{}) (err error) {
	req := Envelope{
		Rid:       rid,
		Timestamp: util.NowMilli(),
		Method:    strings.ToUpper(method),
		Uri:       uri,
		Data:      nil,
		Error:     nil,
	}

	// Marshal request data.
	if reqData != nil {
		if req.Data, err = json.Marshal(reqData); err != nil {
			fmt.Printf("Request JSON marshal error: %v\n", err)
			return util.ErrInvalidInput
		}
	}

	c.Debugf("RID: %s", req.Rid)
	c.Debugf("Method: %s", req.Method)
	c.Debugf("URI: %s", req.Uri)
	c.Debugf("Data: %s", req.Data)

	// Send request.
	c.ws.SetWriteDeadline(time.Now().Add(WriteWait))
	if err := c.ws.WriteJSON(&req); err != nil {
		fmt.Printf("Request write error: %s\n", err)
		return util.ErrNetAccess
	}

	// Timeout for response.
	wait := time.NewTicker(ResponseTimeout * time.Second)
	defer func() {
		wait.Stop()
	}()

	// Wait for response.
	select {
	case resp, ok := <-c.readLoopSync:
		if ok {
			if len(resp.Method) == 0 {
				// websocketReadloop() encountered an error.
				c.Debugf("Read loop error")
				return util.ErrNetAccess
			}

			if resp.Error != nil {
				c.Debugf("ERROR response from server")
				if respErr != nil {
					json.Unmarshal(resp.Error, respErr)
				}
				return util.ErrInternal
			} else {
				c.Debugf("OK response from server")
			}

			if req.Rid != resp.Rid {
				fmt.Printf("Response does not match: %s, %s\n", resp.Method, resp.Rid)
				return util.ErrNotFound
			}

			if respData != nil {
				if err = json.Unmarshal(resp.Data, respData); err != nil {
					fmt.Printf("Response JSON marshal error: %v\n", err)
					return util.ErrJsonDecode
				}
			}

			return nil
		} else {
			c.Debugf("Error in synchronizing")
			return util.ErrNetAccess
		}

	case <-wait.C:
		fmt.Printf("Response timed out [%d]\n", ResponseTimeout)
		return util.ErrTimeout
	}
}
