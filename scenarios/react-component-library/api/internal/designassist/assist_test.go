package designassist

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/stretchr/testify/require"
)

func layer(id, kind string, props map[string]any, children ...Node) Node {
	if props == nil {
		props = map[string]any{}
	}
	return Node{ID: id, Type: kind, Props: props, Children: children}
}

func sampleDocument() Document {
	return Document{
		Name:  "Sample",
		Theme: map[string]string{"accent": "#ffb35c"},
		Frames: []Frame{
			{
				ID:     "desktop",
				Name:   "Desktop",
				Device: "desktop",
				Spec: layer("page", "page", nil,
					layer("hero", "stack", map[string]any{"gap": 18},
						layer("card", "card", map[string]any{"pad": 18},
							layer("title", "text", map[string]any{"text": "Old headline", "variant": "display"}),
						),
						layer("cta", "button", map[string]any{"text": "Open", "variant": "primary"}),
						Node{ID: "kpi", Type: "slot", Role: "metric", Props: map[string]any{"minH": 96}},
					),
					layer("foot", "text", map[string]any{"text": "Footer"}),
				),
			},
			{
				ID:     "phone",
				Name:   "Phone",
				Device: "phone",
				Spec:   layer("phone.page", "page", nil, layer("phone.body", "text", map[string]any{"text": "Phone"})),
			},
		},
	}
}

func scopeLayer(t *testing.T, doc Document) map[string]struct{} {
	t.Helper()
	_, allowed, err := BuildContext(doc, Scope{Kind: ScopeLayer, FrameID: "desktop", NodeID: "hero"})
	require.NoError(t, err)
	return allowed
}

func TestParseDocumentRejectsDuplicateIDs(t *testing.T) {
	raw, err := json.Marshal([]Frame{{
		ID: "one", Name: "One", Device: "desktop",
		Spec: layer("page", "page", nil, layer("same", "text", nil), layer("same", "text", nil)),
	}})
	require.NoError(t, err)
	_, err = ParseDocument(raw, json.RawMessage(`{"accent":"#fff"}`))
	require.ErrorContains(t, err, "reused")
}

func TestBuildContextLayerScopeStopsAtTheSelection(t *testing.T) {
	contextJSON, allowed, err := BuildContext(sampleDocument(), Scope{Kind: ScopeLayer, FrameID: "desktop", NodeID: "hero"})
	require.NoError(t, err)
	var payload struct {
		Selection Node `json:"selection"`
		Parent    struct {
			ID string `json:"id"`
		} `json:"parent"`
		Siblings []struct {
			ID string `json:"id"`
		} `json:"siblings"`
	}
	require.NoError(t, json.Unmarshal(contextJSON, &payload))
	require.Equal(t, "hero", payload.Selection.ID)
	require.Equal(t, "page", payload.Parent.ID)
	require.Len(t, payload.Siblings, 2)
	require.Contains(t, allowed, "hero")
	require.Contains(t, allowed, "title")
	require.Contains(t, allowed, "kpi")
	require.Contains(t, allowed, "page")
	require.Contains(t, allowed, "foot", "a sibling is visible so a layer can be moved beside it")
	require.NotContains(t, allowed, "phone.body")
	require.NotContains(t, allowed, "phone.page")
}

func TestBuildContextDesignScopeIncludesEveryFrame(t *testing.T) {
	_, allowed, err := BuildContext(sampleDocument(), Scope{Kind: ScopeDesign})
	require.NoError(t, err)
	require.Contains(t, allowed, "phone.body")
	require.Contains(t, allowed, "title")
}

func TestPrepareBuildsTheExistingInferenceRole(t *testing.T) {
	request, allowed, err := Prepare(sampleDocument(), Scope{Kind: ScopeFrame, FrameID: "desktop"}, "Make the headline warmer.")
	require.NoError(t, err)
	require.Equal(t, "extract.structured", request.Role)
	require.EqualValues(t, 4096, request.MaxOutputTokens)
	require.JSONEq(t, request.Source, string(request.Context))
	require.Contains(t, request.Instruction, "Make the headline warmer.")
	require.Contains(t, allowed, "title")
	require.NotContains(t, allowed, "phone.body")
}

func TestPrepareRejectsBadInput(t *testing.T) {
	for name, scope := range map[string]Scope{
		"missing frame": {Kind: ScopeFrame},
		"layer without id": {
			Kind: ScopeLayer, FrameID: "desktop",
		},
		"design with frame": {Kind: ScopeDesign, FrameID: "desktop"},
		"unknown kind":      {Kind: "everything"},
	} {
		t.Run(name, func(t *testing.T) {
			_, _, err := Prepare(sampleDocument(), scope, "Do something.")
			require.ErrorIs(t, err, ErrInvalid)
		})
	}
	_, _, err := Prepare(sampleDocument(), Scope{Kind: ScopeDesign}, "   ")
	require.ErrorIs(t, err, ErrInvalid)
	_, _, err = Prepare(sampleDocument(), Scope{Kind: ScopeDesign}, strings.Repeat("x", maxPromptRunes+1))
	require.ErrorIs(t, err, ErrInvalid)
}

func TestValidateAcceptablePatch(t *testing.T) {
	doc := sampleDocument()
	allowed := scopeLayer(t, doc)
	summary := "Sharper copy and a warmer accent."
	text := "Accounts growing this week"
	index := 1
	err := Validate(doc, allowed, Proposal{
		Summary: summary,
		Operations: []Operation{
			{Kind: "setText", NodeID: "title", Text: &text},
			{Kind: "setProp", NodeID: "cta", Key: "variant", Value: "secondary"},
			{Kind: "setTheme", Key: "accent", Value: "#7bd6a4"},
			{Kind: "addLayer", ParentID: "hero", Node: &NewNode{
				Type:  "card",
				Props: map[string]any{"pad": 18.0, "elevated": true},
				Children: []NewNode{
					{Type: "text", Props: map[string]any{"text": "New card"}},
				},
			}},
			{Kind: "moveLayer", NodeID: "foot", ParentID: "hero", Index: &index},
			{Kind: "removeLayer", NodeID: "cta"},
		},
	})
	require.NoError(t, err)
}

