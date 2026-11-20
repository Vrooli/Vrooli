package runannotations

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ErrPackNotFound says the requested pack name is not the loaded pack.
var ErrPackNotFound = errors.New("annotation pack not found")

// ErrSelectionUnavailable says no run selector is wired; the caller must name
// runs explicitly or the orchestrator must provide one.
var ErrSelectionUnavailable = errors.New("annotation run selection unavailable")

// RunSelector supplies a bounded run set for a batch. The SQLite adapter is
// RunSelector in features_sqlite.go; tests use SliceSelector.
type RunSelector interface {
	CandidateRuns(ctx context.Context, since time.Time, limit int) ([]string, error)
}

// SliceSelector is a deterministic selector for tests.
type SliceSelector []string

// CandidateRuns implements RunSelector.
func (s SliceSelector) CandidateRuns(_ context.Context, _ time.Time, limit int) ([]string, error) {
	if limit > 0 && len(s) > limit {
		return append([]string(nil), s[:limit]...), nil
	}
	return append([]string(nil), s...), nil
}

// WindowSelector is an optional selector that also honors an upper bound.
type WindowSelector interface {
	CandidateRunsWindow(ctx context.Context, since, until time.Time, limit int) ([]string, error)
}

// HashedWindowSelector is an optional selector that also knows the pack's
// current definition hashes. A selector that implements it can prefer runs
// that this pack has not annotated yet (or annotated longest ago), which turns
// a bounded pass into forward progress instead of re-reading the same newest
// rows every interval.
type HashedWindowSelector interface {
	CandidateRunsWindowFor(ctx context.Context, since, until time.Time, limit int, definitionHashes []string) ([]string, error)
}

// RunCensus optionally supplies the candidate-run denominator for coverage.
// Without one, coverage's eligible_runs is the annotated population and the
// response says so explicitly.
type RunCensus interface {
	EligibleRuns(ctx context.Context, since, until time.Time) (int, error)
}

// SignalReader reads existing operator signals for one run. The interface is
// narrow so the fixture path never needs the interventions package.
type SignalReader interface {
	RunSignals(ctx context.Context, runID string) (map[string]SignalFact, error)
}

// Options compose a Service. A nil Decisions client makes decision labels
// state unavailable rather than failing the run.
type Options struct {
	Store          *Store
	Features       FeatureSource
	Signals        SignalReader
	Selector       RunSelector
	Decisions      DecisionClient
	Census         RunCensus
	Clock          func() time.Time
	MaxDigestChars int
	CEL            celEvaluator
}

// Service evaluates a pack and persists observations.
type Service struct {
	pack        *CompiledPack
	store       *Store
	features    FeatureSource
	signals     SignalReader
	selector    RunSelector
	decisions   DecisionClient
	census      RunCensus
	now         func() time.Time
	digestLimit int
	cel         celEvaluator
}

// NewService validates its dependencies and returns a service.
func NewService(pack *CompiledPack, opts Options) (*Service, error) {
	if pack == nil {
		return nil, errors.New("annotation pack is required")
	}
	if opts.Store == nil {
		return nil, errors.New("annotation store is required")
	}
	if opts.Features == nil {
		return nil, errors.New("annotation feature source is required")
	}
	if opts.CEL.CostLimit != 0 && opts.CEL.CostLimit != defaultCostLimit {
		recompiled, err := pack.Pack.CompileWithCostLimit(opts.CEL.CostLimit)
		if err != nil {
			return nil, err
		}
		pack = recompiled
	}
	service := &Service{
		pack: pack, store: opts.Store, features: opts.Features, signals: opts.Signals,
		selector: opts.Selector, decisions: opts.Decisions, census: opts.Census, now: opts.Clock,
		digestLimit: opts.MaxDigestChars, cel: opts.CEL,
	}
	if service.now == nil {
		service.now = time.Now
	}
	if service.digestLimit <= 0 {
		service.digestLimit = DefaultDigestLimit
	}
	if service.cel.CostLimit == 0 {
		service.cel.CostLimit = defaultCostLimit
	}
	return service, nil
}

// OpenService loads a compiled pack and applies the annotation schema.
func OpenService(ctx context.Context, packPath string, store *Store, opts Options) (*Service, error) {
	pack, err := LoadCompiled(packPath)
	if err != nil {
		return nil, err
	}
	if err := store.EnsureSchema(ctx); err != nil {
		return nil, err
	}
	return NewService(pack, opts)
}

// Pack returns the compiled pack.
func (s *Service) Pack() *CompiledPack { return s.pack }

