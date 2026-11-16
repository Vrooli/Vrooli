package triposrsource

import (
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"testing"
)

func TestPersistentStateSourceChecks(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	findings, err := buildinfo.PersistentStateSourceFindings(root, "internal/tools/triposr-source")
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range findings {
		t.Error(finding)
	}
}
