package runsignal

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// execStream builds Codex exec calls, their outputs and nested command
// results (the shape the codec emits for rollout CommandExecution items).
type execStream struct {
	run    uuid.UUID
	events []*domain.RunEvent
}

func (s *execStream) exec(id string, commands ...string) *execStream {
	src := ""
	for _, command := range commands {
		encoded, _ := json.Marshal(command)
		src += "await tools.exec_command({cmd:" + string(encoded) + "});\n"
	}
	s.events = append(s.events, domain.NewToolCallEvent(s.run, "exec", id, map[string]any{"input": src}))
	return s
}

func (s *execStream) output(id string) *execStream {
	s.events = append(s.events, domain.NewToolResultEvent(s.run, "", id, "Script completed\nOutput:\n...", nil))
	return s
}

func (s *execStream) ran(command string, exit int, outputBytes int) *execStream {
	var err error
	if exit != 0 {
		err = errors.New("command exited with code " + fmt.Sprint(exit))
	}
	event := domain.NewToolResultEvent(s.run, NestedCommandTool, "exec-"+uuid.NewString(), "out", err)
	data := event.Data.(*domain.ToolResultEventData)
	known, duration := true, int64(40)
	data.ExitCode, data.DurationMS, data.OutcomeKnown = &exit, &duration, &known
	record, _ := json.Marshal(map[string]any{"type": "CommandExecution", "command": []string{"/bin/bash", "-lc", command}, "status": "completed", "exit_code": exit, "output_bytes": outputBytes})
	data.RawOutputJSON = string(record)
	s.events = append(s.events, event)
	return s
}

// took sets the duration of the last nested result.
func (s *execStream) took(ms int64) *execStream {
	s.events[len(s.events)-1].Data.(*domain.ToolResultEventData).DurationMS = &ms
	return s
}

func factsByPath(facts []InvocationFact) map[string]InvocationFact {
	out := map[string]InvocationFact{}
	for _, fact := range facts {
		out[fact.CommandPath] = fact
	}
	return out
}

// A command that outlives its exec call joins the call that ran it by its
// literal command, even after a newer exec call; one with no literal joins
// the exec call still waiting for its result.
func TestNestedCommandsJoinTheExecCallThatRanThem(t *testing.T) {
	s := &execStream{run: uuid.New()}
	s.exec("A", "make test", "git status --short").ran("git status --short", 0, 10).output("A").
		exec("B", "ls").ran("ls", 0, 5).
		ran("make test", 2, 300). // A's long command finishes while B runs
		ran("rg -n built-at-run-time", 1, 0).output("B")
	facts := factsByPath(DeriveInvocationFacts(s.events))
	callA, callB := s.events[0].ID.String(), s.events[3].ID.String()
	for path, want := range map[string]struct {
		call, outcome string
	}{
		"git status": {callA, OutcomeSuccess},
		"make test":  {callA, OutcomeFailure},
		"ls":         {callB, OutcomeSuccess},
		"rg":         {callB, OutcomeNoMatch},
	} {
		fact, ok := facts[path]
		if !ok || fact.CallEventID != want.call || fact.Outcome != want.outcome || fact.PairingBasis != PairingCommandExecution || fact.DurationMS == nil {
			t.Fatalf("%s: fact=%+v ok=%v, want call %s outcome %s", path, fact, ok, want.call, want.outcome)
		}
	}
	if facts["make test"].OutputEventID != s.events[2].ID.String() || facts["make test"].ResultEventID == facts["make test"].OutputEventID {
		t.Fatalf("make test keeps A's output as what the agent saw: %+v", facts["make test"])
	}
}

// Without a nested result an exec call with several commands still cannot
// give one exit status to each of them.
func TestExecWithoutNestedResultsKeepsUnattributableOutcomesUnknown(t *testing.T) {
	s := &execStream{run: uuid.New()}
	s.exec("A", "ls", "pwd")
	output := domain.NewToolResultEvent(s.run, "", "A", "Exit code: 0", nil)
	s.events = append(s.events, output)
	for _, fact := range DeriveInvocationFacts(s.events) {
		if fact.Outcome != OutcomeUnknown || fact.PairingBasis != "call-id" {
			t.Fatalf("fact=%+v", fact)
		}
	}
}

