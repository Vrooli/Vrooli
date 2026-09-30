package runhistory

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"

	"unit-health/internal/evidence"
)

func testIdentity(t *testing.T, source, toolchain string) *ComparisonIdentity {
	t.Helper()
	key, err := evidence.NewKey(evidence.KeyInput{
		SourceDigest: source, ConfigDigest: "config", DependencyLockDigest: "lock",
		ToolchainIdentity: toolchain, AdapterID: "go", AdapterVersion: "1",
		CoverageMode: "test", ArtifactSchema: "unit-health.response.v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	return &ComparisonIdentity{Evidence: key, Selection: "go test ./..."}
}

func TestIdentitySurvivesDatabaseReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "history.sqlite")
	open := func() *sql.DB {
		t.Helper()
		db, err := sql.Open("sqlite", path)
		if err != nil {
			t.Fatal(err)
		}
		db.SetMaxOpenConns(1)
		if err := database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(Schema)); err != nil {
			_ = db.Close()
			t.Fatal(err)
		}
		return db
	}
	db := open()
	rec := sampleRun("persisted", time.Unix(1700000000, 0), "failed", 42)
	rec.Commands[0].Identity = testIdentity(t, "source", "go1")
	seed, retry := "123", 0
	rec.Commands[0].Identity.Seed = &seed
	rec.Commands[0].Identity.RetryOrdinal = &retry
	rec.NativeTests = []NativeTestSample{{RunID: rec.RunID, NativeRunID: "runner-1", WorkspaceID: "ui", File: "a.test.ts", TestID: "case-1", State: "pass", Seed: &seed, RetryCount: &retry}}
	if err := NewRepository(db).Record(ctx, rec); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db = open()
	defer db.Close()
	history, err := NewRepository(db).CommandHistory(ctx, "demo", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || history[0].Status != "failed" || history[0].DurationMS != 42 || !rec.Commands[0].Identity.Comparable(history[0].Identity) {
		t.Fatalf("reopened history differs: %+v", history)
	}
	native, err := NewRepository(db).NativeTestHistory(ctx, "demo", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(native) != 1 || native[0].TestID != "case-1" || native[0].Seed == nil || *native[0].Seed != "123" || native[0].RetryCount == nil || *native[0].RetryCount != 0 || native[0].RetryOrdinal != nil {
		t.Fatalf("native final metadata changed after reopen: %+v", native)
	}
}

func TestRecordCannotReplaceOriginalOutcome(t *testing.T) {
	r, _ := newRepo(t, 10)
	ctx := context.Background()
	rec := sampleRun("same-id", time.Unix(1700000000, 0), "failed", 42)
	if err := r.Record(ctx, rec); err != nil {
		t.Fatal(err)
	}
	rec.Commands[0].Status = "passed"
	if err := r.Record(ctx, rec); err == nil {
		t.Fatal("duplicate run replaced original outcome")
	}
	history, err := r.CommandHistory(ctx, "demo", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || history[0].Status != "failed" {
		t.Fatalf("original outcome changed: %+v", history)
	}
}

func TestComparisonIdentityRequiresKnownCompatibleInputs(t *testing.T) {
	base := testIdentity(t, "source", "go1")
	if !base.Comparable(testIdentity(t, "source", "go1")) {
		t.Fatal("same known inputs must be comparable")
	}
	for _, tc := range []struct {
		name     string
		identity *ComparisonIdentity
	}{
		{"legacy", nil}, {"empty", &ComparisonIdentity{}},
		{"source changed", testIdentity(t, "changed", "go1")},
		{"toolchain changed", testIdentity(t, "source", "go2")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if base.Comparable(tc.identity) {
				t.Fatal("incompatible history joined cohort")
			}
		})
	}
	var absent *ComparisonIdentity
	if absent.Comparable(nil) {
		t.Fatal("two missing identities are not comparable")
	}
	for _, mutate := range []func(*ComparisonIdentity){
		func(i *ComparisonIdentity) { i.Selection = "go test -run TestOne" },
		func(i *ComparisonIdentity) { i.TestID = "TestOne" },
		func(i *ComparisonIdentity) { seed := "0"; i.Seed = &seed },
		func(i *ComparisonIdentity) { ordinal := 0; i.RetryOrdinal = &ordinal },
		func(i *ComparisonIdentity) { i.Evidence.Digest = "corrupt" },
	} {
		changed := testIdentity(t, "source", "go1")
		mutate(changed)
		if base.Comparable(changed) {
			t.Fatal("different execution identity joined cohort")
		}
		if base.CohortDigest() == changed.CohortDigest() {
			t.Fatal("different execution scopes share a cohort label")
		}
	}
}

func TestIdentityPersistencePreservesLegacyAndPrunesMetadata(t *testing.T) {
	r, db := newRepo(t, 2)
	ctx := context.Background()
	base := time.Unix(1700000000, 0)
	legacy := sampleRun("legacy", base, "failed", 12)
	if err := r.Record(ctx, legacy); err != nil {
		t.Fatal(err)
	}
	rec := sampleRun("known", base.Add(time.Minute), "passed", 10)
	rec.Commands[0].Identity = testIdentity(t, "source", "go1")
	if err := r.Record(ctx, rec); err != nil {
		t.Fatal(err)
	}
	// Reapplying the domain schema must neither synthesize old provenance nor
	// overwrite recorded outcomes.
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	history, err := NewRepository(db).CommandHistory(ctx, "demo", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 || !rec.Commands[0].Identity.Comparable(history[0].Identity) || history[1].Identity != nil || history[1].Status != "failed" {
		t.Fatalf("history lost provenance or changed legacy outcome: %+v", history)
	}
	// Corrupt optional metadata leaves the outcome readable but uncomparable.
	if _, err := db.Exec(`UPDATE unit_run_command_identity SET identity_json = '{broken'`); err != nil {
		t.Fatal(err)
	}
	history, err = r.CommandHistory(ctx, "demo", 10)
	if err != nil || history[0].Identity != nil || history[0].Status != "passed" {
		t.Fatalf("corrupt provenance: %+v, %v", history, err)
	}
	for n, id := range []string{"new1", "new2"} {
		if err := r.Record(ctx, sampleRun(id, base.Add(time.Duration(n+2)*time.Minute), "passed", 10)); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM unit_run_command_identity`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("orphaned identity rows: %d", count)
	}
}
