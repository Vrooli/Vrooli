package httpserver

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"test-genie/internal/orchestrator"

	repocontract "github.com/vrooli/repo-contract-go"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := "healthy"
	dbConnected := true

	// Health pings the primary pool directly — liveness is about the underlying
	// connection, independent of any per-request test-pool routing.
	if err := s.db.Primary().PingContext(r.Context()); err != nil {
		status = "unhealthy"
		dbConnected = false
	}

	operations := map[string]interface{}{}
	if s.executionHistory != nil {
		if latest, err := s.executionHistory.Latest(r.Context()); err == nil && latest != nil {
			operations["lastExecution"] = executionSummaryPayload(latest)
		} else if err != nil {
			s.log("latest execution lookup failed", map[string]interface{}{"error": err.Error()})
		}
	}

	response := map[string]interface{}{
		"status":    status,
		"service":   s.serviceName(),
		"version":   "1.0.0",
		"readiness": status == "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"dependencies": map[string]map[string]bool{
			"database": {
				"connected": dbConnected,
			},
		},
		"operations": operations,
	}

	s.writeJSON(w, http.StatusOK, response)
}

func executionSummaryPayload(result *orchestrator.SuiteExecutionResult) map[string]interface{} {
	if result == nil {
		return nil
	}
	return map[string]interface{}{
		"executionId":  result.ExecutionID,
		"scenario":     result.ScenarioName,
		"success":      result.Success,
		"completedAt":  result.CompletedAt.Format(time.RFC3339),
		"startedAt":    result.StartedAt.Format(time.RFC3339),
		"phaseSummary": result.PhaseSummary,
		"preset":       result.PresetUsed,
	}
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
