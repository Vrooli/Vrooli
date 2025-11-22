package detection

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	types "scenario-dependency-analyzer/internal/types"
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

// Slice utilities

// toStringSlice safely converts various types to a string slice
func toStringSlice(value interface{}) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []interface{}:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	default:
		return nil
	}
}

// mergeInitializationFiles merges two lists of initialization files, removing duplicates
func mergeInitializationFiles(existing interface{}, additions []string) []string {
	if len(additions) == 0 {
		return toStringSlice(existing)
	}

	set := map[string]struct{}{}
	merged := make([]string, 0)

	// Add existing items
	for _, item := range toStringSlice(existing) {
		if _, ok := set[item]; ok {
			continue
		}
		set[item] = struct{}{}
		merged = append(merged, item)
	}

	// Add new items
	for _, add := range additions {
		if add == "" {
			continue
		}
		if _, ok := set[add]; ok {
			continue
		}
		set[add] = struct{}{}
		merged = append(merged, add)
	}

	return merged
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

// resolvedResourceMap extracts the resource map from a service config,
// handling both old and new config formats
func resolvedResourceMap(cfg *types.ServiceConfig) map[string]types.Resource {
	if cfg == nil {
		return map[string]types.Resource{}
	}
	if cfg.Dependencies.Resources != nil && len(cfg.Dependencies.Resources) > 0 {
		return cfg.Dependencies.Resources
	}
	if cfg.Resources == nil {
		return map[string]types.Resource{}
	}
	return cfg.Resources
}

// extractInitializationFiles extracts file paths from initialization entries
func extractInitializationFiles(entries []map[string]interface{}) []string {
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		if file, ok := entry["file"].(string); ok && file != "" {
			files = append(files, file)
		}
	}
	return files
}
