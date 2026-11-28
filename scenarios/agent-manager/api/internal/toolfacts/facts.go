// Package toolfacts turns the tool activity of one run into a typed fact
// struct, however that activity was recorded: as run_events rows, or only as
// the harness's own transcript file. It is the single owner of the tool facts
// the annotation engine's rules and digests read (tool_calls, failed_commands,
// repeated_paths, repeated_commands, unchanged_repeats, ...).
//
// It parses nothing of its own. Commands come from runsignal (the shared
// lexer, including the literal exec_command calls inside Codex code-mode
// JavaScript), per-command outcomes from runsignal.DeriveInvocationFacts, and
// transcript lines from the harness's own runner.TranscriptParser, the same
// parser the Transcript Indexer and the importer use. The whole run is read,
// as a stream: memory is bounded by a chunk of events, not by the run.
package toolfacts

import (
	"sort"
	"strconv"
)

// Where a run's facts were read from.
const (
	SourceRunEvents  = "run_events"
	SourceTranscript = "transcript"
	// SourceNone means the run has neither tool events nor a readable
	// transcript, so every count is zero because nothing was observable, not
	// because the agent used no tools.
	SourceNone = "none"
)

// Facts are the tool facts of one whole run.
type Facts struct {
	Source string `json:"source"`
	// ToolCalls is the number of tool calls in the run.
	ToolCalls int `json:"tool_calls"`
	// Commands is the number of shell commands those calls ran (a Codex
	// code-mode script and a piped command line each contribute their own).
	Commands int `json:"commands"`
	// ToolFailures counts tool results that failed with a known outcome.
	ToolFailures int `json:"tool_failures"`
	// FailedCommands counts command segments that ended in a failure with a
	// process exit code. A search that found nothing is not one.
	FailedCommands int `json:"failed_commands"`
	// RepeatedPaths counts accesses beyond the first for each path touched
	// more than once, over the whole run.
	RepeatedPaths int `json:"repeated_paths"`
	// RepeatedCommands counts a command issued again after its first run,
	// over the whole run.
	RepeatedCommands int `json:"repeated_commands"`
	// UnchangedRepeats counts the calls whose normalized command or
	// arguments AND result both equal an earlier call within the last
	// Rules.UnchangedWindow calls: the same question asked again and answered
	// the same way. Polling, waiting and retry loops produce it; a file read
	// again after the agent edited it does not, because the result changed.
	UnchangedRepeats int `json:"unchanged_repeats"`
	// LongestUnchangedRun is the longest chain of such repeats of one call: 0
	// when none repeated, 19 for a call made twenty times in a row.
	LongestUnchangedRun int `json:"longest_unchanged_run"`
	// WaitCalls counts the commands that wait, poll or ask for status (and
	// park). NonWaitUnchangedRepeats and NonWaitLongestUnchangedRun are the two
	// facts above over the calls that are not such commands: a wait loop is not
	// repetition.
	WaitCalls                  int `json:"wait_calls"`
	NonWaitUnchangedRepeats    int `json:"nonwait_unchanged_repeats"`
	NonWaitLongestUnchangedRun int `json:"nonwait_longest_unchanged_run"`
	// ParkCalls counts park commands. LongestSameParkStreak is the longest run
	// of consecutive parks on the same key (other work between them does not
	// break it): a run that wakes and parks on the same thing again.
	ParkCalls             int `json:"park_calls"`
	LongestSameParkStreak int `json:"longest_same_park_streak"`
	// GoalUpdates counts the calls that set a thread goal's status (Codex
	// update_goal). GoalStatuses counts them by the status they set, LastGoalStatus
	// is the status of the last one ("" when none, or when its status was not a
	// literal), and CallsAfterLastGoalUpdate is how many calls followed it: a run
	// that ends on `update_goal blocked` has 0.
	GoalUpdates              int            `json:"goal_updates"`
	GoalStatuses             map[string]int `json:"goal_statuses,omitempty"`
	LastGoalStatus           string         `json:"last_goal_status,omitempty"`
	CallsAfterLastGoalUpdate int            `json:"calls_after_last_goal_update"`
	// Tails are the repeat facts of the final N calls (Rules.TailCalls), each
	// computed on its own window.
	Tails       []TailFacts `json:"tails,omitempty"`
	FailedTools []string    `json:"failed_tools,omitempty"`
	LastError   string      `json:"last_error,omitempty"`
	// EventsRead is the number of tool call and result events read.
	EventsRead int `json:"events_read"`
	// ParseErrors and SkippedLines are transcript lines the parser refused and
	// lines longer than Rules.MaxLineBytes. Zero for a run read from events.
	ParseErrors  int `json:"parse_errors,omitempty"`
	SkippedLines int `json:"skipped_lines,omitempty"`
}

// TailFacts are the repeat facts of the final Window calls of a run (all of
// them when the run is shorter).
type TailFacts struct {
	Window              int `json:"window"`
	Calls               int `json:"calls"`
	UnchangedRepeats    int `json:"unchanged_repeats"`
	LongestUnchangedRun int `json:"longest_unchanged_run"`
	RepeatedCommands    int `json:"repeated_commands"`
}

// Variable is one name in the annotation engine's rule environment.
type Variable struct {
	Name string
	// Type is "int" or "string".
	Type string
}

// Declarations lists every variable Activation returns for these rules, sorted
// by name, so the rule environment can declare them from the rules alone.
func Declarations(rules Rules) []Variable {
	var out []Variable
	for name, value := range (Facts{}).Activation(rules) {
		kind := "int"
		if _, text := value.(string); text {
			kind = "string"
		}
		out = append(out, Variable{Name: name, Type: kind})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Activation is the facts as the rule environment names them: counts as int64
// and tool_source ("run_events", "transcript" or "none") and last_goal_status
// as strings. A run whose tool_source is "none" has no observable tool
// activity: its zeros are absence of evidence, which a rule can test.
func (f Facts) Activation(rules Rules) map[string]any {
	out := map[string]any{
		"tool_source":                   f.Source,
		"tool_calls":                    int64(f.ToolCalls),
		"tool_failures":                 int64(f.ToolFailures),
		"failed_commands":               int64(f.FailedCommands),
		"repeated_paths":                int64(f.RepeatedPaths),
		"repeated_commands":             int64(f.RepeatedCommands),
		"unchanged_repeats":             int64(f.UnchangedRepeats),
		"longest_unchanged_run":         int64(f.LongestUnchangedRun),
		"wait_calls":                    int64(f.WaitCalls),
		"nonwait_unchanged_repeats":     int64(f.NonWaitUnchangedRepeats),
		"nonwait_longest_unchanged_run": int64(f.NonWaitLongestUnchangedRun),
		"park_calls":                    int64(f.ParkCalls),
		"longest_same_park_streak":      int64(f.LongestSameParkStreak),
		"goal_updates":                  int64(f.GoalUpdates),
		"last_goal_status":              f.LastGoalStatus,
		"calls_after_last_goal_update":  int64(f.CallsAfterLastGoalUpdate),
	}
	for _, status := range rules.GoalUpdate.Statuses {
		out["goal_"+status] = int64(f.GoalStatuses[status])
	}
	for i, size := range rules.TailCalls {
		prefix := "tail" + strconv.Itoa(size) + "_"
		var tail TailFacts
		if i < len(f.Tails) {
			tail = f.Tails[i]
		}
		out[prefix+"unchanged_repeats"] = int64(tail.UnchangedRepeats)
		out[prefix+"longest_unchanged_run"] = int64(tail.LongestUnchangedRun)
		out[prefix+"repeated_commands"] = int64(tail.RepeatedCommands)
	}
	return out
}
