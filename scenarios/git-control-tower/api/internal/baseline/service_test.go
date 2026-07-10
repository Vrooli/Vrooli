package baseline

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"git-control-tower/internal/git"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

func terminalResult() ExecResult {
	return ExecResult{
		Success: true, CompletedAt: time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC),
		TreeDigest: "td:tree", PhaseSetDigest: "ps:catalog", CaptureProfile: CaptureProfile,
		DescriptorSnapshotDigest: "ds:catalog", DescriptorSnapshotSchemaVersion: 1,
		Phases: []PhaseStatus{{Name: "unit", Status: "passed"}},
	}
}

func newTestService(t *testing.T, exec Executor, runs RunsClient, gitState git.State) (*Service, *Storage) {
	t.Helper()
	return newTestServiceWith(t, Deps{Exec: exec, Runs: runs, CaptureGit: fixedGit(gitState)})
}

func newTestServiceWith(t *testing.T, d Deps) (*Service, *Storage) {
	t.Helper()
	st := newTestStorage(t)
	d.Storage = st
	if d.CaptureGit == nil {
		d.CaptureGit = fixedGit(git.State{Branch: "agi", Sha: "abc"})
	}
	return NewService(d), st
}

func seedBaseline(t *testing.T, svc *Service, name string) BaselineManifest {
	t.Helper()
	res, err := svc.Create(context.Background(), CreateRequest{
		RepoID: 1, RepoDir: "/repo", Scenario: "foo", Branch: "agi", Name: name,
	})
	if err != nil {
		t.Fatalf("seed baseline: %v", err)
	}
	return res.Manifest
}

func runDiff(t *testing.T, svc *Service, name string) DiffResult {
	t.Helper()
	out, err := svc.StartDiff(context.Background(), StartDiffRequest{
		RepoID: 1, RepoDir: "/repo", Scenario: "foo", Branch: "agi", Name: name,
	})
	if err != nil {
		t.Fatalf("StartDiff: %v", err)
	}
	cd, err := svc.FinalizeDiff(context.Background(), out.Pending)
	if err != nil {
		t.Fatalf("FinalizeDiff: %v", err)
	}
	if cd.Result == nil {
		t.Fatal("finalized diff has no result")
	}
	return *cd.Result
}

func TestServiceCapturePersistsOneRichRunAnchor(t *testing.T) { // [REQ:GCT-BASELINE-V2-P0]
	exec := &fakeExecutor{result: terminalResult()}
	runs := &fakeRuns{}
	svc, _ := newTestService(t, exec, runs, git.State{Branch: "agi", Sha: "abc", Dirty: true, DirtySummary: "3 modified"})

	res, err := svc.Create(context.Background(), CreateRequest{
		RepoID: 1, RepoDir: "/repo", Scenario: "foo", Name: "before", CreatedBy: "agent",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if exec.calls != 1 || len(runs.pins) != 1 {
		t.Fatalf("capture calls=%d pins=%d, want one each", exec.calls, len(runs.pins))
	}
	if res.Manifest.RunID() != "run-1" || res.Manifest.Run.TreeDigest != "td:tree" || res.Manifest.Run.DescriptorSnapshotDigest != "ds:catalog" {
		t.Fatalf("run anchor = %+v", res.Manifest.Run)
	}
	if res.Manifest.Run.DescriptorSnapshotRef != "test-genie-run:run-1#descriptor-snapshot" {
		t.Fatalf("descriptor ref = %q", res.Manifest.Run.DescriptorSnapshotRef)
	}
	if res.DirtyWarning == "" {
		t.Fatal("dirty capture did not remain explicit")
	}
}

func TestServiceCaptureFailureDoesNotPublishEmptyBaseline(t *testing.T) {
	exec := &fakeExecutor{err: errors.New("aborted")}
	svc, _ := newTestService(t, exec, &fakeRuns{}, git.State{Branch: "agi", Sha: "abc"})
	_, err := svc.Create(context.Background(), CreateRequest{RepoID: 1, Scenario: "foo", Name: "bad"})
	if err == nil || !strings.Contains(err.Error(), "aborted") {
		t.Fatalf("capture error = %v", err)
	}
	if _, err := svc.Get(context.Background(), 1, "foo", "agi", "bad"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("failed capture published a manifest: %v", err)
	}
}

func TestFinalizeCaptureAttachmentTimeoutLeavesDurableIntentPending(t *testing.T) { // [REQ:GCT-DURABLE-OPS-P0]
	exec := &fakeExecutor{result: terminalResult()}
	runs := &fakeRuns{}
	svc, store := newTestService(t, exec, runs, git.State{Branch: "agi", Sha: "abc"})
	pending, err := svc.StartCapture(context.Background(), CreateRequest{
		RepoID: 1, RepoDir: "/repo", Scenario: "foo", Name: "detach",
	})
	if err != nil {
		t.Fatal(err)
	}
	exec.err = context.DeadlineExceeded
	if _, err := svc.FinalizeCapture(context.Background(), pending); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("FinalizeCapture error = %v", err)
	}
	intent, found, err := store.LoadSnapshotIntent(1, "foo", "agi", "detach", pending.Run.RunID)
	if err != nil || !found {
		t.Fatalf("intent found=%v err=%v", found, err)
	}
	if intent.Status != "pending" || intent.Error != "" {
		t.Fatalf("intent = %+v, want pending without terminal error", intent)
	}
	if len(runs.pins) != 0 {
		t.Fatalf("attachment timeout pinned run: %+v", runs.pins)
	}
}

func TestFinalizeCaptureIsIdempotentUnderConcurrentRecovery(t *testing.T) { // [REQ:GCT-BASELINE-V2-P0]
	exec := &fakeExecutor{result: terminalResult()}
	runs := &fakeRuns{}
	svc, _ := newTestService(t, exec, runs, git.State{Branch: "agi", Sha: "abc"})
	pending, err := svc.StartCapture(context.Background(), CreateRequest{
		RepoID: 1, RepoDir: "/repo", Scenario: "foo", Name: "recover",
	})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.FinalizeCapture(context.Background(), pending)
			errCh <- err
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("FinalizeCapture: %v", err)
		}
	}
	if len(runs.pins) != 1 || len(runs.unpins) != 0 {
		t.Fatalf("pins=%v unpins=%v, want one retained claim", runs.pins, runs.unpins)
	}
}

