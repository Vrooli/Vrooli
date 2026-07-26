package driver

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

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
