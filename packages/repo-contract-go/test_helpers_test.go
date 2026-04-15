package repocontract

import (
	"os"
	"path/filepath"
	"testing"

	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	root := testkitgo.ProjectRoot(t)
	if _, err := os.Stat(filepath.Join(root, ".vrooli", "repo-contract.json")); err != nil {
		t.Fatalf("repo root missing live contract: %v", err)
	}
	return root
}

func fixtureRoot(t *testing.T, opts ...testkitgo.RepoFixtureOption) string {
	t.Helper()
	fixture := testkitgo.NewRepoFixture(t, opts...)
	fixture.WriteRepoContract(t)
	fixture.WriteScenarioStub(t, "test-genie")
	fixture.WriteResourceStub(t, "postgres")
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "packages", "repo-contract-go", "load.go"), "package repocontract\n")
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, "cmd", "vrooli", "main.go"), "package main\n")
	return fixture.Root
}

func mustLoadDefault(t *testing.T, root string) *Contract {
	t.Helper()
	contract, err := LoadDefault(root)
	if err != nil {
		t.Fatalf("LoadDefault() error = %v", err)
	}
	return contract
}

func writeContractFile(t *testing.T, dir string, doc contractDoc) string {
	t.Helper()
	path := filepath.Join(dir, ".vrooli", "repo-contract.json")
	testkitgo.WriteJSON(t, path, doc)
	return path
}

func validContractDoc(t *testing.T) contractDoc {
	t.Helper()
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	return testkitgo.ReadJSONFileInto[contractDoc](t, filepath.Join(fixture.Root, ".vrooli", "repo-contract.json"))
}

func validContract(t *testing.T) *Contract {
	t.Helper()
	doc := validContractDoc(t)
	return &Contract{doc: deepCopyContractDoc(doc)}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func mustScenarioRoot(t *testing.T, contract *Contract, root, name string) string {
	t.Helper()
	path, err := contract.ScenarioRoot(root, name)
	if err != nil {
		t.Fatalf("ScenarioRoot(%q) error = %v", name, err)
	}
	return path
}

func mustResourceRoot(t *testing.T, contract *Contract, root, name string) string {
	t.Helper()
	path, err := contract.ResourceRoot(root, name)
	if err != nil {
		t.Fatalf("ResourceRoot(%q) error = %v", name, err)
	}
	return path
}

func mustScenarioFile(t *testing.T, contract *Contract, root, name, key string) string {
	t.Helper()
	path, err := contract.ScenarioFile(root, name, key)
	if err != nil {
		t.Fatalf("ScenarioFile(%q, %q) error = %v", name, key, err)
	}
	return path
}

func mustResourceFile(t *testing.T, contract *Contract, root, name, key string) string {
	t.Helper()
	path, err := contract.ResourceFile(root, name, key)
	if err != nil {
		t.Fatalf("ResourceFile(%q, %q) error = %v", name, key, err)
	}
	return path
}

func mustTopLevelDir(t *testing.T, contract *Contract, root, key string) string {
	t.Helper()
	path, err := contract.TopLevelDir(root, key)
	if err != nil {
		t.Fatalf("TopLevelDir(%q) error = %v", key, err)
	}
	return path
}
