// Package runannotations implements Agent Manager's data-driven run annotation
// substrate: versioned label packs declare what to look for, and pluggable
// evaluators (deterministic CEL, existing operator signals, decision models,
// composites, deferred extraction) produce observations.
//
// A label's semantic definition is hashed per label. Pack version is reporting
// only. An observation is keyed by scope, label, definition hash and model, so
// adding a label or changing one label never invalidates the others. A declared
// compatible predecessor hash may be reused and is recorded with state
// "compatible"; a breaking change is never reused.
package runannotations

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
)

// Scope is what a label describes.
type Scope string

const (
	ScopeMessage Scope = "message"
	ScopeEpisode Scope = "episode"
	ScopeRun     Scope = "run"
)

// Method names the evaluator kind. Adding a label with an existing method is
// data; adding a new evaluator kind is code.
type Method string

const (
	MethodDeterministic Method = "deterministic"
	MethodSignal        Method = "signal"
	MethodDecision      Method = "decision"
	MethodExtract       Method = "extract"
	MethodComposite     Method = "composite"
)

// QuestionType mirrors ai-gateway's decision question types.
type QuestionType string

const (
	QuestionChoice QuestionType = "choice"
	QuestionScore  QuestionType = "score"
	QuestionNoul   QuestionType = "noul"
)

// DefaultRuntime values bound inference and reading when a pack omits them.
const (
	DefaultDecisionRole     = "decide.accurate"
	DefaultBatchStates      = 32
	DefaultBatchQuestions   = 64
	DefaultBackfillHorizon  = 30
	DefaultRunsPerPass      = 50
	DefaultDecisionPerPass  = 64
	DefaultMaxRunAnnotation = 64
	DefaultDigestLimit      = 1800
	// DefaultSecondsPerPass bounds one synchronous evaluation pass. It sits
	// below a typical gateway/client timeout so a large backfill returns
	// persisted progress with a clear "run again" signal instead of a
	// cancelled request that saves nothing.
	DefaultSecondsPerPass    = 90
	maximumCandidateExprSize = 4000
)

// PopulationHumanOperator is the coverage population of runs and messages a
// person wrote, as opposed to workflow briefs and harness text.
const PopulationHumanOperator = "human_operator"

// Pack is one versioned vocabulary of labels.
type Pack struct {
	Name    string  `json:"name"`
	Version string  `json:"version"`
	Runtime Runtime `json:"runtime"`
	Labels  []Label `json:"labels"`
}

// Runtime declares evaluator roles, batching, horizons and budgets. It is
// reporting plus admission policy; it never changes a label's definition hash.
type Runtime struct {
	Mode string `json:"mode"`
	// DefaultRole and EscalateRole are the decision roles a label uses unless it
	// declares its own role or escalate_role. The scheduler derives its model
	// affinity from the roles labels actually resolve to, never from a separate
	// pack-wide setting that could drift from them.
	DefaultRole  string     `json:"default_role"`
	EscalateRole string     `json:"escalate_role"`
	Escalation   Escalation `json:"escalation"`
	// BakeoffRole is the decision role a pinned (bake-off) evaluation walks when
	// the caller names none: a role whose candidates include the models under
	// test. Empty means the label's own role.
	BakeoffRole string `json:"bakeoff_role,omitempty"`
	// GoldRoot is the private directory gold files live in: reference rows
	// carry the operator's own words, so they never belong in a tracked path.
	// Empty selects "gold" under the scenario's durable state root (beside its
	// database); a relative value resolves against that root, an absolute one
	// is used as written. gate.measurements[].gold_file and the CLI's --gold,
	// --fit-from and --runs-from resolve against it.
	GoldRoot            string  `json:"gold_root,omitempty"`
	Batch               Batch   `json:"batch"`
	BackfillHorizonDays int     `json:"backfill_horizon_days"`
	Budgets             Budgets `json:"budgets"`
	// Provenance declares which operator text is a person speaking and which is
	// a workflow brief or harness text. It is entirely data: an empty policy
	// treats every operator-source message as a person's. A change to it makes
	// every stored provenance row stale (see Provenance.Hash).
	Provenance Provenance `json:"provenance"`
	// DigestChars bounds the state JSON sent to a decision model. It is pack
	// data because the right value follows the model's context window, not the
	// engine: a large local model can read more evidence than a small one.
	// Zero selects DefaultDigestLimit. An overflow is still retried with a
	// smaller digest, so this is a quality/efficiency knob, not a hard limit.
	DigestChars int `json:"digest_chars,omitempty"`
}