// A nested result with no exec call before it (a transcript tail) is still
// a command with an outcome.
func TestOrphanNestedCommandIsItsOwnFact(t *testing.T) {
	s := &execStream{run: uuid.New()}
	s.ran("vrooli scenario logs agent-manager", 1, 20)
	facts := DeriveInvocationFacts(s.events)
	if len(facts) != 1 || facts[0].ToolName != NestedCommandTool || facts[0].CommandPath != "vrooli scenario logs" || facts[0].Outcome != OutcomeFailure || facts[0].Capability != KindShell {
		t.Fatalf("facts=%+v", facts)
	}
}

// Claude Code refusals are the harness's, not the command's: refused, never
// failure. Other tool_use_errors are the call failing.
func TestHarnessRefusalIsItsOwnOutcome(t *testing.T) {
	for _, tc := range []struct {
		text, want string
	}{
		{"<tool_use_error>Blocked: sleep 60 followed by: echo done. To wait for a condition, use Monitor</tool_use_error>", OutcomeRefused},
		{"PreToolUse:Bash hook error: ['resource-claude-code' permissions hook-guard 'Bash(git push*)']", OutcomeRefused},
		{"<tool_use_error>File has not been read yet. Read it first before writing to it.</tool_use_error>", OutcomeFailure},
	} {
		run := uuid.New()
		call := domain.NewToolCallEvent(run, "Bash", "c1", map[string]any{"command": "git push origin agi"})
		result := domain.NewToolResultEvent(run, "Bash", "c1", "", errors.New(tc.text))
		facts := DeriveInvocationFacts([]*domain.RunEvent{call, result})
		if len(facts) != 1 || facts[0].Outcome != tc.want || (tc.want == OutcomeRefused) != (facts[0].FailureSignature == "") {
			t.Fatalf("%q: facts=%+v, want %s", tc.text, facts, tc.want)
		}
	}
}

// A loop's status is its body's last command: `done` does not hide it.
func TestLoopBodyTakesTheLoopExitStatus(t *testing.T) {
	facts := deriveOne(t, "Bash", map[string]any{"command": `for f in a b; do echo "$f"; jq -c . "$f"; done`}, intp(0))
	if got := paths(facts); got != "echo | jq" || facts[0].Outcome != OutcomeUnknown || facts[1].Outcome != OutcomeSuccess {
		t.Fatalf("facts=%s outcomes=%s/%s", got, facts[0].Outcome, facts[1].Outcome)
	}
}

// One table names every runner's native tools; there is no per-runner map.
func TestToolKindsCoverEveryRunnersNativeTools(t *testing.T) {
	for tool, want := range map[string]string{
		// Claude Code
		"Read": KindFileRead, "Write": KindFileEdit, "Edit": KindFileEdit, "MultiEdit": KindFileEdit, "Bash": KindShell, "Glob": KindSearch, "Grep": KindSearch,
		"WebSearch": KindWebSearch, "WebFetch": KindWebFetch, "Task": KindDelegate, "Agent": KindDelegate, "TodoWrite": KindPlan, "Monitor": KindWait, "ScheduleWakeup": KindWait,
		"mcp__chrome__navigate": KindMCP,
		// Codex
		"exec": KindShell, "exec_command": KindShell, "apply_patch": KindFileEdit, "update_plan": KindPlan, "spawn_agent": KindDelegate, "wait_agent": KindDelegate, "sleep": KindWait, "view_image": KindFileRead, "image_generation": KindImageGeneration,
		// OpenCode / Grok
		"webfetch": KindWebFetch, "web": KindWebSearch, "search": KindSearch,
		// names no runner declares
		"shell_command": KindShell, "read_thread": KindOther, "Skill": KindOther,
	} {
		if got := CapabilityForTool(tool); got != want {
			t.Errorf("CapabilityForTool(%q) = %q, want %q", tool, got, want)
		}
	}
}

