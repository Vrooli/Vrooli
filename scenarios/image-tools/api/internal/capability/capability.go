// Package capability owns the declarative capability manifest set: one record
// per operation in the vocabulary SSOT (internal/operations), plus the loader,
// validator, and accessors.
//
// A manifest answers, as data: what family the operation belongs to, which
// mechanism runs it (compute / infer / fetch), its I/O ports, its environment
// requirements, its ordered backend preference, and its consent weight. It does
// NOT own model facts (weights, checksums, licences) — those stay in
// internal/models — and it does not contain pixel or inference code.
//
// The consent weight is load-bearing today: internal/safety reads it from here,
// so the manifest set is the single source for that table. The remaining fields
// are consumed by the Phase-1 conformance tests and the upcoming CLI/UI codegen.
// See docs/internal/CAPABILITY-MANIFEST.md.
package capability

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"image-tools/internal/paramschema"
)

// Family is the application axis: what the user is trying to achieve. It is
// presentation/discovery only and must never decide execution ownership.
type Family string

const (
	FamilyGenerate Family = "generate"
	FamilyEdit     Family = "edit"
	FamilyEnhance  Family = "enhance"
	FamilyRestore  Family = "restore"
	FamilyExtract  Family = "extract"
	FamilyCompose  Family = "compose"
)

func (f Family) valid() bool {
	switch f {
	case FamilyGenerate, FamilyEdit, FamilyEnhance, FamilyRestore, FamilyExtract, FamilyCompose:
		return true
	default:
		return false
	}
}

// ValidFamily reports whether s names an application family. Discovery filters
// reject an unknown value rather than returning an empty catalog.
func ValidFamily(s string) bool { return Family(s).valid() }

// Mechanism is what actually executes the operation.
type Mechanism string

const (
	// MechanismCompute is closed-form code over pixels/bytes; no learned weights.
	MechanismCompute Mechanism = "compute"
	// MechanismInfer is learned weights on a runtime.
	MechanismInfer Mechanism = "infer"
	// MechanismFetch is a remote, credential-gated capability.
	MechanismFetch Mechanism = "fetch"
)

func (m Mechanism) valid() bool {
	switch m {
	case MechanismCompute, MechanismInfer, MechanismFetch:
		return true
	default:
		return false
	}
}

// ConsentWeight is how much an operation can change who a person is. It is
// policy input, never an availability gate.
type ConsentWeight string

const (
	ConsentWeightNone ConsentWeight = "none"
	ConsentWeightLow  ConsentWeight = "low"
	ConsentWeightHigh ConsentWeight = "high"
)

func (w ConsentWeight) valid() bool {
	switch w {
	case ConsentWeightNone, ConsentWeightLow, ConsentWeightHigh:
		return true
	default:
		return false
	}
}

// ReferencePort is the input name that marks an image port as a reference image
// (the source face of face_swap) rather than the image being edited. The AI
// engine, the REST edge and the CLI generator all key on it, so the name lives
// once, here.
const ReferencePort = "reference"

// PortSpec is one typed input or output. Kind is one of image / mask / text /
// data / audio / video / mesh (see docs/concepts/COMPOSITION.md).
type PortSpec struct {
	Name     string `json:"name,omitempty"`
	Kind     string `json:"kind"`
	Required bool   `json:"required,omitempty"`
}

// IO declares the operation's ports for composition and validation.
type IO struct {
	Inputs []PortSpec `json:"inputs"`
	Output PortSpec   `json:"output"`
}

// Requires is the environment requirement class. The fields are independent
// facts, not a single enum, because an operation can need a process and weights
// (diffusers) or a process only (tesseract).
type Requires struct {
	Process bool `json:"process"`
	Weights bool `json:"weights"`
	Network bool `json:"network"`
}

