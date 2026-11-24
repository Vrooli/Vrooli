package gates

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func openCalibrationTestDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	routed, err := openGateDB(context.Background(), path)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = routed.Close() })
	return routed.Primary()
}

func assertTableColumns(t *testing.T, db *sql.DB, table string, want []string) {
	t.Helper()
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		t.Fatalf("inspect %s columns: %v", table, err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var cid, notnull, pk int
		var name, kind string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &kind, &notnull, &defaultValue, &pk); err != nil {
			t.Fatalf("scan %s columns: %v", table, err)
		}
		got = append(got, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read %s columns: %v", table, err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s columns = %v, want %v", table, got, want)
	}
}

func TestCalibrationDatabaseBuildersCreateExpectedEvidenceSchemasAndRows(t *testing.T) {
	t.Run("surface", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "nested", "surface.db")
		if err := createSurfaceCalibrationDatabase(path, "Button"); err != nil {
			t.Fatal(err)
		}
		db := openCalibrationTestDB(t, path)
		assertCalibrationEvidenceSchema(t, db, true)
		var id, scenario, kind, component, example, state, ax, measurement, checked string
		err := db.QueryRow(`SELECT id,scenario,document_kind,component_id,example_name,state_id,ax_node_json,measurement_json,checked_at FROM reconcile_evidence`).Scan(&id, &scenario, &kind, &component, &example, &state, &ax, &measurement, &checked)
		if err != nil {
			t.Fatal(err)
		}
		if id != "calibration-surface-hard-coded" || scenario != "react-component-library" || kind != "component" || component != "Button" || example != "default" || state != "default" || measurement != `{}` || checked != "2026-01-01T00:00:00Z" {
			t.Fatalf("unexpected surface evidence row: %q %q %q %q %q %q %q", id, scenario, kind, component, example, state, measurement)
		}
		var decoded map[string]any
		if err := json.Unmarshal([]byte(ax), &decoded); err != nil {
			t.Fatalf("surface AX evidence is invalid JSON: %v", err)
		}
		if decoded["dom"].(map[string]any)["attributes"].(map[string]any)["data-rcl-asset"] != "Button" {
			t.Fatalf("surface evidence asset mismatch: %s", ax)
		}
	})

	t.Run("performance", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "reports.db")
		if err := createPerformanceBudgetDatabase(path, "Button"); err != nil {
			t.Fatal(err)
		}
		db := openCalibrationTestDB(t, path)
		assertTableColumns(t, db, "component_test_reports", []string{"id", "results_json", "created_at"})
		var id, raw, created string
		if err := db.QueryRow(`SELECT id,results_json,created_at FROM component_test_reports`).Scan(&id, &raw, &created); err != nil {
			t.Fatal(err)
		}
		if id != "calibration-performance-budget" || created != "2026-01-01T00:00:00Z" {
			t.Fatalf("unexpected performance report identity: %q at %q", id, created)
		}
		var payload struct {
			Results []struct {
				Stage, AssetLibraryID, Subject string
				Evidence                       []struct {
					Kind        string
					Performance struct{ MountMs float64 }
				}
			}
		}
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			t.Fatalf("decode report: %v", err)
		}
		if len(payload.Results) != 1 || payload.Results[0].Stage != "declared_behavior" || payload.Results[0].AssetLibraryID != calibrationDirectoryName("Button") || payload.Results[0].Subject != "slow-story" || len(payload.Results[0].Evidence) != 1 || payload.Results[0].Evidence[0].Kind != "performance" || payload.Results[0].Evidence[0].Performance.MountMs != 10 {
			t.Fatalf("unexpected performance report payload: %s", raw)
		}
	})

	for _, gate := range []string{"rtl", "reduced-motion", "composition"} {
		t.Run(gate, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "nested", "evidence.db")
			var err error
			switch gate {
			case "composition":
				err = createCompositionCalibrationDatabase(path, "Button")
			default:
				err = createDifferentialCalibrationDatabase(path, gate, "Button")
			}
			if err != nil {
				t.Fatal(err)
			}
			db := openCalibrationTestDB(t, path)
			assertCalibrationEvidenceSchema(t, db, gate != "composition")
			var count int
			if err := db.QueryRow(`SELECT count(*) FROM reconcile_evidence`).Scan(&count); err != nil {
				t.Fatal(err)
			}
			want := 1
			if gate != "composition" {
				want = 2
			}
			if count != want {
				t.Fatalf("evidence row count = %d, want %d", count, want)
			}
			if gate == "composition" {
				var id, ax string
				if err := db.QueryRow(`SELECT id,ax_node_json FROM reconcile_evidence`).Scan(&id, &ax); err != nil {
					t.Fatal(err)
				}
				var tree map[string]any
				if err := json.Unmarshal([]byte(ax), &tree); err != nil {
					t.Fatalf("decode composition tree: %v", err)
				}
				if id != "calibration-composition" || len(tree["children"].([]any)) != 40 {
					t.Fatalf("composition fixture = %q with unexpected tree", id)
				}
				return
			}
			var viewportCount int
			if err := db.QueryRow(`SELECT count(*) FROM reconcile_evidence_viewports`).Scan(&viewportCount); err != nil {
				t.Fatal(err)
			}
			if viewportCount != 2 {
				t.Fatalf("viewport count = %d, want 2", viewportCount)
			}
			var contexts string
			if err := db.QueryRow(`SELECT group_concat(measurement_json, ' ') FROM reconcile_evidence`).Scan(&contexts); err != nil {
				t.Fatal(err)
			}
			if gate == "rtl" && (!containsAll(contexts, `"direction":"ltr"`, `"direction":"rtl"`)) {
				t.Fatalf("RTL contexts missing: %s", contexts)
			}
			if gate == "reduced-motion" && (!containsAll(contexts, `"motionPreference":"no-preference"`, `"motionPreference":"reduce"`)) {
				t.Fatalf("motion contexts missing: %s", contexts)
			}
		})
	}
}

