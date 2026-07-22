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
	Kind      string `json:"kind"`
	Role      string `json:"role,omitempty"`
	Name      string `json:"name,omitempty"`
	Value     string `json:"value,omitempty"`
	Selector  string `json:"selector,omitempty"`
	Attribute string `json:"attribute,omitempty"`
}

type StoryDiagnostic struct {
	Pointer string
	Rule    string
	Detail  string
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
	if contract.SchemaVersion != 1 && contract.SchemaVersion != 2 {
		diagnostics = append(diagnostics, storyDiagnostic("/schemaVersion", "supported_version", "schemaVersion must be 1 or 2"))
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
		if contract.SchemaVersion == 1 && (story.Harness != "" || story.Description != "") {
			diagnostics = append(diagnostics, storyDiagnostic(pointer, "schema_version", "harness and description require schemaVersion 2"))
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

var storyPathSegment = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)
var storyID = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
var storyHarnessExport = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)

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
	return map[string]bool{"role": true, "text": true, "attribute": true, "visible": true, "notVisible": true}[kind]
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
