package redisstate

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
)

// This is the security property the durable store exists for, exercised the way
// the scenario wires it rather than through a hand-built fake: the schema is
// applied through the same EnsureSchemas substrate main.go uses, the store is
// wrapped in the same production namespace, and the handle is closed and
// reopened to stand in for a process restart.
//
// A revocation blacklist that does not survive restart re-admits a revoked
// token, so "the process came back up" is not enough — the entry has to still
// be there, and an expired one still has to read as absent.
func TestBlacklistSurvivesRestartWithoutRedis(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "scenario-authenticator.db")

	open := func() (*sql.DB, *NamespacedStore) {
		t.Helper()
		db, err := sql.Open("sqlite", path)
		if err != nil {
			t.Fatalf("open database: %v", err)
		}
		db.SetMaxOpenConns(1)
		if err := database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(Schema)); err != nil {
			t.Fatalf("ensure schemas: %v", err)
		}
		durable, err := NewSQLiteStore(db)
		if err != nil {
			t.Fatalf("build durable store: %v", err)
		}
		namespace, err := storage.ResolveNamespace(storage.NamespaceConfig{FallbackScenario: "scenario-authenticator"})
		if err != nil {
			t.Fatalf("resolve namespace: %v", err)
		}
		scoped, err := NewNamespacedStore(durable, namespace, "auth")
		if err != nil {
			t.Fatalf("scope namespace: %v", err)
		}
		return db, scoped
	}

	const revoked = "blacklist:revoked-token-sha"
	const stillValid = "blacklist:untouched-token-sha"

	db, store := open()
	if err := store.Set(ctx, revoked, "revoked", time.Hour); err != nil {
		t.Fatalf("blacklist token: %v", err)
	}
	if err := store.SAdd(ctx, "refreshfamily:user-1", "token-a", "token-b"); err != nil {
		t.Fatalf("record refresh family: %v", err)
	}
	if _, err := store.Incr(ctx, "ratelimit:user-1"); err != nil {
		t.Fatalf("increment rate limit: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close database: %v", err)
	}

	// Restart.
	db, store = open()
	defer db.Close()

	present, err := store.Exists(ctx, revoked)
	if err != nil {
		t.Fatalf("check blacklist: %v", err)
	}
	if !present {
		t.Fatal("the revoked token is no longer blacklisted after restart; it would be re-admitted")
	}
	if absent, _ := store.Exists(ctx, stillValid); absent {
		t.Fatal("a token that was never blacklisted reads as blacklisted")
	}
	members, err := store.SMembers(ctx, "refreshfamily:user-1")
	if err != nil || len(members) != 2 {
		t.Fatalf("refresh family did not survive restart: %v err=%v", members, err)
	}
	next, err := store.Incr(ctx, "ratelimit:user-1")
	if err != nil || next != 2 {
		t.Fatalf("rate-limit counter reset across restart: got %d err=%v", next, err)
	}
}

// An expired blacklist entry must read as absent the moment it expires, not
// whenever a sweep happens to run, or a revoked token is honoured again for the
// length of the sweep interval.
func TestExpiredBlacklistEntryIsRefusedImmediatelyAfterRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "scenario-authenticator.db")
	clock := time.Unix(1_000_000, 0)

	open := func() *SQLiteStore {
		t.Helper()
		db, err := sql.Open("sqlite", path)
		if err != nil {
			t.Fatalf("open database: %v", err)
		}
		db.SetMaxOpenConns(1)
		if err := database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(Schema)); err != nil {
			t.Fatalf("ensure schemas: %v", err)
		}
		t.Cleanup(func() { _ = db.Close() })
		store, err := NewSQLiteStoreWithClock(db, func() time.Time { return clock })
		if err != nil {
			t.Fatalf("build durable store: %v", err)
		}
		return store
	}

	store := open()
	if err := store.Set(ctx, "blacklist:short-lived", "revoked", time.Minute); err != nil {
		t.Fatalf("blacklist token: %v", err)
	}
	clock = clock.Add(2 * time.Minute)

	reopened := open()
	if present, _ := reopened.Exists(ctx, "blacklist:short-lived"); present {
		t.Fatal("an expired blacklist entry survived restart as present")
	}
	removed, err := reopened.Sweep(ctx)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if removed != 1 {
		t.Fatalf("sweep reclaimed %d rows, want 1", removed)
	}
}
