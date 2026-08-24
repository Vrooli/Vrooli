package hygiene

import (
	"context"
	"fmt"
	"time"

	"github.com/vrooli/vrooli/internal/structureprovider"
)

const structureProviderID = "structure-health"

type structureProvider struct {
	root   string
	client structureprovider.Client
}

func (p structureProvider) ID() string { return structureProviderID }

func (p structureProvider) Budget() time.Duration { return 10 * time.Second }

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

// reportUnavailable records a provider-level failure as a report finding rather
// than an aborting error. Structure Health being unreachable says nothing about
// plan, dependency, or freshness hygiene, so those providers must still run and
// their fixes must still apply. The finding is error-severity, so the run still
// fails; it just fails after doing the work it could do. Mirrors
// sdaFreshnessProvider.reportUnavailable and plansProvider.reportCanonicalFailure.
func (p structureProvider) reportUnavailable(report *Report, reason string) {
	message := fmt.Sprintf("structure-health provider unavailable: %s", reason)
	action := Action{
		Code:       "inspect_structure_health",
		Message:    "Inspect the structure-health validation surface and rerun hygiene after correcting the provider failure.",
		Command:    "vrooli contract validate --json",
		Fixability: FixabilityGuided,
	}
	report.addCheck("repo_contract", false, SeverityError, message)
	report.addFinding(Finding{
		Severity:    SeverityError,
		Code:        "repo_contract_provider",
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
	return structureProvider{root: s.Root, client: structureprovider.NewDefault()}
}
