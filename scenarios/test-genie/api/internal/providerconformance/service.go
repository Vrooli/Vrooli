package providerconformance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/maturity-go/assessment"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"

	"test-genie/internal/orchestrator/providerdescriptor"
	"test-genie/internal/selfhealth"
)

// selfScenario is the orchestrator's own scenario name. Validating it must
// never trigger a live probe: the probe would call this process's own
// ValidateScenario endpoint, which runs this validator, which would probe
// again — an unbounded recursion. Internal descriptor checks are sufficient
// for the self target (D4 in the provider phase completion plan).
const selfScenario = "test-genie"

// defaultProbeTimeout bounds the live provider contract probe.
const defaultProbeTimeout = 30 * time.Second

type Service struct {
	RepoRoot string
	// Probe is the live provider contract seam. Nil disables live checks so
	// the validator stays pure (descriptor-only findings).
	Probe selfhealth.ConformanceProbe
	// ProbeTarget is the fixture scenario the target provider is asked to
	// validate during the live probe (selfhealth.DefaultScanTarget when empty).
	ProbeTarget string
	// ProbeTimeout bounds the live probe (defaultProbeTimeout when zero).
	ProbeTimeout time.Duration
}

func New(repoRoot string) *Service {
	return &Service{RepoRoot: strings.TrimSpace(repoRoot)}
}

// ValidateScenario judges one target scenario's Test Genie provider descriptor.
// Descriptor, maturity, identity, stale-file, and policy rules come from the
// providerdescriptor loader (the registry's own rule source); live contract,
// identity, and metrics rules come from selfhealth.ScanProvider (the fleet
// scan's rule source). Neither rule set is re-implemented here.
func (s *Service) ValidateScenario(ctx context.Context, scenario, path string) (Report, error) {
	scenario, scenarioPath, err := s.resolveTarget(scenario, path)
	if err != nil {
		return Report{}, err
	}
	report := Report{Scenario: scenario, Path: scenarioPath}
	descriptorPath := filepath.Join(scenarioPath, filepath.FromSlash(providerdescriptor.RelPath))
	if _, err := os.Stat(descriptorPath); err != nil {
		if os.IsNotExist(err) {
			report.add(Finding{
				Code:        CodeDescriptorMissing,
				Severity:    SeverityError,
				Title:       "Test Genie provider descriptor missing",
				Message:     "The target was validated as a Test Genie phase provider but has no .vrooli/test-genie.json descriptor.",
				Location:    providerdescriptor.RelPath,
				Remediation: "Add a .vrooli/test-genie.json phase descriptor, or stop requesting the provider-conformance phase for this scenario.",
			})
			report.finish()
			return report, nil
		}
		return Report{}, fmt.Errorf("stat %s: %w", descriptorPath, err)
	}

	load := providerdescriptor.Load(providerdescriptor.LoadOptions{Paths: []string{descriptorPath}})
	for _, diagnostic := range load.Diagnostics {
		report.add(findingFromDiagnostic(diagnostic))
	}
	if len(load.Descriptors) != 1 {
		report.ProbeSkipReason = "descriptor did not load; provider contract cannot be probed"
		report.finish()
		return report, nil
	}
	descriptor := load.Descriptors[0]
	report.Phase = descriptor.Phase

	s.validateDocs(&report, descriptor)
	validateMaturityContract(&report, descriptor)
	validateAutofixDeclaration(&report, descriptor)
	s.probeContract(ctx, &report, descriptor)

	report.finish()
	return report, nil
}

func (s *Service) resolveTarget(scenario, path string) (string, string, error) {
	scenario = normalizeScenario(scenario)
	path = strings.TrimSpace(path)
	if path != "" {
		if scenario == "" {
			scenario = filepath.Base(filepath.Clean(path))
		}
		return scenario, path, nil
	}
	if scenario == "" {
		return "", "", fmt.Errorf("scenario or path is required")
	}
	if s.RepoRoot == "" {
		return "", "", fmt.Errorf("repo root is required when path is omitted")
	}
	return scenario, filepath.Join(s.RepoRoot, "scenarios", scenario), nil
}

