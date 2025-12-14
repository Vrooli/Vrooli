// Package contracts defines engine-agnostic payloads and interfaces used by
// automation engines, executors, recorders, and event sinks. The goal is to
// keep these shapes stable so multiple engine implementations (e.g.,
// Browserless, Desktop/Playwright) can plug in without changing downstream
// consumers.
//
// Where possible, this package uses or re-exports proto-generated types from
// packages/proto/gen/go/browser-automation-studio/v1 to ensure type consistency
// with the API layer. Types that are internal to the execution engine or have
// fields not suitable for API exposure (e.g., raw bytes) remain defined here.
package contracts

import (
	"time"

	"github.com/google/uuid"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basdomain "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
)

// Re-export proto types that are used directly in contracts.
// These are types that don't require Go-specific fields like time.Time.
// Types with API-vs-engine differences (ConsoleLogEntry, NetworkEvent)
// are defined locally below because they have time.Time fields.
type (
	// === Geometry types from proto geometry.proto ===
	// BoundingBox captures the position and dimensions of a rectangular region.
	BoundingBox = basbase.BoundingBox
	// Point represents a 2D coordinate.
	Point = basbase.Point

	// === Selector types from proto selectors.proto ===
	// ElementMeta from proto selectors.proto - DOM element information.
	ElementMeta = basdomain.ElementMeta
	// SelectorCandidate from proto selectors.proto - selector with confidence.
	SelectorCandidate = basdomain.SelectorCandidate
	// HighlightRegion describes an overlay applied to the screenshot for emphasis.
	HighlightRegion = basdomain.HighlightRegion
	// MaskRegion describes areas that were dimmed or masked during capture.
	MaskRegion = basdomain.MaskRegion

	// === Timeline types from proto timeline_entry.proto ===
	// TimelineEntry from proto timeline_entry.proto - unified timeline event.
	TimelineEntry = bastimeline.TimelineEntry
	// ElementFocus captures focus metadata for screenshot framing.
	ElementFocus = bastimeline.ElementFocus

	// === Telemetry types from proto telemetry.proto ===
	// TimelineScreenshot from proto telemetry.proto - screenshot metadata.
	TimelineScreenshot = basdomain.TimelineScreenshot
	// ActionTelemetry from proto telemetry.proto - telemetry container.
	ActionTelemetry = basdomain.ActionTelemetry

	// === Shared types from proto shared.proto ===
	// AssertionResult from proto shared.proto - assertion outcome.
	AssertionResult = basbase.AssertionResult
	// EventContext from proto shared.proto - recording/execution context.
	EventContext = basbase.EventContext

	// === Action types from proto action.proto ===
	// ActionDefinition from proto action.proto - action with params.
	ActionDefinition = basactions.ActionDefinition
)

const (
	// StepOutcomeSchemaVersion tracks the shape of StepOutcome and its nested
	// payloads. Bump when fields/nullability change so replay/export clients can
	// make explicit compatibility decisions instead of silently drifting.
	StepOutcomeSchemaVersion = "automation-step-outcome-v1"
	// TelemetrySchemaVersion governs in-flight telemetry/heartbeat payloads.
	TelemetrySchemaVersion = "automation-telemetry-v1"
	// EventEnvelopeSchemaVersion governs the outer event envelope used by the
	// EventSink. This is distinct from WebSocket contracts so the bridge can
	// enforce ordering/backpressure while keeping UI payloads unchanged.
	EventEnvelopeSchemaVersion = "automation-event-envelope-v1"
	// CapabilitiesSchemaVersion tracks the shape of EngineCapabilities payloads.
	CapabilitiesSchemaVersion = "automation-capabilities-v1"
	// PayloadVersion identifies semantic changes inside the payloads without
	// altering the envelope shape. Keep this stable until the meaning of a
	// field changes.
	PayloadVersion = "1"
)

const (
	// DOMSnapshotMaxBytes caps the HTML snapshot length before truncation.
	DOMSnapshotMaxBytes = 512 * 1024
	// ScreenshotMaxBytes caps inline screenshot payload size to prevent oversized artifacts.
	ScreenshotMaxBytes = 512 * 1024
	// Default screenshot dimensions when engine omits them.
	DefaultScreenshotWidth  = 800
	DefaultScreenshotHeight = 600
	// ConsoleEntryMaxBytes caps the size of a single console entry before
	// truncation.
	ConsoleEntryMaxBytes = 16 * 1024
	// NetworkPayloadPreviewMaxBytes caps request/response previews stored for
	// parity checks; engines should annotate truncation explicitly.
	NetworkPayloadPreviewMaxBytes = 64 * 1024
)

