package agenttranscript

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// goalFixture is one line in testdata/goal_lines.json. Lifecycle is what Agent
// Manager's marker parser reads (its own test checks that); Compaction is what a
// fresh GoalTracker holds after the line, or "none" when the line changes nothing.
type goalFixture struct {
	Name       string `json:"name"`
	Harness    string `json:"harness"`
	Line       string `json:"line"`
	Lifecycle  string `json:"lifecycle"`
	Compaction string `json:"compaction"`
}

func loadGoalFixtures(t *testing.T) []goalFixture {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "goal_lines.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []goalFixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no goal fixtures")
	}
	return fixtures
}

func TestGoalTrackerOnEachFixtureLine(t *testing.T) {
	for _, fixture := range loadGoalFixtures(t) {
		t.Run(fixture.Name, func(t *testing.T) {
			// Starting from an active goal shows whether the line clears it;
			// starting clean shows whether it starts one.
			var clean GoalTracker
			clean.Observe([]byte(fixture.Line), fixture.Harness)
			held := GoalTracker{active: true}
			held.Observe([]byte(fixture.Line), fixture.Harness)
			got := "none"
			switch {
			case clean.Active():
				got = "active"
			case !held.Active():
				got = "inactive"
			}
			if got != fixture.Compaction {
				t.Fatalf("compaction reading = %s, want %s", got, fixture.Compaction)
			}
		})
	}
}

func codexGoalLine(objective, thread, status string) string {
	line, _ := json.Marshal(map[string]any{"type": "event_msg", "payload": map[string]any{
		"type": "thread_goal_updated", "threadId": thread,
		"goal": map[string]string{"threadId": thread, "objective": objective, "status": status},
	}})
	return string(line)
}

func claudeGoalLine(condition string, met bool) string {
	line, _ := json.Marshal(map[string]any{"type": "attachment", "attachment": map[string]any{"type": "goal_status", "condition": condition, "met": met}})
	return string(line)
}

func codexScript(script string) string {
	line, _ := json.Marshal(map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "name": "exec", "call_id": "c", "input": script}})
	return string(line)
}

