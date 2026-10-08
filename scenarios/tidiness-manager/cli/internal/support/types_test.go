package support

import (
	"encoding/json"
	"testing"
)

func TestScanDurationAcceptsAPIAndStringForms(t *testing.T) {
	var result SmartScanResult
	if err := json.Unmarshal([]byte(`{"duration":2000000000,"batch_results":[{"duration":"150ms"}]}`), &result); err != nil {
		t.Fatalf("decode mixed duration forms: %v", err)
	}
	if got, want := result.Duration.String(), "2s"; got != want {
		t.Fatalf("top-level duration = %q, want %q", got, want)
	}
	if got, want := result.BatchResults[0].Duration.String(), "150ms"; got != want {
		t.Fatalf("batch duration = %q, want %q", got, want)
	}
}

func TestScanDurationRejectsAmbiguousNumericForms(t *testing.T) {
	var result SmartScanResult
	if err := json.Unmarshal([]byte(`{"duration":1.5}`), &result); err == nil {
		t.Fatal("expected fractional duration to be rejected")
	}
}
