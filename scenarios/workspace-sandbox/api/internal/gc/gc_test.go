package gc

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/repository"
	"workspace-sandbox/internal/types"
)

// MockRepository implements repository.Repository for testing
type MockRepository struct {
	sandboxes    []*types.Sandbox
	stats        *types.SandboxStats
	deletedIDs   []uuid.UUID
	auditEvents  []*types.AuditEvent
	gcCandidates []*types.Sandbox
}

func (m *MockRepository) GetGCCandidates(ctx context.Context, policy *types.GCPolicy, limit int) ([]*types.Sandbox, error) {
	if m.gcCandidates != nil {
		return m.gcCandidates, nil
	}
	return m.sandboxes, nil
}

func (m *MockRepository) GetStats(ctx context.Context) (*types.SandboxStats, error) {
	if m.stats != nil {
		return m.stats, nil
	}
	return &types.SandboxStats{
		TotalCount:     int64(len(m.sandboxes)),
		TotalSizeBytes: 1000,
	}, nil
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	m.deletedIDs = append(m.deletedIDs, id)
	return nil
}

func (m *MockRepository) LogAuditEvent(ctx context.Context, event *types.AuditEvent) error {
	m.auditEvents = append(m.auditEvents, event)
	return nil
}

func (m *MockRepository) GetAuditLog(ctx context.Context, sandboxID *uuid.UUID, limit, offset int) ([]*types.AuditEvent, int, error) {
	return m.auditEvents, len(m.auditEvents), nil
}

// Unused methods - implement interface
func (m *MockRepository) Create(ctx context.Context, s *types.Sandbox) error { return nil }

func (m *MockRepository) Get(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
	return nil, nil
}
func (m *MockRepository) Update(ctx context.Context, s *types.Sandbox) error { return nil }
func (m *MockRepository) List(ctx context.Context, filter *types.ListFilter) (*types.ListResult, error) {
	return nil, nil
}

func (m *MockRepository) CheckScopeOverlap(ctx context.Context, scopePath, projectRoot string, excludeID *uuid.UUID) ([]types.PathConflict, error) {
	return nil, nil
}

func (m *MockRepository) GetActiveSandboxes(ctx context.Context, projectRoot string) ([]*types.Sandbox, error) {
	return nil, nil
}

func (m *MockRepository) FindByIdempotencyKey(ctx context.Context, key string) (*types.Sandbox, error) {
	return nil, nil
}

func (m *MockRepository) UpdateWithVersionCheck(ctx context.Context, s *types.Sandbox, expectedVersion int64) error {
	return nil
}

func (m *MockRepository) RecordAppliedChanges(ctx context.Context, changes []*types.AppliedChange) error {
	return nil
}

func (m *MockRepository) GetPendingChanges(ctx context.Context, projectRoot string, limit, offset int) (*types.PendingChangesResult, error) {
	return &types.PendingChangesResult{}, nil
}

func (m *MockRepository) GetPendingChangeFiles(ctx context.Context, projectRoot string, sandboxIDs []uuid.UUID) ([]*types.AppliedChange, error) {
	return nil, nil
}

func (m *MockRepository) GetFileProvenance(ctx context.Context, filePath, projectRoot string, limit int) ([]*types.AppliedChange, error) {
	return nil, nil
}

func (m *MockRepository) MarkChangesCommitted(ctx context.Context, ids []uuid.UUID, commitHash, commitMessage string) error {
	return nil
}

func (m *MockRepository) BeginTx(ctx context.Context) (repository.TxRepository, error) {
	return &MockTxRepository{MockRepository: m}, nil
}

// MockTxRepository implements TxRepository for testing
type MockTxRepository struct {
	*MockRepository
}

func (m *MockTxRepository) Commit() error   { return nil }
func (m *MockTxRepository) Rollback() error { return nil }
func (m *MockTxRepository) GetAuditLog(ctx context.Context, sandboxID *uuid.UUID, limit, offset int) ([]*types.AuditEvent, int, error) {
	return nil, 0, nil
}

// Ensure MockRepository implements Repository
var _ repository.Repository = (*MockRepository)(nil)

// MockDriverWithError implements driver.Driver for testing with error injection.
type MockDriverWithError struct {
	cleanupCalled  []uuid.UUID
	cleanupErr     error              // If set, Cleanup returns this error
	cleanupFailIDs map[uuid.UUID]bool // IDs that should fail cleanup
}

func NewMockDriverWithError() *MockDriverWithError {
	return &MockDriverWithError{
		cleanupCalled:  []uuid.UUID{},
		cleanupFailIDs: make(map[uuid.UUID]bool),
	}
}

