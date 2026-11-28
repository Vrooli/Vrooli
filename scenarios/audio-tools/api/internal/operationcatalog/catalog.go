// Package operationcatalog owns the declarative, user-facing operation surface
// of audio-tools. One record per operation declares its identity (family,
// engine, mechanism, transport), typed ports, ordered parameters, CLI binding,
// and UI presentation. The embedded operations.seed.json is the single source
// of truth that the generated UI specs and the CLI/UI coverage tests read.
//
// Only the eight audio transforms carry a full parameter/CLI/presentation
// specification in this slice. Speech, synthesis, and summarization operations
// declare identity + ports + their existing CLI ownership so coverage gates can
// prove the surface is complete.
package operationcatalog

//go:generate go run ../../cmd/gen-operations

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
)

//go:embed operations.seed.json
var seedJSON []byte

// Catalog is the decoded operations.seed.json document.
type Catalog struct {
	Version    int         `json:"version"`
	Operations []Operation `json:"operations"`
}

// Operation is one user-facing operation.
type Operation struct {
	ID           string       `json:"id"`
	Summary      string       `json:"summary"`
	Family       string       `json:"family"`
	Engine       string       `json:"engine"`
	Mechanism    string       `json:"mechanism"`
	Transport    string       `json:"transport"`
	Ports        Ports        `json:"ports"`
	Params       []Param      `json:"params,omitempty"`
	CLI          CLISpec      `json:"cli,omitempty"`
	Presentation Presentation `json:"presentation,omitempty"`
	Requires     []string     `json:"requires,omitempty"`
	Consent      string       `json:"consent,omitempty"`
}

// Ports describes typed inputs and outputs without binding them to a transport.
type Ports struct {
	Inputs  []Port `json:"inputs"`
	Outputs []Port `json:"outputs"`
}

// Port is one typed input or output.
type Port struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// Param is one ordered operation parameter. The control decides which UI
// primitive renders the field; Type names the wire type.
type Param struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Control  string   `json:"control"`
	Required bool     `json:"required,omitempty"`
	Default  string   `json:"default,omitempty"`
	Min      *float64 `json:"min,omitempty"`
	Max      *float64 `json:"max,omitempty"`
	Step     *float64 `json:"step,omitempty"`
	Options  []Option `json:"options,omitempty"`
	Unit     string   `json:"unit,omitempty"`
	Advanced bool     `json:"advanced,omitempty"`
}

// Option is one select choice. LabelKey is an i18n key path.
type Option struct {
	Value    string `json:"value"`
	LabelKey string `json:"labelKey"`
}

// CLISpec is the CLI surface for an operation.
type CLISpec struct {
	Group string    `json:"group,omitempty"`
	Verb  string    `json:"verb,omitempty"`
	Flags []CLIFlag `json:"flags,omitempty"`
	// Service and Method mirror the manifest connect-rpc binding.
	Service string `json:"service,omitempty"`
	Method  string `json:"method,omitempty"`
}

// CLIFlag is one manifest flag the operation's command must declare.
type CLIFlag struct {
	Name      string   `json:"name"`
	Field     string   `json:"field,omitempty"`
	Required  bool     `json:"required,omitempty"`
	LocalOnly bool     `json:"localOnly,omitempty"`
	Repeated  bool     `json:"repeated,omitempty"`
	Default   string   `json:"default,omitempty"`
	Values    []string `json:"values,omitempty"`
}

// Presentation locates the operation in the UI.
type Presentation struct {
	Panel    string `json:"panel,omitempty"`
	Order    int    `json:"order,omitempty"`
	LabelKey string `json:"labelKey,omitempty"`
	DescKey  string `json:"descKey,omitempty"`
}

// Allowed vocabularies. Validation is intentionally closed: adding a family,
// engine, mechanism, or transport is a deliberate catalog change.
var (
	validFamilies   = map[string]struct{}{"transform": {}, "stt": {}, "tts": {}, "summarize": {}}
	validEngines    = map[string]struct{}{"audio": {}, "stt": {}, "tts": {}, "summarize": {}}
	validMechanisms = map[string]struct{}{"compute": {}, "infer": {}, "fetch": {}}
	validTransports = map[string]struct{}{"unary": {}, "server_stream": {}, "bidi": {}}
	validControls   = map[string]struct{}{"number": {}, "text": {}, "select": {}, "file": {}, "files": {}, "checkbox": {}}
	validConsents   = map[string]struct{}{"": {}, "none": {}, "policy": {}, "explicit": {}}
	// familyEngines pins the engineering owner for each product family.
	familyEngines = map[string]string{
		"transform": "audio",
		"stt":       "stt",
		"tts":       "tts",
		"summarize": "summarize",
	}
)

// Default loads and validates the embedded catalog.
func Default() (Catalog, error) {
	return Load(seedJSON)
}

