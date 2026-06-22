package validation

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// hygieneAnalyzers is the Tier-2 persistence-hygiene analyzer set, in registry
// order, used by the dogfood test.
func hygieneAnalyzers() []Analyzer {
	return []Analyzer{
		hygieneRawSQLOpen{},
		hygieneRoutedDriver{},
		hygieneHandleCapture{},
		hygieneRowsClose{},
		hygieneSQLInHandlers{},
		hygieneSQLitePoolDeadlock{},
	}
}

// findRepoRoot walks up from the test's working directory (the package dir)
// until it finds a directory whose scenarios/storage-health/.vrooli/maturity.json
// exists — the repo root. Returns "" if not found (the test then skips).
func findRepoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		marker := filepath.Join(dir, "scenarios", "storage-health", ".vrooli", "maturity.json")
		if _, err := os.Stat(marker); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// TestHygiene_Dogfood asserts storage-health's OWN api/ surface trips ZERO
// persistence-hygiene findings: it uses the api-core/database seams correctly
// (no raw sql.Open, no driver import, no captured *sql.DB, rows always closed),
// and its MaxOpenConns:1 SQLite config carries NO nested-query-in-open-rows.
func TestHygiene_Dogfood(t *testing.T) {
	repoRoot := findRepoRoot()
	if repoRoot == "" {
		t.Skip("repo root not found (storage-health not on disk); dogfood check skipped")
	}
	scenarioDir := filepath.Join(repoRoot, "scenarios", "storage-health")
	apiDir := filepath.Join(scenarioDir, "api")
	if info, err := os.Stat(apiDir); err != nil || !info.IsDir() {
		t.Skip("storage-health api/ surface not found; dogfood check skipped")
	}

	det := FilesystemDetector{}.Detect(context.Background(), "storage-health", scenarioDir)
	ac := AnalyzerContext{
		Scenario:    "storage-health",
		ScenarioDir: scenarioDir,
		APIDir:      apiDir,
		Language:    det.Language,
		Domains:     det.Domains,
	}
	if !ac.IsGo() {
		t.Fatalf("storage-health api language = %q, want go (dogfood relies on Go detection)", ac.Language)
	}

	for _, a := range hygieneAnalyzers() {
		got, err := a.Analyze(context.Background(), ac)
		if err != nil {
			t.Fatalf("analyzer %q failed: %v", a.Name(), err)
		}
		if len(got) != 0 {
			for _, f := range got {
				t.Errorf("analyzer %q flagged storage-health: [%s] %s @ %s", a.Name(), f.Code, f.Title, f.Location)
			}
		}
	}
}
