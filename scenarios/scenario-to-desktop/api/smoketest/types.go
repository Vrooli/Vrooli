package smoketest

import "time"

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
	LastStdout      string `json:"last_stdout,omitempty"`
	LastStderr      string `json:"last_stderr,omitempty"`
	RetryCount      int    `json:"retry_count,omitempty"`
	OutputTruncated bool   `json:"output_truncated,omitempty"`
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