// Load decodes and validates a catalog from raw JSON.
func Load(raw []byte) (Catalog, error) {
	var c Catalog
	if err := json.Unmarshal(raw, &c); err != nil {
		return Catalog{}, fmt.Errorf("operationcatalog: decode: %w", err)
	}
	if err := c.Validate(); err != nil {
		return Catalog{}, err
	}
	return c, nil
}

// MustDefault loads the embedded catalog or panics. Intended for generators and
// tests, never for production request paths.
func MustDefault() Catalog {
	c, err := Default()
	if err != nil {
		panic(err)
	}
	return c
}

// Transform returns the operations in the transform family, ordered by
// presentation order.
func (c Catalog) Transform() []Operation {
	return c.ByFamily("transform")
}

// ByFamily returns operations in one family, ordered by presentation order then
// catalog order.
func (c Catalog) ByFamily(family string) []Operation {
	index := make(map[string]int, len(c.Operations))
	for i, op := range c.Operations {
		index[op.ID] = i
	}
	out := make([]Operation, 0, len(c.Operations))
	for _, op := range c.Operations {
		if op.Family == family {
			out = append(out, op)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		oi, oj := out[i].Presentation.Order, out[j].Presentation.Order
		if oi != oj {
			return oi < oj
		}
		return index[out[i].ID] < index[out[j].ID]
	})
	return out
}

// Lookup returns the operation with id, or false.
func (c Catalog) Lookup(id string) (Operation, bool) {
	for _, op := range c.Operations {
		if op.ID == id {
			return op, true
		}
	}
	return Operation{}, false
}

// Validate enforces every catalog invariant.
func (c Catalog) Validate() error {
	if c.Version <= 0 {
		return fmt.Errorf("operationcatalog: version must be positive")
	}
	if len(c.Operations) == 0 {
		return fmt.Errorf("operationcatalog: at least one operation is required")
	}
	seenIDs := make(map[string]struct{}, len(c.Operations))
	seenOrder := make(map[int]string, len(c.Operations))
	for i, op := range c.Operations {
		where := fmt.Sprintf("operation[%d] %q", i, op.ID)
		if err := validateOperation(op); err != nil {
			return fmt.Errorf("operationcatalog: %s: %w", where, err)
		}
		if _, dup := seenIDs[op.ID]; dup {
			return fmt.Errorf("operationcatalog: %s: duplicate id", where)
		}
		seenIDs[op.ID] = struct{}{}
		if op.Presentation.Order > 0 {
			if prev, dup := seenOrder[op.Presentation.Order]; dup {
				return fmt.Errorf("operationcatalog: %s: presentation.order %d already used by %q", where, op.Presentation.Order, prev)
			}
			seenOrder[op.Presentation.Order] = op.ID
		}
	}
	return nil
}

func validateOperation(op Operation) error {
	if op.ID == "" {
		return fmt.Errorf("id is required")
	}
	if !isSlug(op.ID) {
		return fmt.Errorf("id %q must be lower_snake_case", op.ID)
	}
	if op.Summary == "" {
		return fmt.Errorf("summary is required")
	}
	if _, ok := validFamilies[op.Family]; !ok {
		return fmt.Errorf("family %q is not one of transform|stt|tts|summarize", op.Family)
	}
	if _, ok := validEngines[op.Engine]; !ok {
		return fmt.Errorf("engine %q is not one of audio|stt|tts|summarize", op.Engine)
	}
	if want := familyEngines[op.Family]; want != "" && op.Engine != want {
		return fmt.Errorf("engine %q does not match family %q (want %q)", op.Engine, op.Family, want)
	}
	if _, ok := validMechanisms[op.Mechanism]; !ok {
		return fmt.Errorf("mechanism %q is not one of compute|infer|fetch", op.Mechanism)
	}
	if _, ok := validTransports[op.Transport]; !ok {
		return fmt.Errorf("transport %q is not one of unary|server_stream|bidi", op.Transport)
	}
	if _, ok := validConsents[op.Consent]; !ok {
		return fmt.Errorf("consent %q is not one of none|policy|explicit", op.Consent)
	}
	for _, p := range op.Ports.Inputs {
		if err := validatePort(p); err != nil {
			return fmt.Errorf("input port: %w", err)
		}
	}
	for _, p := range op.Ports.Outputs {
		if err := validatePort(p); err != nil {
			return fmt.Errorf("output port: %w", err)
		}
	}
	if len(op.Params) == 0 && op.Family == "transform" {
		return fmt.Errorf("transform operations must declare parameters")
	}
	if err := validateParams(op); err != nil {
		return err
	}
	if op.Family == "transform" {
		if op.CLI.Group == "" || op.CLI.Verb == "" || op.CLI.Service == "" || op.CLI.Method == "" {
			return fmt.Errorf("transform operations require a complete CLI spec (group, verb, service, method)")
		}
		if op.Presentation.Panel == "" || op.Presentation.Order <= 0 || op.Presentation.LabelKey == "" || op.Presentation.DescKey == "" {
			return fmt.Errorf("transform operations require a complete presentation spec (panel, order, labelKey, descKey)")
		}
	}
	if op.CLI.Group != "" || op.CLI.Verb != "" || op.CLI.Method != "" {
		if err := validateCLI(op); err != nil {
			return err
		}
	}
	return nil
}

