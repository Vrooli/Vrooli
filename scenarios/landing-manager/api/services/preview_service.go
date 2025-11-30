package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"landing-manager/util"
)

// PreviewService handles preview link generation for generated scenarios.
// It uses CommandExecutor as a seam for CLI interactions, enabling testability.
type PreviewService struct {
	// cmdExecutor is the seam for executing CLI commands.
	// If nil, uses util.DefaultCommandExecutor (production behavior).
	cmdExecutor util.CommandExecutor
}

// NewPreviewService creates a new preview service using the default command executor
func NewPreviewService() *PreviewService {
	return &PreviewService{
		cmdExecutor: nil, // will use util.DefaultCommandExecutor
	}
}

// NewPreviewServiceWithExecutor creates a preview service with a custom command executor.
// This constructor is the seam for testing - pass a MockCommandExecutor to avoid
// shelling out to the actual vrooli CLI.
func NewPreviewServiceWithExecutor(executor util.CommandExecutor) *PreviewService {
	return &PreviewService{
		cmdExecutor: executor,
	}
}

// executor returns the command executor, defaulting to the global if none set
func (ps *PreviewService) executor() util.CommandExecutor {
	if ps.cmdExecutor != nil {
		return ps.cmdExecutor
	}
	return util.DefaultCommandExecutor
}

// GetPreviewLinks generates preview URLs for a generated scenario.
// Returns deep links to the public landing page and admin portal.
func (ps *PreviewService) GetPreviewLinks(scenarioID string) (map[string]interface{}, error) {
	root, err := util.GenerationRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve generation root: %w", err)
	}

	scenarioPath := filepath.Join(root, scenarioID)
	if _, err := os.Stat(scenarioPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("generated scenario not found: %s", scenarioID)
	}

	// Get actual allocated UI_PORT from vrooli CLI via the CommandExecutor seam
	result := ps.executor().Execute("vrooli", "scenario", "port", scenarioID, "UI_PORT")
	if result.Err != nil {
		return nil, fmt.Errorf("failed to get UI_PORT for scenario %s (is it running?): %w", scenarioID, result.Err)
	}

	uiPort := strings.TrimSpace(result.Output)
	if uiPort == "" {
		return nil, fmt.Errorf("UI_PORT not allocated for scenario %s (scenario may not be running)", scenarioID)
	}

	// Get API_PORT as well for proxy support
	apiResult := ps.executor().Execute("vrooli", "scenario", "port", scenarioID, "API_PORT")
	apiPort := ""
	if apiResult.Err == nil {
		apiPort = strings.TrimSpace(apiResult.Output)
	}

	baseURL := fmt.Sprintf("http://localhost:%s", uiPort)

	return map[string]interface{}{
		"scenario_id": scenarioID,
		"path":        scenarioPath,
		"base_url":    baseURL,
		"ui_port":     uiPort,
		"api_port":    apiPort,
		"links": map[string]string{
			"public":      baseURL + "/",
			"admin":       baseURL + "/admin",
			"admin_login": baseURL + "/admin/login",
			"health":      baseURL + "/health",
		},
		"instructions": []string{
			fmt.Sprintf("Start the scenario: vrooli scenario start %s", scenarioID),
			fmt.Sprintf("Public landing page: %s/", baseURL),
			fmt.Sprintf("Admin portal: %s/admin", baseURL),
			"Note: Scenario must be running for preview links to work",
		},
		"notes": "Scenario can be started from staging area (generated/) - no need to promote first",
	}, nil
}
