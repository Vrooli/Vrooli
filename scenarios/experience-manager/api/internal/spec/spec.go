// Package spec owns the scenario-experience-spec/v1 parser surface.
package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

const (
	CodeSchemaInvalid       = "experience.schema_invalid"
	CodeIndexParity         = "experience.index_parity"
	CodeRefUnresolved       = "experience.ref_unresolved"
	CodePRDRefUnmatched     = "experience.prd_ref_unmatched"
	CodeBindingOrphan       = "experience.binding_orphan"
	CodeTierViolation       = "experience.tier_violation"
	CodeRouteUnspecced      = "experience.route_unspecced"
	CodeStateMissing        = "experience.state_missing"
	CodeBindingUnresolved   = "experience.binding_unresolved"
	CodeClaimFailed         = "experience.claim_failed"
	CodeClaimUnverifiable   = "experience.claim_unverifiable"
	CodeAffordanceMissing   = "experience.affordance_missing"
	CodeCaptureUnavailable  = "experience.capture_unavailable"
	CodeAttestationExpired  = "experience.attestation_expired"
	CodeClaimUnproven       = "experience.claim_unproven"
	CodeImportanceMismatch  = "experience.importance_mismatch"
	CodeGlanceJudgeMismatch = "experience.glance_judge_mismatch"
	CodeFloorNoDocOverflow  = "experience.floor_no_document_horizontal_overflow"
	CodeFloorViewportFill   = "experience.floor_viewport_fill"
	CodeFloorChromePinned   = "experience.floor_chrome_pinned"
	CodeFloorSafeArea       = "experience.floor_safe_area_tap_targets"
	CodeFloorSingleLine     = "experience.floor_single_line_chrome"
	CodeFloorTapTargetSize  = "experience.floor_tap_target_size"
	SeverityError           = "SEVERITY_ERROR"
	SeverityWarning         = "SEVERITY_WARNING"
	SeverityInfo            = "SEVERITY_INFO"
	contractKind            = "scenario-experience"
	contractSchema          = "scenario-experience-spec/v1"
	kindIndex               = "experience-index"
	kindPage                = "experience-page"
	kindJourney             = "experience-journey"
	kindComponent           = "experience-component"
	defaultState            = "default"
)

var (
	idPattern            = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
	schemaVersionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	prdRefPattern        = regexp.MustCompile(`OT-P[0-9]-[0-9]{3}`)
	routePattern         = regexp.MustCompile(`^/\S*$`)
	rolePattern          = regexp.MustCompile(`^([a-z]+|x-[a-z][a-z0-9-]*)$`)
)

// AllFindingCodes is the frozen experience finding vocabulary from
// docs/reference/experience-alignment.md.
var AllFindingCodes = []string{
	CodeSchemaInvalid,
	CodeIndexParity,
	CodeRefUnresolved,
	CodePRDRefUnmatched,
	CodeBindingOrphan,
	CodeTierViolation,
	CodeRouteUnspecced,
	CodeStateMissing,
	CodeBindingUnresolved,
	CodeClaimFailed,
	CodeClaimUnverifiable,
	CodeAffordanceMissing,
	CodeCaptureUnavailable,
	CodeAttestationExpired,
	CodeClaimUnproven,
	CodeImportanceMismatch,
	CodeGlanceJudgeMismatch,
	CodeFloorNoDocOverflow,
	CodeFloorViewportFill,
	CodeFloorChromePinned,
	CodeFloorSafeArea,
	CodeFloorSingleLine,
	CodeFloorTapTargetSize,
}

// IsFindingCode reports whether code is in the frozen experience finding
// registry.
func IsFindingCode(code string) bool {
	for _, known := range AllFindingCodes {
		if code == known {
			return true
		}
	}
	return false
}

// Report is the parser-facing contract result for one scenario.
type Report struct {
	Scenario        string
	TargetPath      string
	Findings        []Finding
	DegradedReason  string
	Spec            *ScenarioSpec
	PageDepths      map[string]int
	ComponentDepths map[string]int
}

