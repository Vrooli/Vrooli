package baseline

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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
		GateQuality: true,
		Phases:      []PhaseStatus{{Name: "unit", Status: "passed"}},
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

// [REQ:GCT-SHARED-WORKSPACE-BASELINE-P0] A dirty primary worktree is normal
// evidence when the declared scenario scope remained stable. It must produce a
// durable before result rather than the historical volatile-baseline failure.
func TestServiceCaptureAcceptsStableSharedScopedEvidence(t *testing.T) {
	result := terminalResult()
	result.GateQuality = false
	result.GitDirty = true
	result.EvidenceTier = "shared-scoped"
	result.SourceScope = "scenario:foo"
	result.SourceStable = true
	svc, _ := newTestService(t, &fakeExecutor{result: result}, &fakeRuns{}, git.State{Branch: "agi", Sha: "abc", Dirty: true, DirtySummary: "other-agent files"})

	captured, err := svc.Create(context.Background(), CreateRequest{RepoID: 1, RepoDir: "/repo", Scenario: "foo", Name: "before"})
	if err != nil {
		t.Fatalf("shared-scoped capture rejected: %v", err)
	}
	if got := captured.Manifest.Run.EvidenceTier; got != "shared-scoped" {
		t.Fatalf("evidence tier = %q", got)
	}
	if strings.Contains(strings.ToLower(captured.DirtyWarning), "untrustworthy") {
		t.Fatalf("dirty workspace warning used invalid baseline language: %q", captured.DirtyWarning)
	}
}