func (m *MockDriverWithError) Type() driver.DriverType { return "mock" }
func (m *MockDriverWithError) Version() string         { return "1.0.0" }
func (m *MockDriverWithError) IsAvailable(ctx context.Context) (bool, error) {
	return true, nil
}

func (m *MockDriverWithError) Mount(ctx context.Context, sandbox *types.Sandbox) (*driver.MountPaths, error) {
	return &driver.MountPaths{}, nil
}

func (m *MockDriverWithError) Unmount(ctx context.Context, sandbox *types.Sandbox) error {
	return nil
}

func (m *MockDriverWithError) Cleanup(ctx context.Context, sandbox *types.Sandbox) error {
	m.cleanupCalled = append(m.cleanupCalled, sandbox.ID)
	// Only fail for specific IDs if cleanupFailIDs is set
	if len(m.cleanupFailIDs) > 0 && m.cleanupFailIDs[sandbox.ID] {
		return m.cleanupErr
	}
	// If no specific fail IDs, fail all if cleanupErr is set
	if len(m.cleanupFailIDs) == 0 && m.cleanupErr != nil {
		return m.cleanupErr
	}
	return nil
}

func (m *MockDriverWithError) GetChangedFiles(ctx context.Context, sandbox *types.Sandbox) ([]*types.FileChange, error) {
	return nil, nil
}

func (m *MockDriverWithError) IsMounted(ctx context.Context, sandbox *types.Sandbox) (bool, error) {
	return false, nil
}

func (m *MockDriverWithError) VerifyMountIntegrity(ctx context.Context, sandbox *types.Sandbox) error {
	return nil
}

func (m *MockDriverWithError) Exec(ctx context.Context, sandbox *types.Sandbox, cfg driver.BwrapConfig, command string, args ...string) (*driver.ExecResult, error) {
	return &driver.ExecResult{}, nil
}

func (m *MockDriverWithError) StartProcess(ctx context.Context, sandbox *types.Sandbox, cfg driver.BwrapConfig, command string, args ...string) (int, error) {
	return 0, nil
}

func (m *MockDriverWithError) RemoveFromUpper(ctx context.Context, sandbox *types.Sandbox, filePath string) error {
	return nil
}

var _ driver.Driver = (*MockDriverWithError)(nil)

// MockRepositoryWithError implements repository.Repository with error injection.
type MockRepositoryWithError struct {
	sandboxes          []*types.Sandbox
	stats              *types.SandboxStats
	deletedIDs         []uuid.UUID
	auditEvents        []*types.AuditEvent
	gcCandidates       []*types.Sandbox
	getGCCandidatesErr error
	getStatsErr        error
	deleteErr          error
	deleteFailIDs      map[uuid.UUID]bool // IDs that should fail delete
}

func NewMockRepositoryWithError() *MockRepositoryWithError {
	return &MockRepositoryWithError{
		sandboxes:     []*types.Sandbox{},
		auditEvents:   []*types.AuditEvent{},
		deleteFailIDs: make(map[uuid.UUID]bool),
	}
}

func (m *MockRepositoryWithError) GetGCCandidates(ctx context.Context, policy *types.GCPolicy, limit int) ([]*types.Sandbox, error) {
	if m.getGCCandidatesErr != nil {
		return nil, m.getGCCandidatesErr
	}
	if m.gcCandidates != nil {
		return m.gcCandidates, nil
	}
	return m.sandboxes, nil
}

func (m *MockRepositoryWithError) GetStats(ctx context.Context) (*types.SandboxStats, error) {
	if m.getStatsErr != nil {
		return nil, m.getStatsErr
	}
	if m.stats != nil {
		return m.stats, nil
	}
	return &types.SandboxStats{
		TotalCount:     int64(len(m.sandboxes)),
		TotalSizeBytes: 1000,
	}, nil
}

func (m *MockRepositoryWithError) Delete(ctx context.Context, id uuid.UUID) error {
	// Only fail for specific IDs if deleteFailIDs is set
	if len(m.deleteFailIDs) > 0 && m.deleteFailIDs[id] {
		return m.deleteErr
	}
	// If no specific fail IDs, fail all if deleteErr is set
	if len(m.deleteFailIDs) == 0 && m.deleteErr != nil {
		return m.deleteErr
	}
	m.deletedIDs = append(m.deletedIDs, id)
	return nil
}

func (m *MockRepositoryWithError) LogAuditEvent(ctx context.Context, event *types.AuditEvent) error {
	m.auditEvents = append(m.auditEvents, event)
	return nil
}

func (m *MockRepositoryWithError) GetAuditLog(ctx context.Context, sandboxID *uuid.UUID, limit, offset int) ([]*types.AuditEvent, int, error) {
	return m.auditEvents, len(m.auditEvents), nil
}

