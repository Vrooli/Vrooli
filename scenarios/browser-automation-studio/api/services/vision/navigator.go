// Package vision provides the VisionNavigator interface and related types for AI-driven
// browser navigation using different backend implementations.
package vision

import (
	"context"
	"fmt"
	"time"
)

// NavigatorType identifies the type of vision navigator.
type NavigatorType string

const (
	// NavigatorPlaywright uses playwright-driver for vision navigation.
	NavigatorPlaywright NavigatorType = "playwright"

	// NavigatorClaudeCode uses Claude Code CLI with Chrome for navigation.
	NavigatorClaudeCode NavigatorType = "claude_code"
)

// VisionNavigator defines the interface for AI-driven browser navigation backends.
// Each implementation declares its own credit policy and client source restrictions.
type VisionNavigator interface {
	// Navigate starts an AI navigation session for the given request.
	// Returns a NavigationHandle for tracking the navigation progress.
	Navigate(ctx context.Context, req NavigationRequest) (NavigationHandle, error)

	// CreditPolicy returns the credit charging policy for this navigator.
	CreditPolicy() CreditPolicy

	// ClientSourcePolicy returns the client source restrictions for this navigator.
	ClientSourcePolicy() ClientSourcePolicy

	// Type returns the navigator type identifier.
	Type() NavigatorType

	// IsAvailable checks if this navigator is currently available for use.
	// For example, ClaudeCode navigator requires the claude CLI to be installed.
	IsAvailable(ctx context.Context) bool

	// Description returns a human-readable description of the navigator.
	Description() string

	// UnavailableReason returns the reason why the navigator is not available,
	// or empty string if available.
	UnavailableReason(ctx context.Context) string
}

// NavigationHandle represents an active navigation session.
// It provides methods to control and query the navigation state.
type NavigationHandle interface {
	// ID returns the unique navigation session ID.
	ID() string

	// SessionID returns the browser session ID being navigated.
	SessionID() string

	// Status returns the current navigation status.
	Status() NavigationStatus

	// Wait blocks until the navigation completes or the context is cancelled.
	Wait(ctx context.Context) error

	// Abort requests the navigation to stop.
	Abort(ctx context.Context) error

	// Resume resumes navigation after human intervention.
	Resume(ctx context.Context) error
}

// NavigationStatus represents the current state of a navigation session.
type NavigationStatus string

const (
	StatusIdle          NavigationStatus = "idle"
	StatusNavigating    NavigationStatus = "navigating"
	StatusAwaitingHuman NavigationStatus = "awaiting_human"
	StatusCompleted     NavigationStatus = "completed"
	StatusFailed        NavigationStatus = "failed"
	StatusAborted       NavigationStatus = "aborted"
	StatusMaxSteps      NavigationStatus = "max_steps_reached"
	StatusLoopDetected  NavigationStatus = "loop_detected"
)

// Terminal reports whether a status ends (or pauses) the navigation from a
// waiter's point of view. awaiting_human counts as terminal because the
// navigator will not make progress until a human acts, so a caller waiting
// on the session must be woken up to see it.
func (s NavigationStatus) Terminal() bool {
	switch s {
	case StatusCompleted, StatusFailed, StatusAborted, StatusMaxSteps, StatusLoopDetected, StatusAwaitingHuman:
		return true
	}
	return false
}

// MaxRecordedSteps bounds the per-session step history kept by trackers.
// Older entries are dropped once the cap is reached.
const MaxRecordedSteps = 200

// NavigationStepRecord is one recorded navigator action, kept on the
// NavigationSession so a finished navigation can be replayed or turned into
// a saved workflow.
type NavigationStepRecord struct {
	Index       int
	ActionType  string
	Selector    string
	Value       string
	URL         string
	Description string
	Success     bool
	Error       string
	At          time.Time
}

// NavigationSession tracks the state of an active navigation.
// This is used internally by navigators to track progress.
//
// Mutating methods (SetStatus, RecordStep) and Snapshot must be called with
// the owning navigator's lock held; navigators hand out Snapshot copies to
// readers so the live struct is never read without that lock.
type NavigationSession struct {
	NavigationID         string
	SessionID            string
	UserID               string
	Model                string
	StartedAt            time.Time
	StepCount            int
	TotalTokens          int
	Status               NavigationStatus
	AwaitingHuman        bool
	HumanIntervention    *HumanInterventionInfo
	CredentialProvenance CredentialProvenance
	NavigatorType        NavigatorType
	// Steps is the bounded action history (MaxRecordedSteps most recent).
	Steps []NavigationStepRecord

	// changed is closed and replaced on every status transition so waiters
	// can block on a broadcast instead of polling. Created lazily.
	changed chan struct{}
}

