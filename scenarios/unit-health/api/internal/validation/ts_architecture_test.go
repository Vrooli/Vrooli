package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalyzeTSArchitectureTestUtilMissing(t *testing.T) {
	root := t.TempDir()
	// Three co-located component tests under src/, but no src/test-utils module.
	for _, n := range []string{"A", "B", "C"} {
		writeFile(t, filepath.Join(root, "src", n+".test.tsx"), "import {render} from '@testing-library/react';\nit('x',()=>{expect(1).toBe(1)});\n")
	}
	ws := Workspace{ID: "ui", Language: "typescript", RootPath: root}
	findings := analyzeArchitecture("demo", []Workspace{ws}, fixedNowStr)
	if _, ok := findingByCode(findings, codeTestUtilMissing); !ok {
		t.Errorf("expected TEST_UTIL_MISSING for a UI surface with tests but no test-utils, got %v", codes(findings))
	}
}

func TestAnalyzeTSArchitectureNotColocated(t *testing.T) {
	root := t.TempDir()
	// A test outside src/ in a __tests__ dir is not co-located.
	writeFile(t, filepath.Join(root, "__tests__", "App.test.tsx"), "it('x',()=>{expect(1).toBe(1)});\n")
	ws := Workspace{ID: "ui", Language: "typescript", RootPath: root}
	findings := analyzeArchitecture("demo", []Workspace{ws}, fixedNowStr)
	if _, ok := findingByCode(findings, codeTestNotColocated); !ok {
		t.Errorf("expected TEST_NOT_COLOCATED for a test outside src/, got %v", codes(findings))
	}
}

func TestAnalyzeTSArchitectureCleanWithTestUtils(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "src", "test-utils", "render.tsx"), "export const x = 1;\n")
	writeFile(t, filepath.Join(root, "src", "A.test.tsx"), "import {x} from './test-utils/render';\nit('x',()=>{expect(x).toBe(1)});\n")
	ws := Workspace{ID: "ui", Language: "typescript", RootPath: root}
	findings := analyzeArchitecture("demo", []Workspace{ws}, fixedNowStr)
	if _, ok := findingByCode(findings, codeTestUtilMissing); ok {
		t.Errorf("a UI surface with src/test-utils must not be flagged TEST_UTIL_MISSING, got %v", codes(findings))
	}
}

func TestAnalyzeTSArchitectureProjectionDriftMissingSetupFile(t *testing.T) {
	root := t.TempDir()
	writeCanonicalUIProjection(t, root)
	writeFile(t, filepath.Join(root, "vite.config.ts"), strings.ReplaceAll(canonicalViteConfig(), "setupFiles: ['./src/test-setup.ts'],", ""))

	findings := analyzeArchitecture("demo", []Workspace{{ID: "ui", Language: "typescript", RootPath: root, Framework: "vite"}}, fixedNowStr)
	f, ok := findingByCode(findings, codeUnitProjectionDrift)
	if !ok {
		t.Fatalf("expected UNIT_POLICY_PROJECTION_DRIFT, got %v", codes(findings))
	}
	if !strings.Contains(f.Observed, "setupFiles") {
		t.Fatalf("expected setupFiles drift evidence, got %+v", f)
	}
}

func TestAnalyzeTSArchitectureProjectionDriftLoweredThreshold(t *testing.T) {
	root := t.TempDir()
	writeCanonicalUIProjection(t, root)
	writeFile(t, filepath.Join(root, "vite.config.ts"), strings.ReplaceAll(canonicalViteConfig(), "branches: 85", "branches: 70"))

	findings := analyzeArchitecture("demo", []Workspace{{ID: "ui", Language: "typescript", RootPath: root, Framework: "vite"}}, fixedNowStr)
	f, ok := findingByCode(findings, codeUnitProjectionDrift)
	if !ok {
		t.Fatalf("expected UNIT_POLICY_PROJECTION_DRIFT, got %v", codes(findings))
	}
	if !strings.Contains(f.Evidence, "branches=70.0") {
		t.Fatalf("expected lowered branch threshold evidence, got %+v", f)
	}
}

func TestAnalyzeTSArchitectureProjectionDriftMissingRenderHelper(t *testing.T) {
	root := t.TempDir()
	writeCanonicalUIProjection(t, root)
	if err := os.Remove(filepath.Join(root, "src", "test-utils", "renderWithProviders.tsx")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "src", "test-utils", "index.ts"), "export const x = 1;\n")

	findings := analyzeArchitecture("demo", []Workspace{{ID: "ui", Language: "typescript", RootPath: root, Framework: "vite"}}, fixedNowStr)
	f, ok := findingByCode(findings, codeUnitProjectionDrift)
	if !ok {
		t.Fatalf("expected UNIT_POLICY_PROJECTION_DRIFT, got %v", codes(findings))
	}
	if !strings.Contains(f.Observed, "renderWithProviders") {
		t.Fatalf("expected render helper drift, got %+v", f)
	}
}

