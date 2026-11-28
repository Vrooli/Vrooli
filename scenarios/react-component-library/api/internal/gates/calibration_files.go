package gates

// Calibration is the anti-fabrication boundary for catalog gates. A fixture is
// a small, real catalog asset with one deliberately planted defect. The gate
// is run against an isolated overlay containing that asset; a clean result is
// therefore a failed calibration, not evidence of quality.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/storage"
)

func copyDirectoryFiles(source, dest string) error {
	if err := calibrationMkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	sourceRoot, err := os.OpenRoot(source)
	if err != nil {
		return err
	}
	defer sourceRoot.Close()
	destRoot, err := os.OpenRoot(filepath.Dir(dest))
	if err != nil {
		return err
	}
	defer destRoot.Close()
	destPath := filepath.Base(dest)
	if err := destRoot.MkdirAll(destPath, 0o755); err != nil {
		return err
	}
	return copyDirectoryFilesInRoots(sourceRoot, destRoot, ".", destPath)
}

func copyDirectoryFilesInRoots(sourceRoot, destRoot *os.Root, sourcePath, destPath string) error {
	sourceDir, err := sourceRoot.Open(sourcePath)
	if err != nil {
		return err
	}
	entries, err := sourceDir.ReadDir(-1)
	closeErr := sourceDir.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	for _, entry := range entries {
		src := filepath.Join(sourcePath, entry.Name())
		dst := filepath.Join(destPath, entry.Name())
		if entry.IsDir() {
			if err := destRoot.MkdirAll(dst, 0o755); err != nil {
				return err
			}
			if err := copyDirectoryFilesInRoots(sourceRoot, destRoot, src, dst); err != nil {
				return err
			}
		} else {
			input, err := sourceRoot.Open(src)
			if err != nil {
				return err
			}
			data, readErr := io.ReadAll(input)
			closeErr := input.Close()
			if readErr != nil {
				return readErr
			}
			if closeErr != nil {
				return closeErr
			}
			if err := storage.WriteFileAtomicInRoot(destRoot, dst, data, 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyDataOverlay(source, dest string) error {
	if err := calibrationMkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	sourceRoot, err := os.OpenRoot(source)
	if err != nil {
		return err
	}
	defer sourceRoot.Close()
	destRoot, err := os.OpenRoot(filepath.Dir(dest))
	if err != nil {
		return err
	}
	defer destRoot.Close()
	destPath := filepath.Base(dest)
	if err := destRoot.MkdirAll(destPath, 0o755); err != nil {
		return err
	}
	sourceDir, err := sourceRoot.Open(".")
	if err != nil {
		return err
	}
	entries, err := sourceDir.ReadDir(-1)
	closeErr := sourceDir.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	for _, entry := range entries {
		src := filepath.Join(source, entry.Name())
		dst := filepath.Join(destPath, entry.Name())
		if entry.Name() == "react-component-library.db" {
			input, err := sourceRoot.Open(entry.Name())
			if err != nil {
				return err
			}
			data, readErr := io.ReadAll(input)
			closeErr := input.Close()
			if readErr != nil {
				return readErr
			}
			if closeErr != nil {
				return closeErr
			}
			if err := storage.WriteFileAtomicInRoot(destRoot, dst, data, 0o644); err != nil {
				return err
			}
		} else if err := destRoot.Symlink(src, dst); err != nil {
			return err
		}
	}
	return nil
}

func createCalibrationDatabase(targetPath string, fixture CalibrationFixture) error {
	if err := removeGateDB(targetPath); err != nil {
		return err
	}
	target, err := openGateDB(context.Background(), targetPath)
	if err != nil {
		return err
	}
	defer target.Close()
	if _, err := target.ExecContext(context.Background(), `CREATE TABLE component_versions (id TEXT PRIMARY KEY, component_id TEXT NOT NULL, library_id TEXT NOT NULL, version TEXT NOT NULL, status TEXT NOT NULL, source_path TEXT NOT NULL, content TEXT NOT NULL DEFAULT '', content_sha256 TEXT NOT NULL, changelog_md TEXT NOT NULL DEFAULT '', indexed_at TEXT NOT NULL, released_at TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL DEFAULT '', UNIQUE(component_id, version))`); err != nil {
		return err
	}
	// Evidence-freshness reads the report table, not the component index. Keep
	// the calibration database explicit about that seam so a missing report is
	// observed as the planted defect instead of being mistaken for a broken
	// runner with zero inspected inputs.
	if _, err := target.ExecContext(context.Background(), `CREATE TABLE component_test_reports (id TEXT PRIMARY KEY, root_library_id TEXT NOT NULL, root_version TEXT NOT NULL, results_json TEXT NOT NULL DEFAULT '{}', created_at TEXT NOT NULL)`); err != nil {
		return err
	}
	name := calibrationDirectoryName(fixture.AssetID)
	sourcePath := filepath.ToSlash(filepath.Join("components", name, "versions", "1.0.0", name+filepath.Ext(fixture.Source)))
	contentPath := filepath.Join(filepath.Dir(filepath.Dir(targetPath)), "library", sourcePath)
	contentRoot, err := os.OpenRoot(filepath.Dir(contentPath))
	if err != nil {
		return err
	}
	defer contentRoot.Close()
	content, err := contentRoot.ReadFile(filepath.Base(contentPath))
	if err != nil {
		return fmt.Errorf("read calibration source: %w", err)
	}
	_, err = target.ExecContext(context.Background(), `INSERT INTO component_versions(id, component_id, library_id, version, status, source_path, content, content_sha256, changelog_md, indexed_at, released_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, "calibration-version", "calibration-component", name, "1.0.0", "released", sourcePath, string(content), strings.Repeat("0", 64), "", "2026-01-15T12:00:00Z", "2026-01-15T12:00:00Z", "2026-01-15T12:00:00Z")
	return err
}

func createConsoleWarningDatabase(targetPath, assetID string) error {
	if err := removeGateDB(targetPath); err != nil {
		return err
	}
	db, err := openGateDB(context.Background(), targetPath)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.ExecContext(context.Background(), `CREATE TABLE component_test_reports (id TEXT PRIMARY KEY, results_json TEXT NOT NULL, created_at TEXT NOT NULL)`); err != nil {
		return err
	}
	payload := map[string]any{
		"results": []map[string]any{{
			"stage":          "declared_behavior",
			"assetLibraryId": calibrationDirectoryName(assetID),
			"subject":        "warning-story",
			"evidence": []map[string]any{{
				"kind": "console",
				"console": map[string]any{
					"consoleErrors": []string{"Warning: Each child in a list should have a unique key prop."},
				},
			}},
		}},
	}
	results, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(context.Background(), `INSERT INTO component_test_reports(id, results_json, created_at) VALUES (?, ?, ?)`, "calibration-console-warning", string(results), "2026-01-01T00:00:00Z")
	return err
}
