package gates

// Calibration is the anti-fabrication boundary for catalog gates. A fixture is
// a small, real catalog asset with one deliberately planted defect. The gate
// is run against an isolated overlay containing that asset; a clean result is
// therefore a failed calibration, not evidence of quality.

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/api-core/storage"
)

func createSurfaceCalibrationDatabase(targetPath, assetID string) error {
	if err := calibrationMkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
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
	if err := createCalibrationEvidenceSchema(context.Background(), db, true); err != nil {
		return err
	}
	ax := fmt.Sprintf(`{"dom":{"attributes":{"data-rcl-asset":%q}},"computedStyle":{"box-shadow":"0 1px 2px rgba(9, 18, 22, 0.06), 0 1px 3px rgba(9, 18, 22, 0.1)"}}`, assetID)
	_, err = db.ExecContext(context.Background(), `INSERT INTO reconcile_evidence(id,scenario,document_kind,component_id,example_name,state_id,ax_node_json,measurement_json,checked_at) VALUES(?,?,?,?,?,?,?,?,?)`, "calibration-surface-hard-coded", "react-component-library", "component", assetID, "default", "default", ax, `{}`, "2026-01-01T00:00:00Z")
	return err
}

func calibrationMkdirAll(path string, mode os.FileMode) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	rootPath := absolute
	for {
		if _, err := os.Stat(rootPath); err == nil {
			break
		} else if !os.IsNotExist(err) {
			return err
		}
		parent := filepath.Dir(rootPath)
		if parent == rootPath {
			return fmt.Errorf("no existing ancestor for %q", absolute)
		}
		rootPath = parent
	}
	relative, err := filepath.Rel(rootPath, absolute)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return err
	}
	defer root.Close()
	return root.MkdirAll(relative, mode)
}

func copyFile(source, dest string) error {
	sourceRoot, err := os.OpenRoot(filepath.Dir(source))
	if err != nil {
		return err
	}
	defer sourceRoot.Close()
	input, err := sourceRoot.Open(filepath.Base(source))
	if err != nil {
		return err
	}
	defer input.Close()
	data, err := io.ReadAll(input)
	if err != nil {
		return err
	}
	destRoot, err := os.OpenRoot(filepath.Dir(dest))
	if err != nil {
		return err
	}
	defer destRoot.Close()
	return storage.WriteFileAtomicInRoot(destRoot, filepath.Base(dest), data, 0o644)
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
	rootFS, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer rootFS.Close()
	directory, err := rootFS.Open(filepath.Join("scenarios", "react-component-library", "catalog", "calibration"))
	if err != nil {
		return nil, err
	}
	defer directory.Close()
	entries, err := directory.ReadDir(-1)
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
