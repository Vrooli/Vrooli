package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"experience-manager/internal/claimtypes"
)

func checkJourneys(report *Report, spec *ScenarioSpec, prdRefs map[string]bool) {
	indexByID := map[string]DocumentRef{}
	for _, ref := range spec.Index.Journeys {
		indexByID[ref.ID] = ref
	}
	pageStates := map[string]map[string]bool{}
	for id, page := range spec.Pages {
		pageStates[id] = stringSet(stateIDs(page.States))
	}
	for id, journey := range spec.Journeys {
		loc := "experience/" + indexByID[id].Path
		if journey.Kind != kindJourney {
			report.add(CodeSchemaInvalid, SeverityError, "journey kind must be experience-journey", loc, "Set kind to experience-journey.")
		}
		if journey.Journey.ID != id {
			report.add(CodeIndexParity, SeverityError, fmt.Sprintf("journey id %q does not match index id %q", journey.Journey.ID, id), loc, "Keep index and journey ids identical.")
		}
		if journey.Journey.Title == "" || len(journey.Journey.Purpose) < 15 || len(journey.Steps) == 0 {
			report.add(CodeSchemaInvalid, SeverityError, "journey must include title, purpose, and at least one step", loc, "Fill the journey identity and steps.")
		}
		checkPRDRefs(report, loc, "journey", journey.Journey.PRDRefs, prdRefs)
		checkUniqueIDs(report, loc, "claim", claimIDs(journey.Claims))
		for _, claim := range journey.Claims {
			checkClaimShape(report, loc, claim)
		}
		for _, step := range journey.Steps {
			states, ok := pageStates[step.Page]
			if !ok {
				report.add(CodeRefUnresolved, SeverityError, fmt.Sprintf("journey step references unknown page %q", step.Page), loc, "Reference a page declared in experience/index.json.")
				continue
			}
			state := step.State
			if state == "" {
				state = defaultState
			}
			if !states[state] {
				report.add(CodeRefUnresolved, SeverityError, fmt.Sprintf("journey step references unknown state %q on page %q", state, step.Page), loc, "Reference a declared page state.")
			}
		}
	}
}

func checkComponents(report *Report, spec *ScenarioSpec, prdRefs map[string]bool) {
	indexByID := map[string]DocumentRef{}
	for _, ref := range spec.Index.Components {
		indexByID[ref.ID] = ref
	}
	for id, component := range spec.Components {
		loc := "experience/" + indexByID[id].Path
		path := filepath.Join(spec.ExperienceDir, filepath.FromSlash(indexByID[id].Path))
		checkComponentShape(report, loc, component, indexByID[id])
		checkPRDRefs(report, loc, "component", component.Component.PRDRefs, prdRefs)
		checkComponentReferences(report, loc, path, filepath.Dir(spec.ExperienceDir), component)
	}
}

func checkComponentShape(report *Report, loc string, component ComponentDocument, ref DocumentRef) {
	if component.Kind != kindComponent {
		report.add(CodeSchemaInvalid, SeverityError, "component kind must be experience-component", loc, "Set kind to experience-component.")
	}
	if !schemaVersionPattern.MatchString(component.SchemaVersion) {
		report.add(CodeSchemaInvalid, SeverityError, "component schemaVersion must be semver-like", loc, "Use a version like 1.1.0.")
	}
	if component.Component.ID != ref.ID {
		report.add(CodeIndexParity, SeverityError, fmt.Sprintf("component id %q does not match index id %q", component.Component.ID, ref.ID), loc, "Keep index and component ids identical.")
	}
	if component.Component.Title == "" || len(component.Component.Purpose) < 15 || (strings.TrimSpace(component.Component.ExamplesRef) == "" && strings.TrimSpace(component.Component.StoryRef) == "") {
		report.add(CodeSchemaInvalid, SeverityError, "component identity must include title, purpose, and a catalog story or examples reference", loc, "Fill component.title, component.purpose, and component.storyRef (or examplesRef for a legacy catalog).")
	}
	if component.Component.Extends != nil {
		if !idPattern.MatchString(component.Component.Extends.Component) || !schemaVersionPattern.MatchString(component.Component.Extends.Version) {
			report.add(CodeSchemaInvalid, SeverityError, "component extends must pin a kebab-case library component and exact semver version", loc, "Set component.extends.component and component.extends.version to the canonical RCL pin.")
		}
		if component.Component.Extension == nil || len(strings.TrimSpace(component.Component.Extension.Purpose)) < 15 {
			report.add(CodeSchemaInvalid, SeverityError, "component wrapper must declare an additive extension purpose", loc, "Add component.extension.purpose explaining the scenario-local behavior.")
		}
	}
	checkUniqueIDs(report, loc, "state", componentStateIDs(component.States))
	checkUniqueIDs(report, loc, "element", elementIDs(component.Elements))
	checkUniqueIDs(report, loc, "claim", claimIDs(component.Claims))
	for _, state := range component.States {
		if !idPattern.MatchString(state.Example) {
			report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("component state %q has invalid example %q", state.ID, state.Example), loc, "Anchor each component state to a kebab-case example name.")
		}
	}
	for _, el := range component.Elements {
		if !idPattern.MatchString(el.ID) || !rolePattern.MatchString(el.Role) {
			report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("element %q has invalid id or role", el.ID), loc, "Use kebab-case ids and ARIA or x- namespaced roles.")
		}
	}
	for _, claim := range component.Claims {
		checkClaimShape(report, loc, claim)
	}
}

