package reactvitest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"unit-health/internal/adapters"
)

func TestAnalyzerContractAndProjectionChecks(t *testing.T) {
	analyzer := Analyzer{}
	if analyzer.Identity().ID != "react-vitest" || !analyzer.Matches(adapters.Match{Language: "typescript", Framework: "vite"}) || analyzer.Matches(adapters.Match{Language: "go", Framework: "go test"}) {
		t.Fatal("adapter identity or matching failed")
	}
	if err := analyzer.ValidatePolicySettings(map[string]json.RawMessage{"environment": json.RawMessage(`"jsdom"`), "setup_files": json.RawMessage(`[` + "\"setup.ts\"" + `]`), "coverage_provider": json.RawMessage(`"v8"`)}); err != nil {
		t.Fatal(err)
	}
	policy := analyzer.DefaultProjectionPolicy()
	if policy.CoverageProvider != "v8" || len(policy.CoverageExclude) == 0 {
		t.Fatalf("default policy = %+v", policy)
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"scripts":{"test":"vitest run","test:coverage":"vitest run --coverage"},"devDependencies":{"vitest":"1"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "vite.config.js"), []byte("export default { test: { environment: 'jsdom', setupFiles: ['setup.ts'], coverage: { provider: 'v8', reporter: ['json-summary'], include: ['src/**/*.ts'], exclude: [], reportOnFailure: true, thresholds: { lines: 80, functions: 80, branches: 80, statements: 80 } } } }"), 0o600); err != nil {
		t.Fatal(err)
	}
	checks := analyzer.AnalyzeProjectionChecks(adapters.ProjectionInput{RootPath: root, Policy: policy})
	if len(checks) < 10 {
		t.Fatalf("projection checks = %+v", checks)
	}
	if boolText(true) != "true" || boolText(false) != "false" || thresholdValue(0, false) != "" || formatCoverageFloor(80) != "80" {
		t.Fatal("projection formatting helpers failed")
	}
}
