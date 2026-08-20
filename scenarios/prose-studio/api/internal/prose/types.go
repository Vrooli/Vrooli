package prose

import (
	"encoding/json"
	"time"

	"github.com/vrooli/textmetrics"
)

type Style struct {
	Key              string             `json:"key"`
	Version          int                `json:"version"`
	Parent           string             `json:"parent,omitempty"`
	Exemplars        []string           `json:"exemplars,omitempty"`
	Directives       []string           `json:"directives,omitempty"`
	AntiPatterns     []string           `json:"anti_patterns,omitempty"`
	Lexicon          []string           `json:"lexicon,omitempty"`
	Targets          map[string]float64 `json:"targets,omitempty"`
	TargetDirections map[string]string  `json:"target_directions,omitempty"`
	AxisDefaults     map[string]string  `json:"axis_defaults,omitempty"`
	Authority        string             `json:"authority"`
	SourcePath       string             `json:"source_path,omitempty"`
	ContentHash      string             `json:"content_hash,omitempty"`
	Frozen           bool               `json:"frozen"`
	Status           string             `json:"status"`
	CreatedAt        time.Time          `json:"created_at"`
}

type Profile struct {
	Key               string              `json:"key"`
	Version           int                 `json:"version"`
	Parent            string              `json:"parent,omitempty"`
	StyleRefs         []string            `json:"style_refs,omitempty"`
	Sampler           Sampler             `json:"sampler"`
	Constraints       Constraints         `json:"constraints"`
	SelectionPolicy   string              `json:"selection_policy"`
	SelectionParams   map[string]float64  `json:"selection_params,omitempty"`
	MeasurementTiers  []string            `json:"measurement_tiers,omitempty"`
	Budget            Budget              `json:"budget"`
	ContextPolicy     ContextPolicy       `json:"context_policy"`
	OutlineProfileKey string              `json:"outline_profile_key,omitempty"`
	SectionProfileKey string              `json:"section_profile_key,omitempty"`
	AxisSpaceKey      string              `json:"axis_space_key,omitempty"`
	Coherence         CoherenceThresholds `json:"coherence_thresholds,omitempty"`
	Composition       CompositionPolicy   `json:"composition,omitempty"`
	GatewayRole       string              `json:"gateway_role"`
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
	// MinSectionNovelty is the share of a section's content terms that must not
	// already appear in the section before it. It is the one constraint here
	// that a candidate cannot be judged against on its own: a passage is only
	// redundant relative to something, so it is evaluated against the committed
	// prior text a continuation carries and is skipped where there is none.
	//
	// It exists because the length, grade, and format constraints beside it are
	// all satisfiable by a section that re-argues its predecessor in fresh
	// words, and that is the failure this scenario's own long-form output was
	// measured to have.
	MinSectionNovelty float64 `json:"min_section_novelty,omitempty"`
	// MinArtifacts is how many distinct declared artifacts a passage must
	// quote. It is a policy about how concrete prose has to be, which is why it
	// lives on the profile, while the artifacts themselves arrive on the
	// request: what is available to quote is editorial content the consumer
	// owns, and this scenario decides no brief.
	//
	// Without it, "do not invent implementation details" is satisfiable only by
	// writing nothing specific, and an accuracy rule silently becomes a
	// vagueness rule.
	MinArtifacts int `json:"min_artifacts,omitempty"`
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
	Transform       *Transform      `json:"transform,omitempty"`
	AxisCell        string          `json:"axis_cell,omitempty"`
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
	SamplingKey             textmetrics.SamplingKey `json:"sampling_key"`
	SelectionSeed           int64                   `json:"selection_seed"`
	TotalCostMicros         int64                   `json:"total_cost_micros"`
	NegativeContext         NegativeContext         `json:"negative_context,omitempty"`
	LexicalSetMeasurements  any                     `json:"lexical_set_measurements,omitempty"`
	SemanticSetMeasurements any                     `json:"semantic_set_measurements,omitempty"`
	MeasurementBasis        string                  `json:"measurement_basis,omitempty"`
	MeasurementFallback     string                  `json:"measurement_fallback,omitempty"`
}
type NegativeContext struct {
	// These values are prose resolved from session candidate identifiers before
	// they reach the gateway. Identifiers remain in Session for audit linkage.
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
	DistanceBasis          string    `json:"distance_basis,omitempty"`
	OutcomeRef             string    `json:"outcome_ref,omitempty"`
	CreatedAt              time.Time `json:"created_at"`
}

