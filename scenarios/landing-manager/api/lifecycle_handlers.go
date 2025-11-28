package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

// Security: Validate scenario IDs to prevent path traversal and command injection
// Must match: lowercase alphanumeric, hyphens, underscores (1-128 chars)
var scenarioIDPattern = regexp.MustCompile(`^[a-z0-9_-]{1,128}$`)

// validateScenarioID ensures the scenario ID is safe for filesystem and command usage
func validateScenarioID(scenarioID string) error {
	if scenarioID == "" {
		return fmt.Errorf("scenario_id is required")
	}
	if !scenarioIDPattern.MatchString(scenarioID) {
		return fmt.Errorf("invalid scenario_id: must contain only lowercase letters, numbers, hyphens, and underscores (1-128 chars)")
	}
	// Additional check: prevent hidden files and parent directory references
	if strings.HasPrefix(scenarioID, ".") || strings.Contains(scenarioID, "..") {
		return fmt.Errorf("invalid scenario_id: cannot start with . or contain ..")
	}
	return nil
}

// isScenarioNotFound checks if a command error indicates a missing scenario
func isScenarioNotFound(output string) bool {
	return strings.Contains(output, "not found") ||
		strings.Contains(output, "does not exist") ||
		strings.Contains(output, "No such scenario") ||
		strings.Contains(output, "No lifecycle log found")
}

// respondNotFound sends a consistent 404 response for missing scenarios
func (s *Server) respondNotFound(w http.ResponseWriter, scenarioID, reason string) {
	s.respondJSON(w, http.StatusNotFound, map[string]interface{}{
		"success": false,
		"message": fmt.Sprintf("Scenario '%s' %s", scenarioID, reason),
	})
}

// handleLifecycleError processes errors from lifecycle commands
func (s *Server) handleLifecycleError(w http.ResponseWriter, operation, scenarioID, output string, err error) {
	if isScenarioNotFound(output) {
		s.respondNotFound(w, scenarioID, "not found")
		return
	}

	s.log(fmt.Sprintf("scenario_%s_failed", operation), map[string]interface{}{
		"scenario_id": scenarioID,
		"error":       err.Error(),
		"output":      output, // Full output in logs for debugging
	})

	// Security: Sanitize output before sending to client
	s.respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
		"success": false,
		"message": fmt.Sprintf("Failed to %s scenario", operation), // Don't leak error details
		"output":  sanitizeCommandOutput(output),
	})
}

// respondLifecycleSuccess sends a consistent success response for lifecycle operations
func (s *Server) respondLifecycleSuccess(w http.ResponseWriter, scenarioID, operation, output string) {
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":     true,
		"message":     fmt.Sprintf("Scenario %s successfully", operation),
		"scenario_id": scenarioID,
		"output":      sanitizeCommandOutput(output), // Security: sanitize before sending to client
	})
}

// scenarioLocation represents where a scenario is located
type scenarioLocation struct {
	Path     string
	Location string // "staging" or "production"
	Found    bool
}

// getVrooliRoot returns the Vrooli root directory path
func getVrooliRoot() string {
	if root := os.Getenv("VROOLI_ROOT"); root != "" {
		return root
	}
	return filepath.Join(os.Getenv("HOME"), "Vrooli")
}

// resolveScenarioPath finds a scenario in staging or production
func resolveScenarioPath(scenarioID string) scenarioLocation {
	vrooliRoot := getVrooliRoot()

	stagingPath := filepath.Join(vrooliRoot, "scenarios", "landing-manager", "generated", scenarioID)
	if _, err := os.Stat(stagingPath); err == nil {
		return scenarioLocation{Path: stagingPath, Location: "staging", Found: true}
	}

	productionPath := filepath.Join(vrooliRoot, "scenarios", scenarioID)
	if _, err := os.Stat(productionPath); err == nil {
		return scenarioLocation{Path: productionPath, Location: "production", Found: true}
	}

	return scenarioLocation{Found: false}
}