// Finding is the neutral finding shape emitted by the parser and later checks.
type Finding struct {
	Code       string
	Severity   string
	Message    string
	Locations  []string
	Suggestion string
}

// ScenarioSpec is the parsed in-memory model for a target scenario's
// experience/ contract.
type ScenarioSpec struct {
	ExperienceDir string
	Index         IndexDocument
	Pages         map[string]PageDocument
	Journeys      map[string]JourneyDocument
	Components    map[string]ComponentDocument
}

type IndexDocument struct {
	Kind          string                     `json:"kind"`
	Contract      Contract                   `json:"contract"`
	SchemaVersion string                     `json:"schemaVersion"`
	Scenario      string                     `json:"scenario"`
	Description   string                     `json:"description"`
	Pages         []DocumentRef              `json:"pages"`
	Journeys      []DocumentRef              `json:"journeys"`
	Components    []DocumentRef              `json:"components"`
	Extensions    map[string]json.RawMessage `json:"-"`
}

type Contract struct {
	Kind   string `json:"kind"`
	Schema string `json:"schema"`
}

type DocumentRef struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

type PageDocument struct {
	Kind          string                     `json:"kind"`
	SchemaVersion string                     `json:"schemaVersion"`
	Page          PageIdentity               `json:"page"`
	Priorities    []Priority                 `json:"priorities"`
	States        []State                    `json:"states"`
	Elements      []Element                  `json:"elements"`
	Claims        []Claim                    `json:"claims"`
	Regions       []ExperienceRegion         `json:"regions"`
	Bindings      Bindings                   `json:"bindings"`
	FloorOptOuts  []FloorOptOut              `json:"floorOptOuts"`
	Sketch        Sketch                     `json:"sketch"`
	Extensions    map[string]json.RawMessage `json:"-"`
}

type FloorOptOut struct {
	Floor  string `json:"floor"`
	Reason string `json:"reason"`
}

type PageIdentity struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Routes  []string `json:"routes"`
	Purpose string   `json:"purpose"`
	PRDRefs []string `json:"prd_refs"`
}

type Priority struct {
	Statement string `json:"statement"`
	Notes     string `json:"notes"`
}

type State struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Setup       Setup  `json:"setup"`
}

type Setup struct {
	Route    string            `json:"route"`
	Query    map[string]string `json:"query"`
	Hash     string            `json:"hash"`
	SettleMs int               `json:"settleMs"`
}

type Element struct {
	ID          string `json:"id"`
	Role        string `json:"role"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Claim struct {
	ID         string                     `json:"id"`
	Type       string                     `json:"type"`
	Statement  string                     `json:"statement"`
	Tier       string                     `json:"tier"`
	Elements   []string                   `json:"elements"`
	States     []string                   `json:"states"`
	Viewports  []string                   `json:"viewports"`
	Locales    []string                   `json:"locales"`
	Params     map[string]any             `json:"params"`
	Rationale  string                     `json:"rationale"`
	Extensions map[string]json.RawMessage `json:"-"`
}

func (c *Claim) UnmarshalJSON(data []byte) error {
	type alias Claim
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	decoded.Extensions = extensionFields(raw)
	*c = Claim(decoded)
	return nil
}

type Bindings struct {
	Elements map[string]Binding `json:"elements"`
	Regions  map[string]Binding `json:"regions"`
}

// ExperienceRegion is the authored composition boundary for a meaningful
// independently rendered surface. Its stable identity and lifecycle intent are
// separate from the volatile selector/testid held in Bindings.Regions.
type ExperienceRegion struct {
	ID        string             `json:"id"`
	Purpose   string             `json:"purpose"`
	Required  bool               `json:"required"`
	Component ComponentReference `json:"component"`
	Lifecycle RegionLifecycle    `json:"lifecycle"`
}

// UnmarshalJSON applies the authored contract's required-by-default rule.
// A bool alone cannot distinguish an omitted field from explicit false.
func (r *ExperienceRegion) UnmarshalJSON(data []byte) error {
	var raw struct {
		ID        string             `json:"id"`
		Purpose   string             `json:"purpose"`
		Required  *bool              `json:"required"`
		Component ComponentReference `json:"component"`
		Lifecycle RegionLifecycle    `json:"lifecycle"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.ID, r.Purpose, r.Component, r.Lifecycle = raw.ID, raw.Purpose, raw.Component, raw.Lifecycle
	r.Required = true
	if raw.Required != nil {
		r.Required = *raw.Required
	}
	return nil
}