type Document struct {
	ID                string           `json:"id"`
	Title             string           `json:"title"`
	ProfileKey        string           `json:"profile_key"`
	OutlineProfileKey string           `json:"outline_profile_key,omitempty"`
	StyleKey          string           `json:"style_key"`
	OutlineID         string           `json:"outline_id,omitempty"`
	OutlineText       string           `json:"outline_text,omitempty"`
	Outline           []OutlineSection `json:"outline,omitempty"`
	// Artifacts are the literals every section of this document may quote. They
	// belong to the document rather than to a section because a command named
	// once in the opening is still available to the conclusion.
	Artifacts             []string `json:"artifacts,omitempty"`
	OutlineProfileVersion string   `json:"outline_profile_version,omitempty"`
	// SectionCount overrides the profile's composition policy for this document.
	SectionCount          int       `json:"section_count,omitempty"`
	SectionProfileVersion string    `json:"section_profile_version,omitempty"`
	SectionIDs            []string  `json:"section_ids"`
	Status                string    `json:"status"`
	AssembledText         string    `json:"assembled_text,omitempty"`
	Coherence             any       `json:"coherence,omitempty"`
	Sections              []Section `json:"sections,omitempty"`
	// CreatedAt exists so a reader can ask for the most recent documents without
	// already knowing an identifier nobody told them.
	CreatedAt time.Time `json:"created_at,omitempty"`
	// Provenance for the document as a whole. A long-form run costs one outline
	// call plus one call per section plus any repair, and none of that is
	// visible in a single round's accounting; a caller reading a standalone
	// generate call's cost was reading a different request entirely.
	Provenance DocumentProvenance `json:"document_provenance,omitempty"`
}

// DocumentSummary is the listing shape: enough to choose which document to
// read, without carrying the prose of every document into the response.
type DocumentSummary struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	ProfileKey      string    `json:"profile_key,omitempty"`
	Status          string    `json:"status"`
	WordCount       int       `json:"word_count"`
	SectionCount    int       `json:"section_count"`
	TotalCostMicros int64     `json:"total_cost_micros"`
	CreatedAt       time.Time `json:"created_at,omitempty"`
	Coherent        bool      `json:"coherent"`
}

// DocumentProvenance aggregates what an assembled document actually consumed.
type DocumentProvenance struct {
	SectionCount    int      `json:"section_count"`
	WordCount       int      `json:"word_count"`
	TotalCostMicros int64    `json:"total_cost_micros"`
	InputTokens     int      `json:"input_tokens"`
	OutputTokens    int      `json:"output_tokens"`
	Providers       []string `json:"providers,omitempty"`
	Models          []string `json:"models,omitempty"`
}

type OutlineSection struct {
	Intent      string `json:"intent"`
	Summary     string `json:"summary"`
	TargetWords int    `json:"target_words"`
}

type CoherenceThresholds struct {
	MaxCrossSectionRepetition float64 `json:"max_cross_section_repetition,omitempty"`
	MaxStyleDrift             float64 `json:"max_style_drift,omitempty"`
	// MinSectionNovelty is the per-section floor enforced at generation time,
	// as distinct from the whole-document repetition measures beside it. Those
	// report after assembly, when the only remaining move is repair; this one
	// makes a section that added nothing ineligible while a reroll is still
	// cheap.
	MinSectionNovelty float64 `json:"min_section_novelty,omitempty"`
	// MaxSemanticSectionRepetition bounds mean pairwise cosine similarity over
	// section embeddings. It exists because the lexical measure passes text that
	// paraphrases itself: three sections restating one argument in different
	// words score as low lexical overlap while carrying near-identical meaning.
	MaxSemanticSectionRepetition float64 `json:"max_semantic_section_repetition,omitempty"`
}

// CompositionPolicy is the long-form half of a profile. Composition is a
// different problem from candidate variation: a document has a shape, a length
// band per section, and a redundancy ceiling, none of which the sampler or the
// candidate-level constraints can express.
type CompositionPolicy struct {
	// SectionCount pins the outline length. Zero derives it from the word
	// budget and TargetSectionWords instead, because the right number of
	// sections is a function of how long the article is.
	SectionCount int `json:"section_count,omitempty"`
	// TargetSectionWords is the words-per-section the derived count aims for.
	TargetSectionWords int `json:"target_section_words,omitempty"`
	// MinSections and MaxSections bound the derived count only.
	MinSections int `json:"min_sections,omitempty"`
	MaxSections int `json:"max_sections,omitempty"`
	// SectionWordTolerance is the fractional band around a section's outline
	// target that the section gate accepts, e.g. 0.25 admits 75%-125%. A band
	// is required in both directions: a ceiling pinned exactly at the target
	// gives the model only one direction to miss in, and it always misses low.
	SectionWordTolerance float64 `json:"section_word_tolerance,omitempty"`
	// SectionFormat is the required_format applied to each section. Empty means
	// paragraph. A long-form consumer that needs headings, lists, or code sets
	// rich_prose here rather than having paragraph forced on it.
	SectionFormat string `json:"section_format,omitempty"`
	// MaxRepairRounds bounds regeneration after a failed coherence verdict.
	// Zero disables repair and preserves the report-only behaviour.
	MaxRepairRounds int `json:"max_repair_rounds,omitempty"`
	// SectionSamplerKind and SectionCandidates decide how a section's candidate
	// set is drawn. They are separate from the profile's sampler because the two
	// levels want different things: an outline wants a distribution over
	// framings, which is what verbalized sampling elicits, while a section wants
	// the best continuation of a framing already chosen. Drawing sections
	// through the verbalized envelope also buys a failure mode composition has
	// no use for, since one malformed entry in the set fails the whole document.
	// GatewayAttempts bounds how many times one long-form step re-asks the
	// provider before the document fails. Long-form is a multi-request
	// composition -- one outline call, one call per section, plus summarisation
	// and repair -- and at that arity a single malformed provider response is an
	// ordinary event rather than an exceptional one. Zero uses the default.
	GatewayAttempts int `json:"gateway_attempts,omitempty"`
	// SectionMaxOutputTokens overrides the derived per-section output budget.
	SectionMaxOutputTokens int    `json:"section_max_output_tokens,omitempty"`
	SectionSamplerKind     string `json:"section_sampler_kind,omitempty"`
	SectionCandidates      int    `json:"section_candidates,omitempty"`
	// SectionSelectionPolicy overrides the continuation default for sections.
	// It is separate from the profile's SelectionPolicy because those two
	// choices answer different questions: which candidate is most unlike its
	// siblings, and which candidate best continues this document.
	SectionSelectionPolicy string `json:"section_selection_policy,omitempty"`
}

