// Package agenttranscript is the single reader of the goal, context-size and
// compaction records in Claude and Codex transcripts. Web Console and Agent
// Manager both import it and must not depend on each other, so it imports only
// the standard library and adds nothing to any module's go.sum.
//
// A line is read two ways. ParseCodexGoalUpdate and ParseClaudeGoalStatuses
// report what the line says, for a caller that needs an exact record. GoalTracker
// decides whether a session can be compacted, and treats anything it cannot
// prove finished as an active goal.
package agenttranscript

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"regexp"
	"slices"
	"sort"
	"strings"
)

// Harnesses whose transcripts this package reads, by harness ID.
const (
	HarnessClaude = "claude"
	HarnessCodex  = "codex"
)

// Codex goal statuses that leave nothing running. Every other status holds a
// session back from compaction, including blocked, the usage limits and any
// status not seen yet. These are compared exactly.
const (
	codexGoalPaused   = "paused"
	codexGoalComplete = "complete"
)

func codexGoalFinished(status string) bool {
	return status == codexGoalPaused || status == codexGoalComplete
}

var (
	codexGoalUpdateNeedle = []byte(`"thread_goal_updated"`)
	codexGoalToolNeedle   = []byte(`_goal`)
	claudeGoalNeedle      = []byte(`"goal_status"`)

	// codexGoalToolStatusArg finds each status an exec script passes to update_goal.
	codexGoalToolStatusArg = regexp.MustCompile(`["']?status["']?\s*:\s*["']([A-Za-z_]+)["']`)
)

// CodexGoalUpdate is one thread_goal_updated record. A status the vocabulary
// does not list is reported as written.
type CodexGoalUpdate struct {
	Envelope  string
	ThreadID  string
	Objective string
	Status    string
	Iteration int
	Reason    string
}

// ParseCodexGoalUpdate reads a thread_goal_updated record from one rollout line.
func ParseCodexGoalUpdate(line []byte) (CodexGoalUpdate, bool) {
	update, ok, _ := parseCodexGoalUpdate(line)
	return update, ok
}

func parseCodexGoalUpdate(line []byte) (update CodexGoalUpdate, ok, malformed bool) {
	var envelope struct {
		Type    string `json:"type"`
		Payload struct {
			Type     string `json:"type"`
			ThreadID string `json:"threadId"`
			Goal     struct {
				ThreadID  string `json:"threadId"`
				Objective string `json:"objective"`
				Status    string `json:"status"`
				Iteration int    `json:"iteration"`
				Reason    string `json:"reason"`
			} `json:"goal"`
		} `json:"payload"`
	}
	if json.Unmarshal(line, &envelope) != nil {
		return CodexGoalUpdate{}, false, true
	}
	if envelope.Payload.Type != "thread_goal_updated" {
		return CodexGoalUpdate{}, false, false
	}
	goal := envelope.Payload.Goal
	thread := goal.ThreadID
	if thread == "" {
		thread = envelope.Payload.ThreadID
	}
	return CodexGoalUpdate{
		Envelope: envelope.Type, ThreadID: thread, Objective: strings.TrimSpace(goal.Objective),
		Status: goal.Status, Iteration: goal.Iteration, Reason: strings.TrimSpace(goal.Reason),
	}, true, false
}

// ClaudeGoalStatus is one goal_status object. Claude writes it as an attachment,
// and also records met for a goal the session no longer holds. A status that is
// not a JSON boolean reads as not met.
type ClaudeGoalStatus struct {
	Condition string
	Met       bool
	Iteration int
	Reason    string
}

// ParseClaudeGoalStatuses returns every goal_status object on one transcript
// line, in a fixed order: array order, then sorted object keys. It walks JSON
// values only, so assistant prose cannot read as a status. ok is false when the
// line is not JSON.
func ParseClaudeGoalStatuses(line []byte) (statuses []ClaudeGoalStatus, ok bool) {
	var value any
	if json.Unmarshal(bytes.TrimSpace(line), &value) != nil {
		return nil, false
	}
	walkClaudeGoalStatuses(value, &statuses)
	return statuses, true
}

