package main

import (
	"bytes"
	"testing"
)

func TestParseVariantSpaceRequiresPayload(t *testing.T) {
	if _, err := parseVariantSpace(nil); err == nil {
		t.Fatal("expected error for empty payload")
	}
}

func TestParseVariantSpacePreservesRawJSON(t *testing.T) {
	data := []byte(`{
		"_name": "test",
		"_schemaVersion": 1,
		"axes": {
			"persona": {
				"variants": [
					{ "id": "ops", "label": "Ops" }
				]
			}
		}
	}`)

	space, err := parseVariantSpace(data)
	if err != nil {
		t.Fatalf("expected parse success, got %v", err)
	}

	if !bytes.Equal(space.JSONBytes(), data) {
		t.Fatalf("expected raw JSON preserved, got %q", string(space.JSONBytes()))
	}
}

func TestLoadVariantSpaceBytesFallsBackToDefaults(t *testing.T) {
	data := loadVariantSpaceBytes("does-not-exist.json")
	if !bytes.Equal(bytes.TrimSpace(data), bytes.TrimSpace(defaultVariantSpaceJSON)) {
		t.Fatalf("expected fallback to default JSON")
	}
}
