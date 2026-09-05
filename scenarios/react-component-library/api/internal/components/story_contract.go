package components

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
)

// StoryContract is the versioned, declarative source of truth for one catalog
// asset's preview inputs, named baselines, fixtures, interactions, and checks.
// It intentionally contains no executable source or import references.
type StoryContract struct {
	SchemaVersion int               `json:"schemaVersion"`
	Schema        string            `json:"$schema,omitempty"`
	Kind          StoryKind         `json:"kind"`
	Title         string            `json:"title,omitempty"`
	Args          StoryArgsSchema   `json:"args"`
	Environment   StoryEnvironment  `json:"environment"`
	Composition   *StoryComposition `json:"composition,omitempty"`
	Stories       []StoryDefinition `json:"stories"`
}

type StoryKind string

const (
	StoryKindComponent StoryKind = "component"
	StoryKindHook      StoryKind = "hook"
)

type StoryArgsSchema struct {
	Fields []StoryField `json:"fields"`
}

type StoryField struct {
	Path        string            `json:"path"`
	Label       string            `json:"label,omitempty"`
	Kind        StoryFieldKind    `json:"kind"`
	Required    bool              `json:"required,omitempty"`
	Default     json.RawMessage   `json:"default,omitempty"`
	Options     []json.RawMessage `json:"options,omitempty"`
	Minimum     *float64          `json:"minimum,omitempty"`
	Maximum     *float64          `json:"maximum,omitempty"`
	MinLength   *int              `json:"minLength,omitempty"`
	MaxLength   *int              `json:"maxLength,omitempty"`
	Format      string            `json:"format,omitempty"`
	VisibleWhen *StoryCondition   `json:"visibleWhen,omitempty"`
}

type StoryFieldKind string

const (
	StoryFieldText       StoryFieldKind = "text"
	StoryFieldNumber     StoryFieldKind = "number"
	StoryFieldBoolean    StoryFieldKind = "boolean"
	StoryFieldEnum       StoryFieldKind = "enum"
	StoryFieldObject     StoryFieldKind = "object"
	StoryFieldArray      StoryFieldKind = "array"
	StoryFieldStructured StoryFieldKind = "structured"
)

type StoryCondition struct {
	Path   string          `json:"path"`
	Equals json.RawMessage `json:"equals"`
}

type StoryEnvironment struct {
	Fixtures []StoryFixture `json:"fixtures"`
}

type StoryFixture struct {
	Key     string   `json:"key"`
	Adapter string   `json:"adapter"`
	Options []string `json:"options"`
}

// StoryFrame names the catalog-owned composition context for a specimen. It
// is deliberately declarative: the preview service resolves the catalog
// asset and fixture, while the story only chooses the region to fill.
type StoryFrame struct {
	Asset string `json:"asset"`
	// Version pins the frame implementation used by a canonical story.
	Version    string `json:"version,omitempty"`
	Region     string `json:"region"`
	Capability string `json:"capability,omitempty"`
	Fixture    string `json:"fixture"`
}

// StoryHarnessRef selects a reusable Preview-only renderer. The renderer
// receives the subject component from the host; it must not import a
// component-specific production asset.
type StoryHarnessRef struct {
	Asset   string          `json:"asset"`
	Version string          `json:"version"`
	Export  string          `json:"export"`
	Config  json.RawMessage `json:"config,omitempty"`
}

// StorySpecimenRef identifies executable, version-local story content. The
// module is deliberately constrained to the version's story.tsx file; this
// keeps rich composition preview-only and prevents contracts from becoming an
// arbitrary import mechanism.
type StorySpecimenRef struct {
	Module string `json:"module"`
	Export string `json:"export"`
}

// StoryFixtureRef names a deterministic, catalog-owned fixture family. The
// exact version is required so a capture can always be reproduced.
type StoryFixtureRef struct {
	Asset   string `json:"asset"`
	Version string `json:"version"`
	State   string `json:"state,omitempty"`
}

// StoryComposition makes the four preview composition roles explicit. Each
// field is a reference, not executable content. A story-level composition
// replaces the corresponding file-level role.
type StoryComposition struct {
	Specimen *StorySpecimenRef `json:"specimen,omitempty"`
	Harness  *StoryHarnessRef  `json:"harness,omitempty"`
	Fixture  *StoryFixtureRef  `json:"fixture,omitempty"`
	Frame    *StoryFrame       `json:"frame,omitempty"`
}

// StoryEvidence describes the capture review set without moving executable
// content or assertions out of the story contract. It is intentionally small:
// the runner owns the concrete matrix and the author only names the intended
// review purpose and state coverage.
type StoryEvidence struct {
	ReviewSet string   `json:"reviewSet,omitempty"`
	States    []string `json:"states,omitempty"`
}

// storyFixtureAdapters is deliberately server-owned. A story may select a
// fixture id, but it cannot name an arbitrary provider/import to execute.
var storyFixtureAdapters = map[string]struct{}{
	"voice-input": {},
	"file-attach": {},
	"clipboard":   {},
	"network":     {},
}

