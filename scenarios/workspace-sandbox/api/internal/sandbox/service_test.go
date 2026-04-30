// Package sandbox provides unit tests for the sandbox service layer.
//
// These tests verify the business logic of the Service struct, including:
// - CRUD operations (Create, Get, List, Delete)
// - State transitions (Stop, Start)
// - Approval workflow (Approve, Reject, Discard)
// - Idempotency guarantees
// - Error handling paths
// - Policy integration
//
// Tests use the canonical fakes from internal/testutil/mocks
// (FakeRepository, FakeDriver, FakeGitOps) to isolate the service
// layer from infrastructure concerns.
package sandbox

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/audit"
	"workspace-sandbox/internal/clock"
	"workspace-sandbox/internal/diff"
	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/testutil/mocks"
	"workspace-sandbox/internal/types"
)

// --- Test Helpers ---

func newTestService(repo *mocks.FakeRepository, drv *mocks.FakeDriver) *Service {
	clk := clock.System{}
	return NewService(repo, drv, ServiceConfig{
		DefaultProjectRoot: "/tmp/project",
		MaxSandboxes:       100,
		DefaultTTL:         24 * time.Hour,
	}, clk, audit.NewRepoEmitter(repo.LogAuditEvent, clk), process.NewOSExecStarter(), WithGitOps(mocks.NewFakeGitOps()))
}

func createTestSandbox(id uuid.UUID, status types.Status) *types.Sandbox {
	now := time.Now()
	return &types.Sandbox{
		ID:            id,
		ScopePath:     "/tmp/project/src",
		ProjectRoot:   "/tmp/project",
		Owner:         "test-user",
		OwnerType:     types.OwnerTypeUser,
		Status:        status,
		DriverID:      "mock",
		DriverVersion: "1.0.0",
		LowerDir:      "/tmp/lower",
		UpperDir:      "/tmp/upper",
		WorkDir:       "/tmp/work",
		MergedDir:     "/tmp/merged",
		CreatedAt:     now,
		LastUsedAt:    now,
		UpdatedAt:     now,
		Version:       1,
	}
}

// --- Create Tests ---

// [REQ:P0-001] Test sandbox creation happy path.
func TestService_Create_Success(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	req := &types.CreateRequest{
		ScopePath:   "/tmp/project/src",
		ProjectRoot: "/tmp/project",
		Owner:       "test-agent",
		OwnerType:   types.OwnerTypeAgent,
		Tags:        []string{"test"},
	}

	sandbox, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if sandbox == nil {
		t.Fatal("Create() returned nil sandbox")
	}

	if sandbox.Status != types.StatusActive {
		t.Errorf("Create() status = %v, want Active", sandbox.Status)
	}

	if sandbox.Owner != "test-agent" {
		t.Errorf("Create() owner = %v, want test-agent", sandbox.Owner)
	}

	// Verify sandbox was stored
	if _, ok := repo.Sandboxes[sandbox.ID]; !ok {
		t.Error("Sandbox was not stored in repository")
	}

	// Verify audit event was logged
	if len(repo.AuditEvents) != 1 {
		t.Errorf("Expected 1 audit event, got %d", len(repo.AuditEvents))
	}
}

// [REQ:P0-001] Test sandbox creation with idempotency key.
func TestService_Create_IdempotencyKey(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	req := &types.CreateRequest{
		ScopePath:      "/tmp/project/src",
		ProjectRoot:    "/tmp/project",
		Owner:          "test-agent",
		IdempotencyKey: "unique-key-123",
	}

	// First create
	sandbox1, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatalf("First Create() error = %v", err)
	}

	// Second create with same key should return same sandbox
	sandbox2, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatalf("Second Create() error = %v", err)
	}

	if sandbox1.ID != sandbox2.ID {
		t.Errorf("Idempotent create returned different IDs: %v vs %v", sandbox1.ID, sandbox2.ID)
	}

	// Should only have one sandbox in repository
	if len(repo.Sandboxes) != 1 {
		t.Errorf("Expected 1 sandbox, got %d", len(repo.Sandboxes))
	}
}

