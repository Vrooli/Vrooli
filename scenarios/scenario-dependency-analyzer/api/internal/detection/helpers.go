package detection

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

// helpers.go - Utility functions and helper methods
//
// This file contains general-purpose utility functions used across
// the detection package.

// String utilities

// normalizeName converts a name to lowercase and trims whitespace for consistent comparison
func normalizeName(name string) string {
	return strings.TrimSpace(strings.ToLower(name))
}

// contains checks if a string slice contains a target string
func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

// Path utilities

// determineScenariosDir resolves the scenarios directory to an absolute path
func determineScenariosDir(dir string) string {
	if dir == "" {
		dir = "../.."
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return dir
	}
	return abs
}

// Catalog discovery

// discoverAvailableScenarios scans a directory and returns all valid scenarios
// (directories containing .vrooli/service.json)
func discoverAvailableScenarios(dir string) map[string]struct{} {
	results := map[string]struct{}{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return results
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		servicePath := filepath.Join(dir, entry.Name(), ".vrooli", "service.json")
		if _, err := os.Stat(servicePath); err == nil {
			results[normalizeName(entry.Name())] = struct{}{}
		}
	}

	return results
}

// discoverAvailableResources scans the resources directory and returns all available resources
func discoverAvailableResources(dir string) map[string]struct{} {
	results := map[string]struct{}{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return results
	}

	for _, entry := range entries {
		if entry.IsDir() {
			results[normalizeName(entry.Name())] = struct{}{}
		}
	}

	return results
}

// Dependency builders

// newScenarioDependency creates a ScenarioDependency struct for scenario-to-scenario edges
func newScenarioDependency(source, target, purpose, method, file string) types.ScenarioDependency {
	return types.ScenarioDependency{
		ID:             uuid.New().String(),
		ScenarioName:   source,
		DependencyType: "scenario",
		DependencyName: target,
		Required:       method == "scenario_port_cli",
		Purpose:        purpose,
		AccessMethod:   method,
		Configuration: map[string]interface{}{
			"found_in_file": file,
		},
		DiscoveredAt: time.Now(),
		LastVerified: time.Now(),
	}
}

// Service config utilities

// resolvedResourceMap extracts the canonical dependency resource map.
func resolvedResourceMap(cfg *types.Manifest) map[string]types.Resource {
	if cfg == nil || cfg.Dependencies.Resources == nil {
		return map[string]types.Resource{}
	}
	return cfg.Dependencies.Resources
}
