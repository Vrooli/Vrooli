package rolecatalog

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestLoadCatalogRepositoryFixture proves the extracted package still loads and
// validates the maintained catalog without going through package inference.
func TestLoadCatalogRepositoryFixture(t *testing.T) {
	path := filepath.Join("..", "..", "..", "config", "inference-role-catalog.json")
	catalog, err := LoadCatalog(path)
	if err != nil {
		t.Fatalf("LoadCatalog() error = %v", err)
	}
	if catalog.SchemaVersion != 1 {
		t.Fatalf("SchemaVersion = %d, want 1", catalog.SchemaVersion)
	}
	if len(catalog.RoleNames()) == 0 {
		t.Fatal("RoleNames() is empty")
	}
}

// TestValidateRejectsUnknownCodeRoleSchemaKeyword keeps the schema gate wired
// into code-role validation after extraction.
func TestValidateRejectsUnknownCodeRoleSchemaKeyword(t *testing.T) {
	catalog := RoleCatalog{
		SchemaVersion: 1,
		CodeRoles: map[string]CodeRoleContract{
			"sample.code": {
				HandlerID:      "sample/v1",
				MaxSourceBytes: 1024,
				MaxOutputBytes: 1024,
				SchemaContract: map[string]any{
					"type":                  "object",
					"properties":            map[string]any{"value": map[string]any{"type": "string"}},
					"required":              []any{"value"},
					"additionalProperties":  false,
					"unevaluatedProperties": false,
				},
			},
		},
	}
	err := catalog.Validate()
	if err == nil || !strings.Contains(err.Error(), "unevaluatedProperties") {
		t.Fatalf("Validate() error = %v, want unsupported schema construct", err)
	}
}

func TestValidateRejectsUnknownInferenceRoleKind(t *testing.T) {
	catalog := RoleCatalog{
		SchemaVersion: 1,
		Roles: map[string]InferenceRole{
			"sample.role": {
				Kind:       "made_up_kind",
				Candidates: []Candidate{{Provider: "ollama", ResourceRole: "chat.default", Reason: "fixture"}},
			},
		},
	}
	if err := catalog.Validate(); err == nil || !strings.Contains(err.Error(), "unsupported role kind") {
		t.Fatalf("Validate() error = %v, want unsupported role kind", err)
	}
}