type ComponentReference struct {
	Local   string               `json:"local"`
	Library *LibraryComponentRef `json:"library"`
}

type LibraryComponentRef struct {
	Component string `json:"component"`
	Version   string `json:"version"`
}

type ComponentExtension struct {
	Purpose string `json:"purpose"`
}

type RegionLifecycle struct {
	Kind   string   `json:"kind"`
	States []string `json:"states"`
}

type Binding struct {
	TestID   string `json:"testid"`
	Selector string `json:"selector"`
	Note     string `json:"note"`
}

type Sketch struct {
	Regions []SketchRegion `json:"regions"`
}

type SketchRegion struct {
	ID       string   `json:"id"`
	Elements []string `json:"elements"`
}

type JourneyDocument struct {
	Kind          string                     `json:"kind"`
	SchemaVersion string                     `json:"schemaVersion"`
	Journey       JourneyIdentity            `json:"journey"`
	Steps         []JourneyStep              `json:"steps"`
	Claims        []Claim                    `json:"claims"`
	Extensions    map[string]json.RawMessage `json:"-"`
}

type JourneyIdentity struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Purpose string   `json:"purpose"`
	PRDRefs []string `json:"prd_refs"`
}

type JourneyStep struct {
	Page   string `json:"page"`
	State  string `json:"state"`
	Intent string `json:"intent"`
	Via    string `json:"via"`
}

type ComponentDocument struct {
	Kind          string                     `json:"kind"`
	SchemaVersion string                     `json:"schemaVersion"`
	Component     ComponentIdentity          `json:"component"`
	Priorities    []Priority                 `json:"priorities"`
	States        []ComponentState           `json:"states"`
	Elements      []Element                  `json:"elements"`
	Claims        []Claim                    `json:"claims"`
	Bindings      Bindings                   `json:"bindings"`
	FloorOptOuts  []FloorOptOut              `json:"floorOptOuts"`
	Extensions    map[string]json.RawMessage `json:"-"`
}

type ComponentIdentity struct {
	ID          string               `json:"id"`
	Title       string               `json:"title"`
	Purpose     string               `json:"purpose"`
	ExamplesRef string               `json:"examplesRef"`
	Extends     *LibraryComponentRef `json:"extends"`
	Extension   *ComponentExtension  `json:"extension"`
	PRDRefs     []string             `json:"prd_refs"`
}

type ComponentState struct {
	ID          string `json:"id"`
	Example     string `json:"example"`
	Description string `json:"description"`
}

