package wizard

import (
	"encoding/json"
	"testing"
)

func TestSelectionRoundTripPreservesTriStateChoices(t *testing.T) {
	original := Selection{
		Scenarios:     []string{"alpha"},
		ScenarioState: map[string]bool{"alpha": true, "beta": false},
		Resources:     map[string]bool{"postgres": true, "ollama": false},
		HostTools:     map[string]bool{"git": true},
		HostSafeguards: map[string]bool{"firewall": false},
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var roundTripped Selection
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatal(err)
	}
	if roundTripped.ScenarioState["beta"] || roundTripped.Resources["ollama"] || roundTripped.HostSafeguards["firewall"] {
		t.Fatalf("explicit false choices were lost: %#v", roundTripped)
	}
	patch := selectionPatch(roundTripped)
	resources := patch["resources"].(map[string]any)
	if resources["ollama"].(map[string]any)["enabled"] != false {
		t.Fatalf("resource false choice was not emitted: %#v", patch)
	}
}