// StepOutcome captures the normalized result of a single instruction attempt.
// Engines must not embed provider-specific fields; recorder generates durable
// IDs/dedupe keys using the correlation metadata.
//
// Note: StepOutcome is the internal representation used by the execution engine.
// For WebSocket streaming to the UI, StepOutcome is converted to TimelineEntry
// (basv1.TimelineEntry from timeline_entry.proto) via StepOutcomeToTimelineEntry in
// automation/events/unified_convert.go. New UI-facing code should work with
// TimelineEntry directly.
//
// See: docs/plans/bas-unified-timeline-workflow-types.md
type StepOutcome struct {
	SchemaVersion      string            `json:"schema_version"`
	PayloadVersion     string            `json:"payload_version"`
	ExecutionID        uuid.UUID         `json:"execution_id,omitempty"`
	CorrelationID      string            `json:"correlation_id,omitempty"` // Stable per step attempt, assigned by executor/recorder.
	StepIndex          int               `json:"step_index"`
	Attempt            int               `json:"attempt"`
	NodeID             string            `json:"node_id"`
	StepType           string            `json:"step_type"`
	Instruction        string            `json:"instruction,omitempty"`
	Success            bool              `json:"success"`
	StartedAt          time.Time         `json:"started_at"`                     // UTC, monotonic per attempt.
	CompletedAt        *time.Time        `json:"completed_at,omitempty"`         // UTC, nil if never completed.
	DurationMs         int               `json:"duration_ms,omitempty"`          // Derived; CompletedAt-StartedAt preferred source.
	FinalURL           string            `json:"final_url,omitempty"`            // Normalized URL after navigation.
	Screenshot         *Screenshot       `json:"screenshot,omitempty"`           // Final screenshot for the attempt.
	DOMSnapshot        *DOMSnapshot      `json:"dom_snapshot,omitempty"`         // DOM snapshot/html; apply size limits.
	ConsoleLogs        []ConsoleLogEntry `json:"console_logs,omitempty"`         // Ordered by timestamp.
	Network            []NetworkEvent    `json:"network,omitempty"`              // Ordered by timestamp.
	ExtractedData      map[string]any    `json:"extracted_data,omitempty"`       // Structured outputs (typed by workflow spec).
	Assertion          *AssertionOutcome `json:"assertion,omitempty"`            // Present for assert steps.
	Condition          *ConditionOutcome `json:"condition,omitempty"`            // Present for branch conditions.
	ProbeResult        map[string]any    `json:"probe_result,omitempty"`         // For probe nodes; shape defined by workflow spec.
	ElementBoundingBox *BoundingBox      `json:"element_bounding_box,omitempty"` // Target element bounding box (viewport coords).
	ClickPosition      *Point            `json:"click_position,omitempty"`       // Actual click coordinates used (viewport coords).
	FocusedElement     *ElementFocus     `json:"focused_element,omitempty"`      // Focus metadata for framing.
	HighlightRegions   []HighlightRegion `json:"highlight_regions,omitempty"`    // Overlay regions applied for emphasis.
	MaskRegions        []MaskRegion      `json:"mask_regions,omitempty"`         // Regions masked/dimmed during capture.
	ZoomFactor         float64           `json:"zoom_factor,omitempty"`          // Applied zoom for the attempt.
	CursorTrail        []CursorPosition  `json:"cursor_trail,omitempty"`         // Ordered cursor path for timeline playback.
	Notes              map[string]string `json:"notes,omitempty"`                // Freeform annotations (e.g., dedupe hints).
	Failure            *StepFailure      `json:"failure,omitempty"`              // Populated when Success=false.
}

// StepFailure codifies failure taxonomy so the executor can map to retry vs.
// abort policies deterministically.
type StepFailure struct {
	Kind       FailureKind    `json:"kind"`                  // engine|infra|orchestration|user|timeout|cancelled
	Code       string         `json:"code,omitempty"`        // Stable machine-readable error code.
	Message    string         `json:"message,omitempty"`     // Human-readable description.
	Fatal      bool           `json:"fatal,omitempty"`       // True when executor must abort workflow.
	Retryable  bool           `json:"retryable,omitempty"`   // True when executor may retry based on policy.
	OccurredAt *time.Time     `json:"occurred_at,omitempty"` // UTC timestamp of failure.
	Details    map[string]any `json:"details,omitempty"`     // Optional structured context (non-vendor-specific).
	Source     FailureSource  `json:"source,omitempty"`      // engine|executor|recorder
}

// FailureKind enumerates the supported failure taxonomy.
type FailureKind string

const (
	FailureKindNone          FailureKind = ""
	FailureKindEngine        FailureKind = "engine"
	FailureKindInfra         FailureKind = "infra"
	FailureKindOrchestration FailureKind = "orchestration"
	FailureKindUser          FailureKind = "user"
	FailureKindTimeout       FailureKind = "timeout"
	FailureKindCancelled     FailureKind = "cancelled"
)