func checkComponentReferences(report *Report, loc, path, scenarioDir string, component ComponentDocument) {
	elementSet := stringSet(elementIDs(component.Elements))
	stateSet := stringSet(componentStateIDs(component.States))
	specimenSet := componentSpecimens(filepath.Dir(path), component.Component.StoryRef, component.Component.ExamplesRef)
	for _, claim := range component.Claims {
		for _, el := range claim.Elements {
			if !elementSet[el] {
				report.add(CodeRefUnresolved, SeverityError, fmt.Sprintf("claim %q references unknown element %q", claim.ID, el), loc, "Declare the element or remove the reference.")
			}
		}
		for _, state := range claim.States {
			if !stateSet[state] {
				report.add(CodeRefUnresolved, SeverityError, fmt.Sprintf("claim %q references unknown state %q", claim.ID, state), loc, "Declare the component state or remove the reference.")
			}
		}
		if claim.Tier == "machine" {
			for _, el := range claim.Elements {
				if !bindingFor(component.Bindings.Elements, el) {
					report.add(CodeBindingOrphan, SeverityError, fmt.Sprintf("machine claim %q references unbound element %q", claim.ID, el), loc, "Add a binding for every machine-claimed element.")
				}
			}
		}
	}
	for el := range component.Bindings.Elements {
		if !elementSet[el] {
			report.add(CodeBindingOrphan, SeverityError, fmt.Sprintf("binding references unknown element %q", el), loc, "Remove the binding or declare the element.")
		}
	}
	for _, el := range component.Elements {
		if !bindingFor(component.Bindings.Elements, el.ID) {
			report.add(CodeBindingOrphan, SeverityError, fmt.Sprintf("element %q has no binding", el.ID), loc, "Bind each element to a testid or selector.")
		}
	}
	for _, state := range component.States {
		if len(specimenSet) == 0 {
			ref := firstNonEmpty(component.Component.StoryRef, component.Component.ExamplesRef)
			report.add(CodeRefUnresolved, SeverityError, fmt.Sprintf("component catalog reference %q has no readable specimens", ref), loc, "Point storyRef at a readable catalog story.json file (or examplesRef at a legacy catalog file).")
			break
		}
		if !specimenSet[state.Example] {
			report.add(CodeRefUnresolved, SeverityError, fmt.Sprintf("component state %q references unknown specimen %q", state.ID, state.Example), loc, "Reference a named story or legacy example declared by the catalog asset.")
		}
	}
	checkWrapperExtension(report, loc, scenarioDir, component)
}

// checkWrapperExtension makes local extension ownership explicit. Canonical
// contracts own their lifecycle and claims; a scenario wrapper can add new
// behavior but may not silently redefine an existing canonical identifier.
func checkWrapperExtension(report *Report, loc, scenarioDir string, component ComponentDocument) {
	pin := component.Component.Extends
	if pin == nil || !idPattern.MatchString(pin.Component) || !schemaVersionPattern.MatchString(pin.Version) {
		return
	}
	libraryRoot := filepath.Join(filepath.Dir(scenarioDir), "react-component-library", "library", "components")
	contractPath := pinnedLibraryContractPath(libraryRoot, pin.Component, pin.Version)
	if contractPath == "" {
		contractPath = filepath.Join(libraryRoot, pin.Component, "versions", pin.Version, "experience-contract.json")
	}
	raw, err := os.ReadFile(contractPath)
	if err != nil {
		report.add(CodeRefUnresolved, SeverityError, fmt.Sprintf("component wrapper %q pins %s@%s without a canonical experience contract", component.Component.ID, pin.Component, pin.Version), loc, "Promote the canonical contract into the exact RCL version before creating a wrapper.")
		return
	}
	var canonical struct {
		Component struct {
			ID string `json:"id"`
		} `json:"component"`
		States []struct {
			ID string `json:"id"`
		} `json:"states"`
		Claims []struct {
			ID string `json:"id"`
		} `json:"claims"`
	}
	if err := json.Unmarshal(raw, &canonical); err != nil {
		report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("component wrapper %q pins an unreadable canonical contract", component.Component.ID), loc, "Repair the pinned RCL experience-contract.json before extending it.")
		return
	}
	if canonical.Component.ID != "" && canonical.Component.ID != pin.Component {
		report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("component wrapper %q pin %s@%s identifies canonical component %q", component.Component.ID, pin.Component, pin.Version, canonical.Component.ID), loc, "Pin the library component id declared by the canonical contract.")
	}
	canonicalStates := map[string]bool{}
	for _, state := range canonical.States {
		canonicalStates[state.ID] = true
	}
	for _, state := range component.States {
		if canonicalStates[state.ID] {
			report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("component wrapper %q redefines canonical state %q", component.Component.ID, state.ID), loc, "Remove the overlapping state; wrappers may add behavior but cannot override canonical lifecycle semantics.")
		}
	}
	canonicalClaims := map[string]bool{}
	for _, claim := range canonical.Claims {
		canonicalClaims[claim.ID] = true
	}
	for _, claim := range component.Claims {
		if canonicalClaims[claim.ID] {
			report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("component wrapper %q redefines canonical claim %q", component.Component.ID, claim.ID), loc, "Use a new claim id for additive wrapper behavior; canonical claims remain authoritative.")
		}
	}
}

