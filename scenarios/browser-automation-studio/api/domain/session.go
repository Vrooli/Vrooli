// Package domain provides core domain types for browser-automation-studio.
// DOC: docs/architecture/recording.md#recording-session
package domain

import (
	"time"
)

// SessionStatus represents the lifecycle state of a recording session.
// DOC: docs/architecture/recording.md#session-status
type SessionStatus string

const (
	// SessionStatusActive indicates the session is currently recording.
	SessionStatusActive SessionStatus = "active"
	// SessionStatusClosed indicates the session has been closed.
	SessionStatusClosed SessionStatus = "closed"
)

// RecordingSession represents a browser recording session.
// This is the aggregate root for the recording bounded context.
// Each session contains multiple RecordingAction entities.
//
// DOC: docs/architecture/recording.md#recording-session
type RecordingSession struct {
	// ID uniquely identifies this session.
	ID string `json:"id"`

	// ProfileID optionally links this session to a SessionProfile for state restoration.
	ProfileID string `json:"profileId,omitempty"`

	// Status indicates whether the session is active or closed.
	Status SessionStatus `json:"status"`

	// ViewportWidth is the browser viewport width in pixels.
	ViewportWidth int `json:"viewportWidth,omitempty"`

	// ViewportHeight is the browser viewport height in pixels.
	ViewportHeight int `json:"viewportHeight,omitempty"`

	// CreatedAt is when the session was created.
	CreatedAt time.Time `json:"createdAt"`

	// ClosedAt is when the session was closed (nil if still active).
	ClosedAt *time.Time `json:"closedAt,omitempty"`

	// ActionCount is the number of actions recorded in this session.
	// This is computed on read, not stored in the database.
	ActionCount int `json:"actionCount"`
}

// IsActive returns true if the session is currently active.
func (s *RecordingSession) IsActive() bool {
	return s.Status == SessionStatusActive
}

// Close marks the session as closed with the given timestamp.
func (s *RecordingSession) Close(closedAt time.Time) {
	s.Status = SessionStatusClosed
	s.ClosedAt = &closedAt
}
