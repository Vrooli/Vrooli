package phases

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"test-genie/internal/orchestrator/phasepolicy"
	"test-genie/internal/orchestrator/phaseregistry"
	"test-genie/internal/orchestrator/phases/validationprovider"
	"test-genie/internal/orchestrator/providerdescriptor"
	"test-genie/internal/orchestrator/runnability"
	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/shared"

	"github.com/vrooli/vrooli/packages/proto/architecture/findingid"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
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
	DisplayName      string
	Description      string
	IncludeExecution bool
	CapabilitySubset []string
	DeliveryMode     string
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
		DisplayName:    delegated.DisplayName,
		Description:    delegated.Description,
		Source:         phaseSourceValidationProvider,
		FindingSource:  delegated.FindingSource,
		Delegated:      &delegated,
	}
}

// ValidationProviderSpec binds a descriptor-backed phase to the shared
// ScenarioValidationService runner. Descriptor registry construction owns the
// provider metadata; this exported seam keeps the runner implementation in the
// phases package instead of duplicating it in registry code.
func ValidationProviderSpec(delegated Delegated) Spec {
	return delegatedSpec(delegated)
}

func ValidationProviderRegistryBindings() map[string]phaseregistry.RunnerBinding {
	return map[string]phaseregistry.RunnerBinding{
		phaseregistry.SourceValidationProvider: func(descriptor providerdescriptor.Descriptor, findingSource architecturev1.FindingSource) (any, error) {
			return ValidationProviderSpecFromDescriptor(descriptor, findingSource)
		},
	}
}

// ValidationProviderSpecFromDescriptor binds a provider-owned Test Genie
// descriptor to the Test Genie-owned validation-provider runner. Descriptor
// files own phase metadata; this function owns the Spec projection needed by
// the orchestrator.
func ValidationProviderSpecFromDescriptor(descriptor providerdescriptor.Descriptor, findingSource architecturev1.FindingSource) (Spec, error) {
	name, ok := NormalizeName(descriptor.Phase)
	if !ok {
		return Spec{}, fmt.Errorf("invalid phase %q", descriptor.Phase)
	}
	delegated := Delegated{
		Name:             name,
		ProviderScenario: descriptor.Scenario,
		FindingSource:    findingSource,
		Optional:         legacyOptional(descriptor.Policy.Policy),
		Timeout:          descriptor.TimeoutValue,
		DisplayName:      descriptor.DisplayName,
		Description:      descriptor.Description,
		IncludeExecution: descriptor.Validation.Execution,
		DeliveryMode:     descriptor.Validation.DeliveryMode,
		CapabilitySubset: append([]string(nil), descriptor.Validation.CapabilitySubset...),
	}
	spec := delegatedSpec(delegated)
	spec.Description = descriptor.Description
	spec.Source = descriptor.Source
	spec.DefaultTimeout = descriptor.TimeoutValue
	spec.Doc = descriptor.Docs.Path
	spec.Policy = descriptor.Policy.Policy
	spec.Optional = legacyOptional(spec.Policy)
	spec.FindingSource = findingSource
	spec.ProfileMembership = append([]string(nil), descriptor.ProfileMembership...)
	spec.FreshnessRequirement = descriptor.FreshnessRequirement
	spec.PhaseClass = descriptor.PhaseClass
	spec.RuntimeClass = descriptor.RuntimeClass
	spec.Concurrency = Concurrency{Mode: descriptor.Concurrency.Mode, Reason: descriptor.Concurrency.Reason}
	spec.Determinism = Determinism{
		Default:      descriptor.Determinism.Default,
		Inputs:       append([]string(nil), descriptor.Determinism.Inputs...),
		Reason:       descriptor.Determinism.Reason,
		Capabilities: map[string]DeterminismOverride{},
	}
	for capability, override := range descriptor.Determinism.Capabilities {
		spec.Determinism.Capabilities[capability] = DeterminismOverride{Mode: override.Mode, Inputs: append([]string(nil), override.Inputs...), Reason: override.Reason}
	}
	spec.Dimensions = append([]string(nil), descriptor.Dimensions...)
	spec.Capabilities = runnability.PhaseCapabilities{
		Phase:                     name.String(),
		NeedsUI:                   descriptor.Runnability.NeedsUI,
		NeedsAPI:                  descriptor.Runnability.NeedsAPI,
		MutatesLifecycle:          descriptor.Runnability.MutatesLifecycle,
		LifecycleDecisionDeferred: descriptor.Runnability.LifecycleDecisionDeferred,
		DBIsolation:               parseDescriptorDBIsolation(descriptor.Runnability.DBIsolation),
		// Provider-delegated phases own dependency and readiness checks. Test
		// Genie sends the uniform validation request and consumes the provider
		// verdict; it must not interpret a provider's internal resource graph or
		// probe any provider on its behalf.
		Optional: spec.Optional,
	}
	return spec, nil
}