// DefinitionHashes returns the pack's current label definition hashes in a
// stable order. A selector uses them to recognize runs this pack has already
// answered, without knowing which label produced the row.
func (s *Service) DefinitionHashes() []string {
	if s == nil || s.pack == nil {
		return nil
	}
	out := make([]string, 0, len(s.pack.Pack.Labels))
	for _, declaration := range s.pack.Pack.Labels {
		if hash := s.pack.LabelHashes[declaration.ID]; hash != "" {
			out = append(out, hash)
		}
	}
	return out
}

// ModelAffinity is the pack's declared scheduler affinity; empty means the
// scheduler groups the work under no specific family.
func (s *Service) ModelAffinity() string {
	if s == nil || s.pack == nil {
		return ""
	}
	return strings.TrimSpace(s.pack.Pack.Runtime.ModelAffinity)
}

// PackSummary returns the loaded pack identity.
func (s *Service) PackSummary() PackSummary { return s.pack.Summary() }

// PackSummaryFor rejects a pack name that is not this service's pack.
func (s *Service) PackSummaryFor(name string) (PackSummary, error) {
	if name != "" && name != s.pack.Pack.Name {
		return PackSummary{}, fmt.Errorf("%w: %s", ErrPackNotFound, name)
	}
	return s.pack.Summary(), nil
}

// LabelView is one label declaration plus its window coverage.
// runWork is one run's in-progress evaluation.
type runWork struct {
	facts    RunFacts
	features RunFeatures
	digest   string
	// evidenceHash identifies the bounded state every label on this run reads.
	// A cached answer is reused only when it was computed from the same
	// evidence, so new messages, tool failures or signals invalidate it.
	evidenceHash string
	// digestShrunk records that a model could not read the full digest and
	// this work already retried with a smaller one. The second overflow is
	// reported, never retried forever.
	digestShrunk bool
	overflowed   bool
	// retryable marks a transport failure that has not been retried yet.
	// transportRetried bounds that retry to one per pass.
	retryable        bool
	transportRetried bool
	activation       map[string]any
	observations     map[string]Annotation
	evidence         map[string][]string
	order            []string
	toSave           []Annotation
	needs            []string
	pending          []string
	skipped          []SkippedLabel
}

// Annotate evaluates one run and persists new observations.
func (s *Service) Annotate(ctx context.Context, runID string, opts AnnotateOptions) (RunView, error) {
	views, errs := s.annotateRuns(ctx, []string{runID}, opts)
	if len(errs) > 0 {
		return RunView{}, errors.New(errs[0].Reason)
	}
	if len(views) == 0 {
		return RunView{}, ErrScopeNotFound
	}
	return views[0], nil
}

// Evaluate selects a bounded run set (or uses the caller's ids) and annotates
// it, sharing decision batches across runs.
func (s *Service) Evaluate(ctx context.Context, opts EvaluateOptions) (EvaluateResult, error) {
	limit := opts.Limit
	budget := s.pack.Pack.Runtime.Budgets.MaxRunsPerPass
	if limit <= 0 {
		limit = min(20, budget)
	}
	if limit > budget {
		limit = budget
	}
	until := opts.Until
	if until.IsZero() {
		until = s.now()
	}
	runIDs := append([]string(nil), opts.RunIDs...)
	if len(runIDs) == 0 {
		if s.selector == nil {
			return EvaluateResult{}, ErrSelectionUnavailable
		}
		var (
			selected []string
			err      error
		)
		if hashedSelector, ok := s.selector.(HashedWindowSelector); ok {
			selected, err = hashedSelector.CandidateRunsWindowFor(ctx, opts.Since, until, limit, s.DefinitionHashes())
		} else if windowSelector, ok := s.selector.(WindowSelector); ok {
			selected, err = windowSelector.CandidateRunsWindow(ctx, opts.Since, until, limit)
		} else {
			selected, err = s.selector.CandidateRuns(ctx, opts.Since, limit)
		}
		if err != nil {
			return EvaluateResult{}, fmt.Errorf("select runs: %w", err)
		}
		runIDs = selected
	}
	if len(runIDs) > limit {
		runIDs = runIDs[:limit]
	}
	// Bound wall-clock work so one large backfill cannot be cancelled by a
	// client or gateway timeout after doing everything and saving nothing.
	// Groups are persisted as they finish, and the remainder is reported.
	deadline := time.Time{}
	if budget := s.passBudget(opts.TimeBudget); budget > 0 {
		deadline = time.Now().Add(budget)
	}
	views, errs, remaining := s.annotateRunsBounded(ctx, runIDs, AnnotateOptions{
		Force: opts.Force, IncludeOptional: opts.IncludeOptional,
		Labels: opts.Labels, DryRun: opts.DryRun,
	}, deadline)
	result := EvaluateResult{
		Window:    windowRange(opts.Since, until),
		Submitted: len(runIDs) - len(remaining),
		Remaining: len(remaining),
		Truncated: len(remaining) > 0,
		Runs:      views,
		Errors:    errs,
	}
	for _, view := range views {
		result.Counts.Observed += view.Counts.Observed
		result.Counts.Compatible += view.Counts.Compatible
		result.Counts.NotObserved += view.Counts.NotObserved
		result.Counts.Unavailable += view.Counts.Unavailable
		if view.Coverage.Observed > 0 {
			result.Observed++
		} else if view.Counts.Unavailable > 0 {
			result.Unavailable++
		}
	}
	if result.Submitted > 0 {
		result.Coverage = float64(result.Observed) / float64(result.Submitted)
	}
	result.Labels = s.coverageFromViews(views)
	if result.Submitted == 0 {
		result.Labels = []CoverageLabel{}
	}
	return result, nil
}