// [REQ:P0-001] Test sandbox creation fails with scope conflict.
func TestService_Create_ScopeConflict(t *testing.T) {
	repo := mocks.NewFakeRepository()
	repo.ScopeConflicts = []types.PathConflict{
		{ExistingID: uuid.New().String(), ExistingScope: "/tmp/project/src", NewScope: "/tmp/project/src", ConflictType: types.ConflictTypeExact},
	}
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	req := &types.CreateRequest{
		ScopePath:   "/tmp/project/src",
		ProjectRoot: "/tmp/project",
		Owner:       "test-agent",
	}

	_, err := svc.Create(ctx, req)
	if err == nil {
		t.Fatal("Create() expected error for scope conflict")
	}

	var scopeErr *types.ScopeConflictError
	if !errors.As(err, &scopeErr) {
		t.Errorf("Create() error type = %T, want *ScopeConflictError", err)
	}
}

// [REQ:P0-001] Test sandbox creation fails without project root.
func TestService_Create_NoProjectRoot(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	clk := clock.System{}
	svc := NewService(repo, drv, ServiceConfig{}, clk, audit.NewRepoEmitter(repo.LogAuditEvent, clk), process.NewOSExecStarter()) // No default project root
	ctx := context.Background()

	req := &types.CreateRequest{
		ScopePath: "/tmp/project/src",
		Owner:     "test-agent",
	}

	_, err := svc.Create(ctx, req)
	if err == nil {
		t.Fatal("Create() expected error without project root")
	}

	var valErr *types.ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("Create() error type = %T, want *ValidationError", err)
	}
}

// [REQ:P0-001] Test sandbox creation handles mount failure.
func TestService_Create_MountFailure(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	drv.MountErr = errors.New("mount failed: permission denied")
	svc := newTestService(repo, drv)
	ctx := context.Background()

	req := &types.CreateRequest{
		ScopePath:   "/tmp/project/src",
		ProjectRoot: "/tmp/project",
		Owner:       "test-agent",
	}

	sandbox, err := svc.Create(ctx, req)

	// Should still return sandbox (in error state)
	if sandbox == nil {
		t.Fatal("Create() should return sandbox even on mount failure")
	}

	if sandbox.Status != types.StatusError {
		t.Errorf("Create() status = %v, want Error on mount failure", sandbox.Status)
	}

	if err == nil {
		t.Error("Create() should return error on mount failure")
	}
}

// --- Get Tests ---

func TestService_Get_Found(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	// Pre-create a sandbox
	id := uuid.New()
	existing := createTestSandbox(id, types.StatusActive)
	repo.Sandboxes[id] = existing

	sandbox, err := svc.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if sandbox.ID != id {
		t.Errorf("Get() ID = %v, want %v", sandbox.ID, id)
	}
}

func TestService_Get_NotFound(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	_, err := svc.Get(ctx, uuid.New())
	if err == nil {
		t.Fatal("Get() expected error for non-existent sandbox")
	}

	var nfErr *types.NotFoundError
	if !errors.As(err, &nfErr) {
		t.Errorf("Get() error type = %T, want *NotFoundError", err)
	}
}

// --- Stop Tests ---

// [REQ:P0-001] Test stop transitions sandbox to Stopped status.
func TestService_Stop_Success(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	id := uuid.New()
	existing := createTestSandbox(id, types.StatusActive)
	repo.Sandboxes[id] = existing

	sandbox, err := svc.Stop(ctx, id)
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if sandbox.Status != types.StatusStopped {
		t.Errorf("Stop() status = %v, want Stopped", sandbox.Status)
	}

	if sandbox.StoppedAt == nil {
		t.Error("Stop() should set StoppedAt timestamp")
	}
}

// [REQ:P0-001] Test stop is idempotent.
func TestService_Stop_Idempotent(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	id := uuid.New()
	now := time.Now()
	existing := createTestSandbox(id, types.StatusStopped)
	existing.StoppedAt = &now
	repo.Sandboxes[id] = existing

	// Stop already-stopped sandbox should succeed
	sandbox, err := svc.Stop(ctx, id)
	if err != nil {
		t.Fatalf("Stop() on stopped sandbox error = %v", err)
	}

	if sandbox.Status != types.StatusStopped {
		t.Errorf("Stop() status = %v, want Stopped", sandbox.Status)
	}
}

