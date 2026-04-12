package repocontract

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRepoRootFromPathAlias(t *testing.T) {
	root := repoRoot(t)
	got, err := FindRepoRootFromPath(filepath.Join(root, "packages", "repo-contract-go"))
	if err != nil {
		t.Fatalf("FindRepoRootFromPath() error = %v", err)
	}
	if got != root {
		t.Fatalf("FindRepoRootFromPath() = %q, want %q", got, root)
	}
}

func TestFindRepoRootFromCWD(t *testing.T) {
	root := repoRoot(t)
	oldGetwd := getwdPath
	getwdPath = func() (string, error) {
		return filepath.Join(root, "packages", "repo-contract-go"), nil
	}
	t.Cleanup(func() { getwdPath = oldGetwd })

	got, err := FindRepoRootFromCWD()
	if err != nil {
		t.Fatalf("FindRepoRootFromCWD() error = %v", err)
	}
	if got != root {
		t.Fatalf("FindRepoRootFromCWD() = %q, want %q", got, root)
	}
}

func TestFindRepoRootFromEnvOrCWD(t *testing.T) {
	root := repoRoot(t)
	t.Setenv(defaultSourceRootEnvVar, filepath.Join(root, "cmd"))
	t.Setenv(defaultRepoRootEnvVar, "")

	got, err := FindRepoRootFromEnvOrCWD()
	if err != nil {
		t.Fatalf("FindRepoRootFromEnvOrCWD() error = %v", err)
	}
	if got != root {
		t.Fatalf("FindRepoRootFromEnvOrCWD() = %q, want %q", got, root)
	}
}

func TestFindRepoRootFromEnvOrCWDFallsBackToExecutable(t *testing.T) {
	root := repoRoot(t)
	t.Setenv(defaultSourceRootEnvVar, "")
	t.Setenv(defaultRepoRootEnvVar, "")

	oldGetwd := getwdPath
	oldExec := executablePath
	getwdPath = func() (string, error) {
		return "", os.ErrNotExist
	}
	executablePath = func() (string, error) {
		return filepath.Join(root, "cmd", "vrooli", "vrooli"), nil
	}
	t.Cleanup(func() {
		getwdPath = oldGetwd
		executablePath = oldExec
	})

	got, err := FindRepoRootFromEnvOrCWD()
	if err != nil {
		t.Fatalf("FindRepoRootFromEnvOrCWD() error = %v", err)
	}
	if got != root {
		t.Fatalf("FindRepoRootFromEnvOrCWD() = %q, want %q", got, root)
	}
}

func TestLoadDefaultFromEnvOrCWD(t *testing.T) {
	root := repoRoot(t)
	t.Setenv(defaultRepoRootEnvVar, root)
	t.Setenv(defaultSourceRootEnvVar, "")

	contract, gotRoot, err := LoadDefaultFromEnvOrCWD()
	if err != nil {
		t.Fatalf("LoadDefaultFromEnvOrCWD() error = %v", err)
	}
	if gotRoot != root {
		t.Fatalf("LoadDefaultFromEnvOrCWD() root = %q, want %q", gotRoot, root)
	}
	if contract.Version() != "1.0.0" {
		t.Fatalf("LoadDefaultFromEnvOrCWD() contract version = %q", contract.Version())
	}
}

func TestResolveRepoRoot(t *testing.T) {
	root := repoRoot(t)
	t.Setenv(defaultRepoRootEnvVar, root)
	t.Setenv(defaultSourceRootEnvVar, "")

	got, err := ResolveRepoRoot()
	if err != nil {
		t.Fatalf("ResolveRepoRoot() error = %v", err)
	}
	if got != root {
		t.Fatalf("ResolveRepoRoot() = %q, want %q", got, root)
	}
}

func TestResolveScenarioPathAndFile(t *testing.T) {
	root := repoRoot(t)

	scenarioPath, err := ResolveScenarioPath(root, "test-genie")
	if err != nil {
		t.Fatalf("ResolveScenarioPath() error = %v", err)
	}
	wantPath := filepath.Join(root, "scenarios", "test-genie")
	if scenarioPath != wantPath {
		t.Fatalf("ResolveScenarioPath() = %q, want %q", scenarioPath, wantPath)
	}

	servicePath, err := ResolveScenarioFile(root, "test-genie", "service")
	if err != nil {
		t.Fatalf("ResolveScenarioFile() error = %v", err)
	}
	wantService := filepath.Join(wantPath, ".vrooli", "service.json")
	if servicePath != wantService {
		t.Fatalf("ResolveScenarioFile() = %q, want %q", servicePath, wantService)
	}
}

func TestScenarioExists(t *testing.T) {
	root := repoRoot(t)

	ok, err := ScenarioExists(root, "test-genie")
	if err != nil {
		t.Fatalf("ScenarioExists(existing) error = %v", err)
	}
	if !ok {
		t.Fatal("ScenarioExists(existing) = false, want true")
	}

	ok, err = ScenarioExists(root, "definitely-missing-scenario")
	if err != nil {
		t.Fatalf("ScenarioExists(missing) error = %v", err)
	}
	if ok {
		t.Fatal("ScenarioExists(missing) = true, want false")
	}
}

func TestFileMatchCount(t *testing.T) {
	root := repoRoot(t)

	count, err := FileMatchCount(root, "packages/repo-contract-go/*.go")
	if err != nil {
		t.Fatalf("FileMatchCount() error = %v", err)
	}
	if count == 0 {
		t.Fatal("FileMatchCount() = 0, want at least one match")
	}
}
