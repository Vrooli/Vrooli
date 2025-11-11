package database

import (
	"encoding/json"
	"testing"
)

func TestJSONMapScanSupportsByteSlice(t *testing.T) {
	var m JSONMap
	if err := m.Scan([]byte(`{"foo": "bar", "count": 2}`)); err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if got := m["foo"]; got != "bar" {
		t.Fatalf("expected foo=bar got %v", got)
	}
	if got := m["count"]; got != float64(2) {
		t.Fatalf("expected count=2 got %v", got)
	}
}

func TestJSONMapScanSupportsString(t *testing.T) {
	var m JSONMap
	if err := m.Scan("{\"foo\":\"baz\"}"); err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if got := m["foo"]; got != "baz" {
		t.Fatalf("expected foo=baz got %v", got)
	}
}

func TestJSONMapScanSupportsRawMessage(t *testing.T) {
	var m JSONMap
	raw := json.RawMessage([]byte(`{"hello":"world"}`))
	if err := m.Scan(raw); err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if got := m["hello"]; got != "world" {
		t.Fatalf("expected hello=world got %v", got)
	}
}

func TestJSONMapScanRejectsUnsupportedTypes(t *testing.T) {
	var m JSONMap
	if err := m.Scan(123); err == nil {
		t.Fatalf("expected error when scanning unsupported type")
	}
}
