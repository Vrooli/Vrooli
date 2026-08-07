package hygiene

import (
	"context"
	"fmt"

	"github.com/vrooli/vrooli/internal/structureprovider"
)

const structureProviderID = "structure-health"

type structureProvider struct {
	root   string
	client structureprovider.Client
}

func (p structureProvider) ID() string { return structureProviderID }

func (p structureProvider) Run(ctx context.Context, _ Request, report *Report) error {
	if p.client == nil {
		return fmt.Errorf("%w: client is not configured", structureprovider.ErrUnavailable)
	}
	output, err := p.client.Validate(ctx, p.root)
	if err != nil {
		return err
	}
	report.Contract = output
	severity := SeverityInfo
	message := "passed"
	if !output.Success {
		severity = SeverityError
		message = "structure-health project validation failed"
	}
	report.addCheck("repo_contract", output.Success, severity, message)
	for _, check := range output.Report.Checks {
		if check.Passed {
			continue
		}
		report.addFinding(contractFinding(check.Name, check.Message))
	}
	if !output.Schema.Passed {
		report.addFinding(Finding{
			Severity:   SeverityError,
			Code:       "repo_contract_schema",
			Message:    output.Schema.Message,
			Fixability: FixabilityManual,
			NextActions: []Action{{
				Code:    "inspect_repo_contract_schema",
				Message: "Inspect the structure-health project finding and correct the repository contract.",
				Command: "vrooli contract validate --json",
			}},
		})
	}
	return nil
}

func (s Service) structureProvider() Provider {
	if s.StructureProvider != nil {
		return s.StructureProvider
	}
	return structureProvider{root: s.Root, client: structureprovider.NewDefault()}
}
