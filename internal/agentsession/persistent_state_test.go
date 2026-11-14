package agentsession_test

import (
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/buildinfo"
)

// The governed build applies these checks when it stamps the persistent-state
// contract; running them here surfaces a violation, or an unclassified
// constant, during ordinary development instead of as a refused activation.
func TestPersistentStateSourceChecks(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	findings, err := buildinfo.PersistentStateSourceFindings(root, "internal/agentsession")
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range findings {
		t.Error(finding)
	}
}
