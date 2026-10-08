package executor

import (
	"strings"
	"testing"

	"github.com/vrooli/api-core/uiselectors"
	"github.com/vrooli/browser-automation-studio/automation/state"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

func TestRuntimeSelectorsEscapeWithoutMutatingPlan(t *testing.T) {
	manifest := map[string]interface{}{"dynamicSelectors": map[string]interface{}{"row": map[string]interface{}{"testIdPattern": "row-${id}", "params": []interface{}{map[string]interface{}{"name": "id", "type": "string"}}}}}
	frozen := uiselectors.ResolveReference(`@selector/row(id="${@params/id}")`, manifest)
	action := &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_CLICK, Params: &basactions.ActionDefinition_Click{Click: &basactions.ClickParams{Selector: frozen}}}
	for _, id := range []string{`a"b`, `c\d`} {
		resolved, err := resolveRuntimeSelectors(action, state.New(nil, map[string]any{"id": id}, nil))
		if err != nil {
			t.Fatal(err)
		}
		want := `[data-testid="row-` + uiselectors.EscapeCSSValue(id) + `"]`
		if resolved.GetClick().Selector != want {
			t.Fatalf("got %q want %q", resolved.GetClick().Selector, want)
		}
		if action.GetClick().Selector != frozen {
			t.Fatal("compiled plan mutated")
		}
	}
	if _, err := resolveRuntimeSelectors(action, state.New(nil, nil, nil)); err == nil {
		t.Fatal("missing runtime argument accepted")
	}
	expression := uiselectors.EscapeExpressionSelector(frozen, '\'')
	got, err := uiselectors.ResolveDeferred(expression, func(string) (string, error) { return `a"b`, nil })
	if err != nil || !strings.Contains(got, `\\22 `) {
		t.Fatalf("deferred expression escaping: %q %v", got, err)
	}
}
