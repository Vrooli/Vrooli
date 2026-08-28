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

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

func collectionService(t *testing.T) (*Service, *fakeExecutor) {
	t.Helper()
	exec := &fakeExecutor{result: ExecResult{Success: true, CompletedAt: time.Now().UTC(), TreeDigest: "tree", PhaseSetDigest: "phases", Phases: []PhaseStatus{{Name: "unit", Status: "passed"}}, CaptureProfile: CaptureProfile, DescriptorSnapshotDigest: "descriptor", DescriptorSnapshotSchemaVersion: 1}}
	return NewService(Deps{Storage: newTestStorage(t), Exec: exec, Runs: &fakeRuns{}, CaptureGit: fixedGit(git.State{Sha: "abc", Branch: "agi"})}), exec
}

func TestCollectionDiffDetailPreservesComparisonRecovery(t *testing.T) {
	got := collectionDiffDetail(CachedDiff{Result: &DiffResult{Comparison: &runspb.CompareRunsResponse{Diagnostics: []*runspb.ComparisonDiagnostic{{Code: "provider_unavailable", Detail: "provider x is unavailable", Remediation: "restore provider x and rerun"}}}}})
	if got != "provider x is unavailable (recovery: restore provider x and rerun)" {
		t.Fatalf("collection detail = %q", got)
	}
}

func TestCollectionCaptureStartsEachMemberOnceAndFinalizesCoverage(t *testing.T) {
	svc, exec := collectionService(t)
	req := StartCollectionCaptureRequest{RepoID: 1, RepoDir: t.TempDir(), Name: "before", Targets: []CollectionTarget{{Scenario: "plan-manager", BaselineName: "before", Required: true}, {Scenario: "git-control-tower", BaselineName: "before", Required: true}}}
	started, err := svc.StartCollectionCapture(context.Background(), req)
	if err != nil {
		t.Fatalf("StartCollectionCapture: %v", err)
	}
	if exec.calls != 2 || len(started.Pending) != 2 || started.Collection.Coverage().Pending != 2 {
		t.Fatalf("start = %#v calls=%d", started, exec.calls)
	}
	for _, pending := range started.Pending {
		if _, err := svc.FinalizeCollectionCapture(context.Background(), 1, pending); err != nil {
			t.Fatalf("FinalizeCollectionCapture: %v", err)
		}
	}
	collection, err := svc.storage.LoadCollection(1, "agi", "before")
	if err != nil || !collection.Coverage().Complete() {
		t.Fatalf("final collection = %#v err=%v", collection, err)
	}
	for _, member := range collection.Members {
		if member.GitSHA != "abc" {
			t.Fatalf("member lost capture git identity: %#v", member)
		}
	}
	resumed, err := svc.StartCollectionCapture(context.Background(), req)
	if err != nil || !resumed.Resumed || exec.calls != 2 || len(resumed.Pending) != 0 {
		t.Fatalf("resume = %#v err=%v calls=%d", resumed, err, exec.calls)
	}
}

func TestCollectionCaptureReusesExistingImmutableBaseline(t *testing.T) {
	svc, exec := collectionService(t)
	if _, err := svc.Create(context.Background(), CreateRequest{
		RepoID: 1, RepoDir: t.TempDir(), Scenario: "plan-manager", Branch: "agi", Name: "before",
	}); err != nil {
		t.Fatalf("Create existing baseline: %v", err)
	}
	if exec.calls != 1 {
		t.Fatalf("existing baseline runs = %d, want 1", exec.calls)
	}

	started, err := svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{
		RepoID: 1, RepoDir: t.TempDir(), Name: "before",
		Targets: []CollectionTarget{{Scenario: "plan-manager", BaselineName: "before", Required: true}},
	})
	if err != nil {
		t.Fatalf("StartCollectionCapture: %v", err)
	}
	if exec.calls != 1 || len(started.Pending) != 0 {
		t.Fatalf("collection started another run: calls=%d pending=%#v", exec.calls, started.Pending)
	}
	member := started.Collection.Members[0]
	if member.Status != CollectionMemberReady || member.RunID == "" || member.Error != "reused immutable baseline" {
		t.Fatalf("reused member = %#v", member)
	}
	if !started.Collection.Coverage().Complete() {
		t.Fatalf("reused baseline did not complete collection: %#v", started.Collection.Coverage())
	}
}

