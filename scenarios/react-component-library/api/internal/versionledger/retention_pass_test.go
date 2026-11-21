package versionledger

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/databasetest"

	"react-component-library/internal/components"
)

const retentionPassSchema = `
CREATE TABLE components (id TEXT PRIMARY KEY, library_id TEXT NOT NULL, latest_version TEXT NOT NULL, draft_version TEXT NOT NULL, manifest_path TEXT NOT NULL DEFAULT '');
CREATE TABLE component_versions (id TEXT PRIMARY KEY, component_id TEXT NOT NULL, library_id TEXT NOT NULL, version TEXT NOT NULL, status TEXT NOT NULL, source_path TEXT NOT NULL, created_at TEXT NOT NULL DEFAULT '', released_at TEXT NOT NULL DEFAULT '', presence TEXT NOT NULL DEFAULT 'materialized');
CREATE TABLE component_version_files (version_id TEXT NOT NULL, path TEXT NOT NULL, content TEXT NOT NULL, content_sha256 TEXT NOT NULL);
CREATE TABLE version_ledger (library_id TEXT, version TEXT, lifecycle_state TEXT);
CREATE TABLE adoption_records (id TEXT, component_id TEXT, library_id TEXT, adopted_version TEXT, mode TEXT NOT NULL DEFAULT 'copied', scenario TEXT NOT NULL DEFAULT '', adopted_path TEXT NOT NULL DEFAULT '');
CREATE TABLE adoption_files (adoption_id TEXT, source_library_id TEXT, source_version TEXT, adopted_path TEXT NOT NULL DEFAULT '');
CREATE TABLE component_asset_dependencies (component_id TEXT, library_id TEXT, version TEXT);
`

// retentionFixture is a repository-shaped tree: the library lives at
// scenarios/react-component-library/library so consumer scenarios, templates
// and harnesses sit where the real ones do.
type retentionFixture struct {
	t            *testing.T
	db           *sql.DB
	repoRoot     string
	scenarioRoot string
	libraryRoot  string
	latest       map[string]string
}

func newRetentionFixture(t *testing.T) *retentionFixture {
	t.Helper()
	repoRoot := t.TempDir()
	scenarioRoot := filepath.Join(repoRoot, "scenarios", "react-component-library")
	libraryRoot := filepath.Join(scenarioRoot, "library")
	require.NoError(t, os.MkdirAll(libraryRoot, 0o755))
	database := databasetest.NewSQLite(t)
	_, err := database.ExecContext(context.Background(), retentionPassSchema)
	require.NoError(t, err)
	return &retentionFixture{t: t, db: database, repoRoot: repoRoot, scenarioRoot: scenarioRoot, libraryRoot: libraryRoot, latest: map[string]string{}}
}

type pin struct{ asset, version string }

// release writes one materialized release with a schema-v2 lock — the shape
// the lock generator emits — and mirrors its files the way the indexer does.
func (f *retentionFixture) release(asset, version, status string, pins ...pin) {
	f.t.Helper()
	libraryID := "react-component-library:" + asset
	dir := filepath.Join(f.libraryRoot, "components", asset, "versions", version)
	require.NoError(f.t, os.MkdirAll(dir, 0o755))
	source := fmt.Sprintf("export const %s = %q;\n", asset, version)
	dependencies := make([]map[string]any, 0, len(pins))
	for _, p := range pins {
		dependencies = append(dependencies, map[string]any{"libraryId": "react-component-library:" + p.asset, "major": 1, "observed": p.version, "rank": 4})
	}
	lock, err := json.Marshal(map[string]any{"schemaVersion": 2, "libraryId": libraryID, "version": version, "resolvedAt": "2026-11-07T00:00:00Z", "dependencies": dependencies})
	require.NoError(f.t, err)
	files := map[string]string{asset + ".tsx": source, "dependencies.json": string(lock)}
	for name, body := range files {
		require.NoError(f.t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644))
	}
	manifest := filepath.Join(f.libraryRoot, "components", asset, "component.json")
	if _, statErr := os.Stat(manifest); os.IsNotExist(statErr) {
		require.NoError(f.t, os.WriteFile(manifest, []byte(`{"libraryId":"`+libraryID+`"}`), 0o644))
	}
	ctx := context.Background()
	versionID := asset + "@" + version
	_, err = f.db.ExecContext(ctx, `INSERT INTO component_versions(id, component_id, library_id, version, status, source_path, created_at, released_at) VALUES (?, ?, ?, ?, ?, ?, '2025-01-01T00:00:00Z', '2025-01-02T00:00:00Z')`,
		versionID, asset, libraryID, version, status, "components/"+asset+"/versions/"+version+"/"+asset+".tsx")
	require.NoError(f.t, err)
	for name, body := range files {
		_, err = f.db.ExecContext(ctx, `INSERT INTO component_version_files(version_id, path, content, content_sha256) VALUES (?, ?, ?, ?)`, versionID, name, body, fmt.Sprintf("%x", sha256.Sum256([]byte(body))))
		require.NoError(f.t, err)
	}
	_, err = f.db.ExecContext(ctx, `INSERT INTO version_ledger(library_id, version, lifecycle_state) VALUES (?, ?, 'released')`, libraryID, version)
	require.NoError(f.t, err)
	if compareRelease(version, f.latest[asset]) > 0 {
		f.latest[asset] = version
	}
}

