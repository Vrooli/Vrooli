package worktree_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	"git-control-tower/internal/worktree"
	"git-control-tower/internal/worktree/mocks"
)

// All tests in this file exercise the seams via FakeInspector / FakeMutator.
// NO real git is ever invoked.

func TestServiceList(t *testing.T) {
	want := []worktree.Worktree{
		{Path: "/repo", Name: "repo", Branch: "main", IsMain: true},
		{Path: "/tmp/feature", Name: "feature", Branch: "feature"},
	}
	insp := &mocks.FakeInspector{ListResult: want}
	svc := worktree.NewService(insp, nil)

	got, err := svc.List(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 || got[1].Branch != "feature" {
		t.Fatalf("unexpected result: %+v", got)
	}
	if len(insp.ListCalls) != 1 || insp.ListCalls[0] != "/repo" {
		t.Fatalf("expected one list call for /repo, got %v", insp.ListCalls)
	}
}

func TestServiceList_RequiresRepoPath(t *testing.T) {
	svc := worktree.NewService(&mocks.FakeInspector{}, nil)
	_, err := svc.List(context.Background(), "")
	if !errors.Is(err, worktree.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
	if worktree.ErrorCodeFor(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected InvalidArgument code, got %v", worktree.ErrorCodeFor(err))
	}
}

func TestServiceGet(t *testing.T) {
	insp := &mocks.FakeInspector{
		ListResult: []worktree.Worktree{
			{Path: "/repo", Name: "repo", IsMain: true},
			{Path: "/tmp/feature", Name: "feature"},
		},
	}
	svc := worktree.NewService(insp, nil)

	got, err := svc.Get(context.Background(), "/repo", "/tmp/feature")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "feature" {
		t.Fatalf("expected feature, got %+v", got)
	}
}

func TestServiceGet_NotFound(t *testing.T) {
	insp := &mocks.FakeInspector{
		ListResult: []worktree.Worktree{{Path: "/repo", IsMain: true}},
	}
	svc := worktree.NewService(insp, nil)

	_, err := svc.Get(context.Background(), "/repo", "/nope")
	if !errors.Is(err, worktree.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if worktree.ErrorCodeFor(err) != connect.CodeNotFound {
		t.Fatalf("expected NotFound code, got %v", worktree.ErrorCodeFor(err))
	}
}

func TestServiceIdentify(t *testing.T) {
	insp := &mocks.FakeInspector{
		IdentifyResult: worktree.Identity{
			IsLinkedWorktree: true,
			CommonRepoRoot:   "/repo",
			WorktreeName:     "feature",
			WorktreeHead:     "abc",
			Branch:           "feature",
		},
	}
	svc := worktree.NewService(insp, nil)
	id, err := svc.Identify(context.Background(), "/tmp/feature")
	if err != nil {
		t.Fatalf("Identify: %v", err)
	}
	if !id.IsLinkedWorktree || id.WorktreeName != "feature" || id.Branch != "feature" {
		t.Fatalf("unexpected identity: %+v", id)
	}
}

func TestServiceClaimedBranches(t *testing.T) {
	insp := &mocks.FakeInspector{
		ClaimedBranchesResult: map[string]string{"feature": "/tmp/feature"},
	}
	svc := worktree.NewService(insp, nil)
	got, err := svc.ClaimedBranches(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("ClaimedBranches: %v", err)
	}
	if got["feature"] != "/tmp/feature" {
		t.Fatalf("expected feature->/tmp/feature, got %v", got)
	}
}

func TestServiceCreate_ExistingBranch(t *testing.T) {
	mut := &mocks.FakeMutator{}
	svc := worktree.NewService(nil, mut)
	got, err := svc.Create(context.Background(), worktree.CreateInput{
		RepoPath:        "/repo",
		NewWorktreePath: "/tmp/feature",
		ExistingBranch:  "feature",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.Path != "/tmp/feature" || got.Branch != "feature" {
		t.Fatalf("unexpected: %+v", got)
	}
	if len(mut.AddCalls) != 1 || mut.AddCalls[0].ExistingBranch != "feature" {
		t.Fatalf("expected one Add call for feature, got %+v", mut.AddCalls)
	}
}

func TestServiceCreate_NewBranch(t *testing.T) {
	mut := &mocks.FakeMutator{}
	svc := worktree.NewService(nil, mut)
	got, err := svc.Create(context.Background(), worktree.CreateInput{
		RepoPath:        "/repo",
		NewWorktreePath: "/tmp/new",
		NewBranchName:   "topic",
		NewBranchStart:  "main",
		Track:           true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.Branch != "topic" {
		t.Fatalf("expected branch=topic, got %+v", got)
	}
	if mut.AddCalls[0].NewBranchName != "topic" || mut.AddCalls[0].NewBranchStart != "main" || !mut.AddCalls[0].Track {
		t.Fatalf("unexpected call: %+v", mut.AddCalls[0])
	}
}

func TestServiceCreate_DetachedCommit(t *testing.T) {
	mut := &mocks.FakeMutator{}
	svc := worktree.NewService(nil, mut)
	got, err := svc.Create(context.Background(), worktree.CreateInput{
		RepoPath:        "/repo",
		NewWorktreePath: "/tmp/det",
		Commit:          "abc",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !got.Detached || got.HeadCommit != "abc" {
		t.Fatalf("expected detached at abc, got %+v", got)
	}
}

func TestServiceCreate_RejectsMultipleSources(t *testing.T) {
	mut := &mocks.FakeMutator{}
	svc := worktree.NewService(nil, mut)
	_, err := svc.Create(context.Background(), worktree.CreateInput{
		RepoPath:        "/repo",
		NewWorktreePath: "/tmp/x",
		ExistingBranch:  "a",
		Commit:          "b",
	})
	if !errors.Is(err, worktree.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
	if len(mut.AddCalls) != 0 {
		t.Fatalf("Mutator must not be called when validation fails")
	}
}

func TestServiceCreate_RejectsNoSource(t *testing.T) {
	mut := &mocks.FakeMutator{}
	svc := worktree.NewService(nil, mut)
	_, err := svc.Create(context.Background(), worktree.CreateInput{
		RepoPath:        "/repo",
		NewWorktreePath: "/tmp/x",
	})
	if !errors.Is(err, worktree.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestServiceCreate_RejectsTrackWithoutNewBranch(t *testing.T) {
	mut := &mocks.FakeMutator{}
	svc := worktree.NewService(nil, mut)
	_, err := svc.Create(context.Background(), worktree.CreateInput{
		RepoPath:        "/repo",
		NewWorktreePath: "/tmp/x",
		ExistingBranch:  "main",
		Track:           true,
	})
	if !errors.Is(err, worktree.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for track+existing, got %v", err)
	}
}

func TestServiceRemove_RefusesMain(t *testing.T) {
	insp := &mocks.FakeInspector{
		ListResult: []worktree.Worktree{{Path: "/repo", Name: "repo", IsMain: true}},
	}
	mut := &mocks.FakeMutator{}
	svc := worktree.NewService(insp, mut)
	err := svc.Remove(context.Background(), "/repo", "/repo", true)
	if !errors.Is(err, worktree.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
	if len(mut.RemoveCalls) != 0 {
		t.Fatalf("Mutator must not be called when removing main")
	}
}

func TestServiceRemove_PropagatesNotFound(t *testing.T) {
	insp := &mocks.FakeInspector{
		ListResult: []worktree.Worktree{{Path: "/repo", IsMain: true}},
	}
	mut := &mocks.FakeMutator{}
	svc := worktree.NewService(insp, mut)
	err := svc.Remove(context.Background(), "/repo", "/tmp/missing", false)
	if !errors.Is(err, worktree.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if len(mut.RemoveCalls) != 0 {
		t.Fatalf("Mutator must not be called when get fails with NotFound")
	}
}

func TestServiceRemove_DispatchesToMutator(t *testing.T) {
	insp := &mocks.FakeInspector{
		ListResult: []worktree.Worktree{
			{Path: "/repo", IsMain: true},
			{Path: "/tmp/feature", Name: "feature"},
		},
	}
	mut := &mocks.FakeMutator{}
	svc := worktree.NewService(insp, mut)
	if err := svc.Remove(context.Background(), "/repo", "/tmp/feature", true); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if len(mut.RemoveCalls) != 1 || !mut.RemoveCalls[0].Force {
		t.Fatalf("expected one Remove with force, got %+v", mut.RemoveCalls)
	}
}

func TestServiceLock(t *testing.T) {
	mut := &mocks.FakeMutator{}
	svc := worktree.NewService(nil, mut)
	got, err := svc.Lock(context.Background(), worktree.LockInput{
		RepoPath:     "/repo",
		WorktreePath: "/tmp/feature",
		Reason:       "agents in flight",
	})
	if err != nil {
		t.Fatalf("Lock: %v", err)
	}
	if !got.Locked || got.LockReason != "agents in flight" {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestServiceUnlock(t *testing.T) {
	mut := &mocks.FakeMutator{}
	svc := worktree.NewService(nil, mut)
	if _, err := svc.Unlock(context.Background(), "/repo", "/tmp/feature"); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	if len(mut.UnlockCalls) != 1 {
		t.Fatalf("expected one unlock call")
	}
}

func TestServiceMove(t *testing.T) {
	mut := &mocks.FakeMutator{}
	svc := worktree.NewService(nil, mut)
	got, err := svc.Move(context.Background(), worktree.MoveInput{
		RepoPath:        "/repo",
		WorktreePath:    "/tmp/old",
		NewWorktreePath: "/tmp/new",
	})
	if err != nil {
		t.Fatalf("Move: %v", err)
	}
	if got.Path != "/tmp/new" {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestServiceMove_RejectsSameTarget(t *testing.T) {
	mut := &mocks.FakeMutator{}
	svc := worktree.NewService(nil, mut)
	_, err := svc.Move(context.Background(), worktree.MoveInput{
		RepoPath:        "/repo",
		WorktreePath:    "/tmp/x",
		NewWorktreePath: "/tmp/x",
	})
	if !errors.Is(err, worktree.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestServicePrune(t *testing.T) {
	mut := &mocks.FakeMutator{PruneResult: worktree.PruneResult{PrunedPaths: []string{"/old"}}}
	svc := worktree.NewService(nil, mut)
	got, err := svc.Prune(context.Background(), worktree.PruneInput{RepoPath: "/repo"})
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if len(got.PrunedPaths) != 1 || got.PrunedPaths[0] != "/old" {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestServiceWithoutInspector(t *testing.T) {
	svc := worktree.NewService(nil, &mocks.FakeMutator{})
	if _, err := svc.List(context.Background(), "/repo"); err == nil {
		t.Fatalf("expected error when inspector seam missing")
	}
}

func TestServiceWithoutMutator(t *testing.T) {
	svc := worktree.NewService(&mocks.FakeInspector{}, nil)
	if err := svc.Remove(context.Background(), "/repo", "/tmp/x", false); err == nil {
		t.Fatalf("expected error when mutator seam missing")
	}
}

func TestErrorCodeForUnknown(t *testing.T) {
	if c := worktree.ErrorCodeFor(errors.New("boom")); c != connect.CodeInternal {
		t.Fatalf("expected Internal, got %v", c)
	}
}

func TestToConnectError(t *testing.T) {
	if worktree.ToConnectError(nil) != nil {
		t.Fatalf("nil should pass through")
	}
	err := worktree.ToConnectError(errors.New("kaboom"))
	if err == nil {
		t.Fatalf("expected wrapped error")
	}
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("expected Internal, got %v", connect.CodeOf(err))
	}
}
