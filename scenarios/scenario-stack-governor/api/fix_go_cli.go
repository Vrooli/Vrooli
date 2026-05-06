package main

import (
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/mod/modfile"
)

// FixGoCliWorkspaceIndependence fixes missing replace/require directives in scenario go.mod files.
// Uses golang.org/x/mod/modfile for both reads and writes to ensure correct go.mod formatting
// regardless of whitespace, comments, or unusual block layouts.
func FixGoCliWorkspaceIndependence(ctx context.Context, repoRoot, scenarioName string, dryRun bool) []FixResult {
	ruleID := "GO_CLI_WORKSPACE_INDEPENDENCE"

	goMods, err := listScenarioGoModFiles(repoRoot, scenarioName)
	if err != nil || len(goMods) == 0 {
		return nil
	}

	scenarioDir := filepath.Join(repoRoot, "scenarios", scenarioName)
	apiGoMod := filepath.Join(scenarioDir, "api", "go.mod")
	apiModule := ""
	if fileExists(apiGoMod) {
		apiModule = parseGoModModule(apiGoMod)
	}
	defaultGoModPath := filepath.Join(scenarioDir, "cli", "go.mod")
	if !fileExists(defaultGoModPath) && len(goMods) > 0 {
		defaultGoModPath = goMods[0]
	}

	results := make([]FixResult, 0, len(goMods))
	for _, goModPath := range goMods {
		raw, err := os.ReadFile(goModPath)
		if err != nil {
			results = append(results, FixResult{
				ScenarioName: scenarioName,
				RuleID:       ruleID,
				Fixed:        false,
				FilePath:     goModPath,
				Error:        err.Error(),
			})
			continue
		}

		originalText := string(raw)
		var changes []FixChange

		mf, parseErr := modfile.Parse(goModPath, raw, nil)
		if parseErr != nil {
			results = append(results, FixResult{
				ScenarioName: scenarioName,
				RuleID:       ruleID,
				Fixed:        false,
				FilePath:     goModPath,
				Error:        fmt.Sprintf("failed to parse go.mod: %v", parseErr),
			})
			continue
		}

		moduleDir := filepath.Dir(goModPath)

		// Check for CLI imports of the local API module needing replace+require wiring.
		if filepath.Base(moduleDir) == "cli" && apiModule != "" && cliImportsAPI(moduleDir, apiModule, filepath.Join(scenarioDir, "api")) {
			if !goModFileHasReplace(mf, apiModule) {
				if err := mf.AddReplace(apiModule, "", "../api", ""); err == nil {
					changes = append(changes, FixChange{
						Type:   "added_replace",
						Detail: fmt.Sprintf("Added replace %s => ../api", apiModule),
					})
				}
			}
			if !goModFileHasRequire(mf, apiModule) {
				if err := mf.AddRequire(apiModule, "v0.0.0"); err == nil {
					changes = append(changes, FixChange{
						Type:   "added_require",
						Detail: fmt.Sprintf("Added require %s v0.0.0", apiModule),
					})
				}
			}
		}

		if requiresAny(mf, apiCoreModule, cliCoreModule) && !goModFileHasReplace(mf, repoContractModule) {
			relPath, err := filepath.Rel(moduleDir, filepath.Join(repoRoot, "packages", "repo-contract-go"))
			if err != nil {
				relPath = "../../../packages/repo-contract-go"
			}
			if err := mf.AddReplace(repoContractModule, "", relPath, ""); err == nil {
				changes = append(changes, FixChange{
					Type:   "added_replace",
					Detail: fmt.Sprintf("Added replace %s => %s", repoContractModule, relPath),
				})
			}
		}

		if requiresAny(mf, apiCoreModule, cliCoreModule) && !goModFileHasReplace(mf, rootModule) {
			relPath, err := filepath.Rel(moduleDir, repoRoot)
			if err != nil {
				relPath = "../../.."
			}
			if err := mf.AddReplace(rootModule, "", relPath, ""); err == nil {
				changes = append(changes, FixChange{
					Type:   "added_replace",
					Detail: fmt.Sprintf("Added replace %s => %s", rootModule, relPath),
				})
			}
		}

		if goModFileHasRequire(mf, cliCoreModule) && moduleImportsPackage(moduleDir, cliCoreModule+"/cliapp") {
			if !goModFileHasRequire(mf, connectModule) {
				if err := mf.AddRequire(connectModule, connectVersion); err == nil {
					changes = append(changes, FixChange{
						Type:   "added_require",
						Detail: fmt.Sprintf("Added require %s %s", connectModule, connectVersion),
					})
				}
			}
			if !goModFileHasRequire(mf, protobufModule) {
				if err := mf.AddRequire(protobufModule, protobufVersion); err == nil {
					changes = append(changes, FixChange{
						Type:   "added_require",
						Detail: fmt.Sprintf("Added require %s %s", protobufModule, protobufVersion),
					})
				}
			}
		}

		// Check for proto dependency needing local replace.
		// Use the parsed modfile (not raw text) to avoid false positives from comments.
		if goModFileHasRequire(mf, protoModule) && !goModFileHasReplace(mf, protoModule) {
			relPath, err := filepath.Rel(moduleDir, filepath.Join(repoRoot, "packages", "proto"))
			if err != nil {
				relPath = "../../../packages/proto"
			}
			if err := mf.AddReplace(protoModule, "", relPath, ""); err == nil {
				changes = append(changes, FixChange{
					Type:   "added_replace",
					Detail: fmt.Sprintf("Added replace %s => %s", protoModule, relPath),
				})
			}
		}

		if len(changes) == 0 {
			continue
		}

		mf.Cleanup()
		newContent, fmtErr := mf.Format()
		if fmtErr != nil {
			results = append(results, FixResult{
				ScenarioName: scenarioName,
				RuleID:       ruleID,
				Fixed:        false,
				FilePath:     goModPath,
				Error:        fmt.Sprintf("failed to format go.mod: %v", fmtErr),
			})
			continue
		}

		var diff *FileDiff
		if dryRun {
			diff = &FileDiff{Before: originalText, After: string(newContent)}
		} else if err := os.WriteFile(goModPath, newContent, 0o644); err != nil {
			results = append(results, FixResult{
				ScenarioName: scenarioName,
				RuleID:       ruleID,
				Fixed:        false,
				FilePath:     goModPath,
				Error:        err.Error(),
			})
			continue
		}

		results = append(results, FixResult{
			ScenarioName: scenarioName,
			RuleID:       ruleID,
			Fixed:        true,
			FilePath:     goModPath,
			Changes:      changes,
			Diff:         diff,
		})
	}

	if len(results) == 0 && defaultGoModPath != "" {
		return []FixResult{{
			ScenarioName: scenarioName,
			RuleID:       ruleID,
			Fixed:        false,
			FilePath:     defaultGoModPath,
		}}
	}

	return results
}

