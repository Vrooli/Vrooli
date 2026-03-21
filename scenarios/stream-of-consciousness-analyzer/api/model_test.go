package main

import (
	"encoding/json"
	"testing"
	"time"
)

// [REQ:P0-001] Test Scheme JSON serialization
func TestSchemeJSON(t *testing.T) {
	s := Scheme{
		ID:        "test-id",
		Name:      "Test Scheme",
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded Scheme
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if decoded.ID != s.ID || decoded.Name != s.Name {
		t.Errorf("roundtrip failed: got %+v", decoded)
	}
}

// [REQ:P0-003] Test Information JSON serialization with canvas coordinates
func TestInformationJSON(t *testing.T) {
	info := Information{
		ID:       "info-1",
		SchemeID: "scheme-1",
		Type:     "text",
		Content:  "Hello world",
		CanvasX:  150.5,
		CanvasY:  200.0,
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded Information
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if decoded.CanvasX != 150.5 || decoded.CanvasY != 200.0 {
		t.Errorf("canvas coordinates lost: got x=%f y=%f", decoded.CanvasX, decoded.CanvasY)
	}
}

// [REQ:P0-004] Test Thought and ThoughtEdge JSON serialization
func TestThoughtEdgeJSON(t *testing.T) {
	edge := ThoughtEdge{
		ID:       "edge-1",
		SourceID: "thought-a",
		TargetID: "thought-b",
		Label:    "causes",
	}

	data, err := json.Marshal(edge)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded ThoughtEdge
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if decoded.SourceID != "thought-a" || decoded.TargetID != "thought-b" {
		t.Errorf("edge roundtrip failed: got %+v", decoded)
	}
	if decoded.Label != "causes" {
		t.Errorf("expected label=causes, got %s", decoded.Label)
	}
}

// [REQ:P0-002] Test that CreateSchemeInput deserializes correctly
func TestCreateSchemeInputJSON(t *testing.T) {
	body := `{"name":"My Thoughts"}`
	var input CreateSchemeInput
	if err := json.Unmarshal([]byte(body), &input); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if input.Name != "My Thoughts" {
		t.Errorf("expected name=My Thoughts, got %s", input.Name)
	}
}

// [REQ:P0-003] Test that CreateInformationInput with canvas coords deserializes
func TestCreateInformationInputJSON(t *testing.T) {
	body := `{"type":"text","content":"quick note","canvas_x":100,"canvas_y":200}`
	var input CreateInformationInput
	if err := json.Unmarshal([]byte(body), &input); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if input.Type != "text" || input.Content != "quick note" {
		t.Errorf("unexpected input: %+v", input)
	}
	if input.CanvasX != 100 || input.CanvasY != 200 {
		t.Errorf("canvas coords wrong: x=%f y=%f", input.CanvasX, input.CanvasY)
	}
}

// [REQ:P0-001] [REQ:P0-003] Test UpdateThoughtInput partial update fields
func TestUpdateThoughtInputPartialFields(t *testing.T) {
	body := `{"title":"New Title"}`
	var input UpdateThoughtInput
	if err := json.Unmarshal([]byte(body), &input); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if input.Title == nil || *input.Title != "New Title" {
		t.Error("expected title=New Title")
	}
	if input.Body != nil {
		t.Error("expected body=nil for partial update")
	}
	if input.CanvasX != nil {
		t.Error("expected canvas_x=nil for partial update")
	}
}

// [REQ:P0-003] Test UpdateInformationInput partial update fields
func TestUpdateInformationInputPartialFields(t *testing.T) {
	body := `{"content":"Updated content"}`
	var input UpdateInformationInput
	if err := json.Unmarshal([]byte(body), &input); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if input.Content == nil || *input.Content != "Updated content" {
		t.Error("expected content=Updated content")
	}
	if input.Type != nil {
		t.Error("expected type=nil for partial update")
	}
}

// [REQ:P1-001] Test Suggestion JSON serialization
func TestSuggestionJSON(t *testing.T) {
	s := Suggestion{
		ID:         "sug-1",
		SchemeID:   "s1",
		SourceID:   "t1",
		TargetID:   "t2",
		Label:      "relates to",
		Confidence: 0.85,
		Dismissed:  false,
	}

	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded Suggestion
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if decoded.Confidence != 0.85 {
		t.Errorf("expected confidence 0.85, got %f", decoded.Confidence)
	}
	if decoded.Label != "relates to" {
		t.Errorf("expected label 'relates to', got %s", decoded.Label)
	}
}

// [REQ:P1-003] Test Suggestion dismissed field serialization
func TestSuggestionDismissedField(t *testing.T) {
	s := Suggestion{
		ID:         "sug-1",
		SchemeID:   "s1",
		SourceID:   "t1",
		TargetID:   "t2",
		Label:      "relates to",
		Confidence: 0.85,
		Dismissed:  true,
	}

	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded Suggestion
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if !decoded.Dismissed {
		t.Error("expected dismissed=true after roundtrip")
	}
}

// [REQ:P1-003] Test that suggestion confidence is bounded 0-1
func TestSuggestionConfidenceBounds(t *testing.T) {
	cases := []struct {
		name       string
		confidence float64
	}{
		{"zero", 0.0},
		{"mid", 0.5},
		{"max", 1.0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := Suggestion{Confidence: tc.confidence}
			b, _ := json.Marshal(s)
			var decoded Suggestion
			if err := json.Unmarshal(b, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if decoded.Confidence != tc.confidence {
				t.Errorf("expected confidence %f, got %f", tc.confidence, decoded.Confidence)
			}
		})
	}
}

// [REQ:P1-002] Test ExportData JSON serialization
func TestExportDataJSON(t *testing.T) {
	data := ExportData{
		Scheme:       Scheme{ID: "s1", Name: "Test"},
		Information:  []Information{},
		Thoughts:     []Thought{{ID: "t1", Title: "A thought"}},
		Edges:        []ThoughtEdge{},
		ExportFormat: "vrooli-graph-v1",
	}

	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded ExportData
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if decoded.ExportFormat != "vrooli-graph-v1" {
		t.Errorf("expected format vrooli-graph-v1, got %s", decoded.ExportFormat)
	}
	if len(decoded.Thoughts) != 1 || decoded.Thoughts[0].ID != "t1" {
		t.Errorf("thoughts not preserved in export: %+v", decoded.Thoughts)
	}
}

// [REQ:P1-002] Test export format constant
func TestExportFormat(t *testing.T) {
	data := ExportData{
		ExportFormat: "vrooli-graph-v1",
		Information:  []Information{},
		Thoughts:     []Thought{},
		Edges:        []ThoughtEdge{},
	}
	if data.ExportFormat != "vrooli-graph-v1" {
		t.Errorf("expected vrooli-graph-v1 format")
	}
}

// [REQ:P1-004] Test export format string is consistent
func TestExportFormatString(t *testing.T) {
	data := ExportData{
		ExportFormat: "vrooli-graph-v1",
		Information:  []Information{},
		Thoughts:     []Thought{},
		Edges:        []ThoughtEdge{},
	}

	b, _ := json.Marshal(data)
	var decoded map[string]interface{}
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	format, ok := decoded["export_format"].(string)
	if !ok || format != "vrooli-graph-v1" {
		t.Errorf("expected export_format=vrooli-graph-v1, got %v", decoded["export_format"])
	}
}

// [REQ:P1-004] Test export data contains all graph components
func TestExportDataGraphComponents(t *testing.T) {
	schemeID := "scheme-1"
	data := ExportData{
		Scheme: Scheme{ID: schemeID, Name: "Test Scheme"},
		Information: []Information{
			{ID: "i1", SchemeID: schemeID, Type: "text", Content: "note"},
		},
		Thoughts: []Thought{
			{ID: "t1", Title: "Thought A"},
			{ID: "t2", Title: "Thought B"},
		},
		Edges: []ThoughtEdge{
			{ID: "e1", SourceID: "t1", TargetID: "t2", Label: "causes"},
		},
		ExportFormat: "vrooli-graph-v1",
	}

	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded ExportData
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(decoded.Information) != 1 {
		t.Errorf("expected 1 information item, got %d", len(decoded.Information))
	}
	if len(decoded.Thoughts) != 2 {
		t.Errorf("expected 2 thoughts, got %d", len(decoded.Thoughts))
	}
	if len(decoded.Edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(decoded.Edges))
	}
	if decoded.Scheme.ID != schemeID {
		t.Errorf("expected scheme ID %s, got %s", schemeID, decoded.Scheme.ID)
	}
}

// [REQ:P2-002] Test LLMProvider JSON serialization
func TestLLMProviderJSON(t *testing.T) {
	p := LLMProvider{
		Name:     "ollama",
		URL:      "http://localhost:11434",
		Active:   true,
		Fallback: false,
	}

	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded LLMProvider
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if !decoded.Active {
		t.Error("expected active=true")
	}
	if decoded.URL != "http://localhost:11434" {
		t.Errorf("expected ollama URL, got %s", decoded.URL)
	}
}

// [REQ:P2-003] Test provider Active/Fallback JSON fields
func TestProviderActiveAndFallbackFields(t *testing.T) {
	p := LLMProvider{
		Name:     "test-provider",
		URL:      "http://localhost:1234",
		Active:   false,
		Fallback: true,
	}

	b, _ := json.Marshal(p)
	var decoded map[string]interface{}
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if decoded["active"] != false {
		t.Error("expected active=false")
	}
	if decoded["fallback"] != true {
		t.Error("expected fallback=true")
	}
}

// [REQ:P2-004] Test edge creation input with cross-scheme thoughts
func TestCreateEdgeInputJSON(t *testing.T) {
	body := `{"target_id":"thought-b","label":"influences"}`
	var input CreateEdgeInput
	if err := json.Unmarshal([]byte(body), &input); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if input.TargetID != "thought-b" {
		t.Errorf("expected target_id=thought-b, got %s", input.TargetID)
	}
	if input.Label != "influences" {
		t.Errorf("expected label=influences, got %s", input.Label)
	}
}

// [REQ:P2-004] Test ThoughtEdge represents directional connection
func TestThoughtEdgeDirectionality(t *testing.T) {
	edge := ThoughtEdge{
		ID:       "e1",
		SourceID: "thought-a",
		TargetID: "thought-b",
		Label:    "causes",
	}

	if edge.SourceID == edge.TargetID {
		t.Error("edge source and target should be different for directional connection")
	}

	b, _ := json.Marshal(edge)
	var decoded ThoughtEdge
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal edge: %v", err)
	}
	if decoded.SourceID != "thought-a" || decoded.TargetID != "thought-b" {
		t.Error("edge directionality lost in serialization")
	}
}

