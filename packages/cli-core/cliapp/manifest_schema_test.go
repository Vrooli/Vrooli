package cliapp

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
	repocontract "github.com/vrooli/repo-contract-go"
)

// cliManifestSchemaName is the canonical filename of the cli-manifest JSON
// Schema in .vrooli/schemas/. Mirrors the schema's $id "cli-manifest/v1".
//
// Phase 2 of plan cli-manifest-language-agnostic-single-source-of-truth-for-scenario-clis:
// validates the schema parses, that the template manifest at
// templates/scenarios/react-vite/cli/manifest.json validates cleanly, and that
// intentionally-broken variants are rejected per failure mode.
//
// The reusable LoadFromManifest helper lands in Phase 4. This test deliberately
// duplicates the schema-load wiring so Phase 2 stays narrowly scoped to schema
// correctness.
const cliManifestSchemaName = "cli-manifest.schema.json"

func locateRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".vrooli", "repo-contract.json")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate repo root from %s", dir)
		}
		dir = parent
	}
}

func compileCLIManifestSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	repoRoot := locateRepoRoot(t)
	schemaPath, err := repocontract.SchemaPath(repoRoot, cliManifestSchemaName)
	if err != nil {
		t.Fatalf("SchemaPath: %v", err)
	}
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema %s: %v", schemaPath, err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(cliManifestSchemaName, bytes.NewReader(schemaBytes)); err != nil {
		t.Fatalf("AddResource: %v", err)
	}
	schema, err := compiler.Compile(cliManifestSchemaName)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	return schema
}

func validateCLIManifestBytes(t *testing.T, schema *jsonschema.Schema, raw []byte) error {
	t.Helper()
	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return schema.Validate(doc)
}

func TestCLIManifestSchemaAcceptsTemplate(t *testing.T) {
	schema := compileCLIManifestSchema(t)
	repoRoot := locateRepoRoot(t)
	manifestPath := filepath.Join(repoRoot, "templates", "scenarios", "react-vite", "cli", "manifest.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read template manifest: %v", err)
	}
	if err := validateCLIManifestBytes(t, schema, raw); err != nil {
		t.Fatalf("template manifest must validate cleanly; got: %v", err)
	}
}

func TestCLIManifestSchemaRejectsBrokenVariants(t *testing.T) {
	schema := compileCLIManifestSchema(t)

	cases := []struct {
		name       string
		manifest   string
		mustMatch  string // optional substring expected somewhere in the validation error
	}{
		{
			name: "missing top-level groups",
			manifest: `{
				"name": "demo"
			}`,
			mustMatch: "groups",
		},
		{
			name: "binding kind not in enum",
			manifest: `{
				"name": "demo",
				"groups": [{
					"name": "g",
					"commands": [{
						"name": "c",
						"binding": { "kind": "rest" },
						"governance": { "effect": "read", "run_eligible": true }
					}]
				}]
			}`,
			mustMatch: "kind",
		},
		{
			name: "connect-rpc binding missing method",
			manifest: `{
				"name": "demo",
				"groups": [{
					"name": "g",
					"commands": [{
						"name": "c",
						"binding": { "kind": "connect-rpc", "service": "S" },
						"governance": { "effect": "read", "run_eligible": true }
					}]
				}]
			}`,
			mustMatch: "method",
		},
		{
			name: "governance effect not in enum",
			manifest: `{
				"name": "demo",
				"groups": [{
					"name": "g",
					"commands": [{
						"name": "c",
						"binding": { "kind": "connect-rpc", "service": "S", "method": "M" },
						"governance": { "effect": "mutate", "run_eligible": true }
					}]
				}]
			}`,
			mustMatch: "effect",
		},
		{
			name: "permissions value outside vocabulary",
			manifest: `{
				"name": "demo",
				"groups": [{
					"name": "g",
					"commands": [{
						"name": "c",
						"binding": { "kind": "connect-rpc", "service": "S", "method": "M" },
						"governance": { "effect": "read", "run_eligible": true, "permissions": ["sudo"] }
					}]
				}]
			}`,
			mustMatch: "permissions",
		},
		{
			name: "group with empty commands array",
			manifest: `{
				"name": "demo",
				"groups": [{
					"name": "g",
					"commands": []
				}]
			}`,
			mustMatch: "commands",
		},
		{
			name: "omitted entry missing reason",
			manifest: `{
				"name": "demo",
				"groups": [{
					"name": "g",
					"commands": [{
						"name": "c",
						"binding": { "kind": "connect-rpc", "service": "S", "method": "M" },
						"governance": { "effect": "read", "run_eligible": true }
					}]
				}],
				"omitted": [ { "service": "S", "method": "M2" } ]
			}`,
			mustMatch: "reason",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCLIManifestBytes(t, schema, []byte(tc.manifest))
			if err == nil {
				t.Fatalf("expected validation error for %q; got nil", tc.name)
			}
			if tc.mustMatch != "" && !strings.Contains(err.Error(), tc.mustMatch) {
				t.Fatalf("expected error to mention %q; got: %v", tc.mustMatch, err)
			}
		})
	}
}
