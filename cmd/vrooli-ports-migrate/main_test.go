package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFile writes contents to a path, creating parent dirs.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// readFile reads a file into a string.
func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

// setupRepo creates a temp directory laid out like the Vrooli repo enough
// for the migration tool to discover scenarios and templates.
func setupRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	// Marker for resolveRepoRoot.
	writeFile(t, filepath.Join(root, ".git", "HEAD"), "ref: refs/heads/test\n")
	return root
}

func TestShiftRange(t *testing.T) {
	cases := []struct {
		in       string
		wantOut  string
		wantDone bool
	}{
		{"35000-39999", "20000-24999", true},
		{"35000-36000", "20000-21000", true},
		{"36234-36234", "21234-21234", true},
		{"15000-19999", "15000-19999", false}, // already safe
		{"30000-40000", "30000-40000", false}, // partial overlap — leave alone
		{"malformed", "malformed", false},
	}
	for _, c := range cases {
		got, done := shiftRange(c.in)
		if got != c.wantOut || done != c.wantDone {
			t.Errorf("shiftRange(%q) = (%q, %v), want (%q, %v)", c.in, got, done, c.wantOut, c.wantDone)
		}
	}
}

func TestProcessManifest_FixedPortShift(t *testing.T) {
	root := setupRepo(t)
	manifest := filepath.Join(root, "scenarios", "swarm-manager", ".vrooli", "service.json")
	writeFile(t, manifest, `{
  "service": { "name": "swarm-manager" },
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-19999"
    },
    "ui": {
      "env_var": "UI_PORT",
      "port": 36234
    }
  }
}
`)

	res, err := processManifest(root, manifest, true)
	if err != nil {
		t.Fatalf("processManifest: %v", err)
	}
	if len(res.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d: %+v", len(res.Changes), res.Changes)
	}
	if res.Changes[0].NewValue != "21234" {
		t.Errorf("new value = %q, want 21234", res.Changes[0].NewValue)
	}

	got := readFile(t, manifest)
	if !strings.Contains(got, `"port": 21234`) {
		t.Errorf("file not rewritten:\n%s", got)
	}
	if strings.Contains(got, "36234") {
		t.Errorf("old port still present:\n%s", got)
	}
	// Ensure we didn't touch the api range.
	if !strings.Contains(got, `"range": "15000-19999"`) {
		t.Errorf("api range lost:\n%s", got)
	}
}

func TestProcessManifest_RangeShift(t *testing.T) {
	root := setupRepo(t)
	manifest := filepath.Join(root, "scenarios", "demo", ".vrooli", "service.json")
	writeFile(t, manifest, `{
  "ports": {
    "ui": {
      "env_var": "UI_PORT",
      "range": "35000-39999"
    }
  }
}
`)

	res, err := processManifest(root, manifest, true)
	if err != nil {
		t.Fatalf("processManifest: %v", err)
	}
	if len(res.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(res.Changes))
	}
	if res.Changes[0].NewValue != "20000-24999" {
		t.Errorf("new range = %q", res.Changes[0].NewValue)
	}

	got := readFile(t, manifest)
	if !strings.Contains(got, `"range": "20000-24999"`) {
		t.Errorf("range not rewritten:\n%s", got)
	}
}

func TestProcessManifest_Idempotent(t *testing.T) {
	root := setupRepo(t)
	manifest := filepath.Join(root, "scenarios", "demo", ".vrooli", "service.json")
	writeFile(t, manifest, `{
  "ports": {
    "ui": {
      "env_var": "UI_PORT",
      "port": 21234
    }
  }
}
`)
	res, err := processManifest(root, manifest, true)
	if err != nil {
		t.Fatalf("processManifest: %v", err)
	}
	if len(res.Changes) != 0 {
		t.Errorf("already-shifted manifest should produce no changes, got: %+v", res.Changes)
	}
}

func TestProcessManifest_SkipNonUIFixed(t *testing.T) {
	root := setupRepo(t)
	manifest := filepath.Join(root, "scenarios", "demo", ".vrooli", "service.json")
	writeFile(t, manifest, `{
  "ports": {
    "mcp": {
      "env_var": "MCP_PORT",
      "range": "3290-3299"
    }
  }
}
`)
	res, err := processManifest(root, manifest, true)
	if err != nil {
		t.Fatalf("processManifest: %v", err)
	}
	if len(res.Changes) != 0 {
		t.Errorf("non-UI safe range should not be shifted: %+v", res.Changes)
	}
}

func TestProcessManifest_MissingPorts(t *testing.T) {
	root := setupRepo(t)
	manifest := filepath.Join(root, "scenarios", "minimal", ".vrooli", "service.json")
	writeFile(t, manifest, `{
  "service": { "name": "minimal" }
}
`)
	res, err := processManifest(root, manifest, true)
	if err != nil {
		t.Fatalf("processManifest: %v", err)
	}
	if res.Skipped == "" {
		t.Errorf("expected Skipped reason for no ports block")
	}
}