// [REQ:P2-002] Test that cross-scheme thoughts can be created (scheme_id is nullable)
func TestThoughtWithoutScheme(t *testing.T) {
	input := CreateThoughtInput{
		SchemeID: nil,
		Title:    "Cross-scheme thought",
		Body:     "Spans multiple schemes",
	}

	b, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded CreateThoughtInput
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if decoded.SchemeID != nil {
		t.Error("expected nil scheme_id for cross-scheme thought")
	}
	if decoded.Title != "Cross-scheme thought" {
		t.Errorf("expected title 'Cross-scheme thought', got %s", decoded.Title)
	}
}

// [REQ:P0-001] Test Scheme zero-value fields serialize correctly
func TestSchemeZeroValue(t *testing.T) {
	var s Scheme
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal zero scheme: %v", err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := decoded["id"]; !ok {
		t.Error("expected id field in JSON even when empty")
	}
	if _, ok := decoded["name"]; !ok {
		t.Error("expected name field in JSON even when empty")
	}
}

// [REQ:P0-003] Test Information zero-value canvas coordinates
func TestInformationZeroCoords(t *testing.T) {
	info := Information{ID: "i1", SchemeID: "s1", Type: "text", Content: "test"}
	b, _ := json.Marshal(info)
	var decoded Information
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.CanvasX != 0 || decoded.CanvasY != 0 {
		t.Errorf("expected zero coords, got %f,%f", decoded.CanvasX, decoded.CanvasY)
	}
}