func componentSpecimens(baseDir, storyRef, examplesRef string) map[string]bool {
	ref := strings.TrimSpace(firstNonEmpty(storyRef, examplesRef))
	if ref == "" {
		return nil
	}
	data, err := os.ReadFile(filepath.Clean(filepath.Join(baseDir, filepath.FromSlash(ref))))
	if err != nil {
		return nil
	}
	var doc struct {
		Examples []struct {
			Name string `json:"name"`
		} `json:"examples"`
		Stories []struct {
			ID string `json:"id"`
		} `json:"stories"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil
	}
	out := map[string]bool{}
	for _, example := range doc.Examples {
		if example.Name != "" {
			out[example.Name] = true
		}
	}
	for _, story := range doc.Stories {
		if story.ID != "" {
			out[story.ID] = true
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func checkPRDRefs(report *Report, loc, owner string, refs []string, prdRefs map[string]bool) {
	for _, ref := range refs {
		if !prdRefs[ref] {
			report.add(CodePRDRefUnmatched, SeverityError, fmt.Sprintf("%s PRD ref %q does not resolve", owner, ref), loc, "Reference an operational target declared in PRD.md.")
		}
	}
}

func loadPRDRefs(scenarioDir string) map[string]bool {
	data, err := os.ReadFile(filepath.Join(scenarioDir, "PRD.md"))
	if err != nil {
		return map[string]bool{}
	}
	out := map[string]bool{}
	for _, match := range prdRefPattern.FindAllString(string(data), -1) {
		out[match] = true
	}
	return out
}

func validStatus(status string) bool {
	return status == "draft" || status == "active" || status == "deprecated"
}

func validTier(tier string) bool {
	return tier == "machine" || tier == "manual" || tier == "aspirational"
}

func knownClaimType(t string) bool {
	return t == "custom" || claimtypes.IsImplemented(t)
}

func paramString(params map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := params[key].(string); ok {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func numericParam(params map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		switch value := params[key].(type) {
		case float64:
			return value, true
		case float32:
			return float64(value), true
		case int:
			return float64(value), true
		case int64:
			return float64(value), true
		case json.Number:
			parsed, err := value.Float64()
			if err == nil {
				return parsed, true
			}
		}
	}
	return 0, false
}

func checkUniqueIDs(report *Report, loc, kind string, ids []string) {
	seen := map[string]bool{}
	for _, id := range ids {
		if !idPattern.MatchString(id) {
			report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("%s id %q is not kebab-case", kind, id), loc, "Use stable kebab-case ids.")
		}
		if seen[id] {
			report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("duplicate %s id %q", kind, id), loc, "Make ids unique within the document.")
		}
		seen[id] = true
	}
}

func bindingFor(bindings map[string]Binding, id string) bool {
	binding, ok := bindings[id]
	if !ok {
		return false
	}
	return binding.TestID != "" || binding.Selector != ""
}

func (r *Report) add(code, severity, message, location, suggestion string) {
	if !IsFindingCode(code) {
		panic(fmt.Sprintf("unregistered experience finding code %q", code))
	}
	r.Findings = append(r.Findings, Finding{
		Code:       code,
		Severity:   severity,
		Message:    message,
		Locations:  []string{location},
		Suggestion: suggestion,
	})
}

func sortFindings(findings []Finding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Code != findings[j].Code {
			return findings[i].Code < findings[j].Code
		}
		return strings.Join(findings[i].Locations, ",") < strings.Join(findings[j].Locations, ",")
	})
}

func rel(root, path string) string {
	out, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(out)
}

func stateIDs(states []State) []string {
	out := make([]string, 0, len(states))
	for _, state := range states {
		out = append(out, state.ID)
	}
	return out
}

func componentStateIDs(states []ComponentState) []string {
	out := make([]string, 0, len(states))
	for _, state := range states {
		out = append(out, state.ID)
	}
	return out
}

func elementIDs(elements []Element) []string {
	out := make([]string, 0, len(elements))
	for _, element := range elements {
		out = append(out, element.ID)
	}
	return out
}

func regionIDs(regions []ExperienceRegion) []string {
	out := make([]string, 0, len(regions))
	for _, region := range regions {
		out = append(out, region.ID)
	}
	return out
}

func claimIDs(claims []Claim) []string {
	out := make([]string, 0, len(claims))
	for _, claim := range claims {
		out = append(out, claim.ID)
	}
	return out
}

func stringSet(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		out[value] = true
	}
	return out
}
