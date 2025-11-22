package detection

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	types "scenario-dependency-analyzer/internal/types"
)

// resource_scanner.go - Resource dependency detection
//
// This file contains the logic for scanning scenario files to detect
// resource dependencies through CLI commands, heuristics, and initialization files.

// resourceScanner handles resource dependency detection
type resourceScanner struct {
	catalog *catalogManager
}

// newResourceScanner creates a scanner for detecting resource dependencies
func newResourceScanner(catalog *catalogManager) *resourceScanner {
	return &resourceScanner{
		catalog: catalog,
	}
}

// scan walks the scenario directory and detects all resource dependencies
func (s *resourceScanner) scan(scenarioPath, scenarioName string, cfg *types.ServiceConfig) ([]types.ScenarioDependency, error) {
	results := map[string]types.ScenarioDependency{}

	err := filepath.WalkDir(scenarioPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}

		// Skip directories that should be ignored
		if entry.IsDir() {
			if path != scenarioPath && shouldSkipDirectoryEntry(entry) {
				return filepath.SkipDir
			}
			return nil
		}

		// Only scan relevant file types
		ext := strings.ToLower(filepath.Ext(path))
		if !contains(resourceDetectionExtensions, ext) {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		// Calculate relative path for filtering
		rel, relErr := filepath.Rel(scenarioPath, path)
		if relErr != nil {
			rel = path
		}

		// Skip documentation and test files
		if shouldIgnoreDetectionFile(rel) {
			return nil
		}

		// Scan this file for resource dependencies
		s.scanFile(string(content), rel, scenarioName, results)

		return nil
	})

	// Augment with resources declared in initialization
	if cfg != nil {
		s.augmentWithInitialization(results, scenarioName, cfg)
	}

	// Convert map to sorted slice
	deps := make([]types.ScenarioDependency, 0, len(results))
	for _, dep := range results {
		deps = append(deps, dep)
	}
	sort.Slice(deps, func(i, j int) bool {
		return deps[i].DependencyName < deps[j].DependencyName
	})

	return deps, err
}

// scanFile scans a single file for resource references
func (s *resourceScanner) scanFile(content, relPath, scenarioName string, results map[string]types.ScenarioDependency) {
	// Detect explicit resource CLI commands (e.g., "resource-postgres")
	s.detectResourceCLICommands(content, relPath, scenarioName, results)

	// Detect resources via heuristics (e.g., connection strings, env vars)
	s.detectResourceHeuristics(content, relPath, scenarioName, results)
}

// detectResourceCLICommands finds explicit resource-* CLI command usage
func (s *resourceScanner) detectResourceCLICommands(content, relPath, scenarioName string, results map[string]types.ScenarioDependency) {
	// Only look for CLI commands in allowed directories
	if !isAllowedResourceCLIPath(relPath) {
		return
	}

	matches := resourceCommandPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			resourceName := normalizeName(match[1])
			s.recordDetection(
				results,
				scenarioName,
				resourceName,
				"resource_cli",
				"resource-cli",
				relPath,
				resourceName,
			)
		}
	}
}

// detectResourceHeuristics uses pattern matching to detect resource usage
func (s *resourceScanner) detectResourceHeuristics(content, relPath, scenarioName string, results map[string]types.ScenarioDependency) {
	for _, heuristic := range resourceHeuristicCatalog {
		for _, pattern := range heuristic.Patterns {
			if pattern.MatchString(content) {
				s.recordDetection(
					results,
					scenarioName,
					heuristic.Name,
					"heuristic",
					pattern.String(),
					relPath,
					heuristic.Type,
				)
				break // Only record once per heuristic
			}
		}
	}
}

// recordDetection records a resource dependency detection, merging with existing entries
func (s *resourceScanner) recordDetection(
	results map[string]types.ScenarioDependency,
	scenarioName, name, method, pattern, file, resourceType string,
) {
	canonical := normalizeName(name)

	// Skip if empty or not a known resource
	if canonical == "" || !s.catalog.isKnownResource(canonical) {
		return
	}

	// Get existing entry or create new one
	entry, ok := results[canonical]
	if !ok {
		entry = types.ScenarioDependency{
			ID:             uuid.New().String(),
			ScenarioName:   scenarioName,
			DependencyType: "resource",
			DependencyName: canonical,
			Required:       true,
			Purpose:        "Detected via static analysis",
			AccessMethod:   method,
			Configuration:  map[string]interface{}{"source": "detected"},
			DiscoveredAt:   time.Now(),
			LastVerified:   time.Now(),
		}
	}

	// Ensure configuration map exists
	if entry.Configuration == nil {
		entry.Configuration = map[string]interface{}{}
	}

	// Set resource type
	entry.Configuration["resource_type"] = resourceType

	// Append match information
	match := map[string]interface{}{
		"pattern": pattern,
		"method":  method,
		"file":    file,
	}

	if existing, ok := entry.Configuration["matches"].([]map[string]interface{}); ok {
		entry.Configuration["matches"] = append(existing, match)
	} else {
		entry.Configuration["matches"] = []map[string]interface{}{match}
	}

	results[canonical] = entry
}

// augmentWithInitialization adds resources that have initialization files
func (s *resourceScanner) augmentWithInitialization(
	results map[string]types.ScenarioDependency,
	scenarioName string,
	cfg *types.ServiceConfig,
) {
	resources := resolvedResourceMap(cfg)

	for resourceName, resource := range resources {
		if len(resource.Initialization) == 0 {
			continue
		}

		canonical := normalizeName(resourceName)
		if canonical == "" {
			continue
		}

		files := extractInitializationFiles(resource.Initialization)

		// Get existing entry or create new one
		entry, exists := results[canonical]
		if !exists {
			entry = types.ScenarioDependency{
				ID:             uuid.New().String(),
				ScenarioName:   scenarioName,
				DependencyType: "resource",
				DependencyName: canonical,
				Required:       true,
				Purpose:        "Initialization data references this resource",
				AccessMethod:   "initialization",
				Configuration:  map[string]interface{}{},
				DiscoveredAt:   time.Now(),
				LastVerified:   time.Now(),
			}
		}

		// Update configuration
		if entry.Configuration == nil {
			entry.Configuration = map[string]interface{}{}
		}
		entry.Configuration["initialization_detected"] = true

		if len(files) > 0 {
			entry.Configuration["initialization_files"] = mergeInitializationFiles(
				entry.Configuration["initialization_files"],
				files,
			)
		}

		results[canonical] = entry
	}
}
