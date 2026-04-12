package repocontract

import (
	"path/filepath"
	"testing"
)

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

	resourceRoot, err := contract.ResourceRoot(root, "postgres")
	if err != nil {
		t.Fatalf("ResourceRoot() error = %v", err)
	}
	if want := filepath.Join(root, "resources", "postgres"); resourceRoot != want {
		t.Fatalf("ResourceRoot() = %q, want %q", resourceRoot, want)
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

func TestPathHelpersValidateIdentifiersAndKeys(t *testing.T) {
	contract := validContract()

	_, err := contract.ScenarioRoot("/repo", "../bad")
	assertErrorKind(t, err, ErrInvalidInput)

	_, err = contract.ResourceRoot("/repo", "bad/path")
	assertErrorKind(t, err, ErrInvalidInput)

	_, err = contract.ScenarioFile("/repo", "demo", "unknown")
	assertErrorKind(t, err, ErrNotFound)

	_, err = contract.ResourceFile("/repo", "demo", "unknown")
	assertErrorKind(t, err, ErrNotFound)

	_, err = contract.TopLevelDir("/repo", "unknown")
	assertErrorKind(t, err, ErrNotFound)
}

func TestTopLevelDirAndStandaloneScenarioRoot(t *testing.T) {
	contract := validContract()

	got, err := contract.TopLevelDir("/repo", "packages")
	if err != nil {
		t.Fatalf("TopLevelDir() error = %v", err)
	}
	if got != filepath.Join("/repo", "packages") {
		t.Fatalf("TopLevelDir() = %q", got)
	}

	if got := ScenarioRoot("/repo", "demo"); got != filepath.Join("/repo", "scenarios", "demo") {
		t.Fatalf("ScenarioRoot() = %q", got)
	}
}

func TestContractAccessorsAreDefensive(t *testing.T) {
	contract := mustLoadDefault(t, repoRoot(t))

	env := contract.EnvironmentVariables()
	env["repo_root"] = "BROKEN"
	if contract.EnvironmentVariables()["repo_root"] != "VROOLI_ROOT" {
		t.Fatal("EnvironmentVariables() exposed mutable internal state")
	}

	markers := contract.RootMarkers()
	markers.RequiredDirs[0] = "BROKEN"
	if contract.RootMarkers().RequiredDirs[0] == "BROKEN" {
		t.Fatal("RootMarkers() exposed mutable internal state")
	}

	scenario := contract.Scenario()
	scenario.WellKnownPaths["service"] = "BROKEN"
	if contract.Scenario().WellKnownPaths["service"] == "BROKEN" {
		t.Fatal("Scenario() exposed mutable internal state")
	}

	resource := contract.Resource()
	resource.WellKnownPaths["docs"] = "BROKEN"
	if contract.Resource().WellKnownPaths["docs"] == "BROKEN" {
		t.Fatal("Resource() exposed mutable internal state")
	}

	scopes := contract.SandboxFullRepoScopes()
	scopes[0] = "BROKEN"
	if contract.SandboxFullRepoScopes()[0] == "BROKEN" {
		t.Fatal("SandboxFullRepoScopes() exposed mutable internal state")
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