// [REQ:P0-001] Test stop fails for terminal states.
func TestService_Stop_TerminalState(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	terminalStates := []types.Status{
		types.StatusApproved,
		types.StatusRejected,
		types.StatusDeleted,
	}

	for _, status := range terminalStates {
		t.Run(string(status), func(t *testing.T) {
			id := uuid.New()
			existing := createTestSandbox(id, status)
			repo.Sandboxes[id] = existing

			_, err := svc.Stop(ctx, id)
			if err == nil {
				t.Errorf("Stop() on %s sandbox expected error", status)
			}
		})
	}
}

// --- Start Tests ---

// [REQ:P0-001] Test start transitions sandbox to Active status.
func TestService_Start_Success(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	id := uuid.New()
	now := time.Now()
	existing := createTestSandbox(id, types.StatusStopped)
	existing.StoppedAt = &now
	repo.Sandboxes[id] = existing

	sandbox, err := svc.Start(ctx, id)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if sandbox.Status != types.StatusActive {
		t.Errorf("Start() status = %v, want Active", sandbox.Status)
	}

	if sandbox.StoppedAt != nil {
		t.Error("Start() should clear StoppedAt timestamp")
	}
}

// [REQ:P0-001] Test start is idempotent.
func TestService_Start_Idempotent(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	id := uuid.New()
	existing := createTestSandbox(id, types.StatusActive)
	repo.Sandboxes[id] = existing

	// Start already-active sandbox should succeed
	sandbox, err := svc.Start(ctx, id)
	if err != nil {
		t.Fatalf("Start() on active sandbox error = %v", err)
	}

	if sandbox.Status != types.StatusActive {
		t.Errorf("Start() status = %v, want Active", sandbox.Status)
	}
}

// --- Delete Tests ---

// [REQ:P0-001] Test delete removes sandbox.
func TestService_Delete_Success(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	id := uuid.New()
	existing := createTestSandbox(id, types.StatusActive)
	repo.Sandboxes[id] = existing

	err := svc.Delete(ctx, id)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Soft-delete: the sandbox row remains so the diff archive (Phase 2)
	// keeps a foreign-key target and the audit log keeps the row to
	// reference. Status flips to Deleted; deleted_at is stamped.
	got, ok := repo.Sandboxes[id]
	if !ok {
		t.Fatal("Delete() should retain sandbox row (soft-delete)")
	}
	if got.Status != types.StatusDeleted {
		t.Errorf("Delete() status = %s, want %s", got.Status, types.StatusDeleted)
	}
	if got.DeletedAt == nil {
		t.Error("Delete() should stamp DeletedAt")
	}
}

// [REQ:P0-001] Test delete is idempotent for non-existent sandbox.
func TestService_Delete_NotFound(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	// Delete non-existent sandbox should succeed (idempotent)
	err := svc.Delete(ctx, uuid.New())
	if err != nil {
		t.Errorf("Delete() on non-existent sandbox error = %v, want nil", err)
	}
}

// [REQ:P0-001] Test delete is idempotent for already-deleted sandbox.
func TestService_Delete_AlreadyDeleted(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	id := uuid.New()
	existing := createTestSandbox(id, types.StatusDeleted)
	repo.Sandboxes[id] = existing

	err := svc.Delete(ctx, id)
	if err != nil {
		t.Errorf("Delete() on deleted sandbox error = %v, want nil", err)
	}
}

// [REQ:P0-001] Test delete handles driver cleanup failure gracefully.
func TestService_Delete_CleanupFailure(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	drv.CleanupErr = errors.New("cleanup failed")
	svc := newTestService(repo, drv)
	ctx := context.Background()

	id := uuid.New()
	existing := createTestSandbox(id, types.StatusActive)
	repo.Sandboxes[id] = existing

	// Delete should still succeed despite cleanup failure
	err := svc.Delete(ctx, id)
	if err != nil {
		t.Errorf("Delete() should succeed despite cleanup failure, got error = %v", err)
	}
}

// --- GetDiff Tests ---

