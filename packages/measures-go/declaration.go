package measures

import (
	"fmt"
	"sort"
	"strings"
)

// Effect is the side-effect class of a measure, mirroring the manifest
// `governance.effect` enum. It is the single signal the auto-execution gate
// (Phase 3) keys on: only `read` measures may ever be auto-executed.
type Effect string

const (
	// EffectRead is a side-effect-free measure (a pure analytical query). Only
	// these are eligible for auto-execution.
	EffectRead Effect = "read"
	// EffectWrite mutates state; never auto-executed (always confirm).
	EffectWrite Effect = "write"
	// EffectDestructive deletes/irreversibly mutates state; never auto-executed.
	EffectDestructive Effect = "destructive"
)

// Valid reports whether e is one of the known effect classes.
func (e Effect) Valid() bool {
	switch e {
	case EffectRead, EffectWrite, EffectDestructive:
		return true
	default:
		return false
	}
}

// AutoExecutable reports whether a measure with this effect may ever be
// auto-executed. Only read measures qualify; write/destructive always confirm.
func (e Effect) AutoExecutable() bool { return e == EffectRead }

// ResultKind is the shape of a measure's answer.
type ResultKind string

const (
	// ResultScalar is a single value (e.g. a count). value_field names the
	// scalar in the result.
	ResultScalar ResultKind = "scalar"
	// ResultTable is a set of rows (Fields on MeasureResult).
	ResultTable ResultKind = "table"
	// ResultSeries is an ordered set of (time, value) points (Fields).
	ResultSeries ResultKind = "series"
)

// Valid reports whether k is a known result kind.
func (k ResultKind) Valid() bool {
	switch k {
	case ResultScalar, ResultTable, ResultSeries:
		return true
	default:
		return false
	}
}

// Canonical parameter type names. These are the *convention* labels the
// resolver dispatches on (see params.go). Most types are the proto kind
// ("string", "int32", ...); two carry special resolution semantics:
//   - ParamTypeTimeWindow resolves deterministically (timewindow.go), no LLM.
//   - ParamTypeEnum is a constrained set; a static enum derives its values from
//     proto, a dynamic enum supplies them at runtime via ValuesSource.
const (
	ParamTypeTimeWindow = "time_window"
	ParamTypeEnum       = "enum"
)

// Param is a single measure parameter: the proto-derived validation truth
// (Type, EnumValues, Min/Max, Format, Required) overlaid with the manifest's
// curated presentation (Default, ValuesSource). Param types/enums/bounds are
// NEVER duplicated in the manifest — they are derived from the proto descriptor
// (Phase 0 reader). The manifest only contributes what proto cannot express.
type Param struct {
	// Name is the proto field name (snake_case).
	Name string `json:"name"`

	// Type is the canonical convention label the resolver dispatches on
	// ("time_window", "enum", or a proto kind like "string"/"int32").
	Type string `json:"type"`

	// Default is the fallback token/value used when a question does not name
	// the parameter (manifest-supplied). Empty when there is no default.
	Default string `json:"default,omitempty"`

	// ValuesSource names a runtime callback the owning scenario supplies for a
	// *dynamic* enum (e.g. "initiative_names" → live initiative names). Empty
	// for static enums (values come from proto) and non-enum params.
	ValuesSource string `json:"values_source,omitempty"`

	// Required is true when the param must be present for the measure to run
	// (from buf.validate.field.required, or a non-optional scalar with no
	// default). A required param the question does not resolve goes to needs[].
	Required bool `json:"required"`

	// EnumValues are the permitted values for a static enum (proto-derived).
	// Empty for dynamic enums and non-enum params.
	EnumValues []string `json:"enum_values,omitempty"`

	// Min / Max carry proto-derived numeric bounds (nil when unconstrained).
	Min *int64 `json:"min,omitempty"`
	Max *int64 `json:"max,omitempty"`

	// Format is a proto-derived string-format hint ("uuid"). Empty otherwise.
	Format string `json:"format,omitempty"`

	// Description is the field's leading proto comment (grounding for
	// constrained extraction). Empty when absent.
	Description string `json:"description,omitempty"`
}

// IsCanonical reports whether the param resolves deterministically (no LLM).
func (p Param) IsCanonical() bool { return p.Type == ParamTypeTimeWindow }

// IsConstrained reports whether the param has a bounded value space the
// extractor can be constrained against (a static/dynamic enum or numeric
// bounds), as opposed to a free-form best-effort field.
func (p Param) IsConstrained() bool {
	return p.Type == ParamTypeEnum || len(p.EnumValues) > 0 || p.Min != nil || p.Max != nil || p.ValuesSource != ""
}

