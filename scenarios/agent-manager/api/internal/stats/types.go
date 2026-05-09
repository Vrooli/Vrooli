// Package stats provides an incremental metrics engine derived from the
// typed-operational eventlog. It uses a watermark pattern: on startup the
// engine replays events in batches (resumable via the stats_checkpoint
// table), then on each refresh it processes only events appended since
// the saved watermark.
//
// Design notes (the four weakness fixes ported from swarm-manager):
//
//  1. Schema-version dispatch table. processEvent does not switch on
//     event_type alone; it indexes a `processorKey{event_type,
//     schema_version}` map. Adding a new event type or bumping a schema
//     version must register a processor — there is no silent skip.
//  2. Typed Category enum. Filter inputs are validated at the HTTP edge;
//     unknown categories return 400 with the known set (see handler.go).
//  3. Resumable replay. Rebuild paginates in batches and persists the
//     last-processed rowid through CheckpointStore so a crash resumes
//     from the checkpoint, not from zero.
//  4. Ghost-event detection. registry_test.go enforces that every
//     (event_type, schema_version) registered in eventlog has a
//     processor entry here.
//
// DOC: scenarios/agent-manager/docs/internal/SEAMS.md (stats engine seam).
package stats

import "time"

// HistoryWindow reports the span of event history the engine has
// observed. Every metrics response includes this so the UI can refuse
// to render misleading aggregates over thin samples (the swarm-manager
// "InsufficientDataCard" honesty contract, ported).
type HistoryWindow struct {
	EarliestEventAt     time.Time `json:"earliest_event_at"`
	HistoryDays         float64   `json:"history_days"`
	HasHistory          bool      `json:"has_history"`
	MinSampleMeaningful int       `json:"min_sample_meaningful"`
}

// MinSampleMeaningful is the global threshold below which a sample is
// flagged as too small to draw conclusions from. The number itself is a
// product judgment lifted from swarm-manager — see the InsufficientData
// guidance in docs/internal/SEAMS.md.
const MinSampleMeaningful = 5

// FallbackInsights summarizes runner+model fallback activity across the
// observed history window.
type FallbackInsights struct {
	GeneratedAt time.Time     `json:"generated_at"`
	History     HistoryWindow `json:"history"`
	EventCount  int64         `json:"event_count"`

	RunnerAttempts   int            `json:"runner_attempts"`
	RunnerExhausted  int            `json:"runner_exhausted"`
	RunnerByReason   map[string]int `json:"runner_by_reason"`
	RunnerByPair     []FallbackPair `json:"runner_by_pair"`
	RunnerChainDepth map[int]int    `json:"runner_chain_depth"`

	ModelAttempts   int            `json:"model_attempts"`
	ModelExhausted  int            `json:"model_exhausted"`
	ModelByReason   map[string]int `json:"model_by_reason"`
	ModelByPair     []FallbackPair `json:"model_by_pair"`
	ModelChainDepth map[int]int    `json:"model_chain_depth"`
	ModelByPreset   map[string]int `json:"model_by_preset"`
}

// FallbackPair pairs a (from → to) fallback transition with how often it
// occurred across the window. Used for both runner and model fallback
// breakdowns.
type FallbackPair struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

// HealthSummary is the per-(runner, model) and per-runner current
// snapshot computed from health.transition events.
type HealthSummary struct {
	GeneratedAt     time.Time           `json:"generated_at"`
	History         HistoryWindow       `json:"history"`
	Models          []ModelHealthEntry  `json:"models"`
	Runners         []RunnerHealthEntry `json:"runners"`
	FailingLastHour []ModelHealthEntry  `json:"failing_last_hour"`
}

// ModelHealthEntry pairs a (runner, model) with its most-recent observed
// status and the timestamp of that observation.
type ModelHealthEntry struct {
	Runner              string    `json:"runner"`
	Model               string    `json:"model"`
	Status              string    `json:"status"`
	Reason              string    `json:"reason,omitempty"`
	Message             string    `json:"message,omitempty"`
	ObservedAt          time.Time `json:"observed_at"`
	TransitionsObserved int       `json:"transitions_observed"`
}

// RunnerHealthEntry mirrors ModelHealthEntry at the runner level.
type RunnerHealthEntry struct {
	Runner              string    `json:"runner"`
	Status              string    `json:"status"`
	Reason              string    `json:"reason,omitempty"`
	Message             string    `json:"message,omitempty"`
	ObservedAt          time.Time `json:"observed_at"`
	TransitionsObserved int       `json:"transitions_observed"`
}

// SandboxSummary aggregates sandbox.operation outcomes.
type SandboxSummary struct {
	GeneratedAt     time.Time                 `json:"generated_at"`
	History         HistoryWindow             `json:"history"`
	TotalOps        int                       `json:"total_ops"`
	SuccessRate     float64                   `json:"success_rate"`
	SampleSize      int                       `json:"sample_size"`
	ByOperation     map[string]OperationCount `json:"by_operation"`
	AvgDurationMs   float64                   `json:"avg_duration_ms"`
	DurationSamples int                       `json:"duration_samples"`
}

// OperationCount pairs an operation name with success/fail counts.
type OperationCount struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Failure int `json:"failure"`
}

// HeartbeatSummary aggregates heartbeat.miss events.
type HeartbeatSummary struct {
	GeneratedAt time.Time      `json:"generated_at"`
	History     HistoryWindow  `json:"history"`
	TotalMisses int            `json:"total_misses"`
	ByTarget    map[string]int `json:"by_target"`
}

// CheckpointSummary aggregates checkpoint.failure events.
type CheckpointSummary struct {
	GeneratedAt   time.Time      `json:"generated_at"`
	History       HistoryWindow  `json:"history"`
	TotalFailures int            `json:"total_failures"`
	ByStep        map[string]int `json:"by_step"`
	ByPhase       map[string]int `json:"by_phase"`
}

// RetrySummary aggregates retry.attempt events.
type RetrySummary struct {
	GeneratedAt   time.Time      `json:"generated_at"`
	History       HistoryWindow  `json:"history"`
	TotalAttempts int            `json:"total_attempts"`
	ByOperation   map[string]int `json:"by_operation"`
	ByReason      map[string]int `json:"by_reason"`
}

// Summary is the aggregated top-level view returned by GET
// /api/v1/stats/operational. It bundles every per-category summary so
// dashboards can render a single page without N round-trips.
type Summary struct {
	GeneratedAt time.Time         `json:"generated_at"`
	History     HistoryWindow     `json:"history"`
	EventCount  int64             `json:"event_count"`
	Fallback    FallbackInsights  `json:"fallback"`
	Health      HealthSummary     `json:"health"`
	Sandbox     SandboxSummary    `json:"sandbox"`
	Heartbeat   HeartbeatSummary  `json:"heartbeat"`
	Checkpoint  CheckpointSummary `json:"checkpoint"`
	Retry       RetrySummary      `json:"retry"`
}
