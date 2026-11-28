package agentharness

import (
	"strings"
	"testing"
)

func TestHardStopPolicyLoadsAndRefusesTypos(t *testing.T) {
	if _, err := loadHardStops(hardStopsJSON); err != nil {
		t.Fatalf("embedded hard_stops.json: %v", err)
	}
	misspelled := strings.Replace(string(hardStopsJSON), `"transports"`, `"transport"`, 1)
	if _, err := loadHardStops([]byte(misspelled)); err == nil {
		t.Fatal("a misspelled key must be refused, not silently ignored")
	}
	emptied := strings.Replace(string(hardStopsJSON), `"checkout_words": ["checkout"]`, `"checkout_words": []`, 1)
	if _, err := loadHardStops([]byte(emptied)); err == nil || !strings.Contains(err.Error(), "checkout_words") {
		t.Fatalf("an empty list must be refused by name, got %v", err)
	}
}