// Result describes how a measure's answer is shaped and presented.
type Result struct {
	Kind ResultKind `json:"kind"`
	// ValueField names the field in the MeasureResult that carries the answer
	// (for scalar) or the primary column (for table/series). Mandatory.
	ValueField string `json:"value_field"`
	// Unit is a human label for the value ("items", "hours"). Optional.
	Unit string `json:"unit,omitempty"`
	// SummaryTemplate is a one-line answer template with {placeholders} filled
	// from the result + resolved params (e.g. "{count} items ({window})").
	SummaryTemplate string `json:"summary_template,omitempty"`
}

// MeasureDeclaration is the SSOT for what a measure is: a named, typed,
// parameterized analytical query. It is assembled at runtime from three sources
// joined on the manifest binding — see Assemble.
type MeasureDeclaration struct {
	// Name is the measure identifier, conventionally "<domain>.<verb>"
	// (e.g. "backlog.completed").
	Name string `json:"name"`
	// Scenario is the owning scenario (e.g. "swarm-manager") — who computes the
	// answer. Populated by the central index when a measure is harvested; the
	// execution-proxy resolves this to a live base URL. Empty for a bare
	// declaration that has not yet been attributed to a scenario.
	Scenario string `json:"scenario,omitempty"`
	// Domain is the stateful domain the measure covers (e.g. "backlog").
	Domain string `json:"domain"`
	// Intent is curated prose: what the measure answers (manifest-supplied).
	Intent string `json:"intent"`
	// Questions are natural-language phrasings this measure answers; they are
	// the embedding key (Phase 3) for semantic matching.
	Questions []string `json:"questions"`
	// Params is the parameter set keyed by proto field name.
	Params map[string]Param `json:"params"`
	// Result describes the answer shape/presentation.
	Result Result `json:"result"`
	// Effect is the side-effect class (governance-derived); gates auto-exec.
	Effect Effect `json:"effect"`
	// RunEligible mirrors governance.run_eligible; a precondition for auto-exec.
	RunEligible bool `json:"run_eligible"`

	// Service / Method are the manifest binding the params were derived from,
	// retained so the central index can resolve the owning RPC at execution.
	Service string `json:"service"`
	Method  string `json:"method"`
}