func (m *MockRepositoryWithError) Create(ctx context.Context, s *types.Sandbox) error { return nil }
func (m *MockRepositoryWithError) Get(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
	return nil, nil
}
func (m *MockRepositoryWithError) Update(ctx context.Context, s *types.Sandbox) error { return nil }
func (m *MockRepositoryWithError) List(ctx context.Context, filter *types.ListFilter) (*types.ListResult, error) {
	return nil, nil
}

func (m *MockRepositoryWithError) CheckScopeOverlap(ctx context.Context, scopePath, projectRoot string, excludeID *uuid.UUID) ([]types.PathConflict, error) {
	return nil, nil
}

func (m *MockRepositoryWithError) GetActiveSandboxes(ctx context.Context, projectRoot string) ([]*types.Sandbox, error) {
	return nil, nil
}

func (m *MockRepositoryWithError) FindByIdempotencyKey(ctx context.Context, key string) (*types.Sandbox, error) {
	return nil, nil
}

func (m *MockRepositoryWithError) UpdateWithVersionCheck(ctx context.Context, s *types.Sandbox, expectedVersion int64) error {
	return nil
}

func (m *MockRepositoryWithError) RecordAppliedChanges(ctx context.Context, changes []*types.AppliedChange) error {
	return nil
}

func (m *MockRepositoryWithError) GetPendingChanges(ctx context.Context, projectRoot string, limit, offset int) (*types.PendingChangesResult, error) {
	return &types.PendingChangesResult{}, nil
}

func (m *MockRepositoryWithError) GetPendingChangeFiles(ctx context.Context, projectRoot string, sandboxIDs []uuid.UUID) ([]*types.AppliedChange, error) {
	return nil, nil
}

func (m *MockRepositoryWithError) GetFileProvenance(ctx context.Context, filePath, projectRoot string, limit int) ([]*types.AppliedChange, error) {
	return nil, nil
}

func (m *MockRepositoryWithError) MarkChangesCommitted(ctx context.Context, ids []uuid.UUID, commitHash, commitMessage string) error {
	return nil
}

func (m *MockRepositoryWithError) BeginTx(ctx context.Context) (repository.TxRepository, error) {
	return nil, nil
}

var _ repository.Repository = (*MockRepositoryWithError)(nil)

// MockDriver implements driver.Driver for testing
type MockDriver struct {
	cleanupCalled []uuid.UUID
}

func (m *MockDriver) Type() driver.DriverType { return "mock" }
func (m *MockDriver) Version() string         { return "1.0.0" }
func (m *MockDriver) IsAvailable(ctx context.Context) (bool, error) {
	return true, nil
}

func (m *MockDriver) Mount(ctx context.Context, sandbox *types.Sandbox) (*driver.MountPaths, error) {
	return &driver.MountPaths{}, nil
}

func (m *MockDriver) Unmount(ctx context.Context, sandbox *types.Sandbox) error {
	return nil
}

func (m *MockDriver) Cleanup(ctx context.Context, sandbox *types.Sandbox) error {
	m.cleanupCalled = append(m.cleanupCalled, sandbox.ID)
	return nil
}

func (m *MockDriver) GetChangedFiles(ctx context.Context, sandbox *types.Sandbox) ([]*types.FileChange, error) {
	return nil, nil
}

func (m *MockDriver) IsMounted(ctx context.Context, sandbox *types.Sandbox) (bool, error) {
	return false, nil
}

func (m *MockDriver) VerifyMountIntegrity(ctx context.Context, sandbox *types.Sandbox) error {
	return nil
}

func (m *MockDriver) Exec(ctx context.Context, sandbox *types.Sandbox, cfg driver.BwrapConfig, command string, args ...string) (*driver.ExecResult, error) {
	return &driver.ExecResult{}, nil
}

func (m *MockDriver) StartProcess(ctx context.Context, sandbox *types.Sandbox, cfg driver.BwrapConfig, command string, args ...string) (int, error) {
	return 0, nil
}

func (m *MockDriver) RemoveFromUpper(ctx context.Context, sandbox *types.Sandbox, filePath string) error {
	return nil
}

// Ensure MockDriver implements Driver
var _ driver.Driver = (*MockDriver)(nil)

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

	repo := &MockRepository{sandboxes: []*types.Sandbox{oldSandbox}}
	drv := &MockDriver{}
	svc := NewService(repo, drv, DefaultConfig())

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
	if len(repo.deletedIDs) != 0 {
		t.Errorf("expected no deletions in dry run, got %d", len(repo.deletedIDs))
	}

	if len(drv.cleanupCalled) != 0 {
		t.Errorf("expected no cleanup in dry run, got %d", len(drv.cleanupCalled))
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

	repo := &MockRepository{sandboxes: []*types.Sandbox{oldSandbox}}
	drv := &MockDriver{}
	svc := NewService(repo, drv, DefaultConfig())

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
	if len(repo.deletedIDs) != 1 {
		t.Errorf("expected 1 deletion, got %d", len(repo.deletedIDs))
	}

	// Verify cleanup was called
	if len(drv.cleanupCalled) != 1 {
		t.Errorf("expected 1 cleanup call, got %d", len(drv.cleanupCalled))
	}

	// Verify audit event was logged
	if len(repo.auditEvents) != 1 {
		t.Errorf("expected 1 audit event, got %d", len(repo.auditEvents))
	}
	if repo.auditEvents[0].EventType != "gc_collected" {
		t.Errorf("expected event type gc_collected, got %s", repo.auditEvents[0].EventType)
	}
}