// Engine names the runtime that executes an operation. It is the single
// membership fact: the AI job lane, the structured analysis service, the
// in-process compute table, and forward-declared operations each read it, so no
// consumer keeps a second list of which operations it serves.
//
// It replaces the earlier `registry` discriminator, which named the Go table an
// operation lived in rather than the runtime that runs it — the two differed for
// `segment` and `caption` (declared in the vocabulary, executed by the AI engine)
// and for the analysis operations (declared in the vocabulary, executed by the
// analysis service).
type Engine string

const (
	// EngineAI is the durable model-backed job lane (internal/ai).
	EngineAI Engine = "ai"
	// EngineAnalysis is the structured image→data service (internal/analysis).
	EngineAnalysis Engine = "analysis"
	// EngineCompute is the in-process deterministic table (internal/ops).
	EngineCompute Engine = "compute"
	// EngineUnwired is declared (models may serve it) but has no runner yet. An
	// operation in this state is a promise, not a capability: discovery reports it
	// and any attempt to run it fails with an actionable error.
	EngineUnwired Engine = "unwired"
)

func (e Engine) valid() bool {
	switch e {
	case EngineAI, EngineAnalysis, EngineCompute, EngineUnwired:
		return true
	default:
		return false
	}
}

// Runs reports whether the engine has a runner for the operation.
func (e Engine) Runs() bool { return e == EngineAI || e == EngineAnalysis || e == EngineCompute }

// CLISpec declares how an operation appears as a headless CLI subcommand. The
// generated command table is produced from this, so adding an AI operation no
// longer requires editing the CLI by hand.
type CLISpec struct {
	// Group is the CLI command group ("ai").
	Group string `json:"group"`
	// Verb is the subcommand name (e.g. "img2img", "bg-removal").
	Verb string `json:"verb"`
	// Flags are the operation-specific flag names; common flags are appended by
	// the CLI. Names are resolved through the CLI's flag-definition table.
	Flags []string `json:"flags"`
}

// Presentation is the UI-facing metadata for an operation. Labels stay in i18n;
// the manifest carries the keys (and an icon name) so the UI catalog is
// generated rather than hand-maintained per operation.
type Presentation struct {
	Icon     string `json:"icon"`
	LabelKey string `json:"label_key"`
	DescKey  string `json:"desc_key"`
	// Fields declares the parameter form for an in-process operation. The
	// TypeScript spec is generated from this declaration and cross-checked
	// against the OpParams descriptor, so the form cannot name a field the wire
	// does not carry. Model-backed operations leave it empty: their knobs are
	// declared by params_schema.
	Fields []UIField `json:"fields,omitempty"`
	// AcceptsOverlay marks the one operation that consumes a second image part.
	AcceptsOverlay bool `json:"accepts_overlay,omitempty"`
}

// UIField is one control in a deterministic operation's parameter form. It is
// presentation data, not an execution contract: the wire field is still the
// proto message, and the generator fails if Name does not exist there.
type UIField struct {
	Name    string `json:"name"`
	Control string `json:"control"`
	// LabelKey is the i18n path below `workspace.field.` (e.g. `width`).
	LabelKey string `json:"label_key"`
	// Default is the raw JSON literal the control starts at, so an integer stays
	// `0` and a small float stays `0.04` through generation.
	Default json.RawMessage `json:"default"`
	// Options is a closed set of wire tokens for segmented/format/filter controls.
	Options []string `json:"options,omitempty"`
	// OptionsFrom names a shared generated list (`encode_formats`) instead of
	// repeating it in every manifest.
	OptionsFrom string   `json:"options_from,omitempty"`
	Min         *float64 `json:"min,omitempty"`
	Max         *float64 `json:"max,omitempty"`
	Step        *float64 `json:"step,omitempty"`
	Unit        string   `json:"unit,omitempty"`
	Advanced    bool     `json:"advanced,omitempty"`
}

// UIFieldControls is the closed set of controls the Inspector can render.
var UIFieldControls = []string{
	"number", "text", "toggle", "segmented", "slider",
	"position", "color", "format", "targetSize", "filterGrid",
}

func validUIFieldControl(c string) bool {
	for _, known := range UIFieldControls {
		if c == known {
			return true
		}
	}
	return false
}