// [REQ:P0-004] Test Thought with body and canvas coordinates
func TestThoughtFullSerialization(t *testing.T) {
	sid := "s1"
	th := Thought{
		ID:       "t1",
		SchemeID: &sid,
		Title:    "Important",
		Body:     "Detailed description",
		CanvasX:  300.5,
		CanvasY:  400.2,
	}
	b, err := json.Marshal(th)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded Thought
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Body != "Detailed description" {
		t.Errorf("expected body preserved, got %s", decoded.Body)
	}
	if decoded.CanvasX != 300.5 {
		t.Errorf("expected canvas_x=300.5, got %f", decoded.CanvasX)
	}
}

// [REQ:P0-002] Test UpdateSchemeInput JSON roundtrip
func TestUpdateSchemeInputJSON(t *testing.T) {
	body := `{"name":"Renamed Scheme"}`
	var input UpdateSchemeInput
	if err := json.Unmarshal([]byte(body), &input); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if input.Name != "Renamed Scheme" {
		t.Errorf("expected Renamed Scheme, got %s", input.Name)
	}
}

// [REQ:P0-003] Test UpdateInformationInput with all fields set
func TestUpdateInformationInputAllFields(t *testing.T) {
	body := `{"type":"voice","content":"transcribed","canvas_x":10.5,"canvas_y":20.3}`
	var input UpdateInformationInput
	if err := json.Unmarshal([]byte(body), &input); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if input.Type == nil || *input.Type != "voice" {
		t.Error("expected type=voice")
	}
	if input.Content == nil || *input.Content != "transcribed" {
		t.Error("expected content=transcribed")
	}
	if input.CanvasX == nil || *input.CanvasX != 10.5 {
		t.Error("expected canvas_x=10.5")
	}
	if input.CanvasY == nil || *input.CanvasY != 20.3 {
		t.Error("expected canvas_y=20.3")
	}
}

