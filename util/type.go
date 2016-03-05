// Utility package.
package util

import (
	"time"
)

// Get current unix time in milliseconds.
func NowMilli() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// General purpose text object.
type Text struct {
	Text string `json:"text,omitempty"` // Text string.
}

// Media type, aka MIME type.
// Examples: image/png, video/mp4.
type MediaType string

// Media size type.
type MediaSizeType string

const (
	FULL    MediaSizeType = "full" // Orignal size.
	AR_1x1                = "1:1"  // Aspect ratio 1:1.
	AR_2x1                = "2:1"  // Aspect ratio 2:1.
	AR_16x9               = "16:9" // Aspect ratio 16:9.
)

// Media size.
type MediaSize struct {
	X int `json:"x,omitempty"` // X position (applies only to cropped media).
	Y int `json:"y,omitempty"` // Y position (applies only to cropped media).
	W int `json:"w,omitempty"` // Width.
	H int `json:"h,omitempty"` // Height.
}

// Color codes.
const (
	COLOR_BG_LIGHT  = "#d0d0d0" // Light background.
	COLOR_BG_DARK   = "#192d3c" // Dark background.
	COLOR_PRIMARY   = "#16bde6" // Blue.
	COLOR_SECONDARY = "#f47681" // Red.
)
