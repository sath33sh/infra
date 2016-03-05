package push

import (
	"encoding/json"
	"github.com/sath33sh/infra/log"
)

// Module name.
const MODULE = "push"

// Push operations.
type Op string

const (
	UPSERT Op = "UPSERT"
	REMOVE    = "REMOVE"
)

// Push payload.
type Payload struct {
	Kind string          `json:"kind,omitempty"` // Kind (aka type) of payload.
	Op   Op              `json:"op:omitempty"`   // Operation.
	Uri  string          `json:"uri,omitempty"`  // Push topic URI.
	Data json.RawMessage `json:"data,omitempty"` // Data.
}

// Pushable interface. Structs that can be pushed should implement this interface.
type Pushable interface {
	BuildPushPayload() (*Payload, error)
}

// Variables.
var (
	CasMode       = false
	DisableBroker = false
)

func Init(casMode bool) {
	// Debug enable.
	log.EnableDebug(MODULE)

	// Set CAS mode.
	CasMode = casMode

	// CAS mode specific initialization.
	if CasMode {
		// Start topic manager.
		startTopicMgr()

		// Start session manager.
		startSessionMgr()
	}

	// Initialize NATS push broker.
	if err := initNats(); err != nil {
		log.Fatalf("Failed to initialize push broker: %v", err)
		return
	}
}
