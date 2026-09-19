/*
Rule: Raw Primitive Overuse
ID: standard_raw_primitive_overuse
Description: Pages and custom components should not repeatedly hand-roll raw
  interactive or structural primitives when governed component-library
  counterparts are adopted locally or available in react-component-library.
Why: A raw <table>, <button>, <input>, <select>, <nav>, modal surface, status
  pill, or empty-state block is sometimes correct, but repeated use means every
  scenario reinvents sorting, focus, safe-area, density, token binding, and
  accessibility. The component canon lets scenarios adopt only what they need
  while still getting professional defaults.
Category: standards
Severity: medium
Slot: [D]
SlotFile: ui/src
TechStack: React
Recommendation: Adopt the matching component from react-component-library
  (DataTable, Button, Input, Select, BottomNav, Dialog, StatusBadge, EmptyState)
  or compose a local custom
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

<test-case id="raw-primitive-dialog-pattern" should-fail="true">
  <description>Hand-rolled dialog semantics are flagged when Dialog is available</description>
  <input>
    [ui/src/components/ui/dialog.tsx]
    // @vrooliComponentSource react-component-library:Dialog
    // @vrooliComponentVersion 1.0.0
    export function Dialog() { return <div role="dialog" />; }
    [ui/src/pages/Confirm.tsx]
    export function Confirm() {
      return <div role="dialog" aria-modal="true"><button>Close</button></div>;
    }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Dialog</expected-message>
</test-case>

<test-case id="raw-primitive-status-badges" should-fail="true">
  <description>Repeated hand-rolled status pills are flagged when StatusBadge is available</description>
  <input>
    [ui/src/components/ui/status-badge.tsx]
    // @vrooliComponentSource react-component-library:StatusBadge
    // @vrooliComponentVersion 1.0.0
    export function StatusBadge() { return <span />; }
    [ui/src/pages/Jobs.tsx]
    export function Jobs() {
      return <section><span className="rounded-full bg-green-500">Active status</span><span className="rounded-full bg-red-500">Failed status</span></section>;
    }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>StatusBadge</expected-message>
</test-case>

<test-case id="raw-primitive-empty-state" should-fail="true">
  <description>Hand-rolled empty-state copy is flagged when EmptyState is available</description>
  <input>
    [ui/src/components/ui/empty-state.tsx]
    // @vrooliComponentSource react-component-library:EmptyState
    // @vrooliComponentVersion 1.0.0
    export function EmptyState() { return <section />; }
    [ui/src/pages/Jobs.tsx]
    export function Jobs() {
      return <section><h2>No jobs found</h2><p>Nothing to show yet.</p></section>;
    }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>EmptyState</expected-message>
</test-case>

<test-case id="raw-primitive-empty-state-component-clean" should-fail="false">
  <description>Empty-state copy rendered through the governed EmptyState component is clean</description>
  <input>
    [ui/src/components/ui/empty-state.tsx]
    // @vrooliComponentSource react-component-library:EmptyState
    // @vrooliComponentVersion 1.0.0
    export function EmptyState() { return <section />; }
    [ui/src/pages/Jobs.tsx]
    import { EmptyState } from "../components/ui/empty-state";
    export function Jobs() {
      return <EmptyState title="No jobs found" description="Nothing to show yet." />;
    }
  </input>
</test-case>

<test-case id="raw-primitive-declared-library-dependency" should-fail="true">
  <description>Declared component-library dependency is enough intent to flag a hand-rolled table</description>
  <input>
    [ui/package.json]
    {"dependencies":{"react":"^18.3.1","@vrooli/react-component-library":"file:../../react-component-library"}}
    [ui/src/pages/Fleet.tsx]
    export function Fleet() {
      return <table><tbody><tr><td>x</td></tr></tbody></table>;
    }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>DataTable</expected-message>
</test-case>

<test-case id="raw-primitive-design-adapter-intent" should-fail="true">
  <description>React Vite design-adapter intent is enough to flag a hand-rolled table</description>
  <input>
    [.vrooli/service.json]
    {"generation":{"design":{"adapter":"react-vite-tailwind"}}}
    [ui/src/pages/Fleet.tsx]
    export function Fleet() {
      return <table><tbody><tr><td>x</td></tr></tbody></table>;
    }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>DataTable</expected-message>
</test-case>

<test-case id="raw-primitive-ui-manifest-template-intent" should-fail="true">
  <description>React Vite UI manifest template intent is enough to flag a hand-rolled table</description>
  <input>
    [ui/manifest.json]
    {"contract":{"template":"react-vite"}}
    [ui/src/pages/Fleet.tsx]
    export function Fleet() {
      return <table><tbody><tr><td>x</td></tr></tbody></table>;
    }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>DataTable</expected-message>
</test-case>

<test-case id="raw-primitive-comment-clean" should-fail="false">
  <description>Primitive tags mentioned in comments do not count as rendered raw primitives</description>
  <input>
    [ui/src/components/ui/button.tsx]
    // @vrooliComponentSource react-component-library:Button
    // @vrooliComponentVersion 1.1.0
    export function Button() { return <button />; }
    [ui/src/hooks/SpatialGroup.tsx]
    // Example: <button>One</button>
    // Example: <button>Two</button>
    export function SpatialGroup() { return <div />; }
  </input>
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

var (
	jsxPrimitivePattern    = regexp.MustCompile(`<\s*(table|button|input|textarea|select|nav|dialog)(?:\s|>|/)`)
	roleDialogPattern      = regexp.MustCompile(`(?i)role\s*=\s*["'{]?dialog["'}]?`)
	showModalPattern       = regexp.MustCompile(`(?i)\.showModal\s*\(`)
	statusPillPattern      = regexp.MustCompile(`(?i)rounded-full[^"']*(status|active|pending|failed|success|warning|error)?`)
	emptyStatePattern      = regexp.MustCompile(`(?i)(no\s+[\w -]+\s+(found|yet|available)|nothing\s+to\s+show|empty\s+state)`)
	tsxBlockCommentPattern = regexp.MustCompile(`(?s)/\*.*?\*/`)
	tsxLineCommentPattern  = regexp.MustCompile(`(?m)//.*$`)
)