type StoryDefinition struct {
	ID     string                       `json:"id"`
	Name   string                       `json:"name"`
	Role   string                       `json:"role,omitempty"`
	Axis   string                       `json:"axis,omitempty"`
	Covers map[string][]json.RawMessage `json:"covers,omitempty"`
	States []string                     `json:"states,omitempty"`
	// Description is optional specimen context shown by the catalog workbench.
	Description  string             `json:"description,omitempty"`
	Mode         StoryMode          `json:"mode,omitempty"`
	Composition  *StoryComposition  `json:"composition,omitempty"`
	Geometry     *StoryGeometry     `json:"geometry,omitempty"`
	Evidence     *StoryEvidence     `json:"evidence,omitempty"`
	Args         json.RawMessage    `json:"args"`
	Environment  map[string]string  `json:"environment,omitempty"`
	Interactions []StoryInteraction `json:"interactions,omitempty"`
	Expect       []StoryExpectation `json:"expect,omitempty"`
}

// UnmarshalJSON makes the common zero-argument story ergonomic while keeping
// the normalized contract explicit in the registry. Older authors often omit
// args for a story with no knobs; that is equivalent to args: {} and should
// not require a meaningless edit just to satisfy the validator.
func (s *StoryDefinition) UnmarshalJSON(data []byte) error {
	type storyDefinition StoryDefinition
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var decoded storyDefinition
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}
	if len(decoded.Args) == 0 || bytes.Equal(bytes.TrimSpace(decoded.Args), []byte("null")) {
		decoded.Args = json.RawMessage(`{}`)
	}
	*s = StoryDefinition(decoded)
	return nil
}

type StoryMode string

const (
	StoryModePinned StoryMode = "pinned"
	StoryModeLive   StoryMode = "live"
)

// StoryGeometry lets a story make a deliberate exception to the catalog
// archetype. It stays declarative so the preview host can apply the same
// geometry in every surface without importing component code.
type StoryGeometry struct {
	Archetype string `json:"archetype,omitempty"`
	Width     string `json:"width,omitempty"`
	Height    string `json:"height,omitempty"`
	MinHeight string `json:"minHeight,omitempty"`
}

type StoryInteraction struct {
	Kind   string          `json:"kind"`
	Target json.RawMessage `json:"target,omitempty"`
	Text   string          `json:"text,omitempty"`
}

type StoryExpectation struct {
	Kind       string `json:"kind"`
	Role       string `json:"role,omitempty"`
	Name       string `json:"name,omitempty"`
	Value      string `json:"value,omitempty"`
	Selector   string `json:"selector,omitempty"`
	Attribute  string `json:"attribute,omitempty"`
	MinWidth   *int   `json:"minWidth,omitempty"`
	MinHeight  *int   `json:"minHeight,omitempty"`
	MaxWidth   *int   `json:"maxWidth,omitempty"`
	MaxHeight  *int   `json:"maxHeight,omitempty"`
	NoOverflow bool   `json:"noOverflow,omitempty"`
}

type StoryDiagnostic struct {
	Pointer  string                  `json:"pointer"`
	Rule     string                  `json:"rule"`
	Detail   string                  `json:"detail"`
	Severity StoryDiagnosticSeverity `json:"severity"`
}

type StoryDiagnosticSeverity string

const (
	StoryDiagnosticError    StoryDiagnosticSeverity = "error"
	StoryDiagnosticWarning  StoryDiagnosticSeverity = "warning"
	StoryDiagnosticAdvisory StoryDiagnosticSeverity = "advisory"
)

var storyHTMLTagNames = map[string]struct{}{
	"div": {}, "p": {}, "span": {}, "button": {}, "nav": {}, "section": {}, "article": {},
	"ul": {}, "ol": {}, "li": {}, "h1": {}, "h2": {}, "h3": {}, "h4": {}, "h5": {}, "h6": {},
	"header": {}, "footer": {}, "main": {}, "aside": {}, "table": {}, "form": {}, "label": {},
	"img": {}, "a": {}, "strong": {}, "em": {}, "small": {}, "code": {}, "pre": {},
}

// CatalogFrameAsset is the small catalog projection needed to validate a
// story frame. Keeping this interface-shaped projection here lets the story
// grammar remain independent of the catalog loader and makes diagnostics
// deterministic in indexer and preview tests.
type CatalogFrameAsset struct {
	ID                 string
	Kind               string
	Targets            []string
	Regions            []string
	RegionCapabilities map[string]string
	Expects            []CatalogFramePort
	FixtureSatisfies   *CatalogFramePort
}

type CatalogFramePort struct {
	Capability    string
	TypeArguments []string
}

type CatalogFrameRegistry interface {
	LookupCatalogFrameAsset(id string) (CatalogFrameAsset, bool)
}

func validateStoryCompositionShape(pointer string, composition *StoryComposition) []StoryDiagnostic {
	if composition == nil {
		return nil
	}
	var diagnostics []StoryDiagnostic
	if composition.Specimen != nil {
		if composition.Specimen.Module != "./story.tsx" {
			diagnostics = append(diagnostics, storyDiagnostic(pointer+"/specimen/module", "specimen_module", "specimen module must be ./story.tsx"))
		}
		if !validHarnessExport(composition.Specimen.Export) {
			diagnostics = append(diagnostics, storyDiagnostic(pointer+"/specimen/export", "specimen_export", "specimen export must be a valid named JavaScript export identifier"))
		}
	}
	if composition.Harness != nil {
		diagnostics = append(diagnostics, validateStoryHarnessRef(pointer+"/harness", composition.Harness)...)
	}
	if composition.Specimen != nil && composition.Harness != nil {
		diagnostics = append(diagnostics, storyDiagnostic(pointer, "exclusive_renderer", "a composition must choose either a specimen or a composition harness"))
	}
	if composition.Fixture != nil {
		fixture := composition.Fixture
		if !validCatalogAssetID(fixture.Asset) || !strings.HasPrefix(fixture.Asset, "fixtures.") {
			diagnostics = append(diagnostics, storyDiagnostic(pointer+"/fixture/asset", "fixture_asset_id", "fixture asset must use the fixtures.* asset namespace"))
		}
		if !validAssetVersion(fixture.Version) {
			diagnostics = append(diagnostics, storyDiagnostic(pointer+"/fixture/version", "fixture_version", "fixture version must be a stable semantic version"))
		}
		if fixture.State != "" && !validStoryID(fixture.State) {
			diagnostics = append(diagnostics, storyDiagnostic(pointer+"/fixture/state", "fixture_state", "fixture state must be a stable lowercase slug"))
		}
	}
	if composition.Frame != nil {
		if composition.Frame.Version == "" {
			diagnostics = append(diagnostics, storyDiagnostic(pointer+"/frame/version", "frame_version_required", "schema-v4 composition frames require an exact semantic version"))
		}
		diagnostics = append(diagnostics, validateStoryFrameShape(pointer+"/frame", composition.Frame)...)
	}
	return diagnostics
}