func (s *Service) annotateRuns(ctx context.Context, runIDs []string, opts AnnotateOptions) ([]RunView, []RunError) {
	views, errs, _ := s.annotateRunsBounded(ctx, runIDs, opts, time.Time{})
	return views, errs
}

// passBudget resolves the wall-clock budget for one pass. A zero override uses
// the pack's declared budget (or the default); a negative one disables it.
func (s *Service) passBudget(override time.Duration) time.Duration {
	if override < 0 {
		return 0
	}
	if override > 0 {
		return override
	}
	seconds := s.pack.Pack.Runtime.Budgets.MaxSecondsPerPass
	if seconds <= 0 {
		seconds = DefaultSecondsPerPass
	}
	return time.Duration(seconds) * time.Second
}

// annotateRunsBounded evaluates runs in decision-batch-sized groups and saves
// each group before starting the next. A deadline stops the pass between
// groups, returning the run IDs it did not reach. Grouping matters for more
// than latency: a failure or cancellation now costs at most one group instead
// of discarding every answered run in the pass.
func (s *Service) annotateRunsBounded(ctx context.Context, runIDs []string, opts AnnotateOptions, deadline time.Time) ([]RunView, []RunError, []string) {
	groupSize := s.pack.Pack.Runtime.Batch.MaxStates
	if groupSize <= 0 {
		groupSize = DefaultBatchStates
	}
	var (
		views []RunView
		errs  []RunError
	)
	for start := 0; start < len(runIDs); start += groupSize {
		end := min(start+groupSize, len(runIDs))
		if start > 0 && (!deadline.IsZero() && time.Now().After(deadline) || ctx.Err() != nil) {
			return views, errs, runIDs[start:]
		}
		works := make([]*runWork, 0, end-start)
		for _, runID := range runIDs[start:end] {
			work, err := s.prepare(ctx, runID, opts)
			if err != nil {
				errs = append(errs, RunError{RunID: runID, Reason: err.Error()})
				continue
			}
			works = append(works, work)
		}
		// Bound the decision phase by the pass deadline. Without this a single
		// hung gateway call could outlive the budget (its own timeout is
		// longer) and take the client's request down with it, losing the whole
		// group after the work was already done. A cancelled decision marks
		// those labels unavailable, which is never cached, so the next pass
		// retries them.
		decisionCtx := ctx
		cancelDecision := func() {}
		if !deadline.IsZero() {
			decisionCtx, cancelDecision = context.WithDeadline(ctx, deadline)
		}
		s.evaluateDecisions(decisionCtx, works)
		cancelDecision()
		for _, work := range works {
			s.evaluateComposites(work)
			if !opts.DryRun {
				s.save(ctx, work)
			}
		}
		for _, work := range works {
			views = append(views, s.viewFor(work))
		}
	}
	return views, errs, nil
}

