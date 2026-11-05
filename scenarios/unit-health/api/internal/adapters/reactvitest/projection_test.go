package reactvitest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPolicyFromSettingsKeepsProjectionInterpretationInAdapter(t *testing.T) {
	policy := PolicyFromSettings(map[string]json.RawMessage{
		"environment":       json.RawMessage(`"jsdom"`),
		"setup_files":       json.RawMessage(`["./src/test-setup.ts"]`),
		"coverage_provider": json.RawMessage(`"v8"`),
	})
	if policy.Environment != "jsdom" || len(policy.SetupFiles) != 1 || policy.CoverageProvider != "v8" {
		t.Fatalf("policy = %+v", policy)
	}
}

func TestValidatePolicySettingsRejectsUnknownNativeKeys(t *testing.T) {
	err := ValidatePolicySettings(map[string]json.RawMessage{"unknown_runner_flag": json.RawMessage(`true`)})
	if err == nil {
		t.Fatal("unknown adapter setting unexpectedly accepted")
	}
}

func TestParseProjectionExtractsDeclaredVitestContract(t *testing.T) {
	projection := Parse(`
    // test config
    export default { test: { environment: 'jsdom', setupFiles: ['./src/test-setup.ts'], coverage: {
      provider: 'v8', reporter: ['text', 'json-summary'], include: ['src/**/*.{ts,tsx}'],
      exclude: ['**/*.test.tsx'], reportOnFailure: true, thresholds: { lines: 80, functions: 81, branches: 82, statements: 83 }
    } } }
  `, `export default [{ rules: { 'no-restricted-imports': ['error', 'src/test-utils', 'features/*/mocks'] } }]`)
	if !projection.HasVitestConfig || projection.Environment != "jsdom" || projection.CoverageProvider != "v8" {
		t.Fatalf("projection = %+v", projection)
	}
	if !projection.ReportOnFailure || projection.Thresholds["branches"] != 82 || !projection.HasImportBanRule {
		t.Fatalf("projection completeness = %+v", projection)
	}
	if !ContainsAllSetupFiles(projection.SetupFiles, []string{"src/test-setup.ts"}) {
		t.Fatalf("setup files = %v", projection.SetupFiles)
	}
}

func TestAnalyzeProjectionReturnsAdapterOwnedDrift(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"scripts":{"test":"vitest run"},"devDependencies":{"vitest":"1"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "vite.config.ts"), []byte("export default { test: { environment: 'jsdom' } }"), 0o644); err != nil {
		t.Fatal(err)
	}
	drifts := AnalyzeProjection(ProjectionInput{RootPath: root, Policy: ProjectionPolicy{Environment: "jsdom", CoverageFloor: 80, CoverageProvider: "v8", ReportOnFailure: true}})
	if len(drifts) == 0 {
		t.Fatal("expected adapter-owned projection drift")
	}
	for _, drift := range drifts {
		if drift.File == "" || drift.Message == "" || drift.Remediation == "" {
			t.Fatalf("incomplete drift = %+v", drift)
		}
	}
}

func TestAnalyzeArchitectureReturnsReactSpecificDrift(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		path := filepath.Join(src, "Button"+string(rune('A'+i))+".test.tsx")
		if err := os.WriteFile(path, []byte("import { render } from '@testing-library/react'\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	drifts := AnalyzeArchitecture(root)
	seen := map[string]bool{}
	for _, drift := range drifts {
		seen[drift.Code] = true
	}
	if !seen["TEST_UTIL_MISSING"] {
		t.Fatalf("architecture drifts = %+v", drifts)
	}
	utils := filepath.Join(src, "test-utils")
	if err := os.MkdirAll(utils, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(utils, "renderWithProviders.tsx"), []byte("export const renderWithProviders = () => undefined\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	drifts = AnalyzeArchitecture(root)
	seen = map[string]bool{}
	for _, drift := range drifts {
		seen[drift.Code] = true
	}
	if !seen["UNIT_POLICY_PROJECTION_DRIFT"] {
		t.Fatalf("architecture drifts after helper = %+v", drifts)
	}
}
