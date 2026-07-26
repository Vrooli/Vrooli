package orchestration_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/testutil"
)

// TestWakeRun_RecordsAwaitResultForRefetch verifies the durable re-fetch SSOT:
// waking a parked run records the resolved key/result/timestamp on the run so a
// woken agent can re-read it via GetAwaitResult without re-running the producer.
func TestWakeRun_RecordsAwaitResultForRefetch(t *testing.T) {
	ctx := context.Background()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	mockRunner, _, _, done := newContinuationRunner(t)
	registry := runner.NewRegistry()
	if err := registry.Register(mockRunner); err != nil {
		t.Fatalf("register runner: %v", err)
	}
	svc := orchestration.New(
		repos.Profiles, repos.Tasks, repos.Runs,
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(registry),
		orchestration.WithRunStateRoot(t.TempDir()),
	)

	run := newParkableRun(t, ctx, svc, repos)

	if _, err := svc.ParkRun(ctx, orchestration.ParkRunInput{
		RunID: run.ID, Producer: "git-control-tower", Key: "agent-manager/am-park-resume",
	}); err != nil {
		t.Fatalf("ParkRun: %v", err)
	}

	const result = `{"status":"ready","verdict":"clean"}`
	if _, err := svc.WakeRun(ctx, orchestration.WakeRunInput{RunID: run.ID, Result: result}); err != nil {
		t.Fatalf("WakeRun: %v", err)
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for runner.Continue on wake")
	}

	got, err := svc.GetAwaitResult(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetAwaitResult: %v", err)
	}
	if !got.Found {
		t.Fatal("GetAwaitResult.Found = false, want true after a wake")
	}
	if got.Result != result {
		t.Errorf("GetAwaitResult.Result = %q, want %q", got.Result, result)
	}
	if got.Key != "git-control-tower:agent-manager/am-park-resume" {
		t.Errorf("GetAwaitResult.Key = %q, want producer:key form", got.Key)
	}
	if got.ResolvedAt == nil {
		t.Error("GetAwaitResult.ResolvedAt is nil, want a timestamp")
	}
}

// TestParkRunFromAgent_RefusesNoProgressRepark verifies the structural loop
// guard: once a run has already re-parked on the same key without progress
// (SameKeyParkStreak at the limit), a further same-key no-progress park is
// REFUSED — the run stays running, the turn is not ended, and the cached result
// is handed back so the agent uses it instead of re-running the blocking work.
func TestParkRunFromAgent_RefusesNoProgressRepark(t *testing.T) {
	ctx := context.Background()
	svc, run, repos := newParkFromAgentSvcWithRepos(t)

	// Simulate "already woken on this key once and tolerated one re-park": set the
	// last-resolved await + a streak already at the limit, with no transcript
	// progress since the wake (TranscriptLastSeq == LastWakeSeq).
	const result = `{"status":"ready","verdict":"clean"}`
	run.LastAwaitKey = "git-control-tower:agent-manager/am-park-resume"
	run.LastAwaitResult = result
	run.LastWakeSeq = 7
	run.TranscriptLastSeq = 7
	run.SameKeyParkStreak = 1
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatalf("seed run state: %v", err)
	}
	token := activateToken(t, ctx, repos, run)

	res, err := svc.ParkRunFromAgent(ctx, orchestration.ParkRunFromAgentRequest{
		RunID:         run.ID,
		Producer:      "git-control-tower",
		Key:           "agent-manager/am-park-resume",
		IdentityToken: token,
	})
	if err != nil {
		t.Fatalf("ParkRunFromAgent (refused path must not error): %v", err)
	}
	if !res.Refused {
		t.Fatal("expected Refused=true for a no-progress same-key re-park")
	}
	if res.Result != result {
		t.Errorf("refusal must echo the cached result, got %q", res.Result)
	}
	if !strings.Contains(res.Message, "NOT PARKED") {
		t.Errorf("refusal message should carry a steer, got %q", res.Message)
	}
	if !strings.Contains(res.Message, "await-result") {
		t.Errorf("refusal message should point at the re-fetch command, got %q", res.Message)
	}

	// The run must remain RUNNING — refusal never parks or ends the turn.
	reloaded, err := svc.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if reloaded.Status != domain.RunStatusRunning {
		t.Fatalf("run status = %s, want running (refusal must not park)", reloaded.Status)
	}
}

// TestParkRunFromAgent_FirstReparkTolerated verifies lag tolerance: the FIRST
// same-key no-progress re-park (streak below the limit) is admitted (parked),
// not refused — so a single legitimate "edit then re-check" that briefly looks
// like no-progress is never wrongly blocked. The streak is incremented.
func TestParkRunFromAgent_FirstReparkTolerated(t *testing.T) {
	ctx := context.Background()
	svc, run, repos := newParkFromAgentSvcWithRepos(t)

	run.LastAwaitKey = "git-control-tower:agent-manager/am-park-resume"
	run.LastAwaitResult = "prior"
	run.LastWakeSeq = 7
	run.TranscriptLastSeq = 7
	run.SameKeyParkStreak = 0
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatalf("seed run state: %v", err)
	}
	token := activateToken(t, ctx, repos, run)

	res, err := svc.ParkRunFromAgent(ctx, orchestration.ParkRunFromAgentRequest{
		RunID:         run.ID,
		Producer:      "git-control-tower",
		Key:           "agent-manager/am-park-resume",
		IdentityToken: token,
	})
	if err != nil {
		t.Fatalf("ParkRunFromAgent: %v", err)
	}
	if res.Refused {
		t.Fatal("first same-key re-park must be tolerated, not refused")
	}
	reloaded, err := svc.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if reloaded.Status != domain.RunStatusParked {
		t.Fatalf("status = %s, want parked", reloaded.Status)
	}
	if reloaded.SameKeyParkStreak != 1 {
		t.Errorf("SameKeyParkStreak = %d, want 1 (tolerated re-park counted)", reloaded.SameKeyParkStreak)
	}
}
