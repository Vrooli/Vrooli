package detection

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
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
		contract := ext == ".json" && filepath.Dir(path) == filepath.Join(scenarioPath, ".vrooli", "program-runtime")
		if !contains(scenarioDetectionExtensions, ext) && !contract {
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

	if filepath.Ext(relPath) == ".json" {
		return s.detectProgramBindings(content, relPath, scenarioName, normalizedScenario)
	}
	dependencies = append(dependencies, s.detectVrooliCommands(content, relPath, scenarioName, normalizedScenario)...)
	if strings.EqualFold(filepath.Ext(relPath), ".go") {
		dependencies = append(dependencies, s.detectScenarioURLResolvers(content, relPath, scenarioName, normalizedScenario)...)
	}

	return dependencies
}

// A governed program's binding declaration is direct dependency evidence, even
// when its Python companion never resolves a URL or invokes the target CLI.
// Restrict this to canonical contract files rather than searching arbitrary JSON.
func (s *scenarioScanner) detectProgramBindings(content, relPath, scenarioName, normalizedScenario string) []types.ScenarioDependency {
	if filepath.ToSlash(filepath.Dir(relPath)) != ".vrooli/program-runtime" {
		return nil
	}
	var contract struct {
		Name     string `json:"name"`
		Bindings []struct {
			ID string `json:"id"`
		} `json:"bindings"`
	}
	if json.Unmarshal([]byte(content), &contract) != nil || !strings.HasPrefix(contract.Name, scenarioName+".") {
		return nil
	}
	var dependencies []types.ScenarioDependency
	seen := map[string]bool{}
	for _, binding := range contract.Bindings {
		parts := strings.Split(binding.ID, "/")
		if len(parts) < 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
			continue
		}
		target := parts[0]
		if target != normalizeName(target) || target == normalizedScenario || seen[target] || !s.catalog.isKnownScenario(target) {
			continue
		}
		seen[target] = true
		dependencies = append(dependencies, newScenarioDependency(scenarioName, target, "governed program binding", "program_binding", relPath))
	}
	return dependencies
}

// detectScenarioURLResolvers recognizes literal targets passed to the canonical
// api-core discovery resolver. Parsing Go syntax prevents comments, strings,
// and unrelated similarly named calls from becoming dependency evidence.
func (s *scenarioScanner) detectScenarioURLResolvers(content, relPath, scenarioName, normalizedScenario string) []types.ScenarioDependency {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, relPath, content, 0)
	if err != nil {
		return nil
	}

	resolverNames := map[string]struct{}{}
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil || importPath != "github.com/vrooli/api-core/discovery" {
			continue
		}
		name := "discovery"
		if spec.Name != nil {
			name = spec.Name.Name
		}
		if name != "_" && name != "." {
			resolverNames[name] = struct{}{}
		}
	}
	if len(resolverNames) == 0 {
		return nil
	}

	dependencies := []types.ScenarioDependency{}
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) != 2 {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "ResolveScenarioURLDefault" {
			return true
		}
		pkg, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}
		if _, ok := resolverNames[pkg.Name]; !ok {
			return true
		}
		literal, ok := call.Args[1].(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		dependencyName, err := strconv.Unquote(literal.Value)
		if err != nil {
			return true
		}
		dependencyName = normalizeName(dependencyName)
		if dependencyName == "" || dependencyName == normalizedScenario || !s.catalog.isKnownScenario(dependencyName) {
			return true
		}
		dependencies = append(dependencies, newScenarioDependency(scenarioName, dependencyName, "scenario URL discovery", "scenario_url_discovery", relPath))
		return true
	})
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