func validatePort(p Port) error {
	if p.Name == "" {
		return fmt.Errorf("name is required")
	}
	if p.Type == "" {
		return fmt.Errorf("port %q type is required", p.Name)
	}
	return nil
}

func validateParams(op Operation) error {
	seen := make(map[string]struct{}, len(op.Params))
	requiredFile := false
	for _, p := range op.Params {
		if p.Name == "" {
			return fmt.Errorf("param name is required")
		}
		if _, dup := seen[p.Name]; dup {
			return fmt.Errorf("param %q is declared more than once", p.Name)
		}
		seen[p.Name] = struct{}{}
		if p.Type == "" {
			return fmt.Errorf("param %q type is required", p.Name)
		}
		if _, ok := validControls[p.Control]; !ok {
			return fmt.Errorf("param %q control %q is not one of number|text|select|file|files|checkbox", p.Name, p.Control)
		}
		switch p.Control {
		case "number":
			if p.Min == nil || p.Max == nil {
				return fmt.Errorf("param %q number control requires min and max", p.Name)
			}
			if *p.Min >= *p.Max {
				return fmt.Errorf("param %q min (%g) must be less than max (%g)", p.Name, *p.Min, *p.Max)
			}
			if p.Step != nil && *p.Step <= 0 {
				return fmt.Errorf("param %q step must be positive", p.Name)
			}
			if p.Default != "" {
				var v float64
				if _, err := fmt.Sscanf(p.Default, "%f", &v); err != nil {
					return fmt.Errorf("param %q default %q is not numeric", p.Name, p.Default)
				}
				if v < *p.Min || v > *p.Max {
					return fmt.Errorf("param %q default %g is outside [%g,%g]", p.Name, v, *p.Min, *p.Max)
				}
			}
		case "select":
			if len(p.Options) == 0 {
				return fmt.Errorf("param %q select control requires options", p.Name)
			}
			if p.Default != "" && !hasOption(p.Options, p.Default) {
				return fmt.Errorf("param %q default %q is not a declared option", p.Name, p.Default)
			}
		case "file", "files":
			if p.Required {
				requiredFile = true
			}
		}
		if (p.Control == "file" || p.Control == "files") && p.Default != "" {
			return fmt.Errorf("param %q is a file control and must not declare a default path", p.Name)
		}
	}
	if op.Family == "transform" && !requiredFile {
		return fmt.Errorf("transform operations must declare at least one required file input")
	}
	return nil
}

func validateCLI(op Operation) error {
	if op.CLI.Group == "" || op.CLI.Verb == "" {
		return fmt.Errorf("CLI group and verb are required when a CLI spec is declared")
	}
	if op.CLI.Service == "" || op.CLI.Method == "" {
		return fmt.Errorf("CLI service and method are required when a CLI spec is declared")
	}
	params := make(map[string]Param, len(op.Params))
	for _, p := range op.Params {
		params[p.Name] = p
	}
	seen := make(map[string]struct{}, len(op.CLI.Flags))
	hasInput := false
	for _, f := range op.CLI.Flags {
		if f.Name == "" {
			return fmt.Errorf("CLI flag name is required")
		}
		if _, dup := seen[f.Name]; dup {
			return fmt.Errorf("CLI flag %q is declared more than once", f.Name)
		}
		seen[f.Name] = struct{}{}
		if f.LocalOnly {
			if f.Field != "" {
				return fmt.Errorf("CLI flag %q is local-only but declares field %q", f.Name, f.Field)
			}
			continue
		}
		if f.Field == "" {
			return fmt.Errorf("CLI flag %q must declare the request field it fills", f.Name)
		}
		p, ok := params[f.Field]
		if !ok {
			return fmt.Errorf("CLI flag %q maps to unknown param %q", f.Name, f.Field)
		}
		if f.Required != p.Required {
			return fmt.Errorf("CLI flag %q required=%t disagrees with param %q required=%t", f.Name, f.Required, p.Name, p.Required)
		}
		if p.Control == "file" || p.Control == "files" {
			hasInput = true
		}
	}
	if op.Family == "transform" && !hasInput {
		return fmt.Errorf("transform CLI spec must declare the required file input flag")
	}
	return nil
}

func hasOption(options []Option, value string) bool {
	for _, o := range options {
		if o.Value == value {
			return true
		}
	}
	return false
}

func isSlug(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r == '_' && i > 0:
		case r >= '0' && r <= '9' && i > 0:
		default:
			return false
		}
	}
	return true
}
