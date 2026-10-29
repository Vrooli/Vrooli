package orchestration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/domain"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/runsignal"
)

// After an API restart, recovery drains the transcript from the persisted
// cursor with a fresh parser. For run aee95b01 that replay doubled the run
// (EVIDENCE E-5). This feeds a trimmed real rollout through the live capture,
// then the real Reconciler drain of the last turn, and requires the stored
// stream's account, read-model total and terminal summary to equal the
// rollout's final counter.
func TestRecoveryDrainOfTheLastTurnDoesNotInflateTheRun(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "adapters", "runner", "codecs", "testdata", "codex_usage_rollout.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	lines := strings.SplitAfter(content, "\n")
	var session string
	var final, lastTurnOffset, offset int64
	for _, line := range lines {
		var record struct {
			Type    string `json:"type"`
			Payload struct {
				ID   string `json:"id"`
				Type string `json:"type"`
				Info *struct {
					Total struct {
						Total int64 `json:"total_tokens"`
					} `json:"total_token_usage"`
				} `json:"info"`
			} `json:"payload"`
		}
		if json.Unmarshal([]byte(line), &record) == nil {
			switch {
			case record.Type == "session_meta":
				session = record.Payload.ID
			case record.Payload.Type == "task_started":
				lastTurnOffset = offset
			case record.Payload.Type == "token_count" && record.Payload.Info != nil:
				final = record.Payload.Info.Total.Total
			}
		}
		offset += int64(len(line))
	}

	reconciler, repos, store := newRecoveryTestReconciler(t, domain.RunnerTypeCodex)
	run := createRecoveryTestRun(t, repos, domain.RunnerTypeCodex)
	path := filepath.Join(t.TempDir(), "rollout.jsonl")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	// Live capture persisted every report of the whole transcript.
	live := codecs.NewCodexForTest().NewTranscriptParser()
	for _, line := range lines {
		for _, evt := range live.ParseTranscriptLine(run.ID, line).Events {
			evt.RunID = run.ID
			if err := store.Append(t.Context(), run.ID, evt); err != nil {
				t.Fatal(err)
			}
		}
	}
	run.SessionID, run.TranscriptCursor = session, lastTurnOffset
	if err := repos.Runs.Update(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	saved, err := repos.Runs.Get(t.Context(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	terminal, err := reconciler.drainTranscript(t.Context(), saved, path, nil, codecs.NewCodexForTest())
	if err != nil || terminal == nil || !terminal.Success {
		t.Fatalf("recovery drain terminal=%+v err=%v", terminal, err)
	}
	events, err := store.Get(t.Context(), run.ID, event.GetOptions{AfterSequence: -1})
	if err != nil {
		t.Fatal(err)
	}
	if got := runsignal.AccountUsage(events).TotalTokens(); got != final {
		t.Fatalf("account after recovery replay = %d, want the rollout's final counter %d", got, final)
	}
	if fact := invocationreadmodel.ProjectRun(saved, events, saved.CreatedAt); fact.TotalTokens != final {
		t.Fatalf("read-model total = %d, want %d", fact.TotalTokens, final)
	}
	_, summary, err := resolvePersistedRunResult(t.Context(), store, run.ID, true, 0, "completed")
	if err != nil || summary == nil || summary.TokensUsed != final {
		t.Fatalf("terminal summary = %+v err=%v, want %d tokens", summary, err, final)
	}
}