// Escalation raises low-confidence states to a stronger role once. It never
// re-asks the same model: when the escalation role resolves to the model that
// already answered, the call is skipped and that fact is remembered for
// SameModelTTL.
type Escalation struct {
	Enabled       bool    `json:"enabled"`
	MinConfidence float64 `json:"min_confidence"`
	MaxLabels     int     `json:"max_labels"`
	// SameModelTTL is a Go duration ("10m"). How long a learned "this escalation
	// role answers with the same model as the first pass" stays trusted before
	// the engine asks the gateway again, so a model change is noticed.
	SameModelTTL string `json:"same_model_ttl,omitempty"`
}

// DefaultSameModelTTL is how long a futile escalation is remembered when the
// pack declares escalation without a TTL.
const DefaultSameModelTTL = 10 * time.Minute

// SameModelDuration resolves SameModelTTL. A pack validates it at load, so an
// unparseable value cannot reach here; it falls back to the default.
func (e Escalation) SameModelDuration() time.Duration {
	if parsed, err := time.ParseDuration(strings.TrimSpace(e.SameModelTTL)); err == nil && parsed > 0 {
		return parsed
	}
	return DefaultSameModelTTL
}

// Provenance classifies operator-source text. A message that a person did not
// write must never be read as that person's mood.
type Provenance struct {
	// BriefWorkloads are workload kinds whose operator-source text is always a
	// brief: workflow nodes and scheduled runs are prompted by the system.
	BriefWorkloads []string `json:"brief_workloads,omitempty"`
	// SeedBriefWorkloads are workload kinds whose first operator-source message
	// is a brief but whose later messages are a person steering the run (an
	// ad hoc run is usually opened by an agent or script and then steered).
	SeedBriefWorkloads []string `json:"seed_brief_workloads,omitempty"`
	// DelegatedByLineage makes every operator-source message of a run that
	// another run spawned a delegation, not a person's words.
	DelegatedByLineage bool `json:"delegated_by_lineage,omitempty"`
	// AgentMarkers are regular expressions over operator-source text that only
	// an agent or a supervisor writes (an epoch heading, a lineage prefix). They
	// catch steering in runs whose lineage does not say so.
	AgentMarkers []string `json:"agent_markers,omitempty"`
	// DelegatingOpenings are regular expressions over the first operator-source
	// message of a run: a run opened that way was launched by an agent, so every
	// operator-source message in it is delegated.
	DelegatingOpenings []string `json:"delegating_openings,omitempty"`
	agentMarkers       []*regexp.Regexp
	delegatingOpenings []*regexp.Regexp
}

func compilePatterns(patterns []string) []*regexp.Regexp {
	var out []*regexp.Regexp
	for _, pattern := range patterns {
		if compiled, err := regexp.Compile(pattern); err == nil {
			out = append(out, compiled)
		}
	}
	return out
}

// Batch bounds one DecideBatch call. The gateway also caps these.
type Batch struct {
	MaxStates    int `json:"max_states"`
	MaxQuestions int `json:"max_questions"`
}

// Budgets bound one evaluation pass.
type Budgets struct {
	MaxRunsPerPass           int `json:"max_runs_per_pass"`
	MaxDecisionStatesPerPass int `json:"max_decision_states_per_pass"`
	// MaxSecondsPerPass bounds wall-clock work in one pass. Zero selects
	// DefaultSecondsPerPass. Runs are persisted group by group, so a pass that
	// reaches the budget returns what it finished and reports the remainder.
	MaxSecondsPerPass int `json:"max_seconds_per_pass,omitempty"`
	// MaxMessagesPerRun bounds the human messages of one run that message-scope
	// labels answer in a pass. Zero selects DefaultMessagesPerRun.
	MaxMessagesPerRun int `json:"max_messages_per_run,omitempty"`
}

// Label declares one thing to observe.
type Label struct {
	ID          string `json:"id"`
	Group       string `json:"group"`
	Description string `json:"description"`
	Scope       Scope  `json:"scope"`
	Method      Method `json:"method"`

	// Question is required for decision labels and for a deterministic label's
	// decision fallback.
	Question *Question `json:"question,omitempty"`

	// Rule is the CEL expression for deterministic and composite labels.
	Rule string `json:"rule,omitempty"`
	// Source and SourceVersion name an existing operator signal for signal labels.
	Source        string `json:"source,omitempty"`
	SourceVersion string `json:"source_version,omitempty"`
	// PriorRule is a deterministic CEL prior evaluated before an expensive
	// evaluator. A true prior is an observation, not a guess.
	PriorRule string `json:"prior_rule,omitempty"`
	// Fallback names the evaluator used when a deterministic rule cannot decide.
	// The only supported fallback in this slice is "decision".
	Fallback Method `json:"fallback,omitempty"`

	// Role is the decision role this label asks; empty uses runtime.default_role.
	// EscalateRole is the stronger role asked once when the answer is not
	// confident; empty uses runtime.escalate_role. Neither is part of the
	// definition hash: the model that answered is recorded on each observation
	// and proven separately.
	Role         string `json:"role,omitempty"`
	EscalateRole string `json:"escalate_role,omitempty"`
	// Population names the denominator a coverage row reports for this label.
	// Empty means every annotated run; "human_operator" means runs (or, for a
	// message-scope label, messages) a person wrote. It is reporting only.
	Population string `json:"population,omitempty"`

	Evidence   []EvidenceSelector `json:"evidence"`
	Candidates string             `json:"candidates,omitempty"`
	// Require is a CEL predicate over the same activation as candidates. It
	// declares when a label has enough evidence to be asked at all, so a run
	// with no messages and no tool calls never receives a model guess. This is
	// different from candidates (which selects useful work) and from evidence
	// (which names what the digest shows): a require predicate is the guard
	// that keeps an empty state from becoming a fabricated observation.
	Require       string        `json:"require,omitempty"`
	Sampling      Sampling      `json:"sampling"`
	Gate          Gate          `json:"gate"`
	Search        Search        `json:"search"`
	Friction      Friction      `json:"friction"`
	Compatibility Compatibility `json:"compatibility"`
	Supersedes    []string      `json:"supersedes,omitempty"`
	// Optional labels are skipped unless the caller asks for them. This is how
	// a weak label can exist in the vocabulary without producing data.
	Optional bool `json:"optional"`
}

