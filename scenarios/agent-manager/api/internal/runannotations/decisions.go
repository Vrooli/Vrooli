package runannotations

import (
	"context"
	"sort"
	"strings"
	"time"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
)

// This file owns the decision-model path: grouping labels by the role they ask,
// batching units into one DecideBatch per role, splitting questions and
// shrinking the digest when the model overflows, retrying transport failures in
// smaller state groups, escalating only to a different model, and reading the
// typed answers back into observations. It is separate from the evaluator core
// so neither file grows past the scenario source limit.

func (s *Service) evaluateDecisions(ctx context.Context, works []*runWork) {
	if len(works) == 0 {
		return
	}
	if s.decisions == nil {
		s.markNeedsUnavailable(works, "decision_client_unavailable")
		return
	}
	remaining := works
	for attempt := 0; attempt < 2; attempt++ {
		overflowed := s.runDecisionPass(ctx, remaining)
		if len(overflowed) == 0 {
			return
		}
		if attempt == 1 {
			for _, work := range overflowed {
				for _, labelID := range append([]string(nil), work.needs...) {
					s.markLabelUnavailableForWork(work, labelID, "decision_context_overflow")
				}
			}
			return
		}
		remaining = overflowed
		for _, work := range remaining {
			work.digestShrunk = true
			work.overflowed = false
			if shrunk := s.shrinkDigest(work); shrunk != "" {
				work.digest = shrunk
			}
		}
	}
}

// shrinkDigest re-renders the bounded state below the size a model already
// refused, keeping the evidence the labels still needed declared as most
// important for as long as possible.
func (s *Service) shrinkDigest(work *runWork) string {
	if work.build == nil {
		return ""
	}
	size := work.sentChars
	if size == 0 {
		size = len(work.digest)
	}
	body, err := work.build(max(size*3/5, minDigestChars), s.digestPriority(work.needs))
	if err != nil {
		return ""
	}
	return body
}

// digestPriority is how the labels a call asks rank the digest sections: each
// label's evidence names its sections most important first, and labels asked
// together contribute in pack order. A section no asked label names is shrunk
// before any that one does.
func (s *Service) digestPriority(labelIDs []string) []string {
	asked := make(map[string]bool, len(labelIDs))
	for _, id := range labelIDs {
		asked[id] = true
	}
	var evidence []EvidenceSelector
	for _, id := range s.pack.Order() {
		if !asked[id] {
			continue
		}
		if compiled, ok := s.pack.Label(id); ok {
			evidence = append(evidence, compiled.label.Evidence...)
		}
	}
	return DigestPriority(evidence)
}

// runDecisionPass submits every pending label, one role at a time, and returns
// the works whose state the model could not read. It never marks an overflowed
// label unavailable: the caller decides whether a smaller digest is still
// allowed.
func (s *Service) runDecisionPass(ctx context.Context, works []*runWork) []*runWork {
	// The budget bounds decision states (units), not distinct labels: one state
	// carries every question asked about it. Counting label occurrences would
	// allow an unbounded number of runs whenever the pack has few labels.
	budget := s.pack.Pack.Runtime.Budgets.MaxDecisionStatesPerPass
	states := 0
	for _, work := range works {
		if len(work.needs) > 0 {
			states++
		}
	}
	if states > budget {
		allowed := 0
		for _, work := range works {
			if len(work.needs) == 0 {
				continue
			}
			if allowed < budget {
				allowed++
				continue
			}
			for _, labelID := range append([]string(nil), work.needs...) {
				s.markLabelUnavailableForWork(work, labelID, "decision_budget_exhausted")
			}
		}
	}
	var (
		needLabels []string
		needSet    = map[string]bool{}
	)
	for _, work := range works {
		for _, labelID := range work.needs {
			if !needSet[labelID] {
				needSet[labelID] = true
				needLabels = append(needLabels, labelID)
			}
		}
	}
	if len(needLabels) == 0 {
		return nil
	}
	sort.Slice(needLabels, func(i, j int) bool { return s.labelIndex(needLabels[i]) < s.labelIndex(needLabels[j]) })
	for _, work := range works {
		work.overflowed = false
		work.retryable = false
		work.transportRetried = false
	}
	pass := works[0].pass
	var overflowed []*runWork
	for _, group := range s.groupDecisionLabels(pass, needLabels) {
		limits := s.roleLimits(ctx, pass, group.role)
		questions := make(map[string]*inferencev1.DecisionQuestion, len(group.labels))
		var askable []string
		for _, labelID := range group.labels {
			compiled, ok := s.pack.Label(labelID)
			if !ok {
				continue
			}
			if exceedsLimits(compiled.label, limits) {
				s.markLabelUnavailable(works, labelID, "question_exceeds_role_limits: "+group.role)
				continue
			}
			question, err := questionFor(compiled.label)
			if err != nil {
				s.markLabelUnavailable(works, labelID, "question_invalid: "+err.Error())
				continue
			}
			questions[labelID] = question
			askable = append(askable, labelID)
		}
		for _, labelChunk := range chunkStrings(askable, s.questionCap(limits)) {
			overflowed = append(overflowed, s.submitQuestions(ctx, works, labelChunk, questions, group, limits)...)
		}
	}
	return dedupeWorks(overflowed)
}

