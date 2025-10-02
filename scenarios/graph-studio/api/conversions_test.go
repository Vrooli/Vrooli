package main

import (
	"encoding/json"
	"testing"
)

// TestNewConversionEngine tests conversion engine initialization
func TestNewConversionEngine(t *testing.T) {
	engine := NewConversionEngine()

	if engine == nil {
		t.Fatal("NewConversionEngine() returned nil")
	}

	if engine.converters == nil {
		t.Error("converters map is nil")
	}

	// Check that default converters are registered
	expectedPaths := []struct {
		from string
		to   string
	}{
		{"mind-maps", "mermaid"},
		{"mind-maps", "bpmn"},
		{"bpmn", "mermaid"},
		{"bpmn", "mind-maps"},
		{"mermaid", "mind-maps"},
		{"network-graphs", "mermaid"},
	}

	for _, path := range expectedPaths {
		if !engine.CanConvert(path.from, path.to) {
			t.Errorf("Expected conversion path %s -> %s to be registered", path.from, path.to)
		}
	}
}

// TestCanConvert tests conversion path checking
func TestCanConvert(t *testing.T) {
	engine := NewConversionEngine()

	tests := []struct {
		name string
		from string
		to   string
		want bool
	}{
		{"valid path - mind-maps to mermaid", "mind-maps", "mermaid", true},
		{"valid path - bpmn to mermaid", "bpmn", "mermaid", true},
		{"invalid source format", "unknown", "mermaid", false},
		{"invalid target format", "mind-maps", "unknown", false},
		{"both invalid", "unknown1", "unknown2", false},
		{"same format", "mind-maps", "mind-maps", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.CanConvert(tt.from, tt.to)
			if got != tt.want {
				t.Errorf("CanConvert(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

// TestRegisterConverter tests converter registration
func TestRegisterConverter(t *testing.T) {
	engine := &ConversionEngine{
		converters: make(map[string]map[string]Converter),
	}

	// Register a test converter
	converter := &TestConverter{}
	engine.RegisterConverter("test-from", "test-to", converter)

	// Check registration
	if !engine.CanConvert("test-from", "test-to") {
		t.Error("Registered converter not found")
	}

	// Verify the actual converter is stored
	if engine.converters["test-from"]["test-to"] != converter {
		t.Error("Wrong converter stored")
	}
}

// TestGetSupportedConversions tests listing available conversions
func TestGetSupportedConversions(t *testing.T) {
	engine := NewConversionEngine()

	conversions := engine.GetSupportedConversions()

	if len(conversions) == 0 {
		t.Error("Expected some conversion paths, got none")
	}

	// Check structure of returned conversions
	for from, targets := range conversions {
		if from == "" {
			t.Error("Conversion missing 'from' format")
		}
		if len(targets) == 0 {
			t.Errorf("Format %s has no conversion targets", from)
		}
	}
}

// TestConversionMetadata tests metadata extraction
func TestConversionMetadata(t *testing.T) {
	engine := NewConversionEngine()

	// Get a converter and check its metadata
	if engine.CanConvert("mind-maps", "mermaid") {
		converter := engine.converters["mind-maps"]["mermaid"]
		metadata := converter.GetMetadata()

		if metadata.Name == "" {
			t.Error("Converter metadata missing name")
		}
		if metadata.Quality == "" {
			t.Error("Converter metadata missing quality")
		}
	}
}

// TestMindMapToMermaidConverter tests mind map to Mermaid conversion
func TestMindMapToMermaidConverter(t *testing.T) {
	converter := &MindMapToMermaidConverter{}

	// Test valid mind map data
	mindMapData := json.RawMessage(`{
		"root": {
			"text": "Root Node",
			"children": [
				{"text": "Child 1"},
				{"text": "Child 2"}
			]
		}
	}`)

	result, err := converter.Convert(mindMapData, nil)
	if err != nil {
		t.Errorf("Convert() failed with valid data: %v", err)
	}

	if len(result) == 0 {
		t.Error("Convert() returned empty result")
	}

	// Verify it's valid JSON
	var mermaidData map[string]interface{}
	if err := json.Unmarshal(result, &mermaidData); err != nil {
		t.Errorf("Result is not valid JSON: %v", err)
	}
}

// TestBPMNToMermaidConverter tests BPMN to Mermaid conversion
func TestBPMNToMermaidConverter(t *testing.T) {
	converter := &BPMNToMermaidConverter{}

	// Test valid BPMN data with proper structure
	bpmnData := json.RawMessage(`{
		"nodes": [
			{"id": "start", "type": "startEvent", "label": "Start"},
			{"id": "task1", "type": "task", "label": "Task 1"},
			{"id": "end", "type": "endEvent", "label": "End"}
		],
		"edges": [
			{"source": "start", "target": "task1"},
			{"source": "task1", "target": "end"}
		]
	}`)

	result, err := converter.Convert(bpmnData, nil)
	if err != nil {
		t.Errorf("Convert() failed with valid data: %v", err)
	}

	if len(result) == 0 {
		t.Error("Convert() returned empty result")
	}

	// Verify it's valid JSON
	var mermaidData map[string]interface{}
	if err := json.Unmarshal(result, &mermaidData); err != nil {
		t.Errorf("Result is not valid JSON: %v", err)
	}
}

// TestInvalidConversionData tests conversion with invalid data
func TestInvalidConversionData(t *testing.T) {
	converter := &MindMapToMermaidConverter{}

	tests := []struct {
		name string
		data json.RawMessage
	}{
		{"invalid JSON", json.RawMessage(`{invalid json}`)},
		{"empty data", json.RawMessage(`{}`)},
		{"null data", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := converter.Convert(tt.data, nil)
			if err == nil {
				t.Error("Expected error with invalid data, got nil")
			}
		})
	}
}

// TestConverter is a mock converter for testing
type TestConverter struct{}

func (tc *TestConverter) Convert(source json.RawMessage, options map[string]interface{}) (json.RawMessage, error) {
	return json.RawMessage(`{"converted": true}`), nil
}

func (tc *TestConverter) GetMetadata() ConversionMetadata {
	return ConversionMetadata{
		Name:        "Test Converter",
		Description: "A test converter",
		Quality:     "high",
		DataLoss:    false,
		Features:    []string{"test"},
	}
}

// TestConversionWithOptions tests conversion with custom options
func TestConversionWithOptions(t *testing.T) {
	engine := NewConversionEngine()

	mindMapData := json.RawMessage(`{
		"root": {
			"text": "Root",
			"children": [{"text": "Child"}]
		}
	}`)

	options := map[string]interface{}{
		"direction": "LR",
		"theme":     "dark",
	}

	_, err := engine.Convert("mind-maps", "mermaid", mindMapData, options)
	if err != nil {
		t.Errorf("Convert() with options failed: %v", err)
	}
}

// TestGetConversionPath tests finding conversion paths
func TestGetConversionPath(t *testing.T) {
	engine := NewConversionEngine()

	tests := []struct {
		name       string
		from       string
		to         string
		wantPath   bool
		wantDirect bool
	}{
		{"direct conversion", "mind-maps", "mermaid", true, true},
		{"reverse conversion", "mermaid", "mind-maps", true, true},
		{"no conversion", "mind-maps", "mind-maps", false, false},
		{"unsupported source", "unknown", "mermaid", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canConvert := engine.CanConvert(tt.from, tt.to)
			if canConvert != tt.wantPath {
				t.Errorf("CanConvert(%q, %q) = %v, want %v", tt.from, tt.to, canConvert, tt.wantPath)
			}
		})
	}
}