// Question describes the decision question a label asks.
type Question struct {
	Type         QuestionType `json:"type"`
	Instructions string       `json:"instructions"`
	Options      []Option     `json:"options,omitempty"`
	Levels       []string     `json:"levels,omitempty"`
	// Offset maps a SCORE level index to the domain scale, e.g. levels 0..4 with
	// offset -2 yields a valence in -2..+2. The pressed score is index+offset.
	Offset           float64 `json:"offset,omitempty"`
	TrueDescription  string  `json:"true_description,omitempty"`
	FalseDescription string  `json:"false_description,omitempty"`
}

// Option is one CHOICE label.
type Option struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// EvidenceSelector names one bounded digest input.
type EvidenceSelector struct {
	Source string `json:"source"`
	Field  string `json:"field"`
	Label  string `json:"label,omitempty"`
}

// Sampling controls whether a candidate is evaluated and how it is weighted.
// Rate is applied deterministically from a hash so a replay is reproducible.
type Sampling struct {
	Rate    *float64 `json:"rate,omitempty"`
	Stratum string   `json:"stratum,omitempty"`
}

// Search declares the search facet and threshold that tags a run.
type Search struct {
	Facet          string   `json:"facet"`
	MinProbability *float64 `json:"min_probability,omitempty"`
}

// Friction declares whether a label may appear in friction reporting.
type Friction struct {
	Enabled        bool     `json:"enabled"`
	MinProbability *float64 `json:"min_probability,omitempty"`
	// MaxScore makes a low-is-bad signed score a friction positive only at or
	// below this value (a signed valence of -1 = negative).
	MaxScore *float64 `json:"max_score,omitempty"`
	// MinScore makes a high-is-bad score a friction positive only at or above
	// this value (a stall score of 2 = noticeable repetition). Exactly one of
	// MinScore/MaxScore is required for an enabled score label: without a
	// direction even a mild score would count as friction when confident.
	MinScore   *float64 `json:"min_score,omitempty"`
	CostUnit   string   `json:"cost_unit,omitempty"`
	Importance string   `json:"importance,omitempty"`
}

// Compatibility is this revision's declared relation to prior definitions.
// Reuse requires an explicit predecessor hash in compatible_with; change is
// reporting and defaults to breaking.
type Compatibility struct {
	Change         string   `json:"change,omitempty"`
	CompatibleWith []string `json:"compatible_with,omitempty"`
	BreakingWith   []string `json:"breaking_with,omitempty"`
}

// CompiledPack is a validated pack with one compiled evaluator per label.
type CompiledPack struct {
	Pack         *Pack
	LabelHashes  map[string]string
	PackHash     string
	labels       map[string]*compiledLabel
	order        []string
	featureEnv   *cel.Env
	compositeEnv *cel.Env
}

type compiledLabel struct {
	label      Label
	hash       string
	candidates cel.Program
	require    cel.Program
	rule       cel.Program
	prior      cel.Program
	composite  cel.Program
	// refs are the labels a composite rule reads; a composite with any is
	// derived: its proof comes from its inputs.
	refs compositeRefs
}

// derived reports whether the label is a composite over other labels.
func (c *compiledLabel) derived() bool { return c.label.Method == MethodComposite && c.refs.any() }

// packEnvelope accepts both {"pack": {...}} and a bare pack object.
type packEnvelope struct {
	Pack *Pack `json:"pack"`
}

// LoadPack reads and validates a pack file. It does not compile CEL; use
// Compile for an evaluator-ready pack.
func LoadPack(path string) (*Pack, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read pack %s: %w", path, err)
	}
	pack, err := ParsePack(body)
	if err != nil {
		return nil, fmt.Errorf("pack %s: %w", path, err)
	}
	return pack, nil
}