func (s *Service) prepare(ctx context.Context, runID string, opts AnnotateOptions) (*runWork, error) {
	facts, err := s.features.Facts(ctx, ScopeRun, runID)
	if err != nil {
		return nil, err
	}
	if facts.RunID == "" {
		facts.RunID = runID
	}
	if s.signals != nil {
		signals, err := s.signals.RunSignals(ctx, runID)
		if err == nil && len(signals) > 0 {
			if facts.ExistingSignals == nil {
				facts.ExistingSignals = map[string]SignalFact{}
			}
			for label, fact := range signals {
				facts.ExistingSignals[label] = fact
			}
		}
	}
	features := FeaturesFromFacts(facts)
	digest, err := BuildDigestBounded(facts, s.digestLimit)
	if err != nil {
		// A digest that cannot be bounded is a configuration error, not an
		// annotation. Refuse rather than send a partial transcript.
		return nil, err
	}
	work := &runWork{
		facts: facts, features: features, digest: digest,
		evidenceHash: evidenceFingerprint(digest, facts),
		activation:   features.Activation(),
		observations: map[string]Annotation{}, evidence: map[string][]string{},
	}
	allowed := map[string]bool{}
	for _, id := range opts.Labels {
		allowed[id] = true
	}
	maxLabels := opts.MaxLabels
	if maxLabels <= 0 {
		maxLabels = DefaultMaxRunAnnotation
	}
	count := 0
	for _, label := range s.pack.Labels() {
		if len(allowed) > 0 && !allowed[label.ID] {
			work.skipped = append(work.skipped, SkippedLabel{LabelID: label.ID, Reason: "not_requested"})
			continue
		}
		if label.Scope != ScopeRun {
			work.skipped = append(work.skipped, SkippedLabel{LabelID: label.ID, Reason: "scope_not_supported_in_slice"})
			continue
		}
		if label.Optional && !opts.IncludeOptional {
			work.skipped = append(work.skipped, SkippedLabel{LabelID: label.ID, Reason: "optional_not_requested"})
			continue
		}
		if count >= maxLabels {
			work.skipped = append(work.skipped, SkippedLabel{LabelID: label.ID, Reason: "label_budget_exhausted"})
			continue
		}
		compiled, ok := s.pack.Label(label.ID)
		if !ok {
			continue
		}
		if compiled.candidates != nil {
			ok, err := s.cel.evalBool(compiled.candidates, work.activation)
			if err != nil {
				work.observations[label.ID] = s.observation(label, compiled.hash, StateUnavailable, "candidate_error: "+err.Error())
				work.order = append(work.order, label.ID)
				continue
			}
			if !ok {
				work.skipped = append(work.skipped, SkippedLabel{LabelID: label.ID, Reason: "not_a_candidate"})
				s.supersedeIfAnswered(ctx, work, label, compiled.hash, "not_a_candidate", opts.Force)
				continue
			}
		}
		if compiled.require != nil {
			holds, err := s.cel.evalBool(compiled.require, work.activation)
			if err != nil {
				work.observations[label.ID] = s.observation(label, compiled.hash, StateUnavailable, "require_error: "+err.Error())
				work.order = append(work.order, label.ID)
				continue
			}
			if !holds {
				// The run has nothing this label can judge. Asking a model
				// about an empty digest returns a plausible-looking number,
				// which is worse than no observation.
				work.skipped = append(work.skipped, SkippedLabel{LabelID: label.ID, Reason: "insufficient_evidence"})
				s.supersedeIfAnswered(ctx, work, label, compiled.hash, "insufficient_evidence", opts.Force)
				continue
			}
		}
		if !sampled(runID, label) {
			work.skipped = append(work.skipped, SkippedLabel{LabelID: label.ID, Reason: "sampled_out"})
			continue
		}
		count++
		work.evidence[label.ID] = evidenceIDsFor(label, facts)
		s.evaluateLabel(ctx, work, compiled, label, opts.Force)
	}
	return work, nil
}

// evaluateLabel runs the cheap path and records deferred work. Decision labels
// are answered in a shared batch after every run is prepared.
func (s *Service) evaluateLabel(ctx context.Context, work *runWork, compiled *compiledLabel, label Label, force bool) {
	now := s.now().UTC()
	// A deterministic prior is cheaper and more trustworthy than a model. It is
	// an observation, never a guess.
	if compiled.prior != nil {
		holds, err := s.cel.evalBool(compiled.prior, work.activation)
		if err != nil {
			s.record(work, s.observation(label, compiled.hash, StateUnavailable, "prior_error: "+err.Error()))
			return
		}
		if holds {
			observation := s.observation(label, compiled.hash, StateObserved, "deterministic_prior")
			observation.Method = string(MethodDeterministic)
			one := 1.0
			observation.Probability = &one
			observation.Confidence = &one
			observation.Value = Value{Kind: "noul", Noul: &one}
			observation.ObservedAt = now
			s.record(work, observation)
			return
		}
	}
	if !force {
		if existing, err := s.store.Current(ctx, ScopeRun, work.facts.RunID, label.ID, compiled.hash); err == nil && reusableObservation(existing) && existing.EvidenceHash == work.evidenceHash {
			// Same definition and same evidence: reuse a settled answer without
			// inference. It is still an observed row; a replay must not make it
			// look newer.
			work.observations[label.ID] = *existing
			work.order = append(work.order, label.ID)
			return
		}
		if len(label.Compatibility.CompatibleWith) > 0 {
			if prior, err := s.store.Compatible(ctx, ScopeRun, work.facts.RunID, label.ID, label.Compatibility.CompatibleWith); err == nil && prior != nil && prior.EvidenceHash == work.evidenceHash {
				reused := *prior
				reused.DefinitionHash = compiled.hash
				reused.State = StateCompatible
				reused.Reason = "reused compatible predecessor " + shortHash(prior.DefinitionHash)
				reused.ObservedAt = now
				s.record(work, reused)
				return
			}
		}
	}
	switch label.Method {
	case MethodDeterministic:
		s.evaluateDeterministic(work, compiled, label)
	case MethodSignal:
		s.evaluateSignal(work, compiled, label)
	case MethodDecision:
		work.needs = append(work.needs, label.ID)
	case MethodComposite:
		work.pending = append(work.pending, label.ID)
	case MethodExtract:
		s.record(work, s.observation(label, compiled.hash, StateUnavailable, "extract_not_implemented_in_slice"))
	default:
		s.record(work, s.observation(label, compiled.hash, StateUnavailable, "unknown_method"))
	}
}

