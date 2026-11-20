package runannotations

import (
	"context"
	"errors"
	"sort"
	"time"
)

// ErrScopeNotFound says the requested run/scope does not exist. It is not an
// empty annotation set and must be surfaced as 404, never as no friction.
var ErrScopeNotFound = errors.New("annotation scope not found")

// MessageFact is one bounded message the digest may read.
type MessageFact struct {
	EventID string    `json:"event_id"`
	At      time.Time `json:"at"`
	Text    string    `json:"text"`
}

// ToolSummaryFacts aggregates tool behavior for one run.
type ToolSummaryFacts struct {
	Calls          int `json:"calls"`
	Failures       int `json:"failures"`
	FailedCommands int `json:"failed_commands"`
	// RepeatedPaths counts accesses beyond the first for each path touched more
	// than once. It is a lower bound when the event window is bounded.
	RepeatedPaths int `json:"repeated_paths"`
	// RepeatedCommands counts a command issued again after its first run.
	// Most harnesses put the whole action in a shell command, so a repeated
	// command is the only observable form of re-reading the same file.
	RepeatedCommands int      `json:"repeated_commands"`
	FailedTools      []string `json:"failed_tools,omitempty"`
	LastError        string   `json:"last_error,omitempty"`
}

// TimingFacts is observed wall-clock timing.
type TimingFacts struct {
	DurationMinutes    float64  `json:"duration_min"`
	TimeToFirstToolMin *float64 `json:"time_to_first_tool_min,omitempty"`
}

// DeterministicFacts are already-proven observations a label may use as a prior.
type DeterministicFacts struct {
	StepbackTriggers []string `json:"stepback_triggers,omitempty"`
	ToolFindings     []string `json:"tool_findings,omitempty"`
	CompactionCount  int      `json:"compaction_count"`
}

// SignalFact is one aggregated existing operator signal.
type SignalFact struct {
	Label         string   `json:"label"`
	Probability   *float64 `json:"probability,omitempty"`
	Count         int      `json:"count"`
	LatestEventID string   `json:"latest_event_id,omitempty"`
	Version       string   `json:"version,omitempty"`
}

// TerminalFacts is the run's terminal observation.
type TerminalFacts struct {
	Status     string `json:"status"`
	Class      string `json:"class"`
	StopReason string `json:"stop_reason"`
	ExitCode   *int   `json:"exit_code,omitempty"`
}

// WorkloadFacts identifies why the run exists.
type WorkloadFacts struct {
	Kind string `json:"kind"`
	Key  string `json:"key"`
}

// RunFacts is everything a FeatureSource may supply. Every field is bounded
// before it reaches a digest or a decision model.
type RunFacts struct {
	RunID              string                `json:"run_id"`
	Workload           WorkloadFacts         `json:"workload"`
	HarnessKind        string                `json:"harness_kind,omitempty"`
	RunMode            string                `json:"run_mode,omitempty"`
	OperatorMessages   []MessageFact         `json:"operator_messages,omitempty"`
	AgentText          []MessageFact         `json:"agent_text,omitempty"`
	ToolSummary        ToolSummaryFacts      `json:"tool_summary"`
	Timings            TimingFacts           `json:"timings"`
	DeterministicFlags DeterministicFacts    `json:"deterministic_flags"`
	ExistingSignals    map[string]SignalFact `json:"existing_signals,omitempty"`
	Terminal           TerminalFacts         `json:"terminal"`
}

// FeatureSource supplies bounded facts for one scope. The fixture source keeps
// tests independent of the database; the SQLite source is the production
// adapter. The orchestrator may substitute a narrower adapter.
type FeatureSource interface {
	Facts(ctx context.Context, scopeKind Scope, scopeID string) (RunFacts, error)
}

// RunFeatures is the documented CEL environment for candidate rules, priors and
// deterministic rules. Every name here is stable and lowercase.
type RunFeatures struct {
	RunID                string             `json:"run_id"`
	WorkloadKind         string             `json:"workload_kind"`
	WorkloadKey          string             `json:"workload_key"`
	HarnessKind          string             `json:"harness_kind"`
	RunMode              string             `json:"run_mode"`
	Status               string             `json:"status"`
	TerminalClass        string             `json:"terminal_class"`
	StopReason           string             `json:"stop_reason"`
	ExitCode             int                `json:"exit_code"`
	DurationMin          float64            `json:"duration_min"`
	CompactionCount      int                `json:"compaction_count"`
	OperatorMessageCount int                `json:"operator_message_count"`
	AgentTextCount       int                `json:"agent_text_count"`
	ToolCalls            int                `json:"tool_calls"`
	ToolFailures         int                `json:"tool_failures"`
	FailedCommands       int                `json:"failed_commands"`
	RepeatedPaths        int                `json:"repeated_paths"`
	RepeatedCommands     int                `json:"repeated_commands"`
	StepbackTriggers     []string           `json:"stepback_triggers"`
	ToolFindings         []string           `json:"tool_findings"`
	ExistingSignals      map[string]float64 `json:"existing_signals"`
	LastOperatorMessage  string             `json:"last_operator_message"`
	LastAgentText        string             `json:"last_agent_text"`
	LastToolError        string             `json:"last_tool_error"`
}

