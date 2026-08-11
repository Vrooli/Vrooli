// Package vision provides the VisionNavigator interface and related types for AI-driven
// browser navigation using different backend implementations.
package vision

import (
	"context"
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

// NavigationSession tracks the state of an active navigation.
// This is used internally by navigators to track progress.
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
}

// HumanInterventionInfo contains details about human intervention.
type HumanInterventionInfo struct {
	Reason           string `json:"reason"`
	Instructions     string `json:"instructions,omitempty"`
	InterventionType string `json:"interventionType"`
	Trigger          string `json:"trigger"` // "programmatic" or "ai_requested"
}
