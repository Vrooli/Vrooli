package orchestration

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil/mocks"
	"agent-manager/internal/repository"
	"agent-manager/internal/supervision"

	"github.com/google/uuid"
)

type failOnceRunUpdateRepository struct {
	repository.RunRepository
	mu       sync.Mutex
	failNext bool
}

func (r *failOnceRunUpdateRepository) Update(ctx context.Context, run *domain.Run) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failNext {
		r.failNext = false
		return errors.New("injected run state persistence failure")
	}
	return r.RunRepository.Update(ctx, run)
}

func TestRunApprovalRetriesFinalRunPersistenceAfterSandboxApply(t *testing.T) {
	ctx := context.Background()
	repos, run := newPendingReviewRun(t, ctx)
	baseRuns := repos.Runs
	runs := &failOnceRunUpdateRepository{RunRepository: baseRuns}
	sandboxID := *run.SandboxID
	patch := "same reviewed patch before and after terminal archive"
	applied := false
	approveCalls := 0
	provider := mocks.NewFakeSandboxProvider()
	provider.GetDiffFunc = func(_ context.Context, id uuid.UUID) (*sandbox.DiffResult, error) {
		diff := approvalTestDiff(id, patch)
		if applied {
			diff.ArchiveState = sandbox.ArchiveStateComplete
		}
		return diff, nil
	}
	provider.ApproveFunc = func(_ context.Context, req sandbox.ApproveRequest) (*sandbox.ApproveResult, error) {
		approveCalls++
		if req.ExpectedPatchSHA256 != approvalTestDiff(sandboxID, patch).PatchSHA256 {
			t.Fatalf("approval digest=%q, want exact current patch", req.ExpectedPatchSHA256)
		}
		applied = true
		return &sandbox.ApproveResult{Success: true, Applied: 3}, nil
	}
	publisher := &runReviewEventPublisher{}
	o := New(repos.Profiles, repos.Tasks, runs,
		WithSandbox(provider), WithApprovalEventPublisher(publisher),
		WithWebConsoleUIBase("http://localhost:21238"))
	current, err := baseRuns.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	requestRef := approvalTestEventID(t, o, current, approvalTestDiff(sandboxID, patch))
	resolution := supervision.DecisionResolution{
		State: "answered", Answer: "approve", AnsweredBy: "hub-actor",
		SourceEventType: RunApprovalDecisionEventType, RequesterRef: requestRef,
		Correlation: map[string]string{"run_id": run.ID.String()},
	}

	runs.failNext = true
	if handled, err := o.ResolveRunApprovalFromHub(ctx, resolution); !handled || err == nil {
		t.Fatalf("first delivery handled=%v err=%v; persistence failure must remain retryable", handled, err)
	}
	persisted, err := baseRuns.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Status != domain.RunStatusNeedsReview || persisted.ApprovalState != domain.ApprovalStatePending {
		t.Fatalf("first failed update changed persisted run: status=%s approval=%s", persisted.Status, persisted.ApprovalState)
	}
	if approveCalls != 1 || !applied {
		t.Fatalf("first delivery calls=%d applied=%v, want one completed sandbox effect", approveCalls, applied)
	}

	if handled, err := o.ResolveRunApprovalFromHub(ctx, resolution); !handled || err != nil {
		t.Fatalf("redelivery handled=%v err=%v, want successful idempotent completion", handled, err)
	}
	persisted, err = baseRuns.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Status != domain.RunStatusComplete || persisted.ApprovalState != domain.ApprovalStateApproved {
		t.Fatalf("redelivery did not persist approval: status=%s approval=%s", persisted.Status, persisted.ApprovalState)
	}
	if approveCalls != 2 {
		t.Fatalf("sandbox approve calls=%d, want idempotent retry through owner with same digest", approveCalls)
	}
}

func TestApproveRunPreservesProviderFailureAndPartialOutcome(t *testing.T) {
	t.Run("provider failure keeps run pending", func(t *testing.T) {
		ctx := context.Background()
		repos, run := newPendingReviewRun(t, ctx)
		provider := mocks.NewFakeSandboxProvider()
		provider.ApproveFunc = func(context.Context, sandbox.ApproveRequest) (*sandbox.ApproveResult, error) {
			return &sandbox.ApproveResult{Success: false, ErrorMsg: "apply succeeded but archive snapshot failed"}, nil
		}
		o := New(repos.Profiles, repos.Tasks, repos.Runs, WithSandbox(provider))
		result, err := o.ApproveRun(ctx, ApproveRequest{RunID: run.ID, Actor: "reviewer"})
		if err == nil || result != nil || !strings.Contains(err.Error(), "apply succeeded but archive snapshot failed") {
			t.Fatalf("ApproveRun result=%+v err=%v; provider failure must be preserved", result, err)
		}
		persisted, getErr := repos.Runs.Get(ctx, run.ID)
		if getErr != nil || persisted.Status != domain.RunStatusNeedsReview || persisted.ApprovalState != domain.ApprovalStatePending {
			t.Fatalf("provider failure changed persisted run: %+v err=%v", persisted, getErr)
		}
	})

	t.Run("successful partial result keeps review pending", func(t *testing.T) {
		ctx := context.Background()
		repos, run := newPendingReviewRun(t, ctx)
		provider := mocks.NewFakeSandboxProvider()
		provider.ApproveFunc = func(context.Context, sandbox.ApproveRequest) (*sandbox.ApproveResult, error) {
			return &sandbox.ApproveResult{Success: true, Applied: 1, Remaining: 2, IsPartial: true, CommitHash: "partial"}, nil
		}
		o := New(repos.Profiles, repos.Tasks, repos.Runs, WithSandbox(provider))
		result, err := o.ApproveRun(ctx, ApproveRequest{RunID: run.ID, Actor: "reviewer"})
		if err != nil || result == nil || !result.Success || result.Remaining != 2 || !result.IsPartial {
			t.Fatalf("ApproveRun result=%+v err=%v; expected partial outcome", result, err)
		}
		persisted, getErr := repos.Runs.Get(ctx, run.ID)
		if getErr != nil || persisted.Status != domain.RunStatusNeedsReview || persisted.ApprovalState != domain.ApprovalStatePartiallyApproved {
			t.Fatalf("partial result persisted incorrectly: %+v err=%v", persisted, getErr)
		}
	})
}

func TestPartialApproveDoesNotAdvanceRunOnProviderFailure(t *testing.T) {
	ctx := context.Background()
	repos, run := newPendingReviewRun(t, ctx)
	provider := mocks.NewFakeSandboxProvider()
	provider.PartialApproveFunc = func(context.Context, sandbox.PartialApproveRequest) (*sandbox.ApproveResult, error) {
		return &sandbox.ApproveResult{Success: false, ErrorMsg: "partial apply failed"}, nil
	}
	o := New(repos.Profiles, repos.Tasks, repos.Runs, WithSandbox(provider))
	result, err := o.PartialApprove(ctx, PartialApproveRequest{RunID: run.ID, FileIDs: []uuid.UUID{uuid.New()}, Actor: "reviewer"})
	if err == nil || result != nil || !strings.Contains(err.Error(), "partial apply failed") {
		t.Fatalf("PartialApprove result=%+v err=%v; provider failure must be preserved", result, err)
	}
	persisted, getErr := repos.Runs.Get(ctx, run.ID)
	if getErr != nil || persisted.Status != domain.RunStatusNeedsReview || persisted.ApprovalState != domain.ApprovalStatePending {
		t.Fatalf("provider failure changed persisted run: %+v err=%v", persisted, getErr)
	}
}
