package smoketest

import (
	"time"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	"scenario-to-desktop-api/procmetrics"
)

// State represents the current phase of a smoke test.
type State string

const (
	StateInitializing       State = "initializing"
	StateValidatingArtifact State = "validating_artifact"
	StateValidatingPrereqs  State = "validating_prerequisites"
	StateResolvingCommand   State = "resolving_command"
	StateExecuting          State = "executing"
	StateRetrying           State = "retrying"
	StateParsingOutput      State = "parsing_output"
	StateTelemetryUpload    State = "telemetry_upload"
	StateTelemetryFallback  State = "telemetry_fallback"
	StatePassed             State = "passed"
	StateFailed             State = "failed"
)

// ValidStateTransitions defines all valid state transitions in the smoke test state machine.
var ValidStateTransitions = map[State][]State{
	"": { // Initial empty state
		StateInitializing,
	},
	StateInitializing: {
		StateValidatingArtifact,
		StateFailed,
	},
	StateValidatingArtifact: {
		StateValidatingPrereqs,
		StateResolvingCommand,
		StateFailed,
	},
	StateValidatingPrereqs: {
		StateResolvingCommand,
		StateFailed,
	},
	StateResolvingCommand: {
		StateExecuting,
		StateFailed,
	},
	StateExecuting: {
		StateRetrying,
		StateParsingOutput,
		StateFailed,
	},
	StateRetrying: {
		StateExecuting,
		StateParsingOutput,
		StateFailed,
	},
	StateParsingOutput: {
		StateTelemetryUpload,
		StateTelemetryFallback,
		StatePassed,
		StateFailed,
	},
	StateTelemetryUpload: {
		StatePassed,
		StateFailed,
	},
	StateTelemetryFallback: {
		StatePassed,
		StateFailed,
	},
	StatePassed: {}, // Terminal state
	StateFailed: {}, // Terminal state
}

// CanTransitionTo checks if transitioning from this state to the target is valid.
func (s State) CanTransitionTo(target State) bool {
	validTargets, ok := ValidStateTransitions[s]
	if !ok {
		return false
	}
	for _, valid := range validTargets {
		if valid == target {
			return true
		}
	}
	return false
}

// IsTerminal returns true if this is a terminal state (no valid outgoing transitions).
func (s State) IsTerminal() bool {
	transitions, ok := ValidStateTransitions[s]
	return ok && len(transitions) == 0
}

// StateTransition records a state change for debugging.
type StateTransition struct {
	From       State     `json:"from"`
	To         State     `json:"to"`
	Timestamp  time.Time `json:"timestamp"`
	Message    string    `json:"message,omitempty"`
	DurationMs int64     `json:"duration_ms"` // Time spent in From state (ms)
}

