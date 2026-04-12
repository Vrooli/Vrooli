package repocontract

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadLiveContract(t *testing.T) {
	root := repoRoot(t)
	contract, err := LoadDefault(root)
	if err != nil {
		t.Fatalf("LoadDefault() error = %v", err)
	}

	if contract.Version() != "1.0.0" {
		t.Fatalf("Version() = %q", contract.Version())
	}
	if got := contract.Layout().ScenarioDir; got != "scenarios" {
		t.Fatalf("Layout().ScenarioDir = %q", got)
	}
	if got := contract.EnvironmentVariables()["repo_root"]; got != "VROOLI_ROOT" {
		t.Fatalf("repo_root env = %q", got)
	}
}

func TestLoadRejectsInvalidFixtures(t *testing.T) {
	tests := []struct {
		name string
		file string
		kind ErrorKind
	}{
		{name: "unsupported version", file: "invalid-unsupported-version.json", kind: ErrUnsupportedVersion},
		{name: "absolute path", file: "invalid-absolute-path.json", kind: ErrInvalidContract},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(filepath.Join("testdata", tt.file))
			if err == nil {
				t.Fatal("Load() expected error")
			}
			var target *Error
			if !errors.As(err, &target) {
				t.Fatalf("Load() error type = %T", err)
			}
			if target.Kind != tt.kind {
				t.Fatalf("error kind = %q, want %q", target.Kind, tt.kind)
			}
		})
	}
}

func TestFindRepoRoot(t *testing.T) {
	root := repoRoot(t)
	start := filepath.Join(root, "packages", "repo-contract-go")

	got, err := FindRepoRoot(start)
	if err != nil {
		t.Fatalf("FindRepoRoot() error = %v", err)
	}
	if got != root {
		t.Fatalf("FindRepoRoot() = %q, want %q", got, root)
	}
}

func TestScenarioAndResourcePaths(t *testing.T) {
	root := repoRoot(t)
	contract := mustLoadDefault(t, root)

	scenarioRoot, err := contract.ScenarioRoot(root, "test-genie")
	if err != nil {
		t.Fatalf("ScenarioRoot() error = %v", err)
	}
	if want := filepath.Join(root, "scenarios", "test-genie"); scenarioRoot != want {
		t.Fatalf("ScenarioRoot() = %q, want %q", scenarioRoot, want)
	}

	servicePath, err := contract.ScenarioFile(root, "test-genie", "service")
	if err != nil {
		t.Fatalf("ScenarioFile() error = %v", err)
	}
	if want := filepath.Join(root, "scenarios", "test-genie", ".vrooli", "service.json"); servicePath != want {
		t.Fatalf("ScenarioFile() = %q, want %q", servicePath, want)
	}

	resourceManifest, err := contract.ResourceFile(root, "postgres", "manifest")
	if err != nil {
		t.Fatalf("ResourceFile() error = %v", err)
	}
	if want := filepath.Join(root, "resources", "postgres", "resource.json"); resourceManifest != want {
		t.Fatalf("ResourceFile() = %q, want %q", resourceManifest, want)
	}
}

func TestMatchRepoGlob(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
		wantErr bool
	}{
		{name: "doublestar match", pattern: "scenarios/test-genie/**", path: "scenarios/test-genie/api/main.go", want: true},
		{name: "native separators normalize", pattern: `scenarios\test-genie\**`, path: `scenarios\test-genie\api\main.go`, want: true},
		{name: "absolute rejected", pattern: "/tmp/**", path: "scenarios/test-genie/api/main.go", wantErr: true},
		{name: "traversal rejected", pattern: "../**", path: "scenarios/test-genie/api/main.go", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MatchRepoGlob(tt.pattern, tt.path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("MatchRepoGlob() expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("MatchRepoGlob() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("MatchRepoGlob() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAffectedScenarios(t *testing.T) {
	contract := mustLoadDefault(t, repoRoot(t))
	got := contract.AffectedScenarios([]string{
		"scenarios/test-genie/**",
		"scenarios/test-genie/api/*.go",
		"scenarios/swarm-manager/ui/**",
		"packages/api-core/**",
		"scenarios/*/docs/**",
	})

	want := []string{"swarm-manager", "test-genie"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("AffectedScenarios() = %v, want %v", got, want)
	}
}

func TestResolveProfile(t *testing.T) {
	contract := mustLoadDefault(t, repoRoot(t))

	resolved, err := contract.ResolveProfile("mini_vrooli_bundle", ResolveParams{
		Values: map[string]string{
			"scenario": "scenario-to-cloud",
		},
		Lists: map[string][]string{
			"resources": {"postgres", "redis"},
		},
	})
	if err != nil {
		t.Fatalf("ResolveProfile() error = %v", err)
	}

	if !contains(resolved.Include, "scenarios/scenario-to-cloud") {
		t.Fatalf("resolved include missing scenario path: %v", resolved.Include)
	}
	if !contains(resolved.Include, "resources/postgres") || !contains(resolved.Include, "resources/redis") {
		t.Fatalf("resolved include missing resource paths: %v", resolved.Include)
	}
	if !contains(resolved.OptionalInclude, "go.mod") {
		t.Fatalf("resolved optional include missing go.mod: %v", resolved.OptionalInclude)
	}
}

func TestResolveProfileRejectsMissingRequiredValue(t *testing.T) {
	contract := mustLoadDefault(t, repoRoot(t))
	_, err := contract.ResolveProfile("mini_vrooli_bundle", ResolveParams{})
	if err == nil {
		t.Fatal("ResolveProfile() expected error")
	}
}

func TestContractDataIsDefensivelyCopied(t *testing.T) {
	contract := mustLoadDefault(t, repoRoot(t))

	env := contract.EnvironmentVariables()
	env["repo_root"] = "BROKEN"
	if contract.EnvironmentVariables()["repo_root"] != "VROOLI_ROOT" {
		t.Fatal("EnvironmentVariables() exposed mutable internal state")
	}

	profile, err := contract.Profile("mini_vrooli_bundle")
	if err != nil {
		t.Fatalf("Profile() error = %v", err)
	}
	profile.Include[0] = "BROKEN"
	profileAgain, err := contract.Profile("mini_vrooli_bundle")
	if err != nil {
		t.Fatalf("Profile() second call error = %v", err)
	}
	if profileAgain.Include[0] == "BROKEN" {
		t.Fatal("Profile() exposed mutable internal state")
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	root := filepath.Clean(filepath.Join(dir, "..", ".."))
	if _, err := os.Stat(filepath.Join(root, ".vrooli", "repo-contract.json")); err != nil {
		t.Fatalf("repo root missing live contract: %v", err)
	}
	return root
}

func mustLoadDefault(t *testing.T, root string) *Contract {
	t.Helper()
	contract, err := LoadDefault(root)
	if err != nil {
		t.Fatalf("LoadDefault() error = %v", err)
	}
	return contract
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
