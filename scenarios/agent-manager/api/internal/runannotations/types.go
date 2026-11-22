package runannotations

import "time"

type LabelView struct {
	ID             string `json:"id"`
	Group          string `json:"group"`
	Description    string `json:"description"`
	Scope          Scope  `json:"scope"`
	Method         Method `json:"method"`
	Optional       bool   `json:"optional"`
	Role           string `json:"role"`
	EscalateRole   string `json:"escalate_role,omitempty"`
	DefinitionHash string `json:"definition_hash"`
	Gate           Gate   `json:"gate"`
	// GateState and Proven summarize the label across the models it was
	// observed with (or measured for): proven when any model's hold-out clears
	// the targets. GateStatus is that summary's measurement; the per-model
	// measurements are in Gate.Measurements.
	GateState     string        `json:"gate_state"`
	Proven        bool          `json:"proven"`
	GateStatus    GateStatus    `json:"gate_status"`
	Search        Search        `json:"search"`
	Friction      Friction      `json:"friction"`
	Compatibility Compatibility `json:"compatibility"`
	Coverage      StateCounts   `json:"coverage"`
	HasData       bool          `json:"has_data"`
}

// CoverageLabel is one contract coverage row. Rate is set only for a proven
// gate; baseline/ratio are omitted until a random-sample baseline is measured.
type CoverageLabel struct {
	ID string `json:"id"`
	// Scope is what the label is about: run, message or episode.
	Scope     Scope      `json:"scope"`
	Group     string     `json:"group"`
	State     string     `json:"state"`
	Gate      GateStatus `json:"gate"`
	Observed  int        `json:"observed"`
	Eligible  int        `json:"eligible"`
	Evaluated int        `json:"evaluated"`
	// Positives is the count of observations at or above the label's search/
	// friction threshold. Observed is the count with any observation; a label
	// can be observed without crossing the threshold.
	Positives      int      `json:"positives,omitempty"`
	Rate           *float64 `json:"rate"`
	Baseline       *float64 `json:"baseline"`
	Ratio          *float64 `json:"ratio"`
	BaselineState  string   `json:"baseline_state,omitempty"`
	SearchFacet    string   `json:"search_facet"`
	MinProbability *float64 `json:"min_probability,omitempty"`
	Friction       Friction `json:"friction"`
	// Models is the label's observations split by the (model, resource role) that
	// produced them, each with its own proof. Rate counts only ProvenObserved.
	Models          []ModelCoverage `json:"models,omitempty"`
	ProvenObserved  int             `json:"proven_observed,omitempty"`
	ProvenPositives int             `json:"proven_positives,omitempty"`
	// Population is set when the label declares whose text it is about; Eligible
	// is then that population's size rather than the rows evaluated.
	Population *Population `json:"population,omitempty"`
}

// ModelCoverage is one model's share of a label's observations and the proof
// that model's answers carry.
type ModelCoverage struct {
	Model        string     `json:"model"`
	ResourceRole string     `json:"resource_role"`
	Observed     int        `json:"observed"`
	Positives    int        `json:"positives"`
	Proven       bool       `json:"proven"`
	Gate         GateStatus `json:"gate"`
}

// Population is the denominator a label's coverage is reported against.
type Population struct {
	Basis    string `json:"basis"`
	Unit     string `json:"unit"`
	Eligible int    `json:"eligible"`
}

// WindowRange is the RFC3339 coverage window. An unbounded side is empty.
type WindowRange struct {
	Since string `json:"since,omitempty"`
	Until string `json:"until,omitempty"`
}

// PackRef identifies the reporting pack.
type PackRef struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Weighting declares how the population was selected and weighted. The
// per-label rates are the authoritative sampling story; SampleRate is the most
// conservative (smallest) rate in the pack.
type Weighting struct {
	Mode       string             `json:"mode"`
	SampleRate float64            `json:"sample_rate"`
	ByLabel    map[string]float64 `json:"by_label,omitempty"`
}

// CoverageResponse is the contract coverage surface. An empty store returns 200
// with labels present, state not_observed, and coverage 0.
type CoverageResponse struct {
	Window        WindowRange     `json:"window"`
	Pack          PackRef         `json:"pack"`
	EligibleRuns  int             `json:"eligible_runs"`
	ObservedRuns  int             `json:"observed_runs"`
	Coverage      float64         `json:"coverage"`
	CoverageBasis string          `json:"coverage_basis"`
	Weighting     Weighting       `json:"weighting"`
	Labels        []CoverageLabel `json:"labels"`
	Summary       StateCounts     `json:"summary"`
	// Provenance splits the annotated runs by whose operator text they carried:
	// a person, or a workflow brief or harness. Affect labels are read against
	// the human share.
	Provenance *ProvenanceSummary `json:"provenance,omitempty"`
	Notes      []string           `json:"notes,omitempty"`
}