// FeaturesFromFacts flattens facts into the CEL environment.
func FeaturesFromFacts(facts RunFacts) RunFeatures {
	features := RunFeatures{
		RunID:                facts.RunID,
		WorkloadKind:         facts.Workload.Kind,
		WorkloadKey:          facts.Workload.Key,
		HarnessKind:          facts.HarnessKind,
		RunMode:              facts.RunMode,
		Status:               facts.Terminal.Status,
		TerminalClass:        facts.Terminal.Class,
		StopReason:           facts.Terminal.StopReason,
		DurationMin:          facts.Timings.DurationMinutes,
		CompactionCount:      facts.DeterministicFlags.CompactionCount,
		OperatorMessageCount: len(facts.OperatorMessages),
		AgentTextCount:       len(facts.AgentText),
		ToolCalls:            facts.ToolSummary.Calls,
		ToolFailures:         facts.ToolSummary.Failures,
		FailedCommands:       facts.ToolSummary.FailedCommands,
		RepeatedPaths:        facts.ToolSummary.RepeatedPaths,
		RepeatedCommands:     facts.ToolSummary.RepeatedCommands,
		StepbackTriggers:     append([]string(nil), facts.DeterministicFlags.StepbackTriggers...),
		ToolFindings:         append([]string(nil), facts.DeterministicFlags.ToolFindings...),
		ExistingSignals:      map[string]float64{},
		LastToolError:        facts.ToolSummary.LastError,
	}
	if facts.Terminal.ExitCode != nil {
		features.ExitCode = *facts.Terminal.ExitCode
	}
	if len(facts.OperatorMessages) > 0 {
		features.LastOperatorMessage = facts.OperatorMessages[len(facts.OperatorMessages)-1].Text
	}
	if len(facts.AgentText) > 0 {
		features.LastAgentText = facts.AgentText[len(facts.AgentText)-1].Text
	}
	for label, signal := range facts.ExistingSignals {
		if signal.Probability != nil {
			features.ExistingSignals[label] = *signal.Probability
		}
	}
	return features
}

// Activation returns the CEL activation for a run.
func (f RunFeatures) Activation() map[string]any {
	signals := make(map[string]float64, len(f.ExistingSignals))
	for k, v := range f.ExistingSignals {
		signals[k] = v
	}
	return map[string]any{
		"run_id":                 f.RunID,
		"workload_kind":          f.WorkloadKind,
		"workload_key":           f.WorkloadKey,
		"harness_kind":           f.HarnessKind,
		"run_mode":               f.RunMode,
		"status":                 f.Status,
		"terminal_class":         f.TerminalClass,
		"stop_reason":            f.StopReason,
		"exit_code":              int64(f.ExitCode),
		"duration_min":           f.DurationMin,
		"compaction_count":       int64(f.CompactionCount),
		"operator_message_count": int64(f.OperatorMessageCount),
		"agent_text_count":       int64(f.AgentTextCount),
		"tool_calls":             int64(f.ToolCalls),
		"tool_failures":          int64(f.ToolFailures),
		"failed_commands":        int64(f.FailedCommands),
		"repeated_paths":         int64(f.RepeatedPaths),
		"repeated_commands":      int64(f.RepeatedCommands),
		"stepback_triggers":      f.StepbackTriggers,
		"tool_findings":          f.ToolFindings,
		"existing_signals":       signals,
		"last_operator_message":  f.LastOperatorMessage,
		"last_agent_text":        f.LastAgentText,
		"last_tool_error":        f.LastToolError,
	}
}

// FixtureSource is a deterministic in-memory FeatureSource for tests and
// offline evaluation.
type FixtureSource struct {
	FactsByRun map[string]RunFacts
}

// NewFixtureSource returns an empty fixture source.
func NewFixtureSource() *FixtureSource {
	return &FixtureSource{FactsByRun: map[string]RunFacts{}}
}

// Set installs a run's facts.
func (f *FixtureSource) Set(facts RunFacts) {
	if f.FactsByRun == nil {
		f.FactsByRun = map[string]RunFacts{}
	}
	f.FactsByRun[facts.RunID] = facts
}

// Facts implements FeatureSource.
func (f *FixtureSource) Facts(_ context.Context, scopeKind Scope, scopeID string) (RunFacts, error) {
	if scopeKind != ScopeRun {
		return RunFacts{}, ErrScopeNotFound
	}
	facts, ok := f.FactsByRun[scopeID]
	if !ok {
		return RunFacts{}, ErrScopeNotFound
	}
	return facts, nil
}

// SortedSignalLabels lists the existing-signal names a run carries, for
// deterministic rendering.
func (f RunFeatures) SortedSignalLabels() []string {
	labels := make([]string, 0, len(f.ExistingSignals))
	for label := range f.ExistingSignals {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	return labels
}
