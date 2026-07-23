package orchestration

import (
	"context"
	"testing"

	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/testutil"
	"agent-manager/internal/testutil/mocks"

	"github.com/google/uuid"
)

func TestSyncRunFromSandboxPersistsApprovedPartialAndRejectedStates(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "approval", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	newRun := func() *domain.Run {
		sandboxID := uuid.New()
		run := &domain.Run{ID: uuid.New(), TaskID: task.ID, SandboxID: &sandboxID, Status: domain.RunStatusNeedsReview, Phase: domain.RunPhaseAwaitingReview, ApprovalState: domain.ApprovalStatePending}
		if err := repos.Runs.Create(ctx, run); err != nil {
			t.Fatal(err)
		}
		return run
	}
	o := New(repos.Profiles, repos.Tasks, repos.Runs)

	approved := newRun()
	got, err := o.SyncRunFromSandbox(ctx, SandboxSyncRequest{RunID: approved.ID, SandboxID: approved.SandboxID, Status: " approved ", Actor: "operator"})
	if err != nil || got.Status != domain.RunStatusComplete || got.ApprovalState != domain.ApprovalStateApproved || got.ApprovedBy != "operator" || got.EndedAt == nil {
		t.Fatalf("approved=%+v err=%v", got, err)
	}
	persisted, err := repos.Runs.Get(ctx, approved.ID)
	if err != nil || persisted.Status != domain.RunStatusComplete || persisted.ApprovalState != domain.ApprovalStateApproved {
		t.Fatalf("persisted=%+v err=%v", persisted, err)
	}

	partial := newRun()
	got, err = o.SyncRunFromSandbox(ctx, SandboxSyncRequest{RunID: partial.ID, Status: "approved", IsPartial: true})
	if err != nil || got.Status != domain.RunStatusNeedsReview || got.ApprovalState != domain.ApprovalStatePartiallyApproved || got.ApprovedAt != nil {
		t.Fatalf("partial=%+v err=%v", got, err)
	}

	rejected := newRun()
	got, err = o.SyncRunFromSandbox(ctx, SandboxSyncRequest{RunID: rejected.ID, Status: "rejected", Reason: "conflict"})
	if err != nil || got.Status != domain.RunStatusFailed || got.ApprovalState != domain.ApprovalStateRejected || got.ApprovedBy != "workspace-sandbox" || got.ErrorMsg != "conflict" {
		t.Fatalf("rejected=%+v err=%v", got, err)
	}

	if _, err := o.SyncRunFromSandbox(ctx, SandboxSyncRequest{RunID: rejected.ID, SandboxID: ptrUUID(uuid.New()), Status: "approved"}); err == nil {
		t.Fatal("mismatched sandbox accepted")
	}
	if _, err := o.SyncRunFromSandbox(ctx, SandboxSyncRequest{RunID: rejected.ID, Status: "unknown"}); err == nil {
		t.Fatal("unknown status accepted")
	}
}

func TestApprovalStateHelpersPersistTerminalAndPartialTransitions(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "helpers", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	newRun := func() *domain.Run {
		run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusNeedsReview, Phase: domain.RunPhaseAwaitingReview, ApprovalState: domain.ApprovalStatePending}
		if err := repos.Runs.Create(ctx, run); err != nil {
			t.Fatal(err)
		}
		return run
	}
	o := New(repos.Profiles, repos.Tasks, repos.Runs)
	approved := newRun()
	if err := o.markRunApproved(ctx, approved, "operator"); err != nil {
		t.Fatal(err)
	}
	if approved.Status != domain.RunStatusComplete || approved.ApprovalState != domain.ApprovalStateApproved || approved.ApprovedAt == nil {
		t.Fatalf("approved=%+v", approved)
	}
	rejected := newRun()
	if err := o.markRunRejected(ctx, rejected, "operator", "unsafe"); err != nil {
		t.Fatal(err)
	}
	if rejected.Status != domain.RunStatusFailed || rejected.ApprovalState != domain.ApprovalStateRejected || rejected.ErrorMsg != "unsafe" {
		t.Fatalf("rejected=%+v", rejected)
	}
	partial := newRun()
	if err := o.markRunPartiallyApproved(ctx, partial); err != nil {
		t.Fatal(err)
	}
	if partial.Status != domain.RunStatusNeedsReview || partial.ApprovalState != domain.ApprovalStatePartiallyApproved {
		t.Fatalf("partial=%+v", partial)
	}
	mapped := mapApproveResult(&sandbox.ApproveResult{Success: true, Applied: 2, Remaining: 1, IsPartial: true, CommitHash: "abc"})
	if !mapped.Success || mapped.Applied != 2 || mapped.Remaining != 1 || !mapped.IsPartial || mapped.CommitHash != "abc" {
		t.Fatalf("mapped=%+v", mapped)
	}
	if approvalLog() == nil {
		t.Fatal("approval logger is nil")
	}
}

