package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	apihealth "github.com/vrooli/api-core/health"
	repocontract "github.com/vrooli/repo-contract-go"

	"test-genie/internal/dbexec"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// The lifecycle contract is owned by api-core. This path performs exactly one
	// bounded primary-store ping; it must not aggregate execution history or
	// wait behind advisory work longer than its own small health budget.
	check := apihealth.DB(s.db.Primary())
	if s.healthDB != nil {
		check = apihealth.Func("database", func(ctx context.Context) error {
			return checkSQLiteSchema(ctx, s.healthDB)
		})
	}
	apihealth.New(s.serviceName()).
		Version("1.0.0").
		Timeout(250*time.Millisecond).
		Check(check, apihealth.Critical).
		Check(apihealth.Func("self_health_sweep", s.checkSweepStatus), apihealth.Optional).
		Handler()(w, r)
}

func (s *Server) checkSweepStatus(context.Context) error {
	if s.sweepStatus == nil {
		return nil
	}
	status := s.sweepStatus.Snapshot()
	if status.Outcome == "failed" || status.Outcome == "timed_out" {
		return fmt.Errorf("last advisory sweep %s: %s", status.Outcome, status.Error)
	}
	return nil
}

// checkSQLiteSchema is a minimal read-only SQLite probe. Unlike runtime work,
// it uses the dedicated lifecycle connection and performs no history lookup or
// payload hydration. A schema read proves the connection can execute a query,
// not merely allocate a pool slot.
func checkSQLiteSchema(ctx context.Context, db dbexec.HealthProbe) error {
	var version int
	if err := db.QueryRowContext(ctx, "PRAGMA schema_version").Scan(&version); err != nil {
		return fmt.Errorf("query sqlite schema version: %w", err)
	}
	return nil
}

// handleGetConfig returns configuration values needed by the UI.
// This includes paths that should NOT be hardcoded in the frontend.
func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	repoRoot := s.resolveRepoRoot()
	scenariosPath := ""
	testGeniePath := ""
	if repoRoot != "" {
		scenariosPath = s.resolveScenariosRoot()
		if path, err := repocontract.ResolveScenarioPath(repoRoot, "test-genie"); err == nil {
			testGeniePath = path
		}
	}

	response := map[string]interface{}{
		"repoRoot":      repoRoot,
		"testGeniePath": testGeniePath,
		"testGenieCLI":  "test-genie", // CLI command name (should be on PATH)
		"scenariosPath": scenariosPath,
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
	}

	s.writeJSON(w, http.StatusOK, response)
}

func (s *Server) resolveRepoRoot() string {
	if s.scenarios != nil {
		if scenariosRoot := filepath.Clean(s.scenarios.ScenarioRoot()); scenariosRoot != "." && scenariosRoot != "" {
			root := filepath.Dir(scenariosRoot)
			if _, err := repocontract.FindRepoRoot(root); err == nil {
				return root
			}
		}
	}

	for _, start := range []string{
		strings.TrimSpace(os.Getenv("SCENARIOS_ROOT")),
		strings.TrimSpace(os.Getenv("VROOLI_ROOT")),
	} {
		if start == "" {
			continue
		}
		if root, err := repocontract.FindRepoRoot(start); err == nil {
			return root
		}
	}

	if root, err := repocontract.FindRepoRootFromEnvOrCWD(); err == nil {
		return root
	}
	return ""
}

func (s *Server) resolveScenariosRoot() string {
	if s.scenarios != nil {
		if value := filepath.Clean(s.scenarios.ScenarioRoot()); value != "." && value != "" {
			return value
		}
	}

	if root := s.resolveRepoRoot(); root != "" {
		contract, err := repocontract.LoadDefault(root)
		if err == nil {
			if path, pathErr := contract.TopLevelDir(root, "scenarios"); pathErr == nil {
				return path
			}
		}
	}
	return ""
}
