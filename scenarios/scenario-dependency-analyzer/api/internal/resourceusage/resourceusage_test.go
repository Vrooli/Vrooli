package resourceusage

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// writeServiceJSON creates <dir>/<scenario>/.vrooli/service.json with the given
// raw JSON body.
func writeServiceJSON(t *testing.T, dir, scenario, body string) {
	t.Helper()
	vroDir := filepath.Join(dir, scenario, ".vrooli")
	if err := os.MkdirAll(vroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vroDir, "service.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestUsageProviderInversion verifies the fleet scan inverts scenario->resources
// into resource->consuming-scenarios, sorted + deduplicated, preferring
// dependencies.resources over the legacy top-level resources, and skipping dirs
// without a valid service.json.
func TestUsageProviderInversion(t *testing.T) {
	dir := t.TempDir()
	writeServiceJSON(t, dir, "agent-manager", `{
		"resources": {"postgres": {"type": "postgres"}, "redis": {"type": "redis"}}
	}`)
	writeServiceJSON(t, dir, "plan-manager", `{
		"dependencies": {"resources": {"postgres": {"type": "postgres"}, "qdrant": {"type": "qdrant"}}}
	}`)
	writeServiceJSON(t, dir, "search-hub", `{
		"resources": {"qdrant": {"type": "qdrant"}}
	}`)
	// A directory without service.json must be skipped, not error the whole scan.
	if err := os.MkdirAll(filepath.Join(dir, "not-a-scenario"), 0o755); err != nil {
		t.Fatal(err)
	}

	p := NewUsageProvider(func() string { return dir })
	usages, err := p.ResourceUsages(context.Background())
	if err != nil {
		t.Fatalf("ResourceUsages: %v", err)
	}

	got := map[string][]string{}
	for _, u := range usages {
		got[u.Resource] = u.UsedBy
	}
	want := map[string][]string{
		"postgres": {"agent-manager", "plan-manager"},
		"redis":    {"agent-manager"},
		"qdrant":   {"plan-manager", "search-hub"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("inversion =\n%v\nwant\n%v", got, want)
	}

	// Results are sorted by resource name for deterministic indexing.
	if usages[0].Resource != "postgres" || usages[1].Resource != "qdrant" || usages[2].Resource != "redis" {
		t.Fatalf("resources not sorted: %v", []string{usages[0].Resource, usages[1].Resource, usages[2].Resource})
	}
}

// TestUsageProviderEmptyDir returns no records (and no error) for an empty
// scenarios dir or an unset resolver.
func TestUsageProviderEmptyDir(t *testing.T) {
	p := NewUsageProvider(func() string { return "" })
	usages, err := p.ResourceUsages(context.Background())
	if err != nil || len(usages) != 0 {
		t.Fatalf("empty resolver: usages=%v err=%v", usages, err)
	}
}