func TestAnalyzeTSArchitectureProjectionDriftDirectTestingLibraryRender(t *testing.T) {
	root := t.TempDir()
	writeCanonicalUIProjection(t, root)
	writeFile(t, filepath.Join(root, "src", "App.test.tsx"), `
import { render, screen } from "@testing-library/react";
import App from "./App";

it("renders", () => {
  render(<App />);
  expect(screen.getByText("x")).toBeInTheDocument();
});
`)

	findings := analyzeArchitecture("demo", []Workspace{{ID: "ui", Language: "typescript", RootPath: root, Framework: "vite"}}, fixedNowStr)
	var sawDirectRender bool
	for _, f := range findings {
		if f.Code == codeUnitProjectionDrift && strings.Contains(f.Observed, "direct Testing Library render") {
			sawDirectRender = true
		}
	}
	if !sawDirectRender {
		t.Fatalf("expected direct render projection drift, got %+v", findings)
	}
}

func TestAnalyzeTSArchitectureProjectionAllowsRenderWithProviders(t *testing.T) {
	root := t.TempDir()
	writeCanonicalUIProjection(t, root)
	writeFile(t, filepath.Join(root, "src", "App.test.tsx"), `
import { screen } from "@testing-library/react";
import { renderWithProviders } from "./test-utils";
import App from "./App";

it("renders", () => {
  renderWithProviders(<App />);
  expect(screen.getByText("x")).toBeInTheDocument();
});
`)

	findings := analyzeArchitecture("demo", []Workspace{{ID: "ui", Language: "typescript", RootPath: root, Framework: "vite"}}, fixedNowStr)
	for _, f := range findings {
		if f.Code == codeUnitProjectionDrift && strings.Contains(f.Observed, "direct Testing Library render") {
			t.Fatalf("renderWithProviders path should not be flagged as direct render drift: %+v", findings)
		}
	}
}

func TestAnalyzeTSArchitectureProjectionDriftMissingImportBan(t *testing.T) {
	root := t.TempDir()
	writeCanonicalUIProjection(t, root)
	writeFile(t, filepath.Join(root, "eslint.config.js"), "export default [];\n")

	findings := analyzeArchitecture("demo", []Workspace{{ID: "ui", Language: "typescript", RootPath: root, Framework: "vite"}}, fixedNowStr)
	var sawImportBan bool
	for _, f := range findings {
		if f.Code == codeUnitProjectionDrift && strings.Contains(f.Observed, "production import ban") {
			sawImportBan = true
		}
	}
	if !sawImportBan {
		t.Fatalf("expected production import-ban projection drift, got %v", codes(findings))
	}
}

func TestAnalyzeTSArchitectureProjectionClean(t *testing.T) {
	root := t.TempDir()
	writeCanonicalUIProjection(t, root)

	findings := analyzeArchitecture("demo", []Workspace{{ID: "ui", Language: "typescript", RootPath: root, Framework: "vite"}}, fixedNowStr)
	if _, ok := findingByCode(findings, codeUnitProjectionDrift); ok {
		t.Fatalf("canonical UI projection should be clean, got %+v", findings)
	}
}

func TestAnalyzeTSArchitectureProjectionUsesPolicyCoverageFloor(t *testing.T) {
	scenarioRoot := t.TempDir()
	uiRoot := filepath.Join(scenarioRoot, "ui")
	profile := reactViteUnitPolicyProfile()
	uiClass := profile.PolicyClasses["react_vite_ui"]
	uiClass.Coverage.MinimumPercent = 90
	profile.PolicyClasses["react_vite_ui"] = uiClass
	writeUnitPolicyProfile(t, scenarioRoot, profile)
	writeCanonicalUIProjection(t, uiRoot)

	findings := analyzeArchitecture("demo", []Workspace{{ID: "ui", Language: "typescript", RootPath: uiRoot, Framework: "vite"}}, fixedNowStr)
	f, ok := findingByCode(findings, codeUnitProjectionDrift)
	if !ok {
		t.Fatalf("expected stricter policy floor to flag 85%% native thresholds, got %v", codes(findings))
	}
	if !strings.Contains(f.Expected, "90.0") {
		t.Fatalf("expected policy floor in finding detail, got %+v", f)
	}
}