// [REQ:P0-006] Test diff generation for active sandbox.
// Note: Full diff generation requires filesystem access, which is tested in integration tests.
// This unit test verifies the service layer correctly handles the driver's response.
func TestService_GetDiff_Success(t *testing.T) {
	t.Skip("GetDiff requires real filesystem access for diff generation - tested in integration tests")
}

// [REQ:P0-006] Test diff generation with no changes.
func TestService_GetDiff_NoChanges(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	drv.ChangedFiles = []*types.FileChange{} // No changes
	svc := newTestService(repo, drv)
	ctx := context.Background()

	id := uuid.New()
	existing := createTestSandbox(id, types.StatusActive)
	existing.UpperDir = t.TempDir()
	existing.LowerDir = t.TempDir()
	repo.Sandboxes[id] = existing

	result, err := svc.GetDiff(ctx, id)
	if err != nil {
		t.Fatalf("GetDiff() error = %v", err)
	}

	if len(result.Files) != 0 {
		t.Errorf("GetDiff() files = %d, want 0 for no changes", len(result.Files))
	}

	if result.UnifiedDiff != "" {
		t.Error("GetDiff() should return empty diff for no changes")
	}
}

// [REQ:P0-006] Test diff generation fails for terminal states.
func TestService_GetDiff_TerminalState(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	id := uuid.New()
	existing := createTestSandbox(id, types.StatusDeleted)
	repo.Sandboxes[id] = existing

	_, err := svc.GetDiff(ctx, id)
	if err == nil {
		t.Error("GetDiff() on deleted sandbox expected error")
	}
}

// --- Approve Tests ---

// [REQ:P0-007] Test approve is idempotent for already-approved sandbox.
func TestService_Approve_Idempotent(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	id := uuid.New()
	now := time.Now()
	existing := createTestSandbox(id, types.StatusApproved)
	existing.ApprovedAt = &now
	repo.Sandboxes[id] = existing

	req := &types.ApprovalRequest{
		SandboxID: id,
		Actor:     "reviewer",
	}

	result, err := svc.Approve(ctx, req)
	if err != nil {
		t.Fatalf("Approve() on approved sandbox error = %v", err)
	}

	if !result.Success {
		t.Error("Approve() on approved sandbox should succeed")
	}

	if result.Applied != 0 {
		t.Errorf("Approve() on approved sandbox Applied = %d, want 0", result.Applied)
	}
}

// [REQ:P0-007] Test approve fails for terminal states.
func TestService_Approve_TerminalState(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	id := uuid.New()
	existing := createTestSandbox(id, types.StatusRejected)
	repo.Sandboxes[id] = existing

	req := &types.ApprovalRequest{
		SandboxID: id,
		Actor:     "reviewer",
	}

	_, err := svc.Approve(ctx, req)
	if err == nil {
		t.Error("Approve() on rejected sandbox expected error")
	}
}

// --- Reject Tests ---

// [REQ:P0-007] Test reject transitions sandbox to Rejected status.
func TestService_Reject_Success(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	id := uuid.New()
	existing := createTestSandbox(id, types.StatusStopped)
	repo.Sandboxes[id] = existing

	sandbox, err := svc.Reject(ctx, id, "reviewer")
	if err != nil {
		t.Fatalf("Reject() error = %v", err)
	}

	if sandbox.Status != types.StatusRejected {
		t.Errorf("Reject() status = %v, want Rejected", sandbox.Status)
	}
}

// [REQ:P0-007] Test reject is idempotent.
func TestService_Reject_Idempotent(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	id := uuid.New()
	existing := createTestSandbox(id, types.StatusRejected)
	repo.Sandboxes[id] = existing

	sandbox, err := svc.Reject(ctx, id, "reviewer")
	if err != nil {
		t.Fatalf("Reject() on rejected sandbox error = %v", err)
	}

	if sandbox.Status != types.StatusRejected {
		t.Errorf("Reject() status = %v, want Rejected", sandbox.Status)
	}
}

// --- Discard Tests ---