type Transform struct {
	Operation       string         `json:"operation"`
	Parameters      map[string]any `json:"parameters,omitempty"`
	SourceCandidate string         `json:"source_candidate"`
	GatewayRole     string         `json:"gateway_role"`
	CreatedAt       time.Time      `json:"created_at"`
}

type AxisSpace struct {
	Key       string `json:"key"`
	Axes      []Axis `json:"axes"`
	Authority string `json:"authority,omitempty"`
}

type Axis struct {
	Name     string   `json:"name"`
	Variants []string `json:"variants"`
}

type AxisCell struct {
	Key      string            `json:"key"`
	Variants map[string]string `json:"variants"`
}

type CellCoverage struct {
	Planned []AxisCell `json:"planned"`
	Covered []string   `json:"covered"`
	Missed  []string   `json:"missed"`
}

// CompositeGeneration is the auditable result of sampling one candidate set
// per planned axis cell. Candidates retain their ordinary round provenance;
// the aggregate only adds the cell coverage needed to prove the grid was
// actually exercised.
type CompositeGeneration struct {
	Candidates []Candidate  `json:"candidates"`
	Rounds     []Round      `json:"rounds"`
	Sessions   []Session    `json:"sessions"`
	Coverage   CellCoverage `json:"coverage"`
}
type Section struct {
	ID                   string          `json:"id"`
	DocumentID           string          `json:"document_id"`
	Position             int             `json:"position"`
	Intent               string          `json:"intent"`
	Summary              string          `json:"summary,omitempty"`
	TargetWords          int             `json:"target_words,omitempty"`
	ProfileKey           string          `json:"profile_key,omitempty"`
	SessionID            string          `json:"session_id,omitempty"`
	CommittedCandidateID string          `json:"committed_candidate_id,omitempty"`
	Context              ContextSnapshot `json:"context_snapshot"`
}
type ContextSnapshot struct {
	OutlineRef            string   `json:"outline_ref,omitempty"`
	OutlineText           string   `json:"outline_text,omitempty"`
	PriorSectionRefs      []string `json:"prior_section_refs,omitempty"`
	FullTextSectionRefs   []string `json:"full_text_section_refs,omitempty"`
	FollowingIntents      []string `json:"following_intents,omitempty"`
	SummarizedSectionRefs []string `json:"summarized_section_refs,omitempty"`
	// DegradedSummaryRefs names sections whose summary came from a deterministic
	// excerpt because the summarising call could not be completed. The snapshot
	// records what it carried, so a locally derived summary must be
	// distinguishable from a model-written one rather than silently equivalent.
	DegradedSummaryRefs []string          `json:"degraded_summary_refs,omitempty"`
	SectionSummaries    map[string]string `json:"section_summaries,omitempty"`
	EstimatedTokens     int               `json:"estimated_tokens"`
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
	SchemaJSON        string          `json:"schema_json,omitempty"`
	IncludeCandidates bool            `json:"include_candidates"`
	SessionID         string          `json:"session_id,omitempty"`
	Negative          NegativeContext `json:"negative_context,omitempty"`
	// Selection carries the already-committed surroundings a continuation
	// policy needs. It is nil for instance-level generation, where a candidate
	// set has no position in anything.
	Selection *SelectionContext `json:"selection_context,omitempty"`
	// Artifacts are the literals this request is permitted to quote: commands,
	// paths, identifiers, figures. They are carried per request rather than on
	// the profile because they are the caller's brief, and a style record that
	// pinned them would make one voice usable for exactly one subject.
	Artifacts []string `json:"artifacts,omitempty"`
}

// SelectionContext is what a policy needs in order to choose a candidate for a
// position in a document rather than for its own sake. It carries prose, not
// record identifiers, so selection never becomes a second prompt seam.
type SelectionContext struct {
	PriorText   []string `json:"prior_text,omitempty"`
	TargetWords int      `json:"target_words,omitempty"`
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
	SchemaJSON        string          `json:"schema_json,omitempty"`
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
