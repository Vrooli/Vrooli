package checks

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"ui-health/internal/uiinterop"
)

// mainEntryNames lists possible UI entry file names in priority order.
var mainEntryNames = []string{"main.tsx", "main.ts", "main.jsx", "main.js", "index.tsx", "index.ts"}

// serverFileNames lists possible UI server file names.
var serverFileNames = []string{"server.js", "server.ts", "server.mjs", "server.cjs"}

// findMainEntry locates the UI main entry file and returns its content,
// relative path from scenarioRoot, and the absolute path.
// Returns ("", "", nil, error) if not found.
func findMainEntry(scenarioRoot string) (content string, relPath string, data []byte, err error) {
	for _, name := range mainEntryNames {
		p := filepath.Join(scenarioRoot, "ui", "src", name)
		d, readErr := os.ReadFile(p)
		if readErr != nil {
			continue
		}
		return string(d), "ui/src/" + name, d, nil
	}
	return "", "", nil, os.ErrNotExist
}

// lineOf returns the 1-based line number of needle in content, or 0 if not found.
func lineOf(content, needle string) int {
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if strings.Contains(scanner.Text(), needle) {
			return lineNum
		}
	}
	return 0
}

// checkPackageJSONDependency verifies that depName is declared in the scenario's
// ui/package.json (dependencies or devDependencies). It backs the per-dependency
// interop rules (interop_api_base_dep, interop_iframe_bridge_dep), which differ
// only by rule id and package name, so the read/parse/lookup body lives here once.
func checkPackageJSONDependency(ctx uiinterop.CheckContext, ruleID, depName string) uiinterop.RuleResult {
	pkgPath := filepath.Join(ctx.ScenarioRoot, "ui", "package.json")

	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "ui/package.json not found",
			Message:    "ui/package.json not found; skipping dependency check",
		}
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return uiinterop.RuleResult{
			RuleID:  ruleID,
			Passed:  false,
			Message: "failed to parse ui/package.json: " + err.Error(),
			Violations: []uiinterop.Violation{{
				RuleID:         ruleID,
				Severity:       "critical",
				Title:          "Unparseable package.json",
				Description:    "ui/package.json could not be parsed as JSON",
				FilePath:       "ui/package.json",
				Recommendation: "Fix JSON syntax in ui/package.json",
			}},
		}
	}

	if _, ok := pkg.Dependencies[depName]; ok {
		return uiinterop.RuleResult{RuleID: ruleID, Passed: true, Message: depName + " found in dependencies"}
	}
	if _, ok := pkg.DevDependencies[depName]; ok {
		return uiinterop.RuleResult{RuleID: ruleID, Passed: true, Message: depName + " found in devDependencies"}
	}

	return uiinterop.RuleResult{
		RuleID:  ruleID,
		Passed:  false,
		Message: depName + " not found in ui/package.json",
		Violations: []uiinterop.Violation{{
			RuleID:         ruleID,
			Severity:       "critical",
			Title:          "Missing " + depName,
			Description:    depName + " not found in dependencies or devDependencies",
			FilePath:       "ui/package.json",
			Recommendation: "Run `pnpm add " + depName + "` in the ui/ directory",
		}},
	}
}