func TestService_Discard_Success(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()

	fileID1 := uuid.New()
	fileID2 := uuid.New()
	drv.ChangedFiles = []*types.FileChange{
		{ID: fileID1, FilePath: "file1.txt", ChangeType: types.ChangeTypeAdded},
		{ID: fileID2, FilePath: "file2.txt", ChangeType: types.ChangeTypeModified},
	}
	svc := newTestService(repo, drv)
	ctx := context.Background()

	id := uuid.New()
	existing := createTestSandbox(id, types.StatusActive)
	repo.Sandboxes[id] = existing

	req := &types.DiscardRequest{
		SandboxID: id,
		FileIDs:   []uuid.UUID{fileID1},
		Actor:     "user",
	}

	result, err := svc.Discard(ctx, req)
	if err != nil {
		t.Fatalf("Discard() error = %v", err)
	}

	if !result.Success {
		t.Error("Discard() should succeed")
	}

	if result.Discarded != 1 {
		t.Errorf("Discard() discarded = %d, want 1", result.Discarded)
	}

	if result.Remaining != 1 {
		t.Errorf("Discard() remaining = %d, want 1", result.Remaining)
	}
}

func TestService_Discard_InvalidState(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	id := uuid.New()
	existing := createTestSandbox(id, types.StatusApproved)
	repo.Sandboxes[id] = existing

	req := &types.DiscardRequest{
		SandboxID: id,
		FileIDs:   []uuid.UUID{uuid.New()},
		Actor:     "user",
	}

	_, err := svc.Discard(ctx, req)
	if err == nil {
		t.Error("Discard() on approved sandbox expected error")
	}
}

// --- GetWorkspacePath Tests ---

func TestService_GetWorkspacePath_Active(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	id := uuid.New()
	existing := createTestSandbox(id, types.StatusActive)
	repo.Sandboxes[id] = existing

	path, err := svc.GetWorkspacePath(ctx, id)
	if err != nil {
		t.Fatalf("GetWorkspacePath() error = %v", err)
	}

	if path != existing.MergedDir {
		t.Errorf("GetWorkspacePath() = %v, want %v", path, existing.MergedDir)
	}
}

func TestService_GetWorkspacePath_Stopped(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	id := uuid.New()
	existing := createTestSandbox(id, types.StatusStopped)
	repo.Sandboxes[id] = existing

	_, err := svc.GetWorkspacePath(ctx, id)
	if err == nil {
		t.Error("GetWorkspacePath() on stopped sandbox expected error")
	}
}

// --- CheckConflicts Tests ---

// [REQ:P2-002] Test conflict detection.
func TestService_CheckConflicts_NoConflicts(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	drv.ChangedFiles = []*types.FileChange{
		{ID: uuid.New(), FilePath: "file1.txt", ChangeType: types.ChangeTypeModified},
	}
	gitOps := mocks.NewFakeGitOps()
	gitOps.ConflictResult = &diff.ConflictCheckResult{HasChanged: false}

	clk := clock.System{}
	svc := NewService(repo, drv, DefaultServiceConfig(), clk, audit.NewRepoEmitter(repo.LogAuditEvent, clk), process.NewOSExecStarter(), WithGitOps(gitOps))
	ctx := context.Background()

	id := uuid.New()
	existing := createTestSandbox(id, types.StatusActive)
	existing.BaseCommitHash = "abc123"
	repo.Sandboxes[id] = existing

	result, err := svc.CheckConflicts(ctx, id)
	if err != nil {
		t.Fatalf("CheckConflicts() error = %v", err)
	}

	if result.HasConflict {
		t.Error("CheckConflicts() HasConflict = true, want false")
	}
}