// Panels are the workspace surfaces a manifest may name.
var Panels = []string{"create", "enhance", "analyze"}

func validPanel(p string) bool {
	switch p {
	case "", "create", "enhance", "analyze":
		return true
	default:
		return false
	}
}

// Manifest is one capability record.
type Manifest struct {
	ID            string        `json:"id"`
	Engine        Engine        `json:"engine"`
	Summary       string        `json:"summary"`
	Family        Family        `json:"family"`
	Mechanism     Mechanism     `json:"mechanism"`
	IO            IO            `json:"io"`
	Requires      Requires      `json:"requires"`
	Backends      []string      `json:"backends"`
	ConsentWeight ConsentWeight `json:"consent_weight"`
	// ParamsSchema is the operation's parameter contract (JSON Schema, validated
	// by internal/paramschema). It replaces adding a typed AIParams sub-message
	// for every new operation.
	ParamsSchema json.RawMessage `json:"params_schema,omitempty"`
	CLI          *CLISpec        `json:"cli,omitempty"`
	Presentation *Presentation   `json:"presentation,omitempty"`
	// Panel is the workspace surface this operation appears on: create, enhance,
	// or analyze. Empty means "not surfaced in a workspace panel" (it is reached
	// through a dedicated editor, another service's list, or the CLI).
	//
	// It is deliberately separate from Family. The family is what the operation
	// does; the panel is where a person finds it. Conflating the two is what made
	// the old single `category` field lie — `background_removal` is an extraction
	// but was filed as an enhancement because that is where it appears.
	Panel string `json:"panel,omitempty"`
	// Extent declares how much of a person the operation can transfer, as an
	// ordered set of independent stages and the presets composed from them. See
	// ExtentSpec.
	Extent *ExtentSpec `json:"extent,omitempty"`
	// Preset declares that this operation is a named way of running another one,
	// with a default model and a standing adapter stack. See PresetSpec.
	Preset *PresetSpec `json:"preset,omitempty"`
}

// seedDoc is the on-disk shape of manifests.seed.json.
type seedDoc struct {
	SchemaVersion string     `json:"schema_version"`
	Manifests     []Manifest `json:"manifests"`
}

//go:embed manifests.seed.json
var seedBytes []byte

// Registry is the validated manifest set.
type Registry struct {
	schemaVersion string
	manifests     []Manifest
	byID          map[string]Manifest
	cliVerbs      map[string]string
	schemas       map[string]*paramschema.Schema
}

// Load parses and validates the embedded manifest seed. It never reads the
// filesystem at runtime; the seed ships in the binary.
func Load() (*Registry, error) {
	var doc seedDoc
	if err := json.Unmarshal(seedBytes, &doc); err != nil {
		return nil, fmt.Errorf("capability: parse manifests: %w", err)
	}
	if doc.SchemaVersion == "" {
		return nil, fmt.Errorf("capability: manifests.seed.json has no schema_version")
	}
	r := &Registry{
		schemaVersion: doc.SchemaVersion,
		manifests:     doc.Manifests,
		byID:          make(map[string]Manifest, len(doc.Manifests)),
		cliVerbs:      make(map[string]string, len(doc.Manifests)),
		schemas:       make(map[string]*paramschema.Schema, len(doc.Manifests)),
	}
	for i := range r.manifests {
		r.manifests[i].applyDefaults()
	}
	if err := r.validate(); err != nil {
		return nil, err
	}
	return r, nil
}