// A shell call names the scenario whose CLI it runs from the manifest
// catalog, never from a substring of its arguments.
func TestCallScenarioReadsTheCatalog(t *testing.T) {
	for input, want := range map[string]string{
		"cd /x && prompt-manager skill read goal-loop | head": "prompt-manager",
		"echo 'ttsattempts whisper'":                          "",
		"vrooli scenario status agent-manager":                "",
	} {
		if got := CallScenario("Bash", map[string]any{"command": input}); got != want {
			t.Errorf("CallScenario(%q) = %q, want %q", input, got, want)
		}
	}
	if got := CallScenario("exec", map[string]any{"input": `await tools.exec_command({cmd:"test-genie runs wait agent-manager r1"})`}); got != "test-genie" {
		t.Errorf("exec script scenario = %q", got)
	}
}

// One command's wall-clock time is held by one of its segments, the one whose
// cost it is (not a pipe reader or a print), so a window's time sums to the
// time its commands ran instead of once per segment.
func TestACommandsDurationIsCountedOnceAcrossItsSegments(t *testing.T) {
	const resume = "ps -p 1901449 -o pid,args\ncp a.json b.json\ncp a.json c.json\ntimeout 25000s python3 price_series.py --resume"
	for _, tc := range []struct {
		name, command, holder string
		ms                    int64
	}{
		{"sequence", resume, "python3", 23318629},
		{"pipe reader", "vrooli scenario status treasury --json | jq .status", "vrooli scenario status", 674},
		{"print after", "test-genie runs wait --json agent-manager r1 && echo done", "test-genie runs wait", 252000},
		{"only filters", "cat a.json | jq .", "cat", 30},
	} {
		s := &execStream{run: uuid.New()}
		s.exec("A", "pwd", tc.command).ran("pwd", 0, 4).took(5).ran(tc.command, 0, 100).took(tc.ms).output("A")
		var total int64
		var holders []string
		for _, fact := range DeriveInvocationFacts(s.events) {
			if fact.DurationMS != nil {
				total += *fact.DurationMS
				if fact.CommandPath != "pwd" {
					holders = append(holders, fact.CommandPath)
				}
			}
		}
		if total != tc.ms+5 || len(holders) != 1 || holders[0] != tc.holder {
			t.Errorf("%s: total=%d holders=%v, want %d held by %s", tc.name, total, holders, tc.ms+5, tc.holder)
		}
	}
	// A Bash call's own result is one command's time too.
	run := uuid.New()
	result := domain.NewToolResultEvent(run, "Bash", "c1", "ok", nil)
	ms := int64(9000)
	result.Data.(*domain.ToolResultEventData).DurationMS = &ms
	facts := DeriveInvocationFacts([]*domain.RunEvent{domain.NewToolCallEvent(run, "Bash", "c1", map[string]any{"command": "make test && echo ok"}), result})
	if len(facts) != 2 || facts[0].DurationMS == nil || facts[1].DurationMS != nil {
		t.Fatalf("make test holds the call's time, echo none: %+v", facts)
	}
}

// Commands with one path but different words join their own results, in
// whatever order they complete.
func TestSamePathCommandsJoinTheirOwnResults(t *testing.T) {
	s := &execStream{run: uuid.New()}
	s.exec("A", "vrooli scenario status treasury", "vrooli scenario status money-ledger").
		ran("vrooli scenario status money-ledger", 1, 70).took(900).
		ran("vrooli scenario status treasury", 0, 30).took(100).output("A")
	facts := DeriveInvocationFacts(s.events)
	if len(facts) != 2 || facts[0].Outcome != OutcomeSuccess || *facts[0].DurationMS != 100 || facts[1].Outcome != OutcomeFailure || *facts[1].DurationMS != 900 || facts[0].ResultEventID == facts[1].ResultEventID {
		t.Fatalf("facts=%+v", facts)
	}
}
