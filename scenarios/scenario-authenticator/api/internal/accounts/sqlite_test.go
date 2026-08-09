package accounts

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"scenario-authenticator/internal/authorization"
	"scenario-authenticator/internal/clock"
	dbtest "scenario-authenticator/internal/testutil/db"

	apidb "github.com/vrooli/api-core/database"
)

func newSchemaDB(t *testing.T) *sql.DB {
	t.Helper()
	d := dbtest.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema), apidb.SchemaProviderFunc(authorization.Schema)); err != nil {
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

func TestScopeStoreKeepsOpaqueAssignments(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	hash, _ := HashPassword("pw")
	acc, err := repo.Create(ctx, CreateInput{RealmID: "default", Email: "scope@b.co", PasswordHash: hash})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	store, ok := repo.(interface {
		GrantScope(context.Context, string, string) ([]string, error)
		RevokeScope(context.Context, string, string) ([]string, error)
		ListScopes(context.Context, string) ([]string, error)
	})
	if !ok {
		t.Fatal("repository does not expose scope storage")
	}
	want := "some:opaque.scope/v1"
	scopes, err := store.GrantScope(ctx, acc.ID, want)
	if err != nil {
		t.Fatalf("grant: %v", err)
	}
	if len(scopes) != 1 || scopes[0] != want {
		t.Fatalf("scopes after grant = %#v", scopes)
	}
	if _, err := store.GrantScope(ctx, acc.ID, want); err != nil {
		t.Fatalf("duplicate grant: %v", err)
	}
	scopes, err = store.RevokeScope(ctx, acc.ID, want)
	if err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if len(scopes) != 0 {
		t.Fatalf("scopes after revoke = %#v", scopes)
	}
}

func TestMachineBindingsAllowManyRowsButResolveOneDefault(t *testing.T) {
	repo, db := newRepo(t)
	ctx := context.Background()
	hash, _ := HashPassword("pw")
	acc1, err := repo.Create(ctx, CreateInput{RealmID: "default", Email: "machine1@b.co", PasswordHash: hash})
	if err != nil {
		t.Fatal(err)
	}
	acc2, err := repo.Create(ctx, CreateInput{RealmID: "default", Email: "machine2@b.co", PasswordHash: hash})
	if err != nil {
		t.Fatal(err)
	}
	store := repo.(MachineBindingStore)
	first, err := store.LinkMachineBinding(ctx, MachineBinding{MachineID: "mac-1", LocalPrincipal: "unix:1000", AccountID: acc1.ID, RealmID: "default", IsDefault: true, LinkedAt: time.Unix(1, 0)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.LinkMachineBinding(ctx, MachineBinding{MachineID: "mac-1", LocalPrincipal: "unix:1000", AccountID: acc2.ID, RealmID: "default", IsDefault: false, LinkedAt: time.Unix(2, 0)}); err != nil {
		t.Fatal(err)
	}
	got, err := store.ResolveDefaultMachineBinding(ctx, "mac-1", "unix:1000")
	if err != nil || got.ID != first.ID {
		t.Fatalf("default = %+v, %v", got, err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO machine_bindings (id, machine_id, local_principal, account_id, realm_id, is_default, linked_at) VALUES (?, ?, ?, ?, ?, 1, ?)`, "ambiguous", "mac-2", "unix:1000", acc2.ID, "default", time.Now().UTC().Format(timeFormat)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO machine_bindings (id, machine_id, local_principal, account_id, realm_id, is_default, linked_at) VALUES (?, ?, ?, ?, ?, 1, ?)`, "ambiguous-2", "mac-2", "unix:1000", acc1.ID, "default", time.Now().UTC().Format(timeFormat)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ResolveDefaultMachineBinding(ctx, "mac-2", "unix:1000"); err != ErrMachineBindingAmbiguous {
		t.Fatalf("ambiguous resolution = %v", err)
	}
}
