package manifest_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	manifest "github.com/vrooli/vrooli/internal/resources/manifest"
)

// Feature: the JSON schema and the Go contract agree
//
//	As a resource author
//	I want my editor's schema completion to match what the control plane accepts
//	So that a manifest that validates in the editor is not rejected at load.

func loadResourceSchema(t *testing.T) map[string]any {
	t.Helper()
	path := filepath.Join(findRepoRoot(t), ".vrooli", "schemas", "resource.schema.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return schema
}

func schemaPath(t *testing.T, node any, keys ...string) any {
	t.Helper()
	current := node
	for _, key := range keys {
		object, ok := current.(map[string]any)
		if !ok {
			t.Fatalf("schema path %v: %q is not an object", keys, key)
		}
		current, ok = object[key]
		if !ok {
			t.Fatalf("schema path %v: %q is absent", keys, key)
		}
	}
	return current
}

func schemaEnum(t *testing.T, node any, keys ...string) []string {
	t.Helper()
	raw, ok := schemaPath(t, node, keys...).([]any)
	if !ok {
		t.Fatalf("schema path %v is not an enum array", keys)
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		text, ok := item.(string)
		if !ok {
			t.Fatalf("schema path %v has a non-string enum member %v", keys, item)
		}
		out = append(out, text)
	}
	return out
}

// Scenario: the schema's closed sets are the Go contract's closed sets.
func TestSchemaAndContractShareTheSameClosedSets(t *testing.T) {
	// Given the checked-in resource schema
	schema := loadResourceSchema(t)
	accel := schemaPath(t, schema, "properties", "acceleration")

	cases := []struct {
		scenario string
		got      []string
		want     []string
	}{
		{
			scenario: "Given the backend vocabulary, Then the schema enum equals AllowedBackends",
			got:      schemaEnum(t, accel, "properties", "backends", "items", "enum"),
			want:     manifest.AllowedBackends,
		},
		{
			scenario: "Given the require vocabulary, Then the schema enum equals AllowedAccelerationRequire",
			got:      schemaEnum(t, accel, "properties", "require", "enum"),
			want:     manifest.AllowedAccelerationRequire,
		},
		{
			scenario: "Given the verify vocabulary, Then the schema enum equals AllowedVerifyKinds",
			got:      schemaEnum(t, accel, "properties", "cuda", "properties", "verify", "properties", "kind", "enum"),
			want:     manifest.AllowedVerifyKinds,
		},
	}

	for _, tc := range cases {
		t.Run(tc.scenario, func(t *testing.T) {
			// Then the two vocabularies match exactly, order included
			if !slices.Equal(tc.got, tc.want) {
				t.Fatalf("schema enum = %v, Go contract = %v", tc.got, tc.want)
			}
		})
	}

	// And every backend in the closed set has its own config block in the schema
	for _, backend := range manifest.AllowedBackends {
		schemaPath(t, accel, "properties", backend)
	}
	// And the block rejects any key that is not a reserved field or a backend
	if allowed, ok := schemaPath(t, accel, "additionalProperties").(bool); !ok || allowed {
		t.Fatalf("acceleration.additionalProperties = %v, want false so a mistyped backend is rejected", allowed)
	}
	// And the manifest root is closed, so an unknown top-level key is a failure
	if allowed, ok := schemaPath(t, schema, "additionalProperties").(bool); !ok || allowed {
		t.Fatalf("resource schema root additionalProperties = %v, want false", allowed)
	}
}

// Scenario: every top-level key the live fleet uses is described by the schema.
//
// The root is closed, so a key a manifest actually uses but the schema omits
// would make the schema wrong rather than the manifest.
func TestSchemaDescribesEveryTopLevelKeyTheFleetUses(t *testing.T) {
	// Given the schema and every resource manifest in the repository
	schema := loadResourceSchema(t)
	properties, ok := schemaPath(t, schema, "properties").(map[string]any)
	if !ok {
		t.Fatal("schema properties is not an object")
	}
	root := findRepoRoot(t)
	entries, err := os.ReadDir(filepath.Join(root, "resources"))
	if err != nil {
		t.Fatalf("read resources dir: %v", err)
	}

	// When every manifest's top-level keys are collected
	var missing []string
	seen := map[string]bool{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, "resources", entry.Name(), "resource.json"))
		if err != nil {
			continue
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("parse %s manifest: %v", entry.Name(), err)
		}
		for key := range raw {
			if _, described := properties[key]; !described && !seen[key] {
				seen[key] = true
				missing = append(missing, entry.Name()+"."+key)
			}
		}
	}

	// Then the schema describes all of them
	if len(missing) > 0 {
		slices.Sort(missing)
		t.Fatalf("resource schema root is closed but does not describe: %v", missing)
	}
}
