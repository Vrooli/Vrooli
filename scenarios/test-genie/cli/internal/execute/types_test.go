package execute

import (
	"strings"
	"testing"
)

func TestObservationStringFormatsSectionsAndStatuses(t *testing.T) {
	if got := (Observation{Icon: "🔍", Section: "Discovery"}).String(); got != "🔍 Discovery" {
		t.Fatalf("expected section header formatting, got %q", got)
	}
	if got := (Observation{Prefix: "WARNING", Text: "artifact is stale"}).String(); got != "[WARNING] ⚠️ artifact is stale" {
		t.Fatalf("expected warning formatting, got %q", got)
	}
	if got := (Observation{Prefix: "NOTICE", Text: "custom prefix"}).String(); got != "[NOTICE] custom prefix" {
		t.Fatalf("expected custom prefix formatting, got %q", got)
	}
}

func TestObservationListUnmarshalJSONSupportsObjectAndLegacyStringFormats(t *testing.T) {
	var objects ObservationList
	if err := objects.UnmarshalJSON([]byte(`[{"text":"first"},{"section":"Checks"}]`)); err != nil {
		t.Fatalf("expected object format to decode, got %v", err)
	}
	if len(objects) != 2 || objects[0].Text != "first" || !objects[1].IsSection() {
		t.Fatalf("unexpected object observations: %+v", objects)
	}

	var legacy ObservationList
	if err := legacy.UnmarshalJSON([]byte(`["alpha","beta"]`)); err != nil {
		t.Fatalf("expected legacy format to decode, got %v", err)
	}
	if len(legacy) != 2 || legacy[0].Text != "alpha" || legacy[1].Text != "beta" {
		t.Fatalf("unexpected legacy observations: %+v", legacy)
	}
}

func TestObservationListUnmarshalJSONRejectsInvalidJSON(t *testing.T) {
	var observations ObservationList
	err := observations.UnmarshalJSON([]byte(`{"invalid":true}`))
	if err == nil {
		t.Fatal("expected invalid observation payload to fail")
	}
	if !strings.Contains(err.Error(), "cannot unmarshal") {
		t.Fatalf("expected unmarshal error, got %v", err)
	}
}
