package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestV2HostRequirementsDerivesMetadataAndSavedOptIn(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	prior := operatorStatePath
	operatorStatePath = func() (string, error) { return filepath.Join(root, ".vrooli", "operator-state.json"), nil }
	t.Cleanup(func() { operatorStatePath = prior })
	for _, path := range []string{
		filepath.Join(root, "scenarios", "alpha", ".vrooli"),
		filepath.Join(root, "internal", "tools", "demo-tool"),
		filepath.Join(root, "internal", "safeguards", "demo-safeguard"),
		filepath.Join(root, ".vrooli"),
	} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), []byte(`{"service":{"name":"alpha"},"hostTools":[{"name":"demo_tool","required":true,"reason":"required test tool"}],"hostSafeguards":[{"name":"demo_safeguard","required":false,"reason":"optional test safeguard"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "tools", "demo-tool", "tool.json"), []byte(`{"name":"demo_tool","description":"Demo tool","commands":["true"],"platforms":["linux"],"privilege":"user","bundling":"host-required"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "safeguards", "demo-safeguard", "safeguard.json"), []byte(`{"name":"demo_safeguard","description":"Demo safeguard","risk":"high","platforms":["linux"],"privilege":"elevated","bundling":"prohibited"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "operator-state.json"), []byte(`{"version":"1.0.0","scenarios":{"alpha":{"enabled":true}},"host_safeguards":{"demo_safeguard":{"opted_in":true}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	w := doGet(t, NewServer(), "/api/v2/host-requirements")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{`"name":"demo_tool"`, `"status":"required"`, `"privilege":"user"`, `"bundling":"host-required"`, `"name":"demo_safeguard"`, `"status":"opted_in"`, `"risk":"high"`} {
		if !strings.Contains(body, want) {
			t.Errorf("response missing %s: %s", want, body)
		}
	}
}