func legacyOptional(policy phasepolicy.Policy) bool {
	switch policy.Unavailable {
	case phasepolicy.UnavailableSkipWithoutFailing, phasepolicy.UnavailableAdvisory:
		return true
	default:
		return policy.ProviderReadiness == phasepolicy.ProviderReadinessBestEffort
	}
}

func parseDescriptorDBIsolation(value string) runnability.DBIsolation {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "routed":
		return runnability.DBIsolationRouted
	default:
		return runnability.DBIsolationNone
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
		CapabilitySubset: append([]string(nil), d.CapabilitySubset...),
		DeliveryMode:     d.DeliveryMode,
		GateEnvVar:       d.GateEnvVar,
		DefaultGateMode:  d.DefaultGateMode,
	}
}

func providerRunner(provider validationprovider.Provider, client DelegatedClient) Runner {
	if client == nil {
		client = defaultDelegatedClient
	}
	return func(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
		bound := provider
		if bound.DeliveryMode == "durable-run" {
			bound.OnStarted = func(ref validationprovider.RunReference) {
				writeDurableChildReference(env, bound.Phase, bound.ProviderScenario, ref, logWriter)
			}
		}
		return runValidationProviderPhase(ctx, env, logWriter, bound, client)
	}
}

func defaultDelegatedClient(ctx context.Context, env workspace.Environment, _ io.Writer, provider validationprovider.Provider) *validationprovider.Result {
	// env.ScenarioDir is the resolved physical scenario directory; sending it as
	// the request path lets providers validate scenarios that live outside the
	// repo scenarios/ registry (e.g. deep template validation's temp scenario).
	if provider.DeliveryMode == "durable-run" {
		return validationprovider.RunDurable(ctx, provider, env.ScenarioName, env.ScenarioDir, env.RunID)
	}
	if env.TargetKind != "" && env.TargetKind != "scenario" {
		return validationprovider.RunTarget(ctx, provider, &commonv1.ValidationTarget{
			Kind: targetKindProto(env.TargetKind), Id: env.TargetID, Root: env.TargetRoot,
		}, env.ScenarioDir)
	}
	return validationprovider.Run(ctx, provider, env.ScenarioName, env.ScenarioDir)
}

func targetKindProto(kind string) commonv1.ValidationTargetKind {
	key := "VALIDATION_TARGET_KIND_" + strings.ToUpper(strings.ReplaceAll(kind, "-", "_"))
	if value, ok := commonv1.ValidationTargetKind_value[key]; ok {
		return commonv1.ValidationTargetKind(value)
	}
	return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_UNSPECIFIED
}

func runValidationProviderPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer, provider validationprovider.Provider, client DelegatedClient) RunReport {
	if len(env.CapabilitySubset) > 0 {
		provider.CapabilitySubset = append(append([]string(nil), provider.CapabilitySubset...), env.CapabilitySubset...)
	}
	var summary validationprovider.Summary
	var findings []*architecturev1.ArchitectureFinding
	var maturityAssessment *commonv1.MaturityAssessment
	var execMetrics *commonv1.ExecutionMetrics
	var presentation *commonv1.PhasePresentation
	var findingsSummary *runspb.PhaseFindingsSummary
	report := RunPhase(ctx, logWriter, provider.Phase,
		func() (*validationprovider.Result, error) {
			result := client(ctx, env, logWriter, provider)
			if result != nil {
				findings = result.Findings
				maturityAssessment = result.Assessment
				execMetrics = result.Metrics
				presentation = result.Presentation
				findingsSummary = result.FindingsSummary
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
	report.Assessment = maturityAssessment
	report.Metrics = execMetrics
	report.PhasePresentation = presentation
	report.FindingsSummary = findingsSummary
	if provider.FindingSource != architecturev1.FindingSource_FINDING_SOURCE_UNSPECIFIED {
		report.FindingSource = findingid.SourceToken(provider.FindingSource)
	}
	writePhasePointer(env, provider.Phase, report, map[string]any{"summary": summary}, logWriter)
	logPhaseStep(logWriter, "%s summary: %s", provider.Phase, summary.String())
	return report
}
