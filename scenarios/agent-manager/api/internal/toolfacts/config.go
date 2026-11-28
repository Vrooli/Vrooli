package toolfacts

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"agent-manager/internal/domain"
)

//go:embed rules.json
var defaultRules []byte

// Rules is the data that shapes extraction. Nothing a harness or a tool does
// differently is an `if` in the code: payload field names, the repeat window,
// the volatile text two otherwise identical calls differ by, and the bounds
// that keep a long run cheap are all here, in rules.json.
type Rules struct {
	// UnchangedWindow is how many calls back an earlier call with the same
	// command and the same result still counts as the same call answered the
	// same way again.
	UnchangedWindow int `json:"unchanged_window"`
	// ChunkMinEvents and ChunkMaxEvents bound the events held in memory. A
	// chunk is released at the first moment no call awaits its result once it
	// holds ChunkMinEvents, and unconditionally at ChunkMaxEvents.
	ChunkMinEvents int `json:"chunk_min_events"`
	ChunkMaxEvents int `json:"chunk_max_events"`
	// MaxLineBytes is the longest transcript line read; a longer line is
	// counted in Facts.SkippedLines.
	MaxLineBytes int `json:"max_line_bytes"`
	// ResultHeadBytes and ResultTailBytes bound the text of one result that is
	// hashed to decide whether two results are the same. The length is hashed
	// too, so a change in the middle of a huge output still differs.
	ResultHeadBytes int `json:"result_head_bytes"`
	ResultTailBytes int `json:"result_tail_bytes"`
	// PathKeys name the tool input fields that hold the file a call touched.
	PathKeys  []string        `json:"path_keys"`
	Normalize []NormalizeRule `json:"normalize"`
	// TailCalls are the sizes of the trailing windows (the final N calls of the
	// run) whose repeat facts are reported beside the whole run's. A spinning
	// agent repeats now; a long run's totals say little about its end.
	TailCalls []int `json:"tail_calls"`
	// WaitCommands name the calls that wait, poll or ask for status. They are
	// left out of the NonWait repeat facts, so a legitimate wait loop does not
	// count as repetition. ParkCommands are waits too: they park the run.
	WaitCommands []CommandRule `json:"wait_commands"`
	ParkCommands []CommandRule `json:"park_commands"`
	// GoalUpdate recognises the call that sets a thread goal's status (Codex
	// update_goal), however the harness records it.
	GoalUpdate GoalRule `json:"goal_update"`
}

// GoalRule recognises a goal status update. Tool is a substring of the tool
// name of a call that carries the status as an input field (StatusKey); a
// Codex code-mode script carries it inside JavaScript, which CodePattern reads
// (it matches the call itself; its first capture group is the status, empty
// when the status is not a literal). Statuses is the vocabulary counted
// by name.
type GoalRule struct {
	Tool        string   `json:"tool"`
	StatusKey   string   `json:"status_key"`
	CodePattern string   `json:"code_pattern"`
	Statuses    []string `json:"statuses"`

	code *regexp.Regexp
}

// status returns the status a call sets and whether the call updates a goal.
func (g GoalRule) status(call *domain.ToolCallEventData) (string, bool) {
	if g.Tool == "" {
		return "", false
	}
	if strings.Contains(strings.ToLower(call.ToolName), g.Tool) {
		status, _ := call.Input[g.StatusKey].(string)
		return strings.ToLower(strings.TrimSpace(status)), true
	}
	if g.code == nil {
		return "", false
	}
	for _, value := range call.Input {
		text, ok := value.(string)
		if !ok || !strings.Contains(text, g.Tool) {
			continue
		}
		if match := g.code.FindStringSubmatch(text); match != nil {
			return match[1], true
		}
	}
	return "", false
}

// CommandRule recognises a command by its text: the program and its words as
// the agent ran them (wrappers stripped), and the filters its output went
// through. Exclude removes matches such as a --help lookup. Key, for park
// rules, is a pattern whose first non-empty capture group names what the call
// parks on; without a match the key is the rule's id.
type CommandRule struct {
	ID      string `json:"id"`
	Pattern string `json:"pattern"`
	Exclude string `json:"exclude,omitempty"`
	Key     string `json:"key,omitempty"`

	pattern, exclude, key *regexp.Regexp
}

func (r *CommandRule) compile() error {
	var err error
	if strings.TrimSpace(r.ID) == "" {
		return fmt.Errorf("a command rule needs an id")
	}
	if r.pattern, err = regexp.Compile(r.Pattern); err != nil {
		return fmt.Errorf("command rule %q: %w", r.ID, err)
	}
	for _, field := range []struct {
		source string
		into   **regexp.Regexp
	}{{r.Exclude, &r.exclude}, {r.Key, &r.key}} {
		if field.source == "" {
			continue
		}
		if *field.into, err = regexp.Compile(field.source); err != nil {
			return fmt.Errorf("command rule %q: %w", r.ID, err)
		}
	}
	return nil
}