// ValidateStoryFrames checks references that require the desired-state
// catalog. ParseStoryContract performs only shape and schema-version checks;
// this second pass is what produces named diagnostics for unknown assets,
// regions, and incompatible data-source fixtures.
func ValidateStoryFrames(contract *StoryContract, registry CatalogFrameRegistry) []StoryDiagnostic {
	if contract == nil || registry == nil {
		return nil
	}
	var diagnostics []StoryDiagnostic
	if contract.Composition != nil {
		diagnostics = append(diagnostics, validateStoryCompositionCatalog("/composition", contract.Composition, registry)...)
	}
	for index := range contract.Stories {
		if contract.Stories[index].Composition != nil {
			diagnostics = append(diagnostics, validateStoryCompositionCatalog(fmt.Sprintf("/stories/%d/composition", index), contract.Stories[index].Composition, registry)...)
			if contract.Stories[index].Composition.Frame != nil {
				diagnostics = append(diagnostics, validateStoryFrame(fmt.Sprintf("/stories/%d/composition/frame", index), contract.Stories[index].Composition.Frame, registry)...)
			}
		}
	}
	sort.Slice(diagnostics, func(i, j int) bool { return diagnostics[i].Pointer < diagnostics[j].Pointer })
	return diagnostics
}

func validateStoryCompositionCatalog(pointer string, composition *StoryComposition, registry CatalogFrameRegistry) []StoryDiagnostic {
	if composition == nil || composition.Fixture == nil || registry == nil {
		return nil
	}
	fixture, found := registry.LookupCatalogFrameAsset(composition.Fixture.Asset)
	if !found {
		return []StoryDiagnostic{storyDiagnostic(pointer+"/fixture/asset", "fixture_asset_exists", "fixture asset is not declared in the catalog")}
	}
	if fixture.Kind != "fixture" {
		return []StoryDiagnostic{storyDiagnostic(pointer+"/fixture/asset", "fixture_asset_kind", "composition fixture must reference an asset of kind fixture")}
	}
	return nil
}

func validateStoryFrameShape(pointer string, frame *StoryFrame) []StoryDiagnostic {
	if frame == nil {
		return nil
	}
	var diagnostics []StoryDiagnostic
	if !validCatalogAssetID(frame.Asset) {
		diagnostics = append(diagnostics, storyDiagnostic(pointer+"/asset", "frame_asset_id", "frame asset must be a catalog asset id"))
	}
	if frame.Version != "" && !validAssetVersion(frame.Version) {
		diagnostics = append(diagnostics, storyDiagnostic(pointer+"/version", "frame_version", "frame version must be a stable semantic version"))
	}
	if !validStoryID(frame.Region) {
		diagnostics = append(diagnostics, storyDiagnostic(pointer+"/region", "frame_region", "frame region must be a stable lowercase slug"))
	}
	if frame.Capability != "" && !validStoryID(frame.Capability) {
		diagnostics = append(diagnostics, storyDiagnostic(pointer+"/capability", "frame_capability", "frame capability must be a stable lowercase slug"))
	}
	if !validCatalogAssetID(frame.Fixture) {
		diagnostics = append(diagnostics, storyDiagnostic(pointer+"/fixture", "frame_fixture_id", "frame fixture must be a catalog asset id"))
	}
	return diagnostics
}

func validateStoryFrame(pointer string, frame *StoryFrame, registry CatalogFrameRegistry) []StoryDiagnostic {
	if frame == nil {
		return nil
	}
	var diagnostics []StoryDiagnostic
	asset, found := registry.LookupCatalogFrameAsset(frame.Asset)
	if !found {
		return []StoryDiagnostic{storyDiagnostic(pointer+"/asset", "frame_asset_exists", "frame asset is not declared in the catalog")}
	}
	if !containsFrameString(asset.Targets, "react-vite") {
		diagnostics = append(diagnostics, storyDiagnostic(pointer+"/asset", "frame_target", "frame asset does not target react-vite"))
	}
	if !containsFrameString(asset.Regions, frame.Region) {
		diagnostics = append(diagnostics, storyDiagnostic(pointer+"/region", "frame_region_exists", "frame region is not declared by the frame asset"))
	}
	if required := asset.RegionCapabilities[frame.Region]; required != "" && frame.Capability != required {
		diagnostics = append(diagnostics, storyDiagnostic(pointer+"/capability", "frame_region_capability", fmt.Sprintf("frame region requires subject capability %q", required)))
	}
	fixture, found := registry.LookupCatalogFrameAsset(frame.Fixture)
	if !found {
		diagnostics = append(diagnostics, storyDiagnostic(pointer+"/fixture", "frame_fixture_exists", "frame fixture is not declared in the catalog"))
		return diagnostics
	}
	if fixture.Kind != "fixture" {
		diagnostics = append(diagnostics, storyDiagnostic(pointer+"/fixture", "frame_fixture_kind", "frame fixture must reference an asset of kind fixture"))
		return diagnostics
	}
	for _, expect := range asset.Expects {
		if expect.Capability != "data-source" {
			continue
		}
		if fixture.FixtureSatisfies == nil || fixture.FixtureSatisfies.Capability != "data-source" || !compatibleTypeArguments(expect.TypeArguments, fixture.FixtureSatisfies.TypeArguments) {
			diagnostics = append(diagnostics, storyDiagnostic(pointer+"/fixture", "frame_fixture_data_source", "frame fixture does not satisfy the frame asset's data-source port"))
		}
	}
	return diagnostics
}