// applyDefaults fills the facts a manifest's engine already implies. A
// deterministic operation is compute by definition, runs in this process, needs
// no backend and no environment requirement, and cannot alter who a person is —
// so a deterministic record declares only what is specific to it. The zero values
// are the correct ones for every deterministic field; only the mechanism and the
// consent weight need to be made explicit for downstream readers.
//
// The model-backed registry keeps every field explicit: its operations differ in
// every one of them, and the consent weight is the input to a host-policy
// decision that must never be guessed.
func (m *Manifest) applyDefaults() {
	if m.Engine != EngineCompute {
		return
	}
	m.Mechanism = MechanismCompute
	if m.ConsentWeight == "" {
		m.ConsentWeight = ConsentWeightNone
	}
	// An in-process operation edits one image and produces one image. That is what
	// 34 of the 38 compute operations declare, so an omitted io block means exactly
	// that; the four that differ (a metadata read, an SVG trace, an SVG rasterise,
	// and the screenshot trim) declare their ports explicitly.
	if len(m.IO.Inputs) == 0 && m.IO.Output.Kind == "" {
		m.IO = IO{
			Inputs: []PortSpec{{Name: "image", Kind: "image", Required: true}},
			Output: PortSpec{Kind: "image"},
		}
	}
}

// validate enforces structural integrity and vocabulary membership. Model-backend
// agreement is a conformance test (it would couple this package to internal/models,
// which the runtime does not need).
func (r *Registry) validate() error {
	if len(r.manifests) == 0 {
		return fmt.Errorf("capability: manifest set is empty")
	}
	for _, m := range r.manifests {
		if m.ID == "" {
			return fmt.Errorf("capability: manifest with empty id")
		}
		if _, dup := r.byID[m.ID]; dup {
			return fmt.Errorf("capability: duplicate manifest %q", m.ID)
		}
		if !m.Engine.valid() {
			return fmt.Errorf("capability: manifest %q has invalid engine %q", m.ID, m.Engine)
		}
		// Membership is the manifest set itself: this package is the vocabulary
		// of record. The conformance tests prove the runners agree with it (every
		// engine=compute manifest has a compute function, every engine=ai manifest
		// is in the AI catalog, and so on), which is the direction that cannot
		// drift — a runner cannot exist for an undeclared operation.
		if m.Engine == EngineCompute && m.Mechanism != MechanismCompute {
			return fmt.Errorf("capability: deterministic manifest %q must have mechanism compute", m.ID)
		}
		// A deterministic record must not declare an environment requirement or a
		// backend: it runs in this process.
		if m.Engine == EngineCompute && (len(m.Backends) > 0 || m.Requires.Process || m.Requires.Weights || m.Requires.Network) {
			return fmt.Errorf("capability: deterministic manifest %q declares a backend or an external requirement", m.ID)
		}
		if m.Summary == "" {
			return fmt.Errorf("capability: manifest %q has no summary", m.ID)
		}
		if !m.Family.valid() {
			return fmt.Errorf("capability: manifest %q has invalid family %q", m.ID, m.Family)
		}
		if !m.Mechanism.valid() {
			return fmt.Errorf("capability: manifest %q has invalid mechanism %q", m.ID, m.Mechanism)
		}
		if m.Engine != EngineCompute && m.ConsentWeight == "" {
			// Never default this for a model-backed op: a host that enables consent
			// enforcement must not inherit "none" from an omission.
			return fmt.Errorf("capability: model-backed manifest %q must declare consent_weight", m.ID)
		}
		if !m.ConsentWeight.valid() {
			return fmt.Errorf("capability: manifest %q has invalid consent_weight %q", m.ID, m.ConsentWeight)
		}
		// A model-backed manifest must name at least one backend. In-process
		// deterministic ops run in the API binary and have no backend, so an
		// empty list is valid for the deterministic registry.
		if m.Requires.Weights && len(m.Backends) == 0 {
			return fmt.Errorf("capability: manifest %q declares no backends", m.ID)
		}
		if m.IO.Output.Kind == "" {
			return fmt.Errorf("capability: manifest %q has no output kind", m.ID)
		}
		for _, in := range m.IO.Inputs {
			if in.Kind == "" {
				return fmt.Errorf("capability: manifest %q has an input with no kind", m.ID)
			}
		}
		if len(m.ParamsSchema) > 0 {
			sch, err := paramschema.Compile(m.ParamsSchema)
			if err != nil {
				return fmt.Errorf("capability: manifest %q params_schema: %w", m.ID, err)
			}
			r.schemas[m.ID] = sch
		}
		if m.Presentation != nil {
			if m.Presentation.Icon == "" || m.Presentation.LabelKey == "" || m.Presentation.DescKey == "" {
				return fmt.Errorf("capability: manifest %q has an incomplete presentation block", m.ID)
			}
			if err := validateUIFields(m); err != nil {
				return err
			}
		}
		if err := validateExtent(m.ID, m.ParamsSchema, m.Extent); err != nil {
			return fmt.Errorf("capability: %w", err)
		}
		if !validPanel(m.Panel) {
			return fmt.Errorf("capability: manifest %q has unknown panel %q", m.ID, m.Panel)
		}
		if m.CLI != nil {
			if m.CLI.Group == "" || m.CLI.Verb == "" {
				return fmt.Errorf("capability: manifest %q has an incomplete cli spec", m.ID)
			}
			if err := validateVerb(m.CLI.Verb); err != nil {
				return fmt.Errorf("capability: manifest %q: %w", m.ID, err)
			}
			seen := make(map[string]struct{}, len(m.CLI.Flags))
			for _, f := range m.CLI.Flags {
				if f == "" {
					return fmt.Errorf("capability: manifest %q has an empty cli flag", m.ID)
				}
				if _, dup := seen[f]; dup {
					return fmt.Errorf("capability: manifest %q repeats cli flag %q", m.ID, f)
				}
				seen[f] = struct{}{}
			}
			key := m.CLI.Group + " " + m.CLI.Verb
			if _, dup := r.cliVerbs[key]; dup {
				return fmt.Errorf("capability: duplicate cli verb %q in group %q", m.CLI.Verb, m.CLI.Group)
			}
			r.cliVerbs[key] = m.ID
		}
		r.byID[m.ID] = m
	}
	// Presets name other operations, so they are checked once all are known.
	for _, m := range r.manifests {
		if err := validatePreset(r, m); err != nil {
			return err
		}
	}
	return nil
}