// matches reports whether the command text is one the rule recognises.
func (r CommandRule) matches(text string) bool {
	return r.pattern.MatchString(text) && (r.exclude == nil || !r.exclude.MatchString(text))
}

// parkKey is what a matching park command parks on.
func (r CommandRule) parkKey(text string) string {
	if r.key != nil {
		if match := r.key.FindStringSubmatch(text); match != nil {
			for _, group := range match[1:] {
				if group != "" {
					return group
				}
			}
		}
	}
	return r.ID
}

// Rule targets: the call's command or arguments, its result text, or both.
const (
	appliesArgs   = "args"
	appliesResult = "result"
	appliesBoth   = "both"
)

// NormalizeRule rewrites volatile text before two calls or two results are
// compared: a sleep length, a timestamp, the wall time a harness prints. Hint
// is a literal the text must contain for the pattern to be tried at all, so a
// regular expression never runs over text that cannot match.
type NormalizeRule struct {
	ID      string `json:"id"`
	Applies string `json:"applies"`
	Hint    string `json:"hint"`
	Pattern string `json:"pattern"`
	Replace string `json:"replace"`

	compiled *regexp.Regexp
}

// DefaultRules returns the rules embedded in the binary.
func DefaultRules() (Rules, error) { return ParseRules(defaultRules) }

// ParseRules decodes and validates a rule document. Unknown fields are an
// error so a misspelled bound is never silently ignored.
func ParseRules(raw []byte) (Rules, error) {
	var rules Rules
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&rules); err != nil {
		return Rules{}, fmt.Errorf("toolfacts rules: %w", err)
	}
	switch {
	case rules.UnchangedWindow < 1:
		return Rules{}, fmt.Errorf("toolfacts rules: unchanged_window must be at least 1")
	case rules.ChunkMinEvents < 1 || rules.ChunkMaxEvents < rules.ChunkMinEvents:
		return Rules{}, fmt.Errorf("toolfacts rules: chunk_min_events must be at least 1 and no more than chunk_max_events")
	case rules.MaxLineBytes < 1024:
		return Rules{}, fmt.Errorf("toolfacts rules: max_line_bytes must be at least 1024")
	case rules.ResultHeadBytes < 1 || rules.ResultTailBytes < 0:
		return Rules{}, fmt.Errorf("toolfacts rules: result_head_bytes must be positive and result_tail_bytes not negative")
	}
	seen := map[string]bool{}
	for i := range rules.Normalize {
		rule := &rules.Normalize[i]
		if strings.TrimSpace(rule.ID) == "" || seen[rule.ID] {
			return Rules{}, fmt.Errorf("toolfacts rules: normalize rule %d needs a unique id", i)
		}
		seen[rule.ID] = true
		switch rule.Applies {
		case appliesArgs, appliesResult, appliesBoth:
		default:
			return Rules{}, fmt.Errorf("toolfacts rules: normalize rule %q applies to %q, want args, result or both", rule.ID, rule.Applies)
		}
		compiled, err := regexp.Compile(rule.Pattern)
		if err != nil {
			return Rules{}, fmt.Errorf("toolfacts rules: normalize rule %q: %w", rule.ID, err)
		}
		rule.compiled = compiled
	}
	for _, group := range []*[]CommandRule{&rules.WaitCommands, &rules.ParkCommands} {
		for i := range *group {
			if err := (*group)[i].compile(); err != nil {
				return Rules{}, fmt.Errorf("toolfacts rules: %w", err)
			}
		}
	}
	if rules.GoalUpdate.CodePattern != "" {
		compiled, err := regexp.Compile(rules.GoalUpdate.CodePattern)
		if err != nil {
			return Rules{}, fmt.Errorf("toolfacts rules: goal_update: %w", err)
		}
		rules.GoalUpdate.code = compiled
	}
	for _, size := range rules.TailCalls {
		if size < 1 {
			return Rules{}, fmt.Errorf("toolfacts rules: tail_calls must be positive")
		}
	}
	return rules, nil
}

// MaxTail is the largest trailing window.
func (r Rules) MaxTail() int {
	most := 0
	for _, size := range r.TailCalls {
		most = max(most, size)
	}
	return most
}

// wait reports whether a command text waits, polls or parks.
func (r Rules) wait(text string) bool {
	for _, group := range [][]CommandRule{r.WaitCommands, r.ParkCommands} {
		for _, rule := range group {
			if rule.matches(text) {
				return true
			}
		}
	}
	return false
}

// park reports whether a command text parks the run, and on what.
func (r Rules) park(text string) (string, bool) {
	for _, rule := range r.ParkCommands {
		if rule.matches(text) {
			return rule.parkKey(text), true
		}
	}
	return "", false
}

// rewrite applies every rule that targets the given text kind.
func (r Rules) rewrite(text, target string) string {
	for _, rule := range r.Normalize {
		if rule.Applies != target && rule.Applies != appliesBoth {
			continue
		}
		if rule.Hint != "" && !strings.Contains(text, rule.Hint) {
			continue
		}
		text = rule.compiled.ReplaceAllString(text, rule.Replace)
	}
	return text
}
