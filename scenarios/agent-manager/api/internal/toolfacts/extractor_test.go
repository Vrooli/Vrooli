package toolfacts_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"

	"agent-manager/internal/domain"
	"agent-manager/internal/toolfacts"
)

// trace builds tool events the way a harness records them, oldest first.
type trace struct {
	run    uuid.UUID
	events []*domain.RunEvent
	next   int
}

func newTrace() *trace { return &trace{run: uuid.New()} }

func (t *trace) call(tool string, input map[string]any) string {
	t.next++
	id := fmt.Sprintf("call-%d", t.next)
	t.callWithID(tool, id, input)
	return id
}

func (t *trace) callWithID(tool, id string, input map[string]any) {
	t.events = append(t.events, domain.NewToolCallEvent(t.run, tool, id, input))
}

func (t *trace) result(tool, id, output string, exit *int, success bool) {
	event := domain.NewToolResultEvent(t.run, tool, id, output, nil)
	data := event.Data.(*domain.ToolResultEventData)
	data.ExitCode, data.Success = exit, success
	if !success {
		data.Error = "exit status"
	}
	t.events = append(t.events, event)
}

func (t *trace) shell(command, output string, exit int) {
	id := t.call("bash", map[string]any{"command": command})
	t.result("bash", id, output, &exit, exit == 0)
}

// codeMode records a Codex code-mode call: the command is a literal inside
// JavaScript, with no command or cmd field on the tool input, and the result is
// the script's own with the wall time the harness prints.
func (t *trace) codeMode(command, output string, wall float64) {
	script := fmt.Sprintf(`const r = await tools.exec_command({cmd:%q}); text(r.output)`, command)
	id := t.call("exec", map[string]any{"input": script})
	event := domain.NewToolResultEvent(t.run, "exec", id, "", nil)
	data := event.Data.(*domain.ToolResultEventData)
	data.OutputBlocks = []domain.ToolOutputBlock{
		{Type: "input_text", Text: fmt.Sprintf("Script completed\nWall time %.1f seconds\nOutput:\n", wall)},
		{Type: "input_text", Text: output},
	}
	t.events = append(t.events, event)
}

func (t *trace) read(path, content string) {
	id := t.call("Read", map[string]any{"file_path": path})
	t.result("Read", id, content, nil, true)
}

func extract(t *testing.T, events []*domain.RunEvent) toolfacts.Facts {
	t.Helper()
	rules, err := toolfacts.DefaultRules()
	if err != nil {
		t.Fatal(err)
	}
	return extractWith(rules, events)
}

func extractWith(rules toolfacts.Rules, events []*domain.RunEvent) toolfacts.Facts {
	extractor := toolfacts.New(rules)
	for _, event := range events {
		extractor.Add(event)
	}
	return extractor.Facts()
}

func TestCodeModeCommandsAreCommandsAndTheirTargetsPaths(t *testing.T) {
	tr := newTrace()
	for range 3 {
		tr.codeMode("cat /repo/api/run.go", "package api", 0.1)
	}
	tr.codeMode("go test /repo/api", "ok", 2.3)
	tr.codeMode("go test /repo/api", "ok", 2.4)

	facts := extract(t, tr.events)
	if facts.ToolCalls != 5 || facts.Commands != 5 {
		t.Fatalf("calls = %d, commands = %d, want 5 and 5: a code-mode script runs the command in its literal", facts.ToolCalls, facts.Commands)
	}
	// cat twice more and go test once more.
	if facts.RepeatedCommands != 3 {
		t.Fatalf("repeated commands = %d, want 3", facts.RepeatedCommands)
	}
	// /repo/api/run.go read three times, /repo/api tested twice.
	if facts.RepeatedPaths != 3 {
		t.Fatalf("repeated paths = %d, want 3", facts.RepeatedPaths)
	}
}

