package runannotations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// Evaluators for the cheap, non-decision methods and the shared cache rules.
// Decision labels are answered in shared batches by decisions.go.

// evaluateLabel runs the cheap path and records deferred work. Decision labels
// are answered in a shared batch after every run is prepared.
func (s *Service) evaluateLabel(ctx context.Context, work *runWork, compiled *compiledLabel, label Label, force bool) {
	now := s.now().UTC()
	if label.Method == MethodComposite {
		// A composite is a pure function of what the unit has already
		// observed, so it is recomputed every pass at no cost and written only
		// when its inputs changed (see evaluateComposites); a cached row could
		// not know that a message it summarizes was answered since.
		work.pending = append(work.pending, label.ID)
		return
	}
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
		if existing := s.cachedObservation(ctx, work, compiled, label); existing != nil {
			// Same definition and same evidence: reuse a settled answer without
			// inference. It is still an observed row; a replay must not make it
			// look newer.
			work.observations[label.ID] = *existing
			work.order = append(work.order, label.ID)
			return
		}
		if len(label.Compatibility.CompatibleWith) > 0 && !work.pass.pinned() {
			if prior, err := work.pass.store.Compatible(ctx, work.scopeKind, work.scopeID, label.ID, label.Compatibility.CompatibleWith); err == nil && prior != nil && prior.EvidenceHash == work.evidenceHash {
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
	case MethodExtract:
		s.record(work, s.observation(label, compiled.hash, StateUnavailable, "extract_not_implemented_in_slice"))
	default:
		s.record(work, s.observation(label, compiled.hash, StateUnavailable, "unknown_method"))
	}
}

// cachedObservation returns a stored answer that can stand for this unit and
// label, or nil. It must have been computed under the same definition from the
// same evidence, and a decision answer must also have been asked of the role the
// label routes to now: a label re-routed to another model is asked again rather
// than served the previous model's answer. A pinned pass reuses only the
// candidate it measures, from its own table.
func (s *Service) cachedObservation(ctx context.Context, work *runWork, compiled *compiledLabel, label Label) *Annotation {
	store := work.pass.store
	var (
		existing *Annotation
		err      error
	)
	if work.pass.pinned() {
		existing, err = store.CurrentFor(ctx, work.scopeKind, work.scopeID, label.ID, compiled.hash, work.pass.pin)
	} else {
		existing, err = store.Current(ctx, work.scopeKind, work.scopeID, label.ID, compiled.hash)
	}
	if err != nil || !reusableObservation(existing) || existing.EvidenceHash != work.evidenceHash {
		return nil
	}
	if !work.pass.pinned() && !s.roleCurrent(compiled, existing) {
		return nil
	}
	return existing
}

// roleCurrent reports whether a stored answer was asked of the role the label
// routes to. Only a model's answer depends on the role; a rule or signal row
// does not. A row written before the role was recorded was asked of the pack
// default.
func (s *Service) roleCurrent(compiled *compiledLabel, existing *Annotation) bool {
	if isRuleMethod(existing.Method) {
		return true
	}
	asked := existing.RequestedRole
	if asked == "" {
		asked = s.pack.Pack.Runtime.DefaultRole
	}
	// An answer that was escalated was asked of the label's escalation role.
	return asked == s.labelRole(compiled.label) || asked == s.labelEscalateRole(compiled.label)
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
	if force || work.pass.pinned() {
		return
	}
	if prior, err := work.pass.store.Current(ctx, work.scopeKind, work.scopeID, label.ID, hash); err == nil && prior != nil {
		s.record(work, s.observation(label, hash, StateNotObserved, reason))
	}
}

// reusableObservation reports whether a stored row is settled enough to skip
// re-evaluation. An unavailable row is not: a transient gateway failure or a
// busy lock must not freeze a label for that run forever. Observed,
// compatible and deterministic not-observed answers are stable for the
// definition hash.
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

// evaluateComposites computes the composite labels of one unit, each after the
// composites it reads. A composite reads the unit's own observations
// (labels['x']) and, for a run, a summary of its messages' observations
// (message_labels['x']), so the messages of a run are evaluated first. The row
// it writes is fingerprinted by the facts and by exactly the inputs the rule
// names; a changed or new input observation changes the fingerprint, and an
// unchanged one writes nothing.
func (s *Service) evaluateComposites(ctx context.Context, work *runWork) {
	if len(work.pending) == 0 {
		return
	}
	pending := map[string]bool{}
	for _, id := range work.pending {
		pending[id] = true
	}
	work.pending = nil
	messages := s.messageObservations(work)
	for _, labelID := range s.pack.compositeOrder(work.scopeKind) {
		if !pending[labelID] {
			continue
		}
		compiled, _ := s.pack.Label(labelID)
		labels := map[string]float64{}
		present := map[string]bool{}
		for id, observation := range work.observations {
			if number, ok := numericObservation(observation); ok {
				labels[id], present[id] = number, true
			}
		}
		stats := map[string]map[string]float64{}
		summaries := map[string]MessageStats{}
		for _, id := range compiled.refs.Messages {
			if input, ok := s.pack.Label(id); ok {
				if summary, ok := s.pack.messageStats(input, messages[id]); ok {
					stats[id], summaries[id] = summary.activation(), summary
				}
			}
		}
		activation := cloneActivation(work.activation)
		activation["labels"], activation["present"], activation["message_labels"] = labels, present, stats
		var inputs []InputKey
		for _, id := range compiled.refs.Labels {
			if observation, ok := work.observations[id]; ok && (observation.State == StateObserved || observation.State == StateCompatible) {
				inputs = mergeInputs(inputs, leafKeys(id, observation)...)
			}
		}
		for _, id := range compiled.refs.Messages {
			for _, observation := range messages[id] {
				if observation.State == StateObserved || observation.State == StateCompatible {
					inputs = mergeInputs(inputs, leafKeys(id, observation)...)
				}
			}
		}
		fingerprint := compositeFingerprint(work.evidenceHash, compiled.refs, labels, present, summaries, inputs)
		value, err := s.cel.evalNumber(compiled.composite, activation)
		if err != nil {
			s.recordComposite(ctx, work, s.observation(compiled.label, compiled.hash, StateNotObserved, "composite_inputs_missing: "+err.Error()), fingerprint)
			continue
		}
		observation := s.observation(compiled.label, compiled.hash, StateObserved, "")
		observation.Method = string(MethodComposite)
		observation.ObservedAt = s.now().UTC()
		observation.Value.Inputs = inputs
		questionType := QuestionType("")
		if compiled.label.Question != nil {
			questionType = compiled.label.Question.Type
		}
		switch questionType {
		case QuestionNoul:
			clamped := math.Max(0, math.Min(1, value))
			observation.Value.Kind, observation.Value.Noul = "noul", &clamped
			observation.Probability = &clamped
		case QuestionScore:
			observation.Value.Kind, observation.Value.Score = "score", &value
			observation.Value.Levels = append([]string(nil), compiled.label.Question.Levels...)
		default:
			observation.Value.Kind, observation.Value.Number = "number", &value
		}
		s.recordComposite(ctx, work, observation, fingerprint)
	}
}

// recordComposite stores a composite observation unless the stored one was
// computed from the same inputs, in which case the stored one stands and a
// replay does not look newer.
func (s *Service) recordComposite(ctx context.Context, work *runWork, observation Annotation, fingerprint string) {
	observation.EvidenceHash = fingerprint
	if !work.forces(observation.LabelID) && !work.pass.pinned() {
		prior, err := work.pass.store.Current(ctx, work.scopeKind, work.scopeID, observation.LabelID, observation.DefinitionHash)
		if err == nil && reusableObservation(prior) && prior.EvidenceHash == fingerprint {
			if _, exists := work.observations[observation.LabelID]; !exists {
				work.order = append(work.order, observation.LabelID)
			}
			work.observations[observation.LabelID] = *prior
			return
		}
	}
	s.record(work, observation)
}

// messageObservations are the observations of a run's messages by label, as
// this pass holds them (fresh answers and reused ones alike). A message unit has
// none.
func (s *Service) messageObservations(work *runWork) map[string][]Annotation {
	out := map[string][]Annotation{}
	for _, child := range work.children {
		for id, observation := range child.observations {
			out[id] = append(out[id], observation)
		}
	}
	return out
}

// compositeFingerprint identifies everything a composite's value is computed
// from: the unit's facts, the value of each label the rule reads, the summary of
// each message label it reads, and the proof keys of the observations behind
// them (a model change alters a derived proof though not the value).
func compositeFingerprint(evidence string, refs compositeRefs, labels map[string]float64, present map[string]bool, summaries map[string]MessageStats, inputs []InputKey) string {
	var parts []string
	for _, id := range refs.Labels {
		if present[id] {
			parts = append(parts, "label "+id+"="+strconv.FormatFloat(labels[id], 'g', -1, 64))
		} else {
			parts = append(parts, "label "+id+" absent")
		}
	}
	for _, id := range refs.Messages {
		summary, ok := summaries[id]
		if !ok {
			parts = append(parts, "messages "+id+" absent")
			continue
		}
		parts = append(parts, fmt.Sprintf("messages %s n=%d pos=%d max=%s mean=%s", id, summary.Count, summary.Positives,
			strconv.FormatFloat(summary.Max, 'g', -1, 64), strconv.FormatFloat(summary.Mean, 'g', -1, 64)))
	}
	for _, key := range inputs {
		parts = append(parts, "input "+key.Label+" "+key.DefinitionHash+" "+key.Model+" "+key.ResourceRole)
	}
	sum := sha256.Sum256([]byte(evidence + "\n" + strings.Join(parts, "\n")))
	return hex.EncodeToString(sum[:])
}
