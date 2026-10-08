package handlers

import (
	"testing"

	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

func TestRecordingSelectorsRespectOriginAndAmbiguity(t *testing.T) {
	manifest := map[string]interface{}{"selectors": map[string]interface{}{
		"app.save":     map[string]interface{}{"testId": "save", "selector": `[data-testid="save"]`},
		"app.close":    map[string]interface{}{"testId": "close", "selector": `[data-testid="close"]`},
		"dialog.close": map[string]interface{}{"testId": "close", "selector": `[data-testid="close"]`},
	}}
	nav := func(url string) *basworkflows.WorkflowNodeV2 {
		return &basworkflows.WorkflowNodeV2{Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE, Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: url}}}}
	}
	click := func(id string) *basworkflows.WorkflowNodeV2 {
		return &basworkflows.WorkflowNodeV2{Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_CLICK, Params: &basactions.ActionDefinition_Click{Click: &basactions.ClickParams{Selector: `[data-testid="` + id + `"]`}}}}
	}
	flow := &basworkflows.WorkflowDefinitionV2{Nodes: []*basworkflows.WorkflowNodeV2{click("save"), nav("http://localhost:1234/"), click("save"), click("close"), nav("https://external.example/"), click("save")}}
	symbolizeRecordingSelectors(flow, manifest, "http://localhost:1234/")
	for i, want := range map[int]string{0: `[data-testid="save"]`, 2: "@selector/app.save", 3: `[data-testid="close"]`, 5: `[data-testid="save"]`} {
		if got := flow.Nodes[i].Action.GetClick().Selector; got != want {
			t.Fatalf("node %d: %q, want %q", i, got, want)
		}
	}
}
