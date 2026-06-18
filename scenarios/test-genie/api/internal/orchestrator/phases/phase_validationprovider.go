package phases

import (
	"context"
	"fmt"
	"io"
	"time"

	"test-genie/internal/orchestrator/phases/validationprovider"
	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/shared"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

const phaseSourceValidationProvider = "validation-provider"

type Delegated struct {
	Name             Name
	ProviderScenario string
	FindingSource    architecturev1.FindingSource
	Emoji            string
	SkipEnvVar       string
	DetailCommand    string
	Optional         bool
	Timeout          time.Duration
	Description      string
	IncludeExecution bool
	GateEnvVar       string
	DefaultGateMode  validationprovider.GateMode
	Client           DelegatedClient
}

// seam: DelegatedClient lets catalog-declared delegated phases share the
// validation-provider result/pointer path while substituting a provider-specific
// client only when the provider does not expose ScenarioValidationService.
type DelegatedClient func(context.Context, workspace.Environment, io.Writer, validationprovider.Provider) *validationprovider.Result

func delegatedSpec(delegated Delegated) Spec {
	name, ok := NormalizeName(delegated.Name.String())
	if !ok {
		panic("delegated phase name is required")
	}
	delegated.Name = name
	provider := delegated.provider()
	return Spec{
		Name:           name,
		Runner:         providerRunner(provider, delegated.Client),
		Optional:       delegated.Optional,
		DefaultTimeout: delegated.Timeout,
		SkipEnvVar:     delegated.SkipEnvVar,
		Description:    delegated.Description,
		Source:         phaseSourceValidationProvider,
		FindingSource:  delegated.FindingSource,
		Delegated:      &delegated,
	}
}

func (d Delegated) provider() validationprovider.Provider {
	return validationprovider.Provider{
		Phase:            d.Name.String(),
		ProviderScenario: d.ProviderScenario,
		FindingSource:    d.FindingSource,
		Emoji:            d.Emoji,
		DetailCommand:    d.DetailCommand,
		Optional:         d.Optional,
		Timeout:          d.Timeout,
		IncludeExecution: d.IncludeExecution,
		GateEnvVar:       d.GateEnvVar,
		DefaultGateMode:  d.DefaultGateMode,
	}
}

func providerRunner(provider validationprovider.Provider, client DelegatedClient) Runner {
	if client == nil {
		client = defaultDelegatedClient
	}
	return func(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
		return runValidationProviderPhase(ctx, env, logWriter, provider, client)
	}
}

func defaultDelegatedClient(ctx context.Context, env workspace.Environment, _ io.Writer, provider validationprovider.Provider) *validationprovider.Result {
	return validationprovider.Run(ctx, provider, env.ScenarioName)
}

func runValidationProviderPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer, provider validationprovider.Provider, client DelegatedClient) RunReport {
	var summary validationprovider.Summary
	var findings []*architecturev1.ArchitectureFinding
	report := RunPhase(ctx, logWriter, provider.Phase,
		func() (*validationprovider.Result, error) {
			result := client(ctx, env, logWriter, provider)
			if result != nil {
				findings = result.Findings
			}
			return result, nil
		},
		func(r *validationprovider.Result) PhaseResult[shared.Observation] {
			var result shared.RunResult[validationprovider.Summary]
			summaryText := ""
			if r != nil {
				result = r.RunResult
				summary = r.Summary
				if summary.Scenario == "" {
					summary.Scenario = env.ScenarioName
				}
				summaryText = summary.String()
			}
			return ExtractWithSummary(
				result.Success,
				result.Error,
				result.FailureClass,
				result.Remediation,
				result.Observations,
				provider.Emoji,
				fmt.Sprintf("%s validation completed (%s)", provider.Phase, summaryText),
			)
		},
	)

	report.Findings = findings
	writePhasePointer(env, provider.Phase, report, map[string]any{"summary": summary}, logWriter)
	logPhaseStep(logWriter, "%s summary: %s", provider.Phase, summary.String())
	return report
}
