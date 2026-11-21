package monetization

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildAIFeatureRegistrySortsContractsAndOmitsPromptText(t *testing.T) {
	root := t.TempDir()
	for _, scenario := range []string{"zeta", "alpha"} {
		path := filepath.Join(root, "scenarios", scenario, ".vrooli", "monetization.json")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		feature := `"id":"` + scenario + `.summarize","role":"summarize","profile":"balanced","max_input_bytes":1024,"max_output_bytes":512,"max_user_turns":1,"schema":{"type":"object"}`
		data := []byte(`{"version":2,"ai_features":[{` + feature + `,"pinned_instruction_digest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}`)
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	registry, err := BuildAIFeatureRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.Features) != 2 || registry.Features[0].Scenario != "alpha" || registry.Features[1].Scenario != "zeta" {
		t.Fatalf("registry order/content unexpected: %+v", registry)
	}
	rendered, err := RenderAIFeatureRegistry(registry)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(rendered), "prompt text") {
		t.Fatal("generated registry contains prompt text")
	}
}

func TestBuildAIFeatureRegistryRejectsInvalidAndDuplicateContracts(t *testing.T) {
	root := t.TempDir()
	cases := []struct {
		name, manifest string
	}{
		{"multiple_turns", `{"ai_features":[{"id":"demo.bad","role":"r","profile":"p","max_input_bytes":10,"max_output_bytes":10,"max_user_turns":2,"schema":{"type":"object"}}]}`},
		{"duplicate_id", `{"ai_features":[{"id":"demo.same","role":"r","profile":"p","max_input_bytes":10,"max_output_bytes":10,"max_user_turns":1,"schema":{"type":"object"}},{"id":"demo.same","role":"r","profile":"p","max_input_bytes":10,"max_output_bytes":10,"max_user_turns":1,"schema":{"type":"object"}}]}`},
		{"unknown_prompt_field", `{"ai_features":[{"id":"demo.same","role":"r","profile":"p","max_input_bytes":10,"max_output_bytes":10,"max_user_turns":1,"schema":{"type":"object"},"prompt":"must not enter the generated contract"}]}`},
		{"non_object_schema", `{"ai_features":[{"id":"demo.bad","role":"r","profile":"p","max_input_bytes":10,"max_output_bytes":10,"max_user_turns":1,"schema":[]}]}`},
		{"invalid_id", `{"ai_features":[{"id":"Demo.Bad","role":"r","profile":"p","max_input_bytes":10,"max_output_bytes":10,"max_user_turns":1,"schema":{"type":"object"}}]}`},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			writeAIFeatureManifest(t, root, testCase.manifest)
			if _, err := BuildAIFeatureRegistry(root); err == nil {
				t.Fatal("invalid AI feature manifest was accepted")
			}
		})
	}
}

func writeAIFeatureManifest(t *testing.T, root, content string) {
	t.Helper()
	path := filepath.Join(root, "scenarios", "demo", ".vrooli", "monetization.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestCommittedAIFeatureRegistryMatchesDeclarations(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	artifact := filepath.Join(root, "packages", "monetization-go", "ai-features.generated.json")
	if err := CheckAIFeatureRegistry(root, artifact); err != nil {
		t.Fatal(err)
	}
}