// [REQ:P0-004] Test UpdateThoughtInput with body and canvas
func TestUpdateThoughtInputAllFields(t *testing.T) {
	body := `{"title":"New","body":"Updated body","canvas_x":50.0,"canvas_y":60.0}`
	var input UpdateThoughtInput
	if err := json.Unmarshal([]byte(body), &input); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if input.Title == nil || *input.Title != "New" {
		t.Error("expected title=New")
	}
	if input.Body == nil || *input.Body != "Updated body" {
		t.Error("expected body=Updated body")
	}
	if input.CanvasX == nil || *input.CanvasX != 50.0 {
		t.Error("expected canvas_x=50.0")
	}
}

// [REQ:P0-004] Test CreateEdgeInput with empty label
func TestCreateEdgeInputEmptyLabel(t *testing.T) {
	body := `{"target_id":"t2"}`
	var input CreateEdgeInput
	if err := json.Unmarshal([]byte(body), &input); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if input.TargetID != "t2" {
		t.Errorf("expected target_id=t2, got %s", input.TargetID)
	}
	if input.Label != "" {
		t.Errorf("expected empty label, got %s", input.Label)
	}
}

// [REQ:P1-001] Test Suggestion with all fields populated
func TestSuggestionFullFields(t *testing.T) {
	s := Suggestion{
		ID:         "sug-1",
		SchemeID:   "scheme-1",
		SourceID:   "t1",
		TargetID:   "t2",
		Label:      "supports",
		Confidence: 0.92,
		Dismissed:  false,
	}
	b, _ := json.Marshal(s)
	var decoded map[string]interface{}
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	expectedFields := []string{"id", "scheme_id", "source_id", "target_id", "label", "confidence", "dismissed"}
	for _, field := range expectedFields {
		if _, ok := decoded[field]; !ok {
			t.Errorf("missing field %s in JSON output", field)
		}
	}
}

