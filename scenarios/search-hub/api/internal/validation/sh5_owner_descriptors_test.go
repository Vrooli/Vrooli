package validation

import (
	"context"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

func TestSH5OwnerSearchDescriptorsValidateFromRepository(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	service := New(root)
	for _, scenario := range []string{"scenario-completeness-scoring", "test-genie", "scenario-dependency-analyzer", "code-facts", "architecture-cartographer"} {
		t.Run(scenario, func(t *testing.T) {
			report, err := service.ValidateScenarioWithOptions(context.Background(), scenario, "", Options{})
			if err != nil {
				t.Fatalf("validate %s descriptor: %v", scenario, err)
			}
			if report.Summary.Errors != 0 {
				t.Fatalf("descriptor has %d validation errors: %+v", report.Summary.Errors, report.Findings)
			}
		})
	}
}
