package analytics

import (
	"strings"
	"testing"
)

func TestSchema_NotEmpty(t *testing.T) {
	if Schema() == "" {
		t.Fatal("analytics.Schema() returned empty; check go:embed wiring")
	}
}

func TestSchema_ContainsEventsTable(t *testing.T) {
	if !strings.Contains(Schema(), "CREATE TABLE IF NOT EXISTS analytics_events") {
		t.Fatalf("analytics.Schema() missing analytics_events table")
	}
}

func TestSchema_ContainsPlacementsTable(t *testing.T) {
	if !strings.Contains(Schema(), "CREATE TABLE IF NOT EXISTS analytics_placements") {
		t.Fatalf("analytics.Schema() missing analytics_placements table")
	}
}

func TestSchema_ContainsOverridesTable(t *testing.T) {
	if !strings.Contains(Schema(), "CREATE TABLE IF NOT EXISTS analytics_overrides") {
		t.Fatalf("analytics.Schema() missing analytics_overrides table")
	}
}