// ParseScenario validates and parses scenarioDir/experience.
func ParseScenario(scenarioDir string) (Report, error) {
	scenarioDir = filepath.Clean(scenarioDir)
	report := Report{
		Scenario:        filepath.Base(scenarioDir),
		TargetPath:      scenarioDir,
		PageDepths:      map[string]int{},
		ComponentDepths: map[string]int{},
	}
	expDir := filepath.Join(scenarioDir, "experience")
	if info, err := os.Stat(expDir); err != nil || !info.IsDir() {
		if err != nil {
			return report, fmt.Errorf("experience directory %q: %w", expDir, err)
		}
		return report, fmt.Errorf("experience path %q is not a directory", expDir)
	}
	if _, err := os.Stat(filepath.Join(scenarioDir, "DESIGN.md")); err != nil {
		report.DegradedReason = "design contract absent; DESIGN.md state coverage is advisory-only"
	}

	spec := &ScenarioSpec{
		ExperienceDir: expDir,
		Pages:         map[string]PageDocument{},
		Journeys:      map[string]JourneyDocument{},
		Components:    map[string]ComponentDocument{},
	}
	report.Spec = spec

	spec.Index = parseIndex(&report, filepath.Join(expDir, "index.json"))
	if len(report.Findings) > 0 && spec.Index.Kind == "" {
		return report, nil
	}
	prdRefs := loadPRDRefs(scenarioDir)
	parseListedDocuments(&report, spec)
	checkIndexParity(&report, spec)
	checkIndexShape(&report, spec.Index)
	checkPages(&report, spec, prdRefs)
	checkJourneys(&report, spec, prdRefs)
	checkComponents(&report, spec, prdRefs)
	for id, page := range spec.Pages {
		report.PageDepths[id] = ComputeDepth(page, spec)
	}
	for id, component := range spec.Components {
		report.ComponentDepths[id] = ComputeComponentDepth(component)
	}
	sortFindings(report.Findings)
	return report, nil
}

// ComputeComponentDepth returns the L0-L3 experience depth for one component.
// Components do not participate in journeys; example-anchored states are their
// highest parser-era depth.
func ComputeComponentDepth(component ComponentDocument) int {
	if component.Component.ID == "" || component.Component.ExamplesRef == "" {
		return 0
	}
	depth := 0
	if len(component.Priorities) > 0 {
		depth = 1
	}
	if len(component.Claims) > 0 && len(component.Elements) > 0 && len(component.Bindings.Elements) > 0 {
		depth = 2
	}
	if len(component.States) > 0 {
		depth = 3
	}
	return depth
}

// ComputeDepth returns the L0-L4 experience depth for one page.
func ComputeDepth(page PageDocument, spec *ScenarioSpec) int {
	if page.Page.ID == "" || len(page.Page.Routes) == 0 {
		return 0
	}
	depth := 0
	if len(page.Priorities) > 0 {
		depth = 1
	}
	if len(page.Claims) > 0 && len(page.Elements) > 0 && len(page.Bindings.Elements) > 0 {
		depth = 2
	}
	if len(page.States) > 0 {
		depth = 3
	}
	for _, journey := range spec.Journeys {
		for _, step := range journey.Steps {
			if step.Page == page.Page.ID {
				return 4
			}
		}
	}
	return depth
}

func parseIndex(report *Report, path string) IndexDocument {
	var doc IndexDocument
	if !decodeDoc(report, path, &doc) {
		return doc
	}
	doc.Extensions = extensions(path, report)
	return doc
}

func parseListedDocuments(report *Report, spec *ScenarioSpec) {
	for _, ref := range spec.Index.Pages {
		var page PageDocument
		path := filepath.Join(spec.ExperienceDir, filepath.FromSlash(ref.Path))
		if !decodeDoc(report, path, &page) {
			continue
		}
		page.Extensions = extensions(path, report)
		if page.Page.ID != "" {
			spec.Pages[page.Page.ID] = page
		}
	}
	for _, ref := range spec.Index.Journeys {
		var journey JourneyDocument
		path := filepath.Join(spec.ExperienceDir, filepath.FromSlash(ref.Path))
		if !decodeDoc(report, path, &journey) {
			continue
		}
		journey.Extensions = extensions(path, report)
		if journey.Journey.ID != "" {
			spec.Journeys[journey.Journey.ID] = journey
		}
	}
	for _, ref := range spec.Index.Components {
		var component ComponentDocument
		path := filepath.Join(spec.ExperienceDir, filepath.FromSlash(ref.Path))
		if !decodeDoc(report, path, &component) {
			continue
		}
		component.Extensions = extensions(path, report)
		if component.Component.ID != "" {
			spec.Components[component.Component.ID] = component
		}
	}
}

