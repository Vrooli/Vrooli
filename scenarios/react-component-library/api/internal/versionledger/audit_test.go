package versionledger

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"react-component-library/internal/components"
)

func writeAuditLedger(t *testing.T, root string, entries []map[string]string) {
	t.Helper()
	path := filepath.Join(root, "scenarios/react-component-library/library/released-version-hashes.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]any{"schemaVersion": 1, "entries": entries})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func auditEntry(path string, body []byte) map[string]string {
	sum := sha256.Sum256(body)
	return map[string]string{"path": path, "sha256": hex.EncodeToString(sum[:])}
}

func seedAuditVersion(t *testing.T, db *sql.DB, version, status, presence string, mirror []byte) {
	t.Helper()
	id := "v" + version
	path := "components/Button/versions/" + version + "/Button.tsx"
	_, err := db.Exec(`INSERT INTO component_versions (id, component_id, library_id, version, status, source_path, content, content_sha256, indexed_at, presence) VALUES (?, 'button', 'react-component-library:Button', ?, ?, ?, '', '', '2026-01-01T00:00:00Z', ?)`, id, version, status, path, presence)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO version_ledger (library_id, version, lifecycle_state) VALUES ('react-component-library:Button', ?, ?)`, version, status)
	if err != nil {
		t.Fatal(err)
	}
	if mirror != nil {
		sum := sha256.Sum256(mirror)
		_, err = db.Exec(`INSERT INTO component_version_files (version_id, path, content, content_sha256, is_entry) VALUES (?, 'Button.tsx', ?, ?, 1)`, id, string(mirror), hex.EncodeToString(sum[:]))
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestAuditReleaseHashLedgerWithDatabaseUsesOnlyEvictedMirror(t *testing.T) {
	db := archiveDatabase(t)
	root := t.TempDir()
	matching, mutated := []byte("mirror-current"), []byte("mirror-mutated")
	seedAuditVersion(t, db, "1.0.0", "archived", "evicted", matching)
	seedAuditVersion(t, db, "1.0.1", "released", "evicted", mutated)
	seedAuditVersion(t, db, "1.0.2", "released", "materialized", matching)
	entries := []map[string]string{
		auditEntry("components/Button/versions/1.0.0/Button.tsx", matching),
		auditEntry("components/Button/versions/1.0.1/Button.tsx", matching),
		auditEntry("components/Button/versions/1.0.2/Button.tsx", matching),
		auditEntry("components/Button/versions/1.0.3/Button.tsx", matching),
	}
	writeAuditLedger(t, root, entries)
	audit, err := AuditReleaseHashLedgerWithDatabase(root, db)
	if err != nil {
		t.Fatal(err)
	}
	if len(audit.Matching) != 1 || len(audit.Mutated) != 1 || len(audit.Missing) != 2 {
		t.Fatalf("unexpected database audit: %+v", audit)
	}
}

func TestAuditReleaseHashLedgerUsesRequestDatabaseRoute(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	routed, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(dir, "primary.db")})
	require.NoError(t, err)
	t.Cleanup(func() { _ = routed.Close() })
	for _, schema := range []string{components.Schema(), Schema()} {
		_, err = routed.ExecContext(ctx, schema)
		require.NoError(t, err)
	}
	routed.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		for _, schema := range []string{components.Schema(), Schema()} {
			if _, err := pool.ExecContext(ctx, schema); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, routed.InstallTestPool(ctx, filepath.Join(dir, "test.db"), "audit-test", 0))

	requestCtx := database.WithTestMode(ctx)
	_, err = routed.ExecContext(requestCtx, `INSERT INTO component_versions (id, component_id, library_id, version, status, source_path, content, content_sha256, indexed_at, presence) VALUES ('v1.0.0', 'button', 'react-component-library:Button', '1.0.0', 'released', 'components/Button/versions/1.0.0/Button.tsx', '', '', '2026-01-01T00:00:00Z', 'evicted')`)
	require.NoError(t, err)
	_, err = routed.ExecContext(requestCtx, `INSERT INTO component_version_files (version_id, path, content, content_sha256, is_entry) VALUES ('v1.0.0', 'Button.tsx', 'mirror', '', 1)`)
	require.NoError(t, err)
	root := t.TempDir()
	writeAuditLedger(t, root, []map[string]string{auditEntry("components/Button/versions/1.0.0/Button.tsx", []byte("mirror"))})
	audit, err := AuditReleaseHashLedgerWithDatabaseContext(requestCtx, root, routed)
	require.NoError(t, err)
	require.Len(t, audit.Matching, 1)
	require.Empty(t, audit.Missing)
}

func TestRepairReleaseHashLedgerRequiresOptInAndMirrorEvidence(t *testing.T) {
	db := archiveDatabase(t)
	root := t.TempDir()
	base := []byte("original")
	current := []byte("reviewed current")
	mirror := []byte("durable mirror")
	// 1.0.0 is safely rerecorded from the mirror; 1.0.1 needs explicit provenance;
	// 1.0.2 is missing and remains unresolved until prune is authorized.
	seedAuditVersion(t, db, "1.0.0", "released", "materialized", mirror)
	seedAuditVersion(t, db, "1.0.1", "released", "materialized", base)
	seedAuditVersion(t, db, "1.0.2", "released", "materialized", base)
	entries := []map[string]string{
		auditEntry("components/Button/versions/1.0.0/Button.tsx", base),
		auditEntry("components/Button/versions/1.0.1/Button.tsx", base),
		auditEntry("components/Button/versions/1.0.2/Button.tsx", base),
		auditEntry("components/Button/versions/9.9.9/Button.tsx", base),
	}
	for version, body := range map[string][]byte{"1.0.0": mirror, "1.0.1": current} {
		path := filepath.Join(root, "scenarios/react-component-library/library/components/Button/versions", version, "Button.tsx")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	writeAuditLedger(t, root, entries)
	preview, err := RepairReleaseHashLedger(root, db, false)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Applied || len(preview.Rerecorded) != 1 || len(preview.Unresolved) != 2 || len(preview.Pruned) != 1 {
		t.Fatalf("unexpected safe preview: %+v", preview)
	}
	accepted, err := RepairReleaseHashLedgerWithOptions(root, db, true, LedgerRepairOptions{AcceptCurrent: true, PruneMissing: true})
	if err != nil {
		t.Fatal(err)
	}
	if !accepted.Applied || len(accepted.Rerecorded) != 1 || len(accepted.AcceptedCurrent) != 1 || len(accepted.Pruned) != 2 || len(accepted.Unresolved) != 0 {
		t.Fatalf("unexpected authorized repair: %+v", accepted)
	}
	var ledger struct {
		Entries []struct{ Path, SHA256 string } `json:"entries"`
	}
	path := filepath.Join(root, "scenarios/react-component-library/library/released-version-hashes.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &ledger); err != nil {
		t.Fatal(err)
	}
	if len(ledger.Entries) != 2 {
		t.Fatalf("expected derived and retained records only, got %+v", ledger.Entries)
	}
}

func TestAuditReleaseHashLedgerClassifiesAuthoredFilesAndIgnoresLocks(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "scenarios/react-component-library/library/components/Button/versions/1.0.0")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	matching := []byte("matching\n")
	mutated := []byte("mutated\n")
	if err := os.WriteFile(filepath.Join(dir, "Button.tsx"), matching, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "story.tsx"), mutated, 0o600); err != nil {
		t.Fatal(err)
	}
	hash := func(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
	ledger := map[string]any{"schemaVersion": 1, "entries": []map[string]string{
		{"path": "components/Button/versions/1.0.0/Button.tsx", "sha256": hash(matching)},
		{"path": "components/Button/versions/1.0.0/story.tsx", "sha256": hash([]byte("old\n"))},
		{"path": "components/Button/versions/1.0.0/removed.tsx", "sha256": hash([]byte("gone\n"))},
		{"path": "components/Button/versions/1.0.0/dependencies.json", "sha256": hash([]byte("derived\n"))},
	}}
	raw, _ := json.Marshal(ledger)
	if err := os.WriteFile(filepath.Join(root, "scenarios/react-component-library/library/released-version-hashes.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	audit, err := AuditReleaseHashLedger(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(audit.Matching) != 1 || len(audit.Mutated) != 1 || len(audit.Missing) != 1 {
		t.Fatalf("unexpected audit: %+v", audit)
	}
	if audit.Mutated[0].Asset != "Button@1.0.0" || audit.Missing[0].Asset != "Button@1.0.0" {
		t.Fatalf("asset attribution lost: %+v", audit)
	}
}

func TestAuditReleaseHashLedgerRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "Button.tsx")
	body := []byte("outside source must not be read\n")
	require.NoError(t, os.WriteFile(outside, body, 0o600))
	versionDir := filepath.Join(root, "scenarios/react-component-library/library/components/Button/versions/1.0.0")
	require.NoError(t, os.MkdirAll(versionDir, 0o755))
	if err := os.Symlink(outside, filepath.Join(versionDir, "Button.tsx")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	writeAuditLedger(t, root, []map[string]string{auditEntry("components/Button/versions/1.0.0/Button.tsx", body)})

	audit, err := AuditReleaseHashLedger(root)
	require.NoError(t, err)
	require.Empty(t, audit.Matching)
	require.Empty(t, audit.Mutated)
	require.Len(t, audit.Missing, 1)
	actual, err := os.ReadFile(outside)
	require.NoError(t, err)
	require.Equal(t, body, actual)
}