// validateUIFields keeps the generated form honest: a control the Inspector
// cannot render, a default of the wrong shape, options on a free-text control, or
// a form on a model-backed op is a declaration bug, not a runtime surprise.
func validateUIFields(m Manifest) error {
	p := m.Presentation
	if p == nil {
		return nil
	}
	if len(p.Fields) == 0 {
		if p.AcceptsOverlay {
			return fmt.Errorf("capability: manifest %q declares an overlay without a form", m.ID)
		}
		return nil
	}
	if m.Engine != EngineCompute {
		return fmt.Errorf("capability: model-backed manifest %q must not declare presentation.fields", m.ID)
	}
	seen := make(map[string]struct{}, len(p.Fields))
	for _, f := range p.Fields {
		if f.Name == "" {
			return fmt.Errorf("capability: manifest %q has a field with no name", m.ID)
		}
		if _, dup := seen[f.Name]; dup {
			return fmt.Errorf("capability: manifest %q repeats field %q", m.ID, f.Name)
		}
		seen[f.Name] = struct{}{}
		if !validUIFieldControl(f.Control) {
			return fmt.Errorf("capability: manifest %q field %q has unknown control %q", m.ID, f.Name, f.Control)
		}
		if !strings.HasPrefix(f.LabelKey, "workspace.field.") || strings.ContainsAny(f.LabelKey, " \t") {
			return fmt.Errorf("capability: manifest %q field %q has invalid label_key %q", m.ID, f.Name, f.LabelKey)
		}
		if len(f.Default) == 0 || !json.Valid(f.Default) {
			return fmt.Errorf("capability: manifest %q field %q has no JSON default", m.ID, f.Name)
		}
		numeric := f.Control == "number" || f.Control == "slider" || f.Control == "targetSize"
		switch {
		case f.Control == "toggle":
			var v bool
			if json.Unmarshal(f.Default, &v) != nil {
				return fmt.Errorf("capability: manifest %q field %q default must be a boolean", m.ID, f.Name)
			}
		case numeric:
			var v float64
			if json.Unmarshal(f.Default, &v) != nil {
				return fmt.Errorf("capability: manifest %q field %q default must be a number", m.ID, f.Name)
			}
		default:
			var v string
			if json.Unmarshal(f.Default, &v) != nil {
				return fmt.Errorf("capability: manifest %q field %q default must be a string", m.ID, f.Name)
			}
		}
		needsOptions := f.Control == "segmented" || f.Control == "format" || f.Control == "filterGrid"
		hasOptions := len(f.Options) > 0 || f.OptionsFrom != ""
		if needsOptions && !hasOptions {
			return fmt.Errorf("capability: manifest %q field %q needs options", m.ID, f.Name)
		}
		if !needsOptions && hasOptions {
			return fmt.Errorf("capability: manifest %q field %q declares options for a free-form control", m.ID, f.Name)
		}
		if len(f.Options) > 0 && f.OptionsFrom != "" {
			return fmt.Errorf("capability: manifest %q field %q declares both options and options_from", m.ID, f.Name)
		}
		if f.OptionsFrom != "" && f.OptionsFrom != "encode_formats" {
			return fmt.Errorf("capability: manifest %q field %q names unknown options_from %q", m.ID, f.Name, f.OptionsFrom)
		}
		if !numeric && (f.Min != nil || f.Max != nil || f.Step != nil) {
			return fmt.Errorf("capability: manifest %q field %q has numeric bounds on a non-numeric control", m.ID, f.Name)
		}
		if f.Min != nil && f.Max != nil && *f.Min > *f.Max {
			return fmt.Errorf("capability: manifest %q field %q has min > max", m.ID, f.Name)
		}
		if f.Step != nil && *f.Step <= 0 {
			return fmt.Errorf("capability: manifest %q field %q has non-positive step", m.ID, f.Name)
		}
	}
	return nil
}

