package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"
)

// [REQ:ONB-SELECT-UNION-EXPORT]
// [REQ:ONB-TIER-SUBSET-SHIPPING]
func TestV2UnionEvidenceExportsOnlySelectedClosureAndManifestHostRequirements(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "operator-state.json"), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z","scenarios":{"root":{"enabled":true}}}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "root", ".vrooli", "service.json"), `{"service":{"name":"root"},"dependencies":{"scenarios":{"helper":{"required":true}},"resources":{"postgres":{"required":true}}},"hostTools":[{"name":"git","required":true}]}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "helper", ".vrooli", "service.json"), `{"service":{"name":"helper"},"hostSafeguards":[{"name":"firewall","required":false}]}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "unrelated", ".vrooli", "service.json"), `{"service":{"name":"unrelated"}}`)
	writeFixtureFile(t, filepath.Join(root, "resources", "postgres", "resource.json"), `{"name":"postgres","category":"database"}`)
	writeFixtureFile(t, filepath.Join(root, "internal", "tools", "git", "tool.json"), `{"name":"git","description":"source control","commands":["git"]}`)
	writeFixtureFile(t, filepath.Join(root, "internal", "safeguards", "firewall", "safeguard.json"), `{"name":"firewall","description":"network safety","risk":"medium","privilege":"elevated","bundling":"host-required"}`)

	w := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.selection.SelectionService/GetUnion", `{"target":"local"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("union status = %d: %s", w.Code, w.Body.String())
	}
	var body struct {
		Scenarios []struct {
			Name string `json:"name"`
		} `json:"scenarios"`
		Resources []struct {
			Name string `json:"name"`
		} `json:"resources"`
		HostTools  []string `json:"hostTools"`
		Safeguards []string `json:"safeguards"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Scenarios) != 2 || body.Scenarios[0].Name != "root" || body.Scenarios[1].Name != "helper" {
		names := map[string]bool{}
		for _, scenario := range body.Scenarios {
			names[scenario.Name] = true
		}
		if !names["root"] || !names["helper"] {
			t.Fatalf("scenario union = %#v", body.Scenarios)
		}
	}
	if len(body.Resources) != 1 || body.Resources[0].Name != "postgres" {
		t.Fatalf("resource union = %#v", body.Resources)
	}
	if len(body.HostTools) != 1 || body.HostTools[0] != "git" || len(body.Safeguards) != 1 || body.Safeguards[0] != "firewall" {
		t.Fatalf("host union = tools %#v safeguards %#v", body.HostTools, body.Safeguards)
	}
	if containsJSONString(w.Body.Bytes(), "unrelated") {
		t.Fatal("union included an unrelated catalog member")
	}
}
