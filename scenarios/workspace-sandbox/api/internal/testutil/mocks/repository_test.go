package mocks

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"workspace-sandbox/internal/types"
)

func TestFakeRepository_DefaultsRoundTrip(t *testing.T) {
	repo := NewFakeRepository()
	ctx := context.Background()

	id := uuid.New()
	s := &types.Sandbox{ID: id, Status: types.StatusActive}
	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil || got.ID != id {
		t.Fatalf("Get returned %v, want sandbox with id %s", got, id)
	}
	if got.Version != 1 {
		t.Errorf("Create should set Version=1, got %d", got.Version)
	}

	// AuditEvents starts empty.
	if got := repo.AuditEventCount("anything"); got != 0 {
		t.Errorf("AuditEventCount on empty repo = %d", got)
	}
}

func TestFakeRepository_ErrorInjection(t *testing.T) {
	repo := NewFakeRepository()
	repo.GetErr = errors.New("boom")

	if _, err := repo.Get(context.Background(), uuid.New()); err == nil {
		t.Fatal("expected GetErr to surface")
	}
}

func TestFakeRepository_AuditEventCountByType(t *testing.T) {
	repo := NewFakeRepository()
	ctx := context.Background()

	for _, et := range []string{"a", "b", "a", "c", "a"} {
		if err := repo.LogAuditEvent(ctx, &types.AuditEvent{EventType: et}); err != nil {
			t.Fatalf("LogAuditEvent: %v", err)
		}
	}

	if got := repo.AuditEventCount("a"); got != 3 {
		t.Errorf("AuditEventCount(a) = %d, want 3", got)
	}
	if got := repo.AuditEventCount("b"); got != 1 {
		t.Errorf("AuditEventCount(b) = %d, want 1", got)
	}
	if got := repo.AuditEventCount("z"); got != 0 {
		t.Errorf("AuditEventCount(z) = %d, want 0", got)
	}
}

func TestFakeRepository_DeleteFailIDs(t *testing.T) {
	repo := NewFakeRepository()
	ctx := context.Background()

	good := uuid.New()
	bad := uuid.New()
	if err := repo.Create(ctx, &types.Sandbox{ID: good}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, &types.Sandbox{ID: bad}); err != nil {
		t.Fatal(err)
	}

	repo.DeleteErr = errors.New("planned failure")
	repo.DeleteFailIDs[bad] = true

	if err := repo.Delete(ctx, good); err != nil {
		t.Errorf("Delete(good) should succeed (only bad is configured to fail), got %v", err)
	}
	if err := repo.Delete(ctx, bad); err == nil {
		t.Error("Delete(bad) should fail per DeleteFailIDs")
	}
}
