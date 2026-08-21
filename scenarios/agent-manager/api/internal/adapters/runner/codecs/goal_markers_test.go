package codecs

import (
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
