/*
Rule: Banned Imperative Scroll
ID: interop_banned_scroll
Description: Flag cross-boundary imperative scrolling calls
  (element.scrollIntoView(...), window.scrollTo(...)) in UI source. Inside the
  Vrooli host frame these can scroll the wrong viewport or fight the
  spatial-navigation focus manager. Scoped scroll-container scrollTo() is an
  allowed remediation when that container owns the region's scrolling.
Why: When the scenario UI runs embedded in the host iframe, global scrolling
  can move the document rather than the intended region. A named local scroller
  is structurally safe; it must not be reported as banned merely because it is
  imperative.
Category: interop
Severity: low
Slot: [D]
SlotFile: ui/src
TechStack: React
Recommendation: Prefer moving focus (which spatial navigation scrolls into
  view) over cross-boundary scrollIntoView()/window.scrollTo(). For restoration
  or panel navigation, call scrollTo() only on the explicitly owned container.
Standard: ui-health-v1

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

<test-case id="scoped-container-scroll-to" should-fail="false">
  <description>A named panel scroller may restore its own position</description>
  <input>
    [ui/src/FileList.tsx]
    export function FileList() {
      const panel = { scrollTo: (_options: { top: number }) => undefined };
      panel.scrollTo({ top: 40 });
      return null;
    }
  </input>
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

var bannedScrollMarkers = []string{".scrollIntoView(", "window.scrollTo("}

func checkBannedScroll(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "interop_banned_scroll"

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
		if ext != ".tsx" && ext != ".jsx" && ext != ".ts" && ext != ".js" {
			continue
		}
		for _, marker := range bannedScrollMarkers {
			if !strings.Contains(f.Content, marker) {
				continue
			}
			call := strings.TrimPrefix(strings.TrimPrefix(marker, "."), "window.")
			violations = append(violations, uiinterop.Violation{
				RuleID:         ruleID,
				Severity:       "low",
				Title:          "Imperative scroll call inside UI",
				Description:    f.RelPath + " calls cross-boundary " + call + " scrolling inside the host frame",
				FilePath:       f.RelPath,
				Line:           lineOf(f.Content, marker),
				Recommendation: "Prefer moving focus (spatial navigation scrolls it into view) over cross-boundary " + call + "; scoped container scrollTo() is allowed for owned regions",
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
