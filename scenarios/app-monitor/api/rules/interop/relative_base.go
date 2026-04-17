/*
Rule: Vite Relative Base Path
ID: interop_relative_base
Description: Ensures the Vite config sets base to './' so that built assets
  use relative URLs and can be served from any subdirectory or
  iframe origin without broken paths.
Why: When a scenario UI is embedded inside the Vrooli dashboard via an
  iframe, the serving path is not the domain root. An absolute base
  (the Vite default '/') causes asset requests to hit the wrong URL.
  Setting base: './' makes all asset references relative to the HTML
  file's location.
Category: interop
Severity: critical
Slot: [B]
SlotFile: ui/vite.config.ts
TechStack: Vite
Recommendation: In your vite.config.ts, add `base: './'` to the config object
  exported by defineConfig.
Standard: vrooli-ui-interop-v1

GoodExample:
    import { defineConfig } from 'vite';
    export default defineConfig({
      base: './',
      plugins: [react()],
    });

BadExample:
    import { defineConfig } from 'vite';
    export default defineConfig({
      base: '/my-app/',
      plugins: [react()],
    });

<test-case id="relative-base-correct" should-fail="false">
  <description>vite.config.ts has base set to './'</description>
  <input>
    [ui/vite.config.ts]
    import { defineConfig } from 'vite';
    import react from '@vitejs/plugin-react';
    export default defineConfig({
      base: './',
      plugins: [react()],
    });
  </input>
</test-case>

<test-case id="relative-base-wrong-value" should-fail="true">
  <description>vite.config.ts has base set to absolute path</description>
  <input>
    [ui/vite.config.ts]
    import { defineConfig } from 'vite';
    export default defineConfig({
      base: '/my-app/',
      plugins: [react()],
    });
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>has base set but not to './'</expected-message>
</test-case>

<test-case id="relative-base-missing" should-fail="true">
  <description>vite.config.ts has no base config at all</description>
  <input>
    [ui/vite.config.ts]
    import { defineConfig } from 'vite';
    export default defineConfig({
      plugins: [react()],
    });
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>no base config found</expected-message>
</test-case>
*/

package interop

import (
	"app-monitor-api/rules"
	"os"
	"path/filepath"
	"regexp"
)

var (
	relativeBasePattern = regexp.MustCompile(`base\s*:\s*['"]\.\/['"]`)
	anyBasePattern      = regexp.MustCompile(`base\s*:\s*`)
)

func init() {
	rules.Register("interop_relative_base", checkRelativeBase)
}

func checkRelativeBase(ctx rules.CheckContext) rules.RuleResult {
	const ruleID = "interop_relative_base"

	// Try vite.config.ts first, then .js fallback.
	var content string
	var relPath string
	for _, name := range []string{"vite.config.ts", "vite.config.js"} {
		p := filepath.Join(ctx.ScenarioRoot, "ui", name)
		data, err := os.ReadFile(p)
		if err == nil {
			content = string(data)
			relPath = "ui/" + name
			break
		}
	}

	if content == "" {
		return rules.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "no vite.config.ts or vite.config.js found in ui/",
			Message:    "no Vite config found; skipping base path check",
		}
	}

	// Check for correct relative base.
	if relativeBasePattern.MatchString(content) {
		return rules.RuleResult{
			RuleID:  ruleID,
			Passed:  true,
			Message: "base: './' found in " + relPath,
		}
	}

	// Check if base is set to something else.
	if anyBasePattern.MatchString(content) {
		line := lineOf(content, "base")
		return rules.RuleResult{
			RuleID:  ruleID,
			Passed:  false,
			Message: relPath + " has base set but not to './'",
			Violations: []rules.Violation{{
				RuleID:         ruleID,
				Severity:       "critical",
				Title:          "Incorrect Vite base path",
				Description:    relPath + " has base set but not to './'",
				FilePath:       relPath,
				Line:           line,
				Recommendation: "Change base to './' in your Vite config",
			}},
		}
	}

	return rules.RuleResult{
		RuleID:  ruleID,
		Passed:  false,
		Message: relPath + ": no base config found",
		Violations: []rules.Violation{{
			RuleID:         ruleID,
			Severity:       "critical",
			Title:          "Missing Vite base path",
			Description:    "no base config found in " + relPath,
			FilePath:       relPath,
			Recommendation: "Add `base: './'` to your Vite config",
		}},
	}
}
