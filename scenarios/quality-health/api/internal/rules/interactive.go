package rules

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	interactiveTTYPattern    = regexp.MustCompile(`(/dev/tty|term\.ReadPassword|ReadPassword\s*\()`)
	interactiveReaderPattern = regexp.MustCompile(`bufio\.NewReader\s*\(\s*os\.Stdin\s*\)|fmt\.(Fscan|Scanln?)\s*\([^\n]*os\.Stdin`)
)

// evalScenarioInteractiveBoundary is intentionally conservative about what it
// calls interactive. A plain io.ReadAll(os.Stdin) is a supported automation
// channel; terminal acquisition, password masking, and prompt-shaped readers
// are the regressions this gate prevents.
func evalScenarioInteractiveBoundary(ctx EvalContext) []Finding {
	rule, _ := ByID(RuleScenarioInteractiveBoundary)
	root := ctx.Surface.RootPath
	if root == "" {
		root = ctx.Inventory.RootPath
	}
	if root == "" {
		return nil
	}
	var findings []Finding
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry == nil {
			return nil
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "vendor", "node_modules", "dist", "build", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		// A surface can contain both sanctioned and unsanctioned domains (the
		// Bridge CLI is the canonical example). Exempt the sanctioned file
		// trees at the point of inspection instead of exempting the whole scan
		// root, so unrelated prompts remain visible to this gate.
		if interactiveSurfaceExempt(path) {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		text := string(data)
		match := interactiveTTYPattern.FindString(text)
		if match == "" {
			match = interactiveReaderPattern.FindString(text)
		}
		if match == "" {
			return nil
		}
		findings = append(findings, ruleFinding(ctx, rule, path,
			"interactive operator input is outside a sanctioned surface",
			match,
			"operator questions are emitted as typed requests for onboarding to resolve",
			match+"; move the intake to vrooli-onboarding or the Bridge operator intake"))
		return nil
	})
	return findings
}

func interactiveSurfaceExempt(path string) bool {
	clean := filepath.ToSlash(filepath.Clean(path))
	// The detector's own pattern definitions mention the forbidden APIs as
	// literals. Do not report the rule implementation while scanning a target
	// scenario.
	if strings.Contains(clean, "/scenarios/quality-health/api/internal/rules/") {
		return true
	}
	if strings.Contains(clean, "/scenarios/vrooli-onboarding/") {
		return true
	}
	// Bridge has two explicit post-bootstrap credential surfaces: SSH
	// onboarding and authenticated Bridge login. Keep the exemption at the
	// domain boundary so a new unrelated CLI prompt cannot hide in the tree.
	return strings.Contains(clean, "/scenarios/vrooli-bridge/cli/domains/onboard/") ||
		strings.Contains(clean, "/scenarios/vrooli-bridge/cli/domains/auth/")
}