func (s *Service) evaluateDeterministic(work *runWork, compiled *compiledLabel, label Label) {
	value, err := s.cel.evalValue(compiled.rule, work.activation)
	if err != nil {
		s.record(work, s.observation(label, compiled.hash, StateUnavailable, "rule_error: "+err.Error()))
		return
	}
	observation := s.observation(label, compiled.hash, StateObserved, "")
	observation.Method = string(MethodDeterministic)
	observation.ObservedAt = s.now().UTC()
	decided := true
	switch native := value.(type) {
	case string:
		if native == "" {
			decided = false
		} else {
			observation.Value = Value{Kind: "string", Text: native}
			one := 1.0
			observation.Probability = &one
			observation.Confidence = &one
		}
	case bool:
		observation.Value = Value{Kind: "bool", Bool: &native}
		probability := 0.0
		if native {
			probability = 1.0
		}
		observation.Probability = &probability
		observation.Confidence = &probability
	case float64:
		observation.Value = Value{Kind: "number", Number: &native}
	default:
		decided = false
	}
	if decided {
		s.record(work, observation)
		return
	}
	if label.Fallback == MethodDecision && label.Question != nil {
		work.needs = append(work.needs, label.ID)
		return
	}
	observation.State = StateNotObserved
	observation.Reason = "deterministic_rule_undecided"
	s.record(work, observation)
}

// reusableObservation reports whether a stored row is settled enough to skip
// re-evaluation. An unavailable row is not: a transient gateway failure or a
// busy lock must not freeze a label for that run forever. Observed,
// compatible and deterministic not-observed answers are stable for the
// definition hash.
// evidenceFingerprint identifies everything a run-scoped answer reads: the
// bounded digest plus the operator signals it may cite. Definition hashes cover
// the question; this covers the state, so an appended message or a newly
// arrived signal recomputes the answer instead of serving a stale one.
func evidenceFingerprint(digest string, facts RunFacts) string {
	parts := make([]string, 0, len(facts.ExistingSignals))
	for label, signal := range facts.ExistingSignals {
		value := ""
		if signal.Probability != nil {
			value = strconv.FormatFloat(*signal.Probability, 'g', -1, 64)
		}
		parts = append(parts, label+"="+value)
	}
	sort.Strings(parts)
	sum := sha256.Sum256([]byte(digest + "\n" + strings.Join(parts, "\n")))
	return hex.EncodeToString(sum[:])
}

// supersedeIfAnswered writes an explicit not_observed row when a run that was
// answered for this definition no longer qualifies. Facts change: a run
// appends evidence, a classifier improves, a tool summary is corrected. Without
// the supersede the old row stays "latest" and coverage keeps counting an
// answer that no longer applies. The lookup is by definition hash, so rows from
// another revision are left to the stale accounting.
func (s *Service) supersedeIfAnswered(ctx context.Context, work *runWork, label Label, hash, reason string, force bool) {
	if force || s.store == nil {
		return
	}
	if prior, err := s.store.Current(ctx, ScopeRun, work.facts.RunID, label.ID, hash); err == nil && prior != nil {
		s.record(work, s.observation(label, hash, StateNotObserved, reason))
	}
}

func reusableObservation(observation *Annotation) bool {
	if observation == nil {
		return false
	}
	switch observation.State {
	case StateObserved, StateCompatible, StateNotObserved:
		return true
	default:
		return false
	}
}