// ParamNames returns the param names in deterministic (sorted) order so that
// needs[] ordering and iteration are stable.
func (d MeasureDeclaration) ParamNames() []string {
	names := make([]string, 0, len(d.Params))
	for n := range d.Params {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// AutoExecutable reports whether this measure is eligible for auto-execution
// (read effect AND run-eligible). The confidence gate is applied separately by
// the caller (Phase 3).
func (d MeasureDeclaration) AutoExecutable() bool {
	return d.Effect.AutoExecutable() && d.RunEligible
}

// Validate checks the declaration is internally well-formed. It is the static
// contract cli-health/measures-health enforce; calling it at assembly time
// turns drift into an error rather than a silent bad answer.
func (d MeasureDeclaration) Validate() error {
	var errs []string
	if strings.TrimSpace(d.Name) == "" {
		errs = append(errs, "name is empty")
	}
	if strings.TrimSpace(d.Domain) == "" {
		errs = append(errs, "domain is empty")
	}
	if len(d.Questions) == 0 {
		errs = append(errs, "questions[] is empty (no embedding key)")
	}
	if !d.Effect.Valid() {
		errs = append(errs, fmt.Sprintf("effect %q is not one of read|write|destructive", d.Effect))
	}
	if !d.Result.Kind.Valid() {
		errs = append(errs, fmt.Sprintf("result.kind %q is not one of scalar|table|series", d.Result.Kind))
	}
	if strings.TrimSpace(d.Result.ValueField) == "" {
		errs = append(errs, "result.value_field is empty")
	}
	for _, name := range d.ParamNames() {
		p := d.Params[name]
		if strings.TrimSpace(p.Type) == "" {
			errs = append(errs, fmt.Sprintf("param %q has empty type", name))
		}
		if p.Type == ParamTypeEnum && len(p.EnumValues) == 0 && p.ValuesSource == "" {
			errs = append(errs, fmt.Sprintf("param %q is enum but has neither proto enum values nor a values_source", name))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("invalid measure %q: %s", d.Name, strings.Join(errs, "; "))
	}
	return nil
}

// -----------------------------------------------------------------------------
// Assembly: manifest ⊕ governance ⊕ proto-derived param schema
// -----------------------------------------------------------------------------

// ManifestParam is a parameter's curated manifest entry. It carries only what
// the proto descriptor cannot express: a presentation Default, a dynamic-enum
// ValuesSource, and an optional canonical Type annotation. Param types, enum
// membership and numeric bounds are NEVER authored here — they come from proto.
type ManifestParam struct {
	// Type optionally annotates a canonical convention the proto kind cannot
	// imply — specifically marking a plain string field as a dynamic enum.
	// Leave empty to let the proto-derived type stand.
	Type string `json:"type,omitempty"`
	// Default is the fallback token/value when the question omits the param.
	Default string `json:"default,omitempty"`
	// ValuesSource names the runtime values callback for a dynamic enum.
	ValuesSource string `json:"values_source,omitempty"`
}

// ManifestMeasure is the curated `measure` block authored on a CLI manifest
// command (the Go shape of the Phase 2 schema). Curated prose lives here; param
// validation comes from proto.
type ManifestMeasure struct {
	Intent    string                   `json:"intent"`
	Questions []string                 `json:"questions"`
	Params    map[string]ManifestParam `json:"params,omitempty"`
	Result    Result                   `json:"result"`
}

// Governance is the manifest `governance` block fields the measure layer needs:
// the side-effect class and run-eligibility, which together gate auto-exec.
type Governance struct {
	Effect      Effect `json:"effect"`
	RunEligible bool   `json:"run_eligible"`
}

// Binding identifies the Connect-RPC a measure command is bound to. The proto
// param schema is derived from this binding via the Phase 0 SchemaReader.
type Binding struct {
	Service string `json:"service"`
	Method  string `json:"method"`
}

// Assemble joins the three measure sources on the binding into a single
// MeasureDeclaration. The proto-derived param schema is authoritative for
// type/enum/bounds/format/required; the manifest overlays presentation
// (default, dynamic-enum values_source) and curated prose; governance supplies
// the effect/run-eligibility. The returned declaration is Validate-d.
//
// `protoParams` comes from SchemaReader.RequestParams(binding.Service,
// binding.Method). A manifest param naming a field absent from the proto
// request message is drift and returns an error.
func Assemble(name, domain string, b Binding, m ManifestMeasure, g Governance, protoParams []ParamSchema) (MeasureDeclaration, error) {
	bySchema := make(map[string]ParamSchema, len(protoParams))
	for _, ps := range protoParams {
		bySchema[ps.Name] = ps
	}

	// Reject manifest params that do not correspond to a proto request field.
	for mname := range m.Params {
		if _, ok := bySchema[mname]; !ok {
			return MeasureDeclaration{}, fmt.Errorf("measure %q: manifest param %q has no matching field in %s.%s request", name, mname, b.Service, b.Method)
		}
	}

	params := make(map[string]Param, len(protoParams))
	for _, ps := range protoParams {
		mp := m.Params[ps.Name] // zero value if the manifest didn't annotate it
		p := Param{
			Name:         ps.Name,
			Type:         canonicalType(ps, mp),
			Default:      mp.Default,
			ValuesSource: mp.ValuesSource,
			Required:     ps.Required,
			EnumValues:   ps.EnumValues,
			Min:          ps.Min,
			Max:          ps.Max,
			Format:       ps.Format,
			Description:  ps.Description,
		}
		// A non-optional field with no default and no proto-required flag is
		// still effectively required for a meaningful answer; treat a param
		// that is neither optional nor defaulted as required so abstention
		// (needs[]) triggers rather than a silent wrong answer.
		if !p.Required && !ps.Optional && p.Default == "" && p.Type != ParamTypeTimeWindow {
			p.Required = true
		}
		params[ps.Name] = p
	}

	decl := MeasureDeclaration{
		Name:        name,
		Domain:      domain,
		Intent:      m.Intent,
		Questions:   m.Questions,
		Params:      params,
		Result:      m.Result,
		Effect:      g.Effect,
		RunEligible: g.RunEligible,
		Service:     b.Service,
		Method:      b.Method,
	}
	if err := decl.Validate(); err != nil {
		return MeasureDeclaration{}, err
	}
	return decl, nil
}

// canonicalType resolves a param's convention label from the proto schema and
// the optional manifest annotation. Proto is authoritative; the manifest may
// only *upgrade* a plain string field to a dynamic enum (via values_source) —
// it cannot override the proto kind to an incompatible canonical type.
func canonicalType(ps ParamSchema, mp ManifestParam) string {
	switch {
	case ps.Type == ParamTypeTimeWindow:
		return ParamTypeTimeWindow
	case ps.Type == "enum":
		return ParamTypeEnum
	case mp.ValuesSource != "" && ps.Type == "string":
		// Manifest upgrades a free-form string to a dynamic enum.
		return ParamTypeEnum
	case mp.Type != "":
		// Honor an explicit manifest annotation only when it does not contradict
		// the proto kind. time_window/enum require a message/enum proto kind; a
		// scalar field cannot be annotated into one.
		return mp.Type
	default:
		return ps.Type
	}
}
