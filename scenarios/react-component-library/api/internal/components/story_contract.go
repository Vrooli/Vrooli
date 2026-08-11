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
	Kind          StoryKind         `json:"kind"`
	Title         string            `json:"title,omitempty"`
	Args          StoryArgsSchema   `json:"args"`
	Environment   StoryEnvironment  `json:"environment"`
	Frame         *StoryFrame       `json:"frame,omitempty"`
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
	Asset   string `json:"asset"`
	Region  string `json:"region"`
	Fixture string `json:"fixture"`
}

// storyFixtureAdapters is deliberately server-owned. A story may select a
// fixture id, but it cannot name an arbitrary provider/import to execute.
var storyFixtureAdapters = map[string]struct{}{
	"voice-input": {},
}

type StoryDefinition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Description is optional specimen context shown by the catalog workbench.
	Description string `json:"description,omitempty"`
	// Harness selects a named export from the version-local story.tsx file.
	// It is available only in schemaVersion 2 and later.
	Harness      string             `json:"harness,omitempty"`
	Frame        *StoryFrame        `json:"frame,omitempty"`
	Args         json.RawMessage    `json:"args"`
	Environment  map[string]string  `json:"environment,omitempty"`
	Interactions []StoryInteraction `json:"interactions,omitempty"`
	Expect       []StoryExpectation `json:"expect,omitempty"`
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
	Pointer string
	Rule    string
	Detail  string
}

// CatalogFrameAsset is the small catalog projection needed to validate a
// story frame. Keeping this interface-shaped projection here lets the story
// grammar remain independent of the catalog loader and makes diagnostics
// deterministic in indexer and preview tests.
type CatalogFrameAsset struct {
	ID               string
	Kind             string
	Targets          []string
	Regions          []string
	Expects          []CatalogFramePort
	FixtureSatisfies *CatalogFramePort
}

type CatalogFramePort struct {
	Capability    string
	TypeArguments []string
}

type CatalogFrameRegistry interface {
	LookupCatalogFrameAsset(id string) (CatalogFrameAsset, bool)
}

// EffectiveStoryFrame applies the story-level override over the file-level
// declaration. Returning nil is intentional: unframed stories retain the
// existing direct-render path.
func EffectiveStoryFrame(contract *StoryContract, story *StoryDefinition) *StoryFrame {
	if story != nil && story.Frame != nil {
		return story.Frame
	}
	if contract == nil {
		return nil
	}
	return contract.Frame
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
	if contract.Frame != nil {
		diagnostics = append(diagnostics, validateStoryFrame("/frame", contract.Frame, registry)...)
	}
	for index := range contract.Stories {
		if contract.Stories[index].Frame != nil {
			diagnostics = append(diagnostics, validateStoryFrame(fmt.Sprintf("/stories/%d/frame", index), contract.Stories[index].Frame, registry)...)
		}
	}
	sort.Slice(diagnostics, func(i, j int) bool { return diagnostics[i].Pointer < diagnostics[j].Pointer })
	return diagnostics
}

func validateStoryFrameShape(pointer string, frame *StoryFrame) []StoryDiagnostic {
	if frame == nil {
		return nil
	}
	var diagnostics []StoryDiagnostic
	if !validCatalogAssetID(frame.Asset) {
		diagnostics = append(diagnostics, storyDiagnostic(pointer+"/asset", "frame_asset_id", "frame asset must be a catalog asset id"))
	}
	if !validStoryID(frame.Region) {
		diagnostics = append(diagnostics, storyDiagnostic(pointer+"/region", "frame_region", "frame region must be a stable lowercase slug"))
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
	return fmt.Sprintf("%s: %s (%s)", d.Pointer, d.Detail, d.Rule)
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
	return &contract, ValidateStoryContract(&contract)
}

func ValidateStoryContract(contract *StoryContract) []StoryDiagnostic {
	if contract == nil {
		return []StoryDiagnostic{{Pointer: "/", Rule: "required", Detail: "story contract is required"}}
	}
	var diagnostics []StoryDiagnostic
	if contract.SchemaVersion != 1 && contract.SchemaVersion != 2 && contract.SchemaVersion != 3 {
		diagnostics = append(diagnostics, storyDiagnostic("/schemaVersion", "supported_version", "schemaVersion must be 1, 2, or 3"))
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
	if contract.SchemaVersion < 3 && contract.Frame != nil {
		diagnostics = append(diagnostics, storyDiagnostic("/frame", "schema_version", "frame requires schemaVersion 3"))
	}
	if contract.Frame != nil {
		diagnostics = append(diagnostics, validateStoryFrameShape("/frame", contract.Frame)...)
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
		if contract.SchemaVersion == 1 && (story.Harness != "" || story.Description != "" || story.Frame != nil) {
			diagnostics = append(diagnostics, storyDiagnostic(pointer, "schema_version", "harness and description require schemaVersion 2"))
		}
		if contract.SchemaVersion < 3 && story.Frame != nil {
			diagnostics = append(diagnostics, storyDiagnostic(pointer+"/frame", "schema_version", "frame requires schemaVersion 3"))
		}
		if story.Frame != nil {
			diagnostics = append(diagnostics, validateStoryFrameShape(pointer+"/frame", story.Frame)...)
		}
		if story.Harness != "" && !validHarnessExport(story.Harness) {
			diagnostics = append(diagnostics, storyDiagnostic(pointer+"/harness", "javascript_identifier", "harness must be a valid named JavaScript export identifier"))
		}
		diagnostics = append(diagnostics, validateStoryArgs(pointer+"/args", story.Args, fields)...)
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
	sort.Slice(diagnostics, func(i, j int) bool { return diagnostics[i].Pointer < diagnostics[j].Pointer })
	return diagnostics
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
	return StoryDiagnostic{Pointer: pointer, Rule: rule, Detail: detail}
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
	return map[string]bool{"role": true, "text": true, "attribute": true, "visible": true, "notVisible": true, "layout": true}[kind]
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
			if strings.HasPrefix(key, "$") && key != "$text" && key != "$node" && key != "$icon" && key != "$handler" && key != "$rowKey" && key != "$columns" && key != "$filters" {
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
