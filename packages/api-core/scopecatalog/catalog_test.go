package scopecatalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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

func TestLookupVerbReturnsOneConcreteCatalogScope(t *testing.T) {
	catalog := Catalog{Scopes: []Scope{
		{Scenario: ProjectManifestIdentity, Command: "scenario/status", Value: "vrooli:read", Effect: EffectRead, RunEligible: true},
		{Scenario: "web-console", Command: "sessions/list", Value: "web-console:read", Effect: EffectRead, RunEligible: true},
		{Scenario: "human-only", Command: "secrets/show", Value: "human-only:read", Effect: EffectRead},
	}}

	project, ok := catalog.LookupVerb("scenario status")
	if !ok || project.Value != "vrooli:read" {
		t.Fatalf("project lookup = %#v, %v", project, ok)
	}
	scenario, ok := catalog.LookupVerb("web-console sessions list")
	if !ok || scenario.Value != "web-console:read" {
		t.Fatalf("scenario lookup = %#v, %v", scenario, ok)
	}
	if _, ok := catalog.LookupVerb("human-only secrets show"); ok {
		t.Fatal("human-only command must not enter the remote invocation vocabulary")
	}
}

func TestLookupVerbFailsClosedOnConflictingEffects(t *testing.T) {
	catalog := Catalog{Scopes: []Scope{
		{Scenario: ProjectManifestIdentity, Command: "scenario/status", Value: "vrooli:read", RunEligible: true},
		{Scenario: ProjectManifestIdentity, Command: "scenario/status", Value: "vrooli:write", RunEligible: true},
	}}
	if _, ok := catalog.LookupVerb("scenario status"); ok {
		t.Fatal("ambiguous command effects must fail closed")
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
	root := repoRoot(t)
	catalog, err := Build(root)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := manifestPaths(root)
	if err != nil {
		t.Fatal(err)
	}
	if catalog.ManifestCount != len(paths) {
		t.Fatalf("manifest count = %d, discovered paths = %d", catalog.ManifestCount, len(paths))
	}
	if len(paths) == 0 || paths[0] != filepath.Join(root, "cli", "manifest.json") {
		t.Fatalf("project manifest is not the deterministic first catalog input: %v", paths)
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

func TestDeriveManifestHonorsFlatGroups(t *testing.T) {
	tests := []struct {
		name    string
		group   manifestGroup
		command string
	}{
		{
			name:    "flat group emits a top-level command",
			group:   manifestGroup{Name: "lifecycle", Flat: true},
			command: "setup",
		},
		{
			name:    "non-flat group emits a nested command",
			group:   manifestGroup{Name: "scenario", Flat: false},
			command: "scenario/status",
		},
		{
			name:    "omitted flat defaults to nested",
			group:   manifestGroup{Name: "resource"},
			command: "resource/logs",
		},
		{
			name: "two-level nested group emits the full path",
			group: manifestGroup{
				Name:   "credentials",
				Groups: []manifestGroup{{Name: "keyring", Commands: []manifestCommand{{Name: "status"}}}},
			},
			command: "credentials/keyring/status",
		},
		{
			name: "three-level nested group emits the full path",
			group: manifestGroup{
				Name:   "runtime",
				Groups: []manifestGroup{{Name: "recovery", Groups: []manifestGroup{{Name: "policy", Commands: []manifestCommand{{Name: "list"}}}}}},
			},
			command: "runtime/recovery/policy/list",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			catalog := Catalog{}
			group := test.group
			setTestGovernance := func(command *manifestCommand) {
				command.Governance = manifestGovernance{Effect: string(EffectRead), RunEligible: true}
			}
			if len(group.Commands) == 0 && len(group.Groups) == 0 {
				group.Commands = []manifestCommand{{Name: "setup"}}
			}
			if group.Name == "scenario" {
				group.Commands[0].Name = "status"
			}
			if group.Name == "resource" {
				group.Commands[0].Name = "logs"
			}
			var applyGovernance func(*manifestGroup)
			applyGovernance = func(current *manifestGroup) {
				for i := range current.Commands {
					setTestGovernance(&current.Commands[i])
				}
				for i := range current.Groups {
					applyGovernance(&current.Groups[i])
				}
			}
			applyGovernance(&group)

			deriveManifest(&catalog, manifest{Name: "demo", Groups: []manifestGroup{group}})
			if len(catalog.Scopes) != 1 {
				t.Fatalf("derived scope count = %d, want 1", len(catalog.Scopes))
			}
			if got := catalog.Scopes[0].Command; got != test.command {
				t.Fatalf("derived command = %q, want %q", got, test.command)
			}
		})
	}
}

func TestBuildProjectFlatCommandsMatchInvocationVocabulary(t *testing.T) {
	catalog, err := Build(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	commands := make(map[string]bool)
	for _, scope := range catalog.Scopes {
		if scope.Scenario == ProjectManifestIdentity {
			commands[scope.Command] = true
		}
	}

	for _, command := range []string{"setup", "hygiene"} {
		if !commands[command] {
			t.Errorf("project invocation %q is absent from derived vocabulary", command)
		}
	}
	for _, invalid := range []string{"top-level/hygiene", "lifecycle-commands/setup"} {
		if commands[invalid] {
			t.Errorf("organizational manifest path %q leaked into invocation vocabulary", invalid)
		}
	}
}

func TestBuildIncludesProjectManifestWithoutScenarioCollision(t *testing.T) {
	root := repoRoot(t)
	catalog, err := Build(root)
	if err != nil {
		t.Fatal(err)
	}

	var projectScopes int
	for _, scope := range catalog.Scopes {
		if scope.Scenario == ProjectManifestIdentity {
			projectScopes++
		}
	}
	rootManifest, err := os.ReadFile(filepath.Join(root, "cli", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var projectDocument manifest
	if err := json.Unmarshal(rootManifest, &projectDocument); err != nil {
		t.Fatal(err)
	}
	var countGovernedCommands func([]manifestGroup) int
	countGovernedCommands = func(groups []manifestGroup) int {
		count := 0
		for _, group := range groups {
			for _, command := range group.Commands {
				if command.Governance.Effect != "" {
					count++
				}
			}
			count += countGovernedCommands(group.Groups)
		}
		return count
	}
	expectedProjectScopes := countGovernedCommands(projectDocument.Groups)
	if projectScopes != expectedProjectScopes {
		t.Fatalf("project scope count = %d, want one scope per governed root command (%d)", projectScopes, expectedProjectScopes)
	}
	if !catalog.HasScope(ProjectManifestIdentity+":read") || !catalog.HasScope(ProjectManifestIdentity+":write") || !catalog.HasScope(ProjectManifestIdentity+":destructive") {
		t.Fatalf("project effect vocabulary missing from catalog")
	}
	for _, scope := range catalog.Scopes {
		if scope.Scenario == ProjectManifestIdentity {
			continue
		}
		if scope.Value == ProjectManifestIdentity+":"+string(scope.Effect) {
			t.Fatalf("scenario scope collided with project namespace: %#v", scope)
		}
	}
}

func TestBuildOrderingIsDeterministic(t *testing.T) {
	first, err := Build(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	second, err := Build(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("scope catalog changed between identical builds")
	}
	for i := 1; i < len(first.Scopes); i++ {
		if scopeKey(first.Scopes[i-1]) > scopeKey(first.Scopes[i]) {
			t.Fatalf("scopes are not sorted at index %d", i)
		}
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
