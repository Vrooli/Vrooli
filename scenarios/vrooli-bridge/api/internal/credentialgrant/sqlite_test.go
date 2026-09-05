package credentialgrant

import (
	"context"
	"database/sql"
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
