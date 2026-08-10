package detection

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	types "scenario-dependency-analyzer/internal/types"
)

// resource_scanner.go - Resource dependency detection
//
// This file contains the logic for scanning scenario files to detect
// resource dependencies through CLI commands and heuristics.

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
		if entry.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		// Only scan relevant file types
		ext := strings.ToLower(filepath.Ext(path))
		if !contains(resourceDetectionExtensions, ext) {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path) // #nosec G304,G122 -- path comes from WalkDir under scenarioPath and symlink entries are skipped.
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

// detectResourceHeuristics is an interim resource signal retained until resource
// usage is delegated to code-facts/go-code-graph in the
// scenario-dependency-analyzer-code-evidence-via-ast-facts follow-up.
func (s *resourceScanner) detectResourceHeuristics(content, relPath, scenarioName string, results map[string]types.ScenarioDependency) {
	for resourceName := range s.catalog.getResourceCatalog() {
		for _, pattern := range resourceHeuristicPatterns(resourceName) {
			if !pattern.MatchString(content) {
				continue
			}
			s.recordDetection(results, scenarioName, resourceName, "heuristic", pattern.String(), relPath, "declared-resource")
			break
		}
	}
}

func resourceHeuristicPatterns(resourceName string) []*regexp.Regexp {
	name := regexp.QuoteMeta(normalizeName(resourceName))
	upper := regexp.QuoteMeta(strings.ToUpper(normalizeName(resourceName)))
	return []*regexp.Regexp{
		regexp.MustCompile(`resource-` + name + `\b`),
		regexp.MustCompile(`(?i)\b` + name + `://`),
		regexp.MustCompile(`\b` + upper + `_(?:HOST|URL|API|BASE_URL|ENDPOINT|WEBHOOK|KEY)\b`),
		regexp.MustCompile(`(?i)https?://[^"'\s]*\b` + name + `\b`),
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

	entry := ensureResourceEntry(results, scenarioName, canonical, "Detected via static analysis", method, map[string]interface{}{"source": "detected"})

	// Set resource type
	entry.Configuration["resource_type"] = resourceType

	entry = appendMatch(entry, pattern, method, file)

	results[canonical] = entry
}

func ensureResourceEntry(
	results map[string]types.ScenarioDependency,
	scenarioName, canonical, purpose, method string,
	baseConfig map[string]interface{},
) types.ScenarioDependency {
	if entry, ok := results[canonical]; ok {
		if entry.Configuration == nil {
			entry.Configuration = map[string]interface{}{}
		}
		return entry
	}

	config := map[string]interface{}{}
	for k, v := range baseConfig {
		config[k] = v
	}

	return types.ScenarioDependency{
		ID:             uuid.New().String(),
		ScenarioName:   scenarioName,
		DependencyType: "resource",
		DependencyName: canonical,
		Required:       true,
		Purpose:        purpose,
		AccessMethod:   method,
		Configuration:  config,
		DiscoveredAt:   time.Now(),
		LastVerified:   time.Now(),
	}
}

func appendMatch(entry types.ScenarioDependency, pattern, method, file string) types.ScenarioDependency {
	if entry.Configuration == nil {
		entry.Configuration = map[string]interface{}{}
	}

	matches := existingMatches(entry.Configuration["matches"])
	entry.Configuration["matches"] = append(matches, map[string]interface{}{
		"pattern": pattern,
		"method":  method,
		"file":    file,
	})
	return entry
}

func existingMatches(raw interface{}) []map[string]interface{} {
	if cast, ok := raw.([]map[string]interface{}); ok {
		return cast
	}
	return []map[string]interface{}{}
}
