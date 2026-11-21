package runannotations

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// This file orchestrates an evaluation pass: selecting runs, preparing each run
// (and each human message of it) into a unit of work, answering decisions in
// shared batches, and persisting finished groups.

// Annotate evaluates one run and persists new observations.
func (s *Service) Annotate(ctx context.Context, runID string, opts AnnotateOptions) (RunView, error) {
	views, errs, _, _ := s.annotateRunsBounded(ctx, []string{runID}, opts, time.Time{})
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
	explicit := len(opts.RunIDs) > 0
	if limit <= 0 {
		limit = min(20, budget)
		if explicit {
			limit = min(len(opts.RunIDs), budget)
		}
	}
	if limit > budget {
		limit = budget
	}
	until := opts.Until
	if until.IsZero() {
		until = s.now()
	}
	runIDs := append([]string(nil), opts.RunIDs...)
	if !explicit {
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
	// Explicit ids past the pass budget are handed back, never dropped: a caller
	// measuring a gold set must be able to continue until every run is answered.
	var deferred []string
	if len(runIDs) > limit {
		deferred = runIDs[limit:]
		runIDs = runIDs[:limit]
	}
	// Bound wall-clock work so one large backfill cannot be cancelled by a
	// client or gateway timeout after doing everything and saving nothing.
	// Groups are persisted as they finish, and the remainder is reported.
	deadline := time.Time{}
	if budget := s.passBudget(opts.TimeBudget); budget > 0 {
		deadline = time.Now().Add(budget)
	}
	views, errs, remaining, pass := s.annotateRunsBounded(ctx, runIDs, AnnotateOptions{
		Force: opts.Force, IncludeOptional: opts.IncludeOptional,
		Labels: opts.Labels, DryRun: opts.DryRun,
		PinResourceRole: opts.PinResourceRole, Role: opts.Role, MessageIDs: opts.MessageIDs,
	}, deadline)
	result := EvaluateResult{
		Window:    windowRange(opts.Since, until),
		Submitted: len(runIDs) - len(remaining),
		Remaining: len(remaining) + len(deferred),
		Truncated: len(remaining)+len(deferred) > 0,
		Runs:      views,
		Errors:    errs,
		Pinned:    pass.pin,
	}
	if explicit {
		result.RemainingRunIDs = append(append([]string(nil), remaining...), deferred...)
	}
	if stats := pass.escalationStats(); stats != (EscalationStats{}) {
		result.Escalation = &stats
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

// newPass resolves where a pass writes. A pinned pass writes the trials table so
// a candidate's answers can never overwrite or mix with production ones.
func (s *Service) newPass(opts AnnotateOptions) *passState {
	pass := &passState{store: s.store}
	if pin := strings.TrimSpace(opts.PinResourceRole); pin != "" {
		pass.store = s.store.Trials()
		pass.pin = pin
		pass.role = strings.TrimSpace(opts.Role)
		if pass.role == "" {
			pass.role = strings.TrimSpace(s.pack.Pack.Runtime.BakeoffRole)
		}
	}
	return pass
}

// annotateRunsBounded evaluates runs in decision-batch-sized groups and saves
// each group before starting the next. A deadline stops the pass between
// groups, returning the run IDs it did not reach. Grouping matters for more
// than latency: a failure or cancellation now costs at most one group instead
// of discarding every answered run in the pass.
func (s *Service) annotateRunsBounded(ctx context.Context, runIDs []string, opts AnnotateOptions, deadline time.Time) ([]RunView, []RunError, []string, *passState) {
	pass := s.newPass(opts)
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
			return views, errs, runIDs[start:], pass
		}
		var works []*runWork
		for _, runID := range runIDs[start:end] {
			prepared, err := s.prepare(ctx, runID, opts, pass)
			if err != nil {
				errs = append(errs, RunError{RunID: runID, Reason: err.Error()})
				continue
			}
			works = append(works, prepared...)
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
		// A run's composites read its messages' observations, so the messages
		// are finished first.
		for _, scope := range []Scope{ScopeMessage, ScopeRun} {
			for _, work := range works {
				if work.scopeKind == scope {
					s.evaluateComposites(ctx, work)
				}
			}
		}
		for _, work := range works {
			if !opts.DryRun {
				s.save(ctx, work)
				s.saveProvenance(ctx, work)
			}
		}
		byRun := map[string]int{}
		for _, work := range works {
			if work.scopeKind != ScopeRun {
				continue
			}
			byRun[work.runID] = len(views)
			views = append(views, s.viewFor(work))
		}
		for _, work := range works {
			if work.scopeKind != ScopeMessage {
				continue
			}
			if index, ok := byRun[work.runID]; ok {
				views[index].Messages = append(views[index].Messages, s.messageView(work))
			}
		}
	}
	return views, errs, nil, pass
}

// prepare reads one run and builds its units of work: the run itself, then one
// per human operator message when the pack has message-scope labels. Reading is
// shared, so message labels cost no extra database work.
func (s *Service) prepare(ctx context.Context, runID string, opts AnnotateOptions, pass *passState) ([]*runWork, error) {
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
	s.pack.Pack.Runtime.Provenance.Apply(&facts)
	features := FeaturesFromFacts(facts)
	build := func(limit int, priority []string) (string, error) { return BuildDigestBounded(facts, limit, priority) }
	digest, err := build(s.digestLimit, nil)
	if err != nil {
		// A digest that cannot be bounded is a configuration error, not an
		// annotation. Refuse rather than send a partial transcript.
		return nil, err
	}
	work := &runWork{
		scopeKind: ScopeRun, scopeID: runID, runID: facts.RunID, pass: pass, force: opts.Force,
		facts: facts, features: features, digest: digest, build: build,
		evidenceHash: evidenceFingerprint(digest, facts),
		activation:   features.Activation(),
		observations: map[string]Annotation{}, evidence: map[string][]string{},
	}
	s.prepareLabels(ctx, work, opts)
	works := []*runWork{work}
	works = append(works, s.messageWorks(ctx, work, opts)...)
	return works, nil
}

// prepareLabels decides, for each label of the unit's scope, whether it is asked
// at all and queues the cheap or deferred evaluation. The guards are the label's
// own data: the requested set, optional, a candidate rule, a require predicate
// and sampling.
func (s *Service) prepareLabels(ctx context.Context, work *runWork, opts AnnotateOptions) {
	pinned := work.pass.pinned()
	allowed := s.pack.requested(opts.Labels)
	maxLabels := opts.MaxLabels
	if maxLabels <= 0 {
		maxLabels = DefaultMaxRunAnnotation
	}
	count := 0
	for _, label := range s.pack.Labels() {
		if label.Scope != work.scopeKind {
			if work.scopeKind == ScopeRun && label.Scope == ScopeEpisode {
				work.skipped = append(work.skipped, SkippedLabel{LabelID: label.ID, Reason: "scope_not_supported_in_slice"})
			}
			continue
		}
		if allowed != nil && !allowed[label.ID] {
			work.skipped = append(work.skipped, SkippedLabel{LabelID: label.ID, Reason: "not_requested"})
			continue
		}
		if label.Optional && !opts.IncludeOptional {
			work.skipped = append(work.skipped, SkippedLabel{LabelID: label.ID, Reason: "optional_not_requested"})
			continue
		}
		if pinned && label.Method != MethodDecision {
			// A rule, a signal or a composite does not depend on the model a
			// bake-off pins; re-running it would only duplicate production.
			work.skipped = append(work.skipped, SkippedLabel{LabelID: label.ID, Reason: "pinned_not_decision"})
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
		// A pinned pass names its runs, so a candidate rule (which selects useful
		// work) and sampling (which thins it) do not apply; the evidence guard
		// still does, because an empty state has nothing to ask about.
		if compiled.candidates != nil && !pinned {
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
				// The unit has nothing this label can judge. Asking a model
				// about an empty digest returns a plausible-looking number,
				// which is worse than no observation.
				work.skipped = append(work.skipped, SkippedLabel{LabelID: label.ID, Reason: "insufficient_evidence"})
				s.supersedeIfAnswered(ctx, work, label, compiled.hash, "insufficient_evidence", opts.Force)
				continue
			}
		}
		if !pinned && !sampled(work.scopeID, label) {
			work.skipped = append(work.skipped, SkippedLabel{LabelID: label.ID, Reason: "sampled_out"})
			continue
		}
		count++
		if work.subjectEvidence != nil {
			work.evidence[label.ID] = work.subjectEvidence
		} else {
			work.evidence[label.ID] = evidenceIDsFor(label, work.facts)
		}
		s.evaluateLabel(ctx, work, compiled, label, opts.Force)
	}
}