func TestCollectionCaptureCommitsTerminalMemberWhileSiblingWaits(t *testing.T) { // [REQ:GCT-DURABLE-OPS-P0]
	exec := &blockingFinalizeExecutor{
		firstAwaitStarted: make(chan struct{}),
		releaseFirst:      make(chan struct{}),
	}
	svc := NewService(Deps{
		Storage: newTestStorage(t), Exec: exec, Runs: &fakeRuns{},
		CaptureGit: fixedGit(git.State{Sha: "abc", Branch: "agi"}),
	})
	started, err := svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{
		RepoID: 1, RepoDir: t.TempDir(), Name: "before",
		Targets: []CollectionTarget{
			{Scenario: "blocked", BaselineName: "before", Required: true},
			{Scenario: "terminal", BaselineName: "before", Required: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(started.Pending) != 2 {
		t.Fatalf("pending captures = %#v", started.Pending)
	}

	blockedDone := make(chan error, 1)
	go func() {
		_, err := svc.FinalizeCollectionCapture(context.Background(), 1, started.Pending[0])
		blockedDone <- err
	}()
	<-exec.firstAwaitStarted

	terminalDone := make(chan error, 1)
	go func() {
		_, err := svc.FinalizeCollectionCapture(context.Background(), 1, started.Pending[1])
		terminalDone <- err
	}()
	var partial CollectionManifest
	select {
	case err := <-terminalDone:
		if err != nil {
			t.Fatalf("terminal member finalize: %v", err)
		}
		partial, err = svc.storage.LoadCollection(1, "agi", "before")
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("terminal collection member waited for blocked sibling")
	}
	if partial.Coverage().Ready != 1 || partial.Coverage().Pending != 1 || partial.Coverage().Complete() {
		t.Fatalf("partial coverage = %#v", partial.Coverage())
	}

	close(exec.releaseFirst)
	if err := <-blockedDone; err != nil {
		t.Fatalf("blocked member finalize after release: %v", err)
	}
	complete, err := svc.storage.LoadCollection(1, "agi", "before")
	if err != nil || !complete.Coverage().Complete() || complete.Coverage().Ready != 2 {
		t.Fatalf("complete collection = %#v err=%v", complete, err)
	}
}

func TestCollectionCaptureMarksOnlyFailedMemberAndPreservesOtherStarts(t *testing.T) {
	svc, _ := collectionService(t)
	svc.exec = &fakeExecutor{startErr: errors.New("test-genie unavailable")}
	result, err := svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{RepoID: 1, RepoDir: t.TempDir(), Name: "before", Targets: []CollectionTarget{{Scenario: "plan-manager", BaselineName: "before", Required: true}}})
	if err != nil {
		t.Fatalf("collection start should persist a member failure, got %v", err)
	}
	if result.Collection.Coverage().Failed != 1 || result.Collection.Coverage().Complete() {
		t.Fatalf("failure coverage = %#v", result.Collection.Coverage())
	}
}

func TestCollectionCaptureAdmissionSaturationDefersMemberUntilSiblingCompletes(t *testing.T) {
	svc, exec := collectionService(t)
	exec.startErrs = []error{
		nil,
		errors.New("resource_exhausted: test-genie admission is saturated (caller queued run capacity)"),
		nil,
	}
	started, err := svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{
		RepoID: 1, RepoDir: t.TempDir(), Name: "before",
		Targets: []CollectionTarget{
			{Scenario: "first", BaselineName: "before", Required: true},
			{Scenario: "deferred", BaselineName: "before", Required: true},
		},
	})
	if err != nil || len(started.Pending) != 1 || started.Collection.Coverage().Pending != 2 {
		t.Fatalf("capture = %#v err=%v", started, err)
	}
	if started.Collection.Members[0].Status == CollectionMemberFailed || started.Collection.Members[1].Status == CollectionMemberFailed {
		t.Fatalf("admission saturation became terminal: %#v", started.Collection)
	}
	if _, err := svc.FinalizeCollectionCapture(context.Background(), 1, started.Pending[0]); err != nil {
		t.Fatal(err)
	}
	deferred, dispatched, err := svc.StartDeferredCollectionCapture(context.Background(), 1, started.Pending[0])
	if err != nil || !dispatched || deferred.Pending.Run.RunID == "" {
		t.Fatalf("deferred dispatch = %#v started=%v err=%v", deferred, dispatched, err)
	}
	if _, err := svc.FinalizeCollectionCapture(context.Background(), 1, deferred); err != nil {
		t.Fatal(err)
	}
	collection, err := svc.storage.LoadCollection(1, "agi", "before")
	if err != nil || !collection.Coverage().Complete() {
		t.Fatalf("deferred capture did not complete: %#v err=%v", collection, err)
	}
}

func TestCollectionCaptureAdmissionSaturationRetriesWithoutSibling(t *testing.T) {
	svc, exec := collectionService(t)
	exec.startErrs = []error{
		errors.New("resource_exhausted: test-genie admission is saturated (caller queued run capacity)"),
		nil,
	}
	repoDir := t.TempDir()
	started, err := svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{
		RepoID: 1, RepoDir: repoDir, Name: "single-member",
		Targets: []CollectionTarget{{Scenario: "web-console", BaselineName: "single-member", Required: true}},
	})
	if err != nil || len(started.Pending) != 0 || started.Collection.Coverage().Pending != 1 {
		t.Fatalf("saturated single-member capture = %#v err=%v", started, err)
	}
	if started.Collection.RepoDir != repoDir {
		t.Fatalf("deferred capture lost repo metadata: %#v", started.Collection)
	}

	// The first status read respects the durable retry lease rather than
	// hammering Test Genie while its admission queue is still full.
	if _, err := svc.GetCollectionCaptureStatus(context.Background(), 1, "agi", "single-member"); err != nil {
		t.Fatal(err)
	}
	if exec.calls != 1 {
		t.Fatalf("capture retried before its lease expired: calls=%d", exec.calls)
	}
	if _, err := svc.storage.UpdateCollectionMember(1, "agi", "single-member", "web-console", func(member *CollectionMember) error {
		member.UpdatedAt = time.Now().UTC().Add(-collectionCaptureDispatchLease)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	collection, err := svc.GetCollectionCaptureStatus(context.Background(), 1, "agi", "single-member")
	if err != nil {
		t.Fatal(err)
	}
	if exec.calls != 2 || collection.Members[0].RunID == "" || !collection.Coverage().Complete() {
		t.Fatalf("deferred single-member capture did not recover: %#v calls=%d", collection, exec.calls)
	}

	settled, err := svc.ResumeCollectionCapture(context.Background(), 1, "agi", "single-member")
	if err != nil || !settled.Coverage().Complete() {
		t.Fatalf("recovered single-member capture did not finalize: %#v err=%v", settled, err)
	}
}

func TestFinalizeCollectionCaptureRequeuesAdmissionSaturatedRun(t *testing.T) {
	svc, exec := collectionService(t)
	started, err := svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{
		RepoID: 1, RepoDir: t.TempDir(), Name: "terminal-admission-saturation",
		Targets: []CollectionTarget{{Scenario: "calendar", BaselineName: "terminal-admission-saturation", Required: true}},
	})
	if err != nil || len(started.Pending) != 1 {
		t.Fatalf("capture = %#v err=%v", started, err)
	}

	// The run already has a durable handoff, but Test Genie reports admission
	// saturation while that handoff is finalized. The collection must requeue
	// the member instead of converting transient backpressure into failure.
	exec.err = errors.New("resource_exhausted: test-genie admission is saturated (caller queued run capacity)")
	collection, err := svc.FinalizeCollectionCapture(context.Background(), 1, started.Pending[0])
	if err != nil {
		t.Fatalf("admission saturation should be recoverable: %v", err)
	}
	if collection.Members[0].Status != CollectionMemberPending || collection.Members[0].RunID != "" || collection.Coverage().Failed != 0 {
		t.Fatalf("admission-saturated run became terminal: %#v", collection)
	}

	// Once capacity returns and the retry lease expires, status reconciliation
	// dispatches and finalizes a fresh child run.
	exec.err = nil
	if _, err := svc.storage.UpdateCollectionMember(1, "agi", "terminal-admission-saturation", "calendar", func(member *CollectionMember) error {
		member.UpdatedAt = time.Now().UTC().Add(-collectionCaptureDispatchLease)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	recovered, err := svc.GetCollectionCaptureStatus(context.Background(), 1, "agi", "terminal-admission-saturation")
	if err != nil || !recovered.Coverage().Complete() {
		t.Fatalf("requeued capture did not recover: %#v err=%v", recovered, err)
	}
}

func TestDeferredCollectionCaptureStopsAfterRequiredFailure(t *testing.T) {
	svc, exec := collectionService(t)
	exec.startErrs = []error{
		nil,
		errors.New("resource_exhausted: test-genie admission is saturated (caller queued run capacity)"),
	}
	started, err := svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{
		RepoID: 1, RepoDir: t.TempDir(), Name: "before",
		Targets: []CollectionTarget{
			{Scenario: "failed", BaselineName: "before", Required: true},
			{Scenario: "still-pending", BaselineName: "before", Required: true},
		},
	})
	if err != nil || len(started.Pending) != 1 {
		t.Fatalf("capture = %#v err=%v", started, err)
	}
	if _, err := svc.FinalizeCollectionCapture(context.Background(), 1, started.Pending[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.storage.UpdateCollectionMember(1, "agi", "before", "failed", func(member *CollectionMember) error {
		member.Status = CollectionMemberFailed
		member.Error = "synthetic terminal failure"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	collection, err := svc.storage.LoadCollection(1, "agi", "before")
	if err != nil {
		t.Fatal(err)
	}
	if _, dispatched, err := svc.StartDeferredCollectionCapture(context.Background(), 1, started.Pending[0]); err != nil || dispatched {
		t.Fatalf("terminal failure dispatched deferred member: started=%v err=%v", dispatched, err)
	}
	collection, err = svc.storage.LoadCollection(1, "agi", "before")
	if err != nil {
		t.Fatal(err)
	}
	if exec.calls != 2 || collection.Members[1].RunID != "" || collection.Coverage().Pending != 1 {
		t.Fatalf("deferred member started after terminal failure: calls=%d collection=%#v", exec.calls, collection)
	}
}

func TestResumeCollectionCaptureReattachesDurablePendingMembers(t *testing.T) {
	svc, exec := collectionService(t)
	started, err := svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{RepoID: 1, RepoDir: t.TempDir(), Name: "before", Targets: []CollectionTarget{{Scenario: "plan-manager", BaselineName: "before", Required: true}}})
	if err != nil || len(started.Pending) != 1 {
		t.Fatalf("start = %#v err=%v", started, err)
	}
	resumed, err := svc.ResumeCollectionCapture(context.Background(), 1, "agi", "before")
	if err != nil || !resumed.Coverage().Complete() || exec.calls != 1 {
		t.Fatalf("resume = %#v err=%v calls=%d", resumed, err, exec.calls)
	}
}

func TestResumeCollectionCaptureRepairsPendingProjectionFromReadyChild(t *testing.T) {
	svc, _ := collectionService(t)
	started, err := svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{RepoID: 1, RepoDir: t.TempDir(), Name: "before", Targets: []CollectionTarget{{Scenario: "plan-manager", BaselineName: "before", Required: true}}})
	if err != nil || len(started.Pending) != 1 {
		t.Fatalf("start = %#v err=%v", started, err)
	}
	// Simulate a crash after the child commit and before the parent member
	// projection, followed by retention cleanup of the completed intent.
	if _, err := svc.FinalizeCapture(context.Background(), started.Pending[0].Pending); err != nil {
		t.Fatalf("finalize child: %v", err)
	}
	dir, err := svc.storage.branchDir(1, "plan-manager", "agi")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(svc.storage.snapshotIntentPath(dir, "before", started.Pending[0].Pending.Run.RunID)); err != nil {
		t.Fatal(err)
	}

	resumed, err := svc.ResumeCollectionCapture(context.Background(), 1, "agi", "before")
	if err != nil || !resumed.Coverage().Complete() || resumed.Members[0].Status != CollectionMemberReady {
		t.Fatalf("resume did not repair ready child projection: %#v err=%v", resumed, err)
	}
}

func TestCollectionCaptureStatusReconcilesTerminalFailureWithoutWait(t *testing.T) {
	svc, exec := collectionService(t)
	started, err := svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{
		RepoID: 1, RepoDir: t.TempDir(), Name: "before",
		Targets: []CollectionTarget{{Scenario: "plan-manager", BaselineName: "before", Required: true}},
	})
	if err != nil || len(started.Pending) != 1 {
		t.Fatalf("start collection capture = %#v err=%v", started, err)
	}
	// Simulate the lost async finalizer boundary: Test Genie has already made
	// the run terminal, but no client invokes the blocking collection wait.
	exec.statusInfo = &RunStatusInfo{Status: "failed", Terminal: true}
	exec.err = errors.New("terminal comprehensive failure")
	collection, err := svc.GetCollectionCaptureStatus(context.Background(), 1, "agi", "before")
	if err != nil || collection.Coverage().Failed != 1 || collection.Coverage().Pending != 0 {
		t.Fatalf("terminal child was not projected by status: %#v err=%v", collection.Coverage(), err)
	}
	if got := collection.Members[0].Error; !strings.Contains(got, "terminal comprehensive failure") {
		t.Fatalf("terminal failure detail = %q", got)
	}
	standing := CollectionCaptureStanding(collection)
	if standing.GetLifecycle() != "terminal" || standing.GetTerminalOutcome() != "failed" || standing.GetDirective() != "inspect" {
		t.Fatalf("failed capture standing = %#v", standing)
	}
}

func TestCollectionReanchorCannotNarrowExistingTargetSelection(t *testing.T) {
	svc, _ := collectionService(t)
	_, err := svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{
		RepoID: 1, RepoDir: t.TempDir(), Name: "before",
		Targets: []CollectionTarget{
			{Scenario: "first", BaselineName: "before", Required: true},
			{Scenario: "second", BaselineName: "before", Required: true},
		},
	})
	if err != nil {
		t.Fatalf("start collection capture: %v", err)
	}
	if _, err := svc.storage.UpdateCollectionMember(1, "agi", "before", "first", func(member *CollectionMember) error {
		member.Status = CollectionMemberFailed
		member.Error = "source drift"
		return nil
	}); err != nil {
		t.Fatalf("seed failed member: %v", err)
	}

	_, err = svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{
		RepoID: 1, RepoDir: t.TempDir(), Name: "before", AcknowledgeReanchor: true,
		Targets: []CollectionTarget{{Scenario: "first", BaselineName: "before", Required: true}},
	})
	if err == nil || !strings.Contains(err.Error(), "must preserve its existing target selection") {
		t.Fatalf("narrow re-anchor error = %v", err)
	}
	collection, err := svc.storage.LoadCollection(1, "agi", "before")
	if err != nil {
		t.Fatal(err)
	}
	if len(collection.Members) != 2 || collection.Generation != 1 {
		t.Fatalf("narrow re-anchor changed collection: %#v", collection)
	}
}

func TestResumeCollectionCaptureWaitsConcurrentlyAndReturnsTypedIncomplete(t *testing.T) { // [REQ:GCT-DURABLE-OPS-P0]
	secondDone := make(chan struct{})
	exec := &blockingFinalizeExecutor{firstAwaitStarted: make(chan struct{}), releaseFirst: make(chan struct{}), secondAwaitDone: secondDone}
	svc := NewService(Deps{Storage: newTestStorage(t), Exec: exec, Runs: &fakeRuns{}, CaptureGit: fixedGit(git.State{Sha: "abc", Branch: "agi"})})
	started, err := svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{
		RepoID: 1, RepoDir: t.TempDir(), Name: "before",
		Targets: []CollectionTarget{{Scenario: "blocked", BaselineName: "before", Required: true}, {Scenario: "terminal", BaselineName: "before", Required: true}},
	})
	if err != nil || len(started.Pending) != 2 {
		t.Fatalf("start = %#v err=%v", started, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	type resumeResult struct {
		collection CollectionManifest
		err        error
	}
	done := make(chan resumeResult, 1)
	go func() {
		collection, resumeErr := svc.ResumeCollectionCapture(ctx, 1, "agi", "before")
		done <- resumeResult{collection: collection, err: resumeErr}
	}()
	<-exec.firstAwaitStarted
	<-secondDone
	cancel()
	result := <-done
	err = result.err
	var incomplete *CollectionIncompleteError
	if !errors.As(err, &incomplete) || len(incomplete.PendingRunIDs) != 1 || incomplete.PendingRunIDs[0] != "run-1" {
		t.Fatalf("typed incomplete = %#v err=%v", incomplete, err)
	}
	if result.collection.Coverage().Ready != 1 || result.collection.Coverage().Pending != 1 {
		t.Fatalf("concurrent partial = %#v", result.collection.Coverage())
	}
}

func TestDeleteCollectionCleansAttachedSourceEvidence(t *testing.T) {
	svc, _ := collectionService(t)
	repo := t.TempDir()
	initSnapshotGitRepo(t, repo)
	if err := os.WriteFile(filepath.Join(repo, "dirty.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{
		RepoID: 1, RepoDir: repo, Name: "before", PathSelections: []string{"*.txt"},
		Targets: []CollectionTarget{{Scenario: "plan-manager", BaselineName: "before", Required: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.StorageLoadPathSnapshot(1, "agi", "before"); err != nil {
		t.Fatalf("attached snapshot missing: %v", err)
	}
	if err := svc.DeleteCollection(context.Background(), 1, "agi", "before"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.StorageLoadPathSnapshot(1, "agi", "before"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("attached snapshot remained after collection delete: %v", err)
	}
}

func TestCollectionDiffSelectionRejectsOutOfCollectionScenario(t *testing.T) {
	collection := sampleCollection()
	if _, err := selectCollectionMembers(collection, []string{"not-a-member"}); err == nil {
		t.Fatal("out-of-collection selection unexpectedly accepted")
	}
	members, err := selectCollectionMembers(collection, []string{"plan-manager"})
	if err != nil || len(members) != 1 || members[0].Scenario != "plan-manager" {
		t.Fatalf("narrow selection = %#v err=%v", members, err)
	}
}

func TestCollectionDiffOperationIsDurableAndSelectionIdempotent(t *testing.T) {
	svc, _ := collectionService(t)
	captured, err := svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{RepoID: 1, RepoDir: t.TempDir(), Name: "before", Targets: []CollectionTarget{{Scenario: "plan-manager", BaselineName: "before", Required: true}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, pending := range captured.Pending {
		if _, err := svc.FinalizeCollectionCapture(context.Background(), 1, pending); err != nil {
			t.Fatal(err)
		}
	}
	started, err := svc.StartCollectionDiff(context.Background(), StartCollectionDiffRequest{RepoID: 1, RepoDir: t.TempDir(), Branch: "agi", Name: "before", OperationID: "phase-1", Scenarios: []string{"plan-manager"}})
	if err != nil || len(started.Pending) != 1 || started.Operation.ID != "phase-1" || started.Operation.Members[0].Lifecycle != CollectionDiffChildAwaiting {
		t.Fatalf("start collection diff = %#v err=%v", started, err)
	}
	if _, err := svc.FinalizeCollectionDiff(context.Background(), 1, started.Pending[0]); err != nil {
		t.Fatal(err)
	}
	_, settled, err := svc.WaitCollectionDiff(context.Background(), 1, "agi", "before", "phase-1")
	if err != nil || len(settled.Members) != 1 || settled.Members[0].Status != "ready" || settled.Members[0].Lifecycle != CollectionDiffChildPassed {
		t.Fatalf("settled operation = %#v err=%v", settled, err)
	}
	resumed, err := svc.StartCollectionDiff(context.Background(), StartCollectionDiffRequest{RepoID: 1, RepoDir: t.TempDir(), Branch: "agi", Name: "before", OperationID: "phase-1", Scenarios: []string{"plan-manager"}})
	if err != nil || len(resumed.Pending) != 0 || resumed.Operation.ID != "phase-1" {
		t.Fatalf("idempotent operation = %#v err=%v", resumed, err)
	}
}

func TestCollectionDiffResumesPreDispatchGraphExactlyOnceAcrossConcurrentAgents(t *testing.T) {
	svc, exec := collectionService(t)
	captured, err := svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{
		RepoID: 1, RepoDir: t.TempDir(), Name: "before",
		Targets: []CollectionTarget{{Scenario: "plan-manager", BaselineName: "before", Required: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.FinalizeCollectionCapture(context.Background(), 1, captured.Pending[0]); err != nil {
		t.Fatal(err)
	}
	collection, err := svc.storage.LoadCollection(1, "agi", "before")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	// Simulate a server crash immediately after the complete parent graph was
	// persisted and before any child dispatch was attached.
	seed := CollectionDiffOperation{
		ID: "crash-before-dispatch", Collection: "before", Branch: "agi", CollectionSnapshot: collection,
		Members:   []CollectionDiffMember{{Scenario: "plan-manager", BaselineName: "before", Required: true, Status: "pending"}},
		CreatedAt: now, UpdatedAt: now, LastProgressAt: now, Lifecycle: "dispatching",
	}
	if err := svc.storage.SaveCollectionDiffOperation(1, seed, CreateOnly); err != nil {
		t.Fatal(err)
	}

	type result struct {
		started StartCollectionDiffResult
		err     error
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			started, startErr := svc.StartCollectionDiff(context.Background(), StartCollectionDiffRequest{RepoID: 1, RepoDir: t.TempDir(), Branch: "agi", Name: "before", OperationID: "crash-before-dispatch"})
			results <- result{started, startErr}
		}()
	}
	wg.Wait()
	close(results)
	for got := range results {
		if got.err != nil || len(got.started.Members) != 1 || got.started.Members[0].Status != "pending" || got.started.Members[0].RunID == "" {
			t.Fatalf("resume result = %#v err=%v", got.started, got.err)
		}
	}
	if exec.calls != 2 { // one capture run + exactly one resumed current diff run
		t.Fatalf("concurrent resume started duplicate Test Genie work: calls=%d", exec.calls)
	}
	operation, err := svc.storage.LoadCollectionDiffOperation(1, "agi", "before", "crash-before-dispatch")
	if err != nil || operation.Members[0].RunID == "" || operation.Members[0].Status != "pending" {
		t.Fatalf("durable resumed graph = %#v err=%v", operation, err)
	}
}

func TestCollectionDiffDispatchLeaseIsAtomicAcrossServiceInstances(t *testing.T) {
	svc, exec := collectionService(t)
	captured, err := svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{RepoID: 1, RepoDir: t.TempDir(), Name: "before", Targets: []CollectionTarget{{Scenario: "plan-manager", BaselineName: "before", Required: true}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.FinalizeCollectionCapture(context.Background(), 1, captured.Pending[0]); err != nil {
		t.Fatal(err)
	}
	peer := NewService(Deps{Storage: svc.storage, Exec: exec, Runs: &fakeRuns{}, CaptureGit: fixedGit(git.State{Sha: "abc", Branch: "agi"})})
	request := StartCollectionDiffRequest{RepoID: 1, RepoDir: t.TempDir(), Branch: "agi", Name: "before", OperationID: "two-servers"}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, server := range []*Service{svc, peer} {
		wg.Add(1)
		go func(server *Service) {
			defer wg.Done()
			_, err := server.StartCollectionDiff(context.Background(), request)
			errs <- err
		}(server)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	exec.mu.Lock()
	calls := exec.calls
	exec.mu.Unlock()
	if calls != 2 {
		t.Fatalf("two service instances dispatched duplicate child runs: calls=%d", calls)
	}
}

func TestCollectionDiffStatusRetriesExpiredDispatchLease(t *testing.T) {
	svc, exec := collectionService(t)
	captured, err := svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{
		RepoID: 1, RepoDir: t.TempDir(), Name: "before",
		Targets: []CollectionTarget{{Scenario: "plan-manager", BaselineName: "before", Required: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.FinalizeCollectionCapture(context.Background(), 1, captured.Pending[0]); err != nil {
		t.Fatal(err)
	}
	exec.startErr = errors.New("temporary test-genie outage")
	started, err := svc.StartCollectionDiff(context.Background(), StartCollectionDiffRequest{RepoID: 1, RepoDir: t.TempDir(), Branch: "agi", Name: "before", OperationID: "retry-dispatch"})
	if err != nil || started.Operation.Members[0].Status != "pending" || started.Operation.Members[0].RunID != "" || started.Operation.Members[0].DispatchAttempts != 1 {
		t.Fatalf("initial dispatch failure = %#v err=%v", started, err)
	}
	operation, err := svc.storage.LoadCollectionDiffOperation(1, "agi", "before", "retry-dispatch")
	if err != nil {
		t.Fatal(err)
	}
	operation.Members[0].DispatchLeaseExpiresAt = time.Now().Add(-time.Second)
	if err := svc.storage.SaveCollectionDiffOperation(1, operation, Overwrite); err != nil {
		t.Fatal(err)
	}
	exec.startErr = nil
	_, operation, _, err = svc.GetCollectionDiffStatus(context.Background(), 1, "agi", "before", "retry-dispatch")
	if err != nil || operation.Members[0].Status != "ready" || exec.calls != 2 {
		t.Fatalf("expired dispatch was not recovered: %#v calls=%d err=%v", operation, exec.calls, err)
	}
}

func TestCollectionDiffTerminalizesUnrecoverablePreDispatchOperation(t *testing.T) {
	svc, _ := collectionService(t)
	captured, err := svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{
		RepoID: 1, RepoDir: t.TempDir(), Name: "before",
		Targets: []CollectionTarget{{Scenario: "plan-manager", BaselineName: "before", Required: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.FinalizeCollectionCapture(context.Background(), 1, captured.Pending[0]); err != nil {
		t.Fatal(err)
	}
	collection, err := svc.storage.LoadCollection(1, "agi", "before")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	seed := CollectionDiffOperation{
		ID: "missing-repo-path", Collection: "before", Branch: "agi", CollectionSnapshot: collection,
		Members:   []CollectionDiffMember{{Scenario: "plan-manager", BaselineName: "before", Required: true, Status: "pending", Lifecycle: CollectionDiffChildDispatching}},
		CreatedAt: now, UpdatedAt: now, LastProgressAt: now, Lifecycle: "dispatching",
	}
	if err := svc.storage.SaveCollectionDiffOperation(1, seed, CreateOnly); err != nil {
		t.Fatal(err)
	}
	_, operation, standing, err := svc.GetCollectionDiffStatus(context.Background(), 1, "agi", "before", "missing-repo-path")
	if err != nil || operation.Members[0].Status != "failed" || operation.Members[0].Lifecycle != CollectionDiffChildFailed || standing.GetLifecycle() != "terminal" || !strings.Contains(operation.Members[0].Detail, "repository path") {
		t.Fatalf("unrecoverable pre-dispatch operation did not terminalize: %#v standing=%#v err=%v", operation, standing, err)
	}
}

func TestCollectionDiffTerminalizesAfterBoundedDispatchFailures(t *testing.T) {
	svc, exec := collectionService(t)
	captured, err := svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{RepoID: 1, RepoDir: t.TempDir(), Name: "before", Targets: []CollectionTarget{{Scenario: "plan-manager", BaselineName: "before", Required: true}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.FinalizeCollectionCapture(context.Background(), 1, captured.Pending[0]); err != nil {
		t.Fatal(err)
	}
	exec.startErr = errors.New("test-genie unavailable")
	request := StartCollectionDiffRequest{RepoID: 1, RepoDir: t.TempDir(), Branch: "agi", Name: "before", OperationID: "bounded-dispatch"}
	for attempt := 1; attempt <= maxCollectionDiffDispatchAttempts; attempt++ {
		started, startErr := svc.StartCollectionDiff(context.Background(), request)
		if startErr != nil {
			t.Fatalf("attempt %d start error: %v", attempt, startErr)
		}
		member := started.Operation.Members[0]
		if member.DispatchAttempts != attempt {
			t.Fatalf("attempt %d state = %#v", attempt, member)
		}
		if attempt < maxCollectionDiffDispatchAttempts {
			operation, loadErr := svc.storage.LoadCollectionDiffOperation(1, "agi", "before", "bounded-dispatch")
			if loadErr != nil {
				t.Fatal(loadErr)
			}
			operation.Members[0].DispatchLeaseExpiresAt = time.Now().Add(-time.Second)
			if saveErr := svc.storage.SaveCollectionDiffOperation(1, operation, Overwrite); saveErr != nil {
				t.Fatal(saveErr)
			}
		}
	}
	_, operation, standing, err := svc.GetCollectionDiffStatus(context.Background(), 1, "agi", "before", "bounded-dispatch")
	if err != nil || operation.Members[0].Status != "failed" || standing.GetLifecycle() != "terminal" || !strings.Contains(operation.Members[0].Detail, "attempt 3") {
		t.Fatalf("bounded dispatch did not terminalize: %#v standing=%#v err=%v", operation, standing, err)
	}
}

func TestCollectionDiffAdmissionSaturationRemainsPendingUntilCapacityReturns(t *testing.T) {
	svc, exec := collectionService(t)
	captured, err := svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{
		RepoID: 1, RepoDir: t.TempDir(), Name: "before",
		Targets: []CollectionTarget{{Scenario: "plan-manager", BaselineName: "before", Required: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.FinalizeCollectionCapture(context.Background(), 1, captured.Pending[0]); err != nil {
		t.Fatal(err)
	}
	exec.startErr = errors.New("resource_exhausted: test-genie admission is saturated (caller queued run capacity)")
	request := StartCollectionDiffRequest{RepoID: 1, RepoDir: t.TempDir(), Branch: "agi", Name: "before", OperationID: "admission-saturation"}
	started, err := svc.StartCollectionDiff(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	member := started.Operation.Members[0]
	if member.Status != "pending" || member.DispatchAttempts != 0 || member.RunID != "" {
		t.Fatalf("admission saturation should remain queued: %#v", member)
	}

	operation, err := svc.storage.LoadCollectionDiffOperation(1, "agi", "before", "admission-saturation")
	if err != nil {
		t.Fatal(err)
	}
	operation.Members[0].DispatchLeaseExpiresAt = time.Now().Add(-time.Second)
	if err := svc.storage.SaveCollectionDiffOperation(1, operation, Overwrite); err != nil {
		t.Fatal(err)
	}
	exec.startErr = nil
	_, operation, _, err = svc.GetCollectionDiffStatus(context.Background(), 1, "agi", "before", "admission-saturation")
	if err != nil || operation.Members[0].Status != "ready" || exec.calls != 2 {
		t.Fatalf("saturated dispatch did not recover after capacity returned: %#v calls=%d err=%v", operation, exec.calls, err)
	}
}

func TestTransientAdmissionSaturationIncludesWaitTimeout(t *testing.T) {
	if !isTransientAdmissionSaturation(errors.New("wait for test-genie admission: context deadline exceeded")) {
		t.Fatal("admission wait timeout should be treated as transient backpressure")
	}
	if isTransientAdmissionSaturation(errors.New("test-genie binary is missing")) {
		t.Fatal("unrelated dispatch failures must remain terminally bounded")
	}
}

func TestCollectionDiffStatusUsesImmutableSnapshotAfterCollectionDeletion(t *testing.T) {
	svc, _ := collectionService(t)
	captured, err := svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{
		RepoID: 1, RepoDir: t.TempDir(), Name: "before",
		Targets: []CollectionTarget{{Scenario: "plan-manager", BaselineName: "before", Required: true}},
	})
	if err != nil || len(captured.Pending) != 1 {
		t.Fatalf("capture = %#v err=%v", captured, err)
	}
	if _, err := svc.FinalizeCollectionCapture(context.Background(), 1, captured.Pending[0]); err != nil {
		t.Fatal(err)
	}
	started, err := svc.StartCollectionDiff(context.Background(), StartCollectionDiffRequest{
		RepoID: 1, RepoDir: t.TempDir(), Branch: "agi", Name: "before", OperationID: "survives-delete",
	})
	if err != nil || len(started.Pending) != 1 || started.Operation.CollectionSnapshot.Name != "before" {
		t.Fatalf("start diff = %#v err=%v", started, err)
	}
	if err := svc.DeleteCollection(context.Background(), 1, "agi", "before"); err != nil {
		t.Fatal(err)
	}

	collection, operation, _, err := svc.GetCollectionDiffStatus(context.Background(), 1, "agi", "before", "survives-delete")
	if err != nil {
		t.Fatalf("status after collection deletion: %v", err)
	}
	if collection.Name != "before" || len(operation.Members) != 1 || operation.Members[0].Status != "ready" || operation.Lifecycle != "terminal" {
		t.Fatalf("snapshot-backed operation = %#v collection=%#v", operation, collection)
	}
}

func TestCollectionDiffStatusDoesNotAdoptLaterCollectionMembers(t *testing.T) {
	svc, _ := collectionService(t)
	captured, err := svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{
		RepoID: 1, RepoDir: t.TempDir(), Name: "before",
		Targets: []CollectionTarget{{Scenario: "plan-manager", BaselineName: "before", Required: true}},
	})
	if err != nil || len(captured.Pending) != 1 {
		t.Fatalf("capture = %#v err=%v", captured, err)
	}
	if _, err := svc.FinalizeCollectionCapture(context.Background(), 1, captured.Pending[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.StartCollectionDiff(context.Background(), StartCollectionDiffRequest{
		RepoID: 1, RepoDir: t.TempDir(), Branch: "agi", Name: "before", OperationID: "does-not-expand",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ExtendCollection(context.Background(), ExtendCollectionRequest{
		RepoID: 1, RepoDir: t.TempDir(), Branch: "agi", Name: "before",
		Targets: []CollectionTarget{{Scenario: "git-control-tower", BaselineName: "before", Required: true}},
	}); err != nil {
		t.Fatal(err)
	}

	collection, _, _, err := svc.GetCollectionDiffStatus(context.Background(), 1, "agi", "before", "does-not-expand")
	if err != nil {
		t.Fatalf("status after collection expansion: %v", err)
	}
	if len(collection.Members) != 1 || collection.Members[0].Scenario != "plan-manager" {
		t.Fatalf("operation adopted later collection members: %#v", collection.Members)
	}
}

func TestCollectionDiffStatusReconcilesTerminalChildAfterFinalizerLoss(t *testing.T) {
	svc, _ := collectionService(t)
	captured, err := svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{
		RepoID: 1, RepoDir: t.TempDir(), Name: "before",
		Targets: []CollectionTarget{{Scenario: "plan-manager", BaselineName: "before", Required: true}},
	})
	if err != nil || len(captured.Pending) != 1 {
		t.Fatalf("capture = %#v err=%v", captured, err)
	}
	if _, err := svc.FinalizeCollectionCapture(context.Background(), 1, captured.Pending[0]); err != nil {
		t.Fatal(err)
	}
	started, err := svc.StartCollectionDiff(context.Background(), StartCollectionDiffRequest{
		RepoID: 1, RepoDir: t.TempDir(), Branch: "agi", Name: "before", OperationID: "recover-terminal",
	})
	if err != nil || len(started.Pending) != 1 {
		t.Fatalf("start diff = %#v err=%v", started, err)
	}

	if err := svc.ReconcileCollectionDiffOperations(context.Background(), 1); err != nil {
		t.Fatalf("background reconciliation failed: %v", err)
	}
	_, operation, standing, err := svc.GetCollectionDiffStatus(context.Background(), 1, "agi", "before", "recover-terminal")
	if err != nil || len(operation.Members) != 1 || operation.Members[0].Status != "ready" || standing.GetLifecycle() != "terminal" {
		t.Fatalf("status did not repair terminal child projection: %#v standing=%#v err=%v", operation, standing, err)
	}
}

func TestCollectionDiffStatusTerminalizesMissingDurableChildIntent(t *testing.T) {
	svc, _ := collectionService(t)
	captured, err := svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{
		RepoID: 1, RepoDir: t.TempDir(), Name: "before",
		Targets: []CollectionTarget{{Scenario: "plan-manager", BaselineName: "before", Required: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.FinalizeCollectionCapture(context.Background(), 1, captured.Pending[0]); err != nil {
		t.Fatal(err)
	}
	started, err := svc.StartCollectionDiff(context.Background(), StartCollectionDiffRequest{RepoID: 1, RepoDir: t.TempDir(), Branch: "agi", Name: "before", OperationID: "missing-intent"})
	if err != nil || len(started.Pending) != 1 {
		t.Fatalf("start diff = %#v err=%v", started, err)
	}
	dir, err := svc.storage.branchDir(1, "plan-manager", "agi")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(svc.storage.diffIntentPath(dir, "before", started.Members[0].RunID)); err != nil {
		t.Fatal(err)
	}
	_, operation, standing, err := svc.GetCollectionDiffStatus(context.Background(), 1, "agi", "before", "missing-intent")
	if err != nil || operation.Members[0].Status != "failed" || standing.GetLifecycle() != "terminal" {
		t.Fatalf("missing child was not terminalized: %#v standing=%#v err=%v", operation, standing, err)
	}
}

func TestCollectionDiffStatusTerminalizesMissingTestGenieRun(t *testing.T) {
	svc, exec := collectionService(t)
	captured, err := svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{
		RepoID: 1, RepoDir: t.TempDir(), Name: "before",
		Targets: []CollectionTarget{{Scenario: "plan-manager", BaselineName: "before", Required: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.FinalizeCollectionCapture(context.Background(), 1, captured.Pending[0]); err != nil {
		t.Fatal(err)
	}
	started, err := svc.StartCollectionDiff(context.Background(), StartCollectionDiffRequest{RepoID: 1, RepoDir: t.TempDir(), Branch: "agi", Name: "before", OperationID: "missing-run"})
	if err != nil || len(started.Pending) != 1 {
		t.Fatalf("start diff = %#v err=%v", started, err)
	}
	exec.statusInfo = &RunStatusInfo{Missing: true}
	_, operation, standing, err := svc.GetCollectionDiffStatus(context.Background(), 1, "agi", "before", "missing-run")
	if err != nil || operation.Members[0].Status != "failed" || standing.GetLifecycle() != "terminal" || !strings.Contains(operation.Members[0].Detail, "missing") {
		t.Fatalf("missing Test Genie run was not terminalized: %#v standing=%#v err=%v", operation, standing, err)
	}
}

func TestCollectionDiffStatusProjectsChildStanding(t *testing.T) {
	svc, exec := collectionService(t)
	captured, err := svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{RepoID: 1, RepoDir: t.TempDir(), Name: "before", Targets: []CollectionTarget{{Scenario: "plan-manager", BaselineName: "before", Required: true}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.FinalizeCollectionCapture(context.Background(), 1, captured.Pending[0]); err != nil {
		t.Fatal(err)
	}
	started, err := svc.StartCollectionDiff(context.Background(), StartCollectionDiffRequest{RepoID: 1, RepoDir: t.TempDir(), Branch: "agi", Name: "before", OperationID: "project-child"})
	if err != nil {
		t.Fatal(err)
	}
	exec.statusInfo = &RunStatusInfo{Status: "in_progress", Standing: &commonv1.OperationStanding{Owner: "test-genie", OperationId: started.Members[0].RunID, Lifecycle: "executing", Directive: "wait", ActivePhase: "workflow-health", EtaKnown: true, EstimatedRemainingSeconds: 91, RecommendedWaitSeconds: 45}}
	_, _, standing, err := svc.GetCollectionDiffStatus(context.Background(), 1, "agi", "before", "project-child")
	if err != nil {
		t.Fatal(err)
	}
	if standing.GetLifecycle() != "executing" || len(standing.GetChildren()) != 1 || standing.GetChildren()[0].GetActivePhase() != "workflow-health" || standing.GetChildren()[0].GetEstimatedRemainingSeconds() != 91 {
		t.Fatalf("aggregate did not project child standing: %#v", standing)
	}
}

func TestCollectionDiffStatusNeverProjectsRunBackedChildAsQueued(t *testing.T) {
	svc, exec := collectionService(t)
	captured, err := svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{RepoID: 1, RepoDir: t.TempDir(), Name: "before", Targets: []CollectionTarget{{Scenario: "plan-manager", BaselineName: "before", Required: true}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.FinalizeCollectionCapture(context.Background(), 1, captured.Pending[0]); err != nil {
		t.Fatal(err)
	}
	started, err := svc.StartCollectionDiff(context.Background(), StartCollectionDiffRequest{RepoID: 1, RepoDir: t.TempDir(), Branch: "agi", Name: "before", OperationID: "normalize-queued"})
	if err != nil {
		t.Fatal(err)
	}
	exec.statusInfo = &RunStatusInfo{Status: "queued", Standing: &commonv1.OperationStanding{Owner: "test-genie", OperationId: started.Members[0].RunID, Lifecycle: "queued", Directive: "wait"}}
	_, operation, standing, err := svc.GetCollectionDiffStatus(context.Background(), 1, "agi", "before", "normalize-queued")
	if err != nil || operation.Members[0].Lifecycle != CollectionDiffChildAwaiting || len(standing.Children) != 1 || standing.Children[0].GetLifecycle() != "executing" {
		t.Fatalf("run-backed child remained queued: operation=%#v standing=%#v err=%v", operation, standing, err)
	}
}

func TestCollectionDiffWaitsConcurrentlyAndReturnsTypedIncomplete(t *testing.T) { // [REQ:GCT-DURABLE-OPS-P0]
	svc, _ := collectionService(t)
	captured, err := svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{
		RepoID: 1, RepoDir: t.TempDir(), Name: "before",
		Targets: []CollectionTarget{{Scenario: "blocked", BaselineName: "before", Required: true}, {Scenario: "terminal", BaselineName: "before", Required: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, pending := range captured.Pending {
		if _, err := svc.FinalizeCollectionCapture(context.Background(), 1, pending); err != nil {
			t.Fatal(err)
		}
	}
	secondDone := make(chan struct{})
	exec := &blockingFinalizeExecutor{
		fakeExecutor:      fakeExecutor{calls: 2, result: terminalResult()},
		firstAwaitStarted: make(chan struct{}), releaseFirst: make(chan struct{}),
		secondAwaitDone: secondDone, blockedRunID: "run-3",
	}
	svc.exec = exec
	started, err := svc.StartCollectionDiff(context.Background(), StartCollectionDiffRequest{RepoID: 1, RepoDir: t.TempDir(), Branch: "agi", Name: "before", OperationID: "op-concurrent"})
	if err != nil || len(started.Pending) != 2 {
		t.Fatalf("start diff = %#v err=%v", started, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	type diffResult struct {
		operation CollectionDiffOperation
		err       error
	}
	done := make(chan diffResult, 1)
	go func() {
		_, operation, waitErr := svc.WaitCollectionDiff(ctx, 1, "agi", "before", "op-concurrent")
		done <- diffResult{operation: operation, err: waitErr}
	}()
	<-exec.firstAwaitStarted
	<-secondDone
	cancel()
	result := <-done
	var incomplete *CollectionIncompleteError
	if !errors.As(result.err, &incomplete) || len(incomplete.PendingRunIDs) != 1 || incomplete.PendingRunIDs[0] != "run-3" {
		t.Fatalf("typed diff incomplete = %#v err=%v", incomplete, result.err)
	}
	ready, pending := 0, 0
	for _, member := range result.operation.Members {
		switch member.Status {
		case "ready":
			ready++
		case "pending":
			pending++
		}
	}
	if ready != 1 || pending != 1 {
		t.Fatalf("concurrent diff checkpoints = %#v", result.operation.Members)
	}
}

func TestExtendCollectionIsAppendOnlyAndStartsOnlyNewMembers(t *testing.T) {
	svc, exec := collectionService(t)
	started, err := svc.StartCollectionCapture(context.Background(), StartCollectionCaptureRequest{RepoID: 1, RepoDir: t.TempDir(), Name: "before", Targets: []CollectionTarget{{Scenario: "plan-manager", BaselineName: "before", Required: true}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, pending := range started.Pending {
		if _, err := svc.FinalizeCollectionCapture(context.Background(), 1, pending); err != nil {
			t.Fatal(err)
		}
	}
	extended, err := svc.ExtendCollection(context.Background(), ExtendCollectionRequest{RepoID: 1, RepoDir: t.TempDir(), Name: "before", Targets: []CollectionTarget{{Scenario: "git-control-tower", BaselineName: "before", Required: true}}})
	if err != nil {
		t.Fatalf("extend: %v", err)
	}
	if exec.calls != 2 || len(extended.Pending) != 1 || len(extended.Collection.Members) != 2 {
		t.Fatalf("extension = %#v calls=%d", extended, exec.calls)
	}
	if _, err := svc.ExtendCollection(context.Background(), ExtendCollectionRequest{RepoID: 1, RepoDir: t.TempDir(), Name: "before", Targets: []CollectionTarget{{Scenario: "plan-manager", BaselineName: "replacement", Required: true}}}); err == nil {
		t.Fatal("existing member mutation unexpectedly accepted")
	}
}