// Vocabulary returns the non-compute operations (the model-backed, analysed, and
// forward-declared vocabulary) in seed order. It replaces internal/operations:
// the manifest set is the declaration of record, and this is the view consumers
// that used to iterate that table now read.
func (r *Registry) Vocabulary() []Manifest {
	out := make([]Manifest, 0, len(r.manifests))
	for _, m := range r.manifests {
		if m.Engine != EngineCompute {
			out = append(out, m)
		}
	}
	return out
}

// VocabularyNames returns the non-compute operation ids in seed order.
func (r *Registry) VocabularyNames() []string {
	out := make([]string, 0, len(r.manifests))
	for _, m := range r.manifests {
		if m.Engine != EngineCompute {
			out = append(out, m.ID)
		}
	}
	return out
}

// ByEngine returns the manifests an engine serves, in seed order.
func (r *Registry) ByEngine(engine Engine) []Manifest {
	out := make([]Manifest, 0, len(r.manifests))
	for _, m := range r.manifests {
		if m.Engine == engine {
			out = append(out, m)
		}
	}
	return out
}

// Has reports whether the operation is declared by any manifest.
func (r *Registry) Has(id string) bool {
	_, ok := r.byID[id]
	return ok
}

// Runs reports whether the operation has an engine that can execute it.
func (r *Registry) Runs(id string) bool {
	m, ok := r.byID[id]
	return ok && m.Engine.Runs()
}

// Vocabulary returns the default registry's non-compute manifests.
func Vocabulary() []Manifest { return registry.Vocabulary() }

// VocabularyNames returns the default registry's vocabulary ids.
func VocabularyNames() []string { return registry.VocabularyNames() }

// ByEngine returns the default registry's manifests for one engine.
func ByEngine(engine Engine) []Manifest { return registry.ByEngine(engine) }

// Has reports whether the default registry declares the operation.
func Has(id string) bool { return registry.Has(id) }

func (r *Registry) All() []Manifest { return append([]Manifest(nil), r.manifests...) }

// Names returns every manifest id in seed order.
func (r *Registry) Names() []string {
	out := make([]string, len(r.manifests))
	for i, m := range r.manifests {
		out[i] = m.ID
	}
	return out
}