// [REQ:P2-002] Test conflict detection when repo changed.
func TestService_CheckConflicts_WithConflicts(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	drv.ChangedFiles = []*types.FileChange{
		{ID: uuid.New(), FilePath: "file1.txt", ChangeType: types.ChangeTypeModified},
	}
	gitOps := mocks.NewFakeGitOps()
	gitOps.ConflictResult = &diff.ConflictCheckResult{
		HasChanged:       true,
		CurrentHash:      "def456",
		BaseCommitHash:   "abc123",
		RepoChangedFiles: []string{"file1.txt"},
		ConflictingFiles: []string{"file1.txt"},
	}

	clk := clock.System{}
	svc := NewService(repo, drv, DefaultServiceConfig(), clk, audit.NewRepoEmitter(repo.LogAuditEvent, clk), process.NewOSExecStarter(), WithGitOps(gitOps))
	ctx := context.Background()

	id := uuid.New()
	existing := createTestSandbox(id, types.StatusActive)
	existing.BaseCommitHash = "abc123"
	repo.Sandboxes[id] = existing

	result, err := svc.CheckConflicts(ctx, id)
	if err != nil {
		t.Fatalf("CheckConflicts() error = %v", err)
	}

	if !result.HasConflict {
		t.Error("CheckConflicts() HasConflict = false, want true")
	}

	if len(result.ConflictingFiles) != 1 {
		t.Errorf("CheckConflicts() ConflictingFiles = %d, want 1", len(result.ConflictingFiles))
	}
}

// --- Rebase Tests ---

// [REQ:P2-003] Test rebase updates base commit hash.
func TestService_Rebase_Success(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	gitOps := mocks.NewFakeGitOps()
	gitOps.CommitHash = "new-hash-789"

	clk := clock.System{}
	svc := NewService(repo, drv, DefaultServiceConfig(), clk, audit.NewRepoEmitter(repo.LogAuditEvent, clk), process.NewOSExecStarter(), WithGitOps(gitOps))
	ctx := context.Background()

	id := uuid.New()
	existing := createTestSandbox(id, types.StatusActive)
	existing.BaseCommitHash = "old-hash-123"
	repo.Sandboxes[id] = existing

	req := &types.RebaseRequest{
		SandboxID: id,
		Actor:     "user",
	}

	result, err := svc.Rebase(ctx, req)
	if err != nil {
		t.Fatalf("Rebase() error = %v", err)
	}

	if !result.Success {
		t.Errorf("Rebase() Success = false, ErrorMsg = %s", result.ErrorMsg)
	}

	if result.NewBaseHash != "new-hash-789" {
		t.Errorf("Rebase() NewBaseHash = %s, want new-hash-789", result.NewBaseHash)
	}

	if result.PreviousBaseHash != "old-hash-123" {
		t.Errorf("Rebase() PreviousBaseHash = %s, want old-hash-123", result.PreviousBaseHash)
	}
}

// [REQ:P2-003] Test rebase fails for terminal states.
func TestService_Rebase_TerminalState(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	id := uuid.New()
	existing := createTestSandbox(id, types.StatusApproved)
	repo.Sandboxes[id] = existing

	req := &types.RebaseRequest{
		SandboxID: id,
		Actor:     "user",
	}

	_, err := svc.Rebase(ctx, req)
	if err == nil {
		t.Error("Rebase() on approved sandbox expected error")
	}
}

// --- ValidatePath Tests ---

