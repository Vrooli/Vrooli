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

// FixGoCliWorkspaceIndependence fixes missing replace/require directives in CLI go.mod files.
// Uses golang.org/x/mod/modfile for both reads and writes to ensure correct go.mod formatting
// regardless of whitespace, comments, or unusual block layouts.
func FixGoCliWorkspaceIndependence(ctx context.Context, repoRoot, scenarioName string, dryRun bool) []FixResult {
	ruleID := "GO_CLI_WORKSPACE_INDEPENDENCE"

	cliGoMod := filepath.Join(repoRoot, "scenarios", scenarioName, "cli", "go.mod")
	if !fileExists(cliGoMod) {
		return nil
	}

	raw, err := os.ReadFile(cliGoMod)
	if err != nil {
		return []FixResult{{
			ScenarioName: scenarioName,
			RuleID:       ruleID,
			Fixed:        false,
			FilePath:     cliGoMod,
			Error:        err.Error(),
		}}
	}

	originalText := string(raw)
	var changes []FixChange

	// Parse the go.mod using modfile for accurate read/write.
	mf, parseErr := modfile.Parse(cliGoMod, raw, nil)
	if parseErr != nil {
		return []FixResult{{
			ScenarioName: scenarioName,
			RuleID:       ruleID,
			Fixed:        false,
			FilePath:     cliGoMod,
			Error:        fmt.Sprintf("failed to parse go.mod: %v", parseErr),
		}}
	}

	// Check for API internal imports needing replace+require wiring.
	scenarioDir := filepath.Join(repoRoot, "scenarios", scenarioName)
	apiGoMod := filepath.Join(scenarioDir, "api", "go.mod")
	if fileExists(apiGoMod) {
		apiModule := parseGoModModule(apiGoMod)
		if apiModule != "" && cliImportsAPI(filepath.Join(scenarioDir, "cli"), apiModule) {
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
	}

	// Check for proto dependency needing local replace.
	const protoModule = "github.com/vrooli/vrooli/packages/proto"
	if strings.Contains(originalText, protoModule) && !goModFileHasReplace(mf, protoModule) {
		// Calculate relative path from cli/ to packages/proto.
		cliDir := filepath.Join(scenarioDir, "cli")
		protoDir := filepath.Join(repoRoot, "packages", "proto")
		relPath, err := filepath.Rel(cliDir, protoDir)
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
		return []FixResult{{
			ScenarioName: scenarioName,
			RuleID:       ruleID,
			Fixed:        false,
			FilePath:     cliGoMod,
		}}
	}

	// Format the modified go.mod using modfile to ensure correct syntax.
	mf.Cleanup()
	newContent, fmtErr := mf.Format()
	if fmtErr != nil {
		return []FixResult{{
			ScenarioName: scenarioName,
			RuleID:       ruleID,
			Fixed:        false,
			FilePath:     cliGoMod,
			Error:        fmt.Sprintf("failed to format go.mod: %v", fmtErr),
		}}
	}

	var diff *FileDiff
	if dryRun {
		diff = &FileDiff{Before: originalText, After: string(newContent)}
	} else {
		if err := os.WriteFile(cliGoMod, newContent, 0o644); err != nil {
			return []FixResult{{
				ScenarioName: scenarioName,
				RuleID:       ruleID,
				Fixed:        false,
				FilePath:     cliGoMod,
				Error:        err.Error(),
			}}
		}
	}

	return []FixResult{{
		ScenarioName: scenarioName,
		RuleID:       ruleID,
		Fixed:        true,
		FilePath:     cliGoMod,
		Changes:      changes,
		Diff:         diff,
	}}
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
