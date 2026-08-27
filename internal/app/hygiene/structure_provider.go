package hygiene

import (
	"context"
	"fmt"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/internal/structureprovider"
)

const structureProviderID = "structure-health"

// The hygiene lane asks the authority for the project target only. The full
// target walk remains available to contract validation, but duplicating that
// fleet traversal here made a short structural gate depend on unrelated
// resource and code-facts work.
const structureProviderBudget = tuning.StandardOperationTimeout

type structureProvider struct {
	root   string
	client structureprovider.Client
}

func (p structureProvider) ID() string { return structureProviderID }

func (p structureProvider) Budget() time.Duration { return structureProviderBudget }

func (p structureProvider) Run(ctx context.Context, _ Request, report *Report) error {
	if p.client == nil {
		p.reportUnavailable(report, fmt.Sprintf("%v: client is not configured", structureprovider.ErrUnavailable))
		return nil
	}
	output, err := p.client.Validate(ctx, p.root)
	if err != nil {
		p.reportUnavailable(report, err.Error())
		return nil
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
	for _, finding := range output.Findings {
		severity := SeverityWarning
		if finding.Severity == "error" || finding.Severity == "critical" {
			severity = SeverityError
		}
		report.addFinding(Finding{
			Severity:   severity,
			Code:       finding.Code,
			Path:       finding.TargetKind + ":" + finding.TargetID,
			Locations:  []string{finding.Location},
			Message:    finding.Message,
			Why:        finding.Remediation,
			Fixability: FixabilityManual,
		})
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

// reportUnavailable records an unguarded contract rather than pretending that
// the repository failed structural validation. Structure Health being
// unreachable says nothing about plan, dependency, or freshness hygiene, so
// those providers must still run and their fixes must still apply. The warning
// remains visible in the summary, while --fail-on error only fails on actual
// structural findings.
func (p structureProvider) reportUnavailable(report *Report, reason string) {
	message := fmt.Sprintf("structure-health contract unguarded: provider unavailable: %s", reason)
	action := Action{
		Code:       "inspect_structure_health",
		Message:    "Inspect the structure-health validation surface and rerun hygiene after correcting the provider failure.",
		Command:    "vrooli contract validate --json",
		Fixability: FixabilityGuided,
	}
	report.addCheck("repo_contract", false, SeverityWarning, message)
	report.addFinding(Finding{
		Severity:    SeverityWarning,
		Code:        "repo_contract_unguarded",
		Message:     message,
		Why:         "Structure Health owns repository structural validation; root hygiene only aggregates the provider result.",
		Fixability:  FixabilityGuided,
		NextActions: []Action{action},
	})
	report.Actions = append(report.Actions, action)
}

func (s Service) structureProvider() Provider {
	if s.StructureProvider != nil {
		return s.StructureProvider
	}
	return structureProvider{root: s.Root, client: structureprovider.NewProjectDefault()}
}