// Status represents the status of a desktop smoke test run.
type Status struct {
	SmokeTestID          string     `json:"smoke_test_id"`
	ScenarioName         string     `json:"scenario_name"`
	Platform             string     `json:"platform"`
	Status               string     `json:"status"` // running, passed, failed
	ArtifactPath         string     `json:"artifact_path,omitempty"`
	StartedAt            time.Time  `json:"started_at"`
	CompletedAt          *time.Time `json:"completed_at,omitempty"`
	Logs                 []string   `json:"logs,omitempty"`
	Error                string     `json:"error,omitempty"`
	TelemetryUploaded    bool       `json:"telemetry_uploaded,omitempty"`
	TelemetryUploadError string     `json:"telemetry_upload_error,omitempty"`

	// Structured error for programmatic handling
	ErrorKind       *ErrorKind        `json:"error_kind,omitempty"`
	ErrorContext    map[string]string `json:"error_context,omitempty"`
	SuggestedAction string            `json:"suggested_action,omitempty"`

	// State tracking for observability
	CurrentState State             `json:"current_state,omitempty"`
	Transitions  []StateTransition `json:"transitions,omitempty"`

	// Execution details for debugging
	LastStdout              string `json:"last_stdout,omitempty"`
	LastStderr              string `json:"last_stderr,omitempty"`
	RetryCount              int    `json:"retry_count,omitempty"`
	OutputTruncated         bool   `json:"output_truncated,omitempty"`
	ExtractedLifecycleState string `json:"extracted_lifecycle_state,omitempty"` // Debug: lifecycle state extracted from output

	// AppReportedError contains the actual error from the app's telemetry
	// when available. This provides more specific diagnostic information
	// than generic lifecycle state interpretations.
	AppReportedError *TelemetryError `json:"app_reported_error,omitempty"`

	// AppSessionID is the session ID extracted from the SMOKE_TEST_INIT marker.
	// Used to correlate errors from the telemetry file with this specific run.
	AppSessionID string `json:"app_session_id,omitempty"`

	// AppReportedErrorStale indicates the app-reported error is from before the smoke test started.
	// This suggests the error may be from a previous session.
	AppReportedErrorStale bool `json:"app_reported_error_stale,omitempty"`

	// ErrorSessionMismatch indicates the app-reported error's session ID doesn't match the current run.
	// This definitively shows the error is from a different session.
	ErrorSessionMismatch bool `json:"error_session_mismatch,omitempty"`

	// Screen recording configuration and results
	RecordingConfig           *ScreenRecordingConfig `json:"recording_config,omitempty"`
	ScreenRecording           *RecordingStatus       `json:"screen_recording,omitempty"`
	JourneyCaptureID          string                 `json:"journey_capture_id,omitempty"`
	JourneyDisposition        string                 `json:"journey_disposition,omitempty"`
	JourneyDegradedReason     string                 `json:"journey_degraded_reason,omitempty"`
	EvidenceReportDisposition string                 `json:"evidence_report_disposition,omitempty"`
	EvidenceReportError       string                 `json:"evidence_report_error,omitempty"`
	EvidenceReview            *JourneyReview         `json:"evidence_review,omitempty"`

	// Process metrics from app execution
	SplashDurationMs        *int64                         `json:"splash_duration_ms,omitempty"`
	ReadyDurationMs         *int64                         `json:"ready_duration_ms,omitempty"`
	ResourceSummary         *procmetrics.Summary           `json:"resource_summary,omitempty"`
	ProtocolResourceSummary *procmetrics.Summary           `json:"protocol_resource_summary,omitempty"`
	DemoResourceSummary     *procmetrics.Summary           `json:"demo_resource_summary,omitempty"`
	DemoProcessTree         *procmetrics.ProcessTreeReport `json:"demo_process_tree,omitempty"`
	ProtocolTracePath       string                         `json:"protocol_trace_path,omitempty"`
	DemoTracePath           string                         `json:"demo_trace_path,omitempty"`
	ProtocolProfileDir      string                         `json:"protocol_profile_dir,omitempty"`
	DemoProfileDir          string                         `json:"demo_profile_dir,omitempty"`
	PerformanceStatus       string                         `json:"performance_status,omitempty"`
	PerformanceReason       string                         `json:"performance_reason,omitempty"`
	ProtocolPhases          []PerformancePhase             `json:"protocol_phases,omitempty"`
	DemoPhases              []PerformancePhase             `json:"demo_phases,omitempty"`
}