// The sequences a real session records, one step at a time.
func TestGoalTrackerSequences(t *testing.T) {
	createGoal := codexScript(`const g = await tools.create_goal({objective:"ship"}); text(JSON.stringify(g));`)
	complete := codexScript(`const r = await tools.update_goal({status:"complete"}); text(JSON.stringify(r));`)
	blocked := codexScript(`const r = await tools.update_goal({status:"blocked"}); text(JSON.stringify(r));`)
	for _, tc := range []struct {
		name    string
		harness string
		lines   []string
		active  bool
	}{
		{"no goal", HarnessCodex, []string{`{"type":"event_msg","payload":{"type":"task_started"}}`}, false},
		{"active then paused", HarnessCodex, []string{codexGoalLine("ship", "t1", "active"), codexGoalLine("ship", "t1", "paused")}, false},
		{"paused then active", HarnessCodex, []string{codexGoalLine("ship", "t1", "paused"), codexGoalLine("ship", "t1", "active")}, true},
		{"late status for an older objective", HarnessCodex, []string{codexGoalLine("old", "t1", "active"), codexGoalLine("new", "t1", "active"), codexGoalLine("old", "t1", "complete")}, true},
		{"finished status from another thread", HarnessCodex, []string{codexGoalLine("ship", "t1", "active"), codexGoalLine("ship", "t2", "paused")}, true},
		{"finished status naming no objective", HarnessCodex, []string{codexGoalLine("ship", "t1", "active"), codexGoalLine("", "t1", "paused")}, true},
		{"unknown status after paused", HarnessCodex, []string{codexGoalLine("ship", "t1", "paused"), codexGoalLine("ship", "t1", "pursuing")}, true},
		{"malformed line does not lose an active goal", HarnessCodex, []string{codexGoalLine("ship", "t1", "active"), `{"payload":{"type":"thread_goal_updated","goal":{"status":"paused"`}, true},
		{"goal created by tool", HarnessCodex, []string{createGoal}, true},
		{"tool goal paused by event", HarnessCodex, []string{createGoal, codexGoalLine("ship", "t1", "paused")}, false},
		{"tool completion does not clear", HarnessCodex, []string{codexGoalLine("ship", "t1", "active"), complete}, true},
		{"tool completion alone starts nothing", HarnessCodex, []string{complete}, false},
		{"blocked tool call after paused", HarnessCodex, []string{codexGoalLine("ship", "t1", "paused"), blocked}, true},
		{"blocked tool call keeps the goal tracked", HarnessCodex, []string{codexGoalLine("ship", "t1", "active"), blocked, codexGoalLine("ship", "t1", "paused")}, false},
		{"script with two statuses", HarnessCodex, []string{codexScript(`await tools.update_goal({status:"complete"}); await tools.update_goal({status:"active"});`)}, true},
		{"script with no literal status", HarnessCodex, []string{codexScript(`await tools.update_goal(next);`)}, true},
		{"claude met", HarnessClaude, []string{claudeGoalLine("tests", false), claudeGoalLine("tests", true)}, false},
		{"claude cleared with the goal condition", HarnessClaude, []string{claudeGoalLine("tests", false), `{"type":"attachment","attachment":{"type":"goal_status","met":true,"sentinel":true,"condition":"tests"}}`}, false},
		{"claude met for another condition", HarnessClaude, []string{claudeGoalLine("new", false), claudeGoalLine("old", true)}, true},
		{"claude met with no condition", HarnessClaude, []string{claudeGoalLine("tests", false), `{"type":"attachment","attachment":{"type":"goal_status","met":true}}`}, true},
		{"claude failed stays held", HarnessClaude, []string{`{"type":"attachment","attachment":{"type":"goal_status","met":false,"failed":true,"condition":"tests"}}`}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var tracker GoalTracker
			for _, line := range tc.lines {
				tracker.Observe([]byte(line), tc.harness)
			}
			if tracker.Active() != tc.active {
				t.Fatalf("Active = %v, want %v", tracker.Active(), tc.active)
			}
		})
	}
}

