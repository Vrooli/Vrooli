package trials_test

import (
	"context"
	"testing"
	"time"

	"meta-optimization-manager/internal/testutil/db"
	"meta-optimization-manager/internal/testutil/mocks"
	internaltrials "meta-optimization-manager/internal/trials"
)

func TestTrialsRepositoryRecordAndQuery(t *testing.T) {
	h := db.NewSQLite(t)
	if _, err := h.Exec(internaltrials.Schema()); err != nil {
		t.Fatalf("schema: %v", err)
	}
	clk := mocks.NewFakeClock(time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC))
	repo := internaltrials.NewSQLiteRepository(h, clk)
	ctx := context.Background()

	run := internaltrials.TrialRun{
		ID: "run-1", TaskID: "trial/g1", Suite: internaltrials.SuiteAddFeature, Model: "ollama/x",
		GuideTaskID: "g1", Verdict: internaltrials.VerdictPass, Tokens: 1200, DurationMs: 8000,
		SandboxDiffRef: "sbx-1", At: clk.Now(),
	}
	if err := repo.RecordRun(ctx, run); err != nil {
		t.Fatalf("record: %v", err)
	}
	// A second run for the same guide task bumps the gate count (still 1 distinct).
	run2 := run
	run2.ID = "run-2"
	if err := repo.RecordRun(ctx, run2); err != nil {
		t.Fatalf("record 2: %v", err)
	}

	got, ok, err := repo.GetRun(ctx, "run-1")
	if err != nil || !ok || got.SandboxDiffRef != "sbx-1" || got.Verdict != internaltrials.VerdictPass {
		t.Fatalf("get run wrong: %+v ok=%v err=%v", got, ok, err)
	}

	runs, err := repo.Runs(ctx, internaltrials.RunFilter{Suite: internaltrials.SuiteAddFeature}, 0, true)
	if err != nil || len(runs) != 2 {
		t.Fatalf("runs query: len=%d err=%v", len(runs), err)
	}
	// desc order: newest (run-2) first.
	if runs[0].ID != "run-2" {
		t.Fatalf("expected desc order, got %s first", runs[0].ID)
	}

	gated, err := repo.GatedGuideTaskCount(ctx)
	if err != nil || gated != 1 {
		t.Fatalf("gated count: got %d err=%v (want 1 distinct guide task)", gated, err)
	}
}

func TestTrialsRepositoryGetMissing(t *testing.T) {
	h := db.NewSQLite(t)
	if _, err := h.Exec(internaltrials.Schema()); err != nil {
		t.Fatalf("schema: %v", err)
	}
	repo := internaltrials.NewSQLiteRepository(h, mocks.NewFakeClock(time.Now()))
	if _, ok, err := repo.GetRun(context.Background(), "nope"); ok || err != nil {
		t.Fatalf("expected miss, got ok=%v err=%v", ok, err)
	}
}
