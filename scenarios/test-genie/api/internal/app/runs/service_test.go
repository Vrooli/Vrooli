package runs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"

	sharedartifacts "test-genie/internal/shared/artifacts"
	sharedruns "test-genie/internal/shared/runs"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

func newTestService(t *testing.T) (*Service, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	// The read-only index RPCs under test need neither the run manager nor the
	// planner.
	return NewService(root, nil, nil, nil), root
}

func seedRecord(t *testing.T, root string, rec sharedruns.RunRecord) {
	t.Helper()
	if err := sharedruns.NewIndex(filepath.Join(root, "demo")).Append(rec); err != nil {
		t.Fatalf("seed %s: %v", rec.RunID, err)
	}
}

func TestListAndGetRun(t *testing.T) {
	svc, root := newTestService(t)
	seedRecord(t, root, sharedruns.RunRecord{RunID: "r1", Scenario: "demo", StartedAt: time.Now().UTC(), Status: sharedruns.StatusPassed})
	seedRecord(t, root, sharedruns.RunRecord{RunID: "r2", Scenario: "demo", StartedAt: time.Now().UTC().Add(time.Minute), Status: sharedruns.StatusFailed})

	list, err := svc.ListRuns(context.Background(), connect.NewRequest(&runspb.ListRunsRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(list.Msg.GetRuns()) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(list.Msg.GetRuns()))
	}
	if list.Msg.GetRuns()[0].GetRunId() != "r2" {
		t.Fatalf("expected newest-first (r2), got %s", list.Msg.GetRuns()[0].GetRunId())
	}

	// Status filter.
	failed, err := svc.ListRuns(context.Background(), connect.NewRequest(&runspb.ListRunsRequest{Scenario: "demo", Status: "failed"}))
	if err != nil {
		t.Fatalf("ListRuns(failed): %v", err)
	}
	if len(failed.Msg.GetRuns()) != 1 || failed.Msg.GetRuns()[0].GetRunId() != "r2" {
		t.Fatalf("status filter failed: %v", failed.Msg.GetRuns())
	}

	got, err := svc.GetRun(context.Background(), connect.NewRequest(&runspb.GetRunRequest{Scenario: "demo", RunId: "r1"}))
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got.Msg.GetRun().GetStatus() != sharedruns.StatusPassed {
		t.Fatalf("unexpected status: %s", got.Msg.GetRun().GetStatus())
	}

	// Missing run → NotFound.
	if _, err := svc.GetRun(context.Background(), connect.NewRequest(&runspb.GetRunRequest{Scenario: "demo", RunId: "nope"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestPinUnpinAndForceDelete(t *testing.T) {
	svc, root := newTestService(t)
	seedRecord(t, root, sharedruns.RunRecord{RunID: "r1", Scenario: "demo", StartedAt: time.Now().UTC(), Status: sharedruns.StatusPassed})

	pinned, err := svc.PinRun(context.Background(), connect.NewRequest(&runspb.PinRunRequest{Scenario: "demo", RunId: "r1", PinnedBy: "gct:baseline:x", Reason: "baseline"}))
	if err != nil {
		t.Fatalf("PinRun: %v", err)
	}
	if len(pinned.Msg.GetRun().GetPins()) != 1 {
		t.Fatalf("expected 1 pin, got %v", pinned.Msg.GetRun().GetPins())
	}

	// Deleting a pinned run without force → FailedPrecondition.
	if _, err := svc.DeleteRun(context.Background(), connect.NewRequest(&runspb.DeleteRunRequest{Scenario: "demo", RunId: "r1"})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("expected FailedPrecondition for pinned delete, got %v", err)
	}

	if _, err := svc.UnpinRun(context.Background(), connect.NewRequest(&runspb.UnpinRunRequest{Scenario: "demo", RunId: "r1", PinnedBy: "gct:baseline:x"})); err != nil {
		t.Fatalf("UnpinRun: %v", err)
	}

	del, err := svc.DeleteRun(context.Background(), connect.NewRequest(&runspb.DeleteRunRequest{Scenario: "demo", RunId: "r1"}))
	if err != nil {
		t.Fatalf("DeleteRun: %v", err)
	}
	if !del.Msg.GetDeleted() {
		t.Fatal("expected deleted=true")
	}
}

func TestCompareRunsClassification(t *testing.T) {
	svc, root := newTestService(t)
	// Baseline: workflows + tests + structure all pass.
	seedRecord(t, root, sharedruns.RunRecord{
		RunID: "base", Scenario: "demo", StartedAt: time.Now().UTC(), Status: sharedruns.StatusPassed,
		Phases: []sharedruns.PhaseRecord{
			{Name: "playbooks", Status: "passed"},
			{Name: "unit", Status: "failed"}, // preexisting failure
			{Name: "structure", Status: "passed"},
		},
	})
	// Current: workflows regressed, a new phase fails, unit still fails, structure clean.
	seedRecord(t, root, sharedruns.RunRecord{
		RunID: "cur", Scenario: "demo", StartedAt: time.Now().UTC().Add(time.Minute), Status: sharedruns.StatusFailed,
		Phases: []sharedruns.PhaseRecord{
			{Name: "playbooks", Status: "failed"},   // regression (passed → failed)
			{Name: "unit", Status: "failed"},        // preexisting
			{Name: "structure", Status: "passed"},   // clean
			{Name: "integration", Status: "failed"}, // new failure (absent in base)
		},
	})

	resp, err := svc.CompareRuns(context.Background(), connect.NewRequest(&runspb.CompareRunsRequest{Scenario: "demo", RunIdA: "base", RunIdB: "cur"}))
	if err != nil {
		t.Fatalf("CompareRuns: %v", err)
	}
	got := map[string]string{}
	for _, p := range resp.Msg.GetPhases() {
		got[p.GetPhase()] = p.GetVerdict()
	}
	if got["playbooks"] != verdictRegression {
		t.Errorf("playbooks: want regression, got %s", got["playbooks"])
	}
	if got["unit"] != verdictPreexisting {
		t.Errorf("unit: want preexisting, got %s", got["unit"])
	}
	if got["structure"] != verdictClean {
		t.Errorf("structure: want clean, got %s", got["structure"])
	}
	if got["integration"] != verdictNewFailure {
		t.Errorf("integration: want new-failure, got %s", got["integration"])
	}
	// Overall verdict is the worst (regression).
	if resp.Msg.GetVerdict() != verdictRegression {
		t.Errorf("overall verdict: want regression, got %s", resp.Msg.GetVerdict())
	}

	// Phase filter restricts output.
	filtered, err := svc.CompareRuns(context.Background(), connect.NewRequest(&runspb.CompareRunsRequest{Scenario: "demo", RunIdA: "base", RunIdB: "cur", Phase: "structure"}))
	if err != nil {
		t.Fatalf("CompareRuns(filter): %v", err)
	}
	if len(filtered.Msg.GetPhases()) != 1 || filtered.Msg.GetPhases()[0].GetPhase() != "structure" {
		t.Fatalf("phase filter failed: %v", filtered.Msg.GetPhases())
	}
}

func TestGetPhaseArtifact(t *testing.T) {
	svc, root := newTestService(t)
	scenarioDir := filepath.Join(root, "demo")
	seedRecord(t, root, sharedruns.RunRecord{RunID: "r1", Scenario: "demo", StartedAt: time.Now().UTC(), Status: sharedruns.StatusPassed})

	phaseDir := sharedartifacts.RunPhaseResultsDir(scenarioDir, "r1")
	if err := os.MkdirAll(phaseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(phaseDir, "unit.json"), []byte(`{"phase":"unit","status":"passed"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	resp, err := svc.GetPhaseArtifact(context.Background(), connect.NewRequest(&runspb.GetPhaseArtifactRequest{Scenario: "demo", RunId: "r1", Phase: "unit"}))
	if err != nil {
		t.Fatalf("GetPhaseArtifact: %v", err)
	}
	if resp.Msg.GetContentType() != "application/json" {
		t.Errorf("unexpected content type: %s", resp.Msg.GetContentType())
	}
	if resp.Msg.GetContent() == "" {
		t.Error("expected non-empty content")
	}

	// Missing artifact → NotFound.
	if _, err := svc.GetPhaseArtifact(context.Background(), connect.NewRequest(&runspb.GetPhaseArtifactRequest{Scenario: "demo", RunId: "r1", Phase: "missing"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

// FindRun returns the newest completed run that matches every shape filter, and
// found=false when none does. It is the reuse primitive git-control-tower
// queries: only a clean, comprehensive+baseline run at the requested sha should
// match (so a quick run, a dirty run, or a different sha is never reused).
func TestFindRun(t *testing.T) {
	svc, root := newTestService(t)
	base := time.Now().UTC()
	// A matching clean comprehensive+baseline run at sha "abc".
	seedRecord(t, root, sharedruns.RunRecord{
		RunID: "match", Scenario: "demo", StartedAt: base, Status: sharedruns.StatusPassed,
		GitSha: "abc", Preset: "comprehensive", CaptureProfile: "baseline",
	})
	// A newer run that should NOT match: different preset.
	seedRecord(t, root, sharedruns.RunRecord{
		RunID: "wrong-preset", Scenario: "demo", StartedAt: base.Add(time.Minute), Status: sharedruns.StatusPassed,
		GitSha: "abc", Preset: "quick", CaptureProfile: "baseline",
	})
	// A newer run that should NOT match: dirty tree.
	seedRecord(t, root, sharedruns.RunRecord{
		RunID: "dirty", Scenario: "demo", StartedAt: base.Add(2 * time.Minute), Status: sharedruns.StatusPassed,
		GitSha: "abc", Preset: "comprehensive", CaptureProfile: "baseline", GitDirty: true,
	})
	// A newer run that should NOT match: different sha.
	seedRecord(t, root, sharedruns.RunRecord{
		RunID: "other-sha", Scenario: "demo", StartedAt: base.Add(3 * time.Minute), Status: sharedruns.StatusPassed,
		GitSha: "def", Preset: "comprehensive", CaptureProfile: "baseline",
	})

	resp, err := svc.FindRun(context.Background(), connect.NewRequest(&runspb.FindRunRequest{
		Scenario: "demo", GitSha: "abc", Preset: "comprehensive", CaptureProfile: "baseline",
		Status: "passed", RequireClean: true,
	}))
	if err != nil {
		t.Fatalf("FindRun: %v", err)
	}
	if !resp.Msg.GetFound() {
		t.Fatal("expected a matching run")
	}
	if got := resp.Msg.GetRun().GetRunId(); got != "match" {
		t.Fatalf("FindRun matched %q, want the clean comprehensive+baseline run \"match\"", got)
	}

	// No run at this sha → found=false.
	miss, err := svc.FindRun(context.Background(), connect.NewRequest(&runspb.FindRunRequest{
		Scenario: "demo", GitSha: "zzz", Preset: "comprehensive", CaptureProfile: "baseline", RequireClean: true,
	}))
	if err != nil {
		t.Fatalf("FindRun(miss): %v", err)
	}
	if miss.Msg.GetFound() {
		t.Fatalf("expected no match for an unknown sha, got %q", miss.Msg.GetRun().GetRunId())
	}
}