func compatibleTypeArguments(expected, actual []string) bool {
	if len(expected) != len(actual) {
		return false
	}
	for index := range expected {
		if strings.HasPrefix(expected[index], "T") {
			continue
		}
		if expected[index] != actual[index] {
			return false
		}
	}
	return true
}

func validCatalogAssetID(value string) bool {
	parts := strings.Split(strings.TrimSpace(value), ".")
	return len(parts) == 2 && validStoryID(parts[0]) && validStoryID(parts[1])
}

var assetVersionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)

func validAssetVersion(value string) bool {
	return assetVersionPattern.MatchString(strings.TrimSpace(value))
}

func containsFrameString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

// StoryCoverageGap is a release-readiness finding. Enum options are part of
// the public preview surface, so every declared option must be represented by
// at least one named story. Keeping this beside the story contract prevents
// the component-test provider and promotion paths from inventing different
// interpretations of coverage.
type StoryCoverageGap struct {
	Path  string
	Value string
}

func (g StoryCoverageGap) Error() string {
	return fmt.Sprintf("story coverage missing enum value %q for prop %q", g.Value, g.Path)
}

// StoryCoverageGaps returns one deterministic finding for every enum option
// absent from every story args object. Drafts may report these gaps; release
// callers must reject them.
func StoryCoverageGaps(contract *StoryContract) []StoryCoverageGap {
	if contract == nil {
		return nil
	}
	var gaps []StoryCoverageGap
	for _, field := range contract.Args.Fields {
		if field.Kind != StoryFieldEnum {
			continue
		}
		seen := map[string]struct{}{}
		for _, story := range contract.Stories {
			for path, values := range story.Covers {
				if path != field.Path {
					continue
				}
				for _, value := range values {
					var decoded any
					if json.Unmarshal(value, &decoded) == nil {
						if raw, err := json.Marshal(decoded); err == nil {
							seen[string(raw)] = struct{}{}
						}
					}
				}
			}
			var args map[string]any
			if json.Unmarshal(story.Args, &args) != nil {
				continue
			}
			value, ok := valueAtStoryPath(args, field.Path)
			if !ok {
				continue
			}
			raw, err := json.Marshal(value)
			if err == nil {
				seen[string(raw)] = struct{}{}
			}
		}
		for _, option := range field.Options {
			var value any
			if json.Unmarshal(option, &value) != nil {
				continue
			}
			raw, err := json.Marshal(value)
			if err != nil {
				continue
			}
			if _, ok := seen[string(raw)]; !ok {
				gaps = append(gaps, StoryCoverageGap{Path: field.Path, Value: string(raw)})
			}
		}
	}
	sort.Slice(gaps, func(i, j int) bool {
		if gaps[i].Path == gaps[j].Path {
			return gaps[i].Value < gaps[j].Value
		}
		return gaps[i].Path < gaps[j].Path
	})
	return gaps
}

func (d StoryDiagnostic) Error() string {
	severity := string(d.Severity)
	if severity == "" {
		severity = string(StoryDiagnosticError)
	}
	return fmt.Sprintf("%s: %s (%s, %s)", d.Pointer, d.Detail, d.Rule, severity)
}

// ParseStoryContract rejects unknown fields so a misspelling never becomes an
// unvalidated preview capability. Callers keep the diagnostics source-path
// aware when translating them into index findings.
func ParseStoryContract(raw []byte) (*StoryContract, []StoryDiagnostic) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var contract StoryContract
	if err := decoder.Decode(&contract); err != nil {
		return nil, []StoryDiagnostic{{Pointer: "/", Rule: "valid_json", Detail: err.Error()}}
	}
	if decoder.More() {
		return nil, []StoryDiagnostic{{Pointer: "/", Rule: "single_document", Detail: "only one JSON document is allowed"}}
	}
	diagnostics := ValidateStoryContract(&contract)
	sortStoryDiagnostics(diagnostics)
	return &contract, diagnostics
}

func StoryContractErrors(diagnostics []StoryDiagnostic) []StoryDiagnostic {
	errors := make([]StoryDiagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == "" || diagnostic.Severity == StoryDiagnosticError {
			errors = append(errors, diagnostic)
		}
	}
	return errors
}