// ParsePack parses and validates pack JSON.
func ParsePack(body []byte) (*Pack, error) {
	var envelope packEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("decode json: %w", err)
	}
	pack := envelope.Pack
	if pack == nil {
		direct := &Pack{}
		if err := json.Unmarshal(body, direct); err != nil {
			return nil, fmt.Errorf("decode json: %w", err)
		}
		pack = direct
	}
	if pack.Name == "" && len(pack.Labels) == 0 {
		return nil, errors.New("pack is empty")
	}
	pack.normalize()
	if err := pack.Validate(); err != nil {
		return nil, err
	}
	return pack, nil
}

// LoadCompiled loads, validates and compiles a pack.
func LoadCompiled(path string) (*CompiledPack, error) {
	pack, err := LoadPack(path)
	if err != nil {
		return nil, err
	}
	return pack.Compile()
}

func (p *Pack) normalize() {
	if p.Runtime.Mode == "" {
		p.Runtime.Mode = "off"
	}
	if p.Runtime.DefaultRole == "" {
		p.Runtime.DefaultRole = DefaultDecisionRole
	}
	if p.Runtime.Batch.MaxStates <= 0 || p.Runtime.Batch.MaxStates > DefaultBatchStates {
		p.Runtime.Batch.MaxStates = DefaultBatchStates
	}
	if p.Runtime.Batch.MaxQuestions <= 0 || p.Runtime.Batch.MaxQuestions > DefaultBatchQuestions {
		p.Runtime.Batch.MaxQuestions = DefaultBatchQuestions
	}
	if p.Runtime.BackfillHorizonDays <= 0 {
		p.Runtime.BackfillHorizonDays = DefaultBackfillHorizon
	}
	if p.Runtime.Budgets.MaxRunsPerPass <= 0 {
		p.Runtime.Budgets.MaxRunsPerPass = DefaultRunsPerPass
	}
	if p.Runtime.Budgets.MaxDecisionStatesPerPass <= 0 {
		p.Runtime.Budgets.MaxDecisionStatesPerPass = DefaultDecisionPerPass
	}
	if p.Runtime.DigestChars <= 0 {
		p.Runtime.DigestChars = DefaultDigestLimit
	}
	if p.Runtime.Escalation.MaxLabels <= 0 {
		p.Runtime.Escalation.MaxLabels = DefaultBatchQuestions
	}
	// An invalid pattern is reported by Validate; skip it here.
	p.Runtime.Provenance.agentMarkers = compilePatterns(p.Runtime.Provenance.AgentMarkers)
	p.Runtime.Provenance.delegatingOpenings = compilePatterns(p.Runtime.Provenance.DelegatingOpenings)
	for i := range p.Labels {
		label := &p.Labels[i]
		label.Gate.migrateLegacy()
		if label.Sampling.Rate == nil {
			rate := 1.0
			label.Sampling.Rate = &rate
		}
		if label.Sampling.Stratum == "" {
			label.Sampling.Stratum = "candidate"
		}
		if label.Compatibility.Change == "" {
			label.Compatibility.Change = "breaking"
		}
	}
}

// Validate rejects a pack that would make an evaluator ambiguous or unsafe.
func (p *Pack) Validate() error {
	var problems []string
	add := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }
	if strings.TrimSpace(p.Name) == "" {
		add("pack name is empty")
	}
	if strings.TrimSpace(p.Version) == "" {
		add("pack version is empty")
	}
	switch p.Runtime.Mode {
	case "off", "passive", "idle", "balanced":
	default:
		add("runtime mode %q must be off, passive, idle or balanced", p.Runtime.Mode)
	}
	if p.Runtime.Escalation.MinConfidence < 0 || p.Runtime.Escalation.MinConfidence > 1 {
		add("runtime escalation min_confidence must be in [0,1]")
	}
	if root := p.Runtime.GoldRoot; root != "" && strings.TrimSpace(root) != root {
		add("runtime gold_root %q has leading or trailing space", root)
	}
	if ttl := strings.TrimSpace(p.Runtime.Escalation.SameModelTTL); ttl != "" {
		if parsed, err := time.ParseDuration(ttl); err != nil || parsed <= 0 {
			add("runtime escalation same_model_ttl %q must be a positive duration such as 10m", ttl)
		}
	}
	for field, patterns := range map[string][]string{
		"agent_markers": p.Runtime.Provenance.AgentMarkers, "delegating_openings": p.Runtime.Provenance.DelegatingOpenings,
	} {
		for _, pattern := range patterns {
			if _, err := regexp.Compile(pattern); err != nil {
				add("runtime provenance %s %q does not compile: %v", field, pattern, err)
			}
		}
	}
	if len(p.Labels) == 0 {
		add("pack declares no labels")
	}
	seen := map[string]bool{}
	env, envErr := featureEnv()
	compositeEnv, compositeErr := compositeEnvFor(env)
	for _, label := range p.Labels {
		if err := label.validate(env, compositeEnv, envErr, compositeErr, add); err != nil {
			add("%s", err)
		}
		if seen[label.ID] {
			add("duplicate label id %q", label.ID)
		}
		seen[label.ID] = true
	}
	if envErr == nil && compositeErr == nil {
		p.validateComposites(compositeEnv, add)
	}
	if len(problems) > 0 {
		return fmt.Errorf("invalid pack: %s", strings.Join(problems, "; "))
	}
	return nil
}

