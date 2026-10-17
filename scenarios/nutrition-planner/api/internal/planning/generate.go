package planning

import (
	"fmt"
	"math/rand"
)

func Generate(in GenerateInput) Draft {
	result := Draft{InputReferences: append([]string(nil), in.InputReferences...), Seed: in.Seed, RunID: fmt.Sprintf("seed-%d", in.Seed), ObjectiveVersion: ObjectiveVersion}
	eligible := make([]Candidate, 0, len(in.Candidates))
	byID := map[string]Candidate{}
	for _, candidate := range in.Candidates {
		byID[candidate.ID] = candidate
		if candidate.Eligible {
			eligible = append(eligible, candidate)
		}
	}
	rng := rand.New(rand.NewSource(in.Seed))
	scored, err := RankCandidates(eligible, ObjectiveWeights{Cost: in.CostWeight, Effort: in.EffortWeight, Variety: in.RepetitionWeight})
	if err != nil {
		return result
	}
	eligible = eligible[:0]
	for _, item := range scored {
		eligible = append(eligible, item.Candidate)
	}
	if len(eligible) > 1 && scored[0].Score == scored[1].Score && rng.Intn(2) == 1 {
		eligible[0], eligible[1] = eligible[1], eligible[0]
	}
	used := map[string]bool{}
	for _, slot := range in.Slots {
		if slot.Mode == "open" || slot.Mode == "social" {
			result.Occurrences = append(result.Occurrences, Occurrence{Date: slot.Date, SlotName: slot.SlotName, Mode: slot.Mode, Quantity: slot.Quantity, Reason: "Open slot remains user-controlled; no recipe was inferred"})
			continue
		}
		if slot.Locked {
			candidate, ok := byID[slot.RecipeID]
			if !ok || !candidate.Eligible {
				result.Unresolved = append(result.Unresolved, Unresolved{Date: slot.Date, Code: "locked_recipe_ineligible", Message: "locked occurrence no longer satisfies active rules", CandidateIDs: []string{slot.RecipeID}})
				continue
			}
			result.Occurrences = append(result.Occurrences, Occurrence{Date: slot.Date, SlotName: slot.SlotName, Mode: slot.Mode, Quantity: slot.Quantity, RecipeID: candidate.ID, RecipeName: candidate.Name, Reason: "Preserved locked occurrence", Locked: true})
			used[candidate.ID] = true
			continue
		}
		var chosen *Candidate
		for i := range eligible {
			if !used[eligible[i].ID] {
				chosen = &eligible[i]
				break
			}
		}
		if chosen == nil && len(eligible) > 0 {
			chosen = &eligible[0]
		}
		if chosen == nil {
			result.Unresolved = append(result.Unresolved, Unresolved{Date: slot.Date, Code: "no_eligible_meal", Message: "No eligible meal was found without relaxing required rules."})
			continue
		}
		result.Occurrences = append(result.Occurrences, Occurrence{Date: slot.Date, SlotName: slot.SlotName, Mode: slot.Mode, Quantity: slot.Quantity, RecipeID: chosen.ID, RecipeName: chosen.Name, Reason: "Lowest weighted known cost, effort, and repetition score", Locked: false})
		used[chosen.ID] = true
	}
	metrics := make(map[string]Losses, len(in.Candidates))
	for _, candidate := range in.Candidates {
		metrics[candidate.ID] = Losses{Cost: candidate.Cost, Effort: candidate.Effort, Repetition: candidate.Repetition}
	}
	if evaluation, err := EvaluatePlan(result, metrics, ObjectiveWeights{Cost: in.CostWeight, Effort: in.EffortWeight, Variety: in.RepetitionWeight}); err == nil {
		result.Evaluation = evaluation
	}
	return result
}
