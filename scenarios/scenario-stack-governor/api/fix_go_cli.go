package main

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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
		if filepath.Base(moduleDir) == "cli" && apiModule != "" && cliImportsAPI(moduleDir, apiModule) {
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

// cliImportsAPI checks if any Go files under cliDir import a subpackage of apiModule.
// It looks for `"<apiModule>/` in import statements, which matches imports like
// `"myapi/internal/config"` but not string literals like `appName = "myapi"`.
// This aligns with the rule check which looks for `apiModule+"/internal/"`.
func cliImportsAPI(cliDir, apiModule string) bool {
	// The marker is the module path followed by a slash inside a quoted import,
	// e.g. `"github.com/vrooli/test/api/internal/config"`. This avoids matching
	// bare string literals like `appName = "my-scenario"`.
	marker := []byte(`"` + apiModule + `/`)
	found := false
	_ = filepath.WalkDir(cliDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if bytes.Contains(b, marker) {
			found = true
		}
		return nil
	})
	return found
}
