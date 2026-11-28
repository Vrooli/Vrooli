/*
Rule: Component Canon Unengaged
ID: standard_component_canon_unengaged
Description: React UIs should either adopt at least one governed component or
  declare component-library intent so ui-health can steer raw primitive usage
  against the shared component canon.
Why: A React scenario with no adopted catalog component and no declared
  component-library intent is outside the governed-component ecosystem. That
  makes canon adoption invisible until a later migration, so the scenario needs
  an explicit advisory finding rather than a silent pass.
Category: standards
Severity: low
Slot: [D]
SlotFile: ui/src
TechStack: React
Recommendation: Adopt a relevant component from react-component-library under
  ui/src/components/ui or declare react-vite component-library intent in
  package.json, service.json, or ui/manifest.json.
Standard: vrooli-ui-component-canon-v1

GoodExample:
    // @vrooliComponentSource react-component-library:Button
    export function Button() { return <button />; }

BadExample:
    export function App() { return <button type="button">Save</button>; }

<test-case id="component-canon-local-adoption-clean" should-fail="false">
  <description>A local governed component adoption engages the canon</description>
  <input>
    [ui/package.json]
    {"dependencies":{"react":"^18.3.1"}}
    [ui/src/components/ui/button.tsx]
    // @vrooliComponentSource react-component-library:Button
    // @vrooliComponentVersion 1.1.0
    export function Button() { return <button />; }
    [ui/src/App.tsx]
    export function App() { return <Button />; }
  </input>
</test-case>

<test-case id="component-canon-declared-intent-clean" should-fail="false">
  <description>Declared react-vite template intent engages the canon even before local adoption</description>
  <input>
    [ui/package.json]
    {"dependencies":{"react":"^18.3.1"}}
    [ui/manifest.json]
    {"contract":{"template":"react-vite"}}
    [ui/src/App.tsx]
    export function App() { return <button type="button">Save</button>; }
  </input>
</test-case>

<test-case id="component-canon-unengaged-react" should-fail="true">
  <description>A React UI with no adoptions and no intent is reported</description>
  <input>
    [ui/package.json]
    {"dependencies":{"react":"^18.3.1"}}
    [ui/src/App.tsx]
    export function App() { return <button type="button">Save</button>; }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>outside the governed-component ecosystem</expected-message>
</test-case>
*/

package checks

import (
	"path/filepath"

	"ui-health/internal/uiinterop"
)

func init() {
	uiinterop.Register("standard_component_canon_unengaged", checkComponentCanonUnengaged)
}

func checkComponentCanonUnengaged(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "standard_component_canon_unengaged"

	files := sourceFiles(ctx, "ui/src")
	if len(files) == 0 {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "no ui/src directory found",
			Message:    "no ui/src directory found; skipping component canon engagement check",
		}
	}
	if hasLocalGovernedComponentAdoption(files) || declaresComponentLibraryIntent(ctx) {
		return uiinterop.RuleResult{
			RuleID:  ruleID,
			Passed:  true,
			Message: "component canon is engaged through local adoption or declared intent",
		}
	}
	return uiinterop.RuleResult{
		RuleID:  ruleID,
		Passed:  false,
		Message: "React UI is outside the governed-component ecosystem",
		Violations: []uiinterop.Violation{{
			RuleID:         ruleID,
			Severity:       "low",
			Title:          "Component canon not engaged",
			Description:    "React UI is outside the governed-component ecosystem: no local react-component-library adoption and no declared component-library intent were found.",
			FilePath:       filepath.ToSlash(filepath.Join("ui", "src")),
			Recommendation: "Adopt at least one governed component under ui/src/components/ui or declare component-library intent through the react-vite template/design metadata.",
		}},
	}
}