func TestPartialApproveAppliesSelectedFilesAndPersistsResultingState(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "partial approval", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name      string
		remaining int
		wantState domain.ApprovalState
		wantRun   domain.RunStatus
	}{
		{name: "all files applied", remaining: 0, wantState: domain.ApprovalStateApproved, wantRun: domain.RunStatusComplete},
		{name: "files remain", remaining: 2, wantState: domain.ApprovalStatePartiallyApproved, wantRun: domain.RunStatusNeedsReview},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sandboxID := uuid.New()
			run := &domain.Run{ID: uuid.New(), TaskID: task.ID, SandboxID: &sandboxID, Status: domain.RunStatusNeedsReview, Phase: domain.RunPhaseAwaitingReview, ApprovalState: domain.ApprovalStatePending}
			if err := repos.Runs.Create(ctx, run); err != nil {
				t.Fatal(err)
			}
			var gotRequest sandbox.PartialApproveRequest
			provider := mocks.NewFakeSandboxProvider()
			provider.PartialApproveFunc = func(_ context.Context, req sandbox.PartialApproveRequest) (*sandbox.ApproveResult, error) {
				gotRequest = req
				return &sandbox.ApproveResult{Success: true, Applied: 1, Remaining: tc.remaining, IsPartial: tc.remaining > 0, CommitHash: "commit"}, nil
			}
			o := New(repos.Profiles, repos.Tasks, repos.Runs, WithSandbox(provider))
			fileID := uuid.New()
			result, err := o.PartialApprove(ctx, PartialApproveRequest{RunID: run.ID, FileIDs: []uuid.UUID{fileID}, Actor: "reviewer", CommitMsg: "apply selected"})
			if err != nil {
				t.Fatalf("partial approve: %v", err)
			}
			if gotRequest.SandboxID != sandboxID || len(gotRequest.FileIDs) != 1 || gotRequest.FileIDs[0] != fileID || gotRequest.Actor != "reviewer" || gotRequest.CommitMsg != "apply selected" {
				t.Fatalf("sandbox request=%+v", gotRequest)
			}
			if !result.Success || result.Remaining != tc.remaining || result.CommitHash != "commit" {
				t.Fatalf("result=%+v", result)
			}
			persisted, err := repos.Runs.Get(ctx, run.ID)
			if err != nil {
				t.Fatal(err)
			}
			if persisted.ApprovalState != tc.wantState || persisted.Status != tc.wantRun {
				t.Fatalf("persisted run=%+v", persisted)
			}
		})
	}
}

func TestApproveAndRejectRunCoordinateSandboxAndPersistState(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "approval transitions", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	newReviewRun := func() (*domain.Run, uuid.UUID) {
		sandboxID := uuid.New()
		run := &domain.Run{ID: uuid.New(), TaskID: task.ID, SandboxID: &sandboxID, Status: domain.RunStatusNeedsReview, Phase: domain.RunPhaseAwaitingReview, ApprovalState: domain.ApprovalStatePending}
		if err := repos.Runs.Create(ctx, run); err != nil {
			t.Fatal(err)
		}
		return run, sandboxID
	}

	t.Run("approve", func(t *testing.T) {
		run, sandboxID := newReviewRun()
		provider := mocks.NewFakeSandboxProvider()
		var got sandbox.ApproveRequest
		provider.ApproveFunc = func(_ context.Context, req sandbox.ApproveRequest) (*sandbox.ApproveResult, error) {
			got = req
			return &sandbox.ApproveResult{Success: true, Applied: 3, CommitHash: "approved-commit"}, nil
		}
		o := New(repos.Profiles, repos.Tasks, repos.Runs, WithSandbox(provider))
		result, err := o.ApproveRun(ctx, ApproveRequest{RunID: run.ID, Actor: "reviewer", CommitMsg: "approved", Force: true})
		if err != nil {
			t.Fatalf("approve: %v", err)
		}
		if got.SandboxID != sandboxID || got.Actor != "reviewer" || got.CommitMsg != "approved" || !got.Force || !result.Success || result.CommitHash != "approved-commit" {
			t.Fatalf("sandbox request=%+v result=%+v", got, result)
		}
		persisted, err := repos.Runs.Get(ctx, run.ID)
		if err != nil {
			t.Fatal(err)
		}
		if persisted.Status != domain.RunStatusComplete || persisted.ApprovalState != domain.ApprovalStateApproved || persisted.ApprovedBy != "reviewer" {
			t.Fatalf("persisted run=%+v", persisted)
		}
	})

	t.Run("reject", func(t *testing.T) {
		run, sandboxID := newReviewRun()
		provider := mocks.NewFakeSandboxProvider()
		var rejectedID, deletedID uuid.UUID
		var actor string
		provider.RejectFunc = func(_ context.Context, id uuid.UUID, gotActor string) error {
			rejectedID, actor = id, gotActor
			return nil
		}
		provider.DeleteFunc = func(_ context.Context, id uuid.UUID) error {
			deletedID = id
			return nil
		}
		o := New(repos.Profiles, repos.Tasks, repos.Runs, WithSandbox(provider))
		if err := o.RejectRun(ctx, run.ID, "reviewer", "unsafe change"); err != nil {
			t.Fatalf("reject: %v", err)
		}
		if rejectedID != sandboxID || deletedID != sandboxID || actor != "reviewer" {
			t.Fatalf("reject sandbox calls rejected=%s deleted=%s actor=%q", rejectedID, deletedID, actor)
		}
		persisted, err := repos.Runs.Get(ctx, run.ID)
		if err != nil {
			t.Fatal(err)
		}
		if persisted.Status != domain.RunStatusFailed || persisted.ApprovalState != domain.ApprovalStateRejected || persisted.ErrorMsg != "unsafe change" {
			t.Fatalf("persisted run=%+v", persisted)
		}
	})
}

func ptrUUID(id uuid.UUID) *uuid.UUID { return &id }
