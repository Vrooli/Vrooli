package gates

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEveryDeclaredGateIsRegisteredExactlyOnce(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "..", "..", "..", "catalog", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		GateTiers []struct {
			ID string `json:"id"`
		} `json:"gateTiers"`
		Gates []struct {
			ID   string `json:"id"`
			Tier string `json:"tier"`
		} `json:"gates"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	tiers := map[string]bool{}
	for _, tier := range config.GateTiers {
		if tier.ID == "" || tiers[tier.ID] {
			t.Fatalf("gate tier is empty or duplicated: %q", tier.ID)
		}
		tiers[tier.ID] = true
	}
	for _, expected := range []string{"release-integrity", "asset-quality", "consumer-use"} {
		if !tiers[expected] {
			t.Errorf("three-tier gate model is missing %q", expected)
		}
	}
	if len(tiers) != 3 {
		t.Fatalf("config declares %d tiers, want exactly three", len(tiers))
	}

	registered := Definitions()
	seen := make(map[string]bool, len(registered))
	for _, definition := range registered {
		if seen[definition.ID] {
			t.Fatalf("gate %q is registered more than once", definition.ID)
		}
		seen[definition.ID] = true
	}
	if len(config.Gates) != len(registered) {
		t.Fatalf("config declares %d gates, registry contains %d", len(config.Gates), len(registered))
	}
	for _, gate := range config.Gates {
		if !tiers[gate.Tier] {
			t.Errorf("gate %q has missing or unknown tier %q", gate.ID, gate.Tier)
		}
		if !seen[gate.ID] {
			t.Fatalf("declared gate %q is not registered", gate.ID)
		}
	}
	for _, definition := range registered {
		found := false
		for _, gate := range config.Gates {
			if gate.ID == definition.ID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("registered gate %q has no catalog declaration", definition.ID)
		}
	}
}
