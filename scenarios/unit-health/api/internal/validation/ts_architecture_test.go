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
	ws := Workspace{ID: "ui", Language: "typescript", Framework: "vite", RootPath: root}
	findings := analyzeArchitecture("demo", []Workspace{ws}, fixedNowStr)
	if _, ok := findingByCode(findings, codeTestUtilMissing); !ok {
		t.Errorf("expected TEST_UTIL_MISSING for a UI surface with tests but no test-utils, got %v", codes(findings))
	}
}

func TestAnalyzeTSArchitectureNotColocated(t *testing.T) {
	root := t.TempDir()
	// A test outside src/ in a __tests__ dir is not co-located.
	writeFile(t, filepath.Join(root, "__tests__", "App.test.tsx"), "it('x',()=>{expect(1).toBe(1)});\n")
	ws := Workspace{ID: "ui", Language: "typescript", Framework: "vite", RootPath: root}
	findings := analyzeArchitecture("demo", []Workspace{ws}, fixedNowStr)
	if _, ok := findingByCode(findings, codeTestNotColocated); !ok {
		t.Errorf("expected TEST_NOT_COLOCATED for a test outside src/, got %v", codes(findings))
	}
}

func TestAnalyzeTSArchitectureCleanWithTestUtils(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "src", "test-utils", "render.tsx"), "export const x = 1;\n")
	writeFile(t, filepath.Join(root, "src", "A.test.tsx"), "import {x} from './test-utils/render';\nit('x',()=>{expect(x).toBe(1)});\n")
	ws := Workspace{ID: "ui", Language: "typescript", Framework: "vite", RootPath: root}
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

func TestAnalyzeTSArchitectureNativeThresholdsAreASecondCoverageOwner(t *testing.T) {
	root := t.TempDir()
	writeCanonicalUIProjection(t, root)
	writeFile(t, filepath.Join(root, "vite.config.ts"), strings.Replace(canonicalViteConfig(), "reportOnFailure: true,", "reportOnFailure: true,\n      thresholds: { lines: 85, functions: 85, branches: 70, statements: 85 },", 1))

	findings := analyzeArchitecture("demo", []Workspace{{ID: "ui", Language: "typescript", RootPath: root, Framework: "vite"}}, fixedNowStr)
	f, ok := findingByCode(findings, codeUnitProjectionDrift)
	if !ok {
		t.Fatalf("expected UNIT_POLICY_PROJECTION_DRIFT, got %v", codes(findings))
	}
	if f.Evidence != "coverage.thresholds lines=85,statements=85,functions=85,branches=70" || !strings.Contains(f.Message, "coverage.committed") {
		t.Fatalf("native thresholds must be reported as duplicating coverage.committed, got %+v", f)
	}
}

func TestAnalyzeTSArchitectureJudgesTheConfigVitestLoads(t *testing.T) {
	root := t.TempDir()
	writeCanonicalUIProjection(t, root)
	// A vitest.config.ts shadows the canonical vite.config.ts, so the policy
	// projection Vitest actually runs is the incomplete one.
	writeFile(t, filepath.Join(root, "vitest.config.ts"), "export default { test: { environment: 'jsdom', setupFiles: ['./src/test-setup.ts'], coverage: { provider: 'v8' } } }")

	findings := analyzeArchitecture("demo", []Workspace{{ID: "ui", Language: "typescript", RootPath: root, Framework: "vite"}}, fixedNowStr)
	f, ok := findingByCode(findings, codeUnitProjectionDrift)
	if !ok || filepath.Base(f.FilePath) != "vitest.config.ts" {
		t.Fatalf("drift must be reported against the effective vitest.config.ts, got %+v", findings)
	}
}

func TestAnalyzeTSArchitectureProjectionDriftNonVitestCoverageScript(t *testing.T) {
	root := t.TempDir()
	writeCanonicalUIProjection(t, root)
	writeFile(t, filepath.Join(root, "package.json"), `{
  "scripts": {"test": "vitest run", "test:coverage": "jest --coverage"},
  "devDependencies": {"vitest": "^3.0.0"}
}`)

	findings := analyzeArchitecture("demo", []Workspace{{ID: "ui", Language: "typescript", RootPath: root, Framework: "vite"}}, fixedNowStr)
	var sawCoverageScript bool
	for _, f := range findings {
		if f.Code == codeUnitProjectionDrift && strings.Contains(f.Observed, "non-Vitest coverage script") {
			sawCoverageScript = true
		}
	}
	if !sawCoverageScript {
		t.Fatalf("expected non-Vitest coverage script drift, got %+v", findings)
	}
}

