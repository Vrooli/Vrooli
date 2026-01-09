package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"landing-manager/util"

	"github.com/vrooli/api-core/discovery"
)

// PortResolverFunc is a function type for resolving scenario ports.
// This allows tests to inject mock port resolution.
type PortResolverFunc func(ctx context.Context, scenario, portName string) (int, error)

// PreviewService handles preview link generation for generated scenarios.
// It uses PortResolverFunc as a seam for port resolution, enabling testability.
type PreviewService struct {
	// portResolver is the seam for resolving scenario ports.
	// If nil, uses discovery.ResolveScenarioPort (production behavior).
	portResolver PortResolverFunc
}

// NewPreviewService creates a new preview service using discovery for port resolution
func NewPreviewService() *PreviewService {
	return &PreviewService{
		portResolver: nil, // will use discovery.ResolveScenarioPort
	}
}

// NewPreviewServiceWithResolver creates a preview service with a custom port resolver.
// This constructor is the seam for testing - pass a mock resolver to avoid
// calling the actual discovery system.
func NewPreviewServiceWithResolver(resolver PortResolverFunc) *PreviewService {
	return &PreviewService{
		portResolver: resolver,
	}
}

// resolvePort returns the port for a scenario, using discovery by default
func (ps *PreviewService) resolvePort(ctx context.Context, scenario, portName string) (int, error) {
	if ps.portResolver != nil {
		return ps.portResolver(ctx, scenario, portName)
	}
	return discovery.ResolveScenarioPort(ctx, scenario, portName)
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

	ctx := context.Background()

	// Get actual allocated UI_PORT via discovery
	uiPort, err := ps.resolvePort(ctx, scenarioID, "UI_PORT")
	if err != nil {
		return nil, fmt.Errorf("failed to get UI_PORT for scenario %s (is it running?): %w", scenarioID, err)
	}

	// Get API_PORT as well for proxy support
	apiPort, _ := ps.resolvePort(ctx, scenarioID, "API_PORT")

	baseURL := fmt.Sprintf("http://localhost:%d", uiPort)

	// Convert apiPort to string for response (0 means not found)
	apiPortStr := ""
	if apiPort > 0 {
		apiPortStr = fmt.Sprintf("%d", apiPort)
	}

	return map[string]interface{}{
		"scenario_id": scenarioID,
		"path":        scenarioPath,
		"base_url":    baseURL,
		"ui_port":     fmt.Sprintf("%d", uiPort),
		"api_port":    apiPortStr,
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