func (f *retentionFixture) publish() {
	f.t.Helper()
	for asset, latest := range f.latest {
		_, err := f.db.ExecContext(context.Background(), `INSERT INTO components(id, library_id, latest_version, draft_version, manifest_path) VALUES (?, ?, ?, '', ?)`,
			asset, "react-component-library:"+asset, latest, "components/"+asset+"/component.json")
		require.NoError(f.t, err)
	}
}

func (f *retentionFixture) write(relative, body string) {
	f.t.Helper()
	path := filepath.Join(f.repoRoot, filepath.FromSlash(relative))
	require.NoError(f.t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(f.t, os.WriteFile(path, []byte(body), 0o644))
}

func (f *retentionFixture) presence(asset, version string) string {
	f.t.Helper()
	var presence string
	require.NoError(f.t, f.db.QueryRowContext(context.Background(), `SELECT presence FROM component_versions WHERE component_id=? AND version=?`, asset, version).Scan(&presence))
	return presence
}

func (f *retentionFixture) onDisk(asset, version string) bool {
	_, err := os.Stat(filepath.Join(f.libraryRoot, "components", asset, "versions", version))
	return err == nil
}

// ledgerCatalog serves the reconciler's catalog reads straight from the
// fixture database, so each round sees the presence the previous one wrote.
type ledgerCatalog struct {
	components.Service
	db *sql.DB
}

func (c ledgerCatalog) List(ctx context.Context, _ components.SearchQuery) ([]components.Component, error) {
	rows, err := c.db.QueryContext(ctx, `SELECT id, library_id FROM components ORDER BY library_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []components.Component
	for rows.Next() {
		var item components.Component
		if err := rows.Scan(&item.ID, &item.LibraryID); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (c ledgerCatalog) ListVersions(ctx context.Context, componentID string, _ int) ([]components.ComponentVersion, error) {
	rows, err := c.db.QueryContext(ctx, `SELECT version, status, presence FROM component_versions WHERE component_id=? ORDER BY version`, componentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []components.ComponentVersion
	for rows.Next() {
		item := components.ComponentVersion{ComponentID: componentID}
		var status string
		if err := rows.Scan(&item.Version, &status, &item.Presence); err != nil {
			return nil, err
		}
		item.Status = components.ComponentVersionStatus(status)
		out = append(out, item)
	}
	return out, rows.Err()
}

// A retention pass must never evict a release that something still pins
// exactly. Every leaf below is superseded, unadopted and outside every
// latest/draft root; the only thing keeping each one is a single exact pin
// from one of the sources retention has to honour. The orphan and the
// historical chain prove the pass still reclaims what nothing needs.
func TestRetentionPassKeepsExactlyPinnedLeavesMaterialized(t *testing.T) {
	f := newRetentionFixture(t)

	// Pinned by the schema-v2 lock of a live (latest) release.
	f.release("LockLeaf", "1.0.0", "released")
	f.release("LockLeaf", "1.1.0", "released")
	f.release("Parent", "1.0.0", "released", pin{"LockLeaf", "1.0.0"})

	// Pinned by the lock of a retained release that is not itself reachable:
	// OldOwner@1.0.0 is materialized, so what it pins must stay with it until
	// it leaves.
	f.release("RetainedLeaf", "1.0.0", "released")
	f.release("RetainedLeaf", "1.1.0", "released")
	f.release("OldOwner", "1.0.0", "released", pin{"RetainedLeaf", "1.0.0"})
	f.release("OldOwner", "1.1.0", "released")

	// Pinned by a preview harness lock, which belongs to no catalog version.
	f.release("HarnessLeaf", "1.0.0", "released")
	f.release("HarnessLeaf", "1.1.0", "released")
	f.write("scenarios/react-component-library/harnesses/navigation/versions/1.0.0/dependencies.json",
		`{"schemaVersion":2,"libraryId":"preview.navigation","version":"1.0.0","resolvedAt":"2026-11-07T00:00:00Z","dependencies":[{"libraryId":"react-component-library:HarnessLeaf","major":1,"observed":"1.0.0"}]}`)

	// Imported by consumer scenarios with no adoption record: one exact pin,
	// and one major selector on a line that is no longer latest.
	f.release("ExactLeaf", "1.0.0", "released")
	f.release("ExactLeaf", "1.1.0", "released")
	f.write("scenarios/consumer/ui/src/App.tsx", `import { ExactLeaf } from "@vrooli/react-component-library/ExactLeaf/1.0.0";`)
	f.release("MajorLeaf", "0.1.5", "released")
	f.release("MajorLeaf", "0.1.6", "released")
	f.release("MajorLeaf", "1.0.0", "released")
	f.write("templates/scenarios/starter/ui/src/Settings.tsx", `const list = await import("@vrooli/react-component-library/MajorLeaf/0");`)

	// Pinned only from a scenario's top-level docs, which hold evidence, not
	// consumers.
	f.release("Orphan", "1.0.0", "released")
	f.release("Orphan", "1.1.0", "released")
	f.write("scenarios/consumer/docs/history.ts", `import "@vrooli/react-component-library/Orphan/1.0.0";`)

	// A release's own test folder belongs to that release. It does not pin
	// the release, and its imports hold only while the release is on disk:
	// TestedOld@1.0.0 and the leaf its test imports both leave, while the
	// latest LiveOwner's test keeps its leaf.
	f.release("TestedOld", "1.0.0", "released")
	f.release("TestedOld", "1.1.0", "released")
	f.release("TestedLeaf", "1.0.0", "released")
	f.release("TestedLeaf", "1.1.0", "released")
	f.write("scenarios/react-component-library/library/tests/components/TestedOld/1.0.0/TestedOld.test.tsx",
		`import "@vrooli/react-component-library/TestedOld/1.0.0"; import "@vrooli/react-component-library/TestedLeaf/1.0.0";`)
	f.release("LiveOwner", "2.0.0", "released")
	f.release("LiveLeaf", "1.0.0", "released")
	f.release("LiveLeaf", "1.1.0", "released")
	f.write("scenarios/react-component-library/library/tests/components/LiveOwner/2.0.0/LiveOwner.test.tsx",
		`import "@vrooli/react-component-library/LiveLeaf/1.0.0";`)
	f.publish()

	repository := NewRepository(f.db, f.libraryRoot)
	reconciler := NewPresenceReconciler(repository, ledgerCatalog{db: f.db}, nil)
	require.NoError(t, reconciler.ReconcilePresence(context.Background(), "", true))

	for _, kept := range []pin{{"LockLeaf", "1.0.0"}, {"HarnessLeaf", "1.0.0"}, {"ExactLeaf", "1.0.0"}, {"MajorLeaf", "0.1.6"}, {"LiveLeaf", "1.0.0"}} {
		require.Equal(t, "materialized", f.presence(kept.asset, kept.version), "%s@%s is pinned exactly and must survive retention", kept.asset, kept.version)
		require.True(t, f.onDisk(kept.asset, kept.version), "%s@%s must stay on disk", kept.asset, kept.version)
	}
	for _, evicted := range []pin{{"Orphan", "1.0.0"}, {"MajorLeaf", "0.1.5"}, {"OldOwner", "1.0.0"}, {"RetainedLeaf", "1.0.0"}, {"TestedOld", "1.0.0"}, {"TestedLeaf", "1.0.0"}} {
		require.Equal(t, "evicted", f.presence(evicted.asset, evicted.version), "%s@%s is pinned by nothing live and should be reclaimed", evicted.asset, evicted.version)
		require.False(t, f.onDisk(evicted.asset, evicted.version))
	}
}

// The retained-owner rule, observed one round at a time: while OldOwner@1.0.0
// is on disk its exact pin keeps RetainedLeaf@1.0.0 out of the eviction set,
// and once the owner is cold the pin is an ordinary edge again.
func TestRetainedReleasePinsAreLiveUntilTheOwnerIsEvicted(t *testing.T) {
	f := newRetentionFixture(t)
	f.release("RetainedLeaf", "1.0.0", "released")
	f.release("RetainedLeaf", "1.1.0", "released")
	f.release("OldOwner", "1.0.0", "released", pin{"RetainedLeaf", "1.0.0"})
	f.release("OldOwner", "1.1.0", "released")
	f.publish()
	repository := NewRepository(f.db, f.libraryRoot)
	ctx := context.Background()

	candidates, err := repository.RetireCandidates(ctx, "")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"react-component-library:OldOwner@1.0.0"}, candidateKeys(candidates))

	_, hash, err := repository.PlanCleanup(ctx, CleanupScope{ComponentID: "OldOwner"})
	require.NoError(t, err)
	_, err = repository.Transition(ctx, "OldOwner", "1.0.0", "archived", true, hash)
	require.NoError(t, err)

	// The cold owner stays a retirement candidate; its pin no longer holds
	// the leaf.
	candidates, err = repository.RetireCandidates(ctx, "")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"react-component-library:OldOwner@1.0.0", "react-component-library:RetainedLeaf@1.0.0"}, candidateKeys(candidates))
}

func candidateKeys(candidates []Candidate) []string {
	keys := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		keys = append(keys, candidate.LibraryID+"@"+candidate.Version)
	}
	return keys
}

// A v2 lock entry that names a dependency without a release cannot be
// honoured and must not silently drop the edge.
func TestLockEntryWithoutAReleaseFailsClosed(t *testing.T) {
	root := t.TempDir()
	writeVersion(t, root, "components", "Card", "1.0.0", "export const Card = () => null;",
		`{"schemaVersion":2,"libraryId":"react-component-library:Card","version":"1.0.0","dependencies":[{"libraryId":"react-component-library:Icon","major":1}]}`)
	_, err := NewRepository(nil, root).lockReferences(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "without a release")
}

func TestPackageSelectorGrammar(t *testing.T) {
	for _, tc := range []struct{ specifier, libraryID, selector string }{
		{"@vrooli/react-component-library/Input", "react-component-library:Input", ""},
		{"@vrooli/react-component-library/Input/1", "react-component-library:Input", "1"},
		{"@vrooli/react-component-library/Input/1.1.2", "react-component-library:Input", "1.1.2"},
		{"@vrooli/react-component-library/Input/1/1.1.2", "react-component-library:Input", "1.1.2"},
		{"@vrooli/react-component-library/Input/1.1.2/styles.css", "react-component-library:Input", "1.1.2"},
	} {
		libraryID, selector, ok := packageSelector(tc.specifier)
		require.True(t, ok, tc.specifier)
		require.Equal(t, tc.libraryID, libraryID, tc.specifier)
		require.Equal(t, tc.selector, selector, tc.specifier)
	}
	for _, invalid := range []string{"react", "@vrooli/react-component-library/", "@vrooli/react-component-library/Input/next"} {
		_, _, ok := packageSelector(invalid)
		require.False(t, ok, invalid)
	}
}
