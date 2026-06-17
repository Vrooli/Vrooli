package phases

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"test-genie/internal/orchestrator/phases/validationprovider"
	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/shared"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

var contractsValidationProvider = validationprovider.Provider{
	Phase:            "contracts",
	ProviderScenario: "cli-health",
	FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_CLI,
	Emoji:            "📑",
	SkipEnvVar:       "TEST_GENIE_SKIP_CONTRACTS",
	Timeout:          120 * time.Second,
}

var protoValidationProvider = validationprovider.Provider{
	Phase:            "proto",
	ProviderScenario: "proto-health",
	FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_PROTO,
	Emoji:            "🧬",
	SkipEnvVar:       "TEST_GENIE_SKIP_PROTO",
	Optional:         true,
	Timeout:          120 * time.Second,
}

var uiHealthValidationProvider = validationprovider.Provider{
	Phase:            "ui-health",
	ProviderScenario: "ui-health",
	FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_UI,
	Emoji:            "🎨",
	SkipEnvVar:       "TEST_GENIE_SKIP_UI_HEALTH",
	Timeout:          120 * time.Second,
}

var securityValidationProvider = validationprovider.Provider{
	Phase:            "security",
	ProviderScenario: "security-health",
	FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_SECURITY,
	Emoji:            "🔐",
	SkipEnvVar:       "TEST_GENIE_SKIP_SECURITY",
	Optional:         true,
	Timeout:          5 * time.Minute,
}

var qualityValidationProvider = validationprovider.Provider{
	Phase:            "quality",
	ProviderScenario: "quality-health",
	FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_STANDARDS,
	Emoji:            "🧭",
	SkipEnvVar:       "TEST_GENIE_SKIP_QUALITY",
	DetailCommand:    "quality-health audit run {{scenario}}",
	Timeout:          5 * time.Minute,
}

var unitValidationProvider = validationprovider.Provider{
	Phase:            "unit",
	ProviderScenario: "unit-health",
	FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_STANDARDS,
	Emoji:            "🧪",
	SkipEnvVar:       "TEST_GENIE_SKIP_UNIT",
	Timeout:          20 * time.Minute,
	IncludeExecution: true,
}

var measuresValidationProvider = validationprovider.Provider{
	Phase:            "measures",
	ProviderScenario: "measures-health",
	FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_MEASURES,
	Emoji:            "📐",
	SkipEnvVar:       "TEST_GENIE_SKIP_MEASURES",
	Optional:         true,
	Timeout:          5 * time.Minute,
	IncludeExecution: true,
}

var dependenciesValidationProvider = validationprovider.Provider{
	Phase:            "dependencies",
	ProviderScenario: "scenario-dependency-analyzer",
	FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_DEPENDENCY,
	Emoji:            "📦",
	SkipEnvVar:       "TEST_GENIE_SKIP_DEPENDENCIES",
	DetailCommand:    "scenario-dependency-analyzer health {{scenario}}",
	Timeout:          5 * time.Minute,
}

var architectureValidationProvider = validationprovider.Provider{
	Phase:            "architecture",
	ProviderScenario: "architecture-cartographer",
	FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_ARCHITECTURE,
	Emoji:            "🏛️",
	SkipEnvVar:       "TEST_GENIE_SKIP_ARCHITECTURE",
	DetailCommand:    "architecture-cartographer audit run {{scenario}}",
	Optional:         true,
	Timeout:          120 * time.Second,
}

var docsValidationProvider = validationprovider.Provider{
	Phase:            "docs",
	ProviderScenario: "knowledge-observatory",
	FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_DOCS,
	Emoji:            "📄",
	SkipEnvVar:       "TEST_GENIE_SKIP_DOCS",
	DetailCommand:    "knowledge-observatory docs health {{scenario}}",
	Timeout:          120 * time.Second,
}

var tidinessValidationProvider = validationprovider.Provider{
	Phase:            "tidiness",
	ProviderScenario: "tidiness-manager",
	FindingSource:    architecturev1.FindingSource_FINDING_SOURCE_TIDINESS,
	Emoji:            "🧹",
	SkipEnvVar:       "TEST_GENIE_SKIP_TIDINESS",
	DetailCommand:    "tidiness-manager scan {{scenario}} --type tidiness",
	Optional:         true,
	Timeout:          120 * time.Second,
}

func runContractsPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	return runValidationProviderPhase(ctx, env, logWriter, contractsValidationProvider)
}

func runProtoPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	return runValidationProviderPhase(ctx, env, logWriter, protoValidationProvider)
}

func runUIHealthPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	return runValidationProviderPhase(ctx, env, logWriter, uiHealthValidationProvider)
}

func runSecurityPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	return runValidationProviderPhase(ctx, env, logWriter, securityValidationProvider)
}

func runQualityPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	return runValidationProviderPhase(ctx, env, logWriter, qualityValidationProvider)
}

func runUnitPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	return runValidationProviderPhase(ctx, env, logWriter, unitValidationProvider)
}

func runMeasuresPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	return runValidationProviderPhase(ctx, env, logWriter, measuresValidationProvider)
}

func runDependenciesPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	return runValidationProviderPhase(ctx, env, logWriter, dependenciesValidationProvider)
}

func runArchitecturePhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	return runValidationProviderPhase(ctx, env, logWriter, architectureValidationProvider)
}

func runDocsPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	return runValidationProviderPhase(ctx, env, logWriter, docsValidationProvider)
}

func runTidinessPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	return runValidationProviderPhase(ctx, env, logWriter, tidinessValidationProvider)
}

func runValidationProviderPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer, provider validationprovider.Provider) RunReport {
	if provider.SkipEnvVar != "" && os.Getenv(provider.SkipEnvVar) == "1" {
		summary := validationprovider.Summary{Scenario: env.ScenarioName, Status: "skipped", Skipped: true}
		report := RunReport{
			Observations: []Observation{
				NewSkipObservation(fmt.Sprintf("%s phase disabled via %s", provider.Phase, provider.SkipEnvVar)),
			},
		}
		writePhasePointer(env, provider.Phase, report, map[string]any{"summary": summary}, logWriter)
		return report
	}

	var summary validationprovider.Summary
	var findings []*architecturev1.ArchitectureFinding
	report := RunPhase(ctx, logWriter, provider.Phase,
		func() (*validationprovider.Result, error) {
			result := validationprovider.Run(ctx, provider, env.ScenarioName)
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
