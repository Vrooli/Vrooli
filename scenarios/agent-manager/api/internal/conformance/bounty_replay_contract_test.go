package conformance

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBountyReplaySelfDeclarationsMeetRoleOnlyContract(t *testing.T) {
	root := filepath.Join("..", "..", "..", ".vrooli", "agent-manager")
	roles := map[string]bool{"code.default": true, "code.judgment": true}
	for _, name := range []string{
		"bounty-solver.json",
		"bounty-scorer.json",
		"bounty-solver-workflow.json",
		"bounty-scorer-workflow.json",
	} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		var report Report
		path := filepath.Join(root, name)
		if name == "bounty-solver.json" || name == "bounty-scorer.json" {
			validateDeclaredProfile(&report, "agent-manager", roles, path, data)
		} else {
			validateDeclaredWorkflow(&report, "agent-manager", roles, path, data)
		}
		if len(report.Findings) != 0 {
			t.Errorf("%s has conformance findings: %+v", name, report.Findings)
		}
	}
}
