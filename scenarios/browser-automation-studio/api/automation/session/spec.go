// Package session provides unified session management for browser automation.
// It supports recording mode, execution mode, and hybrid mode sessions.
package session

import (
	"encoding/json"

	"github.com/google/uuid"
)

// Mode determines what capabilities the session needs.
type Mode int

const (
	// ModeExecution is for programmatic instruction execution.
	ModeExecution Mode = iota
	// ModeRecording is for human input forwarding and event capture.
	ModeRecording
	// ModeHybrid combines execution with recording-quality telemetry.
	ModeHybrid
)

// String returns the mode name.
func (m Mode) String() string {
	switch m {
	case ModeExecution:
		return "execution"
	case ModeRecording:
		return "recording"
	case ModeHybrid:
		return "hybrid"
	default:
		return "unknown"
	}
}

// Spec is the unified session specification.
type Spec struct {
	// Identity
	ExecutionID uuid.UUID
	WorkflowID  uuid.UUID

	// Mode determines capabilities
	Mode Mode

	// Display
	ViewportWidth  int
	ViewportHeight int

	// Session reuse policy: "fresh", "clean", or "reuse"
	ReuseMode string

	// Recording-specific fields
	StorageState   json.RawMessage       // Restore auth state from session profile
	FrameStreaming *FrameStreamingConfig // Live preview config

	// Execution-specific fields
	BaseURL      string                // Base URL for relative navigation
	Capabilities CapabilityRequirement // HAR, video, tracing needs

	// Labels for debugging/filtering
	Labels map[string]string
}

// FrameStreamingConfig configures live preview frame streaming.
type FrameStreamingConfig struct {
	// CallbackURL is where frames are posted. Auto-generated if empty.
	CallbackURL string
	// Quality is JPEG quality 1-100, default 55.
	Quality int
	// FPS is frames per second, default 6.
	FPS int
	// Scale is "css" or "device".
	Scale string
}

// CapabilityRequirement specifies execution capabilities needed.
type CapabilityRequirement struct {
	NeedsParallelTabs bool
	NeedsIframes      bool
	NeedsFileUploads  bool
	NeedsDownloads    bool
	NeedsHAR          bool
	NeedsVideo        bool
	NeedsTracing      bool
	MinViewportWidth  int
	MinViewportHeight int
}

// IsEmpty returns true if no capabilities are required.
func (c CapabilityRequirement) IsEmpty() bool {
	return c == CapabilityRequirement{}
}