// createLifecycleCommand builds a vrooli CLI command for a scenario
func createLifecycleCommand(operation, scenarioID string, loc scenarioLocation, extraArgs ...string) *exec.Cmd {
	args := []string{"scenario", operation, scenarioID}
	if loc.Location == "staging" {
		args = append(args, "--path", loc.Path)
	}
	args = append(args, extraArgs...)
	return exec.Command("vrooli", args...)
}

func (s *Server) handleScenarioStart(w http.ResponseWriter, r *http.Request) {
	scenarioID := mux.Vars(r)["scenario_id"]
	if err := validateScenarioID(scenarioID); err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	loc := resolveScenarioPath(scenarioID)
	if !loc.Found {
		s.respondNotFound(w, scenarioID, "not found in staging or production")
		return
	}

	cmd := createLifecycleCommand("start", scenarioID, loc)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		s.handleLifecycleError(w, "start", scenarioID, outputStr, err)
		return
	}

	s.log("scenario_started", map[string]interface{}{"scenario_id": scenarioID})
	s.respondLifecycleSuccess(w, scenarioID, "started", outputStr)
}

func (s *Server) handleScenarioStop(w http.ResponseWriter, r *http.Request) {
	scenarioID := mux.Vars(r)["scenario_id"]
	if err := validateScenarioID(scenarioID); err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	loc := resolveScenarioPath(scenarioID)
	// Stop is idempotent - if not found, try anyway (CLI handles gracefully)
	var cmd *exec.Cmd
	if loc.Found {
		cmd = createLifecycleCommand("stop", scenarioID, loc)
	} else {
		cmd = exec.Command("vrooli", "scenario", "stop", scenarioID)
	}

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		// Stop is idempotent - these errors mean the scenario can't be running anyway
		if isScenarioNotFound(outputStr) || strings.Contains(outputStr, "Cannot find Vrooli utilities") {
			s.respondJSON(w, http.StatusOK, map[string]interface{}{
				"success": true,
				"message": fmt.Sprintf("Scenario '%s' already stopped or not found", scenarioID),
				"output":  outputStr,
			})
			return
		}

		s.log("scenario_stop_failed", map[string]interface{}{
			"scenario_id": scenarioID,
			"error":       err.Error(),
			"output":      outputStr,
		})
		s.respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to stop scenario: %v", err),
			"output":  outputStr,
		})
		return
	}

	s.log("scenario_stopped", map[string]interface{}{"scenario_id": scenarioID})
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":     true,
		"message":     "Scenario stopped successfully",
		"scenario_id": scenarioID,
		"output":      outputStr,
	})
}

func (s *Server) handleScenarioRestart(w http.ResponseWriter, r *http.Request) {
	scenarioID := mux.Vars(r)["scenario_id"]
	if err := validateScenarioID(scenarioID); err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	loc := resolveScenarioPath(scenarioID)
	if !loc.Found {
		s.respondNotFound(w, scenarioID, "not found in staging or production")
		return
	}

	cmd := createLifecycleCommand("restart", scenarioID, loc)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		s.handleLifecycleError(w, "restart", scenarioID, outputStr, err)
		return
	}

	s.log("scenario_restarted", map[string]interface{}{"scenario_id": scenarioID})
	s.respondLifecycleSuccess(w, scenarioID, "restarted", outputStr)
}

