package variantspace_test

import (
	"bytes"
	"landing-page-react-vite-api/internal/variantspace"
	"testing"
)

func TestParseRequiresPayload(t *testing.T) {
	if _, err := variantspace.Parse(nil); err == nil {
		t.Fatal("expected error for empty payload")
	}
}

func TestParsePreservesRawJSON(t *testing.T) {
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
	space, err := variantspace.Parse(data)
	if err != nil {
		t.Fatalf("expected parse success, got %v", err)
	}
	if !bytes.Equal(space.JSONBytes(), data) {
		t.Fatalf("expected raw JSON preserved, got %q", string(space.JSONBytes()))
	}
}

func TestValidateSelectionRejectsUnknownAxisAndValue(t *testing.T) {
	space, err := variantspace.Parse([]byte(`{
		"_name": "t", "_schemaVersion": 1,
		"axes": { "persona": { "variants": [ { "id": "ops", "label": "Ops" } ] } }
	}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := space.ValidateSelection(map[string]string{"persona": "ops"}); err != nil {
		t.Fatalf("expected valid selection, got %v", err)
	}
	if err := space.ValidateSelection(map[string]string{"persona": "nope"}); err == nil {
		t.Fatal("expected error for invalid axis value")
	}
	if err := space.ValidateSelection(map[string]string{"unknown": "x"}); err == nil {
		t.Fatal("expected error for unknown axis")
	}
}

func TestDefaultIsValid(t *testing.T) {
	space := variantspace.Default()
	if !bytes.Equal(bytes.TrimSpace(space.JSONBytes()), bytes.TrimSpace(variantspace.DefaultJSON)) {
		t.Fatal("expected default JSON preserved")
	}
}
