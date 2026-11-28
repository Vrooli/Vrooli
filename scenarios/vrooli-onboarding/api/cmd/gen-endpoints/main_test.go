package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/module"
)

func TestValidateTransportRejectsUnregisteredREST(t *testing.T) {
	if err := validateTransport([]module.EndpointDescriptor{{ID: "bad", Path: "/api/v1/bad"}}); err == nil {
		t.Fatal("expected an untagged REST endpoint to be rejected")
	}
}

func TestGenerateCheckRequiresAllModuleMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "endpoints.json")
	if err := os.WriteFile(path, []byte(`{"endpoints":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := generate(path, true); err == nil {
		t.Fatal("expected missing module endpoint metadata to fail")
	}
}

func TestMergeEndpointsReplacesOwnedRouteAndPreservesOtherCatalogRows(t *testing.T) {
	current := []json.RawMessage{
		json.RawMessage(`{"id":"legacy-health","path":"/health","method":"GET","summary":"old"}`),
		json.RawMessage(`{"id":"custom","path":"/api/v1/custom","method":"GET","summary":"keep"}`),
	}

	merged, err := mergeEndpoints(current, []module.EndpointDescriptor{{
		ID: "health", Path: "/health", Method: "GET", Summary: "canonical",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) != 2 {
		t.Fatalf("merged endpoint count = %d, want 2", len(merged))
	}
	var first, second map[string]any
	if err := json.Unmarshal(merged[0], &first); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(merged[1], &second); err != nil {
		t.Fatal(err)
	}
	if first["summary"] != "canonical" || first["path"] != "/health" {
		t.Fatalf("replacement row = %#v", first)
	}
	if second["id"] != "custom" || second["summary"] != "keep" {
		t.Fatalf("unrelated row changed: %#v", second)
	}
}

func TestMergeEndpointsRetiresOnlyMigratedOwnerLegacyPaths(t *testing.T) {
	current := []json.RawMessage{
		json.RawMessage(`{"id":"legacy-credentials","path":"/api/v2/credentials/list","method":"GET"}`),
		json.RawMessage(`{"id":"legacy-apply","path":"/api/v2/apply/start","method":"POST"}`),
		json.RawMessage(`{"id":"credential-policy","path":"/api/v2/credential-policy","method":"GET"}`),
	}

	merged, err := mergeEndpoints(current, []module.EndpointDescriptor{{
		ID: "credentials_new", Path: "/vrooli.example.Credentials/List", Method: "POST",
	}})
	if err != nil {
		t.Fatal(err)
	}
	paths := make([]string, 0, len(merged))
	for _, raw := range merged {
		var endpoint struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(raw, &endpoint); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, endpoint.Path)
	}
	want := []string{"/api/v2/credential-policy", "/vrooli.example.Credentials/List"}
	if len(paths) != len(want) {
		t.Fatalf("merged paths = %v, want %v", paths, want)
	}
	for index := range want {
		if paths[index] != want[index] {
			t.Fatalf("merged paths = %v, want %v", paths, want)
		}
	}
}
