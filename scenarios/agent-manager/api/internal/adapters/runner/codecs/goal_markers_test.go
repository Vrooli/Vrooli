package codecs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"agent-manager/internal/adapters/runner"
)

// The sanitized line in testdata/codex_goal_updated.jsonl was captured from
// the Codex TUI rollout named in the phase context on 2026-08-21. Objective
// text is shortened; the provider envelope and status shape are preserved.
func TestCodexGoalStatusParsesThreadGoalUpdated(t *testing.T) {
	line, err := os.ReadFile(filepath.Join("testdata", "codex_goal_updated.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	marker, ok := codexGoalStatus(string(line))
	if !ok {
		t.Fatal("Codex goal marker was not recognized")
	}
	if marker.Objective != "complete the governed plan" || marker.Status != runner.GoalStatusBudgetLimited {
		t.Fatalf("marker = %+v", marker)
	}
}

func TestGoalStatusRecognizesEveryCodexTerminalVocabulary(t *testing.T) {
	for _, status := range []runner.GoalStatus{
		runner.GoalStatusActive, runner.GoalStatusPaused, runner.GoalStatusBlocked,
		runner.GoalStatusUsageLimited, runner.GoalStatusBudgetLimited, runner.GoalStatusComplete,
	} {
		marker, ok := codexGoalStatus(`{"type":"event_msg","payload":{"type":"thread_goal_updated","goal":{"objective":"finish","status":"` + string(status) + `"}}}`)
		if !ok || marker.Status != status {
			t.Fatalf("status %q parsed as %+v, ok=%v", status, marker, ok)
		}
	}
}

func TestClaudeGoalStatusParsesInteractiveAttachment(t *testing.T) {
	line := `{"type":"attachment","attachment":{"type":"goal_status","condition":"file exists","met":true,"iterations":3,"reason":"verified"}}`
	marker, ok := claudeGoalStatus(line)
	if !ok {
		t.Fatal("Claude goal marker was not recognized")
	}
	if marker.Objective != "file exists" || marker.Status != runner.GoalStatusComplete {
		t.Fatalf("marker = %+v, want completed file-exists goal", marker)
	}
	if marker.Iteration != 3 || marker.LastReason != "verified" {
		t.Fatalf("marker metadata = %+v, want iteration/reason", marker)
	}
}

// TestGoalMarkerParityAcrossCodecs proves Claude and Codex produce the same
// GoalMarker shape for the same objective/status, so the consumer and workflow
// terminal mapping is harness-independent.
func TestGoalMarkerParityAcrossCodecs(t *testing.T) {
	codexMarker, ok := codexGoalStatus(`{"type":"event_msg","payload":{"type":"thread_goal_updated","goal":{"objective":"finish the plan","status":"complete"}}}`)
	if !ok {
		t.Fatal("codex marker not recognized")
	}
	claudeMarker, ok := claudeGoalStatus(`{"type":"attachment","attachment":{"type":"goal_status","condition":"finish the plan","met":true}}`)
	if !ok {
		t.Fatal("claude marker not recognized")
	}
	if codexMarker.Objective != claudeMarker.Objective || codexMarker.Status != claudeMarker.Status {
		t.Fatalf("marker shapes differ: codex=%+v claude=%+v", codexMarker, claudeMarker)
	}
	if claudeMarker.Status != runner.GoalStatusComplete {
		t.Fatalf("expected complete vocabulary, got %q", claudeMarker.Status)
	}
}

func TestClaudeGoalStatusDoesNotTreatProseAsMarker(t *testing.T) {
	if _, ok := claudeGoalStatus(`{"type":"assistant","message":{"content":[{"type":"text","text":"goal_status met true"}]}}`); ok {
		t.Fatal("assistant prose must not become a goal marker")
	}
}

// The lines both consumers of agenttranscript are characterized against. Its own
// test checks the compaction reading of each; this checks the lifecycle reading.
func TestGoalMarkersOnSharedFixtures(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "..", "..", "packages", "cli-core", "agenttranscript", "testdata", "goal_lines.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []struct {
		Name      string `json:"name"`
		Harness   string `json:"harness"`
		Line      string `json:"line"`
		Lifecycle string `json:"lifecycle"`
	}
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no goal fixtures")
	}
	for _, fixture := range fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			var marker runner.GoalMarker
			var ok bool
			switch fixture.Harness {
			case "codex":
				marker, ok = codexGoalStatus(fixture.Line)
			case "claude":
				marker, ok = claudeGoalStatus(fixture.Line)
			}
			got := "none"
			if ok {
				got = string(marker.Status)
			}
			if got != fixture.Lifecycle {
				t.Fatalf("lifecycle reading = %s, want %s", got, fixture.Lifecycle)
			}
		})
	}
}