func TestUnchangedRepeats(t *testing.T) {
	failing := 1
	cases := []struct {
		name          string
		build         func(*trace)
		repeats, runs int
	}{
		{"a polling loop whose only difference is the wall time", func(tr *trace) {
			for i := range 5 {
				tr.codeMode("agent-manager run get 1234", "state: running", 0.1*float64(i+1))
			}
		}, 4, 4},
		{"the same command with a changed result is progress, not a repeat", func(tr *trace) {
			for i := range 5 {
				tr.shell("go test ./...", fmt.Sprintf("FAIL: %d tests", 5-i), 1)
			}
		}, 0, 0},
		{"a result that differs only by its clock time is the same result", func(tr *trace) {
			for i := range 4 {
				tr.shell("date -u; systemctl is-active x", fmt.Sprintf("Fri Nov 20 18:5%d:35 UTC 2026\nactive", i), 0)
			}
		}, 3, 3},
		{"two alternating checks repeat independently", func(tr *trace) {
			for range 3 {
				tr.shell("check a", "ok a", 0)
				tr.shell("check b", "ok b", 0)
			}
		}, 4, 2},
		{"a retry loop that fails the same way is a repeat", func(tr *trace) {
			for range 4 {
				id := tr.call("bash", map[string]any{"command": "curl -sf http://x/health"})
				tr.result("bash", id, "", &failing, false)
			}
		}, 3, 3},
		{"a file read again after an edit is not a repeat", func(tr *trace) {
			tr.read("/repo/a.go", "v1")
			tr.read("/repo/a.go", "v1")
			tr.read("/repo/a.go", "v2")
			tr.read("/repo/a.go", "v2")
		}, 2, 1},
		{"the same call far apart is not the same loop", func(tr *trace) {
			tr.shell("check a", "ok", 0)
			for i := range 70 {
				tr.shell(fmt.Sprintf("work %d", i), "done", 0)
			}
			tr.shell("check a", "ok", 0)
		}, 0, 0},
		{"a call that was never answered has no identity", func(tr *trace) {
			for range 3 {
				tr.call("bash", map[string]any{"command": "sleep 5"})
			}
		}, 0, 0},
		{"a script with no literal command is identified by its input", func(tr *trace) {
			for range 3 {
				id := tr.call("exec", map[string]any{"input": "const r = await tools.write_stdin({session_id: 4}); text(r.output)"})
				tr.result("exec", id, "still running", nil, true)
			}
		}, 2, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tr := newTrace()
			tc.build(tr)
			facts := extract(t, tr.events)
			if facts.UnchangedRepeats != tc.repeats || facts.LongestUnchangedRun != tc.runs {
				t.Fatalf("unchanged repeats = %d (longest %d), want %d (longest %d)", facts.UnchangedRepeats, facts.LongestUnchangedRun, tc.repeats, tc.runs)
			}
		})
	}
}

