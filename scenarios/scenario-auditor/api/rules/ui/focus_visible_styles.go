package ui

import (
	"os"
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

<test-case id="focus-visible-noninteractive" should-fail="false" path="ui/src/components/Heading.tsx">
  <description>Non-interactive component does not need local focus-visible styling</description>
  <input language="typescript">
export function Heading() {
  return &lt;h1 className="text-xl"&gt;Title&lt;/h1&gt;;
}
  </input>
</test-case>

<test-case id="focus-visible-test-file" should-fail="false" path="ui/src/components/button.test.tsx">
  <description>Tests can render interactive fixtures without carrying production focus classes</description>
  <input language="typescript">
it("renders", () => {
  render(&lt;button type="button"&gt;Fixture&lt;/button&gt;);
});
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
	if hasFocusVisibleMarker(source) {
		return nil
	}
	if isFocusStyleTestFile(filePath) {
		return nil
	}
	if !containsInteractiveUI(source, filePath) {
		return nil
	}
	if scenarioHasGlobalFocusVisiblePolicy(filePath) {
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

func hasFocusVisibleMarker(source string) bool {
	return strings.Contains(source, "focus-visible") ||
		strings.Contains(source, "data-spatial-focus") ||
		strings.Contains(source, ":focus-visible")
}

func isFocusStyleTestFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return strings.Contains(base, ".test.") ||
		strings.Contains(base, ".spec.") ||
		strings.HasSuffix(base, "_test.tsx") ||
		strings.HasSuffix(base, "_test.jsx")
}

func containsInteractiveUI(source, path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".css" {
		return strings.Contains(source, ":focus") ||
			strings.Contains(source, "button") ||
			strings.Contains(source, "input") ||
			strings.Contains(source, "textarea") ||
			strings.Contains(source, "select") ||
			strings.Contains(source, "[tabindex]")
	}

	patterns := []string{
		"<button",
		"<a ",
		"<a\n",
		"<input",
		"<textarea",
		"<select",
		"tabIndex",
		"role=\"button\"",
		"role='button'",
	}
	for _, pattern := range patterns {
		if strings.Contains(source, pattern) {
			return true
		}
	}
	return false
}

func scenarioHasGlobalFocusVisiblePolicy(filePath string) bool {
	if !filepath.IsAbs(filePath) {
		return false
	}
	dir := filepath.Dir(filePath)
	for {
		for _, rel := range []string{
			filepath.Join("ui", "src", "styles.css"),
			filepath.Join("ui", "src", "index.css"),
			filepath.Join("src", "styles.css"),
			filepath.Join("src", "index.css"),
		} {
			path := filepath.Join(dir, rel)
			data, err := os.ReadFile(path)
			if err == nil && hasGlobalFocusVisiblePolicy(string(data)) {
				return true
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}

func hasGlobalFocusVisiblePolicy(source string) bool {
	if !strings.Contains(source, ":focus-visible") {
		return false
	}
	return strings.Contains(source, "outline") ||
		strings.Contains(source, "box-shadow") ||
		strings.Contains(source, "ring")
}