func (s *Server) handleScenarioStatus(w http.ResponseWriter, r *http.Request) {
	scenarioID := mux.Vars(r)["scenario_id"]
	if err := validateScenarioID(scenarioID); err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Resolve scenario path (check staging area first, then production)
	vrooliRoot := getVrooliRoot()

	// Try generated/staging area first
	stagingPath := filepath.Join(vrooliRoot, "scenarios", "landing-manager", "generated", scenarioID)
	productionPath := filepath.Join(vrooliRoot, "scenarios", scenarioID)

	// Check if scenario is in staging area
	if _, err := os.Stat(stagingPath); err == nil {
		// Scenario is in staging area - check process metadata directory directly
		// since vrooli scenario status command uses the API (which doesn't know about staging)
		processDir := filepath.Join(os.Getenv("HOME"), ".vrooli", "processes", "scenarios", scenarioID)
		running := false
		statusText := fmt.Sprintf("Scenario '%s' is in staging area (generated/)", scenarioID)

		// Check if any processes are running for this scenario
		if entries, err := os.ReadDir(processDir); err == nil && len(entries) > 0 {
			// Count active processes by checking PIDs
			activeCount := 0
			for _, entry := range entries {
				if !entry.IsDir() && filepath.Ext(entry.Name()) == ".pid" {
					pidFile := filepath.Join(processDir, entry.Name())
					if pidBytes, err := os.ReadFile(pidFile); err == nil {
						pidStr := strings.TrimSpace(string(pidBytes))
						// Check if process is still running
						checkCmd := exec.Command("kill", "-0", pidStr)
						if checkCmd.Run() == nil {
							activeCount++
						}
					}
				}
			}
			if activeCount > 0 {
				running = true
				statusText = fmt.Sprintf("Scenario '%s' is running in staging area (%d active process(es))", scenarioID, activeCount)
			}
		}

		s.respondJSON(w, http.StatusOK, map[string]interface{}{
			"success":     true,
			"scenario_id": scenarioID,
			"running":     running,
			"status_text": statusText,
			"location":    "staging",
		})
		return
	}

	// Not in staging, check production
	var cmd *exec.Cmd
	if _, err := os.Stat(productionPath); err == nil {
		// Scenario is in production location - use standard status
		cmd = exec.Command("vrooli", "scenario", "status", scenarioID)
	} else {
		// Scenario not found anywhere
		s.respondJSON(w, http.StatusNotFound, map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Scenario '%s' not found", scenarioID),
		})
		return
	}

	output, err := cmd.CombinedOutput()
	statusText := string(output)

	if err != nil {
		// Check if scenario doesn't exist (exit status 1 with "not found" message)
		if isScenarioNotFound(statusText) {
			s.respondJSON(w, http.StatusNotFound, map[string]interface{}{
				"success": false,
				"message": fmt.Sprintf("Scenario '%s' not found", scenarioID),
			})
			return
		}

		// Other errors (permissions, CLI issues, etc.)
		s.log("scenario_status_failed", map[string]interface{}{
			"scenario_id": scenarioID,
			"error":       err.Error(),
		})
		s.respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to get scenario status: %v", err),
		})
		return
	}

	// Parse vrooli scenario status output
	running := strings.Contains(statusText, "🟢 RUNNING") || strings.Contains(statusText, "Status:        🟢 RUNNING")

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":     true,
		"scenario_id": scenarioID,
		"running":     running,
		"status_text": statusText,
		"location":    "production",
	})
}

func (s *Server) handleScenarioLogs(w http.ResponseWriter, r *http.Request) {
	scenarioID := mux.Vars(r)["scenario_id"]
	if err := validateScenarioID(scenarioID); err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	tail := getTailParam(r)
	loc := resolveScenarioPath(scenarioID)
	if !loc.Found {
		s.respondNotFound(w, scenarioID, "not found")
		return
	}

	cmd := createLifecycleCommand("logs", scenarioID, loc, "--tail", tail)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		if isScenarioNotFound(outputStr) {
			s.respondNotFound(w, scenarioID, "not found")
			return
		}

		s.log("scenario_logs_failed", map[string]interface{}{
			"scenario_id": scenarioID,
			"error":       err.Error(),
		})
		s.respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to get scenario logs: %v", err),
		})
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":     true,
		"scenario_id": scenarioID,
		"logs":        outputStr,
	})
}

func getTailParam(r *http.Request) string {
	tail := r.URL.Query().Get("tail")
	if tail == "" {
		return "50"
	}
	// Security: validate tail parameter is a positive integer within reasonable bounds
	if n, err := strconv.Atoi(tail); err != nil || n < 1 || n > 10000 {
		return "50" // fallback to safe default
	}
	return tail
}

// sanitizeCommandOutput removes sensitive information from command output before sending to client
func sanitizeCommandOutput(output string) string {
	// Remove absolute paths that might leak system structure
	output = regexp.MustCompile(`/home/[^/\s]+`).ReplaceAllString(output, "/home/user")
	output = regexp.MustCompile(`/Users/[^/\s]+`).ReplaceAllString(output, "/Users/user")
	// Note: Add more sanitization rules as needed (environment variables, credentials, etc.)
	return output
}

