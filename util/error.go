package util

import (
	"encoding/json"
)

// Error type.
type Err int

// Error codes.
// *** ATTENTION ***
// Always add new error codes at the bottom of the list.
// Do NOT forget to add corresponding error messages.
const (
	ErrInvalidInput Err = iota
	ErrInvalidToken
	ErrInvalidMethod
	ErrInvalidSession
	ErrInvalidOp
	ErrInvalidPerm
	ErrJsonDecode
	ErrXmlDecode
	ErrNotFound
	ErrInternal
	ErrFileAccess
	ErrNetAccess
	ErrDbAccess
	ErrInvalidObject
	ErrTimeout
	ErrResourceLimit
	ErrRateLimit
)

// Error messages.
var messages = map[Err]string{
	ErrInvalidInput:   "Invalid input",
	ErrInvalidToken:   "Invalid access token",
	ErrInvalidMethod:  "Invalid method",
	ErrInvalidSession: "Invalid session",
	ErrInvalidOp:      "Invalid operation",
	ErrInvalidPerm:    "Insufficient permission",
	ErrJsonDecode:     "JSON decode error",
	ErrXmlDecode:      "XML decode error",
	ErrNotFound:       "Object not found",
	ErrInternal:       "Internal error",
	ErrFileAccess:     "File I/O error",
	ErrNetAccess:      "Network access error",
	ErrDbAccess:       "Database access error",
	ErrInvalidObject:  "Invalid object",
	ErrTimeout:        "Operation timed out",
	ErrResourceLimit:  "Resource limit exceeded",
	ErrRateLimit:      "Rate limit exceeded",
}

// Stringer.
func (e Err) Error() string {
	return messages[e]
}

// JSON marshaler.
func (e Err) MarshalJSON() ([]byte, error) {
	return json.Marshal(ErrJson{Code: int(e), Message: messages[e]})
}

// Error in JSON format.
type ErrJson struct {
	Code    int    `json:"code"`    // Error code.
	Message string `json:"message"` // Error message.
}