func (s *Service) evaluateSignal(work *runWork, compiled *compiledLabel, label Label) {
	fact, ok := work.facts.ExistingSignals[label.Source]
	reason := ""
	switch {
	case !ok:
		reason = "signal_not_present"
	case label.SourceVersion != "" && fact.Version != "" && fact.Version != label.SourceVersion:
		reason = "signal_version_mismatch"
	}
	if reason != "" {
		// The deterministic signal stays authoritative when it exists. When it
		// does not, a declared decision fallback asks the label's own question
		// instead of leaving the label permanently unavailable; the observation
		// records the decision method, so a consumer can tell them apart.
		if label.Fallback == MethodDecision && label.Question != nil {
			work.needs = append(work.needs, label.ID)
			return
		}
		s.record(work, s.observation(label, compiled.hash, StateUnavailable, reason))
		return
	}
	observation := s.observation(label, compiled.hash, StateObserved, "")
	observation.Method = string(MethodSignal)
	observation.ObservedAt = s.now().UTC()
	observation.EvidenceEventIDs = boundIDs([]string{fact.LatestEventID}, 8)
	if label.Question != nil && label.Question.Type == QuestionNoul {
		observation.Value = Value{Kind: "noul", Noul: fact.Probability}
	} else {
		observation.Value = Value{Kind: "choice", Choice: fact.Label}
		if fact.Probability != nil {
			observation.Value.Probabilities = map[string]float64{fact.Label: *fact.Probability}
		}
	}
	observation.Probability = fact.Probability
	s.record(work, observation)
}

// evaluateDecisions answers the pending decision labels. A model with a small
// context can reject a digest that is still valid JSON; the pass first splits
// the question set and then re-asks the remaining single questions once with a
// smaller digest. Only an overflow is retried; any other failure is reported.
func (s *Service) evaluateComposites(work *runWork) {
	if len(work.pending) == 0 {
		return
	}
	pending := append([]string(nil), work.pending...)
	for pass := 0; pass < 3 && len(pending) > 0; pass++ {
		var next []string
		for _, labelID := range pending {
			compiled, ok := s.pack.Label(labelID)
			if !ok {
				continue
			}
			labels := map[string]float64{}
			present := map[string]bool{}
			for id, observation := range work.observations {
				number, ok := numericObservation(observation)
				if !ok {
					continue
				}
				labels[id] = number
				present[id] = true
			}
			activation := cloneActivation(work.activation)
			activation["labels"] = labels
			activation["present"] = present
			value, err := s.cel.evalNumber(compiled.composite, activation)
			if err != nil {
				if pass == 2 {
					s.record(work, s.observation(compiled.label, compiled.hash, StateNotObserved, "composite_inputs_missing: "+err.Error()))
				} else {
					next = append(next, labelID)
				}
				continue
			}
			observation := s.observation(compiled.label, compiled.hash, StateObserved, "")
			observation.Method = string(MethodComposite)
			observation.ObservedAt = s.now().UTC()
			questionType := QuestionType("")
			if compiled.label.Question != nil {
				questionType = compiled.label.Question.Type
			}
			switch questionType {
			case QuestionNoul:
				clamped := math.Max(0, math.Min(1, value))
				observation.Value = Value{Kind: "noul", Noul: &clamped}
				observation.Probability = &clamped
			case QuestionScore:
				observation.Value = Value{Kind: "score", Score: &value, Levels: append([]string(nil), compiled.label.Question.Levels...)}
			default:
				observation.Value = Value{Kind: "number", Number: &value}
			}
			s.record(work, observation)
		}
		pending = next
	}
	work.pending = nil
}

func (s *Service) record(work *runWork, observation Annotation) {
	observation.ScopeKind = ScopeRun
	observation.ScopeID = work.facts.RunID
	if observation.ObservedAt.IsZero() {
		observation.ObservedAt = s.now().UTC()
	}
	if len(observation.EvidenceEventIDs) == 0 {
		observation.EvidenceEventIDs = work.evidence[observation.LabelID]
	}
	if _, exists := work.observations[observation.LabelID]; !exists {
		work.order = append(work.order, observation.LabelID)
	}
	work.observations[observation.LabelID] = observation
	work.toSave = append(work.toSave, observation)
}

func (s *Service) save(ctx context.Context, work *runWork) {
	if len(work.toSave) == 0 {
		return
	}
	// Every answer in this work read the same bounded state, so it carries the
	// same fingerprint. Without it the cache could not tell a fresh answer from
	// one computed before the run appended more evidence.
	for index := range work.toSave {
		work.toSave[index].EvidenceHash = work.evidenceHash
	}
	// Save errors are surfaced by the view's note; the run is still readable
	// from what the evaluators produced. The caller sees the failure in Notes.
	if err := s.store.Save(ctx, work.toSave); err != nil {
		work.skipped = append(work.skipped, SkippedLabel{LabelID: "*", Reason: "persist_failed: " + err.Error()})
	}
}

func (s *Service) observation(label Label, hash string, state State, reason string) Annotation {
	return Annotation{
		ScopeKind: ScopeRun, ScopeID: "", LabelID: label.ID, DefinitionHash: hash,
		Method: string(label.Method), State: state, Reason: reason,
	}
}

