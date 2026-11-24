package main

import (
	"reflect"
	"testing"
)

func TestHostSafeguardSecretFieldsTreatNestedSecretsAsSecretConfig(t *testing.T) {
	schema := map[string]any{"properties": map[string]any{
		"settings": map[string]any{"type": "object", "properties": map[string]any{
			"api_token": map[string]any{"type": "string"},
		}},
		"bindings": map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{
			"password": map[string]any{"type": "string"},
		}}},
		"mode":          map[string]any{"type": "string", "enum": []any{"guard", "observe"}},
		"private_value": map[string]any{"type": "string", "format": "password"},
	}}
	got := hostSafeguardSecretFields(schema)
	want := []string{"bindings", "private_value", "settings"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("secret config fields = %v, want %v", got, want)
	}
}
