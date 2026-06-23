/*
Rule: ESLint Stability Rules
ID: standard_eslint_stability
Description: A UI's ESLint config (ui/eslint.config.{js,cjs,mjs,ts}) must enable
  the react-stability safety rules at error severity: react-hooks/rules-of-hooks,
  import/no-cycle, @typescript-eslint/no-explicit-any, and
  @typescript-eslint/no-non-null-assertion.
Why: These four rules catch the highest-frequency UI crash sources that the type
  checker alone misses — hooks called conditionally (state corruption), circular
  imports (undefined-at-module-load), `any` escaping the type system, and `!`
  non-null assertions that paper over real nulls. At "warn" they are ignored; at
  "error" they block the build, which is the only severity that actually prevents
  the regression from shipping.
Category: strict-config
Severity: medium
Slot: [A]
SlotFile: ui/eslint.config.js
TechStack: React
Recommendation: Set react-hooks/rules-of-hooks, import/no-cycle,
  @typescript-eslint/no-explicit-any, and @typescript-eslint/no-non-null-assertion
  to "error" in the ESLint flat config. See the react-vite template's
  eslint.config.js.
Standard: vrooli-ui-strict-v1

GoodExample:
    "react-hooks/rules-of-hooks": "error",
    "import/no-cycle": "error",
    "@typescript-eslint/no-explicit-any": "error",
    "@typescript-eslint/no-non-null-assertion": "error",

BadExample:
    "react-hooks/rules-of-hooks": "warn",   // not enforced
    // no-cycle / no-explicit-any / no-non-null-assertion absent

<test-case id="eslint-stability-all" should-fail="false">
  <description>All stability rules set to error</description>
  <input>
    [ui/eslint.config.js]
    export default [{ rules: {
      "react-hooks/rules-of-hooks": "error",
      "import/no-cycle": "error",
      "@typescript-eslint/no-explicit-any": "error",
      "@typescript-eslint/no-non-null-assertion": "error",
    }}];
  </input>
</test-case>

<test-case id="eslint-stability-no-config" should-fail="false">
  <description>No ESLint config; rule not applicable</description>
  <input>
    [api/main.go]
    package main
  </input>
</test-case>

<test-case id="eslint-stability-missing" should-fail="true">
  <description>Stability rules absent or only at warn</description>
  <input>
    [ui/eslint.config.js]
    export default [{ rules: {
      "react-hooks/rules-of-hooks": "warn",
    }}];
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>no-cycle</expected-message>
</test-case>
*/

package checks

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"ui-health/internal/uiinterop"
)

func init() {
	uiinterop.Register("standard_eslint_stability", checkESLintStability)
}

// eslintConfigCandidates are the flat-config filenames an eslint setup may use.
var eslintConfigCandidates = []string{
	"eslint.config.js",
	"eslint.config.cjs",
	"eslint.config.mjs",
	"eslint.config.ts",
}

// requiredStabilityRules are the react-stability ESLint rules that must be set
// to "error".
var requiredStabilityRules = []string{
	"react-hooks/rules-of-hooks",
	"import/no-cycle",
	"@typescript-eslint/no-explicit-any",
	"@typescript-eslint/no-non-null-assertion",
}

func checkESLintStability(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "standard_eslint_stability"

	rel, content, ok := readESLintConfig(ctx.ScenarioRoot)
	if !ok {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "no ESLint flat config found under ui/",
			Message:    "no ESLint config; skipping stability-rule check",
		}
	}

	var missing []string
	for _, rule := range requiredStabilityRules {
		if !ruleSetToError(content, rule) {
			missing = append(missing, rule)
		}
	}
	if len(missing) == 0 {
		return uiinterop.RuleResult{
			RuleID:  ruleID,
			Passed:  true,
			Message: "all react-stability ESLint rules are set to error",
		}
	}
	return uiinterop.RuleResult{
		RuleID:  ruleID,
		Passed:  false,
		Message: "react-stability ESLint rules not enforced at error: " + strings.Join(missing, ", "),
		Violations: []uiinterop.Violation{{
			RuleID:         ruleID,
			Severity:       "medium",
			Title:          "Stability ESLint rules not enforced",
			Description:    rel + " does not set these rules to error: " + strings.Join(missing, ", "),
			FilePath:       rel,
			Recommendation: `Set each listed rule to "error" in the ESLint flat config`,
		}},
	}
}

// ruleSetToError reports whether content sets the given ESLint rule to "error"
// (single or double quotes, any whitespace around the colon).
func ruleSetToError(content, rule string) bool {
	pattern := regexp.MustCompile(`["']` + regexp.QuoteMeta(rule) + `["']\s*:\s*["']error["']`)
	return pattern.MatchString(content)
}

// readESLintConfig returns the relative path and content of the first ESLint
// flat config found under ui/.
func readESLintConfig(root string) (rel, content string, ok bool) {
	for _, name := range eslintConfigCandidates {
		r := "ui/" + name
		data, err := os.ReadFile(filepath.Join(root, "ui", name))
		if err == nil {
			return r, string(data), true
		}
	}
	return "", "", false
}
