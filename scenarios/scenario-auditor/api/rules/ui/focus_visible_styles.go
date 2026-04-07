package ui

import (
	"path/filepath"
	"strings"

	rules "scenario-auditor/rules"
)

/*
Rule: Focus Visible Styles
Description: Ensure scenario UI includes visible focus indicator styles for keyboard and gamepad navigation.
Reason: Without visible focus indicators, keyboard and gamepad users cannot tell which element is focused, making the UI inaccessible.
Category: ui
Severity: low
Standard: ui-a11y-v1
Targets: ui

<test-case id="focus-visible-missing" should-fail="true" path="ui/src/components/button.tsx">
  <description>Component with no focus-visible styling</description>
  <input language="typescript">
export function Button({ children }: { children: React.ReactNode }) {
  return (
    &lt;button className="bg-blue-500 text-white px-4 py-2 rounded"&gt;
      {children}
    &lt;/button&gt;
  );
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>focus-visible</expected-message>
</test-case>

<test-case id="focus-visible-tailwind" should-fail="false" path="ui/src/components/button.tsx">
  <description>Component using Tailwind focus-visible classes</description>
  <input language="typescript">
export function Button({ children }: { children: React.ReactNode }) {
  return (
    &lt;button className="bg-blue-500 text-white px-4 py-2 rounded focus-visible:ring-2 focus-visible:outline-none"&gt;
      {children}
    &lt;/button&gt;
  );
}
  </input>
</test-case>

<test-case id="focus-visible-css" should-fail="false" path="ui/src/styles.css">
  <description>CSS file with :focus-visible selector</description>
  <input language="css">
button:focus-visible {
  outline: 2px solid blue;
  outline-offset: 2px;
}
  </input>
</test-case>

<test-case id="focus-visible-spatial" should-fail="false" path="ui/src/components/button.tsx">
  <description>Component using data-spatial-focus attribute</description>
  <input language="typescript">
export function Button({ children }: { children: React.ReactNode }) {
  return (
    &lt;button className="[data-spatial-focus]:ring-2 bg-blue-500"&gt;
      {children}
    &lt;/button&gt;
  );
}
  </input>
</test-case>
*/

// CheckFocusVisibleStyles checks that UI files include visible focus indicator
// styles for accessibility.
func CheckFocusVisibleStyles(content []byte, filePath string) []rules.Violation {
	if !isFocusStyleTarget(filePath) {
		return nil
	}

	source := string(content)

	// Pass if any of these focus indicator patterns are present.
	if strings.Contains(source, "focus-visible") {
		return nil
	}
	if strings.Contains(source, "data-spatial-focus") {
		return nil
	}
	if strings.Contains(source, ":focus-visible") {
		return nil
	}

	return []rules.Violation{
		{
			RuleID:         "ui-focus-visible-styles",
			Type:           "ui_focus_visible_styles",
			Severity:       "low",
			Title:          "Missing focus-visible styles",
			Message:        "Interactive UI components should include focus-visible styling (Tailwind focus-visible: classes, CSS :focus-visible selector, or data-spatial-focus attribute styling) for keyboard/gamepad accessibility",
			Description:    "Visible focus indicators help keyboard and gamepad users identify which element is focused",
			File:           filePath,
			FilePath:       filePath,
			Line:           1,
			LineNumber:     1,
			Recommendation: "Add focus-visible:ring-2 focus-visible:outline-none to interactive elements, or use [data-spatial-focus] styling",
			Standard:       "ui-a11y-v1",
		},
	}
}

func isFocusStyleTarget(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".tsx", ".jsx", ".css":
		return true
	default:
		return false
	}
}
