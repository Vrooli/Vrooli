package baseline

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"git-control-tower/internal/git"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func collectionService(t *testing.T) (*Service, *fakeExecutor) {
	t.Helper()
	exec := &fakeExecutor{result: ExecResult{Success: true, CompletedAt: time.Now().UTC(), TreeDigest: "tree", PhaseSetDigest: "phases", CaptureProfile: CaptureProfile, DescriptorSnapshotDigest: "descriptor", DescriptorSnapshotSchemaVersion: 1}}
	return NewService(Deps{Storage: newTestStorage(t), Exec: exec, Runs: &fakeRuns{}, CaptureGit: fixedGit(git.State{Sha: "abc", Branch: "agi"})}), exec
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
	if err != nil || len(started.Pending) != 1 || started.Operation.ID != "phase-1" {
		t.Fatalf("start collection diff = %#v err=%v", started, err)
	}
	if _, err := svc.FinalizeCollectionDiff(context.Background(), 1, started.Pending[0]); err != nil {
		t.Fatal(err)
	}
	_, settled, err := svc.WaitCollectionDiff(context.Background(), 1, "agi", "before", "phase-1")
	if err != nil || len(settled.Members) != 1 || settled.Members[0].Status != "ready" {
		t.Fatalf("settled operation = %#v err=%v", settled, err)
	}
	resumed, err := svc.StartCollectionDiff(context.Background(), StartCollectionDiffRequest{RepoID: 1, RepoDir: t.TempDir(), Branch: "agi", Name: "before", OperationID: "phase-1", Scenarios: []string{"plan-manager"}})
	if err != nil || len(resumed.Pending) != 0 || resumed.Operation.ID != "phase-1" {
		t.Fatalf("idempotent operation = %#v err=%v", resumed, err)
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
	if collection.Name != "before" || len(operation.Members) != 1 || operation.Members[0].Status != "pending" {
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

func TestCollectionDiffStatusReportsFinalizingTerminalChildWithoutReconciling(t *testing.T) {
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

	_, operation, standing, err := svc.GetCollectionDiffStatus(context.Background(), 1, "agi", "before", "recover-terminal")
	if err != nil || len(operation.Members) != 1 || operation.Members[0].Status != "pending" || standing.GetLifecycle() != "finalizing" {
		t.Fatalf("pure status should expose finalizing terminal child: %#v standing=%#v err=%v", operation, standing, err)
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
