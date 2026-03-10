package main

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// FixGoCliWorkspaceIndependence fixes missing replace/require directives in CLI go.mod files.
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

	text := string(raw)
	originalText := text
	var changes []FixChange

	// Check for API internal imports needing replace+require wiring.
	scenarioDir := filepath.Join(repoRoot, "scenarios", scenarioName)
	apiGoMod := filepath.Join(scenarioDir, "api", "go.mod")
	if fileExists(apiGoMod) {
		apiModule := parseGoModuleLine(apiGoMod)
		if apiModule != "" && cliImportsAPI(filepath.Join(scenarioDir, "cli"), apiModule) {
			replaceDirective := fmt.Sprintf("replace %s => ../api", apiModule)
			if !strings.Contains(text, replaceDirective) {
				text = addGoModReplace(text, apiModule, "../api")
				changes = append(changes, FixChange{
					Type:   "added_replace",
					Detail: fmt.Sprintf("Added replace %s => ../api", apiModule),
				})
			}
			if !goModHasRequire(text, apiModule) {
				text = addGoModRequire(text, apiModule, "v0.0.0")
				changes = append(changes, FixChange{
					Type:   "added_require",
					Detail: fmt.Sprintf("Added require %s v0.0.0", apiModule),
				})
			}
		}
	}

	// Check for proto dependency needing local replace.
	const protoModule = "github.com/vrooli/vrooli/packages/proto"
	if strings.Contains(text, protoModule) && !strings.Contains(text, "replace "+protoModule+" =>") {
		// Calculate relative path from cli/ to packages/proto.
		cliDir := filepath.Join(scenarioDir, "cli")
		protoDir := filepath.Join(repoRoot, "packages", "proto")
		relPath, err := filepath.Rel(cliDir, protoDir)
		if err != nil {
			relPath = "../../../packages/proto"
		}
		text = addGoModReplace(text, protoModule, relPath)
		changes = append(changes, FixChange{
			Type:   "added_replace",
			Detail: fmt.Sprintf("Added replace %s => %s", protoModule, relPath),
		})
	}

	if len(changes) == 0 {
		return []FixResult{{
			ScenarioName: scenarioName,
			RuleID:       ruleID,
			Fixed:        false,
			FilePath:     cliGoMod,
		}}
	}

	var diff *FileDiff
	if dryRun {
		diff = &FileDiff{Before: originalText, After: text}
	} else {
		if err := os.WriteFile(cliGoMod, []byte(text), 0o644); err != nil {
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

// cliImportsAPI checks if any Go files under cliDir import the given apiModule.
func cliImportsAPI(cliDir, apiModule string) bool {
	found := false
	_ = filepath.WalkDir(cliDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if bytes.Contains(b, []byte(`"`+apiModule)) {
			found = true
		}
		return nil
	})
	return found
}

// goModHasRequire checks if go.mod text contains a require for the given module.
func goModHasRequire(text, module string) bool {
	// Check single-line: require <module> <version>
	if strings.Contains(text, "require "+module+" ") {
		return true
	}
	// Check block: \t<module> <version> inside require ( ... )
	if strings.Contains(text, "\t"+module+" ") {
		return true
	}
	return false
}

// addGoModReplace adds a replace directive to go.mod text.
func addGoModReplace(text, module, replacement string) string {
	directive := fmt.Sprintf("replace %s => %s", module, replacement)

	// Try to add inside existing replace ( ... ) block.
	idx := strings.Index(text, "replace (")
	if idx != -1 {
		// Find the closing paren.
		closeIdx := strings.Index(text[idx:], "\n)")
		if closeIdx != -1 {
			insertAt := idx + closeIdx
			text = text[:insertAt] + "\n\t" + module + " => " + replacement + text[insertAt:]
			return text
		}
	}

	// No replace block; append single-line directive at end.
	text = strings.TrimRight(text, "\n") + "\n\n" + directive + "\n"
	return text
}

// addGoModRequire adds a require directive to go.mod text.
func addGoModRequire(text, module, version string) string {
	// Try to add inside existing require ( ... ) block.
	idx := strings.Index(text, "require (")
	if idx != -1 {
		closeIdx := strings.Index(text[idx:], "\n)")
		if closeIdx != -1 {
			insertAt := idx + closeIdx
			text = text[:insertAt] + "\n\t" + module + " " + version + text[insertAt:]
			return text
		}
	}

	// No require block; append single-line directive.
	text = strings.TrimRight(text, "\n") + "\n\nrequire " + module + " " + version + "\n"
	return text
}