func ValidateStoryContract(contract *StoryContract) []StoryDiagnostic {
	if contract == nil {
		return []StoryDiagnostic{{Pointer: "/", Rule: "required", Detail: "story contract is required"}}
	}
	var diagnostics []StoryDiagnostic
	if contract.SchemaVersion != 5 {
		diagnostics = append(diagnostics, storyDiagnostic("/schemaVersion", "supported_version", "schemaVersion must be 5 for published story contracts"))
	}
	if contract.Kind != StoryKindComponent && contract.Kind != StoryKindHook {
		diagnostics = append(diagnostics, storyDiagnostic("/kind", "asset_kind", "kind must be component or hook"))
	}
	fields := map[string]StoryField{}
	for index, field := range contract.Args.Fields {
		pointer := fmt.Sprintf("/args/fields/%d", index)
		path := strings.TrimSpace(field.Path)
		if !validStoryPath(path) {
			diagnostics = append(diagnostics, storyDiagnostic(pointer+"/path", "field_path", "path must contain dot-separated own-property segments"))
			continue
		}
		if _, exists := fields[path]; exists {
			diagnostics = append(diagnostics, storyDiagnostic(pointer+"/path", "unique", "field paths must be unique"))
			continue
		}
		fields[path] = field
		diagnostics = append(diagnostics, validateStoryField(pointer, field)...)
		if len(field.Default) > 0 {
			diagnostics = append(diagnostics, validateStoryGrammar(pointer+"/default", field.Default, false)...)
		}
	}
	fixtureOptions := map[string]map[string]struct{}{}
	for index, fixture := range contract.Environment.Fixtures {
		pointer := fmt.Sprintf("/environment/fixtures/%d", index)
		key := strings.TrimSpace(fixture.Key)
		if !validStoryPath(key) || strings.Contains(key, ".") {
			diagnostics = append(diagnostics, storyDiagnostic(pointer+"/key", "fixture_key", "fixture key must be one safe segment"))
			continue
		}
		if strings.TrimSpace(fixture.Adapter) == "" {
			diagnostics = append(diagnostics, storyDiagnostic(pointer+"/adapter", "required", "adapter is required"))
		} else if _, supported := storyFixtureAdapters[fixture.Adapter]; !supported {
			diagnostics = append(diagnostics, storyDiagnostic(pointer+"/adapter", "allowlisted_adapter", "fixture adapter is not supported"))
		}
		if _, exists := fixtureOptions[key]; exists {
			diagnostics = append(diagnostics, storyDiagnostic(pointer+"/key", "unique", "fixture keys must be unique"))
			continue
		}
		options := map[string]struct{}{}
		for optionIndex, option := range fixture.Options {
			option = strings.TrimSpace(option)
			if option == "" {
				diagnostics = append(diagnostics, storyDiagnostic(fmt.Sprintf("%s/options/%d", pointer, optionIndex), "required", "fixture option is required"))
				continue
			}
			if _, exists := options[option]; exists {
				diagnostics = append(diagnostics, storyDiagnostic(fmt.Sprintf("%s/options/%d", pointer, optionIndex), "unique", "fixture options must be unique"))
			}
			options[option] = struct{}{}
		}
		fixtureOptions[key] = options
	}
	if contract.Composition != nil {
		diagnostics = append(diagnostics, validateStoryCompositionShape("/composition", contract.Composition)...)
	}
	ids := map[string]struct{}{}
	for index, story := range contract.Stories {
		pointer := fmt.Sprintf("/stories/%d", index)
		if !validStoryID(story.ID) {
			diagnostics = append(diagnostics, storyDiagnostic(pointer+"/id", "story_id", "id must be a stable lowercase slug"))
		} else if _, exists := ids[story.ID]; exists {
			diagnostics = append(diagnostics, storyDiagnostic(pointer+"/id", "unique", "story ids must be unique"))
		} else {
			ids[story.ID] = struct{}{}
		}
		if strings.TrimSpace(story.Name) == "" {
			diagnostics = append(diagnostics, storyDiagnostic(pointer+"/name", "required", "name is required"))
		}
		if story.Role != "" && story.Role != "anatomy" && story.Role != "axis" && story.Role != "boundary" {
			diagnostics = append(diagnostics, storyDiagnostic(pointer+"/role", "story_role", "role must be anatomy, axis, or boundary"))
		}
		if story.Role == "axis" && !validStoryPath(story.Axis) {
			diagnostics = append(diagnostics, storyDiagnostic(pointer+"/axis", "axis_required", "axis stories require a valid prop path"))
		}
		if story.Mode != "" && story.Mode != StoryModePinned && story.Mode != StoryModeLive {
			diagnostics = append(diagnostics, storyDiagnostic(pointer+"/mode", "story_mode", "mode must be pinned or live"))
		}
		if story.Composition != nil {
			diagnostics = append(diagnostics, validateStoryCompositionShape(pointer+"/composition", story.Composition)...)
		}
		if story.Geometry != nil {
			diagnostics = append(diagnostics, validateStoryGeometry(pointer+"/geometry", story.Geometry)...)
		}
		if story.Composition != nil && story.Composition.Specimen != nil && story.Composition.Specimen.Export == "" {
			diagnostics = append(diagnostics, storyDiagnostic(pointer+"/composition/specimen/export", "required", "specimen export is required"))
		}
		diagnostics = append(diagnostics, validateStoryArgs(pointer+"/args", story.Args, fields)...)
		diagnostics = append(diagnostics, validateStoryGrammar(pointer+"/args", story.Args, true)...)
		if story.Composition == nil || story.Composition.Specimen == nil {
			for expectationIndex, expectation := range story.Expect {
				if expectation.Kind != "text" || strings.TrimSpace(expectation.Value) == "" {
					continue
				}
				if !containsTextValue(story.Args, expectation.Value) {
					diagnostics = append(diagnostics, storyDiagnosticSeverity(fmt.Sprintf("%s/expect/%d", pointer, expectationIndex), "expect_unreachable_text", fmt.Sprintf("text expectation %q does not occur in the story args; use a specimen when content is rendered by React composition", expectation.Value), StoryDiagnosticWarning))
				}
			}
		}
		for key, option := range story.Environment {
			options, exists := fixtureOptions[key]
			if !exists {
				diagnostics = append(diagnostics, storyDiagnostic(pointer+"/environment/"+key, "declared_fixture", "fixture key is not declared"))
				continue
			}
			if _, exists := options[option]; !exists {
				diagnostics = append(diagnostics, storyDiagnostic(pointer+"/environment/"+key, "declared_fixture_option", "fixture option is not declared"))
			}
		}
		for interactionIndex, interaction := range story.Interactions {
			diagnostics = append(diagnostics, validateStoryInteraction(fmt.Sprintf("%s/interactions/%d", pointer, interactionIndex), contract.Kind, interaction)...)
		}
		for expectationIndex, expectation := range story.Expect {
			if !allowedExpectation(expectation.Kind) {
				diagnostics = append(diagnostics, storyDiagnostic(fmt.Sprintf("%s/expect/%d/kind", pointer, expectationIndex), "allowlisted_expectation", "expectation kind is not supported"))
			}
		}
	}
	if len(contract.Stories) == 0 {
		diagnostics = append(diagnostics, storyDiagnostic("/stories", "required", "at least one named story is required"))
	}
	sortStoryDiagnostics(diagnostics)
	return diagnostics
}

