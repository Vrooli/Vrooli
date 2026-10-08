package repo_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	"git-control-tower/internal/repo"
	"git-control-tower/internal/worktree"
	"git-control-tower/internal/worktree/mocks"
)

// All tests use FakeInspector — no real git.

func TestGetRepoStatus_LinkedWorktree(t *testing.T) {
	insp := &mocks.FakeInspector{
		IdentifyResult: worktree.Identity{
			IsLinkedWorktree:    true,
			CommonRepoRoot:      "/home/user/repo",
			WorktreeName:        "feature",
			WorktreeHead:        "abcdef",
			LinkedWorktreeCount: 3,
			Branch:              "feature",
		},
	}
	svc := repo.NewService(insp)
	got, err := svc.GetRepoStatus(context.Background(), "/home/user/wt/feature")
	if err != nil {
		t.Fatalf("GetRepoStatus: %v", err)
	}
	if !got.Identity.IsLinkedWorktree {
		t.Fatalf("expected IsLinkedWorktree=true")
	}
	if got.Branch != "feature" || got.Identity.WorktreeName != "feature" {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestGetRepoStatus_MainWorktree(t *testing.T) {
	insp := &mocks.FakeInspector{
		IdentifyResult: worktree.Identity{
			CommonRepoRoot:      "/home/user/repo",
			WorktreeName:        "repo",
			WorktreeHead:        "aaaa",
			LinkedWorktreeCount: 1,
			Branch:              "main",
		},
	}
	svc := repo.NewService(insp)
	got, err := svc.GetRepoStatus(context.Background(), "/home/user/repo")
	if err != nil {
		t.Fatalf("GetRepoStatus: %v", err)
	}
	if got.Identity.IsLinkedWorktree {
		t.Fatalf("expected IsLinkedWorktree=false")
	}
	if got.Branch != "main" {
		t.Fatalf("unexpected branch: %s", got.Branch)
	}
}

func TestGetRepoStatus_Detached(t *testing.T) {
	insp := &mocks.FakeInspector{
		IdentifyResult: worktree.Identity{
			Detached: true,
		},
	}
	svc := repo.NewService(insp)
	got, err := svc.GetRepoStatus(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("GetRepoStatus: %v", err)
	}
	if !got.Detached || got.Branch != "" {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestGetRepoStatus_RequiresPath(t *testing.T) {
	svc := repo.NewService(&mocks.FakeInspector{})
	_, err := svc.GetRepoStatus(context.Background(), "")
	if !errors.Is(err, repo.ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
	if repo.ErrorCodeFor(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", repo.ErrorCodeFor(err))
	}
}

func TestGetRepoStatus_PropagatesInspectorError(t *testing.T) {
	insp := &mocks.FakeInspector{IdentifyErr: errors.New("bad")}
	svc := repo.NewService(insp)
	_, err := svc.GetRepoStatus(context.Background(), "/repo")
	if err == nil {
		t.Fatalf("expected error")
	}
}
