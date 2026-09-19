package detection

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

// scenario_scanner.go - Scenario dependency detection
//
// This file contains the logic for scanning scenario files to detect
// inter-scenario dependencies through explicit Vrooli commands and shared
// workflow references.

// scenarioScanner handles scenario-to-scenario dependency detection
type scenarioScanner struct {
	catalog *catalogManager
}

// newScenarioScanner creates a scanner for detecting scenario dependencies
func newScenarioScanner(catalog *catalogManager) *scenarioScanner {
	return &scenarioScanner{
		catalog: catalog,
	}
}

// walkCodeFiles visits code files that are relevant for scenario detection, applying shared
// directory and documentation filters before invoking the provided callback.
func (s *scenarioScanner) walkCodeFiles(
	scenarioPath string,
	visit func(relPath string, content []byte) error,
) error {
	return filepath.WalkDir(scenarioPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}

		if entry.IsDir() {
			if path != scenarioPath && shouldSkipDirectoryEntry(entry) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !contains(scenarioDetectionExtensions, ext) {
			return nil
		}

		content, err := os.ReadFile(path) // #nosec G304,G122 -- path comes from WalkDir under scenarioPath and symlink entries are skipped.
		if err != nil {
			return nil
		}

		rel, relErr := filepath.Rel(scenarioPath, path)
		if relErr != nil {
			rel = path
		}

		if shouldIgnoreDetectionFile(rel) {
			return nil
		}

		return visit(rel, content)
	})
}

// scanDependencies walks the scenario directory and detects scenario-to-scenario dependencies
func (s *scenarioScanner) scanDependencies(scenarioPath, scenarioName string) ([]types.ScenarioDependency, error) {
	var dependencies []types.ScenarioDependency

	normalizedScenario := normalizeName(scenarioName)

	err := s.walkCodeFiles(scenarioPath, func(rel string, content []byte) error {
		deps := s.scanFile(string(content), rel, scenarioName, normalizedScenario)
		dependencies = append(dependencies, deps...)
		return nil
	})

	return dependencies, err
}

// scanFile scans a single file for scenario references
func (s *scenarioScanner) scanFile(
	content, relPath, scenarioName, normalizedScenario string,
) []types.ScenarioDependency {
	var dependencies []types.ScenarioDependency

	dependencies = append(dependencies, s.detectVrooliCommands(content, relPath, scenarioName, normalizedScenario)...)

	return dependencies
}

// detectVrooliCommands is the remaining interim runtime scenario signal. Import-level
// usage is now sourced from code-facts/proto-health through interfacegraph; this
// command-shell signal is pending replacement by AST facts in the
// scenario-dependency-analyzer-code-evidence-via-ast-facts follow-up.
func (s *scenarioScanner) detectVrooliCommands(
	content, relPath, scenarioName, normalizedScenario string,
) []types.ScenarioDependency {
	var dependencies []types.ScenarioDependency

	for _, match := range vrooliScenarioPattern.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			dep := normalizeName(match[1])

			// Skip self-references and unknown scenarios
			if dep == normalizedScenario || !s.catalog.isKnownScenario(dep) {
				continue
			}

			dependencies = append(dependencies, newScenarioDependency(
				scenarioName,
				dep,
				"vrooli scenario",
				"vrooli_cli",
				relPath,
			))
		}
	}

	return dependencies
}