func walkClaudeGoalStatuses(value any, statuses *[]ClaudeGoalStatus) {
	switch v := value.(type) {
	case []any:
		for _, child := range v {
			walkClaudeGoalStatuses(child, statuses)
		}
	case map[string]any:
		if v["type"] == "goal_status" {
			condition, _ := v["condition"].(string)
			met, _ := v["met"].(bool)
			iteration, _ := v["iterations"].(float64)
			if iteration == 0 {
				iteration, _ = v["iteration"].(float64)
			}
			reason, _ := v["reason"].(string)
			*statuses = append(*statuses, ClaudeGoalStatus{
				Condition: strings.TrimSpace(condition), Met: met, Iteration: int(iteration), Reason: strings.TrimSpace(reason),
			})
		}
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			walkClaudeGoalStatuses(v[key], statuses)
		}
	}
}

// GoalTracker follows whether a transcript holds an active goal, one line at a
// time, so a caller can resume from where it stopped. It only clears on a record
// that names a finished status for the goal it is tracking:
//
//   - a finished status for another objective or thread is a late report of an
//     older goal and changes nothing;
//   - a goal started by a Codex create_goal or update_goal tool call counts from
//     the call itself, because Codex writes no thread_goal_updated for it, and a
//     tool result does not clear it;
//   - a line that names a goal record but cannot be decoded counts as active.
type GoalTracker struct {
	active    bool
	objective string
	thread    string
}

// Active reports whether the lines seen so far leave a goal that may be running.
func (t GoalTracker) Active() bool { return t.active }

// Observe applies one transcript line.
func (t *GoalTracker) Observe(line []byte, harness string) {
	switch harness {
	case HarnessCodex:
		t.observeCodex(line)
	case HarnessClaude:
		t.observeClaude(line)
	}
}

// Read applies every complete line from r and returns the bytes consumed. A
// partial last line is left for the next read, from the same position.
func (t *GoalTracker) Read(r io.Reader, harness string) (int64, error) {
	reader := bufio.NewReaderSize(r, 1<<20)
	var consumed int64
	for {
		line, err := reader.ReadBytes('\n')
		if err == io.EOF {
			return consumed, nil
		}
		if err != nil {
			return consumed, err
		}
		consumed += int64(len(line))
		t.Observe(line, harness)
	}
}

func (t *GoalTracker) observeCodex(line []byte) {
	if bytes.Contains(line, codexGoalUpdateNeedle) {
		update, ok, malformed := parseCodexGoalUpdate(line)
		switch {
		case malformed:
			t.hold()
		case !ok:
		case !codexGoalFinished(update.Status):
			t.start(update.Objective, update.ThreadID)
		case update.Envelope == "event_msg":
			t.finish(update.Objective, update.ThreadID)
		}
		return
	}
	if bytes.Contains(line, codexGoalToolNeedle) {
		t.observeCodexGoalTool(line)
	}
}

// observeCodexGoalTool handles the typed goal tools: direct calls and exec
// scripts that name them. Shell commands quoting the same text are not calls.
func (t *GoalTracker) observeCodexGoalTool(line []byte) {
	var envelope struct {
		Type    string `json:"type"`
		Payload struct {
			Type      string          `json:"type"`
			Name      string          `json:"name"`
			Input     string          `json:"input"`
			Arguments json.RawMessage `json:"arguments"`
		} `json:"payload"`
	}
	if json.Unmarshal(line, &envelope) != nil || envelope.Type != "response_item" {
		return
	}
	p := envelope.Payload
	name := strings.TrimPrefix(strings.TrimPrefix(p.Name, "functions."), "tools.")
	switch {
	case p.Type == "custom_tool_call" && name == "exec":
		if strings.Contains(p.Input, "tools.create_goal(") {
			t.start("", "")
		} else if strings.Contains(p.Input, "tools.update_goal(") && !codexScriptOnlyFinishes(p.Input) {
			t.hold()
		}
	case p.Type == "function_call" && name == "create_goal":
		t.start("", "")
	case p.Type == "function_call" && name == "update_goal":
		arguments := []byte(p.Arguments)
		var encoded string
		if json.Unmarshal(arguments, &encoded) == nil {
			arguments = []byte(encoded)
		}
		var request struct {
			Status string `json:"status"`
		}
		if json.Unmarshal(arguments, &request) != nil || !codexGoalFinished(request.Status) {
			t.hold()
		}
	}
}