func assertCalibrationEvidenceSchema(t *testing.T, db *sql.DB, withViewports bool) {
	t.Helper()
	assertTableColumns(t, db, "reconcile_evidence", []string{"id", "scenario", "document_kind", "component_id", "example_name", "state_id", "ax_node_json", "measurement_json", "checked_at"})
	if withViewports {
		assertTableColumns(t, db, "reconcile_evidence_viewports", []string{"evidence_id", "viewport_id", "viewport_width", "viewport_height"})
	} else {
		var count int
		if err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name='reconcile_evidence_viewports'`).Scan(&count); err != nil || count != 0 {
			t.Fatalf("composition should omit viewport table (count=%d, err=%v)", count, err)
		}
	}
}

func containsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}

func TestCalibrationHelpersNamesCopyAndErrors(t *testing.T) {
	if got := calibrationDirectoryName("Button-Card.v2"); got != "Calibration-Button-Card-v2" {
		t.Errorf("directory name = %q", got)
	}
	if got := calibrationDirectoryName(""); got != "CalibrationFixture" {
		t.Errorf("empty directory name = %q", got)
	}

	root := t.TempDir()
	base := filepath.Join(root, "scenarios", "react-component-library", "catalog", "calibration")
	for _, name := range []string{"zeta", "alpha", "file"} {
		path := filepath.Join(base, name)
		if name == "file" {
			if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
		} else if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	got, err := CalibrationFixtureNames(root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"alpha", "zeta"}) {
		t.Fatalf("fixture names = %v", got)
	}
	if _, err := CalibrationFixtureNames(t.TempDir()); err == nil {
		t.Fatal("missing calibration directory should return an error")
	}

	source := filepath.Join(root, "input", "asset.json")
	dest := filepath.Join(root, "deep", "nested", "copy.json")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatal(err)
	}
	want := []byte(`{"fixture":true}`)
	if err := os.WriteFile(source, want, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(source, dest); err != nil {
		t.Fatal(err)
	}
	gotBytes, err := os.ReadFile(dest)
	if err != nil || string(gotBytes) != string(want) {
		t.Fatalf("copy content = %q, err = %v", gotBytes, err)
	}
	if err := copyFile(filepath.Join(root, "missing"), filepath.Join(root, "absent", "copy")); err == nil {
		t.Fatal("missing source should return an error")
	}
}

func TestCopyFileUsesRootedAtomicDestination(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.txt")
	dest := filepath.Join(root, "dest.txt")
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(source, []byte("copied"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, dest); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(source, dest); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dest)
	if err != nil || string(got) != "copied" {
		t.Fatalf("destination = %q, err = %v", got, err)
	}
	got, err = os.ReadFile(outside)
	if err != nil || string(got) != "keep" {
		t.Fatalf("outside target = %q, err = %v", got, err)
	}
	escape := filepath.Join(root, "escape")
	if err := os.Symlink(outside, escape); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(escape, filepath.Join(root, "blocked.txt")); err == nil {
		t.Fatal("copy followed source symlink outside its root")
	}
}

func TestCalibrationDirectoryCopyIsRootedAndPreservesOutputMode(t *testing.T) {
	base := t.TempDir()
	source := filepath.Join(base, "source")
	dest := filepath.Join(base, "overlay", "harnesses")
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.MkdirAll(filepath.Join(source, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "nested", "fixture.txt"), []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(source, "escape.txt")); err != nil {
		t.Fatal(err)
	}
	if err := copyDirectoryFiles(source, dest); err == nil {
		t.Fatal("directory copy followed source symlink outside its root")
	}
	if got, err := os.ReadFile(outside); err != nil || string(got) != "keep" {
		t.Fatalf("outside source = %q, err = %v", got, err)
	}
	if err := os.Remove(filepath.Join(source, "escape.txt")); err != nil {
		t.Fatal(err)
	}
	if err := copyDirectoryFiles(source, dest); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dest, "nested", "fixture.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("copied mode = %o, want 644", info.Mode().Perm())
	}
}

func TestDataOverlayCopyDoesNotFollowDestinationSymlink(t *testing.T) {
	base := t.TempDir()
	source := filepath.Join(base, "source")
	dest := filepath.Join(base, "overlay", "data")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "notes.txt"), []byte("note"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dest, "notes.txt")); err != nil {
		t.Fatal(err)
	}
	if err := copyDataOverlay(source, dest); err == nil {
		t.Fatal("data overlay replaced a pre-existing destination symlink")
	}
	if got, err := os.ReadFile(outside); err != nil || string(got) != "keep" {
		t.Fatalf("outside destination = %q, err = %v", got, err)
	}
}

func TestCalibrationDatabaseBuildersReplaceDatabaseAndPropagatePathErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "replace.db")
	old := openCalibrationTestDB(t, path)
	if _, err := old.ExecContext(context.Background(), `CREATE TABLE stale (id INTEGER)`); err != nil {
		t.Fatal(err)
	}
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+"-wal", []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+"-shm", []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := createSurfaceCalibrationDatabase(path, "Button"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + "-wal"); !os.IsNotExist(err) {
		t.Fatalf("WAL sidecar remains: %v", err)
	}
	if _, err := os.Stat(path + "-shm"); !os.IsNotExist(err) {
		t.Fatalf("SHM sidecar remains: %v", err)
	}
	db := openCalibrationTestDB(t, path)
	var stale int
	if err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name='stale'`).Scan(&stale); err != nil {
		t.Fatal(err)
	}
	if stale != 0 {
		t.Fatal("old database schema survived replacement")
	}

	blockedParent := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blockedParent, []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := createSurfaceCalibrationDatabase(filepath.Join(blockedParent, "database.db"), "Button"); err == nil {
		t.Fatal("database builder should propagate parent directory creation errors")
	}
}
