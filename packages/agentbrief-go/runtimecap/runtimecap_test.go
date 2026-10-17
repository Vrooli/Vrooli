package runtimecap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLookupCapabilityStates(t *testing.T) {
	root := t.TempDir()
	writeResource := func(name string, value any) {
		t.Helper()
		dir := filepath.Join(root, "resources", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "resource.json"), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeResource("verified", map[string]any{"name": "verified", "agent_hooks": map[string]any{"events": []any{map[string]any{"name": "UserPromptSubmit", "supported": true, "can_inject_context": true, "injection_channel": "stdout_json", "verified_by": "canary", "verified_at": "2026-09-07T00:00:00Z"}}}})
	writeResource("declared", map[string]any{"name": "declared", "agent_hooks": map[string]any{"events": []any{map[string]any{"name": "UserPromptSubmit", "supported": true, "can_inject_context": true, "verified_by": "declared"}}}})
	writeResource("unsupported", map[string]any{"name": "unsupported", "agent_hooks": map[string]any{"events": []any{map[string]any{"name": "UserPromptSubmit", "supported": false, "can_inject_context": false, "reason": "not supported"}}}})

	tests := []struct {
		name                                    string
		runtime                                 string
		wantDeclared, wantSupported, wantInject bool
		wantVerified                            string
	}{
		{"verified", "verified", true, true, true, "canary"},
		{"declared-only", "declared", true, true, true, "declared"},
		{"unsupported", "unsupported", true, false, false, ""},
		{"absent block", "absent", false, false, false, ""},
		{"unknown runtime", "unknown", false, false, false, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Lookup(root, tc.runtime, "UserPromptSubmit")
			if err != nil {
				t.Fatal(err)
			}
			if got.Declared != tc.wantDeclared || got.Supported != tc.wantSupported || got.CanInjectContext != tc.wantInject || got.VerifiedBy != tc.wantVerified {
				t.Fatalf("capability = %+v", got)
			}
		})
	}
}

func TestRepositoryResourceDeclarationsParse(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	entries, err := os.ReadDir(filepath.Join(repoRoot, "resources"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(repoRoot, "resources", entry.Name(), "resource.json")
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		var manifest resourceManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if manifest.AgentHooks == nil {
			continue
		}
		for _, event := range manifest.AgentHooks.Events {
			got, err := Lookup(repoRoot, entry.Name(), event.Name)
			if err != nil {
				t.Fatal(err)
			}
			if !got.Declared {
				t.Fatalf("%s/%s did not round-trip", entry.Name(), event.Name)
			}
		}
	}
}