func (l Label) validate(env, compositeEnv *cel.Env, envErr, compositeErr error, add func(string, ...any)) error {
	id := strings.TrimSpace(l.ID)
	if id == "" {
		add("label id is empty")
	}
	if len(id) > 100 {
		add("%s: label id exceeds 100 characters", id)
	}
	if strings.TrimSpace(l.Group) == "" {
		add("%s: group is empty", id)
	}
	switch l.Scope {
	case ScopeMessage, ScopeEpisode, ScopeRun:
	default:
		add("%s: scope %q must be message, episode or run", id, l.Scope)
	}
	switch l.Method {
	case MethodDeterministic, MethodSignal, MethodDecision, MethodExtract, MethodComposite:
	default:
		add("%s: method %q is not supported", id, l.Method)
	}
	switch l.Population {
	case "", PopulationHumanOperator:
	default:
		add("%s: population %q must be empty or %q", id, l.Population, PopulationHumanOperator)
	}
	if l.Scope == ScopeMessage && l.Method != MethodDecision && l.Method != MethodDeterministic && l.Method != MethodComposite {
		add("%s: a message-scope label must be a decision, deterministic or composite label", id)
	}
	if l.Method == MethodDecision || l.Fallback == MethodDecision {
		if l.Question == nil {
			add("%s: decision evaluation requires a question", id)
		}
	}
	if l.Question != nil {
		if err := l.Question.validate(); err != nil {
			add("%s: %v", id, err)
		}
	}
	if l.Method == MethodDeterministic && strings.TrimSpace(l.Rule) == "" {
		add("%s: deterministic label requires rule", id)
	}
	if l.Method == MethodComposite && strings.TrimSpace(l.Rule) == "" {
		add("%s: composite label requires rule", id)
	}
	if l.Method == MethodSignal && strings.TrimSpace(l.Source) == "" {
		add("%s: signal label requires source", id)
	}
	if l.Fallback != "" && l.Fallback != MethodDecision {
		add("%s: fallback %q must be empty or decision", id, l.Fallback)
	}
	if strings.TrimSpace(l.PriorRule) != "" && (l.Question == nil || l.Question.Type != QuestionNoul) {
		add("%s: prior_rule is supported only for noul labels", id)
	}
	if l.Method == MethodComposite && l.Question != nil && l.Question.Type == QuestionChoice {
		add("%s: composite labels cannot answer a choice question; use noul or score", id)
	}
	if l.Sampling.Rate != nil && (*l.Sampling.Rate < 0 || *l.Sampling.Rate > 1) {
		add("%s: sampling rate must be in [0,1]", id)
	}
	switch l.Sampling.Stratum {
	case "candidate", "random":
	default:
		add("%s: sampling stratum %q must be candidate or random", id, l.Sampling.Stratum)
	}
	if l.Gate.Precision < 0 || l.Gate.Precision > 1 || l.Gate.Recall < 0 || l.Gate.Recall > 1 {
		add("%s: gate precision and recall must be in [0,1]", id)
	}
	if l.Gate.MinGold < 1 {
		add("%s: gate min_gold must be at least 1 (an unproven gate cannot produce a rate)", id)
	}
	l.Gate.validate(id, add)
	if strings.TrimSpace(l.Search.Facet) == "" {
		add("%s: search facet is required", id)
	}
	if l.Search.MinProbability != nil && (*l.Search.MinProbability < 0 || *l.Search.MinProbability > 1) {
		add("%s: search min_probability must be in [0,1]", id)
	}
	if l.Friction.Enabled {
		if l.Search.MinProbability == nil && l.Friction.MinProbability == nil {
			add("%s: enabled friction requires a probability threshold", id)
		}
		if l.Friction.CostUnit == "" {
			add("%s: enabled friction requires cost_unit", id)
		}
	}
	if l.Friction.MinProbability != nil && (*l.Friction.MinProbability < 0 || *l.Friction.MinProbability > 1) {
		add("%s: friction min_probability must be in [0,1]", id)
	}
	if l.Friction.MinScore != nil && l.Friction.MaxScore != nil {
		add("%s: friction takes min_score or max_score, not both", id)
	}
	if (l.Friction.MinScore != nil || l.Friction.MaxScore != nil) && (l.Question == nil || l.Question.Type != QuestionScore) {
		add("%s: a friction score bound requires a score question", id)
	}
	if l.Friction.Enabled && l.Question != nil && l.Question.Type == QuestionScore &&
		l.Friction.MinScore == nil && l.Friction.MaxScore == nil {
		add("%s: enabled score friction must declare min_score or max_score", id)
	}
	switch l.Friction.Importance {
	case "", "low", "medium", "high":
	default:
		add("%s: friction importance %q must be low, medium or high", id, l.Friction.Importance)
	}
	switch l.Compatibility.Change {
	case "compatible", "breaking":
	default:
		add("%s: compatibility change %q must be compatible or breaking", id, l.Compatibility.Change)
	}
	for _, hash := range append(append([]string{}, l.Compatibility.CompatibleWith...), l.Compatibility.BreakingWith...) {
		if len(hash) != 64 {
			add("%s: compatibility hash %q must be 64 hex characters", id, hash)
			continue
		}
		if _, err := hex.DecodeString(hash); err != nil {
			add("%s: compatibility hash %q is not hex", id, hash)
		}
	}
	for _, selector := range l.Evidence {
		if err := selector.validate(); err != nil {
			add("%s: %v", id, err)
		}
	}
	if envErr != nil {
		return envErr
	}
	if compositeErr != nil {
		return compositeErr
	}
	for field, expr := range map[string]string{
		"candidates": l.Candidates,
		"require":    l.Require,
		"rule":       l.Rule,
		"prior_rule": l.PriorRule,
	} {
		if strings.TrimSpace(expr) == "" {
			continue
		}
		if len(expr) > maximumCandidateExprSize {
			add("%s: %s expression exceeds %d characters", id, field, maximumCandidateExprSize)
			continue
		}
		target := env
		if field == "rule" && l.Method == MethodComposite {
			target = compositeEnv
		}
		if _, err := target.Compile(expr); err != nil {
			add("%s: %s CEL does not compile: %v", id, field, err)
		}
	}
	return nil
}

