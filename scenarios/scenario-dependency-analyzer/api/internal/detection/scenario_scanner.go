package detection

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	types "scenario-dependency-analyzer/internal/types"
)

// scenario_scanner.go - Scenario dependency detection
//
// This file contains the logic for scanning scenario files to detect
// inter-scenario dependencies through CLI calls, port resolutions, and
// shared workflow references.

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

		ext := strings.ToLower(filepath.Ext(path))
		if !contains(scenarioDetectionExtensions, ext) {
			return nil
		}

		content, err := os.ReadFile(path)
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
	aliasCatalog := s.buildAliasCatalog(scenarioPath)

	err := s.walkCodeFiles(scenarioPath, func(rel string, content []byte) error {
		deps := s.scanFile(string(content), rel, scenarioName, normalizedScenario, aliasCatalog)
		dependencies = append(dependencies, deps...)
		return nil
	})

	return dependencies, err
}

// scanFile scans a single file for scenario references
func (s *scenarioScanner) scanFile(
	content, relPath, scenarioName, normalizedScenario string,
	aliasCatalog map[string]string,
) []types.ScenarioDependency {
	var dependencies []types.ScenarioDependency

	// Detect "vrooli scenario run/test/status <name>" patterns
	dependencies = append(dependencies, s.detectVrooliCommands(content, relPath, scenarioName, normalizedScenario)...)

	// Detect CLI script references (e.g., "chart-generator-cli.sh")
	dependencies = append(dependencies, s.detectCLIReferences(content, relPath, scenarioName, normalizedScenario)...)

	// Detect scenario port resolution calls
	dependencies = append(dependencies, s.detectPortCalls(content, relPath, scenarioName, normalizedScenario, aliasCatalog)...)

	return dependencies
}

// detectVrooliCommands finds "vrooli scenario ..." command usage
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

// detectCLIReferences finds scenario CLI script references
func (s *scenarioScanner) detectCLIReferences(
	content, relPath, scenarioName, normalizedScenario string,
) []types.ScenarioDependency {
	var dependencies []types.ScenarioDependency

	for _, match := range cliScenarioPattern.FindAllStringSubmatch(content, -1) {
		ref := ""
		if match[1] != "" {
			ref = match[1]
		} else if match[2] != "" {
			ref = match[2]
		}

		ref = normalizeName(ref)

		// Skip empty, self-references, and unknown scenarios
		if ref == "" || ref == normalizedScenario || !s.catalog.isKnownScenario(ref) {
			continue
		}

		dependencies = append(dependencies, newScenarioDependency(
			scenarioName,
			ref,
			"CLI reference in "+filepath.Base(relPath),
			"direct_cli",
			relPath,
		))
	}

	return dependencies
}

// detectPortCalls finds scenario port resolution calls
func (s *scenarioScanner) detectPortCalls(
	content, relPath, scenarioName, normalizedScenario string,
	aliasCatalog map[string]string,
) []types.ScenarioDependency {
	var dependencies []types.ScenarioDependency

	for _, match := range scenarioPortCallPattern.FindAllStringSubmatch(content, -1) {
		depName := ""

		// Extract scenario name from direct string or variable
		if len(match) > 1 && match[1] != "" {
			depName = normalizeName(match[1])
		} else if len(match) > 2 && match[2] != "" {
			// Try to resolve variable name via alias catalog
			if alias, ok := aliasCatalog[match[2]]; ok {
				depName = alias
			} else {
				depName = normalizeName(match[2])
			}
		}

		// Skip empty, self-references, and unknown scenarios
		if depName == "" || depName == normalizedScenario || !s.catalog.isKnownScenario(depName) {
			continue
		}

		dependencies = append(dependencies, newScenarioDependency(
			scenarioName,
			depName,
			"References "+depName+" port via CLI",
			"scenario_port_cli",
			relPath,
		))
	}

	return dependencies
}

// scanWorkflows scans for shared workflow references in initialization files
func (s *scenarioScanner) scanWorkflows(scenarioPath, scenarioName string) ([]types.ScenarioDependency, error) {
	var dependencies []types.ScenarioDependency

	initPath := filepath.Join(scenarioPath, "initialization")
	if _, err := os.Stat(initPath); os.IsNotExist(err) {
		return dependencies, nil
	}

	err := filepath.WalkDir(initPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}

		// Only look at JSON files
		if !strings.HasSuffix(path, ".json") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		// Find shared workflow references
		matches := sharedWorkflowPattern.FindAllStringSubmatch(string(content), -1)
		for _, match := range matches {
			if len(match) > 1 {
				dep := newScenarioDependency(
					scenarioName,
					match[1],
					"Shared workflow dependency",
					"workflow_trigger",
					strings.TrimPrefix(path, scenarioPath),
				)
				dep.DependencyType = "shared_workflow"
				dependencies = append(dependencies, dep)
			}
		}

		return nil
	})

	return dependencies, err
}

// buildAliasCatalog scans for variable declarations that map to scenario names
// This helps resolve indirect references like: `const API_NAME = "chart-generator"`
func (s *scenarioScanner) buildAliasCatalog(scenarioPath string) map[string]string {
	aliases := map[string]string{}

	addAlias := func(identifier, scenario string) {
		if identifier == "" {
			return
		}

		normalized := normalizeName(scenario)
		if !s.catalog.isKnownScenario(normalized) {
			return
		}

		aliases[identifier] = normalized
	}

	aliasPatterns := []*regexp.Regexp{
		scenarioAliasDeclPattern,
		scenarioAliasShortPattern,
		scenarioAliasBlockPattern,
	}

	scanForAliases := func(content string) {
		for _, pattern := range aliasPatterns {
			for _, match := range pattern.FindAllStringSubmatch(content, -1) {
				if len(match) >= 3 {
					addAlias(match[1], match[2])
				}
			}
		}
	}

	// Walk the directory to find alias declarations
	_ = s.walkCodeFiles(scenarioPath, func(_ string, content []byte) error {
		scanForAliases(string(content))
		return nil
	})

	return aliases
}
