package workflow

import "testing"

func TestBuildFlowDefinitionV2ForWriteRejectsLegacyNodeShape(t *testing.T) {
	_, err := BuildFlowDefinitionV2ForWrite(map[string]any{
		"nodes": []any{map[string]any{
			"id":   "legacy-click",
			"type": "click",
			"data": map[string]any{"selector": "#submit"},
		}},
	}, nil, nil)
	if err == nil {
		t.Fatal("expected a newly-authored V1 node to be rejected")
	}
}

func TestBuildFlowDefinitionV2ForWriteAcceptsTypedV2(t *testing.T) {
	definition, err := BuildFlowDefinitionV2ForWrite(map[string]any{
		"nodes": []any{map[string]any{
			"id": "navigate",
			"action": map[string]any{
				"navigate": map[string]any{"url": "https://example.com"},
			},
		}},
	}, nil, nil)
	if err != nil {
		t.Fatalf("build typed V2 flow: %v", err)
	}
	if got := definition.GetNodes()[0].GetAction().GetNavigate().GetUrl(); got != "https://example.com" {
		t.Fatalf("navigate URL = %q", got)
	}
}
