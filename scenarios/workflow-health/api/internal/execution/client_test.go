package execution

import (
	"testing"

	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

func TestDefinitionToProtoPreservesNestedActionParams(t *testing.T) {
	def := map[string]any{
		"metadata": map[string]any{
			"name":           "dashboard-smoke",
			"execution_mode": "observer",
		},
		"nodes": []any{
			map[string]any{
				"id": "assert-page-loaded",
				"action": map[string]any{
					"type": "ACTION_TYPE_ASSERT",
					"assert": map[string]any{
						"selector": "@selector/dashboard.header",
						"mode":     "ASSERTION_MODE_TEXT_CONTAINS",
						"expected": map[string]any{"stringValue": "Test Genie"},
					},
				},
			},
		},
	}

	protoDef, err := definitionToProto(def)
	if err != nil {
		t.Fatalf("definitionToProto returned error: %v", err)
	}
	if len(protoDef.GetNodes()) != 1 {
		t.Fatalf("expected one node, got %d", len(protoDef.GetNodes()))
	}
	action := protoDef.GetNodes()[0].GetAction()
	if got := action.GetType(); got != basactions.ActionType_ACTION_TYPE_ASSERT {
		t.Fatalf("unexpected action type: %v", got)
	}
	assert := action.GetAssert()
	if assert == nil {
		t.Fatal("expected assert params to be preserved")
	}
	if got := assert.GetSelector(); got != "@selector/dashboard.header" {
		t.Fatalf("unexpected assert selector: %q", got)
	}
}
