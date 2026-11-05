package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestResourceManifestUpstreamCLIShape asserts the new upstream_cli
// block in resources/claude-code/resource.json carries the contract
// fields the adapter relies on (version_command/version_pinned/
// docs_source/docs_pinned_at/schema_version).
//
// Direct shape assertion rather than a full schema-validation test:
// the repo's resource.schema.json transitively references a regex
// pattern with ECMA-script lookahead that Go's RE2 engine can't
// compile, which would make schema-level validation fail on grounds
// orthogonal to this change. The cli-core repo-wide schema test runs
// elsewhere; this scenario-local test just guards the manifest shape
// the doctor verb depends on.
func TestResourceManifestUpstreamCLIShape(t *testing.T) {
	repoRoot := locateRepoRoot(t)
	manifestPath := filepath.Join(repoRoot, "resources", "claude-code", "resource.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	upstream, ok := doc["upstream_cli"].(map[string]any)
	if !ok {
		t.Fatalf("missing upstream_cli object: %v", doc["upstream_cli"])
	}
	for _, key := range []string{"version_command", "version_pinned", "docs_source", "docs_pinned_at", "schema_version"} {
		if _, ok := upstream[key]; !ok {
			t.Errorf("upstream_cli missing required field %q", key)
		}
	}
	if vc, ok := upstream["version_command"].([]any); !ok || len(vc) == 0 {
		t.Errorf("upstream_cli.version_command must be a non-empty array: %v", upstream["version_command"])
	}
	if sv, ok := upstream["schema_version"].(float64); !ok || sv < 1 {
		t.Errorf("upstream_cli.schema_version must be integer >=1: %v", upstream["schema_version"])
	}
}

func locateRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".vrooli", "schemas", "resource.schema.json")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repo root from %s", dir)
		}
		dir = parent
	}
}
