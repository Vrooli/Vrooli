package doccontract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSchemaValidatesReactViteManifest(t *testing.T) {
	root := repoRootForTest(t)
	manifestPath := filepath.Join(root, "templates", "scenarios", "react-vite", "docs", "manifest.json")
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	findings := ValidateManifest(manifest, manifestPath)
	for _, f := range findings {
		if f.Code == "schema_violation" || f.Code == "schema_load_error" {
			t.Fatalf("unexpected schema finding: %#v", f)
		}
	}
}

func TestSchemaRejectsManifestVariants(t *testing.T) {
	root := repoRootForTest(t)
	srcPath := filepath.Join(root, "templates", "scenarios", "react-vite", "docs", "manifest.json")
	srcBytes, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("read template manifest: %v", err)
	}
	schemaSrc, err := os.ReadFile(filepath.Join(root, ".vrooli", "schemas", docsManifestSchemaName))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	contractSrc, err := os.ReadFile(filepath.Join(root, ".vrooli", "repo-contract.json"))
	if err != nil {
		t.Fatalf("read contract: %v", err)
	}

	cases := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{
			name: "missing-contract-kind",
			mutate: func(doc map[string]any) {
				delete(doc["contract"].(map[string]any), "kind")
			},
		},
		{
			name: "wrong-completion-enum",
			mutate: func(doc map[string]any) {
				sections := doc["sections"].([]any)
				docs := sections[0].(map[string]any)["documents"].([]any)
				docs[0].(map[string]any)["completion"] = "sometimes"
			},
		},
		{
			name: "missing-doc-title",
			mutate: func(doc map[string]any) {
				sections := doc["sections"].([]any)
				docs := sections[0].(map[string]any)["documents"].([]any)
				delete(docs[0].(map[string]any), "title")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeRepo := t.TempDir()
			if err := os.MkdirAll(filepath.Join(fakeRepo, ".vrooli", "schemas"), 0o755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			if err := os.WriteFile(filepath.Join(fakeRepo, ".vrooli", "repo-contract.json"), contractSrc, 0o644); err != nil {
				t.Fatalf("write contract: %v", err)
			}
			if err := os.WriteFile(filepath.Join(fakeRepo, ".vrooli", "schemas", docsManifestSchemaName), schemaSrc, 0o644); err != nil {
				t.Fatalf("write schema: %v", err)
			}

			var doc map[string]any
			if err := json.Unmarshal(srcBytes, &doc); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			tc.mutate(doc)

			scenarioDocsDir := filepath.Join(fakeRepo, "scenarios", "demo", "docs")
			if err := os.MkdirAll(scenarioDocsDir, 0o755); err != nil {
				t.Fatalf("mkdir scenario: %v", err)
			}
			manifestPath := filepath.Join(scenarioDocsDir, "manifest.json")
			out, _ := json.MarshalIndent(doc, "", "  ")
			if err := os.WriteFile(manifestPath, out, 0o644); err != nil {
				t.Fatalf("write manifest: %v", err)
			}

			loaded, err := LoadManifest(manifestPath)
			if err != nil {
				t.Fatalf("LoadManifest: %v", err)
			}
			findings := ValidateManifest(loaded, manifestPath)
			var schemaViolations []Finding
			for _, f := range findings {
				if f.Code == "schema_violation" {
					schemaViolations = append(schemaViolations, f)
				}
			}
			if len(schemaViolations) == 0 {
				t.Fatalf("expected schema_violation for %s; got findings: %#v", tc.name, findings)
			}
		})
	}
}
