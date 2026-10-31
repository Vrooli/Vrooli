package codecs

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/runsignal"

	"github.com/google/uuid"
)

// parseRolloutFixture replays a rollout fixture through the transcript parser
// and returns its events in stream order with sequences assigned.
func parseRolloutFixture(t *testing.T, name string) []*domain.RunEvent {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(raw)), "/home/") {
		t.Fatal("fixture contains an unredacted home path")
	}
	parser := NewCodexForTest().NewTranscriptParser()
	runID := uuid.New()
	var events []*domain.RunEvent
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		got := parser.ParseTranscriptLine(runID, line)
		if got.Err != nil {
			t.Fatal(got.Err)
		}
		for _, event := range got.Events {
			event.Sequence = int64(len(events) + 1)
			events = append(events, event)
		}
	}
	return events
}

// Web-console rollouts report each command an exec call ran as an
// event_msg item_completed CommandExecution. The codec keeps it as a nested
// result (no tool call of its own) with exit code, duration and output size,
// and runsignal joins it to the exec call: every executed command gets a fact
// with a known outcome.
func TestCodexRolloutCommandExecutionsGiveEveryCommandAnOutcome(t *testing.T) {
	events := parseRolloutFixture(t, "codex_command_executions.jsonl")
	calls := 0
	var nested []*domain.RunEvent
	for _, event := range events {
		switch data := event.Data.(type) {
		case *domain.ToolCallEventData:
			calls++
		case *domain.ToolResultEventData:
			if data.ToolName != runsignal.NestedCommandTool {
				continue
			}
			nested = append(nested, event)
			var record codexCommandRecord
			if err := json.Unmarshal([]byte(data.RawOutputJSON), &record); err != nil || len(record.Command) != 3 || record.Command[1] != "-lc" || record.OutputBytes == 0 {
				t.Fatalf("command record=%q err=%v", data.RawOutputJSON, err)
			}
			if data.ExitCode == nil || data.DurationMS == nil || data.OutcomeKnown == nil || !*data.OutcomeKnown || data.Success != (*data.ExitCode == 0) || len(data.Output) > codexNestedOutputLimit+16 {
				t.Fatalf("nested result=%+v", data)
			}
		}
	}
	if calls != 3 || len(nested) != 11 {
		t.Fatalf("calls=%d nested=%d, want the 3 exec calls and 11 command results", calls, len(nested))
	}

	facts := runsignal.DeriveInvocationFacts(events)
	byResult := map[string][]runsignal.InvocationFact{}
	paths := map[string]bool{}
	codeFacts := 0
	for _, fact := range facts {
		byResult[fact.ResultEventID] = append(byResult[fact.ResultEventID], fact)
		paths[fact.CommandPath] = true
		if fact.Capability == "code" {
			codeFacts++
		}
		if fact.ToolName != "exec" {
			t.Fatalf("a command is a unit of its exec call, not a call: %+v", fact)
		}
		if fact.PairingBasis == runsignal.PairingCommandExecution && (fact.OutputEventID == "" || fact.OutputEventID == fact.ResultEventID) {
			t.Fatalf("a joined fact keeps the exec output it was seen through: %+v", fact)
		}
		// One segment of each command holds what the command printed.
		if fact.PairingBasis == runsignal.PairingCommandExecution && (fact.OutputShare == runsignal.ShareOwn) != (fact.OutputBytes > 0) {
			t.Fatalf("only the command's holder carries its printed bytes: %+v", fact)
		}
	}
	for _, event := range nested {
		known := false
		for _, fact := range byResult[event.ID.String()] {
			known = known || fact.Outcome != runsignal.OutcomeUnknown
		}
		if !known {
			t.Fatalf("command %s has no fact with a known outcome: %+v", event.ID, byResult[event.ID.String()])
		}
	}
	failed := byResult[nested[5].ID.String()]
	if len(failed) != 1 || failed[0].CommandPath != "search-hub query" || failed[0].Outcome != runsignal.OutcomeFailure || failed[0].FailureSignature != "exit_code_1" || failed[0].DurationMS == nil {
		t.Fatalf("failed search-hub fact=%+v", failed)
	}
	// Two commands completed after the exec call returned; the script built
	// the third call's commands at run time, so they join by order.
	for _, path := range []string{"prompt-manager skill read", "vrooli scenario status", "jq", "cat"} {
		if !paths[path] {
			t.Fatalf("no fact for %q in %v", path, paths)
		}
	}
	// Each script also loops over its results (and one calls a clock tool):
	// that work is code, never shell.
	if codeFacts != 3 {
		t.Fatalf("code facts=%d, want one per script", codeFacts)
	}
}

// A command's printed size is measured on the output the codec kept: the
// aggregated output, else the item's output when the aggregate is blank, as
// printed (before ANSI stripping and the stored excerpt).
func TestCodexCommandExecutionMeasuresTheOutputItKept(t *testing.T) {
	printed := "\x1b[31m" + strings.Repeat("x", codexNestedOutputLimit+100) + "\x1b[0m"
	for name, item := range map[string]CodexItem{
		"aggregated output": {AggregatedOutput: printed, Output: "ignored"},
		"blank aggregate":   {AggregatedOutput: " \n", Output: printed},
	} {
		item.Type, item.ID, item.Status, item.Command = "CommandExecution", "exec-1", "completed", codexCommand{"/bin/bash", "-lc", "vrooli locks"}
		events := parseCodexCommandExecution(uuid.New(), &item)
		var record codexCommandRecord
		if len(events) != 1 || json.Unmarshal([]byte(events[0].Data.(*domain.ToolResultEventData).RawOutputJSON), &record) != nil || record.OutputBytes != len(printed) {
			t.Errorf("%s: record=%+v, want %d printed bytes", name, record, len(printed))
		}
	}
}
