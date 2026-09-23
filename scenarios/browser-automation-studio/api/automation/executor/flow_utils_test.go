package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/state"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

func TestVariableConditionUsesExecutionStore(t *testing.T) {
	for _, tc := range []struct {
		operator         basactions.ConditionalOperator
		actual, expected any
		truth            bool
	}{
		{basactions.ConditionalOperator_CONDITIONAL_OPERATOR_EQUALS, "ready", "ready", true},
		{basactions.ConditionalOperator_CONDITIONAL_OPERATOR_EQUALS, "ready", "other", false},
		{basactions.ConditionalOperator_CONDITIONAL_OPERATOR_NOT_EQUALS, false, true, true},
		{basactions.ConditionalOperator_CONDITIONAL_OPERATOR_CONTAINS, "ready now", "now", true},
		{basactions.ConditionalOperator_CONDITIONAL_OPERATOR_STARTS_WITH, "ready now", "ready", true},
		{basactions.ConditionalOperator_CONDITIONAL_OPERATOR_ENDS_WITH, "ready now", "ready", false},
		{basactions.ConditionalOperator_CONDITIONAL_OPERATOR_GT, 4, "3", true},
		{basactions.ConditionalOperator_CONDITIONAL_OPERATOR_GTE, 4, 4, true},
		{basactions.ConditionalOperator_CONDITIONAL_OPERATOR_LT, 4, 3, false},
		{basactions.ConditionalOperator_CONDITIONAL_OPERATOR_LTE, 4, 3, false},
	} {
		for _, negated := range []bool{false, true} {
			t.Run(tc.operator.String()+map[bool]string{true: "/negated", false: "/positive"}[negated], func(t *testing.T) {
				store := state.NewFromStore(map[string]any{"answer": tc.actual})
				name := "answer"
				condition, err := evaluateVariableCondition(&basactions.ConditionalParams{Variable: &name, Operator: &tc.operator, Value: typeconv.AnyToJsonValue(tc.expected), Negate: &negated}, store)
				require.NoError(t, err)
				require.NotNil(t, condition)
				assert.Equal(t, tc.truth != negated, condition.Outcome)
				assert.Equal(t, tc.actual, condition.Actual)
				assert.Equal(t, negated, condition.Negated)
			})
		}
	}
}

func TestVariableConditionErrorsCannotBeNegated(t *testing.T) {
	name := "answer"
	negated := true
	badOperator := basactions.ConditionalOperator(999)
	gt := basactions.ConditionalOperator_CONDITIONAL_OPERATOR_GT
	for _, params := range []*basactions.ConditionalParams{
		nil,
		{Variable: stringPtr("missing"), Negate: &negated},
		{Variable: &name, Operator: &badOperator, Negate: &negated},
		{Variable: &name, Operator: &gt, Value: typeconv.AnyToJsonValue(1), Negate: &negated},
		{Variable: &name, Operator: &gt, Value: typeconv.AnyToJsonValue("NaN"), Negate: &negated},
	} {
		condition, err := evaluateVariableCondition(params, state.NewFromStore(map[string]any{"answer": "not numeric"}))
		require.Error(t, err)
		assert.Nil(t, condition)
	}
	condition, err := evaluateVariableCondition(&basactions.ConditionalParams{Variable: &name, Value: typeconv.AnyToJsonValue(false)}, state.NewFromStore(map[string]any{"answer": false}))
	require.NoError(t, err)
	assert.True(t, condition.Outcome, "an omitted operator defaults to equals")
}

// [REQ:BAS-RH-J24] A failed conditional has no boolean branch result.
func TestConditionalBranchDoesNotInventTruth(t *testing.T) {
	step := contracts.PlanStep{
		Action:   &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_CONDITIONAL},
		Outgoing: []contracts.PlanEdge{{Target: "yes", Condition: "true"}, {Target: "no", Condition: "false"}},
	}
	exec := &SimpleExecutor{}
	failure := contracts.StepOutcome{Failure: &contracts.StepFailure{Message: "evaluation failed"}}
	assert.Empty(t, exec.nextNodeID(step, failure), "an error must not execute the first truth branch")
	step.Outgoing = append(step.Outgoing, contracts.PlanEdge{Target: "recover", Condition: "error"})
	assert.Equal(t, "recover", exec.nextNodeID(step, failure))
	failure.Condition = &contracts.ConditionOutcome{Outcome: true}
	assert.Equal(t, "recover", exec.nextNodeID(step, failure), "failure takes precedence over contradictory condition data")
	for _, truth := range []bool{true, false} {
		outcome := contracts.StepOutcome{Success: true, Condition: &contracts.ConditionOutcome{Outcome: truth}}
		want := "no"
		if truth {
			want = "yes"
		}
		assert.Equal(t, want, exec.nextNodeID(step, outcome))
	}
	step.Outgoing[0].Condition = "IF TRUE"
	step.Outgoing[1].Condition = "IF FALSE"
	assert.Equal(t, "yes", exec.nextNodeID(step, contracts.StepOutcome{Success: true, Condition: &contracts.ConditionOutcome{Outcome: true}}), "builder-authored V2 labels select the same branch")
	assert.Equal(t, "no", exec.nextNodeID(step, contracts.StepOutcome{Success: true, Condition: &contracts.ConditionOutcome{Outcome: false}}))
	step.Outgoing = step.Outgoing[:1]
	assert.Empty(t, exec.nextNodeID(step, contracts.StepOutcome{Success: true, Condition: &contracts.ConditionOutcome{Outcome: false}}), "unwired false branch must not execute true branch")
}

func TestActionPrimarySelectorUsesTypedActionParams(t *testing.T) {
	action := &basactions.ActionDefinition{
		Type:   basactions.ActionType_ACTION_TYPE_CLICK,
		Params: &basactions.ActionDefinition_Click{Click: &basactions.ClickParams{Selector: "#submit"}},
	}

	assert.Equal(t, "#submit", actionPrimarySelector(action))
}

func TestStoreActionResultPreservesDeclaredValue(t *testing.T) {
	for _, value := range []any{"script-value", true, 12.5, nil, map[string]any{"nested": "value"}} {
		action := &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_EVALUATE, Params: &basactions.ActionDefinition_Evaluate{Evaluate: &basactions.EvaluateParams{StoreResult: stringPtr(" result ")}}}
		store := state.NewFromStore(nil)
		storeActionResult(action, map[string]any{"result": value}, store)
		actual, found := store.Get("result")
		assert.True(t, found)
		assert.Equal(t, value, actual, "store the script return value, not its transport envelope")
	}
	action := &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_EXTRACT, Params: &basactions.ActionDefinition_Extract{Extract: &basactions.ExtractParams{StoreAs: stringPtr("result")}}}
	store := state.NewFromStore(nil)
	data := map[string]any{"text": "extracted"}
	storeActionResult(action, data, store)
	actual, found := store.Get("result")
	assert.True(t, found)
	assert.Equal(t, data, actual)
}

func TestActionTimeoutMsUsesTypedActionParams(t *testing.T) {
	timeout := int32(750)
	action := &basactions.ActionDefinition{
		Type:   basactions.ActionType_ACTION_TYPE_WAIT,
		Params: &basactions.ActionDefinition_Wait{Wait: &basactions.WaitParams{TimeoutMs: &timeout}},
	}

	assert.Equal(t, 750, actionTimeoutMs(action))
}

func stringPtr(value string) *string { return &value }
