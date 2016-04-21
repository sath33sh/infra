package wapi

import (
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"github.com/nbio/httpcontext"
	"github.com/sath33sh/infra/log"
	"net/http"
	"strconv"
)

const MODULE = "wapi"

type Router struct {
	mux *httprouter.Router
}

var (
	router Router
)

// Aliases.
type Handler httprouter.Handle
type Param httprouter.Param
type Params httprouter.Params

func GET(path string, h Handler) {
	router.mux.GET(path, httprouter.Handle(h))
}

func POST(path string, h Handler) {
	router.mux.POST(path, httprouter.Handle(h))
}

func DELETE(path string, h Handler) {
	router.mux.DELETE(path, httprouter.Handle(h))
}

func ServeFiles(path, root string) {
	router.mux.ServeFiles(path, http.Dir(root))
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if origin := req.Header.Get("Origin"); origin != "" {
		// log.Debugf(MODULE, "Origin %s: %s", origin, req.URL)

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers",
			"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-UserId, X-AccessToken, X-SessionId, X-AppVersion")
	}

	if req.Method == "OPTIONS" {
		// Preflighted OPTIONS request. Return without invoking API.
		return
	}

	r.mux.ServeHTTP(w, req)
}

// Get JSON data from request.
func DecodeJSON(r *http.Request, v interface{}) error {
	if c, ok := httpcontext.GetOk(r, WS); ok {
		// Websocket request.
		return c.(*Conn).wsGetData(v)
	} else {
		// REST request.
		return json.NewDecoder(r.Body).Decode(v)
	}
}

// Return success.
func ReturnOk(w http.ResponseWriter, r *http.Request, v interface{}) {
	if c, ok := httpcontext.GetOk(r, WS); ok {
		// Websocket request.
		c.(*Conn).wsReturnOk(v)
	} else {
		// REST request.
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(v)
	}
}

// Return error.
func ReturnError(w http.ResponseWriter, r *http.Request, err error) {
	if c, ok := httpcontext.GetOk(r, WS); ok {
		// Websocket request.
		c.(*Conn).wsReturnError(err)
	} else {
		// REST request.
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]error{"error": err})
	}
}

func Ping(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	ReturnOk(w, r, "pong")
}

func init() {
	// Create HTTP mux for REST APIs.
	router.mux = httprouter.New()
}

func runPing(port int) {
	// Create a separate router for ping.
	pingRouter := httprouter.New()

	pingRouter.GET("/ping", httprouter.Handle(Ping))

	// Listen and serve ping.
	err := http.ListenAndServe(":"+strconv.Itoa(port), pingRouter)
	if err != nil {
		log.Fatalf("HTTP serve failed for ping: %v", err)
	}
}

func StartServer(port int, secure bool, certFile, keyFile string) {
	var err error

	if secure {
		// GCE health check does not support HTTPS.
		// As a workaround, start a separate ping service on the next port.
		go runPing(port + 1)

		// Start HTTP service in TLS mode.
		err = http.ListenAndServeTLS(":"+strconv.Itoa(port), certFile, keyFile, &router)
		if err != nil {
			log.Fatalf("HTTP TLS serve failed: %v", err)
		}
	} else {
		log.Infof("Port %d is not secure", port)

		// Register ping handler.
		GET("/ping", Ping)

		// Start HTTP service in unencrypted mode.
		err = http.ListenAndServe(":"+strconv.Itoa(port), &router)
		if err != nil {
			log.Fatalf("HTTP serve failed: %v", err)
		}
	}
}