func TestNormalizeRulesAreData(t *testing.T) {
	tr := newTrace()
	for i := range 3 {
		tr.shell(fmt.Sprintf("sleep %d; cat /var/state", 10*(i+1)), "idle", 0)
	}
	if got := extract(t, tr.events).UnchangedRepeats; got != 0 {
		t.Fatalf("default rules have no sleep rule, repeats = %d, want 0", got)
	}
	rules, err := toolfacts.ParseRules([]byte(`{
		"unchanged_window": 64, "chunk_min_events": 8, "chunk_max_events": 64, "max_line_bytes": 4096,
		"result_head_bytes": 1024, "result_tail_bytes": 256, "path_keys": [],
		"normalize": [{"id": "sleep", "applies": "args", "hint": "sleep", "pattern": "sleep [0-9]+", "replace": "sleep N"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := extractWith(rules, tr.events).UnchangedRepeats; got != 2 {
		t.Fatalf("with a sleep rule supplied as data, repeats = %d, want 2", got)
	}
}

func TestWholeRunIsCoveredWithBoundedMemory(t *testing.T) {
	tr := newTrace()
	for i := range 6000 {
		tr.shell(fmt.Sprintf("cat /repo/file%d.go", i%100), fmt.Sprintf("body %d", i), 0)
	}
	facts := extract(t, tr.events)
	if facts.ToolCalls != 6000 {
		t.Fatalf("calls = %d, want the whole run (6000), not a recent window", facts.ToolCalls)
	}
	if facts.RepeatedPaths != 5900 || facts.RepeatedCommands != 5900 {
		t.Fatalf("repeated paths = %d, commands = %d, want 5900 each (100 distinct)", facts.RepeatedPaths, facts.RepeatedCommands)
	}
	if facts.UnchangedRepeats != 0 {
		t.Fatalf("unchanged repeats = %d, want 0: every result differs", facts.UnchangedRepeats)
	}
}

func TestChunkingChangesNothing(t *testing.T) {
	failing := 2
	tr := newTrace()
	for i := range 400 {
		switch i % 7 {
		case 0:
			tr.codeMode("agent-manager run get abc", "running", 0.1)
		case 1:
			tr.shell("go build ./...", "", 0)
		case 2:
			id := tr.call("bash", map[string]any{"command": "go vet ./pkg"})
			tr.result("bash", id, "vet: bad", &failing, false)
		case 3:
			tr.read(fmt.Sprintf("/repo/f%d.go", i%5), "same")
		case 4:
			tr.shell("cat /repo/a.go | head -20", "package a", 0)
		default:
			tr.shell(fmt.Sprintf("echo %d", i), fmt.Sprintf("%d", i), 0)
		}
	}
	rules, err := toolfacts.DefaultRules()
	if err != nil {
		t.Fatal(err)
	}
	whole := rules
	whole.ChunkMinEvents, whole.ChunkMaxEvents = 1_000_000, 1_000_000
	tiny := rules
	tiny.ChunkMinEvents, tiny.ChunkMaxEvents = 1, 4096
	want, got := extractWith(whole, tr.events), extractWith(tiny, tr.events)
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("one chunk and one-event chunks disagree:\n whole %+v\n tiny  %+v", want, got)
	}
	if want.FailedCommands != 57 || want.ToolFailures != 57 {
		t.Fatalf("failed commands = %d, tool failures = %d, want 57 each", want.FailedCommands, want.ToolFailures)
	}
}

func TestReusedCallIDsAreSeparateCalls(t *testing.T) {
	failing := 2
	tr := newTrace()
	// Codex exec numbers its items per turn: item_1 is a different command each
	// turn.
	tr.callWithID("bash", "item_1", map[string]any{"command": "ls /a"})
	tr.result("bash", "item_1", "a", nil, true)
	tr.callWithID("bash", "item_1", map[string]any{"command": "make lint"})
	tr.result("bash", "item_1", "", &failing, false)
	// A call still in flight is reported twice (started, then completed): one call.
	tr.callWithID("bash", "item_2", map[string]any{"command": "make test"})
	tr.callWithID("bash", "item_2", map[string]any{"command": "make test"})
	tr.result("bash", "item_2", "ok", nil, true)

	facts := extract(t, tr.events)
	if facts.ToolCalls != 3 {
		t.Fatalf("calls = %d, want 3 (a reused id is a new call, a progress update is not)", facts.ToolCalls)
	}
	if facts.FailedCommands != 1 || facts.ToolFailures != 1 {
		t.Fatalf("failed commands = %d, tool failures = %d, want 1 each: the reused id's failure must not be collapsed", facts.FailedCommands, facts.ToolFailures)
	}
}

func TestUnansweredCallsDoNotGrowMemory(t *testing.T) {
	tr := newTrace()
	for i := range 20000 {
		tr.call("bash", map[string]any{"command": fmt.Sprintf("sleep %d", i)})
	}
	rules, err := toolfacts.DefaultRules()
	if err != nil {
		t.Fatal(err)
	}
	extractor := toolfacts.New(rules)
	for _, event := range tr.events {
		extractor.Add(event)
	}
	if got := extractor.Facts().ToolCalls; got != 20000 {
		t.Fatalf("calls = %d, want 20000", got)
	}
	if held := extractor.Pending(); held > rules.ChunkMaxEvents {
		t.Fatalf("holds %d events, want at most the chunk bound %d", held, rules.ChunkMaxEvents)
	}
}

func TestFailedToolsAndLastErrorAreReported(t *testing.T) {
	tr := newTrace()
	id := tr.call("Write", map[string]any{"file_path": "/x"})
	event := domain.NewToolResultEvent(tr.run, "Write", id, "", fmt.Errorf("%s", strings.Repeat("e", 600)))
	tr.events = append(tr.events, event)
	facts := extract(t, tr.events)
	if facts.ToolFailures != 1 || len(facts.FailedTools) != 1 || facts.FailedTools[0] != "Write" {
		t.Fatalf("facts = %+v, want one failure of Write", facts)
	}
	if len(facts.LastError) != 400 {
		t.Fatalf("last error is %d bytes, want it bounded to 400", len(facts.LastError))
	}
}

func TestTailWindowsDescribeTheEndOfTheRun(t *testing.T) {
	// A long run that polled early and then did distinct work, and a short run
	// that ends in a loop: the whole-run totals match, the tails do not.
	early := newTrace()
	for range 12 {
		early.shell("check gate", "closed", 0)
	}
	for i := range 300 {
		early.shell(fmt.Sprintf("work %d", i), "done", 0)
	}
	late := newTrace()
	for i := range 300 {
		late.shell(fmt.Sprintf("work %d", i), "done", 0)
	}
	for range 12 {
		late.shell("check gate", "closed", 0)
	}
	rules, err := toolfacts.DefaultRules()
	if err != nil {
		t.Fatal(err)
	}
	for name, tc := range map[string]struct {
		trace                      *trace
		tail64, tail200, wholeRuns int
	}{
		"polled early":     {early, 0, 0, 11},
		"loops at the end": {late, 11, 11, 11},
	} {
		facts := extractWith(rules, tc.trace.events)
		if facts.UnchangedRepeats != tc.wholeRuns || len(facts.Tails) != 2 {
			t.Fatalf("%s: whole-run unchanged repeats = %d, tails = %d", name, facts.UnchangedRepeats, len(facts.Tails))
		}
		if facts.Tails[0].Window != 64 || facts.Tails[0].UnchangedRepeats != tc.tail64 || facts.Tails[1].Window != 200 || facts.Tails[1].UnchangedRepeats != tc.tail200 {
			t.Fatalf("%s: tails = %+v, want unchanged repeats %d (64) and %d (200)", name, facts.Tails, tc.tail64, tc.tail200)
		}
	}
	vars := extractWith(rules, late.events).Activation(rules)
	if vars["tail64_longest_unchanged_run"] != int64(11) || vars["tail200_repeated_commands"] != int64(11) {
		t.Fatalf("variables = %v", vars)
	}
	declared := toolfacts.Declarations(rules)
	if len(declared) != len(vars) {
		t.Fatalf("Declarations lists %d variables, Activation returns %d", len(declared), len(vars))
	}
	for _, variable := range declared {
		_, isText := vars[variable.Name].(string)
		if _, ok := vars[variable.Name]; !ok || isText != (variable.Type == "string") {
			t.Fatalf("declaration %+v does not match the activation value %v", variable, vars[variable.Name])
		}
	}
}

func TestShortRunsHaveShortTails(t *testing.T) {
	tr := newTrace()
	tr.shell("a", "x", 0)
	tr.shell("a", "x", 0)
	facts := extract(t, tr.events)
	if facts.Tails[0].Calls != 2 || facts.Tails[0].UnchangedRepeats != 1 || facts.Tails[1].Calls != 2 {
		t.Fatalf("tails = %+v", facts.Tails)
	}
}

func TestWaitCallsAreLeftOutOfNonWaitRepeats(t *testing.T) {
	tr := newTrace()
	for range 5 {
		tr.shell("test-genie runs wait --json", "running", 0)
	}
	for range 3 {
		tr.shell("go build ./...", "ok", 0)
	}
	// A status lookup of git is a check on the tree, not a wait; --help is a lookup.
	for range 2 {
		tr.shell("git status --short", "", 0)
		tr.shell("agent-manager run get --help", "usage", 0)
	}
	facts := extract(t, tr.events)
	if facts.WaitCalls != 5 {
		t.Fatalf("wait calls = %d, want 5 (git status and --help are not waits)", facts.WaitCalls)
	}
	// 4 + 2 + 1 + 1 repeats in all; the wait loop's 4 are set aside.
	if facts.UnchangedRepeats != 8 || facts.NonWaitUnchangedRepeats != 4 || facts.NonWaitLongestUnchangedRun != 2 || facts.LongestUnchangedRun != 4 {
		t.Fatalf("unchanged = %d (longest %d), non-wait = %d (longest %d), want 8 (4) and 4 (2)",
			facts.UnchangedRepeats, facts.LongestUnchangedRun, facts.NonWaitUnchangedRepeats, facts.NonWaitLongestUnchangedRun)
	}
}

func TestParkCyclesAndTheirKeys(t *testing.T) {
	tr := newTrace()
	for i := range 4 {
		tr.shell("agent-manager effort lint /work/delivery", "ok", 0)
		tr.shell(fmt.Sprintf("agent-manager effort park /work/delivery --run run-1 --json # %d", i), "parked", 0)
		tr.shell(fmt.Sprintf("work %d", i), "done", 0)
	}
	tr.shell("agent-manager run park run-1 --producer children --key child-a", "parked", 0)
	tr.shell("agent-manager run park run-1 --producer children --key child-b", "parked", 0)
	tr.shell("agent-manager effort park --help", "usage", 0)
	facts := extract(t, tr.events)
	if facts.ParkCalls != 6 {
		t.Fatalf("park calls = %d, want 6 (--help is not a park)", facts.ParkCalls)
	}
	if facts.LongestSameParkStreak != 4 {
		t.Fatalf("longest same-key park streak = %d, want 4: the work between the parks does not break it", facts.LongestSameParkStreak)
	}
}

func TestGoalUpdatesAreCountedByStatus(t *testing.T) {
	tr := newTrace()
	update := func(status string) {
		script := fmt.Sprintf(`const r = await tools.update_goal({status:%q}); text(r)`, status)
		id := tr.call("exec", map[string]any{"input": script})
		tr.result("exec", id, "ok", nil, true)
	}
	tr.shell("go test ./...", "ok", 0)
	update("active")
	tr.shell("go build ./...", "ok", 0)
	update("blocked")
	// A function-call harness passes the status as an input field.
	id := tr.call("functions.update_goal", map[string]any{"status": "Blocked"})
	tr.result("update_goal", id, "ok", nil, true)
	// Mentioning the tool in a command or in unrelated code is not an update.
	tr.shell("rg update_goal docs/", "docs/x.md", 0)
	id = tr.call("exec", map[string]any{"input": "const r = await tools.exec_command({cmd:'ls'}); text(r.output)"})
	tr.result("exec", id, "a", nil, true)
	// A script that updates the goal with a status it computes still counts as an update.
	id = tr.call("exec", map[string]any{"input": "const r = await tools.update_goal(next); text(r)"})
	tr.result("exec", id, "ok", nil, true)

	facts := extract(t, tr.events)
	if facts.GoalUpdates != 4 || facts.GoalStatuses["blocked"] != 2 || facts.GoalStatuses["active"] != 1 {
		t.Fatalf("goal updates = %d, statuses = %v, want 4 with blocked 2 and active 1", facts.GoalUpdates, facts.GoalStatuses)
	}
	if facts.LastGoalStatus != "" || facts.CallsAfterLastGoalUpdate != 0 {
		t.Fatalf("last status %q, calls after %d: the last update has no literal status and nothing follows it", facts.LastGoalStatus, facts.CallsAfterLastGoalUpdate)
	}
	rules, _ := toolfacts.DefaultRules()
	ends := newTrace()
	ends.shell("make audit", "FAIL gate", 1)
	script := `const r = await tools.update_goal({status:"blocked"}); text(r)`
	id = ends.call("exec", map[string]any{"input": script})
	ends.result("exec", id, "ok", nil, true)
	ends.shell("echo done", "done", 0)
	got := extractWith(rules, ends.events).Activation(rules)
	if got["last_goal_status"] != "blocked" || got["calls_after_last_goal_update"] != int64(1) || got["goal_blocked"] != int64(1) || got["goal_complete"] != int64(0) {
		t.Fatalf("activation = %v", got)
	}
}

func TestToolSourceIsAFactForTheRules(t *testing.T) {
	rules, _ := toolfacts.DefaultRules()
	none := toolfacts.Facts{Source: toolfacts.SourceNone}.Activation(rules)
	if none["tool_source"] != "none" || none["tool_calls"] != int64(0) {
		t.Fatalf("activation = %v", none)
	}
}