func TestServiceCaptureRetriesOnlyRacedAttempt(t *testing.T) {
	raced := terminalResult()
	raced.GateQuality = false
	raced.EvidenceTier = "degraded"
	raced.SourceScope = "scenario:foo"
	raced.SourceStable = false
	stable := terminalResult()
	stable.GateQuality = false
	stable.EvidenceTier = "shared-scoped"
	stable.SourceScope = "scenario:foo"
	stable.SourceStable = true
	exec := &fakeExecutor{awaitResults: []ExecResult{raced, stable}}
	svc, _ := newTestService(t, exec, &fakeRuns{}, git.State{Branch: "agi", Sha: "abc", Dirty: true})

	captured, err := svc.Create(context.Background(), CreateRequest{RepoID: 1, RepoDir: "/repo", Scenario: "foo", Name: "retry"})
	if err != nil {
		t.Fatalf("capture did not retry raced attempt: %v", err)
	}
	if exec.calls != 2 || exec.awaitCalls != 2 || captured.Manifest.RunID() != "run-2" {
		t.Fatalf("retry evidence calls=%d awaits=%d manifest=%+v", exec.calls, exec.awaitCalls, captured.Manifest.Run)
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

func TestServiceCaptureRetainsBeforeResultWithIncompletePhaseCoverage(t *testing.T) {
	exec := &fakeExecutor{result: terminalResult()}
	runs := &fakeRuns{compare: CompareResult{Verdict: string(VerdictNotComparable), Phases: []*runspb.PhaseDiff{{Phase: "architecture", Verdict: string(VerdictNotComparable)}}}}
	svc, _ := newTestService(t, exec, runs, git.State{Branch: "agi", Sha: "abc"})
	captured, err := svc.Create(context.Background(), CreateRequest{RepoID: 1, Scenario: "foo", Name: "unusable"})
	if err != nil {
		t.Fatalf("capture rejected incomplete coverage: %v", err)
	}
	if captured.Manifest.RunID() == "" {
		t.Fatal("capture did not retain a baseline run")
	}
	if len(runs.pins) != 1 {
		t.Fatalf("capture did not pin the retained before result: %+v", runs.pins)
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

func TestFinalizeCaptureAnchorsTerminalFailedRun(t *testing.T) { // [REQ:GCT-BASELINE-V2-P0]
	failed := terminalResult()
	failed.Success = false
	failed.Phases = []PhaseStatus{{Name: "unit", Status: "failed"}}
	exec := &fakeExecutor{result: failed}
	runs := &fakeRuns{}
	svc, _ := newTestService(t, exec, runs, git.State{Branch: "agi", Sha: "abc"})
	pending, err := svc.StartCapture(context.Background(), CreateRequest{RepoID: 1, RepoDir: "/repo", Scenario: "foo", Name: "failed-evidence"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := svc.FinalizeCapture(context.Background(), pending)
	if err != nil {
		t.Fatalf("terminal failed run should still anchor baseline evidence: %v", err)
	}
	if result.Manifest.RunID() != pending.Run.RunID || len(runs.pins) != 1 {
		t.Fatalf("failed terminal anchor = %#v pins=%#v", result.Manifest.Run, runs.pins)
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

func TestFinalizeCaptureCommitsTerminalSiblingWhileAnotherWaits(t *testing.T) { // [REQ:GCT-DURABLE-OPS-P0]
	exec := &blockingFinalizeExecutor{
		firstAwaitStarted: make(chan struct{}),
		releaseFirst:      make(chan struct{}),
	}
	runs := &fakeRuns{}
	svc, _ := newTestService(t, exec, runs, git.State{Branch: "agi", Sha: "abc"})
	blocked, err := svc.StartCapture(context.Background(), CreateRequest{RepoID: 1, RepoDir: "/repo", Scenario: "blocked", Name: "before"})
	if err != nil {
		t.Fatal(err)
	}
	terminal, err := svc.StartCapture(context.Background(), CreateRequest{RepoID: 1, RepoDir: "/repo", Scenario: "terminal", Name: "before"})
	if err != nil {
		t.Fatal(err)
	}

	blockedDone := make(chan error, 1)
	go func() {
		_, err := svc.FinalizeCapture(context.Background(), blocked)
		blockedDone <- err
	}()
	<-exec.firstAwaitStarted

	terminalDone := make(chan error, 1)
	go func() {
		_, err := svc.FinalizeCapture(context.Background(), terminal)
		terminalDone <- err
	}()
	select {
	case err := <-terminalDone:
		if err != nil {
			t.Fatalf("terminal sibling finalize: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("terminal sibling waited for an unrelated in-progress capture")
	}
	if len(runs.pins) != 1 {
		t.Fatalf("terminal sibling pins=%d, want 1 before blocked wait is released", len(runs.pins))
	}
	if _, err := svc.Get(context.Background(), 1, "terminal", "agi", "before"); err != nil {
		t.Fatalf("terminal sibling manifest missing: %v", err)
	}

	close(exec.releaseFirst)
	if err := <-blockedDone; err != nil {
		t.Fatalf("blocked finalize after release: %v", err)
	}
	if len(runs.pins) != 2 {
		t.Fatalf("pins=%d, want exactly one per completed capture", len(runs.pins))
	}
}

// blockingFinalizeExecutor makes the ordering contract observable without
// sleeps: run-1 cannot return from AwaitResult until the test explicitly
// releases it, while every other durable run is already terminal.
type blockingFinalizeExecutor struct {
	fakeExecutor
	firstAwaitStarted chan struct{}
	releaseFirst      chan struct{}
	secondAwaitDone   chan struct{}
	blockedRunID      string
}

func (e *blockingFinalizeExecutor) AwaitResult(ctx context.Context, scenario, runID string) (ExecResult, error) {
	blockedRunID := e.blockedRunID
	if blockedRunID == "" {
		blockedRunID = "run-1"
	}
	if runID == blockedRunID {
		close(e.firstAwaitStarted)
		select {
		case <-e.releaseFirst:
		case <-ctx.Done():
			return ExecResult{}, ctx.Err()
		}
	}
	if e.secondAwaitDone != nil {
		close(e.secondAwaitDone)
		e.secondAwaitDone = nil
	}
	result := terminalResult()
	result.RunID = runID
	return result, nil
}

func TestServiceLegacyMigrationReconcilesRetentionPinOnce(t *testing.T) { // [REQ:GCT-BASELINE-V2-P0]
	runs := &fakeRuns{}
	svc, store := newTestService(t, &fakeExecutor{result: terminalResult()}, runs, git.State{Branch: "agi", Sha: "abc"})
	writeRawManifest(t, store, "agi", "legacy", legacyFixture("legacy"))

	const readers = 8
	var wg sync.WaitGroup
	errCh := make(chan error, readers)
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.Get(context.Background(), 1, "foo", "agi", "legacy")
			errCh <- err
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("Get migrated baseline: %v", err)
		}
	}
	if len(runs.pins) != 1 {
		t.Fatalf("migration pin calls = %+v, want one", runs.pins)
	}
	if got := runs.pins[0]; got.runID != "run-legacy" || got.by != PinOwner("legacy") || got.reason != "baseline-migration:legacy" {
		t.Fatalf("migration pin = %+v", got)
	}
	persisted, err := store.Load(1, "foo", "agi", "legacy")
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Migration == nil || persisted.Migration.PinReconciledAt.IsZero() {
		t.Fatalf("migration pin checkpoint = %+v", persisted.Migration)
	}
	if _, err := svc.Get(context.Background(), 1, "foo", "agi", "legacy"); err != nil {
		t.Fatal(err)
	}
	if len(runs.pins) != 1 {
		t.Fatalf("already reconciled migration repinned: %+v", runs.pins)
	}
}

func TestServiceLegacyMigrationPinFailureRemainsRetryable(t *testing.T) { // [REQ:GCT-BASELINE-V2-P0]
	runs := &fakeRuns{pinErr: errors.New("test-genie unavailable")}
	svc, store := newTestService(t, &fakeExecutor{result: terminalResult()}, runs, git.State{Branch: "agi", Sha: "abc"})
	writeRawManifest(t, store, "agi", "retry", legacyFixture("retry"))

	if _, err := svc.Get(context.Background(), 1, "foo", "agi", "retry"); err == nil || !strings.Contains(err.Error(), "retention pin") {
		t.Fatalf("migration pin error = %v", err)
	}
	persisted, err := store.Load(1, "foo", "agi", "retry")
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Migration == nil || !persisted.Migration.PinReconciledAt.IsZero() {
		t.Fatalf("failed pin published checkpoint: %+v", persisted.Migration)
	}

	runs.pinErr = nil
	got, err := svc.Get(context.Background(), 1, "foo", "agi", "retry")
	if err != nil {
		t.Fatalf("retry migration pin: %v", err)
	}
	if got.Migration.PinReconciledAt.IsZero() || len(runs.pins) != 2 {
		t.Fatalf("retry result=%+v pins=%+v", got.Migration, runs.pins)
	}
}

func TestServiceListReconcilesEveryLegacyRetentionPin(t *testing.T) { // [REQ:GCT-BASELINE-V2-P0]
	runs := &fakeRuns{}
	svc, store := newTestService(t, &fakeExecutor{result: terminalResult()}, runs, git.State{Branch: "agi", Sha: "abc"})
	writeRawManifest(t, store, "agi", "legacy-a", legacyFixture("legacy-a"))
	writeRawManifest(t, store, "agi", "legacy-b", legacyFixture("legacy-b"))

	manifests, err := svc.List(context.Background(), 1, "foo", "agi")
	if err != nil {
		t.Fatal(err)
	}
	if len(manifests) != 2 || len(runs.pins) != 2 {
		t.Fatalf("listed=%d pins=%+v", len(manifests), runs.pins)
	}
	for _, manifest := range manifests {
		if manifest.Migration == nil || manifest.Migration.PinReconciledAt.IsZero() {
			t.Fatalf("manifest %q missing pin checkpoint: %+v", manifest.Name, manifest.Migration)
		}
	}
	if _, err := svc.List(context.Background(), 1, "foo", "agi"); err != nil {
		t.Fatal(err)
	}
	if len(runs.pins) != 2 {
		t.Fatalf("second list repinned migrations: %+v", runs.pins)
	}
}

// TestCopiedBaselineMigrationRehearsal is the copy-first Phase 10 proof path
// for real baseline data. It never opens a manifest through production storage:
// every direct scenario/branch manifest below GCT_BASELINE_REHEARSAL_SOURCE is
// copied into t.TempDir before the actual Storage + Service migration runs.
// Normal suites skip it; operators opt in with the data/<repo>/baselines path.
func TestCopiedBaselineMigrationRehearsal(t *testing.T) { // [REQ:GCT-BASELINE-V2-P0]
	source := strings.TrimSpace(os.Getenv("GCT_BASELINE_REHEARSAL_SOURCE"))
	if source == "" {
		t.Skip("set GCT_BASELINE_REHEARSAL_SOURCE to rehearse copied real manifests")
	}

	store := newTestStorage(t)
	runs := &fakeRuns{}
	fixedNow := time.Date(2026, 7, 10, 23, 59, 0, 0, time.UTC)
	svc := NewService(Deps{Storage: store, Runs: runs, Now: func() time.Time { return fixedNow }})
	type counts struct {
		total, migrated, alreadyV2, mixed, incomplete, corrupt, pins int
	}
	var got counts

	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			return nil
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		// Only manifests have scenario/branch/name.json. Snapshot/diff intent
		// subtrees are different contracts and deliberately excluded.
		if len(parts) != 3 || strings.HasPrefix(parts[2], ".") {
			return nil
		}
		scenario, branch := parts[0], parts[1]
		name := strings.TrimSuffix(parts[2], ".json")
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		dir, err := store.branchDir(1, scenario, branch)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		copiedPath := store.manifestPath(dir, name)
		if err := os.WriteFile(copiedPath, raw, 0o644); err != nil {
			return err
		}

		got.total++
		decoded, migrated, decodeErr := decodeManifest(raw, fixedNow)
		if decodeErr != nil {
			_, readErr := svc.Get(context.Background(), 1, scenario, branch, name)
			if readErr == nil {
				t.Fatalf("%s: rehearsal accepted manifest rejected by decoder", rel)
			}
			switch {
			case errors.Is(decodeErr, ErrLegacyMixedRuns):
				got.mixed++
			case errors.Is(decodeErr, ErrLegacyIncomplete):
				got.incomplete++
			default:
				got.corrupt++
			}
			if (errors.Is(decodeErr, ErrLegacyMixedRuns) || errors.Is(decodeErr, ErrLegacyIncomplete)) && !strings.Contains(readErr.Error(), "recapture") {
				t.Fatalf("%s: legacy rejection is not actionable: %v", rel, readErr)
			}
			after, err := os.ReadFile(copiedPath)
			if err != nil || string(after) != string(raw) {
				t.Fatalf("%s: rejected migration modified copied data (err=%v)", rel, err)
			}
			return nil
		}

		needsPin := decoded.Migration != nil && decoded.Migration.PinReconciledAt.IsZero()
		pinsBefore := len(runs.pins)
		first, err := svc.Get(context.Background(), 1, scenario, branch, name)
		if err != nil {
			t.Fatalf("%s: first copied migration: %v", rel, err)
		}
		if migrated {
			got.migrated++
		} else {
			got.alreadyV2++
		}
		if needsPin {
			got.pins++
			if len(runs.pins) != pinsBefore+1 || first.Migration == nil || first.Migration.PinReconciledAt.IsZero() {
				t.Fatalf("%s: retention reconciliation mismatch: manifest=%+v pins=%+v", rel, first.Migration, runs.pins[pinsBefore:])
			}
		} else if len(runs.pins) != pinsBefore {
			t.Fatalf("%s: already reconciled manifest acquired another pin", rel)
		}
		afterFirst, err := os.ReadFile(copiedPath)
		if err != nil {
			return err
		}
		if _, err := svc.Get(context.Background(), 1, scenario, branch, name); err != nil {
			t.Fatalf("%s: second copied migration: %v", rel, err)
		}
		afterSecond, err := os.ReadFile(copiedPath)
		if err != nil {
			return err
		}
		if string(afterFirst) != string(afterSecond) || len(runs.pins) != pinsBefore+btoi(needsPin) {
			t.Fatalf("%s: second pass was not idempotent", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.total == 0 {
		t.Fatalf("no baseline manifests found under %s", source)
	}
	if got.pins != len(runs.pins) {
		t.Fatalf("pin accounting mismatch: summary=%d calls=%d", got.pins, len(runs.pins))
	}
	t.Logf("copied migration rehearsal: total=%d migrated=%d already_v2=%d mixed=%d incomplete=%d corrupt=%d pins=%d", got.total, got.migrated, got.alreadyV2, got.mixed, got.incomplete, got.corrupt, got.pins)
}

func btoi(value bool) int {
	if value {
		return 1
	}
	return 0
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

func TestServiceReportsDegradedArtifactEvidenceAsAdvisory(t *testing.T) { // [REQ:GCT-BASELINE-V2-P0]
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
	if res.Verdict != VerdictClean || res.Evidence.EvidenceStatus != "incomplete" || len(res.Evidence.BlockingReasons) != 0 || !strings.Contains(strings.Join(res.Evidence.AdvisoryWarnings, ";"), "digest mismatch") {
		t.Fatalf("degraded result = %+v", res)
	}
}

func TestServicePreservesBehavioralVerdictWhenAdvisoryEvidenceIsUnavailable(t *testing.T) { // [REQ:GCT-BASELINE-V2-P0]
	runs := &fakeRuns{
		compare:          CompareResult{Verdict: "clean", Phases: []*runspb.PhaseDiff{{Phase: "unit", Verdict: "clean"}}},
		catalogErr:       errors.New("catalog transport unavailable"),
		compareVisualErr: errors.New("visual transport unavailable"),
	}
	svc, _ := newTestService(t, &fakeExecutor{result: terminalResult()}, runs, git.State{Branch: "agi", Sha: "different-sha"})
	seedBaseline(t, svc, "advisory-outage")
	res := runDiff(t, svc, "advisory-outage")
	if res.Verdict != VerdictClean {
		t.Fatalf("advisory evidence masked behavioral verdict: %+v", res)
	}
	if len(res.Evidence.DegradedReasons) == 0 {
		t.Fatal("advisory outage was not retained as evidence warning")
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

func TestServiceDeletePreservesManifestWhenUnpinFails(t *testing.T) { // [REQ:GCT-BASELINE-V2-P0]
	runs := &fakeRuns{}
	svc, _ := newTestService(t, &fakeExecutor{result: terminalResult()}, runs, git.State{Branch: "agi", Sha: "abc"})
	seedBaseline(t, svc, "retry-delete")
	runs.unpinErr = errors.New("test-genie unavailable")

	if err := svc.Delete(context.Background(), 1, "foo", "agi", "retry-delete"); err == nil || !strings.Contains(err.Error(), "retention pin") {
		t.Fatalf("Delete error = %v", err)
	}
	if _, err := svc.Get(context.Background(), 1, "foo", "agi", "retry-delete"); err != nil {
		t.Fatalf("failed unpin removed recovery identity: %v", err)
	}

	runs.unpinErr = nil
	if err := svc.Delete(context.Background(), 1, "foo", "agi", "retry-delete"); err != nil {
		t.Fatalf("retry Delete: %v", err)
	}
	if _, err := svc.Get(context.Background(), 1, "foo", "agi", "retry-delete"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("retried delete left manifest: %v", err)
	}
}

// [REQ:GCT-SHARED-WORKSPACE-CACHE-P0] A dirty repository does not bypass the
// scoped cache decision. Test Genie has already matched declared inputs; GCT
// must reuse that result even while unrelated agents edit elsewhere.
func TestServiceReusesFreshScopedRunForDirtyWorkspaceDiff(t *testing.T) {
	now := time.Now().UTC()
	exec := &fakeExecutor{result: terminalResult(), reusable: ReusableRun{RunID: "reused", CompletedAt: now}, reusableHit: true}
	runs := &fakeRuns{
		compare:  CompareResult{Verdict: "clean", Phases: []*runspb.PhaseDiff{{Phase: "unit", Verdict: "clean"}}},
		catalogs: map[string]ArtifactCatalog{"run-1": {SchemaVersion: 1}, "reused": {SchemaVersion: 1}},
	}
	svc, _ := newTestServiceWith(t, Deps{
		Exec: exec, Runs: runs, CaptureGit: fixedGit(git.State{Branch: "agi", Sha: "abc", Dirty: true, DirtySummary: "other-agent change"}), Now: func() time.Time { return now }, ReuseTTL: time.Hour,
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
