package gates

// Calibration is the anti-fabrication boundary for catalog gates. A fixture is
// a small, real catalog asset with one deliberately planted defect. The gate
// is run against an isolated overlay containing that asset; a clean result is
// therefore a failed calibration, not evidence of quality.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type CalibrationFixture struct {
	Gate                string `json:"gate"`
	AssetID             string `json:"assetId"`
	Kind                string `json:"kind"`
	Runner              string `json:"runner"`
	RequiredFailureCode string `json:"requiredFailureCode"`
	Description         string `json:"description"`
	Source              string `json:"source"`
	CatalogAsset        string `json:"catalogAsset"`
	Mutation            string `json:"mutation"`
}

type CalibrationResult struct {
	Gate                string
	Fixture             string
	RequiredFailureCode string
	ObservedFailureCode string
	Status              string
	Message             string
}

type CalibrationReport struct {
	Results           []CalibrationResult
	NonDiscriminating bool
}

type GateRunner func(string) (Result, error)

// GateRunnerFor returns the production runner for a declared gate. A nil
// runner is intentional: external gates remain visible as quarantined until
// their owning browser/toolchain runner is wired into this process.
func GateRunnerFor(gate string) GateRunner {
	switch gate {
	case "types":
		return ValidateTypes
	case "api":
		return ValidateAPI
	case "tokens":
		return ValidateTokens
	case "conformance":
		return ValidateConformance
	case "token-vocabulary":
		return ValidateTokenVocabulary
	case "token-ramp-complete":
		return ValidateTokenRampComplete
	case "released-version-immutable":
		return ValidateReleasedVersionImmutable
	case "lifecycle":
		return ValidateLifecycle
	case "fixture-adversarial":
		return ValidateFixtures
	case "examples":
		return ValidateExamples
	case "graph-reconciled":
		return ValidateGraphReconciled
	case "rtl":
		return ValidateRTL
	case "reduced-motion":
		return ValidateReducedMotion
	case "documentation":
		return ValidateDocumentation
	case "performance":
		return ValidatePerformance
	case "console-clean":
		return ValidateConsoleClean
	case "surface-discipline":
		return ValidateSurfaceDiscipline
	case "composition":
		return ValidateComposition
	case "composition-contract":
		return ValidateCompositionContract
	default:
		return nil
	}
}

// Calibrate evaluates every fixture owned by gate. A missing fixture is a
// quarantine condition for a blocking gate, not an empty pass.
func Calibrate(root, gate string, runner GateRunner) (CalibrationReport, error) {
	fixtures, err := loadCalibrationFixtures(root, gate)
	if err != nil {
		return CalibrationReport{}, err
	}
	if len(fixtures) == 0 {
		return CalibrationReport{
			Results:           []CalibrationResult{{Gate: gate, Fixture: "", Status: "missing-fixture", Message: "blocking gate owns no calibration fixture"}},
			NonDiscriminating: true,
		}, nil
	}

	report := CalibrationReport{Results: make([]CalibrationResult, 0, len(fixtures))}
	for _, fixture := range fixtures {
		result := CalibrationResult{
			Gate:                gate,
			Fixture:             filepath.ToSlash(filepath.Join("catalog", "calibration", gate, filepath.Base(fixturePath(root, gate, fixture)))),
			RequiredFailureCode: fixture.RequiredFailureCode,
		}
		if fixture.Runner != "static" {
			result.Status = "non-discriminating"
			result.Message = "the declared runner is external to the deterministic catalog gate process"
			report.NonDiscriminating = true
			report.Results = append(report.Results, result)
			continue
		}
		overlay, cleanup, materializeErr := materializeFixture(root, gate, fixture)
		if materializeErr != nil {
			return CalibrationReport{}, materializeErr
		}
		observed, runErr := runner(overlay)
		cleanup()
		if runErr != nil {
			result.Status = "runner-error"
			result.Message = runErr.Error()
			report.NonDiscriminating = true
			report.Results = append(report.Results, result)
			continue
		}
		for _, finding := range observed.Findings {
			if finding.Code != fixture.RequiredFailureCode {
				continue
			}
			if fixture.AssetID == "" || fixture.Mutation != "" || finding.AssetID == fixture.AssetID || finding.AssetID == "" || strings.Contains(strings.ToLower(finding.AssetID), strings.ToLower(fixture.AssetID)) {
				result.ObservedFailureCode = finding.Code
				break
			}
		}
		if result.ObservedFailureCode != "" {
			result.Status = "failed"
			result.Message = "fixture defect was detected"
		} else {
			result.Status = "non-discriminating"
			observedCodes := make([]string, 0, len(observed.Findings)+len(observed.RunnerError))
			for _, finding := range append(observed.Findings, observed.RunnerError...) {
				if finding.Code != "" {
					observedCodes = append(observedCodes, finding.Code)
				}
			}
			result.Message = fmt.Sprintf("fixture completed without its required failure code; observed %v", observedCodes)
			report.NonDiscriminating = true
		}
		report.Results = append(report.Results, result)
	}
	return report, nil
}