func TestService_ValidatePath_Valid(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	ctx := context.Background()

	// Create a temp directory for testing (avoids system path restrictions)
	tmpDir, err := os.MkdirTemp("", "validate-path-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	clk := clock.System{}
	svc := NewService(repo, drv, ServiceConfig{
		DefaultProjectRoot: tmpDir,
		MaxSandboxes:       100,
		DefaultTTL:         24 * time.Hour,
	}, clk, audit.NewRepoEmitter(repo.LogAuditEvent, clk), process.NewOSExecStarter(), WithGitOps(mocks.NewFakeGitOps()))

	// Test with the temp dir which exists, is a directory, and not in dangerous paths
	result, err := svc.ValidatePath(ctx, tmpDir, tmpDir)
	if err != nil {
		t.Fatalf("ValidatePath() error = %v", err)
	}

	if !result.Exists {
		t.Errorf("ValidatePath() %s should exist", tmpDir)
	}

	if !result.IsDirectory {
		t.Errorf("ValidatePath() %s should be a directory", tmpDir)
	}

	if !result.Valid {
		t.Errorf("ValidatePath() should be valid, got error: %s", result.Error)
	}
}

func TestService_ValidatePath_RelativePath(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	result, err := svc.ValidatePath(ctx, "relative/path", "/project")
	if err != nil {
		t.Fatalf("ValidatePath() error = %v", err)
	}

	if result.Valid {
		t.Error("ValidatePath() relative path should be invalid")
	}
}

func TestService_ValidatePath_DangerousPath(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	dangerousPaths := []string{"/", "/bin", "/etc", "/usr", "/var"}

	for _, path := range dangerousPaths {
		t.Run(path, func(t *testing.T) {
			result, err := svc.ValidatePath(ctx, path, "")
			if err != nil {
				t.Fatalf("ValidatePath() error = %v", err)
			}

			if result.Valid {
				t.Errorf("ValidatePath(%s) should be invalid for dangerous path", path)
			}
		})
	}
}

// --- List Tests ---

func TestService_List_Success(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	// Pre-create sandboxes
	for i := 0; i < 3; i++ {
		id := uuid.New()
		repo.Sandboxes[id] = createTestSandbox(id, types.StatusActive)
	}

	filter := &types.ListFilter{
		Limit: 10,
	}

	result, err := svc.List(ctx, filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if result.TotalCount != 3 {
		t.Errorf("List() TotalCount = %d, want 3", result.TotalCount)
	}
}

// --- Error Handling Tests ---

func TestService_Create_RepositoryError(t *testing.T) {
	repo := mocks.NewFakeRepository()
	repo.CreateErr = errors.New("database connection failed")
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	req := &types.CreateRequest{
		ScopePath:   "/tmp/project/src",
		ProjectRoot: "/tmp/project",
		Owner:       "test-agent",
	}

	_, err := svc.Create(ctx, req)
	if err == nil {
		t.Fatal("Create() expected error on repository failure")
	}
}

func TestService_Stop_UnmountError(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	drv.UnmountErr = errors.New("unmount failed: device busy")
	svc := newTestService(repo, drv)
	ctx := context.Background()

	id := uuid.New()
	existing := createTestSandbox(id, types.StatusActive)
	repo.Sandboxes[id] = existing

	_, err := svc.Stop(ctx, id)
	if err == nil {
		t.Error("Stop() expected error on unmount failure")
	}
}

func TestService_GetDiff_DriverError(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	drv.GetChangedFilesErr = errors.New("failed to scan upper directory")
	svc := newTestService(repo, drv)
	ctx := context.Background()

	id := uuid.New()
	existing := createTestSandbox(id, types.StatusActive)
	existing.UpperDir = t.TempDir()
	existing.LowerDir = t.TempDir()
	repo.Sandboxes[id] = existing

	_, err := svc.GetDiff(ctx, id)
	if err == nil {
		t.Error("GetDiff() expected error on driver failure")
	}

	var driverErr *types.DriverError
	if !errors.As(err, &driverErr) {
		t.Errorf("GetDiff() error type = %T, want *DriverError", err)
	}
}

// --- Provenance Tests ---

func TestService_GetPendingChanges_Success(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	result, err := svc.GetPendingChanges(ctx, "/project", 10, 0)
	if err != nil {
		t.Fatalf("GetPendingChanges() error = %v", err)
	}

	if result == nil {
		t.Error("GetPendingChanges() returned nil")
	}
}

func TestService_GetFileProvenance_Success(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	result, err := svc.GetFileProvenance(ctx, "/project/file.txt", "/project", 10)
	if err != nil {
		t.Fatalf("GetFileProvenance() error = %v", err)
	}

	if result == nil {
		t.Error("GetFileProvenance() returned nil")
	}
}

// --- Benchmark Tests ---

func BenchmarkService_Create(b *testing.B) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := &types.CreateRequest{
			ScopePath:   "/tmp/project/src",
			ProjectRoot: "/tmp/project",
			Owner:       "benchmark",
		}
		if _, err := svc.Create(ctx, req); err != nil {
			b.Fatalf("Create() error = %v", err)
		}
	}
}

func BenchmarkService_Get(b *testing.B) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc := newTestService(repo, drv)
	ctx := context.Background()

	id := uuid.New()
	repo.Sandboxes[id] = createTestSandbox(id, types.StatusActive)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := svc.Get(ctx, id); err != nil {
			b.Fatalf("Get() error = %v", err)
		}
	}
}