func (q Question) validate() error {
	if strings.TrimSpace(q.Instructions) == "" {
		return errors.New("question instructions are required")
	}
	if q.Offset < -100 || q.Offset > 100 {
		return errors.New("question offset is out of range")
	}
	switch q.Type {
	case QuestionChoice:
		if len(q.Options) < 2 || len(q.Options) > 255 {
			return fmt.Errorf("choice question needs 2-255 options, got %d", len(q.Options))
		}
		seen := map[string]bool{}
		for _, option := range q.Options {
			if strings.TrimSpace(option.Label) == "" {
				return errors.New("choice option label is empty")
			}
			if seen[option.Label] {
				return fmt.Errorf("duplicate choice option %q", option.Label)
			}
			seen[option.Label] = true
		}
	case QuestionScore:
		if len(q.Levels) < 2 || len(q.Levels) > 10 {
			return fmt.Errorf("score question needs 2-10 levels, got %d", len(q.Levels))
		}
		for i, level := range q.Levels {
			if strings.TrimSpace(level) == "" {
				return fmt.Errorf("score level %d is empty", i)
			}
		}
	case QuestionNoul:
		if strings.TrimSpace(q.TrueDescription) == "" || strings.TrimSpace(q.FalseDescription) == "" {
			return errors.New("noul question needs true_description and false_description")
		}
	default:
		return fmt.Errorf("question type %q must be choice, score or noul", q.Type)
	}
	return nil
}

// EvidenceSources lists every selector source a pack may reference.
var EvidenceSources = map[string]bool{
	"operator_messages":   true,
	"agent_text":          true,
	"tool_summary":        true,
	"timings":             true,
	"deterministic_flags": true,
	"existing_signals":    true,
	"workload":            true,
	"terminal":            true,
}

var evidenceFields = map[string]map[string]bool{
	"operator_messages":   {"text": true, "count": true, "last_text": true},
	"agent_text":          {"text": true, "count": true, "last_text": true},
	"tool_summary":        {"calls": true, "failures": true, "failed_commands": true, "repeated_paths": true, "repeated_commands": true, "failed_tools": true, "last_error": true},
	"timings":             {"duration_min": true, "time_to_first_tool_min": true},
	"deterministic_flags": {"stepback_triggers": true, "tool_findings": true, "compaction_count": true},
	"existing_signals":    {"label": true, "probability": true},
	"workload":            {"kind": true, "key": true},
	"terminal":            {"status": true, "class": true, "stop_reason": true, "exit_code": true},
}

