package testutil

import (
	"os"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
)

type RepoFixture struct {
	Root string
	Home string
}

type RepoSupportDocs struct {
	RepoContractDoc []string
	ContributingDoc []string
	AgentsDoc       []string
	SkillDoc        []string
}

func NewRepoFixture(t *testing.T) RepoFixture {
	t.Helper()
	fixture := testkitgo.NewRepoFixture(t)
	return RepoFixture{
		Root: fixture.Root,
		Home: fixture.Home,
	}
}

func ProjectRoot(t *testing.T) string {
	t.Helper()
	return testkitgo.ProjectRoot(t)
}

func WriteFile(t *testing.T, path, contents string) {
	t.Helper()
	testkitgo.WriteFile(t, path, contents)
}

func WriteExecutable(t *testing.T, path, contents string) string {
	t.Helper()
	return testkitgo.WriteExecutable(t, path, contents)
}

func WriteJSON(t *testing.T, path string, value any) {
	t.Helper()
	testkitgo.WriteJSON(t, path, value)
}

func WriteJSONMode(t *testing.T, path string, value any, mode os.FileMode) {
	t.Helper()
	testkitgo.WriteJSONMode(t, path, value, mode)
}

func WriteRawJSON(t *testing.T, path, raw string, mode os.FileMode) {
	t.Helper()
	testkitgo.WriteRawJSON(t, path, raw, mode)
}

func ReadJSONFile(t *testing.T, path string) map[string]any {
	t.Helper()
	return testkitgo.ReadJSONFile(t, path)
}

func ContainsString(values []string, target string) bool {
	return testkitgo.ContainsString(values, target)
}

func WriteRepoContract(t *testing.T, root, scenarioDir string) {
	t.Helper()
	testkitgo.WriteRepoContract(t, root, scenarioDir)
}

func DefaultRepoSupportDocs() RepoSupportDocs {
	docs := testkitgo.DefaultRepoSupportDocs()
	return RepoSupportDocs{
		RepoContractDoc: docs.RepoContractDoc,
		ContributingDoc: docs.ContributingDoc,
		AgentsDoc:       docs.AgentsDoc,
		SkillDoc:        docs.SkillDoc,
	}
}

func WriteRepoContractExceptions(t *testing.T, root string) {
	t.Helper()
	testkitgo.WriteRepoContractExceptions(t, root)
}

func WriteRepoSupportDocs(t *testing.T, root string, docs RepoSupportDocs) {
	t.Helper()
	testkitgo.WriteRepoSupportDocs(t, root, testkitgo.RepoSupportDocs{
		RepoContractDoc: docs.RepoContractDoc,
		ContributingDoc: docs.ContributingDoc,
		AgentsDoc:       docs.AgentsDoc,
		SkillDoc:        docs.SkillDoc,
	})
}

func MustLoadDefaultContract(t *testing.T, root string) *repocontract.Contract {
	t.Helper()
	contract, err := repocontract.LoadDefault(root)
	if err != nil {
		t.Fatalf("LoadDefault(%s): %v", root, err)
	}
	return contract
}