// cliImportsAPI checks whether CLI Go imports resolve to packages under apiDir.
// Scenario modules may use short paths like "swarm-manager", so prefix matching
// alone is not enough: "swarm-manager/cli/..." is a CLI import, while
// "swarm-manager/internal/..." resolves to the API module.
func cliImportsAPI(cliDir, apiModule, apiDir string) bool {
	apiModule = strings.TrimSuffix(strings.TrimSpace(apiModule), "/")
	if apiModule == "" {
		return false
	}

	found := false
	_ = filepath.WalkDir(cliDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return nil
		}

		for _, spec := range file.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				continue
			}
			if !importPathMatchesModule(importPath, apiModule) {
				continue
			}
			if apiImportResolvesToDir(apiDir, apiModule, importPath) {
				found = true
				return filepath.SkipAll
			}
		}
		return nil
	})
	return found
}

func importPathMatchesModule(importPath, module string) bool {
	return importPath == module || strings.HasPrefix(importPath, module+"/")
}

func apiImportResolvesToDir(apiDir, apiModule, importPath string) bool {
	suffix := strings.TrimPrefix(importPath, apiModule)
	suffix = strings.TrimPrefix(suffix, "/")
	target := apiDir
	if suffix != "" {
		target = filepath.Join(apiDir, filepath.FromSlash(suffix))
	}

	info, err := os.Stat(target)
	return err == nil && info.IsDir()
}

func moduleImportsPackage(moduleDir string, importPaths ...string) bool {
	wanted := make(map[string]struct{}, len(importPaths))
	for _, importPath := range importPaths {
		wanted[importPath] = struct{}{}
	}

	found := false
	_ = filepath.WalkDir(moduleDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return nil
		}

		for _, spec := range file.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				continue
			}
			if _, ok := wanted[importPath]; ok {
				found = true
				return filepath.SkipAll
			}
		}
		return nil
	})
	return found
}