func (s EvidenceSelector) validate() error {
	if !EvidenceSources[s.Source] {
		return fmt.Errorf("evidence source %q is not known", s.Source)
	}
	if !evidenceFields[s.Source][s.Field] {
		return fmt.Errorf("evidence field %q is not known for source %q", s.Field, s.Source)
	}
	if s.Source == "existing_signals" && (s.Field == "label" || s.Field == "probability") && strings.TrimSpace(s.Label) == "" {
		return errors.New("existing_signals evidence requires label")
	}
	return nil
}

// Compile validates the pack and builds one CEL program per expression.
func (p *Pack) Compile() (*CompiledPack, error) {
	return p.CompileWithCostLimit(defaultCostLimit)
}

// CompileWithCostLimit is Compile with an explicit CEL cost ceiling. Tests
// lower it to prove the limit is enforced; production uses the default.
func (p *Pack) CompileWithCostLimit(costLimit uint64) (*CompiledPack, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	env, err := featureEnv()
	if err != nil {
		return nil, err
	}
	composite, err := compositeEnvFor(env)
	if err != nil {
		return nil, err
	}
	cp := &CompiledPack{
		Pack:         p,
		LabelHashes:  map[string]string{},
		labels:       map[string]*compiledLabel{},
		featureEnv:   env,
		compositeEnv: composite,
	}
	for index := range p.Labels {
		label := &p.Labels[index]
		hash, err := LabelDefinitionHash(*label)
		if err != nil {
			return nil, fmt.Errorf("%s: hash: %w", label.ID, err)
		}
		cl := &compiledLabel{label: *label, hash: hash}
		if label.Candidates != "" {
			if cl.candidates, err = compileProgram(env, label.Candidates, costLimit); err != nil {
				return nil, fmt.Errorf("%s: candidates: %w", label.ID, err)
			}
		}
		if label.Require != "" {
			if cl.require, err = compileProgram(env, label.Require, costLimit); err != nil {
				return nil, fmt.Errorf("%s: require: %w", label.ID, err)
			}
		}
		ruleEnv := env
		if label.Method == MethodComposite {
			ruleEnv = composite
		}
		if label.Rule != "" {
			if cl.rule, err = compileProgram(ruleEnv, label.Rule, costLimit); err != nil {
				return nil, fmt.Errorf("%s: rule: %w", label.ID, err)
			}
		}
		if label.PriorRule != "" {
			if cl.prior, err = compileProgram(env, label.PriorRule, costLimit); err != nil {
				return nil, fmt.Errorf("%s: prior_rule: %w", label.ID, err)
			}
		}
		if label.Method == MethodComposite {
			cl.composite = cl.rule
			if cl.refs, err = analyzeComposite(composite, label.Rule); err != nil {
				return nil, fmt.Errorf("%s: rule: %w", label.ID, err)
			}
		}
		cp.labels[label.ID] = cl
		cp.LabelHashes[label.ID] = hash
		cp.order = append(cp.order, label.ID)
	}
	cp.PackHash, err = p.semanticHash(cp.LabelHashes)
	if err != nil {
		return nil, err
	}
	return cp, nil
}

// Label returns the compiled label.
func (cp *CompiledPack) Label(id string) (*compiledLabel, bool) {
	label, ok := cp.labels[id]
	return label, ok
}

// Order returns label ids in pack order.
func (cp *CompiledPack) Order() []string { return append([]string(nil), cp.order...) }

// Labels returns the pack's labels in pack order.
func (cp *CompiledPack) Labels() []Label { return append([]Label(nil), cp.Pack.Labels...) }

// labelSemantic is the canonical JSON whose sha256 identifies one label
// revision. Reporting-only fields (description, gate, search, friction,
// sampling, compatibility, optional) are intentionally excluded.
type labelSemantic struct {
	ID            string             `json:"id"`
	Scope         Scope              `json:"scope"`
	Method        Method             `json:"method"`
	Question      *questionSemantic  `json:"question,omitempty"`
	Rule          string             `json:"rule,omitempty"`
	Source        string             `json:"source,omitempty"`
	SourceVersion string             `json:"source_version,omitempty"`
	PriorRule     string             `json:"prior_rule,omitempty"`
	Fallback      Method             `json:"fallback,omitempty"`
	Evidence      []evidenceSemantic `json:"evidence,omitempty"`
	// DigestFormat and DigestPriority identify the state a model reads for a
	// run-scope decision: how the digest is rendered (digestFormat) and which
	// sections the label ranks first (its evidence order). Either changing
	// changes what the model is asked, so it changes the hash.
	DigestFormat   int      `json:"digest_format,omitempty"`
	DigestPriority []string `json:"digest_priority,omitempty"`
	Candidates     string   `json:"candidates,omitempty"`
	// Require is a CEL predicate over the same activation as candidates. It
	// declares when a label has enough evidence to be asked at all, so a run
	// with no messages and no tool calls never receives a model guess. This is
	// different from candidates (which selects useful work) and from evidence
	// (which names what the digest shows): a require predicate is the guard
	// that keeps an empty state from becoming a fabricated observation.
	Require string `json:"require,omitempty"`
}

