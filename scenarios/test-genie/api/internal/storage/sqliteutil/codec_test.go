package sqliteutil

import (
	"testing"
	"time"
)

func TestFormatAndParseTimestampRoundTrip(t *testing.T) {
	original := time.Date(2026, 3, 28, 15, 4, 5, 123456789, time.FixedZone("local", -4*60*60))

	formatted := FormatTimestamp(original)
	parsed, err := ParseTimestamp(formatted)
	if err != nil {
		t.Fatalf("ParseTimestamp returned error: %v", err)
	}

	if !parsed.Equal(original.UTC()) {
		t.Fatalf("expected %s, got %s", original.UTC(), parsed)
	}
}

func TestParseTimestampRejectsEmptyValue(t *testing.T) {
	if _, err := ParseTimestamp(""); err == nil {
		t.Fatal("expected ParseTimestamp to reject empty values")
	}
}

func TestMarshalAndUnmarshalStringSlice(t *testing.T) {
	encoded, err := MarshalStringSlice([]string{"unit", "integration"})
	if err != nil {
		t.Fatalf("MarshalStringSlice returned error: %v", err)
	}

	values, err := UnmarshalStringSlice(encoded)
	if err != nil {
		t.Fatalf("UnmarshalStringSlice returned error: %v", err)
	}

	if len(values) != 2 || values[0] != "unit" || values[1] != "integration" {
		t.Fatalf("unexpected decoded values: %#v", values)
	}
}

func TestUnmarshalJSONDecodesStructuredPayload(t *testing.T) {
	type sample struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	var decoded sample
	if err := UnmarshalJSON(`{"name":"phase","count":2}`, &decoded); err != nil {
		t.Fatalf("UnmarshalJSON returned error: %v", err)
	}

	if decoded.Name != "phase" || decoded.Count != 2 {
		t.Fatalf("unexpected decoded payload: %#v", decoded)
	}
}

func TestBytesRejectsUnsupportedTypes(t *testing.T) {
	if _, err := Bytes(42); err == nil {
		t.Fatal("expected Bytes to reject unsupported values")
	}
}