func TestAnalyzeTSArchitectureProjectionDriftMissingCoverageDenominator(t *testing.T) {
	root := t.TempDir()
	writeCanonicalUIProjection(t, root)
	writeFile(t, filepath.Join(root, "vite.config.ts"), strings.ReplaceAll(canonicalViteConfig(), "      include: ['src/**/*.{ts,tsx}'],\n", ""))

	findings := analyzeArchitecture("demo", []Workspace{{ID: "ui", Language: "typescript", RootPath: root, Framework: "vite"}}, fixedNowStr)
	f, ok := findingByCode(findings, codeUnitProjectionDrift)
	if !ok {
		t.Fatalf("expected UNIT_POLICY_PROJECTION_DRIFT, got %v", codes(findings))
	}
	if !strings.Contains(f.Observed, "source coverage include") {
		t.Fatalf("expected coverage include drift evidence, got %+v", f)
	}
}

func TestAnalyzeTSArchitectureProjectionDriftMissingReportOnFailure(t *testing.T) {
	root := t.TempDir()
	writeCanonicalUIProjection(t, root)
	writeFile(t, filepath.Join(root, "vite.config.ts"), strings.ReplaceAll(canonicalViteConfig(), "      reportOnFailure: true,\n", ""))

	findings := analyzeArchitecture("demo", []Workspace{{ID: "ui", Language: "typescript", RootPath: root, Framework: "vite"}}, fixedNowStr)
	var sawReportOnFailure bool
	for _, f := range findings {
		if f.Code == codeUnitProjectionDrift && strings.Contains(f.Observed, "reportOnFailure") {
			sawReportOnFailure = true
		}
	}
	if !sawReportOnFailure {
		t.Fatalf("expected reportOnFailure projection drift, got %+v", findings)
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

func TestAnalyzeTSArchitectureProjectionAllowsDocumentedProviderFreeException(t *testing.T) {
	root := t.TempDir()
	writeCanonicalUIProjection(t, root)
	writeFile(t, filepath.Join(root, "src", "ThemeProvider.test.tsx"), `
import { render, screen } from "@testing-library/react";
import { ThemeProvider } from "./ThemeProvider";

// provider-free-exception: this test mounts ThemeProvider itself; the canonical
// wrapper would double-provide and fight over documentElement.
it("provides", () => {
  render(<ThemeProvider>x</ThemeProvider>);
  expect(screen.getByText("x")).toBeInTheDocument();
});
`)

	findings := analyzeArchitecture("demo", []Workspace{{ID: "ui", Language: "typescript", RootPath: root, Framework: "vite"}}, fixedNowStr)
	for _, f := range findings {
		if f.Code == codeUnitProjectionDrift && strings.Contains(f.Observed, "direct Testing Library render") {
			t.Fatalf("documented provider-free exception should not be flagged: %+v", findings)
		}
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

func TestBuildProjectionChecksReportsPolicyAndNativeValues(t *testing.T) {
	scenarioRoot := t.TempDir()
	uiRoot := filepath.Join(scenarioRoot, "ui")
	writeUnitPolicyProfile(t, scenarioRoot, reactViteUnitPolicyProfile())
	writeCanonicalUIProjection(t, uiRoot)
	writeFile(t, filepath.Join(uiRoot, "vite.config.ts"), strings.Replace(canonicalViteConfig(), "reportOnFailure: true,", "reportOnFailure: true,\n      thresholds: { branches: 85 },", 1))

	checks := buildProjectionChecks(scenarioRoot, []Workspace{{ID: "ui", Language: "typescript", RootPath: uiRoot, Framework: "vite"}})
	byID := map[string]ProjectionCheck{}
	for _, check := range checks {
		byID[check.ID] = check
	}

	env := byID["ui:vitest.environment"]
	if env.Status != "pass" || env.PolicyValue != "jsdom" || env.NativeValue != "jsdom" {
		t.Fatalf("environment projection = %+v, want pass jsdom/jsdom", env)
	}
	threshold := byID["ui:coverage.thresholds"]
	if threshold.Status != "drift" || !strings.HasPrefix(threshold.PolicyValue, "absent") || threshold.NativeValue != "branches=85" {
		t.Fatalf("threshold projection = %+v, want drift absent/branches=85", threshold)
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
      "reportOnFailure": true,
      "include": [
        "src/**/*.{ts,tsx}",
      ],
      "exclude": [
        "src/**/*.test.{ts,tsx}",
        "src/**/*.spec.{ts,tsx}",
        "src/**/*.d.ts",
        "src/main.tsx",
        "src/test-setup.ts",
        "src/test-utils/**",
        "src/consts/strings.generated.ts",
        "src/i18n/locales/**",
        "src/**/generated/**",
      ],
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

	findings := analyzeArchitecture("react-vite", []Workspace{{ID: "ui", Language: "typescript", RootPath: root, Framework: "vite"}}, fixedNowStr)
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
      reportOnFailure: true,
      include: ['src/**/*.{ts,tsx}'],
      exclude: [
        'src/**/*.test.{ts,tsx}',
        'src/**/*.spec.{ts,tsx}',
        'src/**/*.d.ts',
        'src/main.tsx',
        'src/test-setup.ts',
        'src/test-utils/**',
        'src/consts/strings.generated.ts',
        'src/i18n/locales/**',
        'src/**/generated/**',
      ],
    },
  },
};`
}
