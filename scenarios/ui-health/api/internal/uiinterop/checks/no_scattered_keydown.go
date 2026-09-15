/*
Rule: No Scattered Keydown Listeners
ID: interop_no_scattered_keydown
Description: Ensures addEventListener('keydown') calls are only placed in
  hooks/ directories and dismissible UI components (dialogs, modals,
  etc.), not scattered across arbitrary source files.
Why: Scattered keydown listeners create unpredictable shortcut behavior
  when the UI runs in an iframe. Centralizing keyboard handling in
  hooks/ makes it easy to add iframe-bridge relay logic in one place.
  Dismissible components (modals, dropdowns) are excepted because they
  legitimately need Escape key handling.
Category: interop
Severity: medium
Slot: [G]
SlotFile: ui/src/
TechStack: React
Recommendation: Move addEventListener('keydown') calls into a dedicated
  hook (e.g., useKeyboard) in ui/src/hooks/ and import it where needed.
Standard: ui-health-v1

GoodExample:
    // ui/src/hooks/useKeyboard.ts — centralized
    export function useKeyboard() {
      useEffect(() => {
        const handler = (e: KeyboardEvent) => { ... };
        window.addEventListener("keydown", handler);
        return () => window.removeEventListener("keydown", handler);
      }, []);
    }

BadExample:
    // ui/src/pages/Editor.tsx — scattered
    useEffect(() => {
      window.addEventListener("keydown", (e) => {
        if (e.key === "s" && e.ctrlKey) save();
      });
    }, []);

<test-case id="keydown-in-hooks-only" should-fail="false">
  <description>keydown listener only in hooks directory</description>
  <input>
    [ui/src/hooks/useKeyboard.ts]
    window.addEventListener("keydown", handler);
    [ui/src/pages/Home.tsx]
    import { useKeyboard } from "../hooks/useKeyboard";
    export default function Home() { useKeyboard(); return <div />; }
  </input>
</test-case>

<test-case id="keydown-in-modal" should-fail="false">
  <description>keydown in a dismissible dialog component is allowed</description>
  <input>
    [ui/src/components/ConfirmDialog.tsx]
    window.addEventListener("keydown", (e) => {
      if (e.key === "Escape") close();
    });
  </input>
</test-case>

<test-case id="keydown-in-page" should-fail="true">
  <description>keydown listener in a page component</description>
  <input>
    [ui/src/pages/Editor.tsx]
    window.addEventListener("keydown", (e) => {
      if (e.key === "s" && e.ctrlKey) save();
    });
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>addEventListener('keydown') found outside</expected-message>
</test-case>

<test-case id="keydown-in-multiple-pages" should-fail="true">
  <description>keydown listeners scattered in multiple pages</description>
  <input>
    [ui/src/pages/Editor.tsx]
    window.addEventListener("keydown", handler1);
    [ui/src/components/Toolbar.tsx]
    document.addEventListener( 'keydown' , handler2);
  </input>
  <expected-violations>2</expected-violations>
  <expected-message>addEventListener('keydown') found outside</expected-message>
</test-case>
*/

package checks

import (
	"path/filepath"
	"regexp"
	"strings"

	"ui-health/internal/uiinterop"
)

func init() {
	uiinterop.Register("interop_no_scattered_keydown", checkNoScatteredKeydown)
}

var keydownListenerPattern = regexp.MustCompile(`addEventListener\s*\(\s*['"]keydown['"]`)

// dismissibleNames are component name fragments that are allowed to have
// keydown listeners because they need Escape key handling.
var dismissibleNames = []string{
	"dialog", "modal", "popup", "popover",
	"overlay", "dropdown", "selector", "menu", "tooltip",
}

func checkNoScatteredKeydown(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "interop_no_scattered_keydown"

	files := sourceFiles(ctx, "ui/src")
	if len(files) == 0 {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "ui/src/ directory not found",
			Message:    "ui/src/ directory not found; skipping",
		}
	}

	var violations []uiinterop.Violation

	for _, f := range files {
		// Allow hooks/ directory.
		if strings.HasPrefix(strings.TrimPrefix(f.RelPath, "ui/src/"), "hooks/") {
			continue
		}

		// Allow dismissible component files.
		lowerName := strings.ToLower(filepath.Base(f.RelPath))
		if isDismissibleFileName(lowerName) {
			continue
		}

		if keydownListenerPattern.MatchString(f.Content) {
			line := 0
			lines := strings.Split(f.Content, "\n")
			for i, l := range lines {
				if keydownListenerPattern.MatchString(l) {
					line = i + 1
					break
				}
			}
			violations = append(violations, uiinterop.Violation{
				RuleID:         ruleID,
				Severity:       "medium",
				Title:          "Scattered keydown listener",
				Description:    "addEventListener('keydown') found outside hooks/ or dismissible component in " + f.RelPath,
				FilePath:       f.RelPath,
				Line:           line,
				Recommendation: "Move the keydown listener into a dedicated hook in ui/src/hooks/",
			})
		}
	}

	if len(violations) > 0 {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Passed:     false,
			Message:    "addEventListener('keydown') found outside allowed locations",
			Violations: violations,
		}
	}

	return uiinterop.RuleResult{
		RuleID:  ruleID,
		Passed:  true,
		Message: "no scattered keydown listeners found in ui/src/",
	}
}

func isDismissibleFileName(lowerName string) bool {
	for _, d := range dismissibleNames {
		if strings.Contains(lowerName, d) {
			return true
		}
	}
	return false
}
