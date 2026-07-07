/*
Rule: Component Location
ID: standard_component_location
Description: Reusable React components should live in the component inventory,
  and governed component-library adoptions should live under ui/src/components/ui
  so the canon can audit them consistently.
Why: Page-local one-off components and misplaced adopted components hide UI
  structure from provider checks. Keeping components in predictable locations
  makes adoption, provenance, reuse, and cleanup machine-checkable.
Category: standards
Severity: medium
Slot: [D]
SlotFile: ui/src/components
TechStack: React
Recommendation: Move reusable components to ui/src/components or ui/src/layout;
  keep governed component-library adoptions under ui/src/components/ui; keep
  page files focused on route composition.
Standard: vrooli-ui-component-canon-v1

GoodExample:
    // ui/src/components/DebtTable.tsx
    export function DebtTable() { return <section />; }
    // ui/src/pages/Fleet.tsx
    export function FleetPage() { return <DebtTable />; }

BadExample:
    // ui/src/pages/Fleet.tsx
    function DebtTable() { return <table />; }
    export function FleetPage() { return <DebtTable />; }

<test-case id="component-location-clean" should-fail="false">
  <description>Reusable components live under components and pages only compose them</description>
  <input>
    [ui/src/components/DebtTable.tsx]
    export function DebtTable() { return <section />; }
    [ui/src/pages/Fleet.tsx]
    import { DebtTable } from "../components/DebtTable";
    export function FleetPage() { return <DebtTable />; }
  </input>
</test-case>

<test-case id="component-location-feature-page-clean" should-fail="false">
  <description>Feature route page files can export the page when reusable components are extracted</description>
  <input>
    [ui/src/components/CaptureGallery.tsx]
    export function CaptureGallery() { return <section />; }
    [ui/src/features/captures/CaptureGalleryPage.tsx]
    import { CaptureGallery } from "../../components/CaptureGallery";
    export function CaptureGalleryPage() { return <CaptureGallery />; }
  </input>
</test-case>

<test-case id="component-location-app-infra-clean" should-fail="false">
  <description>App routes, providers, and hook wrappers are infrastructure rather than reusable UI inventory</description>
  <input>
    [ui/src/app/routes.tsx]
    export function AppRouter() { return <RouterProvider />; }
    [ui/src/app/providers.tsx]
    export function Providers() { return <QueryClientProvider />; }
    [ui/src/hooks/SpatialGroup.tsx]
    export function SpatialGroup() { return <div style={{ display: "contents" }} />; }
    [ui/src/theme/ThemeProvider.tsx]
    export function ThemeProvider() { return <ThemeContext.Provider />; }
  </input>
</test-case>

<test-case id="component-location-governed-ui-clean" should-fail="false">
  <description>Governed component-library adoptions live under components/ui</description>
  <input>
    [ui/src/components/ui/data-table.tsx]
    // @vrooliComponentSource react-component-library:DataTable
    // @vrooliComponentVersion 1.0.0
    export function DataTable() { return <table />; }
  </input>
</test-case>

<test-case id="component-location-page-local" should-fail="true">
  <description>Inline page-local components are flagged for extraction</description>
  <input>
    [ui/src/pages/Fleet.tsx]
    function DebtTable() { return <table />; }
    export function FleetPage() { return <DebtTable />; }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>DebtTable</expected-message>
</test-case>

<test-case id="component-location-misplaced-export" should-fail="true">
  <description>Exported reusable components outside components or layout are flagged</description>
  <input>
    [ui/src/widgets/StatusPanel.tsx]
    export function StatusPanel() { return <section>Status</section>; }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>ui/src/widgets/StatusPanel.tsx</expected-message>
</test-case>

<test-case id="component-location-governed-misplaced" should-fail="true">
  <description>Governed component-library adoptions outside components/ui are flagged</description>
  <input>
    [ui/src/components/DataTable.tsx]
    // @vrooliComponentSource react-component-library:DataTable
    // @vrooliComponentVersion 1.0.0
    export function DataTable() { return <table />; }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>components/ui</expected-message>
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
	uiinterop.Register("standard_component_location", checkComponentLocation)
}

var localComponentPattern = regexp.MustCompile(`(?m)\b(?:function|const)\s+([A-Z][A-Za-z0-9_]*)\b`)

func checkComponentLocation(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "standard_component_location"

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
		if !isReactComponentFile(f.relPath) {
			continue
		}
		if strings.Contains(f.content, "@vrooliComponentSource") && !isGovernedComponentLocation(f.relPath) {
			violations = append(violations, uiinterop.Violation{
				RuleID:         ruleID,
				Severity:       "medium",
				Title:          "Governed component adoption is outside components/ui",
				Description:    f.relPath + " carries component-library provenance but is not under ui/src/components/ui",
				FilePath:       f.relPath,
				Line:           lineOf(f.content, "@vrooliComponentSource"),
				Recommendation: "Move governed component-library adoptions under ui/src/components/ui so the canon can audit provenance and upgrades consistently.",
			})
			continue
		}
		if isComponentSource(f.relPath) {
			continue
		}
		if isAppInfrastructureSource(f.relPath) {
			continue
		}
		if !isPageSource(f.relPath) {
			for _, name := range exportedComponents(f.content) {
				violations = append(violations, misplacedComponentViolation(ruleID, f, name))
			}
			continue
		}
		for _, name := range localPageComponents(f.content) {
			violations = append(violations, uiinterop.Violation{
				RuleID:         ruleID,
				Severity:       "medium",
				Title:          "Page-local component should be extracted",
				Description:    f.relPath + " declares " + name + " beside the route; move reusable UI out of the page file",
				FilePath:       f.relPath,
				Line:           lineOf(f.content, name),
				Recommendation: "Move " + name + " to ui/src/components or ui/src/layout and import it from the page.",
			})
		}
	}

	if len(violations) > 0 {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Passed:     false,
			Message:    "component-location canon violations found",
			Violations: violations,
		}
	}
	return uiinterop.RuleResult{
		RuleID:  ruleID,
		Passed:  true,
		Message: "components live in canonical locations",
	}
}

func isReactComponentFile(relPath string) bool {
	ext := strings.ToLower(filepath.Ext(filepath.ToSlash(relPath)))
	return ext == ".tsx" || ext == ".jsx"
}

func isGovernedComponentLocation(relPath string) bool {
	rel := filepath.ToSlash(relPath)
	return strings.Contains(rel, "/components/ui/")
}

func isPageSource(relPath string) bool {
	rel := filepath.ToSlash(relPath)
	base := filepath.Base(rel)
	return strings.Contains(rel, "/pages/") || strings.Contains(rel, "/routes/") ||
		strings.HasSuffix(base, "Page.tsx") || strings.HasSuffix(base, "Page.jsx")
}

func isAppInfrastructureSource(relPath string) bool {
	rel := filepath.ToSlash(relPath)
	if strings.Contains(rel, "/app/") {
		return true
	}
	return strings.Contains(rel, "/hooks/") || strings.Contains(rel, "/theme/")
}

func misplacedComponentViolation(ruleID string, f uiSourceFile, name string) uiinterop.Violation {
	return uiinterop.Violation{
		RuleID:         ruleID,
		Severity:       "medium",
		Title:          "Reusable component is outside the component inventory",
		Description:    f.relPath + " exports " + name + " outside ui/src/components or ui/src/layout",
		FilePath:       f.relPath,
		Line:           lineOf(f.content, name),
		Recommendation: "Move " + name + " to ui/src/components or ui/src/layout so ui-health can audit the component inventory.",
	}
}

func localPageComponents(source string) []string {
	stripped := stripTSXComments(source)
	matches := localComponentPattern.FindAllStringSubmatch(stripped, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		name := m[1]
		if strings.Contains(stripped, "export function "+name) || strings.Contains(stripped, "export const "+name) {
			continue
		}
		if !strings.Contains(stripped, "<"+name) {
			continue
		}
		out = append(out, name)
	}
	return uniqueStrings(out)
}