// submitQuestions asks one label chunk and, when the model rejects the state as
// too large, halves the chunk and retries. Question instructions are part of
// the prompt, so a small-context model can overflow on the question set alone;
// shrinking the state would not help there. Splitting converges on the same
// one-question-per-call shape the intervention classifier uses, without
// hardcoding any model's window.
func (s *Service) submitQuestions(ctx context.Context, works []*runWork, labelChunk []string, questions map[string]*inferencev1.DecisionQuestion, group decisionGroup, limits ModelLimits) []*runWork {
	chunkSet := map[string]bool{}
	chunkQuestions := make([]*inferencev1.DecisionQuestion, 0, len(labelChunk))
	for _, labelID := range labelChunk {
		if question := questions[labelID]; question != nil {
			chunkSet[labelID] = true
			chunkQuestions = append(chunkQuestions, question)
		}
	}
	if len(chunkQuestions) == 0 {
		return nil
	}
	eligible := make([]*runWork, 0, len(works))
	for _, work := range works {
		for _, labelID := range work.needs {
			if chunkSet[labelID] {
				eligible = append(eligible, work)
				break
			}
		}
	}
	if len(eligible) == 0 {
		return nil
	}
	var overflowed, retryable []*runWork
	maxStates := s.pack.Pack.Runtime.Batch.MaxStates
	for _, stateChunk := range chunkWorks(eligible, maxStates) {
		for _, work := range s.decisionCall(ctx, stateChunk, chunkQuestions, chunkSet, group, limits) {
			overflowed = append(overflowed, work)
			if work.retryable {
				retryable = append(retryable, work)
			}
		}
	}
	// A transport failure is retried once in smaller state groups: one slow
	// item must not lose seven healthy answers. The second failure is reported.
	retryable = dedupeWorks(retryable)
	if len(retryable) > 0 {
		for _, work := range retryable {
			work.transportRetried = true
			work.retryable = false
		}
		for _, stateChunk := range chunkWorks(retryable, 4) {
			for _, work := range s.decisionCall(ctx, stateChunk, chunkQuestions, chunkSet, group, limits) {
				if work.overflowed {
					overflowed = append(overflowed, work)
				}
			}
		}
	}
	overflowed = dedupeWorks(overflowed)
	if len(overflowed) == 0 || len(labelChunk) < 2 {
		return overflowed
	}
	middle := len(labelChunk) / 2
	left := s.submitQuestions(ctx, overflowed, labelChunk[:middle], questions, group, limits)
	right := s.submitQuestions(ctx, overflowed, labelChunk[middle:], questions, group, limits)
	return dedupeWorks(append(left, right...))
}

func dedupeWorks(works []*runWork) []*runWork {
	if len(works) < 2 {
		return works
	}
	seen := make(map[*runWork]bool, len(works))
	unique := works[:0]
	for _, work := range works {
		if seen[work] {
			continue
		}
		seen[work] = true
		unique = append(unique, work)
	}
	return unique
}

// decideRequest builds one gateway call. Conversation content is confidential
// and local-only, and a pinned pass restricts the walk to its candidate.
func (s *Service) decideRequest(pass *passState, role string, questions []*inferencev1.DecisionQuestion, states []string) *inferencev1.DecideBatchRequest {
	contract := s.contract(role)
	items := make([]*inferencev1.DecideBatchItem, len(states))
	for index, state := range states {
		items[index] = &inferencev1.DecideBatchItem{StateJson: state}
	}
	request := &inferencev1.DecideBatchRequest{
		Role:              role,
		Questions:         questions,
		Items:             items,
		Profile:           contract.Profile,
		PrivacyClass:      contract.PrivacyClass,
		PrivacyProvenance: contract.PrivacySource,
	}
	if pass.pinned() {
		request.PinResourceRole = pass.pin
	}
	return request
}

