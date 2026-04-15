package repocontract

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScenarioAndResourcePaths(t *testing.T) {
	root := fixtureRoot(t)
	contract := mustLoadDefault(t, root)

	scenarioRoot, err := contract.ScenarioRoot(root, "test-genie")
	if err != nil {
		t.Fatalf("ScenarioRoot() error = %v", err)
	}
	if want := mustScenarioRoot(t, contract, root, "test-genie"); scenarioRoot != want {
		t.Fatalf("ScenarioRoot() = %q, want %q", scenarioRoot, want)
	}

	resourceRoot, err := contract.ResourceRoot(root, "postgres")
	if err != nil {
		t.Fatalf("ResourceRoot() error = %v", err)
	}
	if want := mustResourceRoot(t, contract, root, "postgres"); resourceRoot != want {
		t.Fatalf("ResourceRoot() = %q, want %q", resourceRoot, want)
	}

	servicePath, err := contract.ScenarioFile(root, "test-genie", "service")
	if err != nil {
		t.Fatalf("ScenarioFile() error = %v", err)
	}
	if want := filepath.Join(scenarioRoot, ".vrooli", "service.json"); servicePath != want {
		t.Fatalf("ScenarioFile() = %q, want %q", servicePath, want)
	}

	resourceManifest, err := contract.ResourceFile(root, "postgres", "manifest")
	if err != nil {
		t.Fatalf("ResourceFile() error = %v", err)
	}
	if want := filepath.Join(resourceRoot, "resource.json"); resourceManifest != want {
		t.Fatalf("ResourceFile() = %q, want %q", resourceManifest, want)
	}
}

func TestPathHelpersValidateIdentifiersAndKeys(t *testing.T) {
	contract := validContract(t)

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
	contract := validContract(t)

	got := mustTopLevelDir(t, contract, "/repo", "packages")
	if got != filepath.Join("/repo", "packages") {
		t.Fatalf("TopLevelDir() = %q", got)
	}

	templateDir := mustTopLevelDir(t, contract, "/repo", "templates")
	if templateDir != filepath.Join("/repo", "templates") {
		t.Fatalf("TopLevelDir(templates) = %q", templateDir)
	}

	if got := ScenarioRoot("/repo", "demo"); got != filepath.Join("/repo", "scenarios", "demo") {
		t.Fatalf("ScenarioRoot() = %q", got)
	}
	if got := ScenarioTemplateRoot("/repo"); got != filepath.Join("/repo", "templates", "scenarios") {
		t.Fatalf("ScenarioTemplateRoot() = %q", got)
	}
	if got := ResourceTemplateRoot("/repo"); got != filepath.Join("/repo", "templates", "resources") {
		t.Fatalf("ResourceTemplateRoot() = %q", got)
	}
}

func TestStandaloneScenarioRootUsesContractLayoutWhenAvailable(t *testing.T) {
	root := t.TempDir()
	doc := validContractDoc(t)
	doc.Root.Markers.RequiredDirs[1] = "apps"
	doc.Layout.ScenarioDir = "apps"

	writeContractFile(t, root, doc)
	for _, dir := range []string{"apps", "resources", "templates", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}

	want := filepath.Join(mustTopLevelDir(t, mustLoadDefault(t, root), root, "scenarios"), "demo")
	if got := ScenarioRoot(root, "demo"); got != want {
		t.Fatalf("ScenarioRoot() = %q, want %q", got, want)
	}
}

func TestContractAccessorsAreDefensive(t *testing.T) {
	contract := mustLoadDefault(t, fixtureRoot(t))

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
