/*
Rule: Unused Custom Components
ID: standard_unused_custom_component
Description: Exported React components under ui/src/components or ui/src/layout
  should be referenced by the UI, unless they are declared widgets or route-only
  roots. Unused component code makes the surface harder to audit and hides
  obsolete custom UI that should either be adopted, wired, or deleted.
Why: Component provenance only helps when the component inventory reflects real
  UI. Dead custom components create false signals for adoption maturity and
  increase maintenance cost without improving the product.
Category: standards
Severity: medium
Slot: [D]
SlotFile: ui/src/components
TechStack: React
Recommendation: Wire the component into a page, convert it to a governed
  adoption if it is meant to be reused, add @vrooliWidget for externally
  embedded widgets, or remove the unused component.
Standard: vrooli-ui-component-canon-v1

GoodExample:
    export function StatusPanel() { return <section>Status</section>; }
    <StatusPanel />

BadExample:
    export function StalePanel() { return <section>Old</section>; }

<test-case id="unused-component-referenced" should-fail="false">
  <description>Exported component is imported and rendered elsewhere</description>
  <input>
    [ui/src/components/StatusPanel.tsx]
    export function StatusPanel() { return <section>Status</section>; }
    [ui/src/pages/Dashboard.tsx]
    import { StatusPanel } from "../components/StatusPanel";
    export function Dashboard() { return <StatusPanel />; }
  </input>
</test-case>

<test-case id="unused-component-widget-exempt" should-fail="false">
  <description>Externally embedded widget components may have no local JSX reference</description>
  <input>
    [ui/src/components/PublicWidget.tsx]
    // @vrooliWidget public-widget
    export function PublicWidget() { return <section>Widget</section>; }
  </input>
</test-case>

<test-case id="unused-component-export" should-fail="true">
  <description>Exported component has no references outside its own file</description>
  <input>
    [ui/src/components/StalePanel.tsx]
    export function StalePanel() { return <section>Old</section>; }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>StalePanel</expected-message>
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
	uiinterop.Register("standard_unused_custom_component", checkUnusedCustomComponent)
}

var exportedComponentPattern = regexp.MustCompile(`(?m)\bexport\s+(?:function|const)\s+([A-Z][A-Za-z0-9_]*)\b`)

func checkUnusedCustomComponent(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "standard_unused_custom_component"

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
		if !isComponentSource(f.relPath) || isComponentExempt(f) {
			continue
		}
		for _, name := range exportedComponents(f.content) {
			if componentReferenced(files, f.relPath, name) {
				continue
			}
			violations = append(violations, uiinterop.Violation{
				RuleID:         ruleID,
				Severity:       "medium",
				Title:          "Unused exported component",
				Description:    f.relPath + " exports " + name + " but no production UI file references it",
				FilePath:       f.relPath,
				Line:           lineOf(f.content, name),
				Recommendation: "Use the component, remove it, or mark externally embedded surfaces with @vrooliWidget.",
			})
		}
	}

	if len(violations) > 0 {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Passed:     false,
			Message:    "unused exported custom components found",
			Violations: violations,
		}
	}
	return uiinterop.RuleResult{
		RuleID:  ruleID,
		Passed:  true,
		Message: "exported custom components are referenced or explicitly exempt",
	}
}

func isComponentSource(relPath string) bool {
	rel := filepath.ToSlash(relPath)
	ext := strings.ToLower(filepath.Ext(rel))
	if ext != ".tsx" && ext != ".jsx" {
		return false
	}
	return strings.Contains(rel, "/components/") || strings.Contains(rel, "/layout/")
}

func isComponentExempt(f uiSourceFile) bool {
	base := strings.ToLower(filepath.Base(f.relPath))
	if base == "index.tsx" || base == "index.jsx" || strings.Contains(f.content, "@vrooliWidget") {
		return true
	}
	return strings.Contains(f.content, "React.lazy(") || strings.Contains(f.content, "lazy(()")
}

func exportedComponents(source string) []string {
	matches := exportedComponentPattern.FindAllStringSubmatch(source, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) > 1 {
			out = append(out, m[1])
		}
	}
	return uniqueStrings(out)
}

func componentReferenced(files []uiSourceFile, ownPath, name string) bool {
	for _, other := range files {
		if other.relPath == ownPath {
			continue
		}
		if strings.Contains(other.content, "<"+name) ||
			strings.Contains(other.content, "{ "+name+" }") ||
			strings.Contains(other.content, "{"+name+"}") ||
			strings.Contains(other.content, name+" as ") ||
			strings.Contains(other.content, " "+name+",") ||
			strings.Contains(other.content, " "+name+" ") {
			return true
		}
	}
	return false
}