func (s *Service) decisionCall(ctx context.Context, works []*runWork, questions []*inferencev1.DecisionQuestion, labelSet map[string]bool, group decisionGroup, limits ModelLimits) []*runWork {
	pass := works[0].pass
	budget := s.digestBudget(limits, questions)
	priority := s.digestPriority(sortedKeys(labelSet))
	states := make([]string, len(works))
	for i, work := range works {
		states[i] = work.stateFor(budget, priority)
	}
	response, err := s.decisions.DecideBatch(ctx, s.decideRequest(pass, group.role, questions, states))
	if err != nil {
		failed := make([]*runWork, 0, len(works))
		for _, work := range works {
			if !work.transportRetried {
				work.retryable = true
				failed = append(failed, work)
				continue
			}
			for labelID := range labelSet {
				if contains(work.needs, labelID) {
					s.markLabelUnavailableForWork(work, labelID, "decision_unavailable: "+err.Error())
				}
			}
		}
		return failed
	}
	if response == nil || len(response.Results) != len(works) {
		for _, work := range works {
			for labelID := range labelSet {
				if contains(work.needs, labelID) {
					s.markLabelUnavailableForWork(work, labelID, "decision_result_count_mismatch")
				}
			}
		}
		return nil
	}
	for i, work := range works {
		result := response.Results[i]
		// A refusal answers no question. Reporting it as a missing answer
		// hides the provider's actual reason, so the batch-level failure is
		// surfaced once for every label this call needed.
		if failure := decisionFailure(result); failure != "" {
			if decisionOverflowed(result) && !work.digestShrunk {
				work.overflowed = true
				continue
			}
			for labelID := range labelSet {
				if contains(work.needs, labelID) {
					s.markLabelUnavailableForWork(work, labelID, failure)
				}
			}
			continue
		}
		if !pass.pinned() {
			// A pinned answer says nothing about what the role resolves to.
			s.roles.note(group.role, result, s.now())
		}
		answers := map[string]*inferencev1.DecisionAnswer{}
		for _, answer := range result.GetAnswers() {
			answers[answer.GetQuestionId()] = answer
		}
		for labelID := range labelSet {
			compiled, ok := s.pack.Label(labelID)
			if !ok || !contains(work.needs, labelID) {
				continue
			}
			answer := answers[labelID]
			if answer == nil {
				s.markLabelUnavailableForWork(work, labelID, "decision_missing_answer")
				continue
			}
			observation := s.decisionObservation(compiled.label, compiled.hash, group.role, result, answer)
			if pass.pinned() && result.GetResourceRole() != pass.pin {
				// The gateway is contracted to answer a pinned request with
				// exactly the pinned candidate. An answer from anyone else must
				// not be stored as that candidate's.
				observation.State = StateUnavailable
				observation.Reason = "pin_mismatch: answered by " + result.GetResourceRole() + ", pinned " + pass.pin
			}
			s.record(work, observation)
			work.needs = remove(work.needs, labelID)
		}
	}
	if !pass.pinned() {
		s.evaluateEscalation(ctx, works, labelSet, group)
	}
	failed := make([]*runWork, 0)
	for _, work := range works {
		if work.overflowed || work.retryable {
			failed = append(failed, work)
		}
	}
	return failed
}