func sortStoryDiagnostics(diagnostics []StoryDiagnostic) {
	sort.SliceStable(diagnostics, func(i, j int) bool {
		if diagnostics[i].Pointer != diagnostics[j].Pointer {
			return diagnostics[i].Pointer < diagnostics[j].Pointer
		}
		return storyDiagnosticSeverityRank(diagnostics[i].Severity) < storyDiagnosticSeverityRank(diagnostics[j].Severity)
	})
}

func storyDiagnosticSeverityRank(severity StoryDiagnosticSeverity) int {
	switch severity {
	case StoryDiagnosticError, "":
		return 0
	case StoryDiagnosticWarning:
		return 1
	case StoryDiagnosticAdvisory:
		return 2
	default:
		return 3
	}
}

func storyDiagnosticSeverity(pointer, rule, detail string, severity StoryDiagnosticSeverity) StoryDiagnostic {
	return StoryDiagnostic{Pointer: pointer, Rule: rule, Detail: detail, Severity: severity}
}

func validateStoryGrammar(pointer string, raw json.RawMessage, storyArgs bool) []StoryDiagnostic {
	if len(raw) == 0 {
		return nil
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil
	}
	diagnostics := make([]StoryDiagnostic, 0)
	var walk func(any, string)
	walk = func(current any, currentPointer string) {
		switch typed := current.(type) {
		case map[string]any:
			if text, ok := typed["$text"].(string); ok {
				diagnostics = append(diagnostics, storyDiagnostic(currentPointer, "raw_text_node", "$text nodes are not part of the story contract grammar; use a string value or story.tsx composition instead"))
				if _, isTag := storyHTMLTagNames[strings.ToLower(strings.TrimSpace(text))]; isTag {
					diagnostics = append(diagnostics, storyDiagnostic(currentPointer, "raw_node_tag_name", fmt.Sprintf("$text value %q is an HTML tag name; move the React composition into story.tsx instead", text)))
				}
			}
			for key, child := range typed {
				walk(child, currentPointer+"/"+escapeJSONPointer(key))
			}
		case []any:
			for index, child := range typed {
				walk(child, fmt.Sprintf("%s/%d", currentPointer, index))
			}
		}
	}
	walk(value, pointer)
	return diagnostics
}

func containsTextValue(raw json.RawMessage, wanted string) bool {
	if len(raw) == 0 {
		return false
	}
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return false
	}
	var found func(any) bool
	found = func(current any) bool {
		switch typed := current.(type) {
		case string:
			return typed == wanted
		case map[string]any:
			for _, child := range typed {
				if found(child) {
					return true
				}
			}
		case []any:
			for _, child := range typed {
				if found(child) {
					return true
				}
			}
		}
		return false
	}
	return found(value)
}

func escapeJSONPointer(value string) string {
	return strings.NewReplacer("~", "~0", "/", "~1").Replace(value)
}

func validateStoryHarnessRef(pointer string, harness *StoryHarnessRef) []StoryDiagnostic {
	if harness == nil {
		return nil
	}
	var diagnostics []StoryDiagnostic
	if !validCatalogAssetID(harness.Asset) || !strings.HasPrefix(harness.Asset, "preview.") {
		diagnostics = append(diagnostics, storyDiagnostic(pointer+"/asset", "composition_harness_asset", "composition harness asset must use the preview.* asset namespace"))
	}
	if !validAssetVersion(harness.Version) {
		diagnostics = append(diagnostics, storyDiagnostic(pointer+"/version", "composition_harness_version", "composition harness version must be a stable semantic version"))
	}
	if !validHarnessExport(harness.Export) {
		diagnostics = append(diagnostics, storyDiagnostic(pointer+"/export", "composition_harness_export", "composition harness export must be a valid named JavaScript export identifier"))
	}
	if len(harness.Config) > 0 && !json.Valid(harness.Config) {
		diagnostics = append(diagnostics, storyDiagnostic(pointer+"/config", "composition_harness_config", "composition harness config must be valid JSON"))
	}
	return diagnostics
}

