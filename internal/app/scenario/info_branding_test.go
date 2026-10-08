package scenarioapp

import (
	"encoding/json"
	"testing"

	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
)

// A scenario's brand declaration must be readable from the control plane.
// Without it, resolving a scenario to its brand slug meant parsing
// .vrooli/service.json by hand, or asking brand-manager — which answers with a
// brand UUID describing applied assignment state, not what the scenario
// declares. Both are indirections around a field the manifest already carries.
func TestBuildInfoDataExposesBrandingDeclaration(t *testing.T) {
	item := scenariomodel.Scenario{
		Slug: "web-console",
		Manifest: scenariomodel.ServiceManifest{
			Branding: &scenariomodel.Branding{Brand: "aquila", Targets: []string{"web-public-v1"}},
		},
	}

	info := BuildInfoData(item)
	if info.Branding == nil {
		t.Fatal("branding declaration was dropped")
	}
	if info.Branding.Brand != "aquila" {
		t.Errorf("brand = %q, want %q", info.Branding.Brand, "aquila")
	}

	encoded, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal info: %v", err)
	}
	var decoded struct {
		Branding *scenariomodel.Branding `json:"branding"`
	}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal info: %v", err)
	}
	if decoded.Branding == nil || decoded.Branding.Brand != "aquila" {
		t.Errorf("branding did not survive JSON output: %s", encoded)
	}
}

// A scenario that declares no branding must stay absent rather than emitting an
// empty object, so a reader can tell "unbranded" from "brand not set".
func TestBuildInfoDataOmitsAbsentBranding(t *testing.T) {
	info := BuildInfoData(scenariomodel.Scenario{Slug: "plain"})
	if info.Branding != nil {
		t.Fatalf("expected no branding, got %+v", info.Branding)
	}
	encoded, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal info: %v", err)
	}
	if string(encoded) != "" && containsKey(encoded, "branding") {
		t.Errorf("unbranded scenario emitted a branding key: %s", encoded)
	}
}

func containsKey(encoded []byte, key string) bool {
	var generic map[string]any
	if err := json.Unmarshal(encoded, &generic); err != nil {
		return false
	}
	_, ok := generic[key]
	return ok
}
