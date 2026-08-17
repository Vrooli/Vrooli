package prose

import (
	"encoding/json"
	"time"

	"github.com/vrooli/textmetrics"
)

type Style struct {
	Key          string             `json:"key"`
	Version      int                `json:"version"`
	Parent       string             `json:"parent,omitempty"`
	Exemplars    []string           `json:"exemplars,omitempty"`
	Directives   []string           `json:"directives,omitempty"`
	AntiPatterns []string           `json:"anti_patterns,omitempty"`
	Lexicon      []string           `json:"lexicon,omitempty"`
	Targets      map[string]float64 `json:"targets,omitempty"`
	AxisDefaults map[string]string  `json:"axis_defaults,omitempty"`
	Authority    string             `json:"authority"`
	SourcePath   string             `json:"source_path,omitempty"`
	ContentHash  string             `json:"content_hash,omitempty"`
	Frozen       bool               `json:"frozen"`
	Status       string             `json:"status"`
	CreatedAt    time.Time          `json:"created_at"`
}

type Profile struct {
	Key              string             `json:"key"`
	Version          int                `json:"version"`
	Parent           string             `json:"parent,omitempty"`
	StyleRefs        []string           `json:"style_refs,omitempty"`
	Sampler          Sampler            `json:"sampler"`
	Constraints      Constraints        `json:"constraints"`
	SelectionPolicy  string             `json:"selection_policy"`
	SelectionParams  map[string]float64 `json:"selection_params,omitempty"`
	MeasurementTiers []string           `json:"measurement_tiers,omitempty"`
	Budget           Budget             `json:"budget"`
	ContextPolicy    ContextPolicy      `json:"context_policy"`
	GatewayRole      string             `json:"gateway_role"`
	// Locality is the caller's stance on where inference may run, named in the
	// gateway's Profile vocabulary without the PROFILE_ prefix. The write roles
	// are local-first by catalog order, so a profile that needs frontier prose
	// quality says so here rather than by reordering the catalog for everyone.
	Locality    string    `json:"locality,omitempty"`
	Authority   string    `json:"authority"`
	SourcePath  string    `json:"source_path,omitempty"`
	ContentHash string    `json:"content_hash,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type Sampler struct {
	Kind              string  `json:"kind"`
	K                 int     `json:"k"`
	Tau               float64 `json:"tau"`
	TemperatureStance string  `json:"temperature_stance"`
	MaxOutputTokens   int     `json:"max_output_tokens"`
}

type Constraints struct {
	MinWords       int      `json:"min_words,omitempty"`
	MaxWords       int      `json:"max_words,omitempty"`
	MinGrade       float64  `json:"min_grade,omitempty"`
	MaxGrade       float64  `json:"max_grade,omitempty"`
	BannedLexicon  []string `json:"banned_lexicon,omitempty"`
	RequiredFormat string   `json:"required_format,omitempty"`
}

type Budget struct {
	MaxOutputTokens int `json:"max_output_tokens"`
	MaxSessionCost  int `json:"max_session_cost_micros"`
}

type ContextPolicy struct {
	FullTextTokenBudget int  `json:"full_text_token_budget"`
	SummarizeBeyond     int  `json:"summarize_beyond"`
	AlwaysFullPrevious  bool `json:"always_full_previous"`
	// DeclaredContextCeiling is a profile-owned worst-case ceiling used by the
	// static feasibility check. It is not a provider context-window fact.
	DeclaredContextCeiling int `json:"declared_context_ceiling,omitempty"`
}

type ResolvedProfile struct {
	Profile         Profile `json:"profile"`
	Styles          []Style `json:"styles"`
	InstructionText string  `json:"instruction_text"`
}

type Provenance struct {
	ProfileVersion           string           `json:"profile_version"`
	StyleVersions            []string         `json:"style_versions"`
	Strategy                 string           `json:"strategy"`
	StrategyParameters       Sampler          `json:"strategy_parameters"`
	Provider                 string           `json:"provider"`
	ResolvedModelRef         string           `json:"resolved_model_ref"`
	GatewayRole              string           `json:"gateway_role"`
	TemperatureSent          float64          `json:"temperature_sent"`
	TemperatureSupport       string           `json:"temperature_support"`
	MaxOutputTokensEffective int              `json:"max_output_tokens_effective"`
	MaxOutputTokensSource    string           `json:"max_output_tokens_source"`
	InputTokens              int              `json:"input_tokens"`
	OutputTokens             int              `json:"output_tokens"`
	CostMicros               int64            `json:"cost_micros"`
	MachineGenerated         bool             `json:"machine_generated"`
	Disclosure               string           `json:"disclosure"`
	ContextSnapshot          *ContextSnapshot `json:"context_snapshot,omitempty"`
}

type Candidate struct {
	ID              string          `json:"id"`
	RoundID         string          `json:"round_id"`
	DerivedFrom     []string        `json:"derived_from,omitempty"`
	Text            string          `json:"text"`
	SetIndex        int             `json:"set_index"`
	Measurements    any             `json:"measurements"`
	SetMeasurements any             `json:"set_measurements"`
	Provenance      Provenance      `json:"provenance"`
	VerbalizedHint  *VerbalizedHint `json:"verbalized_hint,omitempty"`
	Eligibility     Eligibility     `json:"eligibility"`
	Committed       bool            `json:"committed"`
}

type VerbalizedHint struct {
	Ordinal    int  `json:"ordinal"`
	Calibrated bool `json:"calibrated"`
}
type Eligibility struct {
	Eligible bool   `json:"eligible"`
	Reason   string `json:"reason,omitempty"`
}

type Round struct {
	ID           string   `json:"id"`
	SessionID    string   `json:"session_id"`
	Strategy     Sampler  `json:"strategy"`
	CandidateIDs []string `json:"candidate_ids"`
	// SamplingKey is the round's effective generation identity. It is recorded
	// at birth because comparability is otherwise undecidable after the fact:
	// two set-diversity numbers mean nothing side by side unless the conditions
	// that produced them are known to match.
	SamplingKey     textmetrics.SamplingKey `json:"sampling_key"`
	SelectionSeed   int64                   `json:"selection_seed"`
	TotalCostMicros int64                   `json:"total_cost_micros"`
	NegativeContext NegativeContext         `json:"negative_context,omitempty"`
}
type NegativeContext struct {
	Pinned   []string `json:"pinned,omitempty"`
	Rejected []string `json:"rejected,omitempty"`
}
type Session struct {
	ID         string   `json:"id"`
	ProfileKey string   `json:"profile_key"`
	Query      string   `json:"query"`
	Status     string   `json:"status"`
	Pinned     []string `json:"pinned,omitempty"`
	Rejected   []string `json:"rejected,omitempty"`
	RoundIDs   []string `json:"round_ids"`
	BudgetUsed int64    `json:"budget_used_micros"`
}
type SelectionEvent struct {
	ID                     string    `json:"id"`
	SessionID              string    `json:"session_id"`
	CandidateID            string    `json:"candidate_id"`
	ConsideredCandidateIDs []string  `json:"considered_candidate_ids"`
	Measurements           any       `json:"measurements"`
	OutcomeRef             string    `json:"outcome_ref,omitempty"`
	CreatedAt              time.Time `json:"created_at"`
}

type Document struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	ProfileKey    string    `json:"profile_key"`
	StyleKey      string    `json:"style_key"`
	OutlineID     string    `json:"outline_id,omitempty"`
	SectionIDs    []string  `json:"section_ids"`
	Status        string    `json:"status"`
	AssembledText string    `json:"assembled_text,omitempty"`
	Coherence     any       `json:"coherence,omitempty"`
	Sections      []Section `json:"sections,omitempty"`
}
type Section struct {
	ID                   string          `json:"id"`
	DocumentID           string          `json:"document_id"`
	Position             int             `json:"position"`
	Intent               string          `json:"intent"`
	ProfileKey           string          `json:"profile_key,omitempty"`
	SessionID            string          `json:"session_id,omitempty"`
	CommittedCandidateID string          `json:"committed_candidate_id,omitempty"`
	Context              ContextSnapshot `json:"context_snapshot"`
}
type ContextSnapshot struct {
	OutlineRef            string   `json:"outline_ref,omitempty"`
	PriorSectionRefs      []string `json:"prior_section_refs,omitempty"`
	FollowingIntents      []string `json:"following_intents,omitempty"`
	SummarizedSectionRefs []string `json:"summarized_section_refs,omitempty"`
	EstimatedTokens       int      `json:"estimated_tokens"`
}

type Declaration struct {
	Path          string          `json:"path"`
	SchemaVersion string          `json:"schema_version"`
	Kind          string          `json:"kind"`
	Key           string          `json:"key"`
	CreatedBy     string          `json:"created_by"`
	ContentHash   string          `json:"content_hash"`
	Status        string          `json:"status"`
	Error         string          `json:"error,omitempty"`
	Record        json.RawMessage `json:"record,omitempty"`
}

type RegistryKind struct {
	Kind            string         `json:"kind"`
	Description     string         `json:"description"`
	ParameterSchema map[string]any `json:"parameter_schema"`
}
type Registry struct {
	Samplers   []RegistryKind `json:"samplers"`
	Policies   []RegistryKind `json:"policies"`
	Metrics    []RegistryKind `json:"metrics"`
	Transforms []RegistryKind `json:"transforms"`
}

type GenerateRequest struct {
	ProfileKey        string          `json:"profile_key"`
	Query             string          `json:"query"`
	IncludeCandidates bool            `json:"include_candidates"`
	SessionID         string          `json:"session_id,omitempty"`
	Negative          NegativeContext `json:"negative_context,omitempty"`
}
type GenerateResponse struct {
	Session            Session          `json:"session"`
	Round              Round            `json:"round"`
	Selected           *Candidate       `json:"selected,omitempty"`
	Candidates         []Candidate      `json:"candidates,omitempty"`
	SelectedCandidates []Candidate      `json:"selected_candidates,omitempty"`
	Degraded           *DegradedOutcome `json:"degraded,omitempty"`
}

type DegradedOutcome struct {
	Kind                     string `json:"kind"`
	Reason                   string `json:"reason"`
	RequestedCandidates      int    `json:"requested_candidates"`
	ReceivedCandidates       int    `json:"received_candidates"`
	MaxOutputTokensEffective int    `json:"max_output_tokens_effective"`
	MaxOutputTokensSource    string `json:"max_output_tokens_source"`
}

type GatewayRequest struct {
	Role              string          `json:"role"`
	Instruction       string          `json:"instruction"`
	Query             string          `json:"query"`
	Strategy          string          `json:"strategy"`
	K                 int             `json:"k"`
	Tau               float64         `json:"tau"`
	MaxOutputTokens   int             `json:"max_output_tokens"`
	TemperatureStance string          `json:"temperature_stance"`
	Locality          string          `json:"locality,omitempty"`
	Negative          NegativeContext `json:"negative_context,omitempty"`
}
type GatewayCandidate struct {
	Text               string  `json:"text"`
	Provider           string  `json:"provider"`
	Model              string  `json:"model"`
	Temperature        float64 `json:"temperature"`
	TemperatureSupport string  `json:"temperature_support"`
	InputTokens        int     `json:"input_tokens"`
	OutputTokens       int     `json:"output_tokens"`
	CostMicros         int64   `json:"cost_micros"`
	// MaxOutputTokensEffective and its source are the gateway's report of the
	// cap it actually imposed, which is not always the cap the profile asked
	// for: a role policy can supply one, and the gateway can impose none.
	MaxOutputTokensEffective int    `json:"max_output_tokens_effective"`
	MaxOutputTokensSource    string `json:"max_output_tokens_source"`
	// HintOrdinal is the candidate's rank by the probability the model verbalized
	// for it, 1 being the highest, scoped to this round only. It is zero for any
	// strategy that elicits no probability. The probability itself is deliberately
	// not carried: verbalized confidence is uncalibrated and sparse, so a stored
	// float would invite the averaging and thresholding OT-P0-009 forbids, while
	// a within-round rank cannot be compared across rounds by accident.
	HintOrdinal int `json:"hint_ordinal"`
}