func loadCalibrationFixtures(root, gate string) ([]CalibrationFixture, error) {
	dir := filepath.Join(root, "scenarios", "react-component-library", "catalog", "calibration", gate)
	paths, err := filepath.Glob(filepath.Join(dir, "fixture.json"))
	if err != nil {
		return nil, err
	}
	var fixtures []CalibrationFixture
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, readErr
		}
		var fixture CalibrationFixture
		if err := json.Unmarshal(data, &fixture); err != nil {
			return nil, fmt.Errorf("parse calibration fixture %s: %w", path, err)
		}
		if fixture.Gate == "" {
			fixture.Gate = gate
		}
		if fixture.Gate != gate {
			return nil, fmt.Errorf("calibration fixture %s declares gate %q, want %q", path, fixture.Gate, gate)
		}
		if fixture.RequiredFailureCode == "" {
			return nil, fmt.Errorf("calibration fixture %s has no requiredFailureCode", path)
		}
		fixtures = append(fixtures, fixture)
	}
	return fixtures, nil
}

func fixturePath(root, gate string, fixture CalibrationFixture) string {
	return filepath.Join(root, "scenarios", "react-component-library", "catalog", "calibration", gate, "fixture.json")
}

func materializeFixture(root, gate string, fixture CalibrationFixture) (string, func(), error) {
	tmp, err := os.MkdirTemp("", "rcl-catalog-calibration-")
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() { _ = os.RemoveAll(tmp) }
	if err := linkScenarioOverlay(root, tmp); err != nil {
		cleanup()
		return "", func() {}, err
	}
	// Calibration must exercise the real runner, but it does not need to pay
	// the cost of rechecking unrelated corpus assets. Keeping the planted asset
	// as the only library entry also prevents an unrelated existing finding from
	// satisfying the fixture's required code by accident.
	if fixture.Mutation != "released-hash" {
		if err := pruneLibraryOverlay(tmp); err != nil {
			cleanup()
			return "", func() {}, err
		}
	}
	fixtureDir := filepath.Join(root, "scenarios", "react-component-library", "catalog", "calibration", gate)
	assetDir := filepath.Join(tmp, "scenarios", "react-component-library", "catalog", "assets", "calibration")
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		cleanup()
		return "", func() {}, err
	}
	if fixture.CatalogAsset != "" {
		if err := copyFile(filepath.Join(fixtureDir, fixture.CatalogAsset), filepath.Join(assetDir, fixture.AssetID+".json")); err != nil {
			cleanup()
			return "", func() {}, err
		}
	}
	if fixture.Source != "" {
		name := calibrationDirectoryName(fixture.AssetID)
		componentDir := filepath.Join(tmp, "scenarios", "react-component-library", "library", "components", name)
		versionDir := filepath.Join(componentDir, "versions", "1.0.0")
		if err := os.MkdirAll(versionDir, 0o755); err != nil {
			cleanup()
			return "", func() {}, err
		}
		manifest := fmt.Sprintf("{\"libraryId\":\"%s\",\"catalogId\":\"%s\",\"latest\":\"1.0.0\"}\n", name, fixture.AssetID)
		if err := os.WriteFile(filepath.Join(componentDir, "component.json"), []byte(manifest), 0o644); err != nil {
			cleanup()
			return "", func() {}, err
		}
		ext := filepath.Ext(fixture.Source)
		if ext == "" {
			ext = ".tsx"
		}
		if err := copyFile(filepath.Join(fixtureDir, fixture.Source), filepath.Join(versionDir, name+ext)); err != nil {
			cleanup()
			return "", func() {}, err
		}
	}
	if fixture.Mutation == "released-hash" {
		dbPath := filepath.Join(tmp, "scenarios", "react-component-library", "data", "react-component-library.db")
		if err := createCalibrationDatabase(filepath.Join(root, "scenarios", "react-component-library", "data", "react-component-library.db"), dbPath); err != nil {
			cleanup()
			return "", func() {}, err
		}
	}
	if fixture.Mutation == "console-warning" {
		dbPath := filepath.Join(tmp, "scenarios", "react-component-library", "data", "react-component-library.db")
		if err := createConsoleWarningDatabase(dbPath, fixture.AssetID); err != nil {
			cleanup()
			return "", func() {}, err
		}
	}
	if fixture.Mutation == "performance-budget" {
		dbPath := filepath.Join(tmp, "scenarios", "react-component-library", "data", "react-component-library.db")
		if err := createPerformanceBudgetDatabase(dbPath, fixture.AssetID); err != nil {
			cleanup()
			return "", func() {}, err
		}
	}
	if fixture.Mutation == "differential-hard-coded" {
		dbPath := filepath.Join(tmp, "scenarios", "experience-manager", "data", "experience-manager.db")
		if err := createDifferentialCalibrationDatabase(dbPath, fixture.Gate, fixture.AssetID); err != nil {
			cleanup()
			return "", func() {}, err
		}
	}
	if fixture.Mutation == "composition-low-score" {
		dbPath := filepath.Join(tmp, "scenarios", "experience-manager", "data", "experience-manager.db")
		if err := createCompositionCalibrationDatabase(dbPath, fixture.AssetID); err != nil {
			cleanup()
			return "", func() {}, err
		}
	}
	return tmp, cleanup, nil
}