// [REQ:P2-001] Test LLMProvider with fallback set
func TestLLMProviderFallbackSerialization(t *testing.T) {
	p := LLMProvider{
		Name:     "openrouter",
		URL:      "https://openrouter.ai/api/v1",
		Active:   true,
		Fallback: true,
	}
	b, _ := json.Marshal(p)
	var decoded LLMProvider
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !decoded.Fallback {
		t.Error("expected fallback=true")
	}
	if !decoded.Active {
		t.Error("expected active=true")
	}
}

// [REQ:P0-004] Test ThoughtEdge with empty label
func TestThoughtEdgeEmptyLabel(t *testing.T) {
	edge := ThoughtEdge{
		ID:       "e1",
		SourceID: "src",
		TargetID: "tgt",
		Label:    "",
	}
	b, _ := json.Marshal(edge)
	var decoded ThoughtEdge
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Label != "" {
		t.Errorf("expected empty label, got %s", decoded.Label)
	}
}

// [REQ:P1-002] Test ExportData with empty graph components
func TestExportDataEmptyGraph(t *testing.T) {
	data := ExportData{
		Scheme:       Scheme{ID: "s1", Name: "Empty"},
		Information:  []Information{},
		Thoughts:     []Thought{},
		Edges:        []ThoughtEdge{},
		ExportFormat: "vrooli-graph-v1",
	}
	b, _ := json.Marshal(data)
	var decoded ExportData
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(decoded.Information) != 0 || len(decoded.Thoughts) != 0 || len(decoded.Edges) != 0 {
		t.Error("expected empty slices for empty graph")
	}
}

// [REQ:P0-003] Test CreateInformationInput without optional fields
func TestCreateInformationInputMinimal(t *testing.T) {
	body := `{"content":"just content"}`
	var input CreateInformationInput
	if err := json.Unmarshal([]byte(body), &input); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if input.Content != "just content" {
		t.Errorf("expected content='just content', got %s", input.Content)
	}
	if input.Type != "" {
		t.Errorf("expected empty type when not specified, got %s", input.Type)
	}
	if input.CanvasX != 0 || input.CanvasY != 0 {
		t.Error("expected zero canvas coords when not specified")
	}
}

// [REQ:P2-004] Test thoughts with nil vs set scheme_id for cross-scheme linking
func TestThoughtNilSchemeForCrossScheme(t *testing.T) {
	sid := "scheme-1"
	boundThought := CreateThoughtInput{SchemeID: &sid, Title: "Bound"}
	unboundThought := CreateThoughtInput{SchemeID: nil, Title: "Unbound"}

	// Bound thought has scheme
	if boundThought.SchemeID == nil || *boundThought.SchemeID != sid {
		t.Error("expected bound thought to have scheme_id")
	}

	// Unbound thought for cross-scheme linking
	if unboundThought.SchemeID != nil {
		t.Error("expected unbound thought to have nil scheme_id")
	}

	// Both serialize/deserialize correctly
	for _, tc := range []struct {
		name  string
		input CreateThoughtInput
		isNil bool
	}{
		{"bound", boundThought, false},
		{"unbound", unboundThought, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, _ := json.Marshal(tc.input)
			var decoded CreateThoughtInput
			if err := json.Unmarshal(b, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if tc.isNil && decoded.SchemeID != nil {
				t.Error("expected nil scheme_id")
			}
			if !tc.isNil && decoded.SchemeID == nil {
				t.Error("expected non-nil scheme_id")
			}
		})
	}
}