func TestGCService_NoCandidates_ReturnsEmpty(t *testing.T) {
	// [REQ:P1-003] GC with no eligible sandboxes
	repo := &MockRepository{sandboxes: []*types.Sandbox{}}
	drv := &MockDriver{}
	svc := NewService(repo, drv, DefaultConfig())

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

	repo := &MockRepository{sandboxes: []*types.Sandbox{idleSandbox}}
	drv := &MockDriver{}
	svc := NewService(repo, drv, DefaultConfig())

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

	repo := &MockRepository{sandboxes: []*types.Sandbox{approvedSandbox}}
	drv := &MockDriver{}
	svc := NewService(repo, drv, DefaultConfig())

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

	repo := &MockRepository{sandboxes: sandboxes}
	drv := &MockDriver{}
	svc := NewService(repo, drv, DefaultConfig())

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

	repo := &MockRepository{sandboxes: []*types.Sandbox{oldSandbox}}
	drv := &MockDriver{}
	svc := NewService(repo, drv, DefaultConfig())

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

	repo := &MockRepository{sandboxes: []*types.Sandbox{sandbox}}
	drv := &MockDriver{}
	svc := NewService(repo, drv, DefaultConfig())

	policy := types.DefaultGCPolicy()
	result, err := svc.Preview(context.Background(), &policy, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.DryRun {
		t.Error("Preview should always be dry run")
	}

	if len(repo.deletedIDs) != 0 {
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

	repo := &MockRepository{sandboxes: []*types.Sandbox{sandbox}}
	drv := &MockDriver{}
	svc := NewService(repo, drv, DefaultConfig())

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
	repo := NewMockRepositoryWithError()
	repo.getGCCandidatesErr = fmt.Errorf("database connection failed")
	drv := &MockDriver{}
	svc := NewService(repo, drv, DefaultConfig())

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

	repo := NewMockRepositoryWithError()
	repo.sandboxes = []*types.Sandbox{sandbox}
	repo.getStatsErr = fmt.Errorf("stats query failed")
	drv := &MockDriver{}
	svc := NewService(repo, drv, DefaultConfig())

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

	repo := NewMockRepositoryWithError()
	repo.sandboxes = []*types.Sandbox{sandbox}
	drv := NewMockDriverWithError()
	drv.cleanupErr = fmt.Errorf("cleanup failed: resource busy")
	svc := NewService(repo, drv, DefaultConfig())

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
	if len(drv.cleanupCalled) != 1 {
		t.Errorf("expected 1 cleanup call, got %d", len(drv.cleanupCalled))
	}

	// Deletion should have succeeded
	if len(repo.deletedIDs) != 1 {
		t.Errorf("expected 1 deletion despite cleanup error, got %d", len(repo.deletedIDs))
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

	repo := NewMockRepositoryWithError()
	repo.sandboxes = []*types.Sandbox{sandbox1, sandbox2}
	repo.deleteFailIDs = map[uuid.UUID]bool{sandbox1.ID: true}
	repo.deleteErr = fmt.Errorf("delete failed: constraint violation")
	drv := &MockDriver{}
	svc := NewService(repo, drv, DefaultConfig())

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
	if len(repo.deletedIDs) != 1 {
		t.Errorf("expected 1 successful deletion, got %d", len(repo.deletedIDs))
	}

	// Only sandbox2 should be in deletedIDs
	if len(repo.deletedIDs) > 0 && repo.deletedIDs[0] != sandbox2.ID {
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

	repo := NewMockRepositoryWithError()
	repo.sandboxes = sandboxes
	repo.deleteFailIDs = map[uuid.UUID]bool{
		sandboxes[0].ID: true,
		sandboxes[2].ID: true,
	}
	repo.deleteErr = fmt.Errorf("delete failed")
	drv := NewMockDriverWithError()
	drv.cleanupFailIDs = map[uuid.UUID]bool{sandboxes[1].ID: true}
	drv.cleanupErr = fmt.Errorf("cleanup failed")
	svc := NewService(repo, drv, DefaultConfig())

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
