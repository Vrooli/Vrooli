package identity_test

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration/testutil"

	"github.com/google/uuid"
)

func TestTokenHashStoredAndRetrievable(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	ctx := context.Background()

	secret := []byte("test-secret-key-32-bytes-long!!")

	// Create a task (FK requirement).
	task := &domain.Task{
		ID:          uuid.New(),
		Title:       "test task",
		Description: "integration test",
		Status:      domain.TaskStatusQueued,
	}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	// Generate token and hash.
	claims := &identity.Claims{
		RunID:     uuid.New(),
		TaskID:    task.ID,
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(identity.DefaultTTL).Unix(),
		Meta:      map[string]string{},
	}
	token, err := identity.GenerateToken(claims, secret)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	tokenHash := identity.HashToken(token)

	// Create a run with the token hash.
	run := &domain.Run{
		ID:                claims.RunID,
		TaskID:            task.ID,
		Status:            domain.RunStatusRunning,
		RunMode:           domain.RunModeSandboxed,
		Phase:             domain.RunPhaseExecuting,
		IdentityTokenHash: tokenHash,
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	// Look up by token hash.
	found, err := repos.Runs.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		t.Fatalf("GetByTokenHash: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find run by token hash, got nil")
	}
	if found.ID != run.ID {
		t.Errorf("found run ID = %v, want %v", found.ID, run.ID)
	}
	if found.IdentityTokenHash != tokenHash {
		t.Errorf("found hash = %q, want %q", found.IdentityTokenHash, tokenHash)
	}
}

func TestTokenRevocationPersisted(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	ctx := context.Background()

	task := &domain.Task{
		ID:     uuid.New(),
		Title:  "test",
		Status: domain.TaskStatusQueued,
	}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	now := time.Now().Truncate(time.Second)
	run := &domain.Run{
		ID:                     uuid.New(),
		TaskID:                 task.ID,
		Status:                 domain.RunStatusComplete,
		RunMode:                domain.RunModeSandboxed,
		IdentityTokenHash:      "somehash",
		IdentityTokenRevokedAt: &now,
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	got, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if got.IdentityTokenRevokedAt == nil {
		t.Fatal("expected IdentityTokenRevokedAt to be set, got nil")
	}
}

func TestGetByTokenHashNotFound(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	found, err := repos.Runs.GetByTokenHash(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByTokenHash: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil for nonexistent hash, got %v", found)
	}
}
