// internal/models/login.go
package models

import (
	"strconv"
	"time"
)

// LoginEvent represents the raw data we receive from Kafka
type LoginEvent struct {
	UserID     string `json:"user_id"`
	AppVersion string `json:"app_version"`
	DeviceType string `json:"device_type"`
	IP         string `json:"ip"`
	Locale     string `json:"locale"`
	DeviceID   string `json:"device_id"`
	Timestamp  string `json:"timestamp"`
}

// ParsedLogin represents the processed login data we'll store
type ParsedLogin struct {
	LoginEvent  // Embed the original event
	ParsedTime  time.Time
	ProcessedAt time.Time
}

// NewParsedLogin creates a new ParsedLogin from a LoginEvent
func NewParsedLogin(event LoginEvent) (*ParsedLogin, error) {
	ts, err := strconv.ParseInt(event.Timestamp, 10, 64)
	if err != nil {
		return nil, err
	}

	return &ParsedLogin{
		LoginEvent:  event,
		ParsedTime:  time.Unix(ts, 0),
		ProcessedAt: time.Now().UTC(),
	}, nil
}