func stripTSXComments(source string) string {
	withoutBlocks := tsxBlockCommentPattern.ReplaceAllString(source, "")
	return tsxLineCommentPattern.ReplaceAllString(withoutBlocks, "")
}

func checkRawPrimitiveOveruse(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "standard_raw_primitive_overuse"

	files := sourceFiles(ctx, "ui/src")
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
		ext := strings.ToLower(filepath.Ext(f.RelPath))
		if ext != ".tsx" && ext != ".jsx" {
			continue
		}
		counts := rawPrimitiveCounts(f.Content, primitiveToComponent)
		if !primitiveCountsOverThreshold(counts) {
			continue
		}
		components := recommendedComponents(counts, primitiveToComponent)
		violations = append(violations, uiinterop.Violation{
			RuleID:         ruleID,
			Severity:       "medium",
			Title:          "Raw UI primitives over governed component threshold",
			Description:    fmt.Sprintf("%s hand-rolls %s while governed counterpart(s) are available: %s", f.RelPath, describePrimitiveCounts(counts), strings.Join(components, ", ")),
			FilePath:       f.RelPath,
			Line:           lineOfFirstPrimitive(f.Content),
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

func skipRawPrimitiveFile(f uiinterop.SourceFile) bool {
	if strings.Contains(f.Content, "@vrooliComponentSource") {
		return true
	}
	rel := filepath.ToSlash(f.RelPath)
	return strings.Contains(rel, "/components/ui/")
}

func rawPrimitiveCounts(source string, available map[string]governedComponent) map[string]int {
	counts := map[string]int{}
	stripped := stripTSXComments(source)
	for _, m := range jsxPrimitivePattern.FindAllStringSubmatch(stripped, -1) {
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
	if _, ok := available["dialog"]; ok {
		counts["dialog"] += len(roleDialogPattern.FindAllString(stripped, -1))
		counts["dialog"] += len(showModalPattern.FindAllString(stripped, -1))
	}
	if _, ok := available["status"]; ok {
		counts["status"] += len(statusPillPattern.FindAllString(stripped, -1))
	}
	if component, ok := available["empty-state"]; ok && emptyStatePattern.MatchString(stripped) {
		if component.Name == "" || !jsxComponentUsagePattern(component.Name).MatchString(stripped) {
			counts["empty-state"]++
		}
	}
	return counts
}

func jsxComponentUsagePattern(name string) *regexp.Regexp {
	return regexp.MustCompile(`<\s*` + regexp.QuoteMeta(name) + `(?:\s|>|/)`)
}

func primitiveCountsOverThreshold(counts map[string]int) bool {
	total := 0
	for primitive, count := range counts {
		if (primitive == "table" || primitive == "nav" || primitive == "dialog" || primitive == "empty-state") && count > 0 {
			return true
		}
		if primitive == "status" && count >= 2 {
			return true
		}
		if primitive == "button" || primitive == "input" || primitive == "select" {
			total += count
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