// AnnotationView is the contract label result plus gate state.
type AnnotationView struct {
	ID               string     `json:"id"`
	State            State      `json:"state"`
	Method           string     `json:"method"`
	EvaluatorMethod  string     `json:"evaluator_method,omitempty"`
	Value            Value      `json:"value"`
	Probability      *float64   `json:"probability,omitempty"`
	Confidence       *float64   `json:"confidence,omitempty"`
	Model            string     `json:"model,omitempty"`
	ResourceRole     string     `json:"resource_role,omitempty"`
	RequestedRole    string     `json:"requested_role,omitempty"`
	Provider         string     `json:"provider,omitempty"`
	DefinitionHash   string     `json:"definition_hash"`
	ObservedAt       time.Time  `json:"observed_at"`
	EvidenceEventIDs []string   `json:"evidence_event_ids,omitempty"`
	Reason           string     `json:"reason,omitempty"`
	Gate             GateStatus `json:"gate"`
	GateState        string     `json:"gate_state"`
	Proven           bool       `json:"proven"`
	// Friction is the label's declared selector. A client scoring stored
	// observations needs the same polarity rule the service uses, so a signed
	// score is not read as a friction positive just because its level was
	// confident.
	Friction Friction `json:"friction"`
	// FrictionProbability is the calibrated mass in the declared friction
	// region. For a score label it replaces the winning level's probability,
	// which is structurally small on a multi-level scale. It is omitted when
	// the observation carries no level distribution.
	FrictionProbability *float64 `json:"friction_probability,omitempty"`
	// CalibratedProbability is the stored vector's score (noul probability, top
	// class, or friction-region mass) read at the temperature measured for this
	// observation's model; Temperature is that temperature when it is not 1.
	// The stored Probability and Value are never rewritten.
	CalibratedProbability *float64 `json:"calibrated_probability,omitempty"`
	Temperature           float64  `json:"temperature,omitempty"`
	// FrictionPositive says whether this observation is a friction event under
	// its own model's calibration and threshold; absent for a label that is not
	// friction or an observation with nothing to score.
	FrictionPositive *bool      `json:"friction_positive,omitempty"`
	OccurredAt       *time.Time `json:"occurred_at,omitempty"`
}

// SkippedLabel explains why a label produced no row for one run.
type SkippedLabel struct {
	LabelID string `json:"label_id"`
	Reason  string `json:"reason"`
}

// RunCoverage is the compact coverage block a run response leads with.
type RunCoverage struct {
	Observed int `json:"observed"`
	Labels   int `json:"labels"`
}

// RunView is one run's latest annotations.
type RunView struct {
	RunID    string           `json:"run_id"`
	Pack     string           `json:"pack"`
	Version  string           `json:"version"`
	PackHash string           `json:"pack_hash"`
	State    string           `json:"state"`
	Coverage RunCoverage      `json:"coverage"`
	Labels   []AnnotationView `json:"labels"`
	Counts   StateCounts      `json:"counts"`
	Missing  []string         `json:"missing,omitempty"`
	Skipped  []SkippedLabel   `json:"skipped,omitempty"`
	// Messages are the message-scope labels of the run's human messages, oldest
	// message first.
	Messages []MessageView `json:"messages,omitempty"`
	Notes    []string      `json:"notes,omitempty"`
}

// AnnotateOptions bounds one run's evaluation.
type AnnotateOptions struct {
	// ForceLabels names the labels whose cached answers are ignored and asked
	// again. Force is always scoped: a composite is recomputed from stored
	// answers on its own, and forcing it never re-asks the messages it reads
	// unless their labels are named too.
	ForceLabels     []string
	IncludeOptional bool
	Labels          []string
	MaxLabels       int
	DryRun          bool
	// PinResourceRole restricts every decision to one declared candidate and
	// stores the answers in the trials table, so a bake-off measures a model
	// without touching production observations. Role is the decision role walked
	// for it; empty uses runtime.bakeoff_role, then the label's own role.
	PinResourceRole string
	Role            string
	// MessageIDs, when set, restricts message-scope labels to the named human
	// messages (a gold set judges particular messages); empty answers the
	// newest messages of each run up to the pack's bound.
	MessageIDs []string
}

// EvaluateOptions bounds a batch.
type EvaluateOptions struct {
	Since           time.Time
	Until           time.Time
	Limit           int
	RunIDs          []string
	Labels          []string
	IncludeOptional bool
	ForceLabels     []string
	DryRun          bool
	PinResourceRole string
	Role            string
	// MessageIDs restricts message-scope labels to the named messages.
	MessageIDs []string
	// TimeBudget optionally overrides the pack's per-pass wall-clock budget.
	// Zero uses the pack value; a negative value disables the budget (used by
	// single-run callers and tests that must not be time sensitive).
	TimeBudget time.Duration
}

// RunError is one run's failure inside a batch; other runs still complete.
type RunError struct {
	RunID  string `json:"run_id"`
	Reason string `json:"reason"`
}

// EvaluateResult is the contract batch result. Runs/Errors/Counts are additive
// diagnostic fields; the contract fields are Window, Submitted, Observed,
// Unavailable, Coverage and Labels.
type EvaluateResult struct {
	Window    WindowRange `json:"window"`
	Submitted int         `json:"submitted"`
	// Remaining counts selected runs a bounded pass did not reach. Truncated
	// says the pass stopped at its wall-clock budget rather than exhausting
	// the selection, so the caller can continue with another pass.
	Remaining int  `json:"remaining"`
	Truncated bool `json:"truncated"`
	// RemainingRunIDs lists explicitly named runs the pass did not reach, so the
	// caller can continue until every one is answered.
	RemainingRunIDs []string `json:"remaining_run_ids,omitempty"`
	// Pinned is the resource role a bake-off pass was restricted to.
	Pinned string `json:"pinned,omitempty"`
	// Escalation reports what escalation spent and saved in this pass.
	Escalation  *EscalationStats `json:"escalation,omitempty"`
	Observed    int              `json:"observed"`
	Unavailable int              `json:"unavailable"`
	Coverage    float64          `json:"coverage"`
	Labels      []CoverageLabel  `json:"labels"`
	Runs        []RunView        `json:"runs,omitempty"`
	Errors      []RunError       `json:"errors,omitempty"`
	Counts      StateCounts      `json:"counts"`
}
