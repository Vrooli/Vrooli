/*
Rule: Focus Visible Styles
ID: interop_focus_visible_styles
Description: Ensure interactive UI components include visible focus-indicator
  styles (Tailwind focus-visible: classes, a CSS :focus-visible selector, or
  data-spatial-focus styling) for keyboard and gamepad navigation.
Why: Without a visible focus indicator, keyboard and gamepad users cannot tell
  which element is focused, making the UI inaccessible inside the Vrooli host
  frame where pointer input may be unavailable. A scenario-wide
  :focus-visible policy in a global stylesheet satisfies the requirement for
  all components.
Category: interop
Severity: low
Slot: [D]
SlotFile: ui/src
TechStack: React
Recommendation: Add `focus-visible:ring-2 focus-visible:outline-none` to
  interactive elements, a `:focus-visible` rule to a global stylesheet, or
  `[data-spatial-focus]` styling.
Standard: vrooli-ui-a11y-v1

GoodExample:
    <button className="... focus-visible:ring-2 focus-visible:outline-none">Go</button>

BadExample:
    <button className="bg-blue-500 text-white">Go</button>

<test-case id="focus-visible-tailwind" should-fail="false">
  <description>Component uses Tailwind focus-visible classes</description>
  <input>
    [ui/src/components/button.tsx]
    export function Button() {
      return <button className="bg-blue-500 focus-visible:ring-2 focus-visible:outline-none">Go</button>;
    }
  </input>
</test-case>

<test-case id="focus-visible-global-css" should-fail="false">
  <description>Global stylesheet declares a :focus-visible policy</description>
  <input>
    [ui/src/index.css]
    :focus-visible { outline: 2px solid blue; outline-offset: 2px; }
    [ui/src/components/button.tsx]
    export function Button() {
      return <button className="bg-blue-500">Go</button>;
    }
  </input>
</test-case>

<test-case id="focus-visible-noninteractive" should-fail="false">
  <description>Non-interactive component needs no focus styling</description>
  <input>
    [ui/src/components/Heading.tsx]
    export function Heading() { return <h1 className="text-xl">Title</h1>; }
  </input>
</test-case>

<test-case id="focus-visible-missing" should-fail="true">
  <description>Interactive component with no focus-visible styling and no global policy</description>
  <input>
    [ui/src/components/button.tsx]
    export function Button() {
      return <button className="bg-blue-500 text-white px-4 py-2 rounded">Go</button>;
    }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>focus-visible</expected-message>
</test-case>
*/

package checks

import (
	"path/filepath"
	"strings"

	"ui-health/internal/uiinterop"
)

func init() {
	uiinterop.Register("interop_focus_visible_styles", checkFocusVisibleStyles)
}

func checkFocusVisibleStyles(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "interop_focus_visible_styles"

	files := sourceFiles(ctx, "ui/src")
	if len(files) == 0 {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "no ui/src directory found",
			Message:    "no ui/src directory found; skipping",
		}
	}

	// A scenario-wide :focus-visible policy in any global stylesheet satisfies
	// the requirement for every component.
	if scenarioHasGlobalFocusVisiblePolicy(files) {
		return uiinterop.RuleResult{
			RuleID:  ruleID,
			Passed:  true,
			Message: "scenario declares a global :focus-visible policy",
		}
	}

	var violations []uiinterop.Violation
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f.RelPath))
		if ext != ".tsx" && ext != ".jsx" && ext != ".css" {
			continue
		}
		if hasFocusVisibleMarker(f.Content) {
			continue
		}
		if !containsInteractiveUI(f.Content, ext) {
			continue
		}
		violations = append(violations, uiinterop.Violation{
			RuleID:         ruleID,
			Severity:       "low",
			Title:          "Missing focus-visible styles",
			Description:    f.RelPath + " has interactive UI without focus-visible styling; add focus-visible: classes, a :focus-visible rule, or data-spatial-focus styling",
			FilePath:       f.RelPath,
			Line:           1,
			Recommendation: "Add focus-visible:ring-2 focus-visible:outline-none to interactive elements, or use [data-spatial-focus] styling",
		})
	}

	if len(violations) > 0 {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Passed:     false,
			Message:    "interactive components missing focus-visible styles",
			Violations: violations,
		}
	}

	return uiinterop.RuleResult{
		RuleID:  ruleID,
		Passed:  true,
		Message: "interactive components include focus-visible styling",
	}
}

func hasFocusVisibleMarker(source string) bool {
	return strings.Contains(source, "focus-visible") ||
		strings.Contains(source, "data-spatial-focus") ||
		strings.Contains(source, ":focus-visible")
}

func containsInteractiveUI(source, ext string) bool {
	if ext == ".css" {
		return strings.Contains(source, ":focus") ||
			strings.Contains(source, "button") ||
			strings.Contains(source, "input") ||
			strings.Contains(source, "textarea") ||
			strings.Contains(source, "select") ||
			strings.Contains(source, "[tabindex]")
	}
	patterns := []string{"<button", "<a ", "<a\n", "<input", "<textarea", "<select", "tabIndex", "role=\"button\"", "role='button'"}
	for _, p := range patterns {
		if strings.Contains(source, p) {
			return true
		}
	}
	return false
}

// scenarioHasGlobalFocusVisiblePolicy reports whether any global stylesheet in
// the scanned set declares a :focus-visible rule with a visible-indicator
// property (outline/box-shadow/ring).
func scenarioHasGlobalFocusVisiblePolicy(files []uiinterop.SourceFile) bool {
	globalNames := map[string]bool{"styles.css": true, "index.css": true, "globals.css": true, "global.css": true, "main.css": true}
	for _, f := range files {
		if !globalNames[strings.ToLower(filepath.Base(f.RelPath))] {
			continue
		}
		if hasGlobalFocusVisiblePolicy(f.Content) {
			return true
		}
	}
	return false
}

func hasGlobalFocusVisiblePolicy(source string) bool {
	if !strings.Contains(source, ":focus-visible") {
		return false
	}
	return strings.Contains(source, "outline") ||
		strings.Contains(source, "box-shadow") ||
		strings.Contains(source, "ring")
}
