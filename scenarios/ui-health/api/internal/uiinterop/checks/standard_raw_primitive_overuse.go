/*
Rule: Raw Primitive Overuse
ID: standard_raw_primitive_overuse
Description: Pages and custom components should not repeatedly hand-roll raw
  interactive or structural primitives when governed component-library
  counterparts are adopted locally or available in react-component-library.
Why: A raw <table>, <button>, <input>, <select>, or <nav> is sometimes correct,
  but repeated use means every scenario reinvents sorting, focus, safe-area,
  density, token binding, and accessibility. The component canon lets scenarios
  adopt only what they need while still getting professional defaults.
Category: standards
Severity: medium
Slot: [D]
SlotFile: ui/src
TechStack: React
Recommendation: Adopt the matching component from react-component-library
  (DataTable, Button, Input, Select, BottomNav) or compose a local custom
  component with provenance instead of scattering raw primitives through pages.
Standard: vrooli-ui-component-canon-v1

GoodExample:
    import { DataTable } from "../components/ui/data-table";
    <DataTable rows={rows} columns={columns} caption="Fleet" getRowKey={(r) => r.id} />

BadExample:
    <table><tbody>{rows.map((row) => <tr><td>{row.name}</td></tr>)}</tbody></table>

<test-case id="raw-primitive-adopted-clean" should-fail="false">
  <description>Pages use adopted governed components instead of raw primitives</description>
  <input>
    [ui/src/components/ui/data-table.tsx]
    // @vrooliComponentSource react-component-library:DataTable
    // @vrooliComponentVersion 1.0.0
    export function DataTable() { return <table />; }
    [ui/src/pages/Fleet.tsx]
    import { DataTable } from "../components/ui/data-table";
    export function Fleet() { return <DataTable />; }
  </input>
</test-case>

<test-case id="raw-primitive-single-button-clean" should-fail="false">
  <description>A single raw primitive remains below the nudge threshold</description>
  <input>
    [ui/src/components/ui/button.tsx]
    // @vrooliComponentSource react-component-library:Button
    // @vrooliComponentVersion 1.1.0
    export function Button() { return <button />; }
    [ui/src/pages/Settings.tsx]
    export function Settings() { return <button type="button">Save</button>; }
  </input>
</test-case>

<test-case id="raw-primitive-table-and-controls" should-fail="true">
  <description>Hand-rolled table and controls are flagged when governed counterparts exist</description>
  <input>
    [ui/src/components/ui/data-table.tsx]
    // @vrooliComponentSource react-component-library:DataTable
    // @vrooliComponentVersion 1.0.0
    export function DataTable() { return <table />; }
    [ui/src/components/ui/button.tsx]
    // @vrooliComponentSource react-component-library:Button
    // @vrooliComponentVersion 1.1.0
    export function Button() { return <button />; }
    [ui/src/pages/Fleet.tsx]
    export function Fleet() {
      return <section><button>Refresh</button><table><tbody><tr><td>x</td></tr></tbody></table></section>;
    }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>DataTable</expected-message>
</test-case>
*/

package checks

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"ui-health/internal/uiinterop"
)

func init() {
	uiinterop.Register("standard_raw_primitive_overuse", checkRawPrimitiveOveruse)
}

var jsxPrimitivePattern = regexp.MustCompile(`<\s*(table|button|input|textarea|select|nav)(?:\s|>|/)`)

func checkRawPrimitiveOveruse(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "standard_raw_primitive_overuse"

	files := walkUISource(ctx.ScenarioRoot, "ui/src")
	if len(files) == 0 {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "no ui/src directory found",
			Message:    "no ui/src directory found; skipping",
		}
	}
	primitiveToComponent := primitiveComponentIndex(governedComponents(ctx, files))
	if len(primitiveToComponent) == 0 {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "no governed component counterparts found",
			Message:    "no governed component counterparts found; skipping raw primitive nudge",
		}
	}

	var violations []uiinterop.Violation
	for _, f := range files {
		if skipRawPrimitiveFile(f) {
			continue
		}
		ext := strings.ToLower(filepath.Ext(f.relPath))
		if ext != ".tsx" && ext != ".jsx" {
			continue
		}
		counts := rawPrimitiveCounts(f.content, primitiveToComponent)
		if !primitiveCountsOverThreshold(counts) {
			continue
		}
		components := recommendedComponents(counts, primitiveToComponent)
		violations = append(violations, uiinterop.Violation{
			RuleID:         ruleID,
			Severity:       "medium",
			Title:          "Raw UI primitives over governed component threshold",
			Description:    fmt.Sprintf("%s hand-rolls %s while governed counterpart(s) are available: %s", f.relPath, describePrimitiveCounts(counts), strings.Join(components, ", ")),
			FilePath:       f.relPath,
			Line:           lineOfFirstPrimitive(f.content),
			Recommendation: "Adopt " + strings.Join(components, ", ") + " from react-component-library, or compose a local custom component with provenance if the raw markup is intentionally bespoke.",
		})
	}

	if len(violations) > 0 {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Passed:     false,
			Message:    "raw primitives are overused where governed components are available",
			Violations: violations,
		}
	}
	return uiinterop.RuleResult{
		RuleID:  ruleID,
		Passed:  true,
		Message: "raw primitive use stays below the governed-component nudge threshold",
	}
}

func skipRawPrimitiveFile(f uiSourceFile) bool {
	if strings.Contains(f.content, "@vrooliComponentSource") {
		return true
	}
	rel := filepath.ToSlash(f.relPath)
	return strings.Contains(rel, "/components/ui/")
}

func rawPrimitiveCounts(source string, available map[string]governedComponent) map[string]int {
	counts := map[string]int{}
	for _, m := range jsxPrimitivePattern.FindAllStringSubmatch(source, -1) {
		if len(m) < 2 {
			continue
		}
		primitive := m[1]
		if primitive == "textarea" {
			primitive = "input"
		}
		if _, ok := available[primitive]; ok {
			counts[primitive]++
		}
	}
	return counts
}

func primitiveCountsOverThreshold(counts map[string]int) bool {
	total := 0
	for primitive, count := range counts {
		total += count
		if (primitive == "table" || primitive == "nav") && count > 0 {
			return true
		}
	}
	return total >= 2
}

func recommendedComponents(counts map[string]int, available map[string]governedComponent) []string {
	seen := map[string]struct{}{}
	var out []string
	for primitive, count := range counts {
		if count == 0 {
			continue
		}
		component := available[primitive]
		if component.Name == "" {
			continue
		}
		label := component.Name
		if component.Version != "" {
			label += "@" + component.Version
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		out = append(out, label)
	}
	sort.Strings(out)
	return out
}

func describePrimitiveCounts(counts map[string]int) string {
	var parts []string
	for primitive, count := range counts {
		if count > 0 {
			parts = append(parts, fmt.Sprintf("%d <%s>", count, primitive))
		}
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

func lineOfFirstPrimitive(source string) int {
	m := jsxPrimitivePattern.FindString(source)
	if m == "" {
		return 1
	}
	return lineOf(source, m)
}