func (s *Server) handleScenarioDelete(w http.ResponseWriter, r *http.Request) {
	scenarioID := mux.Vars(r)["scenario_id"]
	if err := validateScenarioID(scenarioID); err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Only allow deletion from staging area (generated/) - not production
	vrooliRoot := getVrooliRoot()
	stagingPath := filepath.Join(vrooliRoot, "scenarios", "landing-manager", "generated", scenarioID)

	// Check if scenario exists in staging
	if _, err := os.Stat(stagingPath); os.IsNotExist(err) {
		s.respondJSON(w, http.StatusNotFound, map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Scenario '%s' not found in staging area. Only scenarios in generated/ can be deleted through this endpoint.", scenarioID),
		})
		return
	}

	// Stop the scenario if running (idempotent - ignore errors)
	stopCmd := exec.Command("vrooli", "scenario", "stop", scenarioID, "--path", stagingPath)
	if output, err := stopCmd.CombinedOutput(); err != nil {
		s.log("delete_stop_warning", map[string]interface{}{
			"scenario_id": scenarioID,
			"output":      string(output),
			"error":       err.Error(),
		})
		// Continue anyway - scenario might not have been running
	}

	// Remove the scenario directory
	if err := os.RemoveAll(stagingPath); err != nil {
		s.log("delete_failed", map[string]interface{}{
			"scenario_id": scenarioID,
			"path":        stagingPath,
			"error":       err.Error(),
		})
		s.respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to delete scenario: %v", err),
		})
		return
	}

	s.log("scenario_deleted", map[string]interface{}{
		"scenario_id": scenarioID,
		"path":        stagingPath,
	})

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":     true,
		"message":     "Scenario deleted successfully",
		"scenario_id": scenarioID,
	})
}

func (s *Server) handleScenarioPromote(w http.ResponseWriter, r *http.Request) {
	scenarioID := mux.Vars(r)["scenario_id"]
	if err := validateScenarioID(scenarioID); err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Verify scenario exists in generated/ folder
	vrooliRoot := getVrooliRoot()

	generatedPath := filepath.Join(vrooliRoot, "scenarios", "generated", scenarioID)
	productionPath := filepath.Join(vrooliRoot, "scenarios", scenarioID)

	// Check if scenario exists in generated/
	if _, err := os.Stat(generatedPath); os.IsNotExist(err) {
		s.log("promote_failed_not_found", map[string]interface{}{
			"scenario_id": scenarioID,
			"path":        generatedPath,
		})
		s.respondJSON(w, http.StatusNotFound, map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Scenario '%s' not found in staging area (generated/)", scenarioID),
		})
		return
	}

	// Check if production path already exists
	if _, err := os.Stat(productionPath); err == nil {
		s.log("promote_failed_conflict", map[string]interface{}{
			"scenario_id": scenarioID,
			"path":        productionPath,
		})
		s.respondJSON(w, http.StatusConflict, map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Scenario '%s' already exists in production. Delete or rename it first.", scenarioID),
		})
		return
	}

	// Stop the scenario before moving
	stopCmd := exec.Command("vrooli", "scenario", "stop", scenarioID)
	if output, err := stopCmd.CombinedOutput(); err != nil {
		s.log("promote_stop_warning", map[string]interface{}{
			"scenario_id": scenarioID,
			"output":      string(output),
			"error":       err.Error(),
		})
		// Continue anyway - scenario might not have been running
	}

	// Move the scenario directory
	if err := os.Rename(generatedPath, productionPath); err != nil {
		s.log("promote_failed_move", map[string]interface{}{
			"scenario_id": scenarioID,
			"error":       err.Error(),
		})
		s.respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to move scenario to production: %v", err),
		})
		return
	}

	s.log("scenario_promoted", map[string]interface{}{
		"scenario_id": scenarioID,
		"from":        generatedPath,
		"to":          productionPath,
	})

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":         true,
		"message":         "Scenario promoted to production successfully",
		"scenario_id":     scenarioID,
		"production_path": productionPath,
	})
}
