package driver

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

func TestBuildInstructionPayloadRejectsMissingTypedAction(t *testing.T) {
	_, err := buildInstructionPayload(contracts.CompiledInstruction{NodeID: "missing-action"})
	require.EqualError(t, err, "instruction missing-action has no typed action")
}

func TestBuildInstructionPayloadUsesTypedActionOnly(t *testing.T) {
	payload, err := buildInstructionPayload(contracts.CompiledInstruction{
		Index:  3,
		NodeID: "click-save",
		Action: &basactions.ActionDefinition{
			Type: basactions.ActionType_ACTION_TYPE_CLICK,
			Params: &basactions.ActionDefinition_Click{Click: &basactions.ClickParams{
				Selector: "[data-testid=save]",
			}},
		},
	})
	require.NoError(t, err)

	instruction, ok := payload["instruction"].(map[string]any)
	require.True(t, ok)
	require.NotContains(t, instruction, "type")
	require.NotContains(t, instruction, "params")
	action, ok := instruction["action"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(2), action["type"])
}

func TestBuildInstructionPayloadPreservesTypedActionVariants(t *testing.T) {
	for _, tc := range []struct {
		name       string
		action     *basactions.ActionDefinition
		payloadKey string
	}{
		{
			name: "navigate",
			action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE, Params: &basactions.ActionDefinition_Navigate{
				Navigate: &basactions.NavigateParams{Url: "https://example.test"},
			}},
			payloadKey: "navigate",
		},
		{
			name: "input",
			action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_INPUT, Params: &basactions.ActionDefinition_Input{
				Input: &basactions.InputParams{Selector: "#email", Value: "a@example.test"},
			}},
			payloadKey: "input",
		},
		{
			name: "subflow",
			action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_SUBFLOW, Params: &basactions.ActionDefinition_Subflow{
				Subflow: &basactions.SubflowParams{Target: &basactions.SubflowParams_WorkflowPath{WorkflowPath: "actions/login.json"}},
			}},
			payloadKey: "subflow",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload, err := buildInstructionPayload(contracts.CompiledInstruction{NodeID: tc.name, Action: tc.action})
			require.NoError(t, err)
			instruction := payload["instruction"].(map[string]any)
			action := instruction["action"].(map[string]any)
			require.Contains(t, action, tc.payloadKey)
			require.NotContains(t, instruction, "type")
			require.NotContains(t, instruction, "params")
		})
	}
}