func validateStoryGeometry(pointer string, geometry *StoryGeometry) []StoryDiagnostic {
	if geometry == nil {
		return nil
	}
	if geometry.Archetype != "" {
		switch geometry.Archetype {
		case "primitive", "pattern", "page", "shell", "frame", "overlay":
		default:
			return []StoryDiagnostic{storyDiagnostic(pointer+"/archetype", "geometry_archetype", "geometry archetype is not supported")}
		}
	}
	return nil
}

func validateStoryField(pointer string, field StoryField) []StoryDiagnostic {
	var diagnostics []StoryDiagnostic
	switch field.Kind {
	case StoryFieldText, StoryFieldNumber, StoryFieldBoolean, StoryFieldEnum, StoryFieldObject, StoryFieldArray, StoryFieldStructured:
	default:
		diagnostics = append(diagnostics, storyDiagnostic(pointer+"/kind", "field_kind", "field kind is not supported"))
	}
	if field.Kind == StoryFieldEnum && len(field.Options) == 0 {
		diagnostics = append(diagnostics, storyDiagnostic(pointer+"/options", "required", "enum fields require options"))
	}
	if field.Minimum != nil && field.Maximum != nil && *field.Minimum > *field.Maximum {
		diagnostics = append(diagnostics, storyDiagnostic(pointer, "range", "minimum must not exceed maximum"))
	}
	if field.MinLength != nil && field.MaxLength != nil && *field.MinLength > *field.MaxLength {
		diagnostics = append(diagnostics, storyDiagnostic(pointer, "length", "minLength must not exceed maxLength"))
	}
	if field.Format != "" && field.Format != "plain-text" && field.Format != "identifier" && field.Format != "url" && field.Format != "renderable-text" {
		diagnostics = append(diagnostics, storyDiagnostic(pointer+"/format", "format", "format is not supported"))
	}
	if len(field.Default) > 0 {
		diagnostics = append(diagnostics, validateStoryValue(pointer+"/default", field, field.Default)...)
	}
	for index, option := range field.Options {
		if !isJSONScalar(option) {
			diagnostics = append(diagnostics, storyDiagnostic(fmt.Sprintf("%s/options/%d", pointer, index), "scalar", "enum options must be JSON scalars"))
		}
	}
	return diagnostics
}

func validateStoryArgs(pointer string, raw json.RawMessage, fields map[string]StoryField) []StoryDiagnostic {
	var args map[string]any
	if len(raw) == 0 || json.Unmarshal(raw, &args) != nil {
		return []StoryDiagnostic{storyDiagnostic(pointer, "object", "args must be a JSON object")}
	}
	var diagnostics []StoryDiagnostic
	for path, field := range fields {
		value, exists := valueAtStoryPath(args, path)
		if !exists {
			if field.Required && len(field.Default) == 0 {
				diagnostics = append(diagnostics, storyDiagnostic(pointer+"/"+path, "required", "required field has no story value or default"))
			}
			continue
		}
		rawValue, _ := json.Marshal(value)
		diagnostics = append(diagnostics, validateStoryValue(pointer+"/"+path, field, rawValue)...)
	}
	return diagnostics
}

func validateStoryValue(pointer string, field StoryField, raw json.RawMessage) []StoryDiagnostic {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return []StoryDiagnostic{storyDiagnostic(pointer, "json", "value must be JSON")}
	}
	if tag := removedStructuredTag(value); tag != "" {
		replacement := "a named export in ./story.tsx referenced by composition.specimen"
		if tag == "$rowKey" {
			replacement = "a getRowKey function in the story.tsx specimen"
		} else if tag == "$columns" {
			replacement = "a columns array in the story.tsx specimen"
		} else if tag == "$filters" {
			replacement = "a filters array in the story.tsx specimen"
		}
		return []StoryDiagnostic{storyDiagnostic(pointer, "removed_structured_tag", fmt.Sprintf("structured value %s was removed; use %s", tag, replacement))}
	}
	if !safeStoryValue(value) {
		return []StoryDiagnostic{storyDiagnostic(pointer, "safe_value", "value contains unsupported structured data")}
	}
	switch field.Kind {
	case StoryFieldText:
		if _, ok := value.(string); !ok && field.Format != "renderable-text" {
			return []StoryDiagnostic{storyDiagnostic(pointer, "text", "value must be a string")}
		}
	case StoryFieldNumber:
		number, ok := value.(float64)
		if !ok || math.IsNaN(number) || math.IsInf(number, 0) {
			return []StoryDiagnostic{storyDiagnostic(pointer, "number", "value must be a finite number")}
		}
		if field.Minimum != nil && number < *field.Minimum || field.Maximum != nil && number > *field.Maximum {
			return []StoryDiagnostic{storyDiagnostic(pointer, "range", "value is outside the declared range")}
		}
	case StoryFieldBoolean:
		if _, ok := value.(bool); !ok {
			return []StoryDiagnostic{storyDiagnostic(pointer, "boolean", "value must be a boolean")}
		}
	case StoryFieldEnum:
		encoded, _ := json.Marshal(value)
		for _, option := range field.Options {
			if bytes.Equal(encoded, option) {
				return nil
			}
		}
		return []StoryDiagnostic{storyDiagnostic(pointer, "enum", "value is not one of the declared options")}
	}
	return nil
}