type questionSemantic struct {
	Type             QuestionType `json:"type"`
	Instructions     string       `json:"instructions"`
	Options          []Option     `json:"options,omitempty"`
	Levels           []string     `json:"levels,omitempty"`
	Offset           float64      `json:"offset,omitempty"`
	TrueDescription  string       `json:"true_description,omitempty"`
	FalseDescription string       `json:"false_description,omitempty"`
}

type evidenceSemantic struct {
	Source string `json:"source"`
	Field  string `json:"field"`
	Label  string `json:"label,omitempty"`
}

// readsRunDigest reports whether a model answers the label from a run's digest:
// a run-scope decision, or a rule that falls back to one. Message labels read
// their own message digest, which this format does not shape.
func (l Label) readsRunDigest() bool {
	return l.Scope == ScopeRun && (l.Method == MethodDecision || l.Fallback == MethodDecision)
}

// LabelDefinitionHash is the canonical sha256 of a label's semantic definition.
func LabelDefinitionHash(label Label) (string, error) {
	semantic := labelSemantic{
		ID:            label.ID,
		Scope:         label.Scope,
		Method:        label.Method,
		Rule:          normalizeExpr(label.Rule),
		Source:        label.Source,
		SourceVersion: label.SourceVersion,
		PriorRule:     normalizeExpr(label.PriorRule),
		Fallback:      label.Fallback,
		Candidates:    normalizeExpr(label.Candidates),
		Require:       normalizeExpr(label.Require),
	}
	if label.Question != nil {
		semantic.Question = &questionSemantic{
			Type:             label.Question.Type,
			Instructions:     strings.TrimSpace(label.Question.Instructions),
			Options:          append([]Option(nil), label.Question.Options...),
			Levels:           append([]string(nil), label.Question.Levels...),
			Offset:           label.Question.Offset,
			TrueDescription:  strings.TrimSpace(label.Question.TrueDescription),
			FalseDescription: strings.TrimSpace(label.Question.FalseDescription),
		}
	}
	if label.readsRunDigest() {
		semantic.DigestFormat = digestFormat
		semantic.DigestPriority = DigestPriority(label.Evidence)
	}
	for _, selector := range label.Evidence {
		semantic.Evidence = append(semantic.Evidence, evidenceSemantic{Source: selector.Source, Field: selector.Field, Label: selector.Label})
	}
	sort.Slice(semantic.Evidence, func(i, j int) bool {
		if semantic.Evidence[i].Source != semantic.Evidence[j].Source {
			return semantic.Evidence[i].Source < semantic.Evidence[j].Source
		}
		if semantic.Evidence[i].Field != semantic.Evidence[j].Field {
			return semantic.Evidence[i].Field < semantic.Evidence[j].Field
		}
		return semantic.Evidence[i].Label < semantic.Evidence[j].Label
	})
	body, err := canonicalJSON(semantic)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}

func (p *Pack) semanticHash(hashes map[string]string) (string, error) {
	type entry struct {
		ID   string `json:"id"`
		Hash string `json:"hash"`
	}
	entries := make([]entry, 0, len(hashes))
	for id, hash := range hashes {
		entries = append(entries, entry{ID: id, Hash: hash})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	body, err := canonicalJSON(struct {
		Name    string  `json:"name"`
		Version string  `json:"version"`
		Labels  []entry `json:"labels"`
	}{Name: p.Name, Version: p.Version, Labels: entries})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}

func normalizeExpr(expr string) string { return strings.Join(strings.Fields(expr), " ") }

// canonicalJSON marshals through a map so keys are sorted by encoding/json's
// deterministic map order, then re-encodes compactly.
func canonicalJSON(value any) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var generic any
	if err := decoder.Decode(&generic); err != nil {
		return nil, err
	}
	return json.Marshal(generic)
}

// PackSummary is the pack identity surface HTTP/CLI read.
type PackSummary struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	PackHash    string            `json:"pack_hash"`
	Labels      int               `json:"labels"`
	Mode        string            `json:"mode"`
	LabelHashes map[string]string `json:"label_hashes"`
	// GoldRoot is the resolved absolute directory gold files are read from and
	// written to; empty when the service was composed without a state root.
	GoldRoot string `json:"gold_root,omitempty"`
}

// Summary returns the pack identity.
func (cp *CompiledPack) Summary() PackSummary {
	return PackSummary{
		Name: cp.Pack.Name, Version: cp.Pack.Version, PackHash: cp.PackHash,
		Labels: len(cp.Pack.Labels), Mode: cp.Pack.Runtime.Mode,
		LabelHashes: cp.LabelHashes,
	}
}