func decodeDoc(report *Report, path string, out any) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("read document: %v", err), rel(report.TargetPath, path), "Create or restore the referenced experience document.")
		return false
	}
	if err := json.Unmarshal(data, out); err != nil {
		report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("invalid JSON: %v", err), rel(report.TargetPath, path), "Repair the JSON syntax.")
		return false
	}
	return true
}

func extensions(path string, report *Report) map[string]json.RawMessage {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	return extensionFields(raw)
}

func extensionFields(raw map[string]json.RawMessage) map[string]json.RawMessage {
	out := map[string]json.RawMessage{}
	for key, value := range raw {
		if strings.HasPrefix(key, "x-") {
			out[key] = value
		}
	}
	return out
}

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
		if !listed[relPath] {
			report.add(CodeIndexParity, SeverityError, fmt.Sprintf("%s document %q is not listed in index", dir, relPath), rel(spec.ExperienceDir, match), "Add the document to experience/index.json or remove it.")
		}
	}
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
	for _, el := range page.Elements {
		if !idPattern.MatchString(el.ID) || !rolePattern.MatchString(el.Role) {
			report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("element %q has invalid id or role", el.ID), loc, "Use kebab-case ids and ARIA or x- namespaced roles.")
		}
	}
	for _, claim := range page.Claims {
		checkClaimShape(report, loc, claim)
	}
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
		checkRegionShape(report, loc, region)
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

func checkRegionShape(report *Report, loc string, region ExperienceRegion) {
	if !idPattern.MatchString(region.ID) || len(strings.TrimSpace(region.Purpose)) < 10 {
		report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("region %q must include a kebab-case id and meaningful purpose", region.ID), loc, "Declare a stable region id and purpose.")
	}
	stateSet := stringSet(region.Lifecycle.States)
	if len(stateSet) != len(region.Lifecycle.States) || len(stateSet) == 0 {
		report.add(CodeSchemaInvalid, SeverityError, fmt.Sprintf("region %q lifecycle must declare unique states", region.ID), loc, "Declare one or more lifecycle states.")
	}
	for state := range stateSet {
		if !validRegionLifecycleState(state) {
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

func validRegionLifecycleState(state string) bool {
	switch state {
	case "loading", "ready", "empty", "partial", "error", "static":
		return true
	default:
		return false
	}
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
}

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
	if component.Component.Title == "" || len(component.Component.Purpose) < 15 || strings.TrimSpace(component.Component.ExamplesRef) == "" {
		report.add(CodeSchemaInvalid, SeverityError, "component identity must include title, purpose, and examplesRef", loc, "Fill component.title, component.purpose, and component.examplesRef.")
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
	exampleSet := componentExamples(filepath.Dir(path), component.Component.ExamplesRef)
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
		if len(exampleSet) == 0 {
			report.add(CodeRefUnresolved, SeverityError, fmt.Sprintf("component examplesRef %q has no readable examples", component.Component.ExamplesRef), loc, "Point examplesRef at a readable catalog examples.json file.")
			break
		}
		if !exampleSet[state.Example] {
			report.add(CodeRefUnresolved, SeverityError, fmt.Sprintf("component state %q references unknown example %q", state.ID, state.Example), loc, "Reference an example name declared in examples.json.")
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

func componentExamples(baseDir, examplesRef string) map[string]bool {
	ref := strings.TrimSpace(examplesRef)
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
	return out
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
	switch t {
	case "element-present", "element-absent", "single-dominant-action", "visible-without-scroll", "reading-order", "state-covered", "state-distinct", "keyboard-reachable", "accessible-name", "affordance-present", "no-document-horizontal-overflow", "viewport-fill", "chrome-pinned", "safe-area-tap-targets", "single-line-chrome", "tap-target-size", "custom":
		return true
	default:
		return false
	}
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
