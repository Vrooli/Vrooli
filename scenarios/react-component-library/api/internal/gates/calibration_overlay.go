package gates

// Calibration is the anti-fabrication boundary for catalog gates. A fixture is
// a small, real catalog asset with one deliberately planted defect. The gate
// is run against an isolated overlay containing that asset; a clean result is
// therefore a failed calibration, not evidence of quality.

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

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
		if gate == "token-ramp-complete" || gate == "fallback-parity" || gate == "kit-compatibility" || gate == "affinity-compatible" {
			baseStyles := filepath.Join(tmp, "scenarios", "react-component-library", "library", "foundations", "BaseStyles")
			if err := os.Symlink(filepath.Join(root, "scenarios", "react-component-library", "library", "foundations", "BaseStyles"), baseStyles); err != nil {
				cleanup()
				return "", func() {}, err
			}
			designTemplates := filepath.Join(tmp, "templates", "design")
			if err := os.MkdirAll(filepath.Dir(designTemplates), 0o755); err != nil {
				cleanup()
				return "", func() {}, err
			}
			if err := os.Symlink(filepath.Join(root, "templates", "design"), designTemplates); err != nil {
				cleanup()
				return "", func() {}, err
			}
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
		if fixture.Story != "" {
			if err := copyFile(filepath.Join(fixtureDir, fixture.Story), filepath.Join(versionDir, "story.json")); err != nil {
				cleanup()
				return "", func() {}, err
			}
		}
		// New-only gates must see the synthetic version as newly authored even
		// when the repository policy cutoff is in the future relative to the
		// test fixture's creation time.
		if err := os.Chtimes(versionDir, time.Now(), time.Now()); err != nil {
			cleanup()
			return "", func() {}, err
		}
		if fixture.Mutation == "affinity-incompatible" {
			manifest = fmt.Sprintf("{\"libraryId\":\"%s\",\"catalogId\":\"%s\",\"latest\":\"1.0.0\",\"designStyles\":[{\"styleId\":\"vrooli-default\",\"affinity\":\"native\"}]}\n", name, fixture.AssetID)
			if err := os.WriteFile(filepath.Join(componentDir, "component.json"), []byte(manifest), 0o644); err != nil {
				cleanup()
				return "", func() {}, err
			}
		}
		if fixture.Mutation == "field-ownership" {
			manifest = fmt.Sprintf("{\"libraryId\":\"%s\",\"catalogId\":\"%s\",\"latest\":\"1.0.0\",\"description\":\"duplicated\"}\n", name, fixture.AssetID)
			if err := os.WriteFile(filepath.Join(componentDir, "component.json"), []byte(manifest), 0o644); err != nil {
				cleanup()
				return "", func() {}, err
			}
		}
	}
	if fixture.Mutation == "deprecated-import" {
		controlDir := filepath.Join(tmp, "scenarios", "react-component-library", "library", "components", "ControlBase", "versions", "1.0.0")
		if err := os.MkdirAll(controlDir, 0o755); err != nil {
			cleanup()
			return "", func() {}, err
		}
		manifest := []byte(`{"libraryId":"react-component-library:ControlBase","catalogId":"react-component-library:ControlBase","latest":"1.0.1","deprecatedVersions":["1.0.0"]}` + "\n")
		if err := os.WriteFile(filepath.Join(tmp, "scenarios", "react-component-library", "library", "components", "ControlBase", "component.json"), manifest, 0o644); err != nil {
			cleanup()
			return "", func() {}, err
		}
		if err := os.WriteFile(filepath.Join(controlDir, "ControlBase.tsx"), []byte("export const ControlBase = () => null;\n"), 0o644); err != nil {
			cleanup()
			return "", func() {}, err
		}
	}
	if fixture.Mutation == "consumer-pin" {
		consumerPath := filepath.Join(tmp, "scenarios", "calibration-adopter", "ui", "src", "Consumer.tsx")
		if err := os.MkdirAll(filepath.Dir(consumerPath), 0o755); err != nil {
			cleanup()
			return "", func() {}, err
		}
		name := calibrationDirectoryName(fixture.AssetID)
		source := fmt.Sprintf("import { %s } from \"@vrooli/react-component-library/%s/9.9.9\";\nexport const Consumer = %s;\n", name, name, name)
		if err := os.WriteFile(consumerPath, []byte(source), 0o644); err != nil {
			cleanup()
			return "", func() {}, err
		}
	}
	if fixture.Mutation == "scenario-token-requirements" {
		tokenPath := filepath.Join(tmp, "templates", "design", "_base", "tokens.css")
		if err := os.MkdirAll(filepath.Dir(tokenPath), 0o755); err != nil {
			cleanup()
			return "", func() {}, err
		}
		if err := os.WriteFile(tokenPath, []byte(":root {\n  /* @tier Expression */\n  --color-calibration: blue;\n}\n"), 0o644); err != nil {
			cleanup()
			return "", func() {}, err
		}
		assetDir := filepath.Join(tmp, "scenarios", "react-component-library", "library", "components", "CalibrationScenarioToken")
		versionDir := filepath.Join(assetDir, "versions", "1.0.0")
		if err := os.MkdirAll(versionDir, 0o755); err != nil {
			cleanup()
			return "", func() {}, err
		}
		if err := os.WriteFile(filepath.Join(assetDir, "component.json"), []byte(`{"libraryId":"react-component-library:CalibrationScenarioToken"}`), 0o644); err != nil {
			cleanup()
			return "", func() {}, err
		}
		if err := os.WriteFile(filepath.Join(versionDir, "CalibrationScenarioToken.tsx"), []byte(`export const CalibrationScenarioToken = () => <div style={{color: "var(--color-calibration)"}} />;`), 0o644); err != nil {
			cleanup()
			return "", func() {}, err
		}
		consumerDir := filepath.Join(tmp, "scenarios", "calibration-adopter", "ui", "src")
		if err := os.MkdirAll(consumerDir, 0o755); err != nil {
			cleanup()
			return "", func() {}, err
		}
		if err := os.WriteFile(filepath.Join(consumerDir, "App.tsx"), []byte(`import { CalibrationScenarioToken } from "@vrooli/react-component-library/CalibrationScenarioToken";`), 0o644); err != nil {
			cleanup()
			return "", func() {}, err
		}
		if err := os.WriteFile(filepath.Join(consumerDir, "design-tokens.css"), []byte(":root {\n/* rcl:tokens:begin */\n/* rcl:tokens:end */\n}\n"), 0o644); err != nil {
			cleanup()
			return "", func() {}, err
		}
	}
	if fixture.Mutation == "dependency-rank" {
		library := filepath.Join(tmp, "scenarios", "react-component-library", "library")
		primitive := filepath.Join(library, "primitives", "CalibrationPrimitive")
		component := filepath.Join(library, "components", "CalibrationComponent")
		if err := os.MkdirAll(filepath.Join(primitive, "versions", "1.0.0"), 0o755); err != nil {
			cleanup()
			return "", func() {}, err
		}
		if err := os.MkdirAll(filepath.Join(component, "versions", "1.0.0"), 0o755); err != nil {
			cleanup()
			return "", func() {}, err
		}
		if err := os.WriteFile(filepath.Join(primitive, "component.json"), []byte(`{"libraryId":"react-component-library:CalibrationPrimitive"}`), 0o644); err != nil {
			cleanup()
			return "", func() {}, err
		}
		if err := os.WriteFile(filepath.Join(component, "component.json"), []byte(`{"libraryId":"react-component-library:CalibrationComponent"}`), 0o644); err != nil {
			cleanup()
			return "", func() {}, err
		}
		lock := []byte(`{"libraryId":"react-component-library:CalibrationPrimitive","version":"1.0.0","dependencies":[{"libraryId":"react-component-library:CalibrationComponent","version":"1.0.0"}]}`)
		if err := os.WriteFile(filepath.Join(primitive, "versions", "1.0.0", "dependencies.json"), lock, 0o644); err != nil {
			cleanup()
			return "", func() {}, err
		}
	}
	if fixture.Mutation == "release-provenance" {
		library := filepath.Join(tmp, "scenarios", "react-component-library", "library")
		asset := filepath.Join(library, "components", "CalibrationBypass")
		if err := os.MkdirAll(filepath.Join(asset, "versions", "1.0.0"), 0o755); err != nil {
			cleanup()
			return "", func() {}, err
		}
		if err := os.WriteFile(filepath.Join(asset, "component.json"), []byte(`{"libraryId":"react-component-library:CalibrationBypass"}`), 0o644); err != nil {
			cleanup()
			return "", func() {}, err
		}
		if err := os.WriteFile(filepath.Join(library, "release-provenance.json"), []byte(`{"schemaVersion":1,"entries":[]}`), 0o644); err != nil {
			cleanup()
			return "", func() {}, err
		}
	}
	if fixture.Mutation == "released-hash" {
		dbPath := filepath.Join(tmp, "scenarios", "react-component-library", "data", "react-component-library.db")
		if err := createCalibrationDatabase(dbPath, fixture); err != nil {
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
	if fixture.Mutation == "surface-hard-coded" {
		dbPath := filepath.Join(tmp, "scenarios", "experience-manager", "data", "experience-manager.db")
		if err := createSurfaceCalibrationDatabase(dbPath, fixture.AssetID); err != nil {
			cleanup()
			return "", func() {}, err
		}
	}
	if fixture.Test != "" {
		testPath := filepath.Join(tmp, "scenarios", "calibration-adopter", "ui", "src", "components", "Calibration.test.tsx")
		if err := os.MkdirAll(filepath.Dir(testPath), 0o755); err != nil {
			cleanup()
			return "", func() {}, err
		}
		if err := copyFile(filepath.Join(fixtureDir, fixture.Test), testPath); err != nil {
			cleanup()
			return "", func() {}, err
		}
	}
	if fixture.Mutation == "conformance" && fixture.Source != "" {
		uiDir := filepath.Join(tmp, "scenarios", "react-component-library", "ui", "src")
		if info, err := os.Lstat(uiDir); err == nil && info.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(uiDir); err != nil {
				cleanup()
				return "", func() {}, err
			}
		}
		if err := os.MkdirAll(uiDir, 0o755); err != nil {
			cleanup()
			return "", func() {}, err
		}
		if err := copyFile(filepath.Join(fixtureDir, fixture.Source), filepath.Join(uiDir, "CalibrationConformance.tsx")); err != nil {
			cleanup()
			return "", func() {}, err
		}
	}
	if fixture.Mutation == "harness-missing-implementation" {
		missing := filepath.Join(tmp, "scenarios", "react-component-library", "harnesses", "showcase", "versions", "1.0.0", "Showcase.tsx")
		if err := os.Remove(missing); err != nil {
			cleanup()
			return "", func() {}, err
		}
	}
	return tmp, cleanup, nil
}