func TestProcessManifest_UnparseableJSON(t *testing.T) {
	root := setupRepo(t)
	manifest := filepath.Join(root, "scenarios", "broken", ".vrooli", "service.json")
	writeFile(t, manifest, "{ not json")
	res, err := processManifest(root, manifest, true)
	if err != nil {
		t.Fatalf("processManifest: %v", err)
	}
	if !strings.HasPrefix(res.Skipped, "unparseable JSON") {
		t.Errorf("expected unparseable-JSON skip, got %q", res.Skipped)
	}
}

func TestRunMigration_DetectsCollisions(t *testing.T) {
	root := setupRepo(t)
	writeFile(t, filepath.Join(root, "scenarios", "a", ".vrooli", "service.json"),
		`{"ports":{"ui":{"env_var":"UI_PORT","port":36234}}}`+"\n")
	writeFile(t, filepath.Join(root, "scenarios", "b", ".vrooli", "service.json"),
		`{"ports":{"ui":{"env_var":"UI_PORT","port":36234}}}`+"\n")

	rep, err := runMigration(root, false)
	if err != nil {
		t.Fatalf("runMigration: %v", err)
	}
	if len(rep.Collisions) == 0 {
		t.Fatalf("expected collisions; got none")
	}
	if !strings.Contains(rep.Collisions[0], "21234") {
		t.Errorf("collision message should mention new port: %q", rep.Collisions[0])
	}
}

func TestRunMigration_EndToEndIdempotent(t *testing.T) {
	root := setupRepo(t)
	manifest := filepath.Join(root, "scenarios", "swarm-manager", ".vrooli", "service.json")
	writeFile(t, manifest, `{
  "ports": {
    "api": { "env_var": "API_PORT", "range": "15000-19999" },
    "ui":  { "env_var": "UI_PORT", "port": 36234 }
  }
}
`)

	// First apply: one change.
	rep, err := runMigration(root, true)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	total := 0
	for _, m := range rep.Manifests {
		total += len(m.Changes)
	}
	if total != 1 {
		t.Errorf("expected 1 change on first apply, got %d", total)
	}

	// Second apply: zero changes.
	rep, err = runMigration(root, true)
	if err != nil {
		t.Fatalf("second apply: %v", err)
	}
	total = 0
	for _, m := range rep.Manifests {
		total += len(m.Changes)
	}
	if total != 0 {
		t.Errorf("second apply should be no-op, got %d changes", total)
	}
}

func TestDetectTunneled(t *testing.T) {
	root := setupRepo(t)
	manifest := filepath.Join(root, "scenarios", "tunneled", ".vrooli", "service.json")
	writeFile(t, manifest, `{"ports":{}}`)
	writeFile(t, filepath.Join(root, "scenarios", "tunneled", "README.md"),
		"This scenario sits behind a Cloudflare tunnel.\n")
	if !detectTunneled(root, "tunneled", manifest) {
		t.Errorf("expected tunneled=true for Cloudflare README")
	}

	clean := filepath.Join(root, "scenarios", "plain", ".vrooli", "service.json")
	writeFile(t, clean, `{"ports":{}}`)
	writeFile(t, filepath.Join(root, "scenarios", "plain", "README.md"), "Nothing to see here.\n")
	if detectTunneled(root, "plain", clean) {
		t.Errorf("expected tunneled=false for non-tunnel README")
	}
}

func TestScenarioSlugFromPath(t *testing.T) {
	root := "/repo"
	cases := map[string]string{
		"/repo/scenarios/swarm-manager/.vrooli/service.json":                     "swarm-manager",
		"/repo/templates/scenarios/react-vite/.vrooli/service.json":              "template:react-vite",
		"/repo/templates/scenarios/landing-page-react-vite/.vrooli/service.json": "template:landing-page-react-vite",
	}
	for path, want := range cases {
		if got := scenarioSlugFromPath(root, path); got != want {
			t.Errorf("scenarioSlugFromPath(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestFindMatchingBrace(t *testing.T) {
	in := `{"a": {"b": 1}, "c": 2}`
	got := findMatchingBrace(in, 0)
	if got != len(in)-1 {
		t.Errorf("findMatchingBrace outer = %d, want %d", got, len(in)-1)
	}
	// Nested
	inner := strings.Index(in, `{"b"`)
	got = findMatchingBrace(in, inner)
	if in[got] != '}' || got >= len(in)-1 {
		t.Errorf("findMatchingBrace inner did not match; got idx %d char %q", got, string(in[got]))
	}
}

func TestFindMatchingBrace_StringsIgnored(t *testing.T) {
	in := `{"note": "a } with braces {"}`
	got := findMatchingBrace(in, 0)
	if got != len(in)-1 {
		t.Errorf("findMatchingBrace ignored string braces; got %d, want %d", got, len(in)-1)
	}
}
