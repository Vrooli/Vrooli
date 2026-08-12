// Package spec owns the scenario-experience-spec/v1 parser surface.
package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	CodeRegistryInvalid     = "experience.registry_invalid"
	CodeSchemaInvalid       = "experience.schema_invalid"
	CodeIndexParity         = "experience.index_parity"
	CodeRefUnresolved       = "experience.ref_unresolved"
	CodePRDRefUnmatched     = "experience.prd_ref_unmatched"
	CodeBindingOrphan       = "experience.binding_orphan"
	CodeTierViolation       = "experience.tier_violation"
	CodeVacuousContract     = "experience.vacuous_contract"
	CodeRouteUnspecced      = "experience.route_unspecced"
	CodeStateMissing        = "experience.state_missing"
	CodeBindingUnresolved   = "experience.binding_unresolved"
	CodeBindingsUnjoined    = "experience.capture_bindings_unjoined"
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
	CodeRegistryInvalid,
	CodeSchemaInvalid,
	CodeIndexParity,
	CodeRefUnresolved,
	CodePRDRefUnmatched,
	CodeBindingOrphan,
	CodeTierViolation,
	CodeVacuousContract,
	CodeRouteUnspecced,
	CodeStateMissing,
	CodeBindingUnresolved,
	CodeBindingsUnjoined,
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
	StoryRef    string               `json:"storyRef"`
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
	checkPortableLibraryContracts(&report, scenarioDir)
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
// Components do not participate in journeys; catalog-specimen anchored states are their
// highest parser-era depth.
func ComputeComponentDepth(component ComponentDocument) int {
	if component.Component.ID == "" || (component.Component.ExamplesRef == "" && component.Component.StoryRef == "") {
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
