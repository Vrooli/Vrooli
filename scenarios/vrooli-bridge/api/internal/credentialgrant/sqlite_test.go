package credentialgrant

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestSQLiteRepositoryBumpGenerationContinuesExistingGrantGeneration(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, statement := range []string{
		`CREATE TABLE credential_grants (logical_id TEXT NOT NULL, field TEXT NOT NULL, generation INTEGER NOT NULL, revoked_at TEXT NOT NULL DEFAULT '')`,
		`CREATE TABLE credential_generations (logical_id TEXT NOT NULL, field TEXT NOT NULL, generation INTEGER NOT NULL, updated_at TEXT NOT NULL, PRIMARY KEY (logical_id, field))`,
		`INSERT INTO credential_grants (logical_id, field, generation, revoked_at) VALUES ('vrooli/test', 'token', 7, '')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}

	repo := NewSQLiteRepository(db, func() time.Time { return time.Unix(10, 0) })
	generation, err := repo.BumpGeneration(context.Background(), "vrooli/test", "token")
	if err != nil {
		t.Fatal(err)
	}
	if generation != 8 {
		t.Fatalf("first rotation generation = %d, want 8", generation)
	}

	generation, err = repo.BumpGeneration(context.Background(), "vrooli/test", "token")
	if err != nil {
		t.Fatal(err)
	}
	if generation != 9 {
		t.Fatalf("second rotation generation = %d, want 9", generation)
	}
}

func TestSQLiteRepositoryPersistsCredentialReceiptMetadata(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(schemaSQL); err != nil {
		t.Fatal(err)
	}
	now := time.Unix(20, 0).UTC()
	repo := NewSQLiteRepository(db, func() time.Time { return now })
	grant, err := repo.Create(context.Background(), Grant{NodeID: "node-1", LogicalID: "vrooli/test", Field: "token", Class: ClassUserPrompt, Retention: RetentionDurable, Generation: 2})
	if err != nil {
		t.Fatal(err)
	}
	receiptAt := now.Add(time.Minute)
	if err := repo.RecordReceipt(context.Background(), grant.ID, 2, true, "", receiptAt); err != nil {
		t.Fatal(err)
	}
	grants, err := repo.List(context.Background(), "node-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(grants) != 1 || grants[0].AckedGeneration != 2 || !grants[0].ReceiptAccepted || !grants[0].ReceiptAt.Equal(receiptAt) {
		t.Fatalf("receipt metadata = %#v, want accepted generation 2 at %s", grants, receiptAt)
	}
}

func TestSQLiteRepositoryTracksPendingAndAcknowledgedPurge(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(schemaSQL); err != nil {
		t.Fatal(err)
	}
	now := time.Unix(30, 0).UTC()
	repo := NewSQLiteRepository(db, func() time.Time { return now })
	grant, err := repo.Create(context.Background(), Grant{NodeID: "node-1", LogicalID: "vrooli/test", Field: "token", Class: ClassUserPrompt, Retention: RetentionEphemeral, Generation: 4})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Revoke(context.Background(), grant.ID); err != nil {
		t.Fatal(err)
	}
	revoked, err := repo.ListRevoked(context.Background(), "node-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(revoked) != 1 || revoked[0].PurgeState != "pending" {
		t.Fatalf("revoked purge state = %#v, want one pending grant", revoked)
	}
	receiptAt := now.Add(time.Minute)
	if err := repo.RecordPurgeReceipt(context.Background(), grant.ID, "node-1", 4, true, "deleted", receiptAt); err != nil {
		t.Fatal(err)
	}
	revoked, err = repo.ListRevoked(context.Background(), "node-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(revoked) != 1 || revoked[0].PurgeState != "acknowledged" || !revoked[0].PurgeAccepted || !revoked[0].PurgeReceiptAt.Equal(receiptAt) {
		t.Fatalf("purge receipt = %#v, want acknowledged receipt at %s", revoked, receiptAt)
	}
}

func TestSQLiteMutationReceiptsReplayAcrossRestartAndRefuseChangedBindings(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "grants.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	if _, err = db.Exec(schemaSQL); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`CREATE TABLE nodes (id TEXT PRIMARY KEY, kind TEXT NOT NULL, enrollment_generation INTEGER NOT NULL, revoked_at TEXT NOT NULL DEFAULT '')`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO nodes (id,kind,enrollment_generation,revoked_at) VALUES ('node-1','agent',7,'')`); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	repo := NewSQLiteRepository(db, func() time.Time { return now })
	createBinding := MutationBinding{RequestID: "request-create", ActorID: "owner-a", NodeID: "node-1", EnrollmentGeneration: 7, Subject: CreateGrantSubject, ExpectedRevision: 0, PayloadDigest: "sha256:" + strings.Repeat("a", 64)}
	grant, created, err := repo.CreateWithReceipt(context.Background(), Grant{NodeID: "node-1", LogicalID: "service/api", Field: "token", Class: ClassUserPrompt, Retention: RetentionEphemeral, Generation: 1}, createBinding)
	if err != nil || created.Replayed || created.ResultRevision != 1 {
		t.Fatalf("create=%+v receipt=%+v err=%v", grant, created, err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	repo = NewSQLiteRepository(db, func() time.Time { return now.Add(time.Minute) })
	replayedGrant, replay, err := repo.CreateWithReceipt(context.Background(), Grant{NodeID: "node-1", LogicalID: "service/api", Field: "token", Class: ClassUserPrompt, Retention: RetentionEphemeral, Generation: 1}, createBinding)
	if err != nil || !replay.Replayed || replay.RequestID != created.RequestID || replay.GrantID != grant.ID || replayedGrant.ID != grant.ID {
		t.Fatalf("create replay=%+v receipt=%+v err=%v", replayedGrant, replay, err)
	}
	changed := createBinding
	changed.ActorID = "owner-b"
	if _, _, err = repo.CreateWithReceipt(context.Background(), Grant{NodeID: "node-1", LogicalID: "service/api", Field: "token", Class: ClassUserPrompt, Retention: RetentionEphemeral}, changed); !errors.Is(err, ErrMutationConflict) {
		t.Fatalf("changed actor=%v", err)
	}
	changed = createBinding
	changed.PayloadDigest = "sha256:" + strings.Repeat("b", 64)
	if _, _, err = repo.CreateWithReceipt(context.Background(), Grant{NodeID: "node-1", LogicalID: "service/api", Field: "token", Class: ClassUserPrompt, Retention: RetentionEphemeral}, changed); !errors.Is(err, ErrMutationConflict) {
		t.Fatalf("changed payload=%v", err)
	}
	staleCreate := createBinding
	staleCreate.RequestID = "stale-create"
	staleCreate.EnrollmentGeneration = 6
	if _, _, err = repo.CreateWithReceipt(context.Background(), Grant{NodeID: "node-1", LogicalID: "other", Field: "token", Class: ClassUserPrompt, Retention: RetentionEphemeral}, staleCreate); !errors.As(err, new(ErrStaleNodeGeneration)) {
		t.Fatalf("stale enrollment=%v", err)
	}

	revokeBinding := MutationBinding{RequestID: "request-revoke", ActorID: "owner-a", NodeID: "node-1", EnrollmentGeneration: 7, Subject: RevokeGrantSubject + ":" + grant.ID, ExpectedRevision: 1, PayloadDigest: "sha256:" + strings.Repeat("c", 64)}
	revoked, revReceipt, err := repo.RevokeWithReceipt(context.Background(), grant.ID, revokeBinding)
	if err != nil || revoked.Revision != 2 || revReceipt.ResultRevision != 2 || revReceipt.Outcome != "revoked" {
		t.Fatalf("revoke=%+v receipt=%+v err=%v", revoked, revReceipt, err)
	}
	replayGrant, replayReceipt, err := repo.RevokeWithReceipt(context.Background(), grant.ID, revokeBinding)
	if err != nil || !replayReceipt.Replayed || replayGrant.Revision != 2 || replayReceipt.ResultRevision != 2 {
		t.Fatalf("revoke replay=%+v receipt=%+v err=%v", replayGrant, replayReceipt, err)
	}
	changedRevoke := revokeBinding
	changedRevoke.NodeID = "node-other"
	if _, _, err = repo.RevokeWithReceipt(context.Background(), grant.ID, changedRevoke); !errors.Is(err, ErrMutationConflict) {
		t.Fatalf("changed target=%v", err)
	}
	staleRevoke := revokeBinding
	staleRevoke.RequestID = "stale-revoke"
	staleRevoke.ExpectedRevision = 1
	if _, _, err = repo.RevokeWithReceipt(context.Background(), grant.ID, staleRevoke); !errors.Is(err, ErrGrantRevisionConflict) {
		t.Fatalf("stale revision=%v", err)
	}
}