func TestSnapshotStatusRecoversPersistedIntentAfterRestart(t *testing.T) {
	exec := &fakeExecutor{result: terminalResult()}
	runs := &fakeRuns{}
	svc, store := newTestService(t, exec, runs, git.State{Branch: "agi", Sha: "abc"})
	pending, err := svc.StartCapture(context.Background(), CreateRequest{
		RepoID: 1, RepoDir: "/repo", Scenario: "foo", Name: "restart",
	})
	if err != nil {
		t.Fatal(err)
	}
	recovered := NewService(Deps{Storage: store, Exec: exec, Runs: runs, CaptureGit: fixedGit(git.State{Branch: "agi", Sha: "abc"})})
	status, err := recovered.GetSnapshotStatus(context.Background(), SnapshotStatusRequest{
		RepoID: 1, RepoDir: "/repo", Scenario: "foo", Branch: "agi", Name: "restart", RunID: pending.Run.RunID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != "ready" || status.Baseline == nil || status.Baseline.RunID() != pending.Run.RunID {
		t.Fatalf("recovered status = %+v", status)
	}
}

func TestServiceForwardsUnknownPhaseAndOpaqueArtifacts(t *testing.T) { // [REQ:GCT-BASELINE-V2-P0]
	phase := &runspb.PhaseDiff{
		Phase: "future-phase", Verdict: "regression", Regressions: []string{"future-check"},
		DescriptorB: &runspb.RunPhaseDescriptor{Phase: "future-phase", DisplayName: "Future", EvidenceKinds: []string{"future/evidence"}},
		Reasons:     []*runspb.PhaseComparisonReason{{Code: runspb.PhaseComparisonReasonCode_PHASE_COMPARISON_REASON_CODE_NEW_PHASE, Detail: "new catalog entry"}},
	}
	runs := &fakeRuns{
		compare: CompareResult{Verdict: "regression", Phases: []*runspb.PhaseDiff{phase}},
		catalogs: map[string]ArtifactCatalog{
			"run-1": {SchemaVersion: 1, Digest: "base-cat", Artifacts: []*runspb.ArtifactRef{{Id: "opaque-base", Kind: "future/evidence", ProducingPhase: "future-phase"}}},
			"run-2": {SchemaVersion: 1, Digest: "cur-cat", Artifacts: []*runspb.ArtifactRef{{Id: "opaque-current", Kind: "future/evidence", ProducingPhase: "future-phase"}}},
		},
		visualDeltas: []VisualDelta{{Page: "/", Status: "changed", ChangedFraction: 0.25}},
	}
	svc, _ := newTestService(t, &fakeExecutor{result: terminalResult()}, runs, git.State{Branch: "agi", Sha: "abc"})
	seedBaseline(t, svc, "dynamic")
	res := runDiff(t, svc, "dynamic")

	if res.Verdict != VerdictRegression || len(res.Phases) != 1 || res.Phases[0] != phase {
		t.Fatalf("dynamic phase projection = verdict %q phases %+v", res.Verdict, res.Phases)
	}
	if got := res.Evidence.BaseCatalog.Artifacts[0].GetId(); got != "opaque-base" {
		t.Fatalf("base artifact id = %q", got)
	}
	if len(res.Evidence.VisualDeltas) != 1 || res.Evidence.VisualDeltas[0].Status != "changed" {
		t.Fatalf("visual advisory evidence = %+v", res.Evidence.VisualDeltas)
	}
}

func TestServiceRejectsDegradedArtifactEvidence(t *testing.T) {
	runs := &fakeRuns{
		compare: CompareResult{Verdict: "clean", Phases: []*runspb.PhaseDiff{{Phase: "unit", Verdict: "clean"}}},
		catalogs: map[string]ArtifactCatalog{
			"run-1": {SchemaVersion: 1, DegradedReasons: []string{"catalog digest mismatch"}},
			"run-2": {SchemaVersion: 1},
		},
	}
	svc, _ := newTestService(t, &fakeExecutor{result: terminalResult()}, runs, git.State{Branch: "agi", Sha: "abc"})
	seedBaseline(t, svc, "degraded")
	res := runDiff(t, svc, "degraded")
	if res.Verdict != VerdictNotComparable || !strings.Contains(strings.Join(res.Evidence.DegradedReasons, ";"), "digest mismatch") {
		t.Fatalf("degraded result = %+v", res)
	}
}

func TestServiceDeleteUnpinsSingleRunOnce(t *testing.T) {
	runs := &fakeRuns{}
	svc, _ := newTestService(t, &fakeExecutor{result: terminalResult()}, runs, git.State{Branch: "agi", Sha: "abc"})
	seedBaseline(t, svc, "delete-me")
	if err := svc.Delete(context.Background(), 1, "foo", "agi", "delete-me"); err != nil {
		t.Fatal(err)
	}
	if len(runs.unpins) != 1 || runs.unpins[0].runID != "run-1" {
		t.Fatalf("unpins = %+v", runs.unpins)
	}
}

func TestServiceReusesFreshCleanRunForDiff(t *testing.T) {
	now := time.Now().UTC()
	exec := &fakeExecutor{result: terminalResult(), reusable: ReusableRun{RunID: "reused", CompletedAt: now}, reusableHit: true}
	runs := &fakeRuns{
		compare:  CompareResult{Verdict: "clean", Phases: []*runspb.PhaseDiff{{Phase: "unit", Verdict: "clean"}}},
		catalogs: map[string]ArtifactCatalog{"run-1": {SchemaVersion: 1}, "reused": {SchemaVersion: 1}},
	}
	svc, _ := newTestServiceWith(t, Deps{
		Exec: exec, Runs: runs, CaptureGit: fixedGit(git.State{Branch: "agi", Sha: "abc"}), Now: func() time.Time { return now }, ReuseTTL: time.Hour,
	})
	seedBaseline(t, svc, "reuse")
	out, err := svc.StartDiff(context.Background(), StartDiffRequest{RepoID: 1, RepoDir: "/repo", Scenario: "foo", Branch: "agi", Name: "reuse"})
	if err != nil {
		t.Fatal(err)
	}
	if !out.ReusedRun || out.RunID != "reused" || exec.calls != 1 {
		t.Fatalf("reuse outcome=%+v start calls=%d", out, exec.calls)
	}
}

func TestFinalizeDiffAttachmentTimeoutDoesNotCacheTerminalVerdict(t *testing.T) { // [REQ:GCT-DURABLE-OPS-P0]
	exec := &fakeExecutor{result: terminalResult()}
	runs := &fakeRuns{}
	svc, store := newTestService(t, exec, runs, git.State{Branch: "agi", Sha: "abc"})
	seedBaseline(t, svc, "detach-diff")
	out, err := svc.StartDiff(context.Background(), StartDiffRequest{
		RepoID: 1, RepoDir: "/repo", Scenario: "foo", Branch: "agi", Name: "detach-diff",
	})
	if err != nil {
		t.Fatal(err)
	}
	exec.err = context.DeadlineExceeded
	cd, err := svc.FinalizeDiff(context.Background(), out.Pending)
	if !errors.Is(err, context.DeadlineExceeded) || cd.Status != "in_progress" {
		t.Fatalf("FinalizeDiff = %+v, %v", cd, err)
	}
	if _, found, err := store.LoadDiffResult(1, "foo", "agi", "detach-diff", out.RunID); err != nil || found {
		t.Fatalf("terminal cache found=%v err=%v", found, err)
	}
	intent, found, err := store.LatestDiffIntent(1, "foo", "agi", "detach-diff")
	if err != nil || !found || intent.Status != "pending" {
		t.Fatalf("intent found=%v err=%v value=%+v", found, err, intent)
	}
}