func TestBuildProjectionChecksReportsPolicyAndNativeValues(t *testing.T) {
	scenarioRoot := t.TempDir()
	uiRoot := filepath.Join(scenarioRoot, "ui")
	profile := reactViteUnitPolicyProfile()
	uiClass := profile.PolicyClasses["react_vite_ui"]
	uiClass.Coverage.MinimumPercent = 90
	profile.PolicyClasses["react_vite_ui"] = uiClass
	writeUnitPolicyProfile(t, scenarioRoot, profile)
	writeCanonicalUIProjection(t, uiRoot)

	checks := buildProjectionChecks(scenarioRoot, []Workspace{{ID: "ui", Language: "typescript", RootPath: uiRoot, Framework: "vite"}})
	byID := map[string]ProjectionCheck{}
	for _, check := range checks {
		byID[check.ID] = check
	}

	env := byID["ui:vitest.environment"]
	if env.Status != "pass" || env.PolicyValue != "jsdom" || env.NativeValue != "jsdom" {
		t.Fatalf("environment projection = %+v, want pass jsdom/jsdom", env)
	}
	threshold := byID["ui:coverage.threshold.branches"]
	if threshold.Status != "drift" || threshold.PolicyValue != "90" || threshold.NativeValue != "85" {
		t.Fatalf("threshold projection = %+v, want drift 90/85", threshold)
	}
	if threshold.FindingCode != codeUnitProjectionDrift {
		t.Fatalf("threshold finding code = %q, want %s", threshold.FindingCode, codeUnitProjectionDrift)
	}
}

func TestAnalyzeTSArchitectureProjectionCleanWithEquivalentConfigShape(t *testing.T) {
	root := t.TempDir()
	writeCanonicalUIProjection(t, root)
	writeFile(t, filepath.Join(root, "vite.config.ts"), `export default defineConfig({
  "test": {
    // Comments mentioning setupFiles or branches: 0 must not drive parsing.
    "environment": "jsdom",
    "setupFiles": [
      "./src/test-setup.ts",
    ],
    "coverage": {
      "provider": "v8",
      "reporter": [
        "text",
        "json-summary",
        "json",
      ],
      "thresholds": {
        "lines": 85,
        "functions": 85,
        "branches": 85,
        "statements": 85,
      },
    },
  },
});
`)
	writeFile(t, filepath.Join(root, "eslint.config.js"), `export default [{
  rules: {
    "no-restricted-imports": ["error", {
      patterns: [{ group: [
        "@/test-utils",
        "@/features/*/mocks/*",
      ] }],
    }],
  },
}];`)

	findings := analyzeArchitecture("demo", []Workspace{{ID: "ui", Language: "typescript", RootPath: root, Framework: "vite"}}, fixedNowStr)
	if _, ok := findingByCode(findings, codeUnitProjectionDrift); ok {
		t.Fatalf("equivalent quoted/multiline projection should be clean, got %+v", findings)
	}
}

func TestAnalyzeTSArchitectureReactViteTemplateProjectionClean(t *testing.T) {
	repoRoot := findRepoRoot(t)
	root := filepath.Join(repoRoot, "templates", "scenarios", "react-vite", "ui")
	if _, err := os.Stat(filepath.Join(root, "vite.config.ts")); err != nil {
		t.Fatalf("react-vite template UI not found: %v", err)
	}

	findings := analyzeTSArchitecture("react-vite", Workspace{ID: "ui", Language: "typescript", RootPath: root, Framework: "vite"}, fixedNowStr)
	if _, ok := findingByCode(findings, codeUnitProjectionDrift); ok {
		t.Fatalf("react-vite template UI projection should be clean, got %+v", findings)
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		if fileExists(filepath.Join(dir, "templates", "scenarios", "react-vite", ".vrooli", "testing.json")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("repo root not found")
	return ""
}

func writeCanonicalUIProjection(t *testing.T, root string) {
	t.Helper()
	writeFile(t, filepath.Join(root, "package.json"), `{
  "scripts": {"test": "vitest run", "test:coverage": "vitest run --coverage"},
  "devDependencies": {"vitest": "^3.0.0"}
}`)
	writeFile(t, filepath.Join(root, "vite.config.ts"), canonicalViteConfig())
	writeFile(t, filepath.Join(root, "eslint.config.js"), `export default [{
  rules: {
    "no-restricted-imports": ["error", {patterns: [{group: ["**/test-utils", "@/test-utils/*", "**/features/*/mocks"]}]}],
  },
}];`)
	writeFile(t, filepath.Join(root, "src", "test-utils", "renderWithProviders.tsx"), "export function renderWithProviders() {}\n")
	writeFile(t, filepath.Join(root, "src", "test-utils", "index.ts"), "export { renderWithProviders } from './renderWithProviders';\n")
	writeFile(t, filepath.Join(root, "src", "App.test.tsx"), "import { renderWithProviders } from './test-utils';\nit('x',()=>{renderWithProviders(null); expect(1).toBe(1)});\n")
}

func canonicalViteConfig() string {
	return `export default {
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test-setup.ts'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json-summary', 'json'],
      thresholds: {
        lines: 85,
        functions: 85,
        branches: 85,
        statements: 85,
      },
    },
  },
};`
}