// evaluateEscalation re-asks only low-confidence answers, and only of a model
// that has not already answered. An escalation role that resolves to the same
// model as the first pass cannot say anything new, so the call is skipped and
// the fact is remembered for the pack's same_model_ttl; the saving is counted.
// A call that is made and answered by the same model teaches the engine the same
// thing, so it is made at most once per ttl.
func (s *Service) evaluateEscalation(ctx context.Context, works []*runWork, labelSet map[string]bool, group decisionGroup) {
	escalation := s.pack.Pack.Runtime.Escalation
	if !escalation.Enabled || group.escalate == "" || s.decisions == nil || len(works) == 0 {
		return
	}
	pass := works[0].pass
	low := map[string]bool{}
	var eligible []*runWork
	for _, work := range works {
		isLow := false
		for labelID := range labelSet {
			observation, ok := work.observations[labelID]
			if ok && observation.State == StateObserved && observation.Confidence != nil && *observation.Confidence < escalation.MinConfidence {
				low[labelID] = true
				isLow = true
			}
		}
		if isLow {
			eligible = append(eligible, work)
		}
	}
	if len(eligible) == 0 {
		return
	}
	maxStates := s.pack.Pack.Runtime.Batch.MaxStates
	escalationLimits := s.roleLimits(ctx, pass, group.escalate)
	labels := make([]string, 0, len(low))
	for labelID := range low {
		labels = append(labels, labelID)
	}
	sort.Slice(labels, func(i, j int) bool { return s.labelIndex(labels[i]) < s.labelIndex(labels[j]) })
	if len(labels) > escalation.MaxLabels {
		labels = labels[:escalation.MaxLabels]
	}
	questions := map[string]*inferencev1.DecisionQuestion{}
	var askable []string
	for _, labelID := range labels {
		compiled, ok := s.pack.Label(labelID)
		if !ok || exceedsLimits(compiled.label, escalationLimits) {
			continue
		}
		if question, err := questionFor(compiled.label); err == nil {
			questions[labelID] = question
			askable = append(askable, labelID)
		}
	}
	if len(askable) == 0 {
		return
	}
	labelChunks := chunkStrings(askable, s.questionCap(escalationLimits))
	stats := EscalationStats{EligibleStates: len(eligible)}
	ttl := escalation.SameModelDuration()
	if s.roles.sameModel(group.role, group.escalate, s.now(), ttl) {
		stats.SkippedStates = len(eligible)
		stats.SkippedCalls = len(chunkWorks(eligible, maxStates)) * len(labelChunks)
		pass.addEscalation(stats)
		return
	}
	sameModel := false
	for _, stateChunk := range chunkWorks(eligible, maxStates) {
		for _, labelChunk := range labelChunks {
			chunkQuestions := make([]*inferencev1.DecisionQuestion, 0, len(labelChunk))
			for _, labelID := range labelChunk {
				chunkQuestions = append(chunkQuestions, questions[labelID])
			}
			budget := s.digestBudget(escalationLimits, chunkQuestions)
			priority := s.digestPriority(labelChunk)
			states := make([]string, len(stateChunk))
			for index, work := range stateChunk {
				states[index] = work.stateFor(budget, priority)
			}
			started := time.Now()
			response, err := s.decisions.DecideBatch(ctx, s.decideRequest(pass, group.escalate, chunkQuestions, states))
			stats.Calls++
			stats.StatesAsked += len(stateChunk)
			stats.CallSeconds += time.Since(started).Seconds()
			if err != nil || response == nil || len(response.Results) != len(stateChunk) {
				continue
			}
			for index, work := range stateChunk {
				result := response.Results[index]
				if decisionFailure(result) != "" {
					continue
				}
				s.roles.note(group.escalate, result, s.now())
				answers := map[string]*inferencev1.DecisionAnswer{}
				for _, answer := range result.GetAnswers() {
					answers[answer.GetQuestionId()] = answer
				}
				askedSame := false
				for _, labelID := range labelChunk {
					existing, observed := work.observations[labelID]
					if !observed || existing.State != StateObserved || existing.Confidence == nil || *existing.Confidence >= escalation.MinConfidence {
						continue
					}
					if result.GetModel() == existing.Model && result.GetResourceRole() == existing.ModelVersion {
						// The same model answered again: nothing new, whatever
						// the escalation role is called.
						askedSame = true
						continue
					}
					answer := answers[labelID]
					compiled, ok := s.pack.Label(labelID)
					if answer == nil || !ok {
						continue
					}
					observation := s.decisionObservation(compiled.label, compiled.hash, group.escalate, result, answer)
					if observation.State != StateObserved || observation.Confidence == nil {
						continue
					}
					if existing.Confidence != nil && *observation.Confidence <= *existing.Confidence {
						continue
					}
					// Carry the prior evidence forward; the escalation answers the
					// same digest, not a different state.
					observation.EvidenceEventIDs = existing.EvidenceEventIDs
					s.record(work, observation)
					stats.Adopted++
				}
				if askedSame {
					stats.SameModelAnswers++
					sameModel = true
				}
			}
			if sameModel {
				break
			}
		}
		if sameModel {
			break
		}
	}
	if sameModel {
		s.roles.markFutile(group.role, group.escalate, s.now(), ttl)
	}
	pass.addEscalation(stats)
}

// decisionFailure describes a decision result that carries no usable answer
// set. An empty result with no error is still a failure: the caller asked for
// an answer and received none.
func decisionFailure(result *inferencev1.DecideResponse) string {
	if result == nil {
		return "decision_empty_result"
	}
	if failure := result.GetError(); failure != nil && failure.GetCode() != inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNSPECIFIED {
		code := strings.ToLower(strings.TrimPrefix(failure.GetCode().String(), "INFERENCE_ERROR_CODE_"))
		if message := strings.TrimSpace(failure.GetMessage()); message != "" {
			return code + ": " + message
		}
		return code
	}
	if len(result.GetAnswers()) == 0 {
		return "decision_empty_answers"
	}
	return ""
}