// Get returns the manifest and whether it exists.
func (r *Registry) Get(id string) (Manifest, bool) {
	m, ok := r.byID[id]
	return m, ok
}

// ConsentWeight returns the declared weight, defaulting to none for an unknown op.
func (r *Registry) ConsentWeight(id string) ConsentWeight {
	if m, ok := r.byID[id]; ok {
		return m.ConsentWeight
	}
	return ConsentWeightNone
}

// Weights returns a fresh copy of the id→weight map.
func (r *Registry) Weights() map[string]ConsentWeight {
	out := make(map[string]ConsentWeight, len(r.manifests))
	for _, m := range r.manifests {
		out[m.ID] = m.ConsentWeight
	}
	return out
}

// SchemaVersion returns the seed schema version.
func (r *Registry) SchemaVersion() string { return r.schemaVersion }

// SortedNames returns the manifest ids sorted, for deterministic reporting.
func (r *Registry) SortedNames() []string {
	out := r.Names()
	sort.Strings(out)
	return out
}

// registry is the process-wide validated set. A bad seed fails at package init
// (the same failure model as internal/operations), so a malformed manifest cannot
// reach a running server.
var registry = func() *Registry {
	r, err := Load()
	if err != nil {
		panic(err)
	}
	return r
}()

// Default returns the process-wide manifest registry.
func Default() *Registry { return registry }

// All returns every manifest from the default registry.
func All() []Manifest { return registry.All() }

// Names returns every manifest id from the default registry.
func Names() []string { return registry.Names() }

// Get returns a manifest from the default registry.
func Get(id string) (Manifest, bool) { return registry.Get(id) }

// ConsentWeightFor returns the declared consent weight for an operation,
// defaulting to none for an unknown op.
func ConsentWeightFor(id string) ConsentWeight { return registry.ConsentWeight(id) }

// Weights returns a fresh id→weight copy from the default registry.
func Weights() map[string]ConsentWeight { return registry.Weights() }

// validateVerb enforces the kebab-case CLI verb shape.
func validateVerb(verb string) error {
	if strings.HasPrefix(verb, "-") || strings.HasSuffix(verb, "-") {
		return fmt.Errorf("cli verb %q must not start or end with a hyphen", verb)
	}
	for _, r := range verb {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return fmt.Errorf("cli verb %q must be kebab-case [a-z0-9-]", verb)
		}
	}
	return nil
}

// CLICommands returns the manifests that declare a CLI surface, in seed order.
func (r *Registry) CLICommands() []Manifest {
	var out []Manifest
	for _, m := range r.manifests {
		if m.CLI != nil {
			out = append(out, m)
		}
	}
	return out
}

// CLICommands returns the default registry's CLI-surface manifests.
func CLICommands() []Manifest { return registry.CLICommands() }

// ParamsSchema returns the compiled parameter schema for an operation, if any.
func (r *Registry) ParamsSchema(id string) (*paramschema.Schema, bool) {
	s, ok := r.schemas[id]
	return s, ok
}

// ValidateOpParams validates operation-specific parameters against the operation's
// declared schema. When the operation declares no schema, any non-empty values are
// rejected — an operation must opt in to extra parameters explicitly rather than
// accepting silently-ignored input.
func (r *Registry) ValidateOpParams(id string, values map[string]any) error {
	s, ok := r.schemas[id]
	if !ok {
		if len(values) > 0 {
			return fmt.Errorf("operation %q does not accept op_params", id)
		}
		return nil
	}
	if err := s.Validate(values); err != nil {
		return fmt.Errorf("operation %q op_params: %w", id, err)
	}
	return nil
}

// ValidateOpParams validates against the default registry.
func ValidateOpParams(id string, values map[string]any) error {
	return registry.ValidateOpParams(id, values)
}

// ParamsSchemaFor returns the compiled schema from the default registry.
func ParamsSchemaFor(id string) (*paramschema.Schema, bool) { return registry.ParamsSchema(id) }