// markLabelUnavailable records an unavailable state for a label on every work
// that needs it.
func (s *Service) markLabelUnavailable(works []*runWork, labelID, reason string) {
	for _, work := range works {
		if contains(work.needs, labelID) {
			s.markLabelUnavailableForWork(work, labelID, reason)
		}
	}
}

func (s *Service) markLabelUnavailableForWork(work *runWork, labelID, reason string) {
	compiled, ok := s.pack.Label(labelID)
	if !ok {
		return
	}
	observation := s.observation(compiled.label, compiled.hash, StateUnavailable, reason)
	observation.ObservedAt = s.now().UTC()
	s.record(work, observation)
	work.needs = remove(work.needs, labelID)
}

func (s *Service) markNeedsUnavailable(works []*runWork, reason string, filter ...func(string) bool) {
	allow := func(string) bool { return true }
	if len(filter) > 0 && filter[0] != nil {
		allow = filter[0]
	}
	for _, work := range works {
		for _, labelID := range append([]string(nil), work.needs...) {
			if allow(labelID) {
				s.markLabelUnavailableForWork(work, labelID, reason)
			}
		}
	}
}

func (s *Service) labelIndex(labelID string) int {
	for i, id := range s.pack.Order() {
		if id == labelID {
			return i
		}
	}
	return len(s.pack.Order())
}

func (s *Service) viewFor(work *runWork) RunView {
	view := RunView{
		RunID: work.facts.RunID, Pack: s.pack.Pack.Name, Version: s.pack.Pack.Version,
		PackHash: s.pack.PackHash, State: "evaluated", Skipped: work.skipped,
	}
	for _, labelID := range s.pack.Order() {
		observation, ok := work.observations[labelID]
		if !ok {
			if !skippedFor(work.skipped, labelID) {
				view.Missing = append(view.Missing, labelID)
			}
			continue
		}
		view.Labels = append(view.Labels, s.annotationView(observation))
		addCount(&view.Counts, observation.State)
	}
	view.Coverage = RunCoverage{Observed: view.Counts.Observed + view.Counts.Compatible, Labels: len(s.pack.Pack.Labels)}
	if len(view.Labels) == 0 {
		view.Notes = append(view.Notes, "no annotations recorded; unknown is not zero friction")
	}
	for _, skipped := range work.skipped {
		if skipped.LabelID == "*" && strings.HasPrefix(skipped.Reason, "persist_failed") {
			view.Notes = append(view.Notes, skipped.Reason)
		}
	}
	return view
}

// Run reads the latest annotations for one run without evaluating.
func (s *Service) Run(ctx context.Context, runID string) (RunView, error) {
	if s.features != nil {
		if _, err := s.features.Facts(ctx, ScopeRun, runID); err != nil {
			return RunView{}, err
		}
	}
	annotations, err := s.store.LatestForScope(ctx, ScopeRun, runID)
	if err != nil {
		return RunView{}, err
	}
	view := RunView{
		RunID: runID, Pack: s.pack.Pack.Name, Version: s.pack.Pack.Version,
		PackHash: s.pack.PackHash, State: "read",
	}
	for _, annotation := range annotations {
		view.Labels = append(view.Labels, s.annotationView(annotation))
		addCount(&view.Counts, annotation.State)
	}
	for _, label := range s.pack.Labels() {
		if !hasAnnotation(annotations, label.ID) {
			view.Missing = append(view.Missing, label.ID)
		}
	}
	view.Coverage = RunCoverage{Observed: view.Counts.Observed + view.Counts.Compatible, Labels: len(s.pack.Pack.Labels)}
	if len(view.Labels) == 0 {
		view.Notes = append(view.Notes, "no annotations recorded; unknown is not zero friction")
	}
	return view, nil
}

func (s *Service) annotationView(annotation Annotation) AnnotationView {
	view := AnnotationView{
		ID: annotation.LabelID, State: annotation.State, Method: coarseMethod(annotation.Method),
		EvaluatorMethod: annotation.Method, Value: annotation.Value, Probability: annotation.Probability,
		Confidence: annotation.Confidence, Model: annotation.Model, Provider: annotation.Provider,
		DefinitionHash: annotation.DefinitionHash, ObservedAt: annotation.ObservedAt,
		EvidenceEventIDs: annotation.EvidenceEventIDs, Reason: annotation.Reason,
	}
	if compiled, ok := s.pack.Label(annotation.LabelID); ok {
		view.Gate = gateStatus(compiled.label.Gate)
		view.GateState = compiled.label.Gate.State()
		view.Proven = compiled.label.Gate.Proven()
		view.Friction = compiled.label.Friction
		if bound, above, directional := frictionScoreBound(compiled.label); directional {
			if mass, ok := scoreRegionMass(annotation.Value, bound, above); ok {
				view.FrictionProbability = &mass
			}
		}
	}
	return view
}