// decisionOverflowed reports a state no candidate could read whole. It mirrors
// the intervention classifier: a top-level overflow code, or a candidate walk
// that records context_overflow (or limits_exceeded: the request is outside the
// model's declared bounds, which splitting the questions or shrinking the state
// fixes the same way) while the fallback answered.
func decisionOverflowed(result *inferencev1.DecideResponse) bool {
	if result == nil {
		return false
	}
	if result.GetError().GetCode() == inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_CONTEXT_OVERFLOW {
		return true
	}
	for _, candidate := range result.GetCandidates() {
		if reason := candidate.GetReason(); reason == "context_overflow" || reason == "limits_exceeded" {
			return true
		}
	}
	return false
}

func (s *Service) decisionObservation(label Label, hash, requestedRole string, result *inferencev1.DecideResponse, answer *inferencev1.DecisionAnswer) Annotation {
	observation := s.observation(label, hash, StateObserved, "")
	observation.ObservedAt = s.now().UTC()
	observation.RequestedRole = requestedRole
	observation.Method = strings.TrimPrefix(result.GetMethod().String(), "DECISION_METHOD_")
	if result.GetMethod() == inferencev1.DecisionMethod_DECISION_METHOD_UNSPECIFIED {
		observation.Method = "decision"
	}
	observation.Provider = result.GetProvider()
	observation.Model = result.GetModel()
	observation.ModelVersion = result.GetResourceRole()
	if err := result.GetError(); err != nil && err.GetCode() != inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNSPECIFIED {
		observation.State = StateUnavailable
		observation.Reason = strings.ToLower(strings.TrimPrefix(err.GetCode().String(), "INFERENCE_ERROR_CODE_")) + ": " + err.GetMessage()
		return observation
	}
	observation.Probability = nil
	observation.Confidence = nil
	if result.GetMethod() == inferencev1.DecisionMethod_DECISION_METHOD_LLM_ENUM {
		// LLM_ENUM carries a label only: no calibrated distribution. Storing a
		// probability here would invent confidence the method does not report.
		observation.Value = valueFromEnum(label, answer)
		observation.Reason = "llm_enum_no_probabilities"
		return observation
	}
	probabilities := map[string]float64{}
	for _, probability := range answer.GetProbabilities() {
		probabilities[probability.GetOption()] = probability.GetProbability()
	}
	if answer.Confidence != nil {
		confidence := answer.GetConfidence()
		observation.Confidence = &confidence
	}
	switch label.Question.Type {
	case QuestionChoice:
		choice := answer.GetChoice()
		observation.Value = Value{Kind: "choice", Choice: choice, Probabilities: probabilities}
		if probability, ok := probabilities[choice]; ok {
			observation.Probability = &probability
		}
		observation.Value.Options = optionLabels(label.Question.Options)
	case QuestionScore:
		score := answer.GetScore()
		if answer.Score == nil {
			observation.State = StateUnavailable
			observation.Reason = "decision_missing_score"
			return observation
		}
		score += label.Question.Offset
		observation.Value = Value{Kind: "score", Score: &score, Levels: append([]string(nil), label.Question.Levels...), Probabilities: probabilities}
		observation.Value.Options = append([]string(nil), label.Question.Levels...)
		if maximum, ok := maxProbability(probabilities); ok {
			observation.Probability = &maximum
		}
	case QuestionNoul:
		if answer.Noul == nil {
			observation.State = StateUnavailable
			observation.Reason = "decision_missing_noul"
			return observation
		}
		noul := answer.GetNoul()
		observation.Value = Value{Kind: "noul", Noul: &noul}
		observation.Probability = &noul
	}
	return observation
}

func valueFromEnum(label Label, answer *inferencev1.DecisionAnswer) Value {
	switch label.Question.Type {
	case QuestionChoice:
		return Value{Kind: "choice", Choice: answer.GetChoice(), Options: optionLabels(label.Question.Options)}
	case QuestionScore:
		if answer.Score != nil {
			score := answer.GetScore() + label.Question.Offset
			return Value{Kind: "score", Score: &score, Levels: append([]string(nil), label.Question.Levels...)}
		}
		return Value{Kind: "score", Levels: append([]string(nil), label.Question.Levels...)}
	case QuestionNoul:
		if answer.Noul != nil {
			noul := answer.GetNoul()
			return Value{Kind: "noul", Noul: &noul}
		}
		return Value{Kind: "noul"}
	default:
		return Value{Kind: "unknown"}
	}
}
