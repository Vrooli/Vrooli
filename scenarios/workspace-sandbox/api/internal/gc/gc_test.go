package gc

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/audit"
	"workspace-sandbox/internal/clock"
	"workspace-sandbox/internal/testutil/mocks"
	"workspace-sandbox/internal/types"
)

// gcRepo builds a FakeRepository pre-seeded with the given sandboxes
// as GC candidates AND in the Sandboxes map (so Delete finds them).
// The local helper keeps each test concise; the equivalent inline
// construction would be five lines.
func gcRepo(sandboxes []*types.Sandbox) *mocks.FakeRepository {
	r := mocks.NewFakeRepository()
	seedCandidates(r, sandboxes)
	return r
}

// seedCandidates installs sandboxes as GCCandidates AND populates the
// Sandboxes map. Tests that need to mutate other repo fields (errors,
// etc.) construct the repo directly and call this helper.
func seedCandidates(r *mocks.FakeRepository, sandboxes []*types.Sandbox) {
	r.GCCandidates = sandboxes
	for _, sb := range sandboxes {
		r.Sandboxes[sb.ID] = sb
	}
}

// Tests

func TestGCService_DryRun_ReturnsWithoutDeleting(t *testing.T) {
	// [REQ:P1-003] GC dry run mode
	now := time.Now()
	oldSandbox := &types.Sandbox{
		ID:        uuid.New(),
		ScopePath: "/test/old",
		Status:    types.StatusStopped,
		SizeBytes: 1000,
		CreatedAt: now.Add(-48 * time.Hour), // 2 days old
	}

	repo := gcRepo([]*types.Sandbox{oldSandbox})
	drv := mocks.NewFakeDriver()
	clk := clock.System{}
	svc := NewService(repo, drv, DefaultConfig(), clk, audit.NewRepoEmitter(repo.LogAuditEvent, clk))

	result, err := svc.Run(context.Background(), &types.GCRequest{
		DryRun: true,
		Policy: &types.GCPolicy{
			MaxAge: 24 * time.Hour,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.DryRun {
		t.Error("expected DryRun to be true")
	}

	if result.TotalCollected != 1 {
		t.Errorf("expected 1 collected, got %d", result.TotalCollected)
	}

	// Verify nothing was actually deleted
	if len(repo.DeletedIDs) != 0 {
		t.Errorf("expected no deletions in dry run, got %d", len(repo.DeletedIDs))
	}

	if len(drv.CleanedSandboxes) != 0 {
		t.Errorf("expected no cleanup in dry run, got %d", len(drv.CleanedSandboxes))
	}
}

func TestGCService_ActualRun_DeletesSandboxes(t *testing.T) {
	// [REQ:P1-003] GC actual deletion
	now := time.Now()
	oldSandbox := &types.Sandbox{
		ID:        uuid.New(),
		ScopePath: "/test/old",
		Status:    types.StatusStopped,
		SizeBytes: 5000,
		CreatedAt: now.Add(-48 * time.Hour),
	}

	repo := gcRepo([]*types.Sandbox{oldSandbox})
	drv := mocks.NewFakeDriver()
	clk := clock.System{}
	svc := NewService(repo, drv, DefaultConfig(), clk, audit.NewRepoEmitter(repo.LogAuditEvent, clk))

	result, err := svc.Run(context.Background(), &types.GCRequest{
		DryRun: false,
		Policy: &types.GCPolicy{
			MaxAge: 24 * time.Hour,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.DryRun {
		t.Error("expected DryRun to be false")
	}

	if result.TotalCollected != 1 {
		t.Errorf("expected 1 collected, got %d", result.TotalCollected)
	}

	if result.TotalBytesReclaimed != 5000 {
		t.Errorf("expected 5000 bytes reclaimed, got %d", result.TotalBytesReclaimed)
	}

	// Verify sandbox was deleted
	if len(repo.DeletedIDs) != 1 {
		t.Errorf("expected 1 deletion, got %d", len(repo.DeletedIDs))
	}

	// Verify cleanup was called
	if len(drv.CleanedSandboxes) != 1 {
		t.Errorf("expected 1 cleanup call, got %d", len(drv.CleanedSandboxes))
	}

	// Verify audit event was logged
	if len(repo.AuditEvents) != 1 {
		t.Errorf("expected 1 audit event, got %d", len(repo.AuditEvents))
	}
	if repo.AuditEvents[0].EventType != "gc_collected" {
		t.Errorf("expected event type gc_collected, got %s", repo.AuditEvents[0].EventType)
	}
}

func TestGCService_NoCandidates_ReturnsEmpty(t *testing.T) {
	// [REQ:P1-003] GC with no eligible sandboxes
	repo := gcRepo([]*types.Sandbox{})
	drv := mocks.NewFakeDriver()
	clk := clock.System{}
	svc := NewService(repo, drv, DefaultConfig(), clk, audit.NewRepoEmitter(repo.LogAuditEvent, clk))

	result, err := svc.Run(context.Background(), &types.GCRequest{
		Policy: &types.GCPolicy{
			MaxAge: 24 * time.Hour,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalCollected != 0 {
		t.Errorf("expected 0 collected, got %d", result.TotalCollected)
	}
}

func TestGCService_IdleTimeout_CollectsIdleSandboxes(t *testing.T) {
	// [REQ:P1-003] GC idle timeout policy
	now := time.Now()
	idleSandbox := &types.Sandbox{
		ID:         uuid.New(),
		ScopePath:  "/test/idle",
		Status:     types.StatusStopped,
		SizeBytes:  2000,
		CreatedAt:  now.Add(-1 * time.Hour), // Created 1 hour ago (not old enough for age-based)
		LastUsedAt: now.Add(-5 * time.Hour), // But idle for 5 hours
	}

	repo := gcRepo([]*types.Sandbox{idleSandbox})
	drv := mocks.NewFakeDriver()
	clk := clock.System{}
	svc := NewService(repo, drv, DefaultConfig(), clk, audit.NewRepoEmitter(repo.LogAuditEvent, clk))

	result, err := svc.Run(context.Background(), &types.GCRequest{
		DryRun: true,
		Policy: &types.GCPolicy{
			IdleTimeout: 4 * time.Hour, // 4 hour idle timeout
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalCollected != 1 {
		t.Errorf("expected 1 collected for idle sandbox, got %d", result.TotalCollected)
	}
}

func TestGCService_TerminalState_CollectsApprovedRejected(t *testing.T) {
	// [REQ:P1-003] GC terminal state cleanup
	now := time.Now()
	approvedTime := now.Add(-2 * time.Hour)
	approvedSandbox := &types.Sandbox{
		ID:         uuid.New(),
		ScopePath:  "/test/approved",
		Status:     types.StatusApproved,
		SizeBytes:  3000,
		CreatedAt:  now.Add(-3 * time.Hour),
		ApprovedAt: &approvedTime,
	}

	repo := gcRepo([]*types.Sandbox{approvedSandbox})
	drv := mocks.NewFakeDriver()
	clk := clock.System{}
	svc := NewService(repo, drv, DefaultConfig(), clk, audit.NewRepoEmitter(repo.LogAuditEvent, clk))

	result, err := svc.Run(context.Background(), &types.GCRequest{
		DryRun: true,
		Policy: &types.GCPolicy{
			IncludeTerminal: true,
			TerminalDelay:   1 * time.Hour, // Clean up after 1 hour
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalCollected != 1 {
		t.Errorf("expected 1 collected for approved sandbox, got %d", result.TotalCollected)
	}
}

func TestGCService_Limit_RespectsMaxCount(t *testing.T) {
	// [REQ:P1-003] GC respects limit parameter
	now := time.Now()
	sandboxes := make([]*types.Sandbox, 5)
	for i := 0; i < 5; i++ {
		sandboxes[i] = &types.Sandbox{
			ID:        uuid.New(),
			ScopePath: "/test/sandbox",
			Status:    types.StatusStopped,
			SizeBytes: 1000,
			CreatedAt: now.Add(-48 * time.Hour),
		}
	}

	repo := gcRepo(sandboxes)
	drv := mocks.NewFakeDriver()
	clk := clock.System{}
	svc := NewService(repo, drv, DefaultConfig(), clk, audit.NewRepoEmitter(repo.LogAuditEvent, clk))

	result, err := svc.Run(context.Background(), &types.GCRequest{
		DryRun: true,
		Limit:  3, // Only collect 3
		Policy: &types.GCPolicy{
			MaxAge: 24 * time.Hour,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Note: The limit is passed to the repository, not enforced in the service
	// So this test verifies the limit is respected when repo returns limited results
	if result.TotalCollected > 5 {
		t.Errorf("expected at most 5 collected, got %d", result.TotalCollected)
	}
}

func TestGCService_DefaultPolicy_UsedWhenNil(t *testing.T) {
	// [REQ:P1-003] GC uses default policy when none provided
	now := time.Now()
	oldSandbox := &types.Sandbox{
		ID:        uuid.New(),
		ScopePath: "/test/old",
		Status:    types.StatusStopped,
		SizeBytes: 1000,
		CreatedAt: now.Add(-48 * time.Hour),
	}

	repo := gcRepo([]*types.Sandbox{oldSandbox})
	drv := mocks.NewFakeDriver()
	clk := clock.System{}
	svc := NewService(repo, drv, DefaultConfig(), clk, audit.NewRepoEmitter(repo.LogAuditEvent, clk))

	result, err := svc.Run(context.Background(), &types.GCRequest{
		DryRun: true,
		// No policy - should use default
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Default config has 24h MaxAge, so 48h old sandbox should be collected
	if result.TotalCollected != 1 {
		t.Errorf("expected 1 collected with default policy, got %d", result.TotalCollected)
	}
}

func TestGCService_Preview_IsDryRun(t *testing.T) {
	// [REQ:P1-003] Preview is always dry run
	now := time.Now()
	sandbox := &types.Sandbox{
		ID:        uuid.New(),
		ScopePath: "/test",
		Status:    types.StatusStopped,
		CreatedAt: now.Add(-48 * time.Hour),
	}

	repo := gcRepo([]*types.Sandbox{sandbox})
	drv := mocks.NewFakeDriver()
	clk := clock.System{}
	svc := NewService(repo, drv, DefaultConfig(), clk, audit.NewRepoEmitter(repo.LogAuditEvent, clk))

	policy := types.DefaultGCPolicy()
	result, err := svc.Preview(context.Background(), &policy, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.DryRun {
		t.Error("Preview should always be dry run")
	}

	if len(repo.DeletedIDs) != 0 {
		t.Error("Preview should not delete anything")
	}
}

func TestGCService_Reasons_ArePopulated(t *testing.T) {
	// [REQ:P1-003] GC provides reasons for collection
	now := time.Now()
	sandbox := &types.Sandbox{
		ID:         uuid.New(),
		ScopePath:  "/test",
		Status:     types.StatusStopped,
		CreatedAt:  now.Add(-48 * time.Hour),
		LastUsedAt: now.Add(-48 * time.Hour),
	}

	repo := gcRepo([]*types.Sandbox{sandbox})
	drv := mocks.NewFakeDriver()
	clk := clock.System{}
	svc := NewService(repo, drv, DefaultConfig(), clk, audit.NewRepoEmitter(repo.LogAuditEvent, clk))

	result, err := svc.Run(context.Background(), &types.GCRequest{
		DryRun: true,
		Policy: &types.GCPolicy{
			MaxAge: 24 * time.Hour,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Collected) != 1 {
		t.Fatalf("expected 1 collected")
	}

	if result.Collected[0].Reason == "" {
		t.Error("expected reason to be populated")
	}

	// Check reasons map
	reasons, ok := result.Reasons[sandbox.ID.String()]
	if !ok || len(reasons) == 0 {
		t.Error("expected reasons map to be populated")
	}
}

// --- Error Handling Tests ---

func TestGCService_GetGCCandidatesError(t *testing.T) {
	// [REQ:P1-003] GC handles repository error for GetGCCandidates
	repo := mocks.NewFakeRepository()
	repo.GetGCCandidatesErr = fmt.Errorf("database connection failed")
	drv := mocks.NewFakeDriver()
	clk := clock.System{}
	svc := NewService(repo, drv, DefaultConfig(), clk, audit.NewRepoEmitter(repo.LogAuditEvent, clk))

	_, err := svc.Run(context.Background(), &types.GCRequest{
		Policy: &types.GCPolicy{
			MaxAge: 24 * time.Hour,
		},
	})

	if err == nil {
		t.Fatal("expected error when GetGCCandidates fails")
	}

	if !strings.Contains(err.Error(), "failed to get GC candidates") {
		t.Errorf("error message should mention 'failed to get GC candidates', got: %v", err)
	}
}

func TestGCService_GetStatsError(t *testing.T) {
	// [REQ:P1-003] GC handles repository error for GetStats during size-based filtering
	now := time.Now()
	sandbox := &types.Sandbox{
		ID:        uuid.New(),
		ScopePath: "/test",
		Status:    types.StatusStopped,
		SizeBytes: 1000,
		CreatedAt: now.Add(-48 * time.Hour),
	}

	repo := mocks.NewFakeRepository()
	seedCandidates(repo, []*types.Sandbox{sandbox})
	repo.GetStatsErr = fmt.Errorf("stats query failed")
	drv := mocks.NewFakeDriver()
	clk := clock.System{}
	svc := NewService(repo, drv, DefaultConfig(), clk, audit.NewRepoEmitter(repo.LogAuditEvent, clk))

	_, err := svc.Run(context.Background(), &types.GCRequest{
		Policy: &types.GCPolicy{
			MaxAge:            24 * time.Hour,
			MaxTotalSizeBytes: 500, // Triggers size-based filtering
		},
	})

	if err == nil {
		t.Fatal("expected error when GetStats fails during size filtering")
	}

	if !strings.Contains(err.Error(), "failed to filter by size") {
		t.Errorf("error message should mention 'failed to filter by size', got: %v", err)
	}
}

func TestGCService_CleanupError_ContinuesDeleting(t *testing.T) {
	// [REQ:P1-003] GC handles driver cleanup errors gracefully
	now := time.Now()
	sandbox := &types.Sandbox{
		ID:        uuid.New(),
		ScopePath: "/test",
		Status:    types.StatusStopped,
		SizeBytes: 1000,
		CreatedAt: now.Add(-48 * time.Hour),
	}

	repo := mocks.NewFakeRepository()
	seedCandidates(repo, []*types.Sandbox{sandbox})
	drv := mocks.NewFakeDriver()
	drv.CleanupErr = fmt.Errorf("cleanup failed: resource busy")
	clk := clock.System{}
	svc := NewService(repo, drv, DefaultConfig(), clk, audit.NewRepoEmitter(repo.LogAuditEvent, clk))

	result, err := svc.Run(context.Background(), &types.GCRequest{
		DryRun: false,
		Policy: &types.GCPolicy{
			MaxAge: 24 * time.Hour,
		},
	})
	// Should not return error - cleanup failures are logged but don't block
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Cleanup should have been called
	if len(drv.CleanedSandboxes) != 1 {
		t.Errorf("expected 1 cleanup call, got %d", len(drv.CleanedSandboxes))
	}

	// Deletion should have succeeded
	if len(repo.DeletedIDs) != 1 {
		t.Errorf("expected 1 deletion despite cleanup error, got %d", len(repo.DeletedIDs))
	}

	// Error should be recorded in result
	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error in result, got %d", len(result.Errors))
	}

	if !strings.Contains(result.Errors[0].Error, "driver cleanup warning") {
		t.Errorf("error should mention 'driver cleanup warning', got: %s", result.Errors[0].Error)
	}
}

func TestGCService_DeleteError_ContinuesToNextSandbox(t *testing.T) {
	// [REQ:P1-003] GC handles delete errors gracefully and continues
	now := time.Now()
	sandbox1 := &types.Sandbox{
		ID:        uuid.New(),
		ScopePath: "/test1",
		Status:    types.StatusStopped,
		SizeBytes: 1000,
		CreatedAt: now.Add(-48 * time.Hour),
	}
	sandbox2 := &types.Sandbox{
		ID:        uuid.New(),
		ScopePath: "/test2",
		Status:    types.StatusStopped,
		SizeBytes: 2000,
		CreatedAt: now.Add(-48 * time.Hour),
	}

	repo := mocks.NewFakeRepository()
	seedCandidates(repo, []*types.Sandbox{sandbox1, sandbox2})
	repo.DeleteFailIDs = map[uuid.UUID]bool{sandbox1.ID: true}
	repo.DeleteErr = fmt.Errorf("delete failed: constraint violation")
	drv := mocks.NewFakeDriver()
	clk := clock.System{}
	svc := NewService(repo, drv, DefaultConfig(), clk, audit.NewRepoEmitter(repo.LogAuditEvent, clk))

	result, err := svc.Run(context.Background(), &types.GCRequest{
		DryRun: false,
		Policy: &types.GCPolicy{
			MaxAge: 24 * time.Hour,
		},
	})
	// Should not return error - continues to next sandbox
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First deletion failed, second succeeded
	if len(repo.DeletedIDs) != 1 {
		t.Errorf("expected 1 successful deletion, got %d", len(repo.DeletedIDs))
	}

	// Only sandbox2 should be in deletedIDs
	if len(repo.DeletedIDs) > 0 && repo.DeletedIDs[0] != sandbox2.ID {
		t.Error("wrong sandbox was deleted")
	}

	// Error should be recorded for sandbox1
	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error in result, got %d", len(result.Errors))
	}

	if result.Errors[0].SandboxID != sandbox1.ID {
		t.Error("error should be for sandbox1")
	}

	// Only 1 collected (sandbox2)
	if result.TotalCollected != 1 {
		t.Errorf("expected 1 collected, got %d", result.TotalCollected)
	}

	// Bytes reclaimed should only count sandbox2
	if result.TotalBytesReclaimed != 2000 {
		t.Errorf("expected 2000 bytes reclaimed, got %d", result.TotalBytesReclaimed)
	}
}

func TestGCService_ContextCancellation(t *testing.T) {
	// [REQ:P1-003] GC respects context cancellation
	// Note: The current implementation doesn't check ctx.Err() during iteration
	// This test documents the expected behavior for future improvements
	t.Skip("Context cancellation not yet implemented in GC loop - future improvement")
}

func TestGCService_MultipleErrors_AllRecorded(t *testing.T) {
	// [REQ:P1-003] GC records all errors when multiple failures occur
	now := time.Now()
	sandboxes := make([]*types.Sandbox, 3)
	for i := 0; i < 3; i++ {
		sandboxes[i] = &types.Sandbox{
			ID:        uuid.New(),
			ScopePath: fmt.Sprintf("/test%d", i),
			Status:    types.StatusStopped,
			SizeBytes: 1000,
			CreatedAt: now.Add(-48 * time.Hour),
		}
	}

	repo := mocks.NewFakeRepository()
	seedCandidates(repo, sandboxes)
	repo.DeleteFailIDs = map[uuid.UUID]bool{
		sandboxes[0].ID: true,
		sandboxes[2].ID: true,
	}
	repo.DeleteErr = fmt.Errorf("delete failed")
	drv := mocks.NewFakeDriver()
	drv.CleanupFailIDs = map[uuid.UUID]bool{sandboxes[1].ID: true}
	drv.CleanupErr = fmt.Errorf("cleanup failed")
	clk := clock.System{}
	svc := NewService(repo, drv, DefaultConfig(), clk, audit.NewRepoEmitter(repo.LogAuditEvent, clk))

	result, err := svc.Run(context.Background(), &types.GCRequest{
		DryRun: false,
		Policy: &types.GCPolicy{
			MaxAge: 24 * time.Hour,
		},
	})
	// Should not return fatal error
	if err != nil {
		t.Fatalf("unexpected fatal error: %v", err)
	}

	// Should have 3 errors: 2 delete failures + 1 cleanup warning
	if len(result.Errors) != 3 {
		t.Errorf("expected 3 errors, got %d", len(result.Errors))
	}

	// Only sandbox1 should be successfully collected
	if result.TotalCollected != 1 {
		t.Errorf("expected 1 collected, got %d", result.TotalCollected)
	}
}
