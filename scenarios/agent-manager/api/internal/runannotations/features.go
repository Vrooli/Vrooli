package runannotations

import (
	"context"
	"errors"
	"sort"
	"time"

	"agent-manager/internal/toolfacts"
)

// ErrScopeNotFound says the requested run/scope does not exist. It is not an
// empty annotation set and must be surfaced as 404, never as no friction.
var ErrScopeNotFound = errors.New("annotation scope not found")

// MessageFact is one bounded message the digest may read.
type MessageFact struct {
	EventID string    `json:"event_id"`
	At      time.Time `json:"at"`
	Text    string    `json:"text"`
	// Sender is who delivered the message through the API (domain.MessageSender*:
	// person or agent) when it was recorded; empty for text no authenticated
	// caller sent, which the provenance policy then classifies by lineage.
	Sender string `json:"sender,omitempty"`
	// Seed marks the first operator-source message of a run whose history was
	// read to its beginning: the message that opened the run. The source sets
	// it; the pack's provenance policy decides whether a seed is a brief.
	Seed bool `json:"seed,omitempty"`
	// Human marks operator-source text a person wrote. It is derived from the
	// pack's provenance policy, never stored by the source.
	Human bool `json:"human,omitempty"`
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

// LineageFacts is how a run came to exist: who opened it decides whose words its
// operator-source messages are.
type LineageFacts struct {
	// Imported marks a transcript read in from a harness: a person's own
	// session, whatever run it was later linked under.
	Imported bool `json:"imported,omitempty"`
	// ParentRunID is the run that spawned this one; Depth counts the spawns
	// above it (0 for a root).
	ParentRunID string `json:"parent_run_id,omitempty"`
	Depth       int    `json:"depth,omitempty"`
	// Opening is the bounded start of the run's first operator-source message,
	// read when the source reached the run's beginning. A run whose opening says
	// an agent launched it ("Agent Manager lineage: you are run ...") was
	// delegated even when no parent is recorded.
	Opening string `json:"opening,omitempty"`
}

// Spawned reports whether another run opened this one. An agent that spawns a
// run writes its operator-source text (the brief, then every steering message).
func (l LineageFacts) Spawned() bool { return !l.Imported && l.ParentRunID != "" }

// RunFacts is everything a FeatureSource may supply. Every field is bounded
// before it reaches a digest or a decision model.
type RunFacts struct {
	RunID            string        `json:"run_id"`
	Workload         WorkloadFacts `json:"workload"`
	HarnessKind      string        `json:"harness_kind,omitempty"`
	RunMode          string        `json:"run_mode,omitempty"`
	Lineage          LineageFacts  `json:"lineage"`
	OperatorMessages []MessageFact `json:"operator_messages,omitempty"`
	// OperatorHistory is every operator-source message the source read for this
	// run, oldest first, beyond the digest tail OperatorMessages. Provenance and
	// message-scope labels read it so a long conversation is not cut to its last
	// twelve messages. When the source supplies none, OperatorMessages stands in.
	OperatorHistory []MessageFact `json:"operator_history,omitempty"`
	// HumanMessages are the operator-source messages a person wrote, oldest
	// first. BriefMessages counts the rest. Both come from Provenance.Apply.
	HumanMessages      []MessageFact         `json:"human_messages,omitempty"`
	BriefMessages      int                   `json:"brief_messages,omitempty"`
	AgentText          []MessageFact         `json:"agent_text,omitempty"`
	ToolSummary        toolfacts.Facts       `json:"tool_summary"`
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
	RunID                string  `json:"run_id"`
	WorkloadKind         string  `json:"workload_kind"`
	WorkloadKey          string  `json:"workload_key"`
	HarnessKind          string  `json:"harness_kind"`
	RunMode              string  `json:"run_mode"`
	Status               string  `json:"status"`
	TerminalClass        string  `json:"terminal_class"`
	StopReason           string  `json:"stop_reason"`
	ExitCode             int     `json:"exit_code"`
	DurationMin          float64 `json:"duration_min"`
	CompactionCount      int     `json:"compaction_count"`
	OperatorMessageCount int     `json:"operator_message_count"`
	// HumanMessageCount counts operator-source messages a person wrote;
	// BriefMessageCount counts the operator-source messages no person wrote:
	// workflow briefs, harness text and agents steering agents.
	HumanMessageCount int `json:"human_message_count"`
	BriefMessageCount int `json:"brief_message_count"`
	// SpawnedByRun is true when another run opened this one; ImportedSession
	// when it is a transcript read in from a harness.
	SpawnedByRun    bool `json:"spawned_by_run"`
	ImportedSession bool `json:"imported_session"`
	AgentTextCount  int  `json:"agent_text_count"`
	// Tools are the whole-run tool facts under the names the rule environment
	// declares (toolfacts.Declarations): tool_calls, failed_commands,
	// repeated_paths, unchanged_repeats, tool_source and the rest. The
	// toolfacts package owns them; the engine carries them through.
	Tools               map[string]any     `json:"-"`
	StepbackTriggers    []string           `json:"stepback_triggers"`
	ToolFindings        []string           `json:"tool_findings"`
	ExistingSignals     map[string]float64 `json:"existing_signals"`
	LastOperatorMessage string             `json:"last_operator_message"`
	LastHumanMessage    string             `json:"last_human_message"`
	// MessageText, MessageIndex and MessageCount describe the subject of a
	// message-scope evaluation; they are empty and zero for a run.
	MessageText   string `json:"message_text"`
	MessageIndex  int    `json:"message_index"`
	MessageCount  int    `json:"message_count"`
	LastAgentText string `json:"last_agent_text"`
	LastToolError string `json:"last_tool_error"`
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
		HumanMessageCount:    len(facts.HumanMessages),
		BriefMessageCount:    facts.BriefMessages,
		SpawnedByRun:         facts.Lineage.Spawned(),
		ImportedSession:      facts.Lineage.Imported,
		AgentTextCount:       len(facts.AgentText),
		Tools:                toolActivation(facts.ToolSummary),
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
	if len(facts.HumanMessages) > 0 {
		features.LastHumanMessage = facts.HumanMessages[len(facts.HumanMessages)-1].Text
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
	activation := map[string]any{
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
		"human_message_count":    int64(f.HumanMessageCount),
		"brief_message_count":    int64(f.BriefMessageCount),
		"spawned_by_run":         f.SpawnedByRun,
		"imported_session":       f.ImportedSession,
		"agent_text_count":       int64(f.AgentTextCount),
		"stepback_triggers":      f.StepbackTriggers,
		"tool_findings":          f.ToolFindings,
		"existing_signals":       signals,
		"last_operator_message":  f.LastOperatorMessage,
		"last_human_message":     f.LastHumanMessage,
		"message_text":           f.MessageText,
		"message_index":          int64(f.MessageIndex),
		"message_count":          int64(f.MessageCount),
		"last_agent_text":        f.LastAgentText,
		"last_tool_error":        f.LastToolError,
	}
	tools := f.Tools
	if tools == nil {
		tools = toolActivation(toolfacts.Facts{})
	}
	for name, value := range tools {
		activation[name] = value
	}
	return activation
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
