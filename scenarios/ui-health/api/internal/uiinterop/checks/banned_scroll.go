/*
Rule: Banned Imperative Scroll
ID: interop_banned_scroll
Description: Flag direct imperative scrolling calls (element.scrollIntoView(...),
  window.scrollTo(...)) in UI source. Inside the Vrooli host frame these fight
  the spatial-navigation focus manager and can scroll the wrong (outer)
  viewport. This is a report-only advisory — there is no safe automatic rewrite.
Why: When the scenario UI runs embedded in the host iframe, imperative scroll
  calls operate on the iframe's own scroll position and frequently conflict
  with spatial navigation, which moves focus and brings elements into view via
  the bridge. Surfacing these calls lets authors decide whether to route the
  intent through spatial focus instead.
Category: interop
Severity: low
Slot: [D]
SlotFile: ui/src
TechStack: React
Recommendation: Prefer moving focus (which spatial navigation scrolls into
  view) over calling scrollIntoView()/window.scrollTo() directly; if a manual
  scroll is genuinely required, confirm it targets the in-frame element.
Standard: vrooli-ui-interop-v1

GoodExample:
    // Move focus; spatial-nav brings it into view.
    ref.current?.focus();

BadExample:
    ref.current?.scrollIntoView({ behavior: "smooth" });

<test-case id="banned-scroll-none" should-fail="false">
  <description>No imperative scroll calls</description>
  <input>
    [ui/src/App.tsx]
    export function App() {
      const ref = useRef(null);
      return <div ref={ref} onClick={() => ref.current?.focus()} />;
    }
  </input>
</test-case>

<test-case id="banned-scroll-into-view" should-fail="true">
  <description>Component calls scrollIntoView</description>
  <input>
    [ui/src/App.tsx]
    export function App() {
      const ref = useRef(null);
      return <div ref={ref} onClick={() => ref.current?.scrollIntoView({ behavior: "smooth" })} />;
    }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>scrollIntoView</expected-message>
</test-case>
*/

package checks

import (
	"path/filepath"
	"strings"

	"ui-health/internal/uiinterop"
)

func init() {
	uiinterop.Register("interop_banned_scroll", checkBannedScroll)
}

var bannedScrollMarkers = []string{".scrollIntoView(", "window.scrollTo(", ".scrollTo("}

func checkBannedScroll(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "interop_banned_scroll"

	files := walkUISource(ctx.ScenarioRoot, "ui/src")
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
		ext := strings.ToLower(filepath.Ext(f.relPath))
		if ext != ".tsx" && ext != ".jsx" && ext != ".ts" && ext != ".js" {
			continue
		}
		for _, marker := range bannedScrollMarkers {
			if !strings.Contains(f.content, marker) {
				continue
			}
			call := strings.TrimPrefix(strings.TrimPrefix(marker, "."), "window.")
			violations = append(violations, uiinterop.Violation{
				RuleID:         ruleID,
				Severity:       "low",
				Title:          "Imperative scroll call inside UI",
				Description:    f.relPath + " calls " + call + " imperative scrolling fights spatial navigation inside the host frame",
				FilePath:       f.relPath,
				Line:           lineOf(f.content, marker),
				Recommendation: "Prefer moving focus (spatial navigation scrolls it into view) over calling " + call + " directly",
			})
		}
	}

	if len(violations) > 0 {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Passed:     false,
			Message:    "imperative scroll calls found (report-only advisory)",
			Violations: violations,
		}
	}

	return uiinterop.RuleResult{
		RuleID:  ruleID,
		Passed:  true,
		Message: "no imperative scroll calls found",
	}
}
