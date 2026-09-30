package scenario

import (
	"encoding/json"
	"testing"
)

func TestServiceManifestBrandingRoundTrip(t *testing.T) {
	withBranding := `{"service":{"name":"aquila"},"branding":{"brand":"aquila","targets":["web-public-v1","electron-v1"]}}`
	var m ServiceManifest
	if err := json.Unmarshal([]byte(withBranding), &m); err != nil {
		t.Fatal(err)
	}
	if m.Branding == nil || m.Branding.Brand != "aquila" {
		t.Fatalf("branding = %+v", m.Branding)
	}
	if len(m.Branding.Targets) != 2 || m.Branding.Targets[0] != "web-public-v1" {
		t.Fatalf("targets = %v", m.Branding.Targets)
	}

	without := `{"service":{"name":"plain"}}`
	var m2 ServiceManifest
	if err := json.Unmarshal([]byte(without), &m2); err != nil {
		t.Fatal(err)
	}
	if m2.Branding != nil {
		t.Fatalf("expected no branding, got %+v", m2.Branding)
	}
}
