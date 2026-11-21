package journeytest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func supplierOwners(t *testing.T) map[string]string {
	t.Helper()
	root := filepath.Clean("../../../../resources")
	owners := map[string]string{}
	add := func(supplier, path string) {
		if supplier == "" {
			t.Fatalf("missing supplier in %s", path)
		}
		if prior := owners[supplier]; prior != "" {
			t.Fatalf("supplier %s has two owners: %s and %s", supplier, prior, path)
		}
		owners[supplier] = path
	}
	paths, err := filepath.Glob(filepath.Join(root, "*", "model-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var catalog struct {
			Provider *struct {
				ID    string          `json:"id"`
				Terms json.RawMessage `json:"terms"`
			} `json:"provider"`
		}
		if err := json.Unmarshal(data, &catalog); err != nil {
			t.Fatal(err)
		}
		if catalog.Provider != nil && len(catalog.Provider.Terms) > 0 {
			add(catalog.Provider.ID, path)
		}
	}
	path := filepath.Join(root, "provider-rule-proposals.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var proposals struct {
		Records []struct {
			Supplier string `json:"supplier"`
			Terms    struct {
				Review struct {
					Status string `json:"status"`
				} `json:"review"`
			} `json:"terms"`
		} `json:"records"`
	}
	if err := json.Unmarshal(data, &proposals); err != nil {
		t.Fatal(err)
	}
	for _, record := range proposals.Records {
		if record.Terms.Review.Status != "proposed" {
			t.Fatal("unwired supplier proposal became approved")
		}
		add(record.Supplier, path)
	}
	return owners
}
func TestJ3_EveryWiredSupplierAndOperatorRouterHasARecord(t *testing.T) {
	owners := supplierOwners(t)
	for _, supplier := range []string{"ollama", "openrouter", "orcarouter", "abliteration", "pii-detector", "ollaya", "nanogpt", "router-one", "kilo", "inferray", "indierouter", "apmix", "allgate", "nexusapi", "kovar", "apinex", "relaymodels", "uuapi", "opengonka", "llm-api", "featherless", "groq", "openai", "anthropic"} {
		if owners[supplier] == "" {
			t.Errorf("missing supplier record: %s", supplier)
		}
	}
}
func TestJ3_OneOwnerPerFactLintPasses(t *testing.T) {
	if len(supplierOwners(t)) < 24 {
		t.Fatal("supplier ownership inventory incomplete")
	}
}