var removedStructuredTagNames = []string{"$node", "$icon", "$handler", "$rowKey", "$columns", "$filters"}

func removedStructuredTag(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		for _, tag := range removedStructuredTagNames {
			if _, exists := typed[tag]; exists {
				return tag
			}
		}
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if tag := removedStructuredTag(typed[key]); tag != "" {
				return tag
			}
		}
	case []any:
		for _, item := range typed {
			if tag := removedStructuredTag(item); tag != "" {
				return tag
			}
		}
	}
	return ""
}

var (
	storyPathSegment   = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)
	storyID            = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	storyHarnessExport = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)
)

var javascriptReservedWords = map[string]struct{}{
	"await": {}, "break": {}, "case": {}, "catch": {}, "class": {}, "const": {}, "continue": {}, "debugger": {}, "default": {}, "delete": {}, "do": {}, "else": {}, "enum": {}, "export": {}, "extends": {}, "false": {}, "finally": {}, "for": {}, "function": {}, "if": {}, "implements": {}, "import": {}, "in": {}, "instanceof": {}, "interface": {}, "let": {}, "new": {}, "null": {}, "package": {}, "private": {}, "protected": {}, "public": {}, "return": {}, "super": {}, "switch": {}, "static": {}, "this": {}, "throw": {}, "true": {}, "try": {}, "typeof": {}, "var": {}, "void": {}, "while": {}, "with": {}, "yield": {},
}

func validStoryPath(path string) bool {
	if path == "" || strings.HasPrefix(path, ".") || strings.HasSuffix(path, ".") {
		return false
	}
	for _, segment := range strings.Split(path, ".") {
		if !storyPathSegment.MatchString(segment) || segment == "__proto__" || segment == "prototype" || segment == "constructor" {
			return false
		}
	}
	return true
}

func validStoryID(value string) bool { return storyID.MatchString(strings.TrimSpace(value)) }
func validHarnessExport(value string) bool {
	value = strings.TrimSpace(value)
	if !storyHarnessExport.MatchString(value) {
		return false
	}
	_, reserved := javascriptReservedWords[value]
	return !reserved
}

func storyDiagnostic(pointer, rule, detail string) StoryDiagnostic {
	return StoryDiagnostic{Pointer: pointer, Rule: rule, Detail: detail, Severity: StoryDiagnosticError}
}

func allowedInteraction(kind string) bool {
	return map[string]bool{"click": true, "type": true, "key": true, "focus": true, "blur": true, "waitFor": true, "settle": true}[kind]
}

func validateStoryInteraction(pointer string, assetKind StoryKind, interaction StoryInteraction) []StoryDiagnostic {
	if !allowedInteraction(interaction.Kind) {
		return []StoryDiagnostic{storyDiagnostic(pointer+"/kind", "allowlisted_interaction", "interaction kind is not supported")}
	}
	if interaction.Kind == "settle" {
		if len(interaction.Target) > 0 || interaction.Text != "" {
			return []StoryDiagnostic{storyDiagnostic(pointer, "settle_shape", "settle accepts no target or text")}
		}
		return nil
	}
	if interaction.Kind == "waitFor" {
		if len(interaction.Target) > 0 {
			return []StoryDiagnostic{storyDiagnostic(pointer+"/target", "wait_shape", "waitFor accepts no target")}
		}
		return nil
	}
	requiresTarget := interaction.Kind == "click" || interaction.Kind == "type" || interaction.Kind == "focus" || interaction.Kind == "blur"
	if requiresTarget && assetKind != StoryKindHook && len(interaction.Target) == 0 {
		return []StoryDiagnostic{storyDiagnostic(pointer+"/target", "required", "component interaction requires a declared target")}
	}
	if len(interaction.Target) > 0 {
		var target struct {
			Selector string `json:"selector"`
			Role     string `json:"role"`
			Name     string `json:"name"`
		}
		decoder := json.NewDecoder(bytes.NewReader(interaction.Target))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&target); err != nil || (strings.TrimSpace(target.Selector) == "" && strings.TrimSpace(target.Role) == "") {
			return []StoryDiagnostic{storyDiagnostic(pointer+"/target", "safe_locator", "target must be a selector or role/name locator")}
		}
	}
	if interaction.Kind == "type" && interaction.Text == "" {
		return []StoryDiagnostic{storyDiagnostic(pointer+"/text", "required", "type interaction requires text")}
	}
	if interaction.Kind == "key" && interaction.Text == "" {
		return []StoryDiagnostic{storyDiagnostic(pointer+"/text", "required", "key interaction requires a key")}
	}
	return nil
}

func allowedExpectation(kind string) bool {
	return map[string]bool{"role": true, "text": true, "attribute": true, "visible": true, "notVisible": true, "layout": true, "count": true}[kind]
}

func isJSONScalar(raw json.RawMessage) bool {
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return false
	}
	switch value.(type) {
	case nil, bool, float64, string:
		return true
	default:
		return false
	}
}

func valueAtStoryPath(root map[string]any, path string) (any, bool) {
	var current any = root
	for _, segment := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[segment]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func safeStoryValue(value any) bool {
	switch typed := value.(type) {
	case nil, bool, float64, string:
		return true
	case []any:
		for _, item := range typed {
			if !safeStoryValue(item) {
				return false
			}
		}
		return true
	case map[string]any:
		for key, item := range typed {
			if key == "__proto__" || key == "prototype" || key == "constructor" {
				return false
			}
			if strings.HasPrefix(key, "$") && key != "$text" {
				return false
			}
			if !safeStoryValue(item) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