func coarseMethod(method string) string {
	normalized := strings.ToLower(method)
	switch {
	case normalized == "deterministic", normalized == "signal", normalized == "composite":
		return normalized
	case strings.Contains(normalized, "decision"), strings.Contains(normalized, "llm"):
		return "decision"
	default:
		return normalized
	}
}

// Labels returns every label declaration with its coverage in the window.

func sampled(runID string, label Label) bool {
	rate := 1.0
	if label.Sampling.Rate != nil {
		rate = *label.Sampling.Rate
	}
	if rate >= 1 {
		return true
	}
	if rate <= 0 {
		return false
	}
	sum := sha256.Sum256([]byte(runID + "|" + label.ID))
	value := binary.BigEndian.Uint64(sum[:8])
	fraction := float64(value) / float64(^uint64(0))
	return fraction < rate
}

func optionLabels(options []Option) []string {
	out := make([]string, len(options))
	for i, option := range options {
		out[i] = option.Label
	}
	return out
}

func maxProbability(probabilities map[string]float64) (float64, bool) {
	maximum, found := 0.0, false
	for _, probability := range probabilities {
		if !found || probability > maximum {
			maximum, found = probability, true
		}
	}
	return maximum, found
}

func numericObservation(observation Annotation) (float64, bool) {
	if observation.State != StateObserved && observation.State != StateCompatible {
		return 0, false
	}
	if observation.Probability != nil {
		return *observation.Probability, true
	}
	if observation.Value.Noul != nil {
		return *observation.Value.Noul, true
	}
	if observation.Value.Score != nil {
		return *observation.Value.Score, true
	}
	if observation.Value.Number != nil {
		return *observation.Value.Number, true
	}
	if observation.Value.Bool != nil {
		if *observation.Value.Bool {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

func cloneActivation(activation map[string]any) map[string]any {
	out := make(map[string]any, len(activation)+2)
	for key, value := range activation {
		out[key] = value
	}
	return out
}

func chunkStrings(values []string, size int) [][]string {
	if size <= 0 {
		size = len(values)
	}
	var out [][]string
	for start := 0; start < len(values); start += size {
		end := min(start+size, len(values))
		out = append(out, values[start:end])
	}
	return out
}

func chunkWorks(values []*runWork, size int) [][]*runWork {
	if size <= 0 {
		size = len(values)
	}
	var out [][]*runWork
	for start := 0; start < len(values); start += size {
		end := min(start+size, len(values))
		out = append(out, values[start:end])
	}
	return out
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func remove(values []string, target string) []string {
	out := values[:0]
	for _, value := range values {
		if value != target {
			out = append(out, value)
		}
	}
	return out
}

func addCount(counts *StateCounts, state State) {
	switch state {
	case StateObserved:
		counts.Observed++
	case StateCompatible:
		counts.Compatible++
	case StateNotObserved:
		counts.NotObserved++
	case StateUnavailable:
		counts.Unavailable++
	}
}

func hasAnnotation(annotations []Annotation, labelID string) bool {
	for _, annotation := range annotations {
		if annotation.LabelID == labelID {
			return true
		}
	}
	return false
}

func skippedFor(skipped []SkippedLabel, labelID string) bool {
	for _, entry := range skipped {
		if entry.LabelID == labelID {
			return true
		}
	}
	return false
}

func shortHash(hash string) string {
	if len(hash) <= 12 {
		return hash
	}
	return hash[:12]
}

func boundIDs(ids []string, max int) []string {
	out := make([]string, 0, max)
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		out = append(out, id)
		if len(out) >= max {
			break
		}
	}
	return out
}

// evidenceIDsFor resolves declarative selectors to event ids. Selectors that
// name an aggregate (timings, workload, terminal) have no event id; the stored
// digest already carries their values.
func evidenceIDsFor(label Label, facts RunFacts) []string {
	var ids []string
	for _, selector := range label.Evidence {
		switch selector.Source {
		case "operator_messages":
			for _, message := range tailMessages(facts.OperatorMessages, 4) {
				ids = append(ids, message.EventID)
			}
		case "agent_text":
			for _, message := range tailMessages(facts.AgentText, 2) {
				ids = append(ids, message.EventID)
			}
		case "existing_signals":
			if selector.Label == "" {
				for _, signal := range facts.ExistingSignals {
					ids = append(ids, signal.LatestEventID)
				}
			} else if signal, ok := facts.ExistingSignals[selector.Label]; ok {
				ids = append(ids, signal.LatestEventID)
			}
		}
	}
	return boundIDs(ids, 8)
}
