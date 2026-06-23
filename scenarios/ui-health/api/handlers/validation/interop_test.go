package validation

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile writes content to root/rel, creating parents.
func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunInteropFindingsFlagsMissingBridgeDeps(t *testing.T) {
	root := t.TempDir()
	// A ui/ surface whose package.json is missing the interop deps triggers the
	// dependency-presence interop rules.
	writeFile(t, root, "ui/package.json", `{"name":"demo","dependencies":{"react":"^18.0.0"}}`)

	finds := runInteropFindings(root, "demo")
	if len(finds) == 0 {
		t.Fatal("expected interop findings for a ui/ surface missing bridge deps, got none")
	}
	seen := map[string]bool{}
	for _, f := range finds {
		seen[f.Code] = true
	}
	for _, want := range []string{"interop_api_base_dep", "interop_iframe_bridge_dep"} {
		if !seen[want] {
			t.Errorf("expected interop finding %q in %v", want, seen)
		}
	}
}

func TestRunInteropFindingsSkipsNonUIScenario(t *testing.T) {
	root := t.TempDir() // no ui/ directory at all
	if finds := runInteropFindings(root, "demo"); len(finds) != 0 {
		t.Fatalf("expected no interop findings for a scenario with no ui/ surface, got %d: %v", len(finds), finds)
	}
}