type PerformancePhase struct {
	Name       string `json:"name"`
	Available  bool   `json:"available"`
	DurationMs int64  `json:"duration_ms,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

type JourneyReview struct {
	SchemaVersion     string                                   `json:"schema_version"`
	Capability        string                                   `json:"capability"`
	PlanID            string                                   `json:"plan_id"`
	Profile           string                                   `json:"profile"`
	Disposition       string                                   `json:"disposition"`
	Reason            string                                   `json:"reason,omitempty"`
	EventCount        int                                      `json:"event_count"`
	DeploymentMode    string                                   `json:"deployment_mode,omitempty"`
	ProviderTier      string                                   `json:"provider_tier,omitempty"`
	ServiceIdentity   string                                   `json:"service_identity,omitempty"`
	Readiness         string                                   `json:"readiness,omitempty"`
	FallbackDecision  string                                   `json:"fallback_decision,omitempty"`
	SafeRouteClass    string                                   `json:"safe_route_class,omitempty"`
	WorkflowRequired  bool                                     `json:"workflow_required,omitempty"`
	WorkflowReference *deliveryramp.WorkflowExecutionReference `json:"workflow_reference,omitempty"`
	Chapters          []JourneyChapter                         `json:"chapters"`
}

type JourneyChapter struct {
	ID                 string   `json:"id"`
	Purpose            string   `json:"purpose"`
	Action             string   `json:"action"`
	Disposition        string   `json:"disposition"`
	AssertionID        string   `json:"assertion_id,omitempty"`
	Expected           string   `json:"expected,omitempty"`
	Observed           string   `json:"observed,omitempty"`
	Error              string   `json:"error,omitempty"`
	VideoStartOffsetMs *int64   `json:"video_start_offset_ms,omitempty"`
	VideoEndOffsetMs   *int64   `json:"video_end_offset_ms,omitempty"`
	EvidenceIDs        []string `json:"evidence_ids,omitempty"`
}

// ScreenRecordingConfig controls whether the smoke test records the display.
type ScreenRecordingConfig struct {
	Enabled        bool `json:"enabled"`
	DisplayWidth   int  `json:"display_width,omitempty"`  // Default: 1920
	DisplayHeight  int  `json:"display_height,omitempty"` // Default: 1080
	FPS            int  `json:"fps,omitempty"`            // Default: 15
	MaxDurationSec int  `json:"max_duration_sec,omitempty"`
}

// RecordingStatus holds only the durable capture identity and producer-side
// checksum for a screen recording. Artifact paths and bytes never live here.
type RecordingStatus struct {
	Recorded  bool   `json:"recorded"`
	CaptureID string `json:"capture_id,omitempty"`
	Checksum  string `json:"checksum,omitempty"`
	Error     string `json:"error,omitempty"`
}

// CancelResponse represents the response from cancelling a smoke test.
type CancelResponse struct {
	Status string `json:"status"`
}

// OutputResult represents the parsed result of smoke test output.
type OutputResult struct {
	// Passed indicates if the smoke test passed.
	Passed bool

	// TelemetryUploaded indicates if telemetry was successfully uploaded.
	TelemetryUploaded bool

	// TelemetryUploadError indicates if telemetry upload failed.
	TelemetryUploadError bool

	// InitComplete indicates the app completed the init sequence.
	InitComplete bool

	// CleanShutdown indicates the app exited cleanly after success.
	CleanShutdown bool

	// Warnings contains non-fatal issues detected during parsing.
	Warnings []string

	// TruncatedBytes indicates how many bytes were truncated from output (0 if none).
	TruncatedBytes int
}

// SequenceValidation provides detailed validation of the smoke test output sequence.
type SequenceValidation struct {
	// Valid indicates if the sequence follows the expected order.
	Valid bool

	// Stages contains the detected stages in order.
	Stages []SequenceStage

	// MissingStages lists stages that were expected but not found.
	MissingStages []string

	// OutOfOrderStages lists stages that appeared in unexpected order.
	OutOfOrderStages []string

	// Errors contains validation error messages.
	Errors []string
}

// SequenceStage represents a detected stage in the smoke test output.
type SequenceStage struct {
	// Name is the stage identifier (init, ready, passed, exit).
	Name string

	// LineNumber is the 1-based line number where the stage was detected.
	LineNumber int

	// Timestamp is the detected timestamp if present in the output.
	Timestamp string
}

// TelemetrySource identifies where telemetry data was obtained from.
type TelemetrySource string

const (
	// TelemetrySourceUpload indicates telemetry was uploaded directly by the app.
	TelemetrySourceUpload TelemetrySource = "upload"

	// TelemetrySourceOutputExtraction indicates telemetry path was extracted from output.
	TelemetrySourceOutputExtraction TelemetrySource = "output_extraction"

	// TelemetrySourceArtifactResolution indicates telemetry path was resolved from artifact.
	TelemetrySourceArtifactResolution TelemetrySource = "artifact_resolution"

	// TelemetrySourceNone indicates no telemetry was obtained.
	TelemetrySourceNone TelemetrySource = "none"
)

// TelemetryResult captures the outcome of telemetry collection.
type TelemetryResult struct {
	// Source indicates where the telemetry was obtained from.
	Source TelemetrySource

	// Path is the telemetry file path (if applicable).
	Path string

	// EventsFound is the number of events discovered.
	EventsFound int

	// EventsIngested is the number of events successfully ingested.
	EventsIngested int

	// PartialFailure indicates some events failed to ingest.
	PartialFailure bool

	// AttemptedPaths shows the chain of paths attempted.
	AttemptedPaths []PathAttempt

	// Error is set if telemetry collection failed.
	Error string
}

// PathAttempt records a single attempt to obtain telemetry.
type PathAttempt struct {
	// Source is the method used.
	Source TelemetrySource

	// Path is the path attempted (if applicable).
	Path string

	// Success indicates if this attempt succeeded.
	Success bool

	// Error is set if this attempt failed.
	Error string
}
