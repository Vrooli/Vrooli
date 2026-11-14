package main

import (
	"encoding/json"
	"testing"
)

func TestInjectJSONField_InjectsWhenMissing(t *testing.T) {
	payload := json.RawMessage(`{"name":"test","kind":"idea"}`)
	result := injectJSONField(payload, "spawned_from", "research/my-research")

	var m map[string]any
	if err := json.Unmarshal(result, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got, ok := m["spawned_from"].(string); !ok || got != "research/my-research" {
		t.Errorf("spawned_from = %v, want %q", m["spawned_from"], "research/my-research")
	}
	// Original fields preserved
	if m["name"] != "test" {
		t.Errorf("name = %v, want 'test'", m["name"])
	}
}

func TestInjectJSONField_DoesNotOverride(t *testing.T) {
	payload := json.RawMessage(`{"name":"test","spawned_from":"existing/value"}`)
	result := injectJSONField(payload, "spawned_from", "should-not-override")

	var m map[string]any
	if err := json.Unmarshal(result, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := m["spawned_from"].(string); got != "existing/value" {
		t.Errorf("spawned_from = %q, want %q (should not override)", got, "existing/value")
	}
}

func TestInjectJSONField_InvalidJSON(t *testing.T) {
	payload := json.RawMessage(`not valid json`)
	result := injectJSONField(payload, "key", "value")
	if string(result) != "not valid json" {
		t.Error("should return original payload on invalid JSON")
	}
}

func TestParseProtoStructJSON(t *testing.T) {
	got, err := parseProtoStructJSON(`{"caller.count":7,"caller.payload":{"enabled":true}}`)
	if err != nil {
		t.Fatalf("parseProtoStructJSON: %v", err)
	}
	values := got.AsMap()
	if values["caller.count"] != float64(7) || values["caller.payload"].(map[string]any)["enabled"] != true {
		t.Fatalf("values = %#v", values)
	}
	if _, err := parseProtoStructJSON(`["not","an","object"]`); err == nil {
		t.Fatal("expected non-object rejection")
	}
}
