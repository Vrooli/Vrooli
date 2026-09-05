package manifest_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// Feature: the schema rejects a malformed accelerator declaration
//
//	As a resource author
//	I want my editor to reject a mistyped backend or an unsteppable VRAM claim
//	So that the failure surfaces while I am writing the manifest, not when the
//	control plane refuses to start the resource.

// schemaNamespace is the absolute base the checked-in schemas declare in their
// $id, and therefore the base every cross-file $ref resolves against.
const schemaNamespace = "https://vrooli.com/schemas/"

// compileResourceSchema loads the checked-in schema with its sibling schemas
// resolvable, so $refs across .vrooli/schemas resolve without network access.
func compileResourceSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	schemaDir := filepath.Join(findRepoRoot(t), ".vrooli", "schemas")
	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft7
	entries, err := os.ReadDir(schemaDir)
	if err != nil {
		t.Fatalf("read schema dir: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(schemaDir, entry.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		var identified struct {
			ID string `json:"$id"`
		}
		if err := json.Unmarshal(data, &identified); err != nil {
			t.Fatalf("parse %s: %v", entry.Name(), err)
		}
		// Register under both the relative filename siblings use and the
		// absolute $id the schemas declare, so nothing reaches the network.
		names := []string{entry.Name(), schemaNamespace + entry.Name()}
		if identified.ID != "" && identified.ID != schemaNamespace+entry.Name() {
			names = append(names, identified.ID)
		}
		for _, name := range names {
			if err := compiler.AddResource(name, bytes.NewReader(data)); err != nil {
				t.Fatalf("add %s as %s: %v", entry.Name(), name, err)
			}
		}
	}
	schema, err := compiler.Compile("resource.schema.json")
	if err != nil {
		t.Fatalf("compile resource.schema.json: %v", err)
	}
	return schema
}

// Scenario: the schema enforces the accelerator contract's closed sets and rules.
func TestResourceSchemaRejectsMalformedAccelerationBlocks(t *testing.T) {
	schema := compileResourceSchema(t)

	cases := []struct {
		scenario   string
		accelerate map[string]any
		wantReject bool
	}{
		{
			scenario: "Given a backend named opencl, Then the schema rejects it",
			accelerate: map[string]any{
				"backends": []any{"opencl"},
				"opencl":   map[string]any{},
			},
			wantReject: true,
		},
		{
			scenario: "Given a vram claim with no profile, Then the schema rejects it",
			accelerate: map[string]any{
				"backends": []any{"cuda", "cpu"},
				"cuda":     map[string]any{},
				"cpu":      map[string]any{},
				"claim": map[string]any{
					"resource_kind":   "vram",
					"preferred_bytes": 1 << 30,
					"yield_when_idle": true,
				},
			},
			wantReject: true,
		},
		{
			scenario: "Given a vram claim that never yields when idle, Then the schema rejects it",
			accelerate: map[string]any{
				"backends": []any{"cuda", "cpu"},
				"cuda":     map[string]any{},
				"cpu":      map[string]any{},
				"claim": map[string]any{
					"resource_kind":   "vram",
					"preferred_bytes": 1 << 30,
					"profile": map[string]any{
						"steps": []any{map[string]any{"label": "floor", "amount_bytes": 0}},
						"apply": map[string]any{"verb": "capacity"},
					},
				},
			},
			wantReject: true,
		},
		{
			scenario: "Given backends names cuda but there is no cuda config block, Then the schema rejects it",
			accelerate: map[string]any{
				"backends": []any{"cuda", "cpu"},
				"cpu":      map[string]any{},
			},
			wantReject: true,
		},
		{
			scenario: "Given require is required with only a cpu backend, Then the schema rejects it",
			accelerate: map[string]any{
				"backends": []any{"cpu"},
				"require":  "required",
				"cpu":      map[string]any{},
			},
			wantReject: true,
		},
		{
			scenario: "Given a mistyped verify kind, Then the schema rejects it",
			accelerate: map[string]any{
				"backends": []any{"cuda"},
				"cuda":     map[string]any{"verify": map[string]any{"kind": "guess"}},
			},
			wantReject: true,
		},
		{
			scenario: "Given a well-formed block with a steppable vram claim, Then the schema accepts it",
			accelerate: map[string]any{
				"backends": []any{"cuda", "cpu"},
				"require":  "preferred",
				"cuda":     map[string]any{"min_compute": "8.9", "env": map[string]any{"DEVICE": "cuda"}},
				"cpu":      map[string]any{},
				"claim": map[string]any{
					"resource_kind":   "vram",
					"preferred_bytes": 2 << 30,
					"floor_bytes":     1 << 30,
					"priority":        "service",
					"yield_when_idle": true,
					"profile": map[string]any{
						"steps": []any{
							map[string]any{"label": "full", "amount_bytes": 2 << 30},
							map[string]any{"label": "reduced", "amount_bytes": 1 << 30},
						},
						"apply":   map[string]any{"verb": "capacity", "argv": []any{"degrade", "--to", "{label}"}},
						"upshift": true,
					},
				},
			},
			wantReject: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.scenario, func(t *testing.T) {
			// Given a fixture manifest carrying only the acceleration block under test
			document := map[string]any{"acceleration": tc.accelerate}

			// When the schema validates it
			err := schema.Validate(document)

			// Then only the acceleration rules decide the verdict. Everything
			// else the fixture omits is reported as a separate missing-property
			// error, so the assertion asks specifically about acceleration.
			rejected := mentionsAcceleration(err)
			if rejected != tc.wantReject {
				t.Fatalf("acceleration rejected = %v, want %v (schema said: %v)", rejected, tc.wantReject, err)
			}
		})
	}
}

// mentionsAcceleration reports whether any validation error is anchored inside
// the acceleration block.
func mentionsAcceleration(err error) bool {
	if err == nil {
		return false
	}
	detailed, ok := err.(*jsonschema.ValidationError)
	if !ok {
		return strings.Contains(err.Error(), "acceleration")
	}
	var walk func(*jsonschema.ValidationError) bool
	walk = func(node *jsonschema.ValidationError) bool {
		if strings.Contains(node.InstanceLocation, "/acceleration") {
			return true
		}
		for _, child := range node.Causes {
			if walk(child) {
				return true
			}
		}
		return false
	}
	return walk(detailed)
}
