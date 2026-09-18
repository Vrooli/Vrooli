package planning

import (
	"errors"
	"fmt"
	"sort"
)

const ObjectiveVersion = "normalized-loss-v1"

type ObjectiveWeights struct{ Cost, Effort, Variety float64 }
type Losses struct{ Cost, Effort, Repetition float64 }

// Score uses fixed normalized loss inputs and normalizes only the configured
// preference weights. Candidate pools never change the metric scales.
func Score(loss Losses, weights ObjectiveWeights) (float64, error) {
	denominator := weights.Cost + weights.Effort + weights.Variety
	if denominator <= 0 {
		return 0, errors.New("objective weights must have a positive sum")
	}
	return (weights.Cost*loss.Cost + weights.Effort*loss.Effort + weights.Variety*loss.Repetition) / denominator, nil
}

type ScoredCandidate struct {
	Candidate Candidate
	Score     float64
	Reason    string
}

func RankCandidates(candidates []Candidate, weights ObjectiveWeights) ([]ScoredCandidate, error) {
	if weights.Cost+weights.Effort+weights.Variety <= 0 {
		weights = ObjectiveWeights{Cost: 1, Effort: 1, Variety: 1}
	}
	out := make([]ScoredCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if !candidate.Eligible {
			continue
		}
		score, err := Score(Losses{Cost: candidate.Cost, Effort: candidate.Effort, Repetition: candidate.Repetition}, weights)
		if err != nil {
			return nil, err
		}
		out = append(out, ScoredCandidate{Candidate: candidate, Score: score, Reason: fmt.Sprintf("objective %s: normalized cost %.3f, effort %.3f, repetition %.3f", ObjectiveVersion, candidate.Cost, candidate.Effort, candidate.Repetition)})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score == out[j].Score {
			return out[i].Candidate.ID < out[j].Candidate.ID
		}
		return out[i].Score < out[j].Score
	})
	return out, nil
}

type PlanEvaluation struct {
	Complete         bool     `json:"complete"`
	Score            float64  `json:"score"`
	ObjectiveVersion string   `json:"objectiveVersion"`
	UnresolvedSlots  []string `json:"unresolvedSlots,omitempty"`
	Reasons          []string `json:"reasons,omitempty"`
}

// EvaluatePlan produces a bounded, honest plan-level score. Missing metrics or
// unresolved slots make completeness false; they are never treated as zero.
func EvaluatePlan(draft Draft, metrics map[string]Losses, weights ObjectiveWeights) (PlanEvaluation, error) {
	if len(draft.Occurrences) == 0 && len(draft.Unresolved) == 0 {
		return PlanEvaluation{}, errors.New("plan has no slots")
	}
	if weights.Cost+weights.Effort+weights.Variety <= 0 {
		weights = ObjectiveWeights{Cost: 1, Effort: 1, Variety: 1}
	}
	result := PlanEvaluation{Complete: len(draft.Unresolved) == 0, ObjectiveVersion: ObjectiveVersion}
	var total float64
	count := 0
	for _, occurrence := range draft.Occurrences {
		loss, ok := metrics[occurrence.RecipeID]
		if !ok {
			result.Complete = false
			result.UnresolvedSlots = append(result.UnresolvedSlots, occurrence.Date)
			continue
		}
		score, err := Score(loss, weights)
		if err != nil {
			return PlanEvaluation{}, err
		}
		total += score
		count++
	}
	for _, unresolved := range draft.Unresolved {
		result.UnresolvedSlots = append(result.UnresolvedSlots, unresolved.Date)
		result.Reasons = append(result.Reasons, unresolved.Code+": "+unresolved.Message)
	}
	if count > 0 {
		result.Score = total / float64(count)
	}
	return result, nil
}
