// Package session provides unified session management for browser automation.
// It supports recording mode, execution mode, and hybrid mode sessions.
package session

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	sessionprofilepersistence "github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
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
	Recording      *RecordingCallbacks   // Recording callbacks (injected by service layer)

	// Execution-specific fields
	BaseURL      string                // Base URL for relative navigation
	Capabilities CapabilityRequirement // HAR, video, tracing needs
	// Absolute WAV path served as a deterministic fake microphone.
	// The driver launches a dedicated browser instance per distinct value.
	FakeMicrophoneWav string

	// Labels for debugging/filtering
	Labels map[string]string

	// Anti-detection and human-like behavior configuration
	BrowserProfile *sessionprofilepersistence.BrowserProfile

	// AppTarget and ValidationContext are required together for an
	// Electron validation session. The driver attaches to the target; it does
	// not own the desktop process.
	AppTarget         *driver.AppTarget
	ValidationContext *driver.ValidationContext
}

// RecordingCallbacks configures callbacks for recording sessions.
// When set, all browser actions and page events will be reported
// through these callbacks, enabling unified recording regardless of
// how actions are initiated (manual, AI, playback).
type RecordingCallbacks struct {
	// OnAction is called when a browser action is performed.
	OnAction func(sessionID string, action *RecordedActionInfo)

	// OnPageEvent is called when a page lifecycle event occurs.
	OnPageEvent func(sessionID string, event *PageEventInfo)

	// OnFrame is called when a new frame is captured for live preview.
	OnFrame func(sessionID string, frame *FrameInfo)
}

// RecordedActionInfo contains action details for the callback.
type RecordedActionInfo struct {
	ID          string
	ActionType  string
	URL         string
	PageTitle   string
	Selector    string
	PageID      uuid.UUID
	Timestamp   string
	SequenceNum int
	Confidence  float64
	Payload     map[string]interface{}
	Source      string // "manual", "ai", "playback"
}

// PageEventInfo contains page event details for the callback.
type PageEventInfo struct {
	Type      string // "page_created", "page_navigated", "page_closed"
	PageID    uuid.UUID
	URL       string
	Title     string
	OpenerID  *uuid.UUID
	Timestamp string
}

// FrameInfo contains frame data for live preview.
type FrameInfo struct {
	Data        []byte
	MediaType   string
	Width       int
	Height      int
	CapturedAt  string
	ContentHash string
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
	NeedsParallelTabs  bool
	NeedsIframes       bool
	NeedsFileUploads   bool
	NeedsDownloads     bool
	NeedsHAR           bool
	NeedsVideo         bool
	NeedsTracing       bool
	NeedsPerfTrace     bool
	NeedsAccessibility bool
	MinViewportWidth   int
	MinViewportHeight  int
}

// IsEmpty returns true if no capabilities are required.
func (c CapabilityRequirement) IsEmpty() bool {
	return c == CapabilityRequirement{}
}
