// Package telemetry defines the unified intermediate representation for
// browser actions, supporting both recording (human-captured) and execution
// (programmatic) paths with a single conversion to TimelineEntry.
//
// Architecture:
//
//	Recording Path:                    Execution Path:
//	driver.RecordedAction              contracts.StepOutcome
//	       │                                  │
//	       ▼                                  ▼
//	RecordedActionToTelemetry          StepOutcomeToTelemetry
//	       │                                  │
//	       └──────────┬───────────────────────┘
//	                  ▼
//	           ActionTelemetry
//	                  │
//	                  ▼
//	     TelemetryToTimelineEntry
//	                  │
//	                  ▼
//	       bastimeline.TimelineEntry
//
// This design eliminates code duplication between the recording and execution
// converters while enabling new capabilities like hybrid recording+execution.
package telemetry

import (
	"time"

	"github.com/google/uuid"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basdomain "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
)

// ActionTelemetry is the unified intermediate representation of a browser action.
// Both recording (captured human actions) and execution (programmatic steps)
// produce this type before final conversion to TimelineEntry.
//
// Design principles:
//   - All fields are optional (pointer or slice types)
//   - Origin discriminates recording vs execution metadata
//   - Common fields are at the top level
//   - Origin-specific fields are in the Origin union
type ActionTelemetry struct {
	// === Identity ===
	ID          string // Unique identifier for this action/step
	SequenceNum int    // Order in recording/execution sequence
	NodeID      string // Workflow node ID (execution) or generated (recording)
	StepIndex   int    // Step index in execution plan (execution only)

	// === Action Definition ===
	ActionType basactions.ActionType // click, navigate, input, etc.
	Params     map[string]any        // Action-specific parameters
	Label      string                // Human-readable label for UI

	// === Timing ===
	Timestamp  time.Time // When the action started
	DurationMs int       // How long it took

	// === Element Context ===
	// These fields capture the target element when the action interacts with DOM
	Selector           string                 // The selector used (primary for recording, actual for execution)
	SelectorConfidence float64                // Match confidence (0-1)
	SelectorMatchCount int                    // Number of elements matched
	ElementSnapshot    *basdomain.ElementMeta // Tag, ID, class, role, aria-label, etc.
	BoundingBox        *basbase.BoundingBox   // Element position in viewport

	// === Page Context ===
	URL      string // Page URL at time of action
	FrameID  string // Frame identifier for iframe actions
	FinalURL string // URL after navigation (for navigate actions)

	// === Captured Artifacts ===
	Screenshot  *Screenshot       // Screenshot at action completion
	DOMSnapshot *DOMSnapshot      // HTML snapshot
	ConsoleLogs []ConsoleLogEntry // Console messages during action
	Network     []NetworkEvent    // Network activity during action

	// === Visual Context ===
	CursorPosition   *basbase.Point               // Cursor position at action time
	ClickPosition    *basbase.Point               // Actual click coordinates (may differ from cursor)
	CursorTrail      []*basbase.Point             // Cursor path for animations
	HighlightRegions []*basdomain.HighlightRegion // Regions highlighted in screenshot
	MaskRegions      []*basdomain.MaskRegion      // Regions masked in screenshot
	ZoomFactor       float64                      // Applied zoom level

	// === Result ===
	Success bool         // Whether the action succeeded
	Failure *FailureInfo // Failure details if Success=false

	// === Origin ===
	// Discriminated union: exactly one of these will be non-nil
	Origin ActionOrigin
}

// ActionOrigin discriminates between recording and execution sources.
// Use type switch to access origin-specific fields.
type ActionOrigin interface {
	isActionOrigin()
	OriginType() string // "recording" or "execution"
}

// RecordingOrigin contains metadata specific to recorded actions.
type RecordingOrigin struct {
	SessionID          string                         // Recording session ID
	Confidence         float64                        // ML confidence score (0-1)
	SelectorCandidates []*basdomain.SelectorCandidate // Alternative selectors with scores
	NeedsConfirmation  bool                           // Whether user should verify selector
	Source             basbase.RecordingSource        // auto, manual, ai-suggested
}

func (r *RecordingOrigin) isActionOrigin() {}

// OriginType returns "recording".
func (r *RecordingOrigin) OriginType() string { return "recording" }

// ExecutionOrigin contains metadata specific to executed steps.
type ExecutionOrigin struct {
	ExecutionID uuid.UUID // Execution run ID
	WorkflowID  uuid.UUID // Parent workflow ID
	StepIndex   int       // Position in execution plan
	Attempt     int       // Retry attempt number (1-based)
	MaxAttempts int       // Configured max retries

	// Execution-specific outputs
	ExtractedData map[string]any    // Data extraction results
	Assertion     *AssertionOutcome // Assertion step result
	Condition     *ConditionOutcome // Branch condition result
	ProbeResult   map[string]any    // Probe step result
}

func (e *ExecutionOrigin) isActionOrigin() {}

// OriginType returns "execution".
func (e *ExecutionOrigin) OriginType() string { return "execution" }

// Screenshot captures image data from the browser.
type Screenshot struct {
	Data      []byte // Raw image bytes (PNG/JPEG)
	MediaType string // MIME type (image/png, image/jpeg)
	Width     int    // Image width in pixels
	Height    int    // Image height in pixels
}

// DOMSnapshot captures the HTML state.
type DOMSnapshot struct {
	HTML    string // Full HTML (may be truncated)
	Preview string // Short text preview for UI
}

// ConsoleLogEntry represents a browser console message.
type ConsoleLogEntry struct {
	Type      string    // log, warn, error, info, debug
	Text      string    // Message content
	Timestamp time.Time // When the message was logged
	Stack     string    // Stack trace (if available)
	Location  string    // Source location (if available)
}

// NetworkEvent represents network activity.
type NetworkEvent struct {
	Type         string    // request, response, failure
	URL          string    // Request URL
	Method       string    // HTTP method
	ResourceType string    // Resource type (document, script, etc.)
	Status       int       // Response status code
	OK           bool      // Whether the response was successful
	Failure      string    // Failure reason (if failed)
	Timestamp    time.Time // When the event occurred
}

// FailureInfo captures failure details.
type FailureInfo struct {
	Kind      string // engine, infra, timeout, cancelled, etc.
	Code      string // Machine-readable error code
	Message   string // Human-readable description
	Retryable bool   // Whether retry is possible
}

// AssertionOutcome captures assertion step results.
type AssertionOutcome struct {
	Mode          string // exists, visible, contains, equals, etc.
	Selector      string // Target selector
	Expected      any    // Expected value
	Actual        any    // Actual value found
	Success       bool   // Whether assertion passed
	Negated       bool   // Whether assertion was negated (not exists)
	CaseSensitive bool   // Whether comparison is case-sensitive
	Message       string // Human-readable result message
}

// ConditionOutcome captures branch condition results.
type ConditionOutcome struct {
	Type     string // selector, variable, expression
	Outcome  bool   // Condition result
	Negated  bool   // Whether condition was negated
	Operator string // Comparison operator (if applicable)
	Variable string // Variable name (if applicable)
	Selector string // Selector (if applicable)
	Actual   any    // Actual value evaluated
	Expected any    // Expected value (if applicable)
}
