package database

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/authn"
	requestidentity "github.com/vrooli/api-core/identity"
)

func runCreationStandingGrant(id, subject, owner string, exp time.Time) *domain.StandingGrant {
	return &domain.StandingGrant{
		ID:      id,
		Subject: subject,
		Grant: &requestidentity.OperationGrant{
			ID: id, Approver: owner, Policy: "run-creation",
			Operations: []string{"run.create"}, Targets: []string{"agent-manager:*"}, ExpiresAt: exp,
		},
		Limits:     authn.StandingGrantLimits{MaxConcurrentRuns: 3},
		State:      domain.StandingGrantApproved,
		ApprovedBy: owner,
		ApprovedAt: time.Now().Add(-time.Minute),
		ExpiresAt:  exp,
	}
}

func TestStandingGrantStorePutAndActiveForSubject(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewStandingGrantRepository(db)
	ctx := context.Background()
	now := time.Now()

	g := runCreationStandingGrant(uuid.NewString(), "prompt-manager", "operator", now.Add(time.Hour))
	if err := repo.Put(ctx, g); err != nil {
		t.Fatalf("put: %v", err)
	}
	active, err := repo.ActiveForSubject(ctx, "prompt-manager", now)
	if err != nil {
		t.Fatalf("active: %v", err)
	}
	if len(active) != 1 || active[0].ID != g.ID || active[0].Grant.Approver != "operator" {
		t.Fatalf("expected one active grant for the subject, got %+v", active)
	}
	if active[0].Limits.MaxConcurrentRuns != 3 {
		t.Fatalf("limits not round-tripped: %+v", active[0].Limits)
	}
	// A different subject sees nothing.
	if other, _ := repo.ActiveForSubject(ctx, "swarm-manager", now); len(other) != 0 {
		t.Fatalf("grant leaked across subjects: %+v", other)
	}
}

func TestStandingGrantStoreExcludesExpired(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewStandingGrantRepository(db)
	ctx := context.Background()
	now := time.Now()

	g := runCreationStandingGrant(uuid.NewString(), "prompt-manager", "operator", now.Add(-time.Second))
	if err := repo.Put(ctx, g); err != nil {
		t.Fatalf("put: %v", err)
	}
	active, err := repo.ActiveForSubject(ctx, "prompt-manager", now)
	if err != nil {
		t.Fatalf("active: %v", err)
	}
	if len(active) != 0 {
		t.Fatalf("expired grant was returned as active: %+v", active)
	}
}

func TestStandingGrantStoreRevokeDeniesNextLookup(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewStandingGrantRepository(db)
	ctx := context.Background()
	now := time.Now()

	g := runCreationStandingGrant(uuid.NewString(), "prompt-manager", "operator", now.Add(time.Hour))
	if err := repo.Put(ctx, g); err != nil {
		t.Fatalf("put: %v", err)
	}
	transitioned, err := repo.Revoke(ctx, g.ID, now)
	if err != nil || !transitioned {
		t.Fatalf("revoke should transition an approved grant: %v %v", transitioned, err)
	}
	if active, _ := repo.ActiveForSubject(ctx, "prompt-manager", now); len(active) != 0 {
		t.Fatalf("revoked grant still active on next lookup: %+v", active)
	}
	// Re-revoking is a no-op.
	if again, _ := repo.Revoke(ctx, g.ID, now); again {
		t.Fatal("re-revoking reported a transition")
	}
}

func TestStandingGrantListForApproverFiltersBeforeLimit(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewStandingGrantRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	for i := 0; i < 70; i++ {
		grant := runCreationStandingGrant(uuid.NewString(), "prompt-manager", "other-owner", now.Add(time.Hour))
		grant.ApprovedAt = now.Add(time.Duration(i) * time.Second)
		if err := repo.Put(ctx, grant); err != nil {
			t.Fatalf("put foreign decoy %d: %v", i, err)
		}
	}
	// An inconsistent JSON approver must not consume the owner's bounded page.
	mismatched := runCreationStandingGrant(uuid.NewString(), "prompt-manager", "other-owner", now.Add(time.Hour))
	mismatched.ApprovedBy = "owner-1"
	mismatched.ApprovedAt = now.Add(71 * time.Second)
	if err := repo.Put(ctx, mismatched); err != nil {
		t.Fatalf("put inconsistent decoy: %v", err)
	}
	wanted := runCreationStandingGrant("owner-grant", "prompt-manager", "owner-1", now.Add(time.Hour))
	wanted.ApprovedAt = now.Add(-time.Hour)
	if err := repo.Put(ctx, wanted); err != nil {
		t.Fatalf("put owner grant: %v", err)
	}

	grants, err := repo.ListForApprover(ctx, "owner-1", 1)
	if err != nil {
		t.Fatalf("list owner grants: %v", err)
	}
	if len(grants) != 1 || grants[0].ID != wanted.ID || grants[0].ApprovedBy != "owner-1" || grants[0].Grant.Approver != "owner-1" {
		t.Fatalf("owner-scoped limit was applied before both approver filters: %+v", grants)
	}
}
