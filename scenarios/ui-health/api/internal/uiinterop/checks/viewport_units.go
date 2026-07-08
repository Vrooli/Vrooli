/*
Rule: Viewport-Relative Sizing
ID: interop_h_screen
Description: Flag viewport-relative full-size utilities (Tailwind h-screen /
  w-screen / min-h-screen / min-w-screen and raw 100vh / 100vw) in UI source.
  Inside the Vrooli host iframe, viewport units resolve against the top-level
  window rather than the frame, so a "full screen" element overflows or
  mis-sizes. Use container-relative sizing (h-full / 100%) instead.
Why: When the scenario UI is embedded, vh/vw refer to the outer viewport, not
  the iframe's content box. An h-screen root makes the app taller than its
  frame, producing double scrollbars or clipped content. h-full (100% of the
  parent) sizes correctly in both standalone and embedded contexts.
Category: interop
Severity: medium
Slot: [D]
SlotFile: ui/src
TechStack: React
Recommendation: Replace h-screen with h-full (and min-h-screen with min-h-full,
  w-screen with w-full); replace raw 100vh/100vw with 100%.
Standard: vrooli-ui-interop-v1

GoodExample:
    <div className="h-full w-full">...</div>

BadExample:
    <div className="h-screen w-screen">...</div>

<test-case id="viewport-units-none" should-fail="false">
  <description>Container-relative sizing</description>
  <input>
    [ui/src/App.tsx]
    export function App() { return <div className="h-full w-full">app</div>; }
  </input>
</test-case>

<test-case id="viewport-units-h-screen" should-fail="true">
  <description>Component uses h-screen</description>
  <input>
    [ui/src/App.tsx]
    export function App() { return <div className="h-screen w-full">app</div>; }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>h-screen</expected-message>
</test-case>
*/

package checks

import (
	"path/filepath"
	"strings"

	"ui-health/internal/uiinterop"
)

func init() {
	uiinterop.Register("interop_h_screen", checkViewportUnits)
}

// viewportTokens are the substrings that indicate viewport-relative full
// sizing. h-screen / w-screen also match the min-h-screen / min-w-screen
// variants (substring), so listing the base tokens avoids double-counting.
var viewportTokens = []string{"h-screen", "w-screen", "100vh", "100vw"}

func checkViewportUnits(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "interop_h_screen"

	files := sourceFiles(ctx, "ui/src")
	if len(files) == 0 {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "no ui/src directory found",
			Message:    "no ui/src directory found; skipping",
		}
	}

	var violations []uiinterop.Violation
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f.RelPath))
		if ext != ".tsx" && ext != ".jsx" && ext != ".css" {
			continue
		}
		for _, tok := range viewportTokens {
			if !strings.Contains(f.Content, tok) {
				continue
			}
			violations = append(violations, uiinterop.Violation{
				RuleID:         ruleID,
				Severity:       "medium",
				Title:          "Viewport-relative sizing breaks iframe embedding",
				Description:    f.RelPath + " uses " + tok + " viewport units resolve against the outer window inside the host frame; use container-relative sizing",
				FilePath:       f.RelPath,
				Line:           lineOf(f.Content, tok),
				Recommendation: "Replace " + tok + " with the container-relative equivalent (h-screen→h-full, w-screen→w-full, 100vh/100vw→100%)",
			})
		}
	}

	if len(violations) > 0 {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Passed:     false,
			Message:    "viewport-relative full-size utilities found",
			Violations: violations,
		}
	}

	return uiinterop.RuleResult{
		RuleID:  ruleID,
		Passed:  true,
		Message: "no viewport-relative full-size utilities found",
	}
}