// SetStatus records a status transition and wakes every waiter blocked on
// the previous Changed() channel. Caller holds the owning lock.
func (s *NavigationSession) SetStatus(status NavigationStatus) {
	s.Status = status
	if s.changed != nil {
		close(s.changed)
	}
	s.changed = make(chan struct{})
}

// Changed returns a channel that is closed on the next status transition.
// Read it from a Snapshot: the snapshot carries the channel that was current
// when it was taken, so a transition after the snapshot always wakes the
// waiter. Caller holds the owning lock when called on the live session.
func (s *NavigationSession) Changed() <-chan struct{} {
	if s.changed == nil {
		s.changed = make(chan struct{})
	}
	return s.changed
}

// RecordStep appends one action to the bounded history. When Index is zero it
// is assigned the next 1-based position; when At is zero it is set to now.
// Caller holds the owning lock.
func (s *NavigationSession) RecordStep(rec NavigationStepRecord) {
	if rec.Index == 0 {
		rec.Index = len(s.Steps) + 1
	}
	if rec.At.IsZero() {
		rec.At = time.Now()
	}
	if len(s.Steps) >= MaxRecordedSteps {
		s.Steps = append(s.Steps[:0], s.Steps[1:]...)
	}
	s.Steps = append(s.Steps, rec)
}

// Snapshot returns a copy safe to read without the owning lock. The copy
// shares the current Changed() channel so a waiter can block on it.
// Caller holds the owning lock.
func (s *NavigationSession) Snapshot() *NavigationSession {
	s.Changed() // ensure the channel exists so the copy can be waited on
	cp := *s
	if s.Steps != nil {
		cp.Steps = make([]NavigationStepRecord, len(s.Steps))
		copy(cp.Steps, s.Steps)
	}
	if s.HumanIntervention != nil {
		hi := *s.HumanIntervention
		cp.HumanIntervention = &hi
	}
	return &cp
}

// SessionTracker is the seam navigators expose for session-state lookups.
// GetSession returns a Snapshot, never the live struct.
type SessionTracker interface {
	GetSession(navigationID string) (*NavigationSession, bool)
	AbortNavigation(ctx context.Context, navigationID string) error
	ResumeNavigation(ctx context.Context, navigationID string) error
}

// MultiTracker fans session lookups out over several navigators so one
// status/abort/resume surface covers every navigator type. The first tracker
// that knows the navigation ID wins.
type MultiTracker []SessionTracker

// GetSession returns the first tracker's snapshot for navigationID.
func (m MultiTracker) GetSession(navigationID string) (*NavigationSession, bool) {
	for _, t := range m {
		if t == nil {
			continue
		}
		if s, ok := t.GetSession(navigationID); ok {
			return s, true
		}
	}
	return nil, false
}

// AbortNavigation aborts on the tracker that owns navigationID.
func (m MultiTracker) AbortNavigation(ctx context.Context, navigationID string) error {
	if t := m.owner(navigationID); t != nil {
		return t.AbortNavigation(ctx, navigationID)
	}
	return fmt.Errorf("navigation not found: %s", navigationID)
}

// ResumeNavigation resumes on the tracker that owns navigationID.
func (m MultiTracker) ResumeNavigation(ctx context.Context, navigationID string) error {
	if t := m.owner(navigationID); t != nil {
		return t.ResumeNavigation(ctx, navigationID)
	}
	return fmt.Errorf("navigation not found: %s", navigationID)
}

func (m MultiTracker) owner(navigationID string) SessionTracker {
	for _, t := range m {
		if t == nil {
			continue
		}
		if _, ok := t.GetSession(navigationID); ok {
			return t
		}
	}
	return nil
}

// HumanInterventionInfo contains details about human intervention.
type HumanInterventionInfo struct {
	Reason           string `json:"reason"`
	Instructions     string `json:"instructions,omitempty"`
	InterventionType string `json:"interventionType"`
	Trigger          string `json:"trigger"` // "programmatic" or "ai_requested"
}
