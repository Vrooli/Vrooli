/*
Rule: Protective Comments
ID: interop_protective_comments
Description: Checks that INTEROP-CRITICAL comments are present in both the
  Vite config and the main entry file to warn future developers not to
  remove interop-sensitive configuration.
Why: Interop settings in vite.config and the main entry (iframe-bridge
  init, base path resolution) are easy to accidentally delete during
  refactoring. A clearly marked INTEROP-CRITICAL comment acts as a
  guardrail, signaling that the annotated code is required for the
  scenario to work inside the Vrooli host frame.
Category: interop
Severity: low
Slot: [B],[D]
SlotFile: ui/vite.config.ts, ui/src/main.tsx
TechStack: Vite, iframe-bridge
Recommendation: Add // INTEROP-CRITICAL comments above interop-sensitive
  code in both vite.config.ts and the main entry file.
Standard: vrooli-ui-interop-v1

GoodExample:
    // vite.config.ts
    export default defineConfig({
      // INTEROP-CRITICAL: base must use VROOLI_BASE for sub-path mounting
      base: process.env.VROOLI_BASE || "/",
    });

    // main.tsx
    // INTEROP-CRITICAL: iframe-bridge must init before React render
    import { initBridge } from "@vrooli/iframe-bridge";
    initBridge();

BadExample:
    // vite.config.ts — no INTEROP-CRITICAL comment
    export default defineConfig({
      base: process.env.VROOLI_BASE || "/",
    });

    // main.tsx — no INTEROP-CRITICAL comment
    import { initBridge } from "@vrooli/iframe-bridge";
    initBridge();

<test-case id="protective-both-present" should-fail="false">
  <description>Both vite config and main entry have INTEROP-CRITICAL</description>
  <input>
    [ui/vite.config.ts]
    // INTEROP-CRITICAL: base path for sub-path mounting
    export default defineConfig({ base: "/" });
    [ui/src/main.tsx]
    // INTEROP-CRITICAL: iframe-bridge init
    import { initBridge } from "@vrooli/iframe-bridge";
    initBridge();
  </input>
</test-case>

<test-case id="protective-vite-js-variant" should-fail="false">
  <description>vite.config.js (not .ts) has INTEROP-CRITICAL along with main entry</description>
  <input>
    [ui/vite.config.js]
    // INTEROP-CRITICAL: base path
    export default defineConfig({ base: "/" });
    [ui/src/main.tsx]
    // INTEROP-CRITICAL: bridge init
    initBridge();
  </input>
</test-case>

<test-case id="protective-missing-in-vite" should-fail="true">
  <description>Main entry has comment but vite config does not</description>
  <input>
    [ui/vite.config.ts]
    export default defineConfig({ base: "/" });
    [ui/src/main.tsx]
    // INTEROP-CRITICAL: bridge init
    initBridge();
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>vite config</expected-message>
</test-case>

<test-case id="protective-missing-in-main" should-fail="true">
  <description>Vite config has comment but main entry does not</description>
  <input>
    [ui/vite.config.ts]
    // INTEROP-CRITICAL: base path
    export default defineConfig({ base: "/" });
    [ui/src/main.tsx]
    import { initBridge } from "@vrooli/iframe-bridge";
    initBridge();
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>main entry</expected-message>
</test-case>

<test-case id="protective-missing-both" should-fail="true">
  <description>Neither file has INTEROP-CRITICAL comments</description>
  <input>
    [ui/vite.config.ts]
    export default defineConfig({ base: "/" });
    [ui/src/main.tsx]
    import { initBridge } from "@vrooli/iframe-bridge";
  </input>
  <expected-violations>2</expected-violations>
  <expected-message>INTEROP-CRITICAL</expected-message>
</test-case>
*/

package interop

import (
	"os"
	"path/filepath"
	"strings"

	"app-monitor-api/rules"
)

func init() {
	rules.Register("interop_protective_comments", checkProtectiveComments)
}

var viteConfigNames = []string{"vite.config.ts", "vite.config.js"}

func checkProtectiveComments(ctx rules.CheckContext) rules.RuleResult {
	const ruleID = "interop_protective_comments"
	const marker = "INTEROP-CRITICAL"

	// Check vite config files.
	viteFound := false
	viteHasMarker := false
	viteFile := ""
	for _, name := range viteConfigNames {
		p := filepath.Join(ctx.ScenarioRoot, "ui", name)
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		viteFound = true
		viteFile = "ui/" + name
		if strings.Contains(string(data), marker) {
			viteHasMarker = true
		}
		break
	}

	// Check main entry file.
	mainContent, mainRelPath, _, mainErr := findMainEntry(ctx.ScenarioRoot)
	mainFound := mainErr == nil
	mainHasMarker := false
	if mainFound && strings.Contains(mainContent, marker) {
		mainHasMarker = true
	}

	// If neither file exists, skip.
	if !viteFound && !mainFound {
		return rules.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "no vite config or main entry file found",
			Message:    "no vite config or main entry file found; skipping",
		}
	}

	var violations []rules.Violation

	if viteFound && !viteHasMarker {
		violations = append(violations, rules.Violation{
			RuleID:         ruleID,
			Severity:       "low",
			Title:          "Missing INTEROP-CRITICAL in vite config",
			Description:    viteFile + " does not contain an INTEROP-CRITICAL comment",
			FilePath:       viteFile,
			Recommendation: "Add // INTEROP-CRITICAL comments above interop-sensitive settings in " + viteFile,
		})
	}

	if mainFound && !mainHasMarker {
		violations = append(violations, rules.Violation{
			RuleID:         ruleID,
			Severity:       "low",
			Title:          "Missing INTEROP-CRITICAL in main entry",
			Description:    mainRelPath + " does not contain an INTEROP-CRITICAL comment",
			FilePath:       mainRelPath,
			Recommendation: "Add // INTEROP-CRITICAL comments above interop-sensitive code in " + mainRelPath,
		})
	}

	if len(violations) > 0 {
		return rules.RuleResult{
			RuleID:     ruleID,
			Passed:     false,
			Message:    "INTEROP-CRITICAL comments missing in some files",
			Violations: violations,
		}
	}

	return rules.RuleResult{
		RuleID:  ruleID,
		Passed:  true,
		Message: "INTEROP-CRITICAL comments found in vite config and main entry",
	}
}
