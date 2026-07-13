package baseline

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"git-control-tower/internal/git"
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

func TestDeleteCollectionCleansAttachedSourceEvidence(t *testing.T) {
	svc, _ := collectionService(t)
	repo := t.TempDir()
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
	_, settled, err := svc.GetCollectionDiff(context.Background(), 1, "agi", "before", "phase-1", true)
	if err != nil || len(settled.Members) != 1 || settled.Members[0].Status != "ready" {
		t.Fatalf("settled operation = %#v err=%v", settled, err)
	}
	resumed, err := svc.StartCollectionDiff(context.Background(), StartCollectionDiffRequest{RepoID: 1, RepoDir: t.TempDir(), Branch: "agi", Name: "before", OperationID: "phase-1", Scenarios: []string{"plan-manager"}})
	if err != nil || len(resumed.Pending) != 0 || resumed.Operation.ID != "phase-1" {
		t.Fatalf("idempotent operation = %#v err=%v", resumed, err)
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
