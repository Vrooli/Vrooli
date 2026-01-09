package httpserver

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"test-genie/internal/shared"
	"test-genie/internal/structure"
)

// StructureViolation represents a single structure validation violation.
// This format is designed to be consumed by scenario-auditor's external rules provider.
type StructureViolation struct {
	RuleID         string         `json:"rule_id"`
	Severity       string         `json:"severity"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	FilePath       string         `json:"file_path,omitempty"`
	Recommendation string         `json:"recommendation,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// StructureStandardsResponse is the response format for structure validation.
type StructureStandardsResponse struct {
	EntityName  string               `json:"entity_name"`
	ValidatedAt string               `json:"validated_at"`
	Success     bool                 `json:"success"`
	Violations  []StructureViolation `json:"violations"`
	Summary     string               `json:"summary,omitempty"`
}

// handleStructureStandards validates a scenario's structure and returns violations.
// This endpoint is designed to be called by scenario-auditor's external rules provider.
//
// GET /api/v1/quality/scenario/{name}/structure
func (s *Server) handleStructureStandards(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	name := strings.TrimSpace(params["name"])
	if name == "" {
		s.writeError(w, http.StatusBadRequest, "scenario name is required")
		return
	}

	// Resolve scenario directory
	scenarioDir := s.resolveScenarioDir(name)
	if scenarioDir == "" {
		s.writeError(w, http.StatusNotFound, "scenario not found")
		return
	}

	// Determine schemas directory
	appRoot := getAppRoot()
	schemasDir := filepath.Join(appRoot, "scenarios", "test-genie", "schemas")
	if info, err := os.Stat(schemasDir); err != nil || !info.IsDir() {
		schemasDir = ""
	}

	// Load expectations from .vrooli/testing.json
	expectations, err := structure.LoadExpectations(scenarioDir)
	if err != nil {
		s.log("failed to load structure expectations", map[string]interface{}{
			"error":    err.Error(),
			"scenario": name,
		})
		// Use defaults if loading fails
		expectations = structure.DefaultExpectations()
	}

	// Run structure validation
	runner := structure.New(structure.Config{
		ScenarioDir:  scenarioDir,
		ScenarioName: name,
		SchemasDir:   schemasDir,
		Expectations: expectations,
	}, structure.WithLogger(io.Discard))

	result := runner.Run(r.Context())

	// Convert observations to violations
	violations := observationsToViolations(result.Observations, name)

	response := StructureStandardsResponse{
		EntityName:  name,
		ValidatedAt: time.Now().Format(time.RFC3339),
		Success:     result.Success,
		Violations:  violations,
		Summary:     result.Summary.String(),
	}

	s.writeJSON(w, http.StatusOK, response)
}

// observationsToViolations converts structure observations to the violation format
// expected by scenario-auditor.
func observationsToViolations(observations []structure.Observation, scenarioName string) []StructureViolation {
	var violations []StructureViolation

	for _, obs := range observations {
		// Only convert errors and warnings to violations
		if obs.Type != shared.ObservationError && obs.Type != shared.ObservationWarning {
			continue
		}

		violation := StructureViolation{
			RuleID:      inferRuleID(obs),
			Severity:    observationTypeToSeverity(obs.Type),
			Title:       obs.Message,
			Description: obs.Message,
		}

		// Add recommendation based on the message content
		violation.Recommendation = inferRecommendation(obs.Message)

		violations = append(violations, violation)
	}

	return violations
}

// inferRuleID generates a rule ID from an observation message.
func inferRuleID(obs structure.Observation) string {
	msg := strings.ToLower(obs.Message)

	switch {
	case strings.Contains(msg, "bas/") || strings.Contains(msg, "registry.json"):
		return "structure_bas"
	case strings.Contains(msg, "directory"):
		return "structure_required_dirs"
	case strings.Contains(msg, "file"):
		return "structure_required_files"
	case strings.Contains(msg, "service.json"):
		return "structure_service_json"
	case strings.Contains(msg, "cli"):
		return "structure_cli"
	default:
		return "structure_general"
	}
}

// observationTypeToSeverity maps observation types to severity strings.
func observationTypeToSeverity(obsType shared.ObservationType) string {
	switch obsType {
	case shared.ObservationError:
		return "high"
	case shared.ObservationWarning:
		return "medium"
	default:
		return "low"
	}
}

// inferRecommendation generates a recommendation based on the violation message.
func inferRecommendation(message string) string {
	msg := strings.ToLower(message)

	switch {
	case strings.Contains(msg, "bas/") || strings.Contains(msg, "registry.json"):
		return "Create the bas/ directory with cases/, flows/, actions/, and seeds/ subdirectories. Run 'test-genie registry build' to generate bas/registry.json."
	case strings.Contains(msg, "directory"):
		return "Create the missing directory. See docs/phases/structure/README.md for required layout."
	case strings.Contains(msg, "file"):
		return "Create the missing file. See docs/phases/structure/README.md for required files."
	case strings.Contains(msg, "service.json"):
		return "Ensure .vrooli/service.json exists and is valid. See the service.json schema."
	default:
		return "Review structure requirements in docs/phases/structure/README.md."
	}
}

// getAppRoot returns the Vrooli app root directory.
func getAppRoot() string {
	if root := os.Getenv("VROOLI_ROOT"); root != "" {
		return root
	}
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, "Vrooli")
	}
	return "/home/matthalloran8/Vrooli"
}
