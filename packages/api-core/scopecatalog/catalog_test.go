package scopecatalog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name     string
		held     []string
		required string
		want     bool
	}{
		{"exact", []string{"bridge:read"}, "bridge:read", true},
		{"global wildcard", []string{"*"}, "bridge:destructive", true},
		{"scenario wildcard", []string{"bridge:*"}, "bridge:write", true},
		{"effect wildcard", []string{"*:read"}, "bridge:read", true},
		{"empty held", nil, "bridge:read", false},
		{"unknown scope", []string{"other:read"}, "bridge:write", false},
		{"case mismatch", []string{"Bridge:read"}, "bridge:read", false},
		{"whitespace", []string{"bridge:read "}, "bridge:read", false},
		{"required whitespace", []string{"bridge:read"}, " bridge:read", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Resolve(test.held, test.required); got != test.want {
				t.Fatalf("Resolve(%q, %q) = %v, want %v", test.held, test.required, got, test.want)
			}
		})
	}
}

func TestHasScopeUsesDerivedConcreteValues(t *testing.T) {
	catalog := Catalog{
		Scopes:             []Scope{{Value: "vrooli-bridge:write"}},
		OmittedResolutions: []OmittedResolution{{Scope: "scenario-authenticator:destructive"}},
	}
	if !catalog.HasScope("vrooli-bridge:write") {
		t.Fatal("expected governed scope to resolve")
	}
	if !catalog.HasScope("scenario-authenticator:destructive") {
		t.Fatal("expected omitted scope default to resolve")
	}
	if catalog.HasScope("vrooli-bridge:unknown") || catalog.HasScope(" ") {
		t.Fatal("unknown or empty scope resolved")
	}
}

func TestMaterializeExpandsAndExcludesHumanOnlyScopes(t *testing.T) {
	got := Materialize([]string{"vrooli-bridge:*"}, []Scope{
		{Value: "vrooli-bridge:read", RunEligible: true},
		{Value: "vrooli-bridge:write", RunEligible: false},
	}, false)
	if len(got) != 1 || got[0] != "vrooli-bridge:read" {
		t.Fatalf("materialized scopes = %#v", got)
	}
}

func TestBuildCatalog(t *testing.T) {
	catalog, err := Build(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if catalog.ManifestCount != 59 {
		t.Fatalf("manifest count = %d, want 59", catalog.ManifestCount)
	}
	if catalog.GovernedCommandCount < 1789 {
		t.Fatalf("governed command count = %d, unexpectedly below recorded baseline", catalog.GovernedCommandCount)
	}
	if catalog.MostRestrictiveDefaultCount == 0 {
		t.Fatal("expected omitted RPCs to resolve to a restrictive default")
	}
	t.Logf("scope catalog: manifests=%d governed_commands=%d rpc_scopes=%d omitted=%d most_restrictive_defaults=%d scenarios_without_manifest=%d", catalog.ManifestCount, catalog.GovernedCommandCount, catalog.RPCScopeCount, catalog.OmittedCount, catalog.MostRestrictiveDefaultCount, len(catalog.ScenariosWithoutManifest))
	if len(catalog.Scopes) != catalog.GovernedCommandCount {
		t.Fatalf("scope count = %d, governed commands = %d", len(catalog.Scopes), catalog.GovernedCommandCount)
	}
}

func TestBuildDoesNotWriteScenarios(t *testing.T) {
	root := repoRoot(t)
	before := scenarioFiles(t, root)
	if _, err := Build(root); err != nil {
		t.Fatal(err)
	}
	after := scenarioFiles(t, root)
	if len(before) != len(after) {
		t.Fatalf("scenario file count changed: before=%d after=%d", len(before), len(after))
	}
	for path, mode := range before {
		if after[path] != mode {
			t.Fatalf("scenario file metadata changed for %s", path)
		}
	}
}

func TestWriteJSONArtifact(t *testing.T) {
	root := repoRoot(t)
	catalog, err := Build(root)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "scope-catalog.json")
	if err := catalog.WriteJSON(root, target); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("catalog artifact was not written: %v", err)
	}
	if err := catalog.WriteJSON(root, filepath.Join("scenarios", "scope-catalog.json")); err == nil {
		t.Fatal("expected catalog write below scenarios/ to be refused")
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	working, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(working, "../../.."))
}

func scenarioFiles(t *testing.T, root string) map[string]os.FileMode {
	t.Helper()
	files := make(map[string]os.FileMode)
	paths, err := filepath.Glob(filepath.Join(root, "scenarios", "*", "cli", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		files[path] = info.Mode()
	}
	return files
}
