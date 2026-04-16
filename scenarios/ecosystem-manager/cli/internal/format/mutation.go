package format

import (
	"os"

	"github.com/vrooli/cli-core/cliapp"
)

// MutationResult renders the standard mutation contract with optional details and next steps.
func MutationResult(result string, details string, nextSteps []string) error {
	report := cliapp.MutationReport{
		Result:      []string{result},
		NextCommand: nextSteps,
	}
	if details != "" {
		report.Changes = []string{details}
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
