package usageattribution

import (
	"strings"
	"testing"
)

// P2-D12 / "Where it went": no burner label is a raw UUID in any dimension,
// and request counts are named requests.
func TestBurnerLabelsNeverShowARawUUID(t *testing.T) {
	f := newWeekFixture()
	const id = "5f51d3a9-1c2b-4e0a-9d3f-7a6b5c4d3e2f"
	f.runs[id] = Run{ID: id, Source: SourceAgentManager, Runner: codex, Session: id, Workload: "workflow_node:draft"}
	f.runs["orchestrator"] = Run{ID: "orchestrator", Label: "codex session " + id, Source: SourceIndexed, Runner: codex, Session: "orchestrator", SessionLabel: "session codex session " + id}
	turn := f.turns[0]
	turn.RunID = id
	f.turns = append(f.turns, turn)
	result := Attribute(f.window, f.window.Start, f.turns, fixturePrices)
	for _, by := range Dimensions {
		for _, burner := range Group(result, by, f.runs, "") {
			if uuidPattern.MatchString(burner.Label) {
				t.Errorf("by %s: label %q holds a raw UUID", by, burner.Label)
			}
			if by != ByRun && by != BySession {
				continue
			}
			if burner.Key == id && !strings.HasPrefix(burner.Label, "Workflow draft · 5f51…") && by == ByRun {
				t.Errorf("run label = %q, want the workload in words and a short id", burner.Label)
			}
		}
	}
	if got := workloadLabel("workflow_node:draft", ""); got != "Workflow draft" {
		t.Errorf("workload label = %q", got)
	}
}

func TestBurnersCountRequestsNotTurns(t *testing.T) {
	f := newWeekFixture()
	result := Attribute(f.window, f.window.Start, f.turns, fixturePrices)
	var requests int
	for _, turn := range result.Turns {
		requests += turn.Reports
	}
	var got int
	for _, burner := range Group(result, ByRunner, f.runs, "") {
		got += burner.Requests
	}
	if got != requests || got == 0 {
		t.Fatalf("burner requests = %d, want the %d provider reports", got, requests)
	}
}