func TestGoalTrackerReadLeavesAPartialLastLine(t *testing.T) {
	complete := codexGoalLine("ship", "t1", "active") + "\n"
	partial := codexGoalLine("ship", "t1", "paused")
	var tracker GoalTracker
	consumed, err := tracker.Read(strings.NewReader(complete+partial), HarnessCodex)
	if err != nil {
		t.Fatal(err)
	}
	if consumed != int64(len(complete)) || !tracker.Active() {
		t.Fatalf("consumed %d of %d, active=%v", consumed, len(complete), tracker.Active())
	}
	consumed, err = tracker.Read(strings.NewReader(partial+"\n"), HarnessCodex)
	if err != nil {
		t.Fatal(err)
	}
	if consumed != int64(len(partial))+1 || tracker.Active() {
		t.Fatalf("consumed %d, active=%v after the line completed", consumed, tracker.Active())
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, os.ErrClosed }

func TestGoalTrackerReadReportsAReadFailure(t *testing.T) {
	var tracker GoalTracker
	if _, err := tracker.Read(failingReader{}, HarnessCodex); err == nil {
		t.Fatal("a failed read must be reported, not read as an empty transcript")
	}
}

func TestParseClaudeGoalStatusesIsDeterministic(t *testing.T) {
	line := []byte(`{"z":{"type":"goal_status","condition":"later","met":true},"a":{"type":"goal_status","condition":"first","met":false,"iterations":2,"reason":" r "}}`)
	for range 50 {
		statuses, ok := ParseClaudeGoalStatuses(line)
		if !ok || len(statuses) != 2 || statuses[0].Condition != "first" || statuses[0].Iteration != 2 || statuses[0].Reason != "r" || statuses[1].Condition != "later" {
			t.Fatalf("statuses = %+v, ok=%v", statuses, ok)
		}
	}
	if _, ok := ParseClaudeGoalStatuses([]byte(`{"type":`)); ok {
		t.Fatal("a line that is not JSON must report not ok")
	}
}

func TestLastContextTokens(t *testing.T) {
	claude := func(kind string, sidechain bool, input, read, create int) string {
		line, _ := json.Marshal(map[string]any{"type": kind, "isSidechain": sidechain, "message": map[string]any{"role": "assistant", "usage": map[string]int{
			"input_tokens": input, "cache_read_input_tokens": read, "cache_creation_input_tokens": create,
		}}})
		return string(line)
	}
	codex := func(input int) string {
		line, _ := json.Marshal(map[string]any{"type": "event_msg", "payload": map[string]any{"type": "token_count", "info": map[string]any{"last_token_usage": map[string]int{"input_tokens": input}}}})
		return string(line)
	}
	for _, tc := range []struct {
		name    string
		harness string
		lines   []string
		tokens  int64
		found   bool
	}{
		{"claude sums usage", HarnessClaude, []string{claude("assistant", false, 1, 2, 3)}, 6, true},
		{"claude newest wins", HarnessClaude, []string{claude("assistant", false, 1, 0, 0), claude("assistant", false, 5, 0, 0)}, 5, true},
		{"claude skips a sidechain", HarnessClaude, []string{claude("assistant", false, 7, 0, 0), claude("assistant", true, 9, 0, 0)}, 7, true},
		{"claude ignores other entry types", HarnessClaude, []string{claude("user", false, 9, 0, 0)}, 0, false},
		{"codex newest wins", HarnessCodex, []string{codex(10), codex(20)}, 20, true},
		{"codex zero report is not a reading", HarnessCodex, []string{codex(10), codex(0)}, 10, true},
		{"codex null info", HarnessCodex, []string{`{"payload":{"type":"token_count","info":null}}`}, 0, false},
		{"codex cut-off first line", HarnessCodex, []string{`_count","info":{"last_token_usage":{"input_tokens":3}}}}`, codex(4)}, 4, true},
		{"nothing", HarnessCodex, nil, 0, false},
		{"unknown harness", "grok", []string{codex(10)}, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tokens, found := LastContextTokens([]byte(strings.Join(tc.lines, "\n")+"\n"), tc.harness)
			if tokens != tc.tokens || found != tc.found {
				t.Fatalf("got %d,%v want %d,%v", tokens, found, tc.tokens, tc.found)
			}
		})
	}
}

func TestCompactionRecorded(t *testing.T) {
	for _, tc := range []struct {
		name    string
		harness string
		data    string
		want    bool
	}{
		{"claude boundary", HarnessClaude, `{"type":"system","subtype":"compact_boundary","compactMetadata":{"trigger":"manual"}}` + "\n", true},
		{"claude cut-off boundary", HarnessClaude, `{"type":"system","subtype":"compact_boundary"`, false},
		{"claude word in a tool result", HarnessClaude, `{"type":"user","toolUseResult":{"subtype":"compact_boundary"}}` + "\n", false},
		{"claude prose", HarnessClaude, `{"type":"assistant","text":"\"compact_boundary\""}` + "\n", false},
		{"codex entry", HarnessCodex, `{"timestamp":"t","type":"compacted","payload":{"message":""}}` + "\n", true},
		{"codex entry with spacing", HarnessCodex, `{"type": "compacted", "payload": {}}` + "\n", true},
		{"codex nested type", HarnessCodex, `{"type":"event_msg","payload":{"type":"compacted"}}` + "\n", false},
		{"unknown harness", "grok", `{"type":"compacted"}` + "\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := CompactionRecorded([]byte(tc.data), tc.harness); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
