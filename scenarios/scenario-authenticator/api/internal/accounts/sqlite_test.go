package accounts

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"scenario-authenticator/internal/clock"
	dbtest "scenario-authenticator/internal/testutil/db"

	apidb "github.com/vrooli/api-core/database"
)

func newSchemaDB(t *testing.T) *sql.DB {
	t.Helper()
	d := dbtest.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("ensure schemas: %v", err)
	}
	return d
}

func newRepo(t *testing.T) (Repository, *sql.DB) {
	d := newSchemaDB(t)
	return NewSQLiteRepository(d, clock.System{}), d
}

// TestSchemaIdempotentAndSeedsDefaultRealm boots the schema twice (no drift
// error) and asserts the default realm row + its audience are seeded.
func TestSchemaIdempotentAndSeedsDefaultRealm(t *testing.T) {
	d := newSchemaDB(t)
	// Boot again — must be a clean no-op (EnsureSchemas drift check included).
	if err := apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("second EnsureSchemas: %v", err)
	}
	repo := NewSQLiteRepository(d, clock.System{})
	aud, err := repo.RealmAudience(context.Background(), "default")
	if err != nil {
		t.Fatalf("realm audience: %v", err)
	}
	if aud != "scenario-authenticator:default" {
		t.Fatalf("default realm aud = %q", aud)
	}
	if _, err := repo.RealmAudience(context.Background(), "nope"); err != ErrRealmNotFound {
		t.Fatalf("want ErrRealmNotFound, got %v", err)
	}
}

func TestCreateFindDuplicate(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	hash, _ := HashPassword("pw")

	acc, err := repo.Create(ctx, CreateInput{RealmID: "default", Email: "a@b.co", Username: "alice", PasswordHash: hash})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if acc.ID == "" || len(acc.Roles) != 1 || acc.Roles[0] != "user" {
		t.Fatalf("unexpected account: %+v", acc)
	}

	got, gotHash, err := repo.FindByEmail(ctx, "default", "a@b.co")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.ID != acc.ID || gotHash != hash {
		t.Fatalf("find mismatch: %+v hash=%q", got, gotHash)
	}

	if _, err := repo.Create(ctx, CreateInput{RealmID: "default", Email: "a@b.co", PasswordHash: hash}); err != ErrEmailTaken {
		t.Fatalf("want ErrEmailTaken, got %v", err)
	}

	if _, _, err := repo.FindByEmail(ctx, "default", "missing@b.co"); err != ErrAccountNotFound {
		t.Fatalf("want ErrAccountNotFound, got %v", err)
	}
}

func TestLoginFailureAndSuccess(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	hash, _ := HashPassword("pw")
	acc, _ := repo.Create(ctx, CreateInput{RealmID: "default", Email: "a@b.co", PasswordHash: hash})

	lockUntil := time.Now().Add(15 * time.Minute).UTC().Truncate(time.Second)
	if err := repo.SetLoginFailure(ctx, acc.ID, 5, lockUntil); err != nil {
		t.Fatalf("set failure: %v", err)
	}
	got, _ := repo.FindByID(ctx, acc.ID)
	if got.FailedLoginAttempts != 5 {
		t.Fatalf("attempts = %d", got.FailedLoginAttempts)
	}
	if !got.Locked(time.Now()) {
		t.Fatal("account should be locked")
	}

	now := time.Now().UTC()
	if err := repo.SetLoginSuccess(ctx, acc.ID, now); err != nil {
		t.Fatalf("set success: %v", err)
	}
	got, _ = repo.FindByID(ctx, acc.ID)
	if got.FailedLoginAttempts != 0 || got.Locked(time.Now()) {
		t.Fatalf("lock not cleared: %+v", got)
	}
	if got.LastLogin.IsZero() {
		t.Fatal("last_login not set")
	}
}
