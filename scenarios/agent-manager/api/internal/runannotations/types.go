package runannotations

import "time"

type LabelView struct {
	ID             string        `json:"id"`
	Group          string        `json:"group"`
	Description    string        `json:"description"`
	Scope          Scope         `json:"scope"`
	Method         Method        `json:"method"`
	Optional       bool          `json:"optional"`
	DefinitionHash string        `json:"definition_hash"`
	Gate           Gate          `json:"gate"`
	GateState      string        `json:"gate_state"`
	Proven         bool          `json:"proven"`
	Search         Search        `json:"search"`
	Friction       Friction      `json:"friction"`
	Compatibility  Compatibility `json:"compatibility"`
	Coverage       StateCounts   `json:"coverage"`
	HasData        bool          `json:"has_data"`
}

// GateStatus is the contract gate object: measured precision/recall/gold plus
// the pack's targets and an explicit state.
type GateStatus struct {
	State           string  `json:"state"`
	Precision       float64 `json:"precision"`
	Recall          float64 `json:"recall"`
	Gold            int     `json:"gold"`
	MinGold         int     `json:"min_gold"`
	Reason          string  `json:"reason"`
	TargetPrecision float64 `json:"target_precision"`
	TargetRecall    float64 `json:"target_recall"`
}

// CoverageLabel is one contract coverage row. Rate is set only for a proven
// gate; baseline/ratio are omitted until a random-sample baseline is measured.
type CoverageLabel struct {
	ID        string     `json:"id"`
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
	Notes         []string        `json:"notes,omitempty"`
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
	Notes    []string         `json:"notes,omitempty"`
}

// AnnotateOptions bounds one run's evaluation.
type AnnotateOptions struct {
	Force           bool
	IncludeOptional bool
	Labels          []string
	MaxLabels       int
	DryRun          bool
}

// EvaluateOptions bounds a batch.
type EvaluateOptions struct {
	Since           time.Time
	Until           time.Time
	Limit           int
	RunIDs          []string
	Labels          []string
	IncludeOptional bool
	Force           bool
	DryRun          bool
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
	Remaining   int             `json:"remaining"`
	Truncated   bool            `json:"truncated"`
	Observed    int             `json:"observed"`
	Unavailable int             `json:"unavailable"`
	Coverage    float64         `json:"coverage"`
	Labels      []CoverageLabel `json:"labels"`
	Runs        []RunView       `json:"runs,omitempty"`
	Errors      []RunError      `json:"errors,omitempty"`
	Counts      StateCounts     `json:"counts"`
}
