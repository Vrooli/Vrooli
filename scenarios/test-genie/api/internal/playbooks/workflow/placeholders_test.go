package workflow

import (
	"reflect"
	"testing"
)

func TestSubstitutePlaceholdersBaseURL(t *testing.T) {
	doc := map[string]any{
		"url": "${BASE_URL}/login",
	}
	SubstitutePlaceholders(doc, "http://localhost:3000")

	expected := "http://localhost:3000/login"
	if doc["url"] != expected {
		t.Errorf("expected %s, got %s", expected, doc["url"])
	}
}

func TestSubstitutePlaceholdersUIPort(t *testing.T) {
	doc := map[string]any{
		"port": "{{UI_PORT}}",
	}
	SubstitutePlaceholders(doc, "http://localhost:3000")

	expected := "3000"
	if doc["port"] != expected {
		t.Errorf("expected %s, got %s", expected, doc["port"])
	}
}

func TestSubstitutePlaceholdersNested(t *testing.T) {
	doc := map[string]any{
		"nodes": []any{
			map[string]any{
				"data": map[string]any{
					"url": "${BASE_URL}/page",
				},
			},
		},
	}
	SubstitutePlaceholders(doc, "http://localhost:8080")

	nodes := doc["nodes"].([]any)
	node := nodes[0].(map[string]any)
	data := node["data"].(map[string]any)
	expected := "http://localhost:8080/page"
	if data["url"] != expected {
		t.Errorf("expected %s, got %s", expected, data["url"])
	}
}

func TestSubstitutePlaceholdersArray(t *testing.T) {
	doc := []any{
		"${BASE_URL}/a",
		"${BASE_URL}/b",
	}
	SubstitutePlaceholders(doc, "http://localhost:3000")

	expected := []any{
		"http://localhost:3000/a",
		"http://localhost:3000/b",
	}
	if !reflect.DeepEqual(doc, expected) {
		t.Errorf("expected %v, got %v", expected, doc)
	}
}

func TestSubstitutePlaceholdersNoPlaceholders(t *testing.T) {
	doc := map[string]any{
		"static": "value",
		"number": 42,
	}
	SubstitutePlaceholders(doc, "http://localhost:3000")

	if doc["static"] != "value" {
		t.Error("static value should remain unchanged")
	}
	if doc["number"] != 42 {
		t.Error("number should remain unchanged")
	}
}

func TestSubstitutePlaceholdersEmptyBaseURL(t *testing.T) {
	doc := map[string]any{
		"url":  "${BASE_URL}/path",
		"port": "{{UI_PORT}}",
	}
	SubstitutePlaceholders(doc, "")

	// ${BASE_URL} should be replaced with empty string
	if doc["url"] != "/path" {
		t.Errorf("expected /path, got %s", doc["url"])
	}
	// {{UI_PORT}} should remain unchanged (no port to extract)
	if doc["port"] != "{{UI_PORT}}" {
		t.Errorf("expected {{UI_PORT}} unchanged, got %s", doc["port"])
	}
}

func TestExtractPort(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"http://localhost:3000", "3000"},
		{"http://localhost:8080", "8080"},
		{"http://example.com:443", "443"},
		{"http://localhost", ""},
		{"", ""},
		{"localhost:3000", "3000"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := extractPort(tt.url)
			if result != tt.expected {
				t.Errorf("extractPort(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

func TestCleanDefinitionSimple(t *testing.T) {
	doc := map[string]any{
		"nodes":    []any{map[string]any{"id": "1"}},
		"edges":    []any{},
		"settings": map[string]any{"timeout": 30},
		"extra":    "should be removed",
	}

	result := CleanDefinition(doc)

	if result["nodes"] == nil {
		t.Error("expected nodes to be preserved")
	}
	if result["edges"] == nil {
		t.Error("expected edges to be preserved")
	}
	if result["settings"] == nil {
		t.Error("expected settings to be preserved")
	}
	if result["extra"] != nil {
		t.Error("expected extra to be removed")
	}
}

func TestCleanDefinitionWrapped(t *testing.T) {
	doc := map[string]any{
		"flow_definition": map[string]any{
			"nodes": []any{map[string]any{"id": "1"}},
			"edges": []any{},
		},
		"metadata": "should be removed",
	}

	result := CleanDefinition(doc)

	if result["nodes"] == nil {
		t.Error("expected nodes to be extracted from flow_definition")
	}
	if result["metadata"] != nil {
		t.Error("expected metadata to be removed")
	}
}

func TestCleanDefinitionNestedWorkflows(t *testing.T) {
	doc := map[string]any{
		"nodes": []any{
			map[string]any{
				"id": "1",
				"data": map[string]any{
					"workflowDefinition": map[string]any{
						"nodes": []any{map[string]any{"id": "nested"}},
						"edges": []any{},
						"junk":  "should be removed",
					},
				},
			},
		},
		"edges": []any{},
	}

	result := CleanDefinition(doc)

	nodes := result["nodes"].([]any)
	node := nodes[0].(map[string]any)
	data := node["data"].(map[string]any)
	nestedWf := data["workflowDefinition"].(map[string]any)

	if nestedWf["nodes"] == nil {
		t.Error("expected nested nodes to be preserved")
	}
	if nestedWf["junk"] != nil {
		t.Error("expected junk to be removed from nested workflow")
	}
}

func TestCleanDefinitionEmpty(t *testing.T) {
	doc := map[string]any{}
	result := CleanDefinition(doc)

	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}
