package gates

// Calibration is the anti-fabrication boundary for catalog gates. A fixture is
// a small, real catalog asset with one deliberately planted defect. The gate
// is run against an isolated overlay containing that asset; a clean result is
// therefore a failed calibration, not evidence of quality.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
)

func createPerformanceBudgetDatabase(targetPath, assetID string) error {
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
			"subject":        "slow-story",
			"evidence": []map[string]any{{
				"kind": "performance",
				"performance": map[string]any{
					"mountMs": 10.0,
				},
			}},
		}},
	}
	results, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(context.Background(), `INSERT INTO component_test_reports(id, results_json, created_at) VALUES (?, ?, ?)`, "calibration-performance-budget", string(results), "2026-01-01T00:00:00Z")
	return err
}

func createDifferentialCalibrationDatabase(targetPath, gate, assetID string) error {
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
	contexts := []struct{ id, context, ax string }{}
	if gate == "rtl" {
		contexts = append(contexts,
			struct{ id, context, ax string }{"ltr", `{"captureContext":{"locale":"en","direction":"ltr","motionPreference":"no-preference"}}`, `{"bounds":{"x":10,"y":10,"width":20,"height":20},"dom":{"attributes":{"data-rcl-asset":"` + assetID + `"}}}`},
			struct{ id, context, ax string }{"rtl", `{"captureContext":{"locale":"ar","direction":"rtl","motionPreference":"no-preference"}}`, `{"bounds":{"x":10,"y":10,"width":20,"height":20},"dom":{"attributes":{"data-rcl-asset":"` + assetID + `"}}}`},
		)
	} else {
		contexts = append(contexts,
			struct{ id, context, ax string }{"full", `{"captureContext":{"locale":"en","direction":"ltr","motionPreference":"no-preference"}}`, `{"bounds":{"x":10,"y":10,"width":20,"height":20},"dom":{"attributes":{"data-rcl-asset":"` + assetID + `"}},"computedStyle":{"transitionDuration":"200ms"}}`},
			struct{ id, context, ax string }{"reduce", `{"captureContext":{"locale":"en","direction":"ltr","motionPreference":"reduce"}}`, `{"bounds":{"x":10,"y":10,"width":20,"height":20},"dom":{"attributes":{"data-rcl-asset":"` + assetID + `"}},"computedStyle":{"transitionDuration":"200ms"}}`},
		)
	}
	for _, item := range contexts {
		id := "calibration-" + gate + "-" + item.id
		if _, err := db.ExecContext(context.Background(), `INSERT INTO reconcile_evidence(id,scenario,document_kind,component_id,example_name,state_id,ax_node_json,measurement_json,checked_at) VALUES(?,?,?,?,?,?,?,?,?)`, id, "react-component-library", "component", assetID, "default", "default", item.ax, item.context, "2026-01-01T00:00:00Z"); err != nil {
			return err
		}
		if _, err := db.ExecContext(context.Background(), `INSERT INTO reconcile_evidence_viewports(evidence_id,viewport_id,viewport_width,viewport_height) VALUES(?,?,?,?)`, id, "desktop", 100, 100); err != nil {
			return err
		}
	}
	return nil
}

func createCompositionCalibrationDatabase(targetPath, assetID string) error {
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
	children := make([]map[string]any, 0, 40)
	for index := 0; index < 40; index++ {
		children = append(children, map[string]any{"dom": map[string]any{"tag": "div", "attributes": map[string]string{}}})
	}
	tree, err := json.Marshal(map[string]any{
		"dom":      map[string]any{"tag": "section", "attributes": map[string]string{"data-rcl-asset": assetID}},
		"children": children,
	})
	if err != nil {
		return err
	}
	_, err = db.ExecContext(context.Background(), `INSERT INTO reconcile_evidence(id,scenario,document_kind,component_id,example_name,state_id,ax_node_json,measurement_json,checked_at) VALUES(?,?,?,?,?,?,?,?,?)`, "calibration-composition", "react-component-library", "component", assetID, "default", "default", string(tree), `{}`, "2026-01-01T00:00:00Z")
	return err
}