func codexScriptOnlyFinishes(input string) bool {
	matches := codexGoalToolStatusArg.FindAllStringSubmatch(input, -1)
	return len(matches) > 0 && !slices.ContainsFunc(matches, func(match []string) bool { return !codexGoalFinished(match[1]) })
}

func (t *GoalTracker) observeClaude(line []byte) {
	if !bytes.Contains(line, claudeGoalNeedle) {
		return
	}
	statuses, ok := ParseClaudeGoalStatuses(line)
	if !ok {
		t.hold()
		return
	}
	for _, status := range statuses {
		if !status.Met {
			t.start(status.Condition, "")
			return
		}
	}
	for _, status := range statuses {
		t.finish(status.Condition, "")
	}
}

// start records a new goal, replacing what was tracked.
func (t *GoalTracker) start(objective, thread string) {
	*t = GoalTracker{active: true, objective: objective, thread: thread}
}

// hold marks a goal active without changing which goal is tracked.
func (t *GoalTracker) hold() { t.active = true }

func (t *GoalTracker) finish(objective, thread string) {
	if t.objective != "" && objective != t.objective {
		return
	}
	if t.thread != "" && thread != "" && thread != t.thread {
		return
	}
	*t = GoalTracker{}
}

// LastContextTokens finds the request size of the newest response in buf:
// Claude's main-thread assistant usage (input plus cache reads and writes) or
// Codex's last_token_usage input. A report of zero is not a reading, so the
// search carries on to the one before it.
func LastContextTokens(buf []byte, harness string) (int64, bool) {
	lines := bytes.Split(buf, []byte("\n"))
	for i := len(lines) - 1; i >= 0; i-- {
		var tokens int64
		switch harness {
		case HarnessClaude:
			tokens = claudeContextTokens(lines[i])
		case HarnessCodex:
			tokens = codexContextTokens(lines[i])
		}
		if tokens > 0 {
			return tokens, true
		}
	}
	return 0, false
}

func claudeContextTokens(line []byte) int64 {
	if !bytes.Contains(line, []byte(`"usage"`)) || !bytes.Contains(line, []byte(`"assistant"`)) {
		return 0
	}
	var entry struct {
		Type        string `json:"type"`
		IsSidechain bool   `json:"isSidechain"`
		Message     struct {
			Usage struct {
				Input  int64 `json:"input_tokens"`
				Read   int64 `json:"cache_read_input_tokens"`
				Create int64 `json:"cache_creation_input_tokens"`
			} `json:"usage"`
		} `json:"message"`
	}
	if json.Unmarshal(line, &entry) != nil || entry.Type != "assistant" || entry.IsSidechain {
		return 0
	}
	return entry.Message.Usage.Input + entry.Message.Usage.Read + entry.Message.Usage.Create
}

func codexContextTokens(line []byte) int64 {
	if !bytes.Contains(line, []byte(`"token_count"`)) {
		return 0
	}
	var entry struct {
		Payload struct {
			Type string `json:"type"`
			Info struct {
				Last struct {
					Input int64 `json:"input_tokens"`
				} `json:"last_token_usage"`
			} `json:"info"`
		} `json:"payload"`
	}
	if json.Unmarshal(line, &entry) != nil || entry.Payload.Type != "token_count" {
		return 0
	}
	return entry.Payload.Info.Last.Input
}

// CompactionRecorded reports whether data holds a complete compaction record:
// Claude's system compact_boundary or Codex's compacted entry. A line cut off
// by a read limit or a write in progress does not count.
func CompactionRecorded(data []byte, harness string) bool {
	var needle []byte
	switch harness {
	case HarnessClaude:
		needle = []byte(`"compact_boundary"`)
	case HarnessCodex:
		needle = []byte(`"compacted"`)
	default:
		return false
	}
	for _, line := range bytes.Split(data, []byte("\n")) {
		if !bytes.Contains(line, needle) {
			continue
		}
		var entry struct {
			Type    string `json:"type"`
			Subtype string `json:"subtype"`
		}
		if json.Unmarshal(line, &entry) != nil {
			continue
		}
		if (harness == HarnessClaude && entry.Type == "system" && entry.Subtype == "compact_boundary") ||
			(harness == HarnessCodex && entry.Type == "compacted") {
			return true
		}
	}
	return false
}
