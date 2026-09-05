package credentialinventory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/credentialauthority"
	"github.com/vrooli/vrooli/internal/resources/securestore"
)

func TestCollectEmptyRootIsValueFreeAndEmpty(t *testing.T) {
	result, err := Collect("")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 0 || len(result.Declared) != 0 || len(result.RequiredAbsent) != 0 {
		t.Fatalf("empty inventory = %+v", result)
	}
}

func TestCollectFixtureHasNoDiscoveredMinusCollectedAddresses(t *testing.T) {
	authority, err := credentialauthority.NewAuthority(securestore.Absent("isolated inventory fixture"))
	if err != nil {
		t.Fatal(err)
	}
	previous := credentialauthority.DefaultAuthority
	credentialauthority.DefaultAuthority = func() (*credentialauthority.Authority, error) { return authority, nil }
	t.Cleanup(func() { credentialauthority.DefaultAuthority = previous })

	root := t.TempDir()
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "service.json"), map[string]any{
		"credentials": map[string]any{"descriptors": []map[string]any{
			{"logical_id": "vrooli/fixture/project", "field": "token", "required": true},
		}},
	})
	writeFixtureFile(t, filepath.Join(root, "resources", "fixture", "resource.json"), map[string]any{
		"name":   "fixture",
		"driver": "external-cli",
		"cli": map[string]any{
			"enabled": true, "command": "fixture",
			"adapter":      map[string]any{"kind": "go_module", "module_dir": "cli"},
			"source_build": map[string]any{"kind": "go_module"},
			"invoke":       map[string]any{"kind": "installed_command", "command": "fixture"},
			"freshness":    map[string]any{"inputs": []string{"cli/**", "resource.json"}},
		},
		"credentials": map[string]any{"descriptors": []map[string]any{
			{"logical_id": "vrooli/fixture/configured", "field": "token", "required": true},
			{"logical_id": "vrooli/fixture/absent", "field": "token", "required": true},
		}},
	})
	writeFixtureFile(t, filepath.Join(root, "scenarios", "fixture-scenario", ".vrooli", "service.json"), map[string]any{
		"service": map[string]any{"name": "fixture-scenario"},
		"credentials": map[string]any{"descriptors": []map[string]any{
			{"logical_id": "vrooli/fixture/configured", "field": "token", "required": true},
			{"logical_id": "vrooli/fixture/absent", "field": "token", "required": true},
		}},
	})

	result, err := Collect(root)
	if err != nil {
		t.Fatal(err)
	}
	collected := make(map[string]struct{}, len(result.Entries))
	for _, entry := range result.Entries {
		collected[string(entry.Identity)+":"+entry.Field] = struct{}{}
	}
	for _, address := range []string{"vrooli/fixture/configured:token", "vrooli/fixture/absent:token"} {
		if _, missing := collected[address]; missing {
			if !contains(result.RequiredAbsent, address) {
				t.Fatalf("declared address %q was neither collected nor marked required-absent", address)
			}
		}
	}
	if !contains(result.RequiredAbsent, "vrooli/fixture/absent:token") {
		t.Fatalf("required absent fixture address missing: %v", result.RequiredAbsent)
	}
	if !contains(result.RequiredAbsent, "vrooli/fixture/project:token") {
		t.Fatalf("project service credential was not collected: entries=%v absent=%v", result.Entries, result.RequiredAbsent)
	}
	if _, present := collected["vrooli/fixture/configured:token"]; !present {
		// The fixture remains portable across hosts whose live store does not
		// contain this deliberately named address; the important invariant is
		// that a declared address is classified, never silently dropped.
		if !contains(result.RequiredAbsent, "vrooli/fixture/configured:token") {
			t.Fatalf("configured fixture address was not classified: entries=%v absent=%v", result.Entries, result.RequiredAbsent)
		}
	}
}

func writeFixtureFile(t *testing.T, path string, value any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