func TestValidateRejectsUnsafeOrInapplicablePatches(t *testing.T) {
	text := "New copy"
	cases := map[string]struct {
		proposal Proposal
		contains string
	}{
		"outside the scope": {
			proposal: Proposal{Summary: "x", Operations: []Operation{{Kind: "setText", NodeID: "phone.body", Text: &text}}},
			contains: "outside the selected context",
		},
		"unknown id": {
			proposal: Proposal{Summary: "x", Operations: []Operation{{Kind: "setText", NodeID: "nope", Text: &text}}},
			contains: "outside the selected context",
		},
		"page removal": {
			proposal: Proposal{Summary: "x", Operations: []Operation{{Kind: "removeLayer", NodeID: "page"}}},
			contains: "page root",
		},
		"slot removal": {
			proposal: Proposal{Summary: "x", Operations: []Operation{{Kind: "removeLayer", NodeID: "kpi"}}},
			contains: "structural",
		},
		"move into itself": {
			proposal: Proposal{Summary: "x", Operations: []Operation{{Kind: "moveLayer", NodeID: "hero", ParentID: "card"}}},
			contains: "cannot move into itself",
		},
		"src prop": {
			proposal: Proposal{Summary: "x", Operations: []Operation{{Kind: "setProp", NodeID: "title", Key: "src", Value: "https://example.com"}}},
			contains: "cannot be set",
		},
		"placement on text": {
			proposal: Proposal{Summary: "x", Operations: []Operation{{Kind: "setProp", NodeID: "title", Key: "x", Value: 10.0}}},
			contains: "only valid on a slot",
		},
		"event handler": {
			proposal: Proposal{Summary: "x", Operations: []Operation{{Kind: "setProp", NodeID: "title", Key: "onClick", Value: "alert(1)"}}},
			contains: "cannot be set",
		},
		"add page": {
			proposal: Proposal{Summary: "x", Operations: []Operation{{Kind: "addLayer", ParentID: "hero", Node: &NewNode{Type: "page", Props: map[string]any{}}}}},
			contains: "cannot be added",
		},
		"add image": {
			proposal: Proposal{Summary: "x", Operations: []Operation{{Kind: "addLayer", ParentID: "hero", Node: &NewNode{Type: "image", Props: map[string]any{"src": "/x.png"}}}}},
			contains: "cannot be added",
		},
		"deep add": {
			proposal: Proposal{Summary: "x", Operations: []Operation{{Kind: "addLayer", ParentID: "hero", Node: &NewNode{Type: "card", Props: map[string]any{}, Children: []NewNode{{Type: "card", Props: map[string]any{}, Children: []NewNode{{Type: "text", Props: map[string]any{}}}}}}}}},
			contains: "one level",
		},
		"bad colour": {
			proposal: Proposal{Summary: "x", Operations: []Operation{{Kind: "setTheme", Key: "accent", Value: "url(evil)"}}},
			contains: "colour",
		},
		"unknown theme key": {
			proposal: Proposal{Summary: "x", Operations: []Operation{{Kind: "setTheme", Key: "shadow", Value: "#000"}}},
			contains: "palette token",
		},
		"no summary": {
			proposal: Proposal{Summary: "   ", Operations: []Operation{{Kind: "setText", NodeID: "title", Text: &text}}},
			contains: "summary",
		},
		"too many operations": {
			proposal: Proposal{Summary: "x", Operations: repeatText("title", &text, 9)},
			contains: "1 to 8 operations",
		},
		"remove then edit": {
			proposal: Proposal{Summary: "x", Operations: []Operation{
				{Kind: "removeLayer", NodeID: "title"},
				{Kind: "setText", NodeID: "title", Text: &text},
			}},
			contains: "does not exist",
		},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			err := Validate(sampleDocument(), scopeLayer(t, sampleDocument()), testCase.proposal)
			require.ErrorIs(t, err, ErrInvalid)
			require.ErrorContains(t, err, testCase.contains)
		})
	}
}

func repeatText(nodeID string, text *string, count int) []Operation {
	operations := make([]Operation, count)
	for index := range operations {
		operations[index] = Operation{Kind: "setText", NodeID: nodeID, Text: text}
	}
	return operations
}

func TestSchemaCompilesAndValidatesTheContract(t *testing.T) {
	schemaJSON, err := Schema()
	require.NoError(t, err)
	schema, err := jsonschema.CompileString("design-assist.json", schemaJSON)
	require.NoError(t, err)

	var value any
	require.NoError(t, json.Unmarshal([]byte(`{
		"summary":"Sharper headline",
		"operations":[
			{"kind":"setText","nodeId":"title","text":"New headline"},
			{"kind":"setTheme","key":"accent","value":"#7bd6a4"},
			{"kind":"addLayer","parentId":"hero","node":{"type":"card","props":{"pad":18},"children":[{"type":"text","props":{"text":"Hi"}}]}}
		]
	}`), &value))
	require.NoError(t, schema.Validate(value))

	require.NoError(t, json.Unmarshal([]byte(`{"summary":"x","operations":[{"kind":"addLayer","parentId":"hero","node":{"type":"page","props":{}}}]}`), &value))
	require.Error(t, schema.Validate(value), "page is not an addable type even in the schema")
}
