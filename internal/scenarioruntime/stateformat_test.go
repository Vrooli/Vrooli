package scenarioruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/stateformat"
)

func TestStateFormatDescribesTheCurrentSchema(t *testing.T) {
	store := StateFormat()
	if store.Name != "runtime-registry" {
		t.Fatalf("store name = %q", store.Name)
	}
	if store.Write != stateformat.Version(SchemaVersion) {
		t.Fatalf("write = %d, want %d", store.Write, SchemaVersion)
	}
	if _, ok := store.Schemas[store.Write]; !ok {
		t.Fatalf("schemas has no digest for the write version %d", store.Write)
	}
	if err := stateformat.SortStores([]stateformat.Store{store}); err != nil {
		t.Fatalf("state format is invalid: %v", err)
	}
}

// TestSchemaHistoryIsMinimalAndCurrent enforces the greenfield rule: a store
// keeps the format it writes and at most the one it migrates from. A third
// snapshot means a dead branch is being carried.
func TestSchemaHistoryIsMinimalAndCurrent(t *testing.T) {
	digests := registrySchemaDigests()
	if _, ok := digests[SchemaVersion]; !ok {
		t.Fatalf("schema history has no v%d snapshot", SchemaVersion)
	}
	for version := range digests {
		if version != SchemaVersion && version != SchemaVersion-1 {
			t.Fatalf("schema history contains v%d; keep only v%d and v%d", version, SchemaVersion-1, SchemaVersion)
		}
	}
	if got, wantDigest := digests[SchemaVersion], liveSchemaDigest(); got != wantDigest {
		t.Fatalf("v%d snapshot digest = %s, live schema digest = %s; regenerate the snapshot", SchemaVersion, got, wantDigest)
	}
}

// TestRuntimeRegistryAdditiveFromFixture proves the additive claim against the
// previous format's reader. It writes data with the current schema and then
// runs the column list a v16 reader used. An additive claim that this test
// cannot exercise must stay zero.
func TestRuntimeRegistryAdditiveFromFixture(t *testing.T) {
	if additiveFromRuntimeRegistry == 0 {
		t.Skip("the runtime registry makes no additive claim")
	}
	if additiveFromRuntimeRegistry != SchemaVersion-1 {
		t.Fatalf("additive_from = %d but the only frozen prior reader is v%d; re-earn the claim for the new version", additiveFromRuntimeRegistry, SchemaVersion-1)
	}
	store := openStateFormatFixture(t)
	ctx := t.Context()
	created, err := store.CreateInstance(ctx, Instance{InstanceID: "fixture", Scenario: "demo", Phase: "api"})
	if err != nil {
		t.Fatalf("write current-format row: %v", err)
	}
	// The v16 reader's column list. It must keep working against v17 data, and
	// the values it reads must be unchanged.
	var (
		instanceID, scenario, build, variant, status string
		generation                                   int64
		startedAt, updatedAt                         string
	)
	if err := store.db.QueryRowContext(ctx, `
SELECT instance_id, scenario, build_identity, variant, generation, status, started_at, updated_at
FROM runtime_instances WHERE instance_id = 'fixture'`).Scan(
		&instanceID, &scenario, &build, &variant, &generation, &status, &startedAt, &updatedAt,
	); err != nil {
		t.Fatalf("v16 reader cannot read v17 data, so the additive claim is false: %v", err)
	}
	if instanceID != created.InstanceID || scenario != created.Scenario || variant != created.Variant || generation != created.Generation || status != string(created.Status) || build != created.BuildIdentity {
		t.Fatalf("v16 reader observed rewritten values: %q %q %q %q %d %q %q", instanceID, scenario, build, variant, generation, status, startedAt)
	}
}

// TestUpdateSchemaHistorySnapshot regenerates the write version's frozen
// snapshot. Run it only when SchemaVersion changes:
//
//	UPDATE_SCHEMA_HISTORY=1 go test ./internal/scenarioruntime -run TestUpdateSchemaHistorySnapshot
func TestUpdateSchemaHistorySnapshot(t *testing.T) {
	if os.Getenv("UPDATE_SCHEMA_HISTORY") == "" {
		t.Skip("set UPDATE_SCHEMA_HISTORY=1 to rewrite the current schema snapshot")
	}
	dir := "schema_history"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		version, err := parseSchemaSnapshotName(entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		if version != SchemaVersion && version != SchemaVersion-1 {
			t.Fatalf("refusing to rewrite: %s is neither the current nor the previous format", entry.Name())
		}
		if version >= SchemaVersion {
			if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
				t.Fatal(err)
			}
		}
	}
	content := canonicalDDL(schemaSQL) + "\n"
	if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("v%d.sql", SchemaVersion)), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCanonicalDDLIsLayoutInsensitive(t *testing.T) {
	a := canonicalDDL("CREATE TABLE t (\n  id TEXT\n);\n")
	b := canonicalDDL("CREATE   TABLE t ( id TEXT );")
	if a != b {
		t.Fatalf("canonical DDL depends on layout:\n%s\n%s", a, b)
	}
	if canonicalDDL("CREATE TABLE t (id TEXT);") == canonicalDDL("CREATE TABLE t (id INTEGER);") {
		t.Fatal("canonical DDL ignored a declaration change")
	}
}

func TestSchemaSnapshotNamesAreParsed(t *testing.T) {
	if version, err := parseSchemaSnapshotName("v17.sql"); err != nil || version != 17 {
		t.Fatalf("v17.sql: version=%d err=%v", version, err)
	}
	for _, bad := range []string{"17.sql", "vx.sql", "v0.sql", "v17.txt"} {
		if _, err := parseSchemaSnapshotName(bad); err == nil {
			t.Fatalf("%q must be rejected", bad)
		}
	}
}

func TestDigestSchemaTextIgnoresLayout(t *testing.T) {
	first := sha256.Sum256([]byte(canonicalDDL("a  b")))
	second := sha256.Sum256([]byte(canonicalDDL("a b")))
	if hex.EncodeToString(first[:]) != hex.EncodeToString(second[:]) || digestSchemaText("a  b") != digestSchemaText("a\tb") {
		t.Fatal("schema digest depends on layout")
	}
	_ = strings.TrimSpace
}

// openStateFormatFixture opens a fresh runtime registry so fixtures can inspect
// the exact schema this binary creates.
func openStateFormatFixture(t *testing.T) *SQLiteStore {
	t.Helper()
	store, err := NewSQLiteStore(context.Background(), Config{DBPath: filepath.Join(t.TempDir(), "runtime.db")})
	if err != nil {
		t.Fatalf("open runtime registry: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}