// FailureSource indicates which layer surfaced the failure.
type FailureSource string

const (
	FailureSourceEngine   FailureSource = "engine"
	FailureSourceExecutor FailureSource = "executor"
	FailureSourceRecorder FailureSource = "recorder"
)

// Screenshot describes a captured frame. Raw bytes are intentionally omitted
// from JSON to avoid accidental large payload emission; recorder/event sinks
// decide how to serialize/store.
type Screenshot struct {
	Data        []byte    `json:"-"`                      // PNG/JPEG bytes.
	MediaType   string    `json:"media_type,omitempty"`   // e.g., image/png.
	CaptureTime time.Time `json:"capture_time,omitempty"` // UTC capture time.
	Width       int       `json:"width,omitempty"`
	Height      int       `json:"height,omitempty"`
	Hash        string    `json:"hash,omitempty"`       // Content hash for dedupe.
	FromCache   bool      `json:"from_cache,omitempty"` // True when reused.
	Truncated   bool      `json:"truncated,omitempty"`  // True when size limits applied.
	Source      string    `json:"source,omitempty"`     // e.g., "page", "replay", "recording".
}

// DOMSnapshot captures HTML + preview metadata.
type DOMSnapshot struct {
	HTML        string    `json:"html,omitempty"`         // Full HTML (may be truncated).
	Preview     string    `json:"preview,omitempty"`      // Optional text preview for timelines.
	Hash        string    `json:"hash,omitempty"`         // Content hash for dedupe.
	CollectedAt time.Time `json:"collected_at,omitempty"` // UTC timestamp.
	Truncated   bool      `json:"truncated,omitempty"`    // True when HTML exceeded limits.
}

// ConsoleLogEntry represents a console message emitted during a step.
type ConsoleLogEntry struct {
	Type      string    `json:"type"` // log|warn|error|info|debug
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"` // UTC
	Stack     string    `json:"stack,omitempty"`
	Location  string    `json:"location,omitempty"` // Source location when available.
}

// NetworkEvent captures network activity for a step.
type NetworkEvent struct {
	Type                string            `json:"type"` // request|response|failure
	URL                 string            `json:"url"`
	Method              string            `json:"method,omitempty"`
	ResourceType        string            `json:"resource_type,omitempty"`
	Status              int               `json:"status,omitempty"`
	OK                  bool              `json:"ok,omitempty"`
	Failure             string            `json:"failure,omitempty"`
	Timestamp           time.Time         `json:"timestamp"`                 // UTC
	RequestHeaders      map[string]string `json:"request_headers,omitempty"` // Canonicalized header keys.
	ResponseHeaders     map[string]string `json:"response_headers,omitempty"`
	RequestBodyPreview  string            `json:"request_body_preview,omitempty"` // Truncated per NetworkPayloadPreviewMaxBytes.
	ResponseBodyPreview string            `json:"response_body_preview,omitempty"`
	Truncated           bool              `json:"truncated,omitempty"` // True when previews truncated.
}

// AssertionOutcome captures the outcome of an assert step.
type AssertionOutcome struct {
	Mode          string `json:"mode,omitempty"`
	Selector      string `json:"selector,omitempty"`
	Expected      any    `json:"expected,omitempty"`
	Actual        any    `json:"actual,omitempty"`
	Success       bool   `json:"success"`
	Negated       bool   `json:"negated,omitempty"`
	CaseSensitive bool   `json:"case_sensitive,omitempty"`
	Message       string `json:"message,omitempty"`
}

// ConditionOutcome captures the outcome of a conditional step evaluation.
type ConditionOutcome struct {
	Type       string `json:"type,omitempty"`
	Outcome    bool   `json:"outcome"`
	Negated    bool   `json:"negated,omitempty"`
	Operator   string `json:"operator,omitempty"`
	Variable   string `json:"variable,omitempty"`
	Selector   string `json:"selector,omitempty"`
	Expression string `json:"expression,omitempty"`
	Actual     any    `json:"actual,omitempty"`
	Expected   any    `json:"expected,omitempty"`
}

// NOTE: BoundingBox, Point, ElementFocus, HighlightRegion, MaskRegion are now
// type aliases to proto types defined in the type() block above. This eliminates
// duplication and ensures type consistency with the API layer.

// CursorPosition represents a point along the cursor trail for a step attempt.
type CursorPosition struct {
	Point      Point     `json:"point"`
	RecordedAt time.Time `json:"recorded_at,omitempty"` // UTC timestamp of capture.
	ElapsedMs  int64     `json:"elapsed_ms,omitempty"`  // Relative to step start for ordering.
}
