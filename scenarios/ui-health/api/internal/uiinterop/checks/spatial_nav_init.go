/*
Rule: Spatial Nav Init
ID: interop_spatial_nav_init
Description: Ensure the UI main entry initialises spatial navigation
  (initSpatialNav from @vrooli/iframe-bridge/spatial) for gamepad/controller
  support, or explicitly opts out with a marker comment.
Why: Scenario UIs run inside the Vrooli host frame and must be navigable with
  game controllers (Xbox, PlayStation) via spatial navigation. Without an
  initSpatialNav() call, console-browser users cannot reliably move focus
  between interactive elements. An explicit "// spatial-nav: disabled" comment
  is honored as a deliberate opt-out.
Category: interop
Severity: medium
Slot: [D]
SlotFile: ui/src/main.tsx
TechStack: React
Recommendation: Add `import { initSpatialNav } from '@vrooli/iframe-bridge/spatial';`
  and call `initSpatialNav();` in the main entry, or add a
  `// spatial-nav: disabled` comment to opt out.
Standard: ui-health-v1

GoodExample:
    // main.tsx
    import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
    initSpatialNav();

BadExample:
    // main.tsx — no initSpatialNav and no opt-out comment
    ReactDOM.createRoot(el).render(<App />);

<test-case id="spatial-nav-present" should-fail="false">
  <description>main entry initialises spatial navigation</description>
  <input>
    [ui/src/main.tsx]
    import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
    import App from "./App";
    initSpatialNav();
    ReactDOM.createRoot(document.getElementById("root")).render(<App />);
  </input>
</test-case>

<test-case id="spatial-nav-opt-out" should-fail="false">
  <description>main entry explicitly opts out</description>
  <input>
    [ui/src/main.tsx]
    import App from "./App";
    // spatial-nav: disabled
    ReactDOM.createRoot(document.getElementById("root")).render(<App />);
  </input>
</test-case>

<test-case id="spatial-nav-missing" should-fail="true">
  <description>main entry neither initialises spatial nav nor opts out</description>
  <input>
    [ui/src/main.tsx]
    import App from "./App";
    ReactDOM.createRoot(document.getElementById("root")).render(<App />);
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>initSpatialNav</expected-message>
</test-case>
*/

package checks

import (
	"strings"

	"ui-health/internal/uiinterop"
)

func init() {
	uiinterop.Register("interop_spatial_nav_init", checkSpatialNavInit)
}

func checkSpatialNavInit(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "interop_spatial_nav_init"

	content, relPath, _, err := findMainEntry(ctx.ScenarioRoot)
	if err != nil {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "no UI main entry file found",
			Message:    "no UI main entry file found; skipping",
		}
	}

	if strings.Contains(content, "initSpatialNav") || strings.Contains(content, "spatial-nav: disabled") {
		return uiinterop.RuleResult{
			RuleID:  ruleID,
			Passed:  true,
			Message: "spatial navigation initialised (or explicitly opted out)",
		}
	}

	line := lineOf(content, "ReactDOM")
	return uiinterop.RuleResult{
		RuleID:  ruleID,
		Passed:  false,
		Message: "UI main entry does not call initSpatialNav()",
		Violations: []uiinterop.Violation{{
			RuleID:         ruleID,
			Severity:       "medium",
			Title:          "Missing spatial navigation initialisation",
			Description:    relPath + " must call initSpatialNav() from @vrooli/iframe-bridge/spatial for gamepad support, or include a '// spatial-nav: disabled' comment to opt out",
			FilePath:       relPath,
			Line:           line,
			Recommendation: "Add `import { initSpatialNav } from '@vrooli/iframe-bridge/spatial';` and call `initSpatialNav();` in " + relPath,
		}},
	}
}
