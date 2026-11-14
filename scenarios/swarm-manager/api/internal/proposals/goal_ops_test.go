package proposals

import (
	"strings"
	"testing"
)

const notInGoalVocabulary = "is not valid for a goal proposal"

// GoalOps is rendered into the goal workflow prompts, so an op listed there
// that ValidateGoal rejects would send agents to write proposals the server
// refuses — the exact drift this list exists to prevent.
func TestGoalOpsMatchesWhatValidateGoalAccepts(t *testing.T) {
	state := goalStateFixture()
	for _, op := range GoalOps() {
		err := ValidateGoal(goalProposal(state, Mutation{ID: "m1", Op: op}), state)
		// A payload complaint means the op is in the vocabulary and only this
		// bare mutation is incomplete, which is fine here.
		if err != nil && strings.Contains(err.Error(), notInGoalVocabulary) {
			t.Fatalf("GoalOps lists %q but ValidateGoal rejects the op itself: %v", op, err)
		}
	}
	for _, op := range AllOps() {
		if opInList(op, GoalOps()) {
			continue
		}
		err := ValidateGoal(goalProposal(state, Mutation{ID: "m1", Op: op}), state)
		if err == nil || !strings.Contains(err.Error(), notInGoalVocabulary) {
			t.Fatalf("op %q is absent from GoalOps but ValidateGoal did not reject it as out-of-vocabulary: %v", op, err)
		}
	}
}

func opInList(op Op, list []Op) bool {
	for _, candidate := range list {
		if candidate == op {
			return true
		}
	}
	return false
}
