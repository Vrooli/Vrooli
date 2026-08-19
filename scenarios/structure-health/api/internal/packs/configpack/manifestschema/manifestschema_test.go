package manifestschema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot walks up to the repository that owns this scenario so the rule is
// exercised against the real schema rather than a fixture that could drift
// from it. The blind spot this rule closes was created precisely by checks
// that never touched the canonical schema.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".vrooli", "schemas", "service.schema.json")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Skip("repository schema directory not reachable from the test working directory")
		}
		dir = parent
	}
}

func TestShouldCheckOnlyScenarioManifests(t *testing.T) {
	cases := map[string]bool{
		"/repo/scenarios/alpha/.vrooli/service.json": true,
		"scenarios/alpha/.vrooli/service.json":       true,
		"/repo/.vrooli/service.json":                 false, // the project manifest is not a scenario target
		"/repo/scenarios/alpha/.vrooli/testing.json": false,
		"service.json": false, // synthetic doc-test path: no repository to resolve
		"":             false,
	}
	for path, want := range cases {
		if got := ShouldCheck(path); got != want {
			t.Errorf("ShouldCheck(%q) = %v, want %v", path, got, want)
		}
	}
}

// TestRepositorySchemaCompiles is the guard that matters most. If
// service.schema.json stops compiling, every scenario silently becomes
// unvalidatable — the exact failure the repository already lived through when a
// generated "../resources.schema.json" ref pointed one directory too high.
func TestRepositorySchemaCompiles(t *testing.T) {
	root := repoRoot(t)
	if _, err := loadSchema(filepath.Join(root, ".vrooli", "schemas")); err != nil {
		t.Fatalf("canonical service.schema.json does not compile: %v", err)
	}
}

func TestValidManifestProducesNoViolations(t *testing.T) {
	root := repoRoot(t)
	manifest := map[string]any{
		"$schema": "../../../.vrooli/schemas/service.schema.json",
		"version": "1.0.0",
		"service": map[string]any{
			"name":    "alpha",
			"version": "1.0.0",
		},
		"dependencies": map[string]any{
			"resources": map[string]any{
				"postgres": map[string]any{"enabled": true, "required": true, "purpose": "primary store"},
			},
		},
		"lifecycle": map[string]any{
			"setup":   map[string]any{"steps": []any{map[string]any{"name": "build-api", "run": "true"}}},
			"develop": map[string]any{"steps": []any{map[string]any{"name": "start-api", "run": "true"}}},
		},
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	path := filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json")
	if got := CheckServiceManifestSchema(raw, path); len(got) != 0 {
		for _, v := range got {
			t.Logf("unexpected: %s", v.Description)
		}
		t.Fatalf("valid manifest produced %d violations, want 0", len(got))
	}
}

func TestInvalidManifestIsReported(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json")

	// A phase name no runtime resolves. Before this rule existed, 68 manifests
	// carried exactly this class of drift with nothing to surface it.
	raw := []byte(`{
  "version": "1.0.0",
  "service": {"name": "alpha", "version": "1.0.0"},
  "dependencies": {"resources": {}},
  "lifecycle": {
    "setup": {"steps": [{"name": "build-api", "run": "true"}]},
    "teleport": {"steps": [{"name": "beam", "run": "true"}]}
  }
}`)
	got := CheckServiceManifestSchema(raw, path)
	if len(got) == 0 {
		t.Fatal("an undeclared lifecycle phase produced no violations")
	}
	joined := strings.ToLower(describeAll(got))
	if !strings.Contains(joined, "teleport") {
		t.Fatalf("violation does not name the offending key: %s", joined)
	}
	if got[0].LineNumber <= 0 {
		t.Fatalf("violation line = %d, want a positive line number", got[0].LineNumber)
	}
}

func TestMalformedJSONIsReported(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json")
	got := CheckServiceManifestSchema([]byte("{not json"), path)
	if len(got) != 1 || !strings.Contains(got[0].Description, "not valid JSON") {
		t.Fatalf("malformed manifest violations = %+v", got)
	}
}

func TestUnreachableSchemaIsReportedNotSkipped(t *testing.T) {
	// A scenario path with no repository above it must produce a finding. A
	// silent pass here would recreate the invisible-validation problem.
	path := filepath.Join(t.TempDir(), "scenarios", "alpha", ".vrooli", "service.json")
	got := CheckServiceManifestSchema([]byte(`{}`), path)
	if len(got) != 1 || !strings.Contains(got[0].Description, "cannot locate") {
		t.Fatalf("unreachable schema violations = %+v", got)
	}
}

func describeAll(violations []Violation) string {
	parts := make([]string, 0, len(violations))
	for _, v := range violations {
		parts = append(parts, v.Description)
	}
	return strings.Join(parts, " | ")
}