func pruneLibraryOverlay(root string) error {
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		dir := filepath.Join(root, "scenarios", "react-component-library", "library", kind)
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

func linkScenarioOverlay(root, tmp string) error {
	realScenario := filepath.Join(root, "scenarios", "react-component-library")
	tmpScenario := filepath.Join(tmp, "scenarios", "react-component-library")
	for _, path := range []string{"ui", "catalog/config.json", "catalog/weights.json", "templates", "data"} {
		source := filepath.Join(realScenario, path)
		if _, err := os.Stat(source); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		dest := filepath.Join(tmpScenario, path)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if path == "ui" {
			if err := copyUIOverlay(source, dest); err != nil {
				return err
			}
			continue
		}
		if path == "data" {
			if err := copyDataOverlay(source, dest); err != nil {
				return err
			}
			continue
		}
		if err := os.Symlink(source, dest); err != nil {
			return err
		}
	}
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		destKind := filepath.Join(tmpScenario, "library", kind)
		if err := os.MkdirAll(destKind, 0o755); err != nil {
			return err
		}
		entries, err := os.ReadDir(filepath.Join(realScenario, "library", kind))
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := os.Symlink(filepath.Join(realScenario, "library", kind, entry.Name()), filepath.Join(destKind, entry.Name())); err != nil {
				return err
			}
		}
	}
	assetsRoot := filepath.Join(tmpScenario, "catalog", "assets")
	if err := os.MkdirAll(assetsRoot, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(filepath.Join(realScenario, "catalog", "assets"))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := os.Symlink(filepath.Join(realScenario, "catalog", "assets", entry.Name()), filepath.Join(assetsRoot, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func copyUIOverlay(source, dest string) error {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		src := filepath.Join(source, entry.Name())
		dst := filepath.Join(dest, entry.Name())
		if !entry.IsDir() {
			if err := os.WriteFile(dst, nil, 0o644); err != nil {
				return err
			}
			continue
		}
		if entry.Name() == "scripts" {
			if err := copyDirectoryFiles(src, dst); err != nil {
				return err
			}
			continue
		}
		if err := os.Symlink(src, dst); err != nil {
			return err
		}
	}
	return nil
}

func copyDirectoryFiles(source, dest string) error {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		src := filepath.Join(source, entry.Name())
		dst := filepath.Join(dest, entry.Name())
		if entry.IsDir() {
			if err := copyDirectoryFiles(src, dst); err != nil {
				return err
			}
		} else if err := copyFile(src, dst); err != nil {
			return err
		}
	}
	return nil
}

func copyDataOverlay(source, dest string) error {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		src := filepath.Join(source, entry.Name())
		dst := filepath.Join(dest, entry.Name())
		if entry.Name() == "react-component-library.db" {
			if err := copyFile(src, dst); err != nil {
				return err
			}
		} else if err := os.Symlink(src, dst); err != nil {
			return err
		}
	}
	return nil
}

func createCalibrationDatabase(sourcePath, targetPath string) error {
	source, err := openGateDB(context.Background(), sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()
	var row struct {
		id, componentID, libraryID, version, status, sourcePath, content, changelog, indexedAt, releasedAt, createdAt string
	}
	err = source.QueryRowContext(context.Background(), `SELECT id, component_id, library_id, version, status, source_path, content, changelog_md, indexed_at, released_at, created_at FROM component_versions WHERE status = 'released' LIMIT 1`).Scan(&row.id, &row.componentID, &row.libraryID, &row.version, &row.status, &row.sourcePath, &row.content, &row.changelog, &row.indexedAt, &row.releasedAt, &row.createdAt)
	if err != nil {
		return err
	}
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
	_, err = target.ExecContext(context.Background(), `INSERT INTO component_versions(id, component_id, library_id, version, status, source_path, content, content_sha256, changelog_md, indexed_at, released_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, row.id, row.componentID, row.libraryID, row.version, row.status, row.sourcePath, row.content, strings.Repeat("0", 64), row.changelog, row.indexedAt, row.releasedAt, row.createdAt)
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
