package validation

import (
	"context"
	"path/filepath"
	"testing"
)

// hygieneACForTempScenario lays out a minimal Go scenario under a temp
// scenarios/ root containing the given api-relative files, then returns an
// AnalyzerContext resolved exactly like Service.ValidateScenario builds it
// (FilesystemDetector + real api/ dir). files keys are paths under api/.
func hygieneACForTempScenario(t *testing.T, scenario string, files map[string]string) AnalyzerContext {
	t.Helper()
	repoRoot := t.TempDir()
	scenarioDir := filepath.Join(repoRoot, "scenarios", scenario)
	writeFile(t, filepath.Join(scenarioDir, "api", "go.mod"), "module "+scenario+"\n\ngo 1.25.0\n")
	writeFile(t, filepath.Join(scenarioDir, ".vrooli", "service.json"), `{"maturity":"production"}`)
	for rel, content := range files {
		writeFile(t, filepath.Join(scenarioDir, "api", filepath.FromSlash(rel)), content)
	}
	det := FilesystemDetector{}.Detect(context.Background(), scenario, scenarioDir)
	return AnalyzerContext{
		Scenario:    scenario,
		ScenarioDir: scenarioDir,
		APIDir:      filepath.Join(scenarioDir, "api"),
		Language:    det.Language,
		Domains:     det.Domains,
	}
}

func TestHygieneSQLInHandlers(t *testing.T) {
	a := hygieneSQLInHandlers{}

	t.Run("positive_sql_in_handler_dir", func(t *testing.T) {
		ac := hygieneACForTempScenario(t, "demo", map[string]string{
			"handlers/orders/list.go": `package orders

import "context"

type repo struct{ db queryer }

type queryer interface {
	QueryContext(ctx context.Context, q string, args ...any) (any, error)
}

func (r *repo) List(ctx context.Context) error {
	_, err := r.db.QueryContext(ctx, "SELECT id FROM orders")
	return err
}
`,
		})
		got, err := a.Analyze(context.Background(), ac)
		if err != nil {
			t.Fatalf("analyze: %v", err)
		}
		if len(got) == 0 {
			t.Fatalf("expected DIRECT_SQL_IN_HANDLERS finding, got none")
		}
		if got[0].Code != "DIRECT_SQL_IN_HANDLERS" || got[0].Severity != SeverityWarning {
			t.Fatalf("unexpected finding %+v", got[0])
		}
	})

	t.Run("positive_http_handler_signature", func(t *testing.T) {
		ac := hygieneACForTempScenario(t, "demo", map[string]string{
			"internal/orders/web.go": `package orders

import (
	"database/sql"
	"net/http"
)

func ListHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	_, _ = db.Exec("DELETE FROM orders WHERE id = 1")
}
`,
		})
		got, err := a.Analyze(context.Background(), ac)
		if err != nil {
			t.Fatalf("analyze: %v", err)
		}
		if len(got) == 0 {
			t.Fatalf("expected finding for http handler with raw SQL, got none")
		}
	})

	t.Run("negative_repo_layer_no_sql", func(t *testing.T) {
		// A handler that calls a repository method (no SQL string) is clean.
		ac := hygieneACForTempScenario(t, "demo", map[string]string{
			"handlers/orders/list.go": `package orders

import "context"

type store interface{ List(ctx context.Context) error }

type Handler struct{ store store }

func (h *Handler) List(ctx context.Context) error {
	return h.store.List(ctx)
}
`,
		})
		got, err := a.Analyze(context.Background(), ac)
		if err != nil {
			t.Fatalf("analyze: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("expected no finding (no raw SQL), got %+v", got)
		}
	})

	t.Run("negative_sql_outside_handler", func(t *testing.T) {
		// Raw SQL in a repository file (not a handler) is NOT this analyzer's
		// concern — it only flags SQL inside transport handlers.
		ac := hygieneACForTempScenario(t, "demo", map[string]string{
			"internal/orders/repo.go": `package orders

import (
	"context"
	"database/sql"
)

func List(ctx context.Context, db *sql.DB) error {
	_, err := db.QueryContext(ctx, "SELECT id FROM orders")
	return err
}
`,
		})
		got, err := a.Analyze(context.Background(), ac)
		if err != nil {
			t.Fatalf("analyze: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("expected no finding (repo file, not a handler), got %+v", got)
		}
	})

	t.Run("exempt_test_file", func(t *testing.T) {
		// _test.go handler with raw SQL is exempt (CollectGoFiles drops it).
		ac := hygieneACForTempScenario(t, "demo", map[string]string{
			"handlers/orders/list_test.go": `package orders

import (
	"context"
	"database/sql"
	"testing"
)

func TestList(t *testing.T) {
	var db *sql.DB
	_, _ = db.QueryContext(context.Background(), "SELECT id FROM orders")
}
`,
		})
		got, err := a.Analyze(context.Background(), ac)
		if err != nil {
			t.Fatalf("analyze: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("expected no finding (test file exempt), got %+v", got)
		}
	})
}
