package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

func TestActionPrimarySelectorUsesTypedActionParams(t *testing.T) {
	action := &basactions.ActionDefinition{
		Type:   basactions.ActionType_ACTION_TYPE_CLICK,
		Params: &basactions.ActionDefinition_Click{Click: &basactions.ClickParams{Selector: "#submit"}},
	}

	assert.Equal(t, "#submit", actionPrimarySelector(action))
}

func TestActionStoreResultUsesExtractContract(t *testing.T) {
	action := &basactions.ActionDefinition{
		Type:   basactions.ActionType_ACTION_TYPE_EXTRACT,
		Params: &basactions.ActionDefinition_Extract{Extract: &basactions.ExtractParams{StoreAs: stringPtr("result")}},
	}

	assert.Equal(t, "result", actionStoreResult(action))
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