func (s *Service) validateDocs(report *Report, descriptor providerdescriptor.Descriptor) {
	docsPath := strings.TrimSpace(descriptor.Docs.Path)
	if docsPath == "" {
		report.add(Finding{
			Code:        CodeDocsMissing,
			Severity:    SeverityWarning,
			Title:       "Provider phase has no operator documentation",
			Message:     "docs.path is empty, so operators cannot be routed to phase documentation.",
			Location:    providerdescriptor.RelPath + ":docs.path",
			Remediation: "Point docs.path at the phase README under scenarios/test-genie/docs/phases/.",
		})
		return
	}
	if s.RepoRoot == "" {
		return
	}
	resolved := filepath.Join(s.RepoRoot, filepath.FromSlash(docsPath))
	if _, err := os.Stat(resolved); err != nil {
		report.add(Finding{
			Code:        CodeDocsMissing,
			Severity:    SeverityWarning,
			Title:       "Provider phase documentation file is absent",
			Message:     fmt.Sprintf("docs.path %q does not resolve to a file in the repository.", docsPath),
			Location:    providerdescriptor.RelPath + ":docs.path",
			Remediation: "Create the referenced phase documentation or fix docs.path.",
		})
		return
	}
	// The doc exists — enforce the required remediation-doc skeleton.
	validateDocsSkeleton(report, resolved, docsPath)
}

func validateAutofixDeclaration(report *Report, descriptor providerdescriptor.Descriptor) {
	if descriptor.MaturitySpec == nil {
		return
	}
	coverage := assessment.ComputeAutofixCoverage(*descriptor.MaturitySpec)
	if coverage.DeclarationComplete {
		return
	}
	report.add(Finding{
		Code:     CodeAutofixDeclarationIncomplete,
		Severity: SeverityWarning,
		Title:    "Provider autofix declaration is incomplete",
		Message: fmt.Sprintf(
			"%d of %d maturity findings declare a fix class; every finding needs an explicit fix_class (manual findings need a reason).",
			coverage.Declared, coverage.Total),
		Location:    providerdescriptor.RelPath + ":maturity.findings",
		Remediation: "Declare fix_class (and reason for manual/detection-only rules) on every maturity finding.",
	})
}

// probeContract runs the live provider contract probe unless the seam is
// absent or the target is test-genie itself (recursion guard).
func (s *Service) probeContract(ctx context.Context, report *Report, descriptor providerdescriptor.Descriptor) {
	if s.Probe == nil {
		report.ProbeSkipReason = "live contract probe disabled"
		return
	}
	if report.Scenario == selfScenario {
		report.ProbeSkipReason = "self target: descriptor checks only, no recursive validation call"
		return
	}
	target := strings.TrimSpace(s.ProbeTarget)
	if target == "" {
		target = selfhealth.ProviderDefaultTarget(report.Scenario)
	}
	timeout := s.ProbeTimeout
	if timeout <= 0 {
		timeout = defaultProbeTimeout
	}
	report.Probed = true
	conformance := selfhealth.ScanProvider(ctx, s.Probe, s.RepoRoot, target, descriptor.Phase, report.Scenario, timeout)
	if !conformance.Reachable {
		report.add(Finding{
			Code:        CodeProviderUnreachable,
			Severity:    SeverityWarning,
			Title:       "Provider validation service is unreachable",
			Message:     strings.Join(conformance.Violations, "; "),
			Location:    "scenario-validation/v1.ScenarioValidationService",
			Remediation: "Start the provider scenario (vrooli scenario start " + report.Scenario + ") and re-run; unreachable providers are an environmental signal, not a descriptor defect.",
		})
		return
	}
	if !conformance.ContractValid {
		report.add(Finding{
			Code:        CodeContractInvalid,
			Severity:    SeverityError,
			Title:       "Provider validation response violates the shared contract",
			Message:     strings.Join(conformance.Violations, "; "),
			Location:    "scenario-validation/v1.ScenarioValidationService.ValidateScenario",
			Remediation: "Return a contract-valid MaturityAssessment (assessment.BuildValidationResponse) from the provider's ValidateScenario.",
		})
	}
	if !conformance.IdentityOK {
		report.add(Finding{
			Code:        CodeContractIdentityMismatch,
			Severity:    SeverityError,
			Title:       "Provider assessment identity mismatch",
			Message:     fmt.Sprintf("live assessment does not identify as provider %q phase %q.", report.Scenario, descriptor.Phase),
			Location:    "scenario-validation/v1.ScenarioValidationService.ValidateScenario",
			Remediation: "Stamp the descriptor's provider and phase identity on every assessment the provider returns.",
		})
	}
	if !conformance.MetricsAdopted {
		report.add(Finding{
			Code:        CodeMetricsMissing,
			Severity:    SeverityError,
			Title:       "Provider omits execution metrics",
			Message:     "The provider is reachable but its validation response carries no common.v1.ExecutionMetrics.",
			Location:    "scenario-validation/v1.ScenarioValidationService.ValidateScenario",
			Remediation: "Attach ExecutionMetrics to every validation response, including degraded and error paths.",
		})
	}
}

