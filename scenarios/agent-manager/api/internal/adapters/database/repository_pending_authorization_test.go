package database

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/domain"
)

func pendingRequest(id, subject string, now time.Time) *domain.PendingAuthorizationRequest {
	return &domain.PendingAuthorizationRequest{
		ID:                       id,
		Subject:                  subject,
		Operation:                "run.create",
		Target:                   "agent-manager:*",
		Kind:                     domain.PendingAuthorizationKindStanding,
		MaxConcurrentRuns:        3,
		DecisionBigPicture:       "A verified caller requests a bounded grant.",
		DecisionTechnicalDetails: "Operation: run.create; target: agent-manager:*.",
		State:                    domain.PendingAuthorizationPending,
		RaisedAt:                 now,
		ExpiresAt:                now.Add(24 * time.Hour),
	}
}

func TestPendingAuthorizationStorePutGetAndCount(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewPendingAuthorizationRepository(db)
	ctx := context.Background()
	now := time.Now()

	// Unknown id → nil, no error.
	if got, err := repo.Get(ctx, "auth-missing"); err != nil || got != nil {
		t.Fatalf("missing get: %v %v", got, err)
	}

	p := pendingRequest("auth-1", "prompt-manager", now)
	p.EventID = "event-generation-1"
	publishedAt := now.Add(time.Second)
	p.PublishedAt = &publishedAt
	if err := repo.Put(ctx, p); err != nil {
		t.Fatalf("put: %v", err)
	}
	got, err := repo.Get(ctx, "auth-1")
	if err != nil || got == nil || got.Subject != "prompt-manager" || got.MaxConcurrentRuns != 3 || got.DecisionBigPicture != p.DecisionBigPicture || got.DecisionTechnicalDetails != p.DecisionTechnicalDetails || got.State != domain.PendingAuthorizationPending || got.EventID != p.EventID || got.PublishedAt == nil || !got.PublishedAt.Equal(publishedAt) {
		t.Fatalf("get round-trip lost fields: %+v %v", got, err)
	}

	// One outstanding ask for the subject; a different subject sees zero.
	if n, _ := repo.CountOutstandingForSubject(ctx, "prompt-manager", now); n != 1 {
		t.Fatalf("expected 1 outstanding, got %d", n)
	}
	if n, _ := repo.CountOutstandingForSubject(ctx, "other", now); n != 0 {
		t.Fatalf("expected 0 outstanding for a different subject, got %d", n)
	}
}

func TestPendingAuthorizationCountExcludesDecidedAndExpired(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewPendingAuthorizationRepository(db)
	ctx := context.Background()
	now := time.Now()

	// Approved (decided) → not outstanding.
	approved := pendingRequest("auth-approved", "prompt-manager", now)
	approved.State = domain.PendingAuthorizationApproved
	_ = repo.Put(ctx, approved)

	// Expired → not outstanding.
	expired := pendingRequest("auth-expired", "prompt-manager", now.Add(-48*time.Hour))
	expired.ExpiresAt = now.Add(-time.Hour)
	_ = repo.Put(ctx, expired)

	// One genuinely outstanding.
	_ = repo.Put(ctx, pendingRequest("auth-live", "prompt-manager", now))

	n, err := repo.CountOutstandingForSubject(ctx, "prompt-manager", now)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("decided+expired must not count; expected 1 outstanding, got %d", n)
	}
}

func TestPendingAuthorizationPutReplacesByID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewPendingAuthorizationRepository(db)
	ctx := context.Background()
	now := time.Now()

	p := pendingRequest("auth-1", "prompt-manager", now)
	_ = repo.Put(ctx, p)
	// Resolve it (same id) → the row is replaced, not duplicated.
	p.State = domain.PendingAuthorizationApproved
	resolvedAt := now
	p.ResolvedAt = &resolvedAt
	p.ResolvedBy = "operator-human"
	p.GrantID = "sg-x"
	if err := repo.Put(ctx, p); err != nil {
		t.Fatal(err)
	}
	got, _ := repo.Get(ctx, "auth-1")
	if got == nil || got.State != domain.PendingAuthorizationApproved || got.ResolvedBy != "operator-human" || got.GrantID != "sg-x" {
		t.Fatalf("resolution did not persist by id: %+v", got)
	}
	if n, _ := repo.CountOutstandingForSubject(ctx, "prompt-manager", now); n != 0 {
		t.Fatalf("resolved ask must not be outstanding, got %d", n)
	}
}
