package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"experience-manager/internal/statevocab"
)

func checkIndexShape(report *Report, spec IndexDocument) {
	loc := "experience/index.json"
	if spec.Kind != kindIndex {
		report.add(CodeSchemaInvalid, SeverityError, "index kind must be experience-index", loc, "Set kind to experience-index.")
	}
	if spec.Contract.Kind != contractKind || spec.Contract.Schema != contractSchema {
		report.add(CodeSchemaInvalid, SeverityError, "index contract must be scenario-experience/spec v1", loc, "Set contract.kind and contract.schema to the scenario experience contract.")
	}
	if !schemaVersionPattern.MatchString(spec.SchemaVersion) {
		report.add(CodeSchemaInvalid, SeverityError, "schemaVersion must be semver-like", loc, "Use a version like 1.0.0.")
	}
	if !idPattern.MatchString(spec.Scenario) {
		report.add(CodeSchemaInvalid, SeverityError, "scenario must be a kebab-case id", loc, "Set scenario to the target scenario slug.")
	}
	checkDocRefs(report, loc, "page", spec.Pages)
	checkDocRefs(report, loc, "journey", spec.Journeys)
	checkDocRefs(report, loc, "component", spec.Components)
}

func checkDocRefs(report *Report, loc, kind string, refs []DocumentRef) {
	seenIDs := map[string]bool{}
	seenPaths := map[string]bool{}
	prefix := kind + "s/"
	for _, ref := range refs {
		if !idPattern.MatchString(ref.ID) {
			report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("%s ref id %q is not kebab-case", kind, ref.ID), loc, "Use stable kebab-case ids.")
		}
		if seenIDs[ref.ID] {
			report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("duplicate %s ref id %q", kind, ref.ID), loc, "Make each ref id unique.")
		}
		seenIDs[ref.ID] = true
		if !strings.HasPrefix(ref.Path, prefix) || !strings.HasSuffix(ref.Path, ".json") {
			report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("%s ref path %q must live under %s", kind, ref.Path, prefix), loc, "Point refs at pages/*.json, journeys/*.json, or components/*.json.")
		}
		if seenPaths[ref.Path] {
			report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("duplicate %s ref path %q", kind, ref.Path), loc, "List each document path once.")
		}
		seenPaths[ref.Path] = true
		if !validStatus(ref.Status) {
			report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("%s ref status %q is invalid", kind, ref.Status), loc, "Use draft, active, or deprecated.")
		}
	}
}

func checkIndexParity(report *Report, spec *ScenarioSpec) {
	checkDirParity(report, spec, "pages", spec.Index.Pages)
	checkDirParity(report, spec, "journeys", spec.Index.Journeys)
	checkDirParity(report, spec, "components", spec.Index.Components)
}

func checkDirParity(report *Report, spec *ScenarioSpec, dir string, refs []DocumentRef) {
	listed := map[string]bool{}
	for _, ref := range refs {
		listed[ref.Path] = true
		if _, err := os.Stat(filepath.Join(spec.ExperienceDir, filepath.FromSlash(ref.Path))); err != nil {
			report.add(CodeIndexParity, SeverityError, fmt.Sprintf("listed %s document %q is missing", dir, ref.Path), "experience/index.json", "Create the file or remove the stale index entry.")
		}
	}
	matches, _ := filepath.Glob(filepath.Join(spec.ExperienceDir, dir, "*.json"))
	for _, match := range matches {
		relPath := filepath.ToSlash(strings.TrimPrefix(match, spec.ExperienceDir+string(filepath.Separator)))
		if listed[relPath] {
			continue
		}
		// A component contract copied in by an adoption carries its library
		// provenance and is validated by the library that owns and harnesses
		// it. Requiring an index entry here puts the consumer in a bind: leave
		// it unlisted and index parity fails, list it and the contract's own
		// specimen reference and per-state bindings fail, because a consuming
		// scenario has no component harness to capture them against.
		if dir == "components" && adoptedComponentProvenance(match) != "" {
			continue
		}
		report.add(CodeIndexParity, SeverityError, fmt.Sprintf("%s document %q is not listed in index", dir, relPath), rel(spec.ExperienceDir, match), "Add the document to experience/index.json or remove it.")
	}
}

