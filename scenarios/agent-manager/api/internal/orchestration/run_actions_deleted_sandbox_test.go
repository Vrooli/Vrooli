package orchestration

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil/mocks"
)

func TestDeletedSandboxReviewActionsExposeReadOnlyUnavailableReason(t *testing.T) {
	ctx := context.Background()
	repos, run := newPendingReviewRun(t, ctx)
	provider := mocks.NewFakeSandboxProvider()
	provider.GetFunc = func(_ context.Context, id uuid.UUID) (*sandbox.Sandbox, error) {
		return &sandbox.Sandbox{ID: id, Status: sandbox.SandboxStatusDeleted}, nil
	}
	o := New(repos.Profiles, repos.Tasks, repos.Runs, WithSandbox(provider))
	status := domain.RunStatusNeedsReview
	listed, err := o.ListRuns(ctx, RunListOptions{Status: &status})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("listed runs=%d, want 1", len(listed))
	}
	detail, err := o.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, got := range []*domain.Run{listed[0], detail} {
		if got.Actions == nil {
			t.Fatal("review actions missing")
		}
		if !got.Actions.CanReview {
			t.Fatal("deleted sandbox archive should remain available for read-only review")
		}
		if got.Actions.CanApprove || got.Actions.CanReject {
			t.Fatalf("deleted sandbox still offers decision actions: %+v", got.Actions)
		}
		if !strings.Contains(got.Actions.ReviewUnavailableReason, "sandbox was deleted") {
			t.Fatalf("missing deleted-sandbox reason: %q", got.Actions.ReviewUnavailableReason)
		}
	}
	persisted, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Status != domain.RunStatusNeedsReview || persisted.ApprovalState != domain.ApprovalStatePending || persisted.ErrorMsg != "" {
		t.Fatalf("projection changed persisted review state: status=%s approval=%s error=%q", persisted.Status, persisted.ApprovalState, persisted.ErrorMsg)
	}
}

func TestActiveSandboxPendingReviewKeepsDecisionActions(t *testing.T) {
	ctx := context.Background()
	repos, run := newPendingReviewRun(t, ctx)
	provider := mocks.NewFakeSandboxProvider()
	provider.GetFunc = func(_ context.Context, id uuid.UUID) (*sandbox.Sandbox, error) {
		return &sandbox.Sandbox{ID: id, Status: sandbox.SandboxStatusActive}, nil
	}
	o := New(repos.Profiles, repos.Tasks, repos.Runs, WithSandbox(provider))
	got, err := o.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Actions == nil || !got.Actions.CanApprove || !got.Actions.CanReject || !got.Actions.CanReview {
		t.Fatalf("active sandbox review actions changed: %+v", got.Actions)
	}
	if got.Actions.ReviewUnavailableReason != "" {
		t.Fatalf("active review got unavailable reason: %q", got.Actions.ReviewUnavailableReason)
	}
}

func TestReviewUnavailableRequiresConfirmedDeletedPendingManualReview(t *testing.T) {
	cases := []struct {
		name       string
		get        func(context.Context, uuid.UUID) (*sandbox.Sandbox, error)
		terminal   domain.RunTerminalClass
		stopReason domain.RunStopReason
		wantCalls  int
	}{
		{
			name: "provider error preserves ordinary actions",
			get: func(context.Context, uuid.UUID) (*sandbox.Sandbox, error) {
				return nil, context.DeadlineExceeded
			},
			wantCalls: 1,
		},
		{
			name: "missing sandbox response is not confirmed deletion",
			get: func(context.Context, uuid.UUID) (*sandbox.Sandbox, error) {
				return nil, nil
			},
			wantCalls: 1,
		},
		{
			name: "unknown sandbox status is not confirmed deletion",
			get: func(_ context.Context, id uuid.UUID) (*sandbox.Sandbox, error) {
				return &sandbox.Sandbox{ID: id, Status: sandbox.SandboxStatus("unknown")}, nil
			},
			wantCalls: 1,
		},
		{
			name: "mismatched sandbox ID is not confirmed deletion",
			get: func(context.Context, uuid.UUID) (*sandbox.Sandbox, error) {
				return &sandbox.Sandbox{ID: uuid.New(), Status: sandbox.SandboxStatusDeleted}, nil
			},
			wantCalls: 1,
		},
		{
			name:      "interruption review does not hydrate sandbox",
			get:       func(context.Context, uuid.UUID) (*sandbox.Sandbox, error) { panic("unexpected sandbox lookup") },
			terminal:  domain.RunTerminalClassInterruption,
			wantCalls: 0,
		},
		{
			name:       "crash review does not hydrate sandbox",
			get:        func(context.Context, uuid.UUID) (*sandbox.Sandbox, error) { panic("unexpected sandbox lookup") },
			stopReason: domain.RunStopReasonCrash,
			wantCalls:  0,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repos, run := newPendingReviewRun(t, ctx)
			if tt.terminal != "" || tt.stopReason != "" {
				run.TerminalClass = tt.terminal
				run.StopReason = tt.stopReason
				if err := repos.Runs.Update(ctx, run); err != nil {
					t.Fatalf("update run: %v", err)
				}
			}
			calls := 0
			provider := mocks.NewFakeSandboxProvider()
			provider.GetFunc = func(ctx context.Context, id uuid.UUID) (*sandbox.Sandbox, error) {
				calls++
				return tt.get(ctx, id)
			}
			o := New(repos.Profiles, repos.Tasks, repos.Runs, WithSandbox(provider))
			got, err := o.GetRun(ctx, run.ID)
			if err != nil {
				t.Fatalf("GetRun: %v", err)
			}
			if calls != tt.wantCalls {
				t.Fatalf("sandbox lookups=%d, want %d", calls, tt.wantCalls)
			}
			if got.Actions == nil || got.Actions.ReviewUnavailableReason != "" {
				t.Fatalf("unconfirmed deletion changed review actions: %+v", got.Actions)
			}
			if tt.terminal == "" && tt.stopReason == "" && (!got.Actions.CanApprove || !got.Actions.CanReject) {
				t.Fatalf("lookup uncertainty removed ordinary decision actions: %+v", got.Actions)
			}
		})
	}
}
