package paramschema

import (
	"strings"
	"testing"
)

func mustCompile(t *testing.T, raw string) *Schema {
	t.Helper()
	s, err := Compile([]byte(raw))
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	return s
}

const objectRemovalSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "grow_px": {"type": "integer", "minimum": 0, "maximum": 64},
    "halo_threshold": {"type": "integer", "minimum": 0, "maximum": 255},
    "mode": {"type": "string", "enum": ["soft", "hard"]},
    "feather": {"type": "boolean"}
  }
}`

func TestValidateAcceptsValidValues(t *testing.T) {
	s := mustCompile(t, objectRemovalSchema)
	if err := s.Validate(map[string]any{"grow_px": float64(12), "mode": "soft", "feather": true}); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestValidateRejectsUnknownKey(t *testing.T) {
	s := mustCompile(t, objectRemovalSchema)
	err := s.Validate(map[string]any{"nope": float64(1)})
	if err == nil || !strings.Contains(err.Error(), "unknown parameter") {
		t.Fatalf("expected unknown-parameter error, got %v", err)
	}
}

func TestValidateRejectsNonInteger(t *testing.T) {
	s := mustCompile(t, objectRemovalSchema)
	err := s.Validate(map[string]any{"grow_px": 1.5})
	if err == nil || !strings.Contains(err.Error(), "integer") {
		t.Fatalf("expected integer error, got %v", err)
	}
}

func TestValidateRejectsOutOfRange(t *testing.T) {
	s := mustCompile(t, objectRemovalSchema)
	if err := s.Validate(map[string]any{"grow_px": float64(65)}); err == nil {
		t.Fatal("expected maximum violation")
	}
	if err := s.Validate(map[string]any{"grow_px": float64(-1)}); err == nil {
		t.Fatal("expected minimum violation")
	}
}

func TestValidateRejectsWrongType(t *testing.T) {
	s := mustCompile(t, objectRemovalSchema)
	if err := s.Validate(map[string]any{"grow_px": "12"}); err == nil {
		t.Fatal("expected type error for string-integer")
	}
	if err := s.Validate(map[string]any{"feather": "yes"}); err == nil {
		t.Fatal("expected type error for string-boolean")
	}
}

func TestValidateEnum(t *testing.T) {
	s := mustCompile(t, objectRemovalSchema)
	if err := s.Validate(map[string]any{"mode": "medium"}); err == nil {
		t.Fatal("expected enum violation")
	}
}

func TestValidateRequired(t *testing.T) {
	s := mustCompile(t, `{"type":"object","required":["x"],"properties":{"x":{"type":"number"}}}`)
	err := s.Validate(map[string]any{})
	if err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("expected required error, got %v", err)
	}
}

func TestValidateNestedObjectAndArray(t *testing.T) {
	s := mustCompile(t, `{
	  "type":"object",
	  "properties":{
	    "roi":{"type":"object","additionalProperties":false,"properties":{"x":{"type":"number","minimum":0}}},
	    "boxes":{"type":"array","items":{"type":"object","properties":{"w":{"type":"number","maximum":1}}}}
	  }
	}`)
	if err := s.Validate(map[string]any{
		"roi":   map[string]any{"x": 0.5},
		"boxes": []any{map[string]any{"w": 0.9}},
	}); err != nil {
		t.Fatalf("expected valid nested value, got %v", err)
	}
	err := s.Validate(map[string]any{"roi": map[string]any{"x": float64(-1)}})
	if err == nil {
		t.Fatal("expected nested minimum violation")
	}
	err = s.Validate(map[string]any{"boxes": []any{map[string]any{"w": 2.0}}})
	if err == nil {
		t.Fatal("expected array item maximum violation")
	}
}

func TestCompileRejectsUnsupportedSchema(t *testing.T) {
	cases := []string{
		``,
		`{`,
		`{"type":"null"}`,
		`{"type":"object","properties":"nope"}`,
		`{"type":"object","additionalProperties":"nope"}`,
	}
	for _, raw := range cases {
		if _, err := Compile([]byte(raw)); err == nil {
			t.Errorf("Compile(%q) unexpectedly succeeded", raw)
		}
	}
}

func TestValidateRequiresTrue(t *testing.T) {
	s := mustCompile(t, `{
	  "type":"object",
	  "properties":{
	    "crop_to_mask":{"type":"boolean"},
	    "context_pixels":{"type":"integer","minimum":0,"maximum":512}
	  },
	  "requiresTrue":{"context_pixels":["crop_to_mask"]}
	}`)
	if err := s.Validate(map[string]any{"crop_to_mask": true, "context_pixels": float64(32)}); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
	if err := s.Validate(map[string]any{"context_pixels": float64(32)}); err == nil {
		t.Fatal("expected missing crop_to_mask to fail")
	}
	if err := s.Validate(map[string]any{"crop_to_mask": false, "context_pixels": float64(32)}); err == nil {
		t.Fatal("expected crop_to_mask=false to fail")
	}
	if err := s.Validate(map[string]any{"crop_to_mask": false}); err != nil {
		t.Fatalf("crop_to_mask alone should be valid, got %v", err)
	}
}

func TestValidateMaxItems(t *testing.T) {
	s := mustCompile(t, `{"type":"object","properties":{"xs":{"type":"array","maxItems":2,"items":{"type":"number"}}}}`)
	if err := s.Validate(map[string]any{"xs": []any{float64(1), float64(2)}}); err != nil {
		t.Fatalf("two items should pass: %v", err)
	}
	if err := s.Validate(map[string]any{"xs": []any{float64(1), float64(2), float64(3)}}); err == nil {
		t.Fatal("three items should fail maxItems")
	}
}

func TestValidateStringLength(t *testing.T) {
	s := mustCompile(t, `{"type":"object","properties":{"text":{"type":"string","minLength":1,"maxLength":5}}}`)
	for _, ok := range []string{"a", "abcde", "héllo"} {
		if err := s.Validate(map[string]any{"text": ok}); err != nil {
			t.Errorf("%q should pass: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "abcdef"} {
		if err := s.Validate(map[string]any{"text": bad}); err == nil {
			t.Errorf("%q should fail the length bounds", bad)
		}
	}
	if _, err := Compile([]byte(`{"type":"object","properties":{"text":{"type":"string","minLength":-1}}}`)); err == nil {
		t.Error("a negative minLength should not compile")
	}
}
