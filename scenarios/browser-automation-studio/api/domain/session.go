// Package domain provides core domain types for browser-automation-studio.
// DOC: docs/architecture/recording.md#recording-session
package domain

import (
	"errors"
	"fmt"
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

var ErrInvalidSessionStatusTransition = errors.New("invalid recording session status transition")

// Valid reports whether status is part of the persisted session lifecycle.
func (status SessionStatus) Valid() bool {
	return status == SessionStatusActive || status == SessionStatusClosed
}

// CanTransitionTo reports whether the persisted lifecycle permits next.
func (status SessionStatus) CanTransitionTo(next SessionStatus) bool {
	if status == next {
		return status.Valid()
	}
	return status == SessionStatusActive && next == SessionStatusClosed
}

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

// TransitionStatus is the only domain owner of recording-session state changes.
func (s *RecordingSession) TransitionStatus(next SessionStatus) error {
	if s == nil || !s.Status.Valid() || !next.Valid() || !s.Status.CanTransitionTo(next) {
		current := SessionStatus("<nil>")
		if s != nil {
			current = s.Status
		}
		return fmt.Errorf("%w: %q -> %q", ErrInvalidSessionStatusTransition, current, next)
	}
	s.Status = next
	return nil
}

// Close records the terminal status and timestamp through the lifecycle owner.
func (s *RecordingSession) Close(closedAt time.Time) error {
	if err := s.TransitionStatus(SessionStatusClosed); err != nil {
		return err
	}
	s.ClosedAt = &closedAt
	return nil
}