func findingFromDiagnostic(diagnostic providerdescriptor.Diagnostic) Finding {
	location := providerdescriptor.RelPath
	base := Finding{
		Severity: SeverityError,
		Message:  diagnostic.Message,
		Location: location,
	}
	switch {
	case diagnostic.Code == "scenario_mismatch":
		base.Code = CodeIdentityMismatch
		base.Title = "Descriptor scenario does not match its directory"
		base.Remediation = "Set the descriptor's scenario field to the owning scenario directory name."
	case diagnostic.Code == "leftover_maturity_json":
		base.Code = CodeStaleMaturityFile
		base.Title = "Retired .vrooli/maturity.json still exists"
		base.Remediation = "Delete .vrooli/maturity.json; the maturity block lives in .vrooli/test-genie.json after the descriptor cutover."
	case diagnostic.Code == "invalid_maturity":
		base.Code = CodeMaturityInvalid
		base.Title = "Embedded maturity block is invalid"
		base.Remediation = "Fix the descriptor's maturity block so it satisfies the shared maturity contract."
	case strings.Contains(diagnostic.Code, "policy"):
		base.Code = CodePolicyUnsafe
		base.Title = "Descriptor policy is invalid or unsafe"
		base.Remediation = "Choose a valid policy combination (see the phase descriptor schema)."
	default:
		base.Code = CodeDescriptorInvalid
		base.Title = "Provider descriptor is invalid"
		base.Remediation = "Fix .vrooli/test-genie.json so it satisfies the Test Genie phase descriptor schema (" + diagnostic.Code + ")."
	}
	return base
}

// BuildMaturityAssessment projects a report into the shared maturity contract
// using Test Genie's own provider-conformance spec.
func BuildMaturityAssessment(scenario string, findings []Finding, spec assessment.Spec) (*commonv1.MaturityAssessment, error) {
	assessed := make([]assessment.Finding, 0, len(findings))
	for _, f := range findings {
		assessed = append(assessed, assessment.Finding{
			Code:        f.Code,
			Severity:    severityToAssessment(f.Severity),
			Title:       f.Title,
			Message:     f.Message,
			Location:    filepath.ToSlash(f.Location),
			Remediation: f.Remediation,
			Source:      architecturev1.FindingSource_FINDING_SOURCE_UNSPECIFIED,
			Phase:       spec.Phase,
		})
	}
	return assessment.BuildProtoAssessment(assessment.BuildInput{
		Scenario: scenario,
		Spec:     spec,
		Findings: assessed,
	})
}
