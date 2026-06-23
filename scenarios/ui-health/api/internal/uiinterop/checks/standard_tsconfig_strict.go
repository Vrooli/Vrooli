/*
Rule: TypeScript Strict Config
ID: standard_tsconfig_strict
Description: A UI's ui/tsconfig.json must enable TypeScript strict mode
  ("strict": true). Strict mode is the foundation of the react-stability safety
  net (null-safety, implicit-any bans, strict function types) that prevents the
  most common class of runtime UI crashes.
Why: "X is not a function" and "cannot read property Y of undefined" are the #1
  production UI failures, and they are exactly what strict mode catches at
  compile time. A tsconfig with strict disabled (or omitted) silently ships code
  the type checker would otherwise reject. Strict is a single flag that turns a
  whole family of latent crashes into build errors.
Category: strict-config
Severity: high
Slot: [A]
SlotFile: ui/tsconfig.json
TechStack: React
Recommendation: Set "strict": true in ui/tsconfig.json compilerOptions. The
  deterministic fixer flips an explicit "strict": false to true; if the flag is
  absent, add it to compilerOptions. Pair with "noUncheckedIndexedAccess": true
  for array/index null-safety.
Standard: vrooli-ui-strict-v1

GoodExample:
    { "compilerOptions": { "strict": true } }

BadExample:
    { "compilerOptions": { "strict": false } }

<test-case id="tsconfig-strict-true" should-fail="false">
  <description>tsconfig enables strict mode</description>
  <input>
    [ui/tsconfig.json]
    { "compilerOptions": { "strict": true, "noEmit": true } }
  </input>
</test-case>

<test-case id="tsconfig-strict-no-ui" should-fail="false">
  <description>No ui/tsconfig.json; strict-config not applicable</description>
  <input>
    [api/main.go]
    package main
  </input>
</test-case>

<test-case id="tsconfig-strict-false" should-fail="true">
  <description>tsconfig disables strict mode</description>
  <input>
    [ui/tsconfig.json]
    { "compilerOptions": { "strict": false, "noEmit": true } }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>strict</expected-message>
</test-case>

<test-case id="tsconfig-strict-missing" should-fail="true">
  <description>tsconfig omits the strict flag entirely</description>
  <input>
    [ui/tsconfig.json]
    { "compilerOptions": { "noEmit": true } }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>strict</expected-message>
</test-case>
*/

package checks

import (
	"os"
	"path/filepath"
	"regexp"

	"ui-health/internal/uiinterop"
)

func init() {
	uiinterop.Register("standard_tsconfig_strict", checkTSConfigStrict)
}

// strictTruePattern matches "strict": true (any whitespace around the colon).
var strictTruePattern = regexp.MustCompile(`"strict"\s*:\s*true`)

// strictFalsePattern matches "strict": false (the auto-fixable case).
var strictFalsePattern = regexp.MustCompile(`"strict"\s*:\s*false`)

func checkTSConfigStrict(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "standard_tsconfig_strict"
	const tsconfigRel = "ui/tsconfig.json"

	data, err := os.ReadFile(filepath.Join(ctx.ScenarioRoot, filepath.FromSlash(tsconfigRel)))
	if err != nil {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "ui/tsconfig.json not found",
			Message:    "no ui/tsconfig.json; skipping strict-config check",
		}
	}
	content := string(data)

	if strictTruePattern.MatchString(content) {
		return uiinterop.RuleResult{
			RuleID:  ruleID,
			Passed:  true,
			Message: "tsconfig enables strict mode",
		}
	}

	desc := tsconfigRel + " does not enable TypeScript strict mode"
	rec := `Set "strict": true in compilerOptions`
	if strictFalsePattern.MatchString(content) {
		desc = tsconfigRel + ` sets "strict": false — strict mode is disabled`
		rec = `Change "strict": false to "strict": true in compilerOptions (the deterministic fixer can do this)`
	}
	return uiinterop.RuleResult{
		RuleID:  ruleID,
		Passed:  false,
		Message: "tsconfig does not enable strict mode",
		Violations: []uiinterop.Violation{{
			RuleID:         ruleID,
			Severity:       "high",
			Title:          "TypeScript strict mode disabled",
			Description:    desc,
			FilePath:       tsconfigRel,
			Line:           lineOf(content, `"strict"`),
			Recommendation: rec,
		}},
	}
}
