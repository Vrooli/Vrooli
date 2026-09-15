/*
Rule: Component Adoption Contracts
ID: standard_component_adoption_contracts
Description: Linked React Component Library adopters keep their generated
  locale and selector registries at the application boundary and keep library
  assertions owned by the library.
Why: Adoption files are scenario-owned integration surfaces. Checking them in
  ui-health lets every adopter receive the same contract without making the
  component library scan unrelated scenarios.
Category: standards
Severity: medium
Slot: [D]
SlotFile: ui/src
TechStack: React
Recommendation: Re-run the governed component-library adoption link operation,
  and move library assertions back beside the versioned library source.
Standard: vrooli-ui-component-adoption-v1

<test-case id="adoption-contracts-unlinked" should-fail="false">
  <description>Unlinked scenarios do not owe adoption files</description>
  <input>
    [ui/package.json]
    {"dependencies":{"react":"^18.3.1"}}
  </input>
</test-case>

<test-case id="adoption-contracts-linked" should-fail="true">
  <description>A linked adopter must mount the provider and compose selectors</description>
  <input>
    [ui/package.json]
    {"dependencies":{"@vrooli/react-component-library":"workspace:*"}}
    [ui/src/main.tsx]
    export function Main() { return null; }
  </input>
  <expected-violations>3</expected-violations>
  <expected-message>adoption contracts are incomplete</expected-message>
</test-case>
*/

package checks

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vrooli/api-core/uimanifest"

	"ui-health/internal/uiinterop"
)

func init() {
	uiinterop.Register("standard_component_adoption_contracts", checkComponentAdoptionContracts)
}

var (
	adoptionRegistryCompositionRE = regexp.MustCompile(`createSelectorRegistry\s*\([^;]*,\s*librarySelectors\s*,?\s*\)`)
	adoptionLibraryImportRE       = regexp.MustCompile(`@vrooli/react-component-library`)
	adoptionSelectorIDRE          = regexp.MustCompile(`(?m)^\s*["']?id[0-9]+["']?\s*:`)
)

func checkComponentAdoptionContracts(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "standard_component_adoption_contracts"
	packageBytes, err := os.ReadFile(filepath.Join(ctx.ScenarioRoot, "ui", "package.json"))
	if err != nil || !adoptionLibraryImportRE.Match(packageBytes) {
		return uiinterop.RuleResult{RuleID: ruleID, Skipped: true, SkipReason: "react-component-library is not linked", Message: "react-component-library is not linked; skipping adoption contracts"}
	}

	var violations []uiinterop.Violation
	layout, layoutErr := uimanifest.LoadAt(ctx.ScenarioRoot)
	if layoutErr != nil {
		return uiinterop.RuleResult{RuleID: ruleID, Passed: false, Message: "Invalid UI file declarations: " + layoutErr.Error()}
	}
	mainPath := filepath.Join(ctx.ScenarioRoot, layout.ResolveFile("appEntry", "ui/src/main.tsx"))
	main, mainErr := os.ReadFile(mainPath)
	if mainErr != nil || !strings.Contains(string(main), "LibraryStringsProvider") || !strings.Contains(string(main), "i18n.t") {
		violations = append(violations, uiinterop.Violation{RuleID: ruleID, Severity: "medium", Title: "Library strings provider is not mounted", Description: "linked adopter main.tsx does not mount LibraryStringsProvider with its translator", FilePath: "ui/src/main.tsx", Recommendation: "Run the governed adoption link operation and keep LibraryStringsProvider at the application root."})
	}
	selectorsRel := layout.ResolveFile("librarySelectors", "ui/src/consts/selectors.library.ts")
	selectorsPath := filepath.Join(ctx.ScenarioRoot, selectorsRel)
	selectors, selectorsErr := os.ReadFile(selectorsPath)
	if selectorsErr != nil || !strings.Contains(string(selectors), "librarySelectors") {
		violations = append(violations, uiinterop.Violation{RuleID: ruleID, Severity: "medium", Title: "Library selector registry is not composed", Description: "linked adopter does not expose the generated library selector registry", FilePath: selectorsRel, Recommendation: "Re-run the governed adoption link operation."})
	} else {
		body := string(selectors)
		if match := adoptionSelectorIDRE.FindStringIndex(body); match != nil {
			violations = append(violations, uiinterop.Violation{RuleID: ruleID, Severity: "medium", Title: "Selector registry uses positional names", Description: "the generated library selector registry contains idN names", FilePath: selectorsRel, Line: strings.Count(body[:match[0]], "\n") + 1, Recommendation: "Use semantic selector names rooted at the dotted catalog id."})
		}

	}
	registryPath := layout.ResolveFile("selectorRegistry", "ui/src/consts/selectors.ts")
	registry, registryErr := os.ReadFile(filepath.Join(ctx.ScenarioRoot, registryPath))
	manifestRaw, manifestErr := os.ReadFile(filepath.Join(ctx.ScenarioRoot, layout.SelectorManifestPath()))
	var manifest struct {
		Selectors map[string]json.RawMessage `json:"selectors"`
		Sources   map[string]string          `json:"sources"`
	}
	if manifestErr == nil {
		manifestErr = json.Unmarshal(manifestRaw, &manifest)
	}
	fresh := manifest.Sources[registryPath] != "" && manifest.Sources[selectorsRel] != ""
	for source, digest := range manifest.Sources {
		clean := filepath.Clean(source)
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			fresh = false
			break
		}
		data, err := os.ReadFile(filepath.Join(ctx.ScenarioRoot, clean))
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(data)) != digest {
			fresh = false
			break
		}
	}
	if !fresh || registryErr != nil || !adoptionRegistryCompositionRE.Match(registry) || manifestErr != nil || manifest.Selectors == nil {
		violations = append(violations, uiinterop.Violation{RuleID: ruleID, Severity: "medium", Title: "Library selector composition is not exported", Description: "The declared application registry must compose librarySelectors and export a current generated manifest.", FilePath: registryPath, Recommendation: "Compose the shared selector registry and run selector:manifest."})
	}
	for _, file := range ctx.TestSources {
		if !adoptionLibraryImportRE.MatchString(file.Content) {
			continue
		}
		violations = append(violations, uiinterop.Violation{RuleID: ruleID, Severity: "medium", Title: "Library assertion remains in adopter", Description: "a scenario test imports the React Component Library directly", FilePath: file.RelPath, Recommendation: "Move library assertions beside the matching versioned library source."})
	}
	if len(violations) > 0 {
		return uiinterop.RuleResult{RuleID: ruleID, Passed: false, Violations: violations, Message: "linked component-library adoption contracts are incomplete"}
	}
	return uiinterop.RuleResult{RuleID: ruleID, Passed: true, Message: "linked component-library adoption contracts are healthy"}
}
