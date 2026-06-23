/*
Rule: Accessibility Test Harness
ID: standard_a11y_harness
Description: A UI scenario must ship an automated accessibility harness: an
  axe-core-based dependency (axe-core / jest-axe / vitest-axe) AND at least one
  *.a11y.test.* file that runs axe against rendered components. This is the
  static precondition for the runtime axe/WCAG check group.
Why: Accessibility regressions are invisible in normal development and code
  review — a missing label or a contrast failure looks fine on screen. An
  automated axe sweep in the test suite catches WCAG violations on every run,
  the same way type-checking catches type errors. Without the harness present,
  the runtime a11y assertion has nothing to execute.
Category: accessibility
Severity: medium
Slot: [D]
SlotFile: ui
TechStack: React
Recommendation: Add a dev dependency on axe-core (or jest-axe / vitest-axe) and
  author at least one *.a11y.test.tsx that calls axe.run / toHaveNoViolations
  against your top-level shell. See the react-vite template's
  src/test-utils/a11y.ts.
Standard: vrooli-ui-a11y-v1

GoodExample:
    package.json devDependencies: { "axe-core": "^4.x" }
    ui/src/layout/AppShell.a11y.test.tsx exists

BadExample:
    no axe dependency and no *.a11y.test.* file

<test-case id="a11y-harness-present" should-fail="false">
  <description>axe dependency and an a11y test file are present</description>
  <input>
    [ui/package.json]
    { "devDependencies": { "axe-core": "^4.8.0" } }
    [ui/src/layout/AppShell.a11y.test.tsx]
    test("a11y", () => {});
  </input>
</test-case>

<test-case id="a11y-harness-no-ui" should-fail="false">
  <description>No ui/package.json; a11y harness not applicable</description>
  <input>
    [api/main.go]
    package main
  </input>
</test-case>

<test-case id="a11y-harness-missing" should-fail="true">
  <description>UI present but no axe dependency and no a11y test</description>
  <input>
    [ui/package.json]
    { "devDependencies": { "vitest": "^1.0.0" } }
    [ui/src/App.tsx]
    export function App() { return null; }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>accessibility</expected-message>
</test-case>
*/

package checks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"ui-health/internal/uiinterop"
)

func init() {
	uiinterop.Register("standard_a11y_harness", checkA11yHarness)
}

// a11yDepNames are the accessibility-testing dependencies any one of which
// satisfies the harness requirement.
var a11yDepNames = []string{"axe-core", "jest-axe", "vitest-axe", "@axe-core/react", "@axe-core/playwright"}

func checkA11yHarness(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "standard_a11y_harness"

	pkgPath := filepath.Join(ctx.ScenarioRoot, "ui", "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "ui/package.json not found",
			Message:    "no UI surface; skipping accessibility harness check",
		}
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "ui/package.json unparseable",
			Message:    "ui/package.json unparseable; skipping accessibility harness check",
		}
	}

	hasDep := false
	for _, name := range a11yDepNames {
		if _, ok := pkg.Dependencies[name]; ok {
			hasDep = true
			break
		}
		if _, ok := pkg.DevDependencies[name]; ok {
			hasDep = true
			break
		}
	}
	hasTest := hasA11yTestFile(filepath.Join(ctx.ScenarioRoot, "ui"))

	if hasDep && hasTest {
		return uiinterop.RuleResult{
			RuleID:  ruleID,
			Passed:  true,
			Message: "accessibility harness present (axe dependency + *.a11y.test file)",
		}
	}

	var missing []string
	if !hasDep {
		missing = append(missing, "an axe-based dependency (axe-core/jest-axe/vitest-axe)")
	}
	if !hasTest {
		missing = append(missing, "at least one *.a11y.test.* file")
	}
	return uiinterop.RuleResult{
		RuleID:  ruleID,
		Passed:  false,
		Message: "accessibility test harness incomplete",
		Violations: []uiinterop.Violation{{
			RuleID:         ruleID,
			Severity:       "medium",
			Title:          "Missing accessibility test harness",
			Description:    "UI present but the accessibility harness is incomplete: missing " + strings.Join(missing, " and "),
			FilePath:       "ui/package.json",
			Recommendation: "Add an axe dependency and an *.a11y.test.tsx that asserts no violations on the app shell",
		}},
	}
}

// hasA11yTestFile reports whether any *.a11y.test.* file exists under uiDir.
func hasA11yTestFile(uiDir string) bool {
	found := false
	_ = filepath.WalkDir(uiDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if d.IsDir() {
			if _, skip := skipDirectories[d.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		name := strings.ToLower(d.Name())
		if strings.Contains(name, ".a11y.test.") || strings.Contains(name, ".a11y.spec.") {
			found = true
		}
		return nil
	})
	return found
}