// adoptedComponentProvenance reports the library provenance recorded on a
// component document, or "" when the document is authored locally. Adoptions
// stamp this when they copy a library contract into a consuming scenario.
func adoptedComponentProvenance(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var doc struct {
		Contract struct {
			Provenance string `json:"provenance"`
		} `json:"contract"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return ""
	}
	return strings.TrimSpace(doc.Contract.Provenance)
}

func checkPages(report *Report, spec *ScenarioSpec, prdRefs map[string]bool) {
	indexByID := map[string]DocumentRef{}
	for _, ref := range spec.Index.Pages {
		indexByID[ref.ID] = ref
	}
	for id, page := range spec.Pages {
		loc := "experience/" + indexByID[id].Path
		checkPageShape(report, loc, page, indexByID[id])
		checkPRDRefs(report, loc, "page", page.Page.PRDRefs, prdRefs)
		checkPageReferences(report, loc, page, spec.Components)
		checkPinnedLibraryContracts(report, loc, filepath.Dir(spec.ExperienceDir), page)
	}
}

// checkPinnedLibraryContracts verifies canonical ownership when parsing inside
// a repository that contains the RCL library. Parser fixtures intentionally
// remain self-contained, so an absent sibling library means there is no local
// artifact to verify rather than a fabricated resolution failure.
func checkPinnedLibraryContracts(report *Report, loc, scenarioDir string, page PageDocument) {
	scenariosDir := filepath.Dir(scenarioDir)
	libraryRoot := filepath.Join(scenariosDir, "react-component-library", "library", "components")
	if info, err := os.Stat(libraryRoot); err != nil || !info.IsDir() {
		return
	}
	for _, region := range page.Regions {
		library := region.Component.Library
		if library == nil || !idPattern.MatchString(library.Component) || !schemaVersionPattern.MatchString(library.Version) {
			continue
		}
		contract := pinnedLibraryContractPath(libraryRoot, library.Component, library.Version)
		if contract == "" {
			contract = filepath.Join(libraryRoot, library.Component, "versions", library.Version, "experience-contract.json")
		}
		if _, err := os.Stat(contract); err != nil {
			report.add(CodeRefUnresolved, SeverityError, fmt.Sprintf("region %q pins library component %s@%s without a canonical experience contract", region.ID, library.Component, library.Version), loc, "Promote the component contract into the exact RCL version or change the pin to an available canonical version.")
		}
	}
}

func pinnedLibraryContractPath(libraryRoot, component, version string) string {
	entries, err := os.ReadDir(libraryRoot)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if !entry.IsDir() || canonicalLibraryComponentID(entry.Name()) != component {
			continue
		}
		return filepath.Join(libraryRoot, entry.Name(), "versions", version, "experience-contract.json")
	}
	return ""
}

func canonicalLibraryComponentID(name string) string {
	var out []rune
	for index, r := range strings.TrimSpace(name) {
		if unicode.IsUpper(r) && index > 0 {
			out = append(out, '-')
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out = append(out, unicode.ToLower(r))
		}
	}
	return strings.Trim(string(out), "-")
}

func checkPageShape(report *Report, loc string, page PageDocument, ref DocumentRef) {
	if page.Kind != kindPage {
		report.add(CodeSchemaInvalid, SeverityError, "page kind must be experience-page", loc, "Set kind to experience-page.")
	}
	if !schemaVersionPattern.MatchString(page.SchemaVersion) {
		report.add(CodeSchemaInvalid, SeverityError, "page schemaVersion must be semver-like", loc, "Use a version like 1.0.0.")
	}
	if page.Page.ID != ref.ID {
		report.add(CodeIndexParity, SeverityError, fmt.Sprintf("page id %q does not match index id %q", page.Page.ID, ref.ID), loc, "Keep index and page ids identical.")
	}
	if page.Page.Title == "" || len(page.Page.Routes) == 0 || len(page.Page.Purpose) < 15 {
		report.add(CodeSchemaInvalid, SeverityError, "page identity must include title, route, and purpose", loc, "Fill page.title, page.routes, and page.purpose.")
	}
	for _, route := range page.Page.Routes {
		if !routePattern.MatchString(route) {
			report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("route %q is invalid", route), loc, "Routes must start with / and contain no whitespace.")
		}
	}
	checkUniqueIDs(report, loc, "state", stateIDs(page.States))
	checkUniqueIDs(report, loc, "element", elementIDs(page.Elements))
	checkUniqueIDs(report, loc, "claim", claimIDs(page.Claims))
	checkUniqueIDs(report, loc, "region", regionIDs(page.Regions))
	checkFloorOptOuts(report, loc, page.FloorOptOuts)
	if pageRequiresRuntimeReadiness(page) && pageDeclaresAsyncLifecycle(page) && !pageHasRequiredAsyncRegion(page) {
		report.add(CodeSchemaInvalid, SeverityError, "page declares loading or recovery states without a required async region", loc, "Declare the primary async region with lifecycle.kind async and a runtime binding, or remove lifecycle states from a static page.")
	}
	for _, el := range page.Elements {
		if !idPattern.MatchString(el.ID) || !rolePattern.MatchString(el.Role) {
			report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("element %q has invalid id or role", el.ID), loc, "Use kebab-case ids and ARIA or x- namespaced roles.")
		}
	}
	for _, claim := range page.Claims {
		checkClaimShape(report, loc, claim)
	}
}

// Schema v1.1 is the readiness-contract revision. Older documents retain their
// existing descriptive page states while scenarios migrate deliberately; new
// v1.1+ contracts cannot name async page modes without machine-readable
// runtime ownership.
func pageRequiresRuntimeReadiness(page PageDocument) bool {
	return strings.TrimSpace(page.SchemaVersion) >= "1.1.0"
}

// Page states describe user-visible modes, while regions are the runtime
// contract consumed by BAS and UI Health. A page that names loading or recovery
// modes without a required async region is ambiguous: automation cannot know
// what must settle, and a shell mount can be mistaken for usable content.
func pageDeclaresAsyncLifecycle(page PageDocument) bool {
	for _, state := range page.States {
		id := strings.TrimSpace(state.ID)
		if id == "loading" || strings.HasSuffix(id, "-loading") || id == "error" || strings.HasSuffix(id, "-error") {
			return true
		}
	}
	return false
}

func pageHasRequiredAsyncRegion(page PageDocument) bool {
	for _, region := range page.Regions {
		if region.Required && region.Lifecycle.Kind == "async" {
			return true
		}
	}
	return false
}

func checkPageReferences(report *Report, loc string, page PageDocument, components map[string]ComponentDocument) {
	elementSet := stringSet(elementIDs(page.Elements))
	stateSet := stringSet(stateIDs(page.States))
	for _, claim := range page.Claims {
		for _, el := range claim.Elements {
			if !elementSet[el] {
				report.add(CodeRefUnresolved, SeverityError, fmt.Sprintf("claim %q references unknown element %q", claim.ID, el), loc, "Declare the element or remove the reference.")
			}
		}
		for _, state := range claim.States {
			if !stateSet[state] {
				report.add(CodeRefUnresolved, SeverityError, fmt.Sprintf("claim %q references unknown state %q", claim.ID, state), loc, "Declare the state or remove the reference.")
			}
		}
		if claim.Tier == "machine" {
			for _, el := range claim.Elements {
				if !bindingFor(page.Bindings.Elements, el) {
					report.add(CodeBindingOrphan, SeverityError, fmt.Sprintf("machine claim %q references unbound element %q", claim.ID, el), loc, "Add a binding for every machine-claimed element.")
				}
			}
		}
	}
	for el := range page.Bindings.Elements {
		if !elementSet[el] {
			report.add(CodeBindingOrphan, SeverityError, fmt.Sprintf("binding references unknown element %q", el), loc, "Remove the binding or declare the element.")
		}
	}
	for _, region := range page.Regions {
		checkRegionShape(report, loc, region, filepath.Dir(filepath.Dir(report.TargetPath)))
		if region.Component.Local != "" && region.Component.Library != nil {
			report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("region %q must reference exactly one local or library component", region.ID), loc, "Use either component.local or component.library.")
		}
		if region.Component.Local == "" && region.Component.Library == nil {
			report.add(CodeRefUnresolved, SeverityError, fmt.Sprintf("region %q has no component reference", region.ID), loc, "Reference a local component or a pinned library component.")
		}
		if region.Component.Local != "" {
			if !idPattern.MatchString(region.Component.Local) {
				report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("region %q has invalid local component id %q", region.ID, region.Component.Local), loc, "Use a kebab-case local component id.")
			}
			if _, ok := components[region.Component.Local]; !ok {
				report.add(CodeRefUnresolved, SeverityError, fmt.Sprintf("region %q references unknown local component %q", region.ID, region.Component.Local), loc, "Reference a component declared in experience/index.json.")
			}
		}
		if library := region.Component.Library; library != nil {
			if !idPattern.MatchString(library.Component) || !schemaVersionPattern.MatchString(library.Version) {
				report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("region %q has an invalid pinned library component", region.ID), loc, "Use a kebab-case component name and exact semver version.")
			}
		}
		if !bindingFor(page.Bindings.Regions, region.ID) {
			report.add(CodeBindingOrphan, SeverityError, fmt.Sprintf("region %q has no runtime binding", region.ID), loc, "Bind each meaningful region to a data-testid or selector.")
		}
	}
	for region := range page.Bindings.Regions {
		if !stringSet(regionIDs(page.Regions))[region] {
			report.add(CodeBindingOrphan, SeverityError, fmt.Sprintf("region binding references unknown region %q", region), loc, "Remove the binding or declare the region.")
		}
	}
	for _, el := range page.Elements {
		if !bindingFor(page.Bindings.Elements, el.ID) {
			report.add(CodeBindingOrphan, SeverityError, fmt.Sprintf("element %q has no binding", el.ID), loc, "Bind each element to a testid or selector.")
		}
	}
	for _, region := range page.Sketch.Regions {
		for _, el := range region.Elements {
			if !elementSet[el] {
				report.add(CodeRefUnresolved, SeverityError, fmt.Sprintf("sketch region %q references unknown element %q", region.ID, el), loc, "Keep sketch element refs aligned with declared elements.")
			}
		}
	}
}

func checkRegionShape(report *Report, loc string, region ExperienceRegion, repoRoot string) {
	if !idPattern.MatchString(region.ID) || len(strings.TrimSpace(region.Purpose)) < 10 {
		report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("region %q must include a kebab-case id and meaningful purpose", region.ID), loc, "Declare a stable region id and purpose.")
	}
	stateSet := stringSet(region.Lifecycle.States)
	if len(stateSet) != len(region.Lifecycle.States) || len(stateSet) == 0 {
		report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("region %q lifecycle must declare unique states", region.ID), loc, "Declare one or more lifecycle states.")
	}
	for state := range stateSet {
		if !validRegionLifecycleState(repoRoot, state, region.Lifecycle.Kind) {
			report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("region %q has unsupported lifecycle state %q", region.ID, state), loc, "Use loading, ready, empty, partial, error, or static.")
		}
	}
	switch region.Lifecycle.Kind {
	case "static":
		if len(stateSet) != 1 || !stateSet["static"] {
			report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("static region %q must declare only static lifecycle", region.ID), loc, "Static regions use lifecycle.kind static with states [static].")
		}
	case "async":
		if stateSet["static"] {
			report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("async region %q cannot declare static lifecycle", region.ID), loc, "Use loading, ready, empty, partial, or error for async regions.")
		}
	default:
		report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("region %q lifecycle kind %q is invalid", region.ID, region.Lifecycle.Kind), loc, "Use async or static.")
	}
}

func validRegionLifecycleState(repoRoot, state, kind string) bool {
	if kind == "static" && state == "static" {
		return true
	}
	return statevocab.RegionState(repoRoot, state)
}

func checkClaimShape(report *Report, loc string, claim Claim) {
	if !idPattern.MatchString(claim.ID) || !idPattern.MatchString(claim.Type) || len(claim.Statement) < 10 {
		report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("claim %q has invalid id, type, or statement", claim.ID), loc, "Use kebab-case ids/types and a meaningful statement.")
	}
	if !validTier(claim.Tier) {
		report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("claim %q tier %q is invalid", claim.ID, claim.Tier), loc, "Use machine, manual, or aspirational.")
		return
	}
	if claim.Type == "custom" && claim.Tier == "machine" {
		report.add(CodeTierViolation, SeverityError, fmt.Sprintf("custom claim %q cannot be machine tier", claim.ID), loc, "Use manual or aspirational for custom claims.")
	}
	if !knownClaimType(claim.Type) && claim.Tier != "aspirational" {
		report.add(CodeTierViolation, SeverityError, fmt.Sprintf("unknown claim type %q must be aspirational", claim.Type), loc, "Promote unknown claim types only after the validator learns them.")
	}
	checkStructuredClaimShape(report, loc, claim)
}

// checkStructuredClaimShape keeps the parameters of machine claims honest.
// Params remain open-world JSON, but a claim that names a built-in checker must
// carry the inputs that make that checker deterministic.
func checkStructuredClaimShape(report *Report, loc string, claim Claim) {
	if claim.Tier != "machine" {
		return
	}
	switch claim.Type {
	case "spacing":
		if len(claim.Elements) != 2 {
			report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("spacing claim %q must name exactly two elements", claim.ID), loc, "Name the two elements whose separation is part of the contract.")
		}
		if value, ok := numericParam(claim.Params, "minSeparation", "minGap"); !ok || value < 0 {
			report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("spacing claim %q must declare a non-negative params.minSeparation", claim.ID), loc, "Set params.minSeparation to the minimum CSS-pixel gap.")
		}
	case "state-contrast":
		if len(claim.Elements) == 0 {
			report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("state-contrast claim %q must name a control element", claim.ID), loc, "Reference the element whose foreground must remain readable.")
		}
		if strings.TrimSpace(paramString(claim.Params, "state")) == "" {
			report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("state-contrast claim %q must declare params.state", claim.ID), loc, "Name the interaction state being checked, such as hover.")
		}
		if strings.TrimSpace(paramString(claim.Params, "background", "backgroundElement")) == "" {
			report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("state-contrast claim %q must declare params.background", claim.ID), loc, "Name the background element or color used for the contrast check.")
		}
		if value, ok := numericParam(claim.Params, "minContrastRatio", "minContrast"); !ok || value < 1 {
			report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("state-contrast claim %q must declare params.minContrastRatio >= 1", claim.ID), loc, "Set the minimum WCAG contrast ratio.")
		}
	case "size-parity":
		if len(claim.Elements) == 0 || len(claim.Elements) > 2 {
			report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("size-parity claim %q must name one control or exactly two elements", claim.ID), loc, "Name control for a documented size rung, or name the two elements whose heights must match.")
		}
		if value, ok := numericParam(claim.Params, "tolerance"); ok && value < 0 {
			report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("size-parity claim %q tolerance must be non-negative", claim.ID), loc, "Use a non-negative CSS-pixel tolerance.")
		}
	case "differential":
		if strings.TrimSpace(claim.Subject) == "" || strings.TrimSpace(claim.Metric) == "" || len(claim.Contexts) < 2 || claim.Require != "contexts-differ" {
			report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("differential claim %q must declare subject, metric, two contexts, and require=contexts-differ", claim.ID), loc, "Declare two captured contexts with expected values and require contexts-differ.")
		}
		for _, context := range claim.Contexts {
			if strings.TrimSpace(context.ID) == "" || strings.TrimSpace(context.Story) == "" || strings.TrimSpace(context.Expect) == "" {
				report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("differential claim %q has an incomplete context", claim.ID), loc, "Each differential context needs id, story, and expect.")
			}
		}
	}
}
