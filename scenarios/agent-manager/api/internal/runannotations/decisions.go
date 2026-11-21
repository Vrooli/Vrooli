package runannotations

import (
	"context"
	"sort"
	"strings"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
)

// This file owns the decision-model path: batching runs into one DecideBatch,
// splitting questions and shrinking the digest when the model overflows,
// retrying transport failures in smaller state groups, and reading the typed
// answers back into observations. It is separate from the evaluator core so
// neither file grows past the scenario source limit.

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
			if shrunk := shrinkDigest(work.facts, len(work.digest)); shrunk != "" {
				work.digest = shrunk
			}
		}
	}
}

// shrinkDigest re-renders the bounded digest below the size a model already
// refused. The ladder keeps operator text longer than agent text, so affect
// labels retain the words they need for as long as possible.
func shrinkDigest(facts RunFacts, current int) string {
	target := current * 3 / 5
	if target < 400 {
		target = 400
	}
	body, err := BuildDigestBounded(facts, target)
	if err != nil {
		return body
	}
	return body
}

// runDecisionPass submits every pending label and returns the works whose
// state the model could not read. It never marks an overflowed label
// unavailable: the caller decides whether a smaller digest is still allowed.
func (s *Service) runDecisionPass(ctx context.Context, works []*runWork) []*runWork {
	// The budget bounds decision states (runs), not distinct labels: one
	// state carries every question asked about it. Counting label occurrences
	// would allow an unbounded number of runs whenever the pack has few labels.
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
	questions := make(map[string]*inferencev1.DecisionQuestion, len(needLabels))
	for _, labelID := range needLabels {
		compiled, ok := s.pack.Label(labelID)
		if !ok {
			continue
		}
		question, err := questionFor(compiled.label)
		if err != nil {
			s.markLabelUnavailable(works, labelID, "question_invalid: "+err.Error())
			continue
		}
		questions[labelID] = question
	}
	if len(questions) == 0 {
		return nil
	}
	for _, work := range works {
		work.overflowed = false
		work.retryable = false
		work.transportRetried = false
	}
	maxQuestions := s.pack.Pack.Runtime.Batch.MaxQuestions
	var overflowed []*runWork
	for _, labelChunk := range chunkStrings(needLabels, maxQuestions) {
		overflowed = append(overflowed, s.submitQuestions(ctx, works, labelChunk, questions, s.pack.Pack.Runtime.DefaultRole)...)
	}
	return dedupeWorks(overflowed)
}

