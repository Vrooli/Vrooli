package gates

// Calibration is the anti-fabrication boundary for catalog gates. A fixture is
// a small, real catalog asset with one deliberately planted defect. The gate
// is run against an isolated overlay containing that asset; a clean result is
// therefore a failed calibration, not evidence of quality.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func createSurfaceCalibrationDatabase(targetPath, assetID string) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}
	if err := removeGateDB(targetPath); err != nil {
		return err
	}
	db, err := openGateDB(context.Background(), targetPath)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.ExecContext(context.Background(), `CREATE TABLE reconcile_evidence (id TEXT PRIMARY KEY, scenario TEXT NOT NULL, document_kind TEXT NOT NULL, component_id TEXT NOT NULL, example_name TEXT NOT NULL, state_id TEXT NOT NULL, ax_node_json TEXT NOT NULL, measurement_json TEXT NOT NULL, checked_at TEXT NOT NULL)`); err != nil {
		return err
	}
	if _, err := db.ExecContext(context.Background(), `CREATE TABLE reconcile_evidence_viewports (evidence_id TEXT PRIMARY KEY, viewport_id TEXT NOT NULL, viewport_width INTEGER NOT NULL, viewport_height INTEGER NOT NULL)`); err != nil {
		return err
	}
	ax := fmt.Sprintf(`{"dom":{"attributes":{"data-rcl-asset":%q}},"computedStyle":{"box-shadow":"0 1px 2px rgba(9, 18, 22, 0.06), 0 1px 3px rgba(9, 18, 22, 0.1)"}}`, assetID)
	_, err = db.ExecContext(context.Background(), `INSERT INTO reconcile_evidence(id,scenario,document_kind,component_id,example_name,state_id,ax_node_json,measurement_json,checked_at) VALUES(?,?,?,?,?,?,?,?,?)`, "calibration-surface-hard-coded", "react-component-library", "component", assetID, "default", "default", ax, `{}`, "2026-01-01T00:00:00Z")
	return err
}

func copyFile(source, dest string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dest, data, 0o644)
}

func calibrationDirectoryName(assetID string) string {
	var b strings.Builder
	for _, r := range assetID {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	if b.Len() == 0 {
		return "CalibrationFixture"
	}
	return "Calibration-" + b.String()
}

// CalibrationFixtureNames is used by diagnostics and tests to make ownership
// visible without exposing the internal fixture loader.
func CalibrationFixtureNames(root string) ([]string, error) {
	base := filepath.Join(root, "scenarios", "react-component-library", "catalog", "calibration")
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}