// submitQuestions asks one label chunk and, when the model rejects the state as
// too large, halves the chunk and retries. Question instructions are part of
// the prompt, so a small-context model can overflow on the question set alone;
// shrinking the state would not help there. Splitting converges on the same
// one-question-per-call shape the intervention classifier uses, without
// hardcoding any model's window.
func (s *Service) submitQuestions(ctx context.Context, works []*runWork, labelChunk []string, questions map[string]*inferencev1.DecisionQuestion, role string) []*runWork {
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
		for _, work := range s.decisionCall(ctx, stateChunk, chunkQuestions, chunkSet, role, false) {
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
			for _, work := range s.decisionCall(ctx, stateChunk, chunkQuestions, chunkSet, role, false) {
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
	left := s.submitQuestions(ctx, overflowed, labelChunk[:middle], questions, role)
	right := s.submitQuestions(ctx, overflowed, labelChunk[middle:], questions, role)
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

// evaluateEscalation re-asks only low-confidence answers with escalate_role.
func (s *Service) evaluateEscalation(ctx context.Context, works []*runWork, questionSet map[string]bool) {
	role := s.pack.Pack.Runtime.EscalateRole
	escalation := s.pack.Pack.Runtime.Escalation
	if !escalation.Enabled || role == "" || s.decisions == nil {
		return
	}
	var (
		eligible []*runWork
		labels   []string
	)
	for _, work := range works {
		low := false
		for labelID := range questionSet {
			observation, ok := work.observations[labelID]
			if !ok || observation.State != StateObserved || observation.Confidence == nil {
				continue
			}
			if *observation.Confidence < escalation.MinConfidence {
				low = true
			}
		}
		if low {
			eligible = append(eligible, work)
		}
	}
	for labelID := range questionSet {
		labels = append(labels, labelID)
	}
	sort.Slice(labels, func(i, j int) bool { return s.labelIndex(labels[i]) < s.labelIndex(labels[j]) })
	if len(labels) > escalation.MaxLabels {
		labels = labels[:escalation.MaxLabels]
	}
	questions := make([]*inferencev1.DecisionQuestion, 0, len(labels))
	chunkSet := map[string]bool{}
	for _, labelID := range labels {
		compiled, ok := s.pack.Label(labelID)
		if !ok {
			continue
		}
		question, err := questionFor(compiled.label)
		if err != nil {
			continue
		}
		questions = append(questions, question)
		chunkSet[labelID] = true
	}
	if len(questions) == 0 {
		return
	}
	for _, chunk := range chunkWorks(eligible, s.pack.Pack.Runtime.Batch.MaxStates) {
		s.decisionCall(ctx, chunk, questions, chunkSet, role, true)
	}
}

func (s *Service) decisionCall(ctx context.Context, works []*runWork, questions []*inferencev1.DecisionQuestion, labelSet map[string]bool, role string, escalationPass bool) []*runWork {
	items := make([]*inferencev1.DecideBatchItem, len(works))
	for i, work := range works {
		items[i] = &inferencev1.DecideBatchItem{StateJson: work.digest}
	}
	contract := s.contract(role)
	request := &inferencev1.DecideBatchRequest{
		Role:              role,
		Questions:         questions,
		Items:             items,
		Profile:           contract.Profile,
		PrivacyClass:      contract.PrivacyClass,
		PrivacyProvenance: contract.PrivacySource,
	}
	response, err := s.decisions.DecideBatch(ctx, request)
	if err != nil {
		if escalationPass {
			return nil
		}
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
	if response == nil || len(response.Results) != len(items) {
		if !escalationPass {
			for _, work := range works {
				for labelID := range labelSet {
					if contains(work.needs, labelID) {
						s.markLabelUnavailableForWork(work, labelID, "decision_result_count_mismatch")
					}
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
			if !escalationPass {
				if decisionOverflowed(result) && !work.digestShrunk {
					work.overflowed = true
					continue
				}
				for labelID := range labelSet {
					if contains(work.needs, labelID) {
						s.markLabelUnavailableForWork(work, labelID, failure)
					}
				}
			}
			continue
		}
		answers := map[string]*inferencev1.DecisionAnswer{}
		for _, answer := range result.GetAnswers() {
			answers[answer.GetQuestionId()] = answer
		}
		for labelID := range labelSet {
			compiled, ok := s.pack.Label(labelID)
			if !ok {
				continue
			}
			if escalationPass {
				existing, observed := work.observations[labelID]
				if !observed || existing.State != StateObserved || existing.Confidence == nil {
					continue
				}
				if *existing.Confidence >= s.pack.Pack.Runtime.Escalation.MinConfidence {
					continue
				}
			} else if !contains(work.needs, labelID) {
				continue
			}
			answer := answers[labelID]
			if answer == nil {
				if !escalationPass {
					s.markLabelUnavailableForWork(work, labelID, "decision_missing_answer")
				}
				continue
			}
			observation := s.decisionObservation(compiled.label, compiled.hash, result, answer)
			if escalationPass {
				existing := work.observations[labelID]
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
				continue
			}
			s.record(work, observation)
			work.needs = remove(work.needs, labelID)
		}
	}
	if !escalationPass {
		s.evaluateEscalation(ctx, works, labelSet)
	}
	failed := make([]*runWork, 0)
	for _, work := range works {
		if work.overflowed || work.retryable {
			failed = append(failed, work)
		}
	}
	return failed
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
// that records context_overflow while the fallback answered.
func decisionOverflowed(result *inferencev1.DecideResponse) bool {
	if result == nil {
		return false
	}
	if result.GetError().GetCode() == inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_CONTEXT_OVERFLOW {
		return true
	}
	for _, candidate := range result.GetCandidates() {
		if candidate.GetReason() == "context_overflow" {
			return true
		}
	}
	return false
}

func (s *Service) decisionObservation(label Label, hash string, result *inferencev1.DecideResponse, answer *inferencev1.DecisionAnswer) Annotation {
	observation := s.observation(label, hash, StateObserved, "")
	observation.ObservedAt = s.now().UTC()
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
