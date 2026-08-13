package providerconformance

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/maturity-go/assessment"
	repocontract "github.com/vrooli/repo-contract-go"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	factsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts/facts_v1connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"

	"test-genie/internal/orchestrator/providerdescriptor"
	"test-genie/internal/selfhealth"
	"test-genie/internal/targetexecution"
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
	// DurableProbe verifies the descriptor-declared DurableValidationRunService
	// lifecycle. It is intentionally a generic transport seam, not a
	// Workflow-Health-specific check.
	DurableProbe DurableConformanceProbe
	// ProbeTarget is the optional fixture scenario the target provider is asked
	// to validate during the live probe. Empty resolves from its descriptor.
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
	s.validateTargetDeclaration(ctx, &report, descriptor)
	if !descriptor.DeterminismDeclared() {
		report.add(Finding{
			Code:        CodeDeterminismUndeclared,
			Severity:    SeverityWarning,
			Title:       "Provider determinism policy is implicit",
			Message:     "The provider descriptor omits determinism, so phase-result reuse remains observational and cannot be audited as an explicit provider decision.",
			Location:    providerdescriptor.RelPath + ":determinism",
			Remediation: "Declare determinism.default=file-determined with complete inputs, or declare observational with a reason.",
		})
	}

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
		target = selfhealth.ResolveConformanceTarget(s.RepoRoot, report.Scenario)
	}
	timeout := s.ProbeTimeout
	if timeout <= 0 {
		timeout = defaultProbeTimeout
	}
	report.Probed = true
	conformance, response := selfhealth.CheckProvider(ctx, s.Probe, nil, s.RepoRoot, target, descriptor.Phase, report.Scenario, timeout)
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
	if describe, err := selfhealth.DefaultDescribeProvider(ctx, report.Scenario, timeout); err == nil {
		declared := map[commonv1.ValidationTargetKind]struct{}{}
		for _, kind := range descriptor.Targets.EffectiveKinds() {
			declared[targetKindEnum(kind)] = struct{}{}
		}
		runningKinds := describe.GetCapabilities().GetTargetKinds()
		for _, kind := range runningKinds {
			if _, ok := declared[kind]; !ok {
				report.add(Finding{Code: CodeDeclaredKindsUnsupported, Severity: SeverityError, Title: "Running provider declares a different target coverage", Message: fmt.Sprintf("DescribeProvider reports target kind %s that the descriptor does not declare.", kind.String()), Location: "scenario-validation/v1.DescribeProvider.capabilities.target_kinds", Remediation: "Keep the running provider's target declaration and descriptor in sync."})
			}
		}
		// An older provider may omit the additive target_kinds field entirely;
		// preserve the descriptor-default scenario contract in that case. Once a
		// provider publishes target coverage, enforce the comparison in both
		// directions so a descriptor cannot promise a kind the binary rejects.
		if len(runningKinds) > 0 {
			for kind := range declared {
				if kind == commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_UNSPECIFIED {
					continue
				}
				if !slices.Contains(runningKinds, kind) {
					report.add(Finding{Code: CodeDeclaredKindsUnsupported, Severity: SeverityError, Title: "Descriptor target coverage is not supported by the running provider", Message: fmt.Sprintf("descriptor declares target kind %s, but DescribeProvider does not advertise it.", kind.String()), Location: providerdescriptor.RelPath + ":targets.kinds", Remediation: "Keep the running provider's target declaration and descriptor in sync."})
				}
			}
		}
		s.validateRunningSpecVersion(report, descriptor, describe.GetSpecVersion())
	}
	for _, finding := range conformanceFindings(response, descriptor) {
		report.add(finding)
	}
	if descriptor.Validation.DeliveryMode == "durable-run" {
		if s.DurableProbe == nil {
			report.add(Finding{
				Code: CodeDurableContractInvalid, Severity: SeverityError,
				Title:       "Durable provider lifecycle probe is unavailable",
				Message:     "The provider declares durable-run delivery but Test Genie was not wired with the generic durable lifecycle probe.",
				Location:    "scenario-validation/v1.DurableValidationRunService",
				Remediation: "Wire the generic durable conformance probe and verify Start, replay, Get, Abort, and Wait before enabling durable delivery.",
			})
			return
		}
		if err := s.DurableProbe(ctx, report.Scenario, target, timeout); err != nil {
			report.add(Finding{
				Code: CodeDurableContractInvalid, Severity: SeverityError,
				Title:   "Provider durable lifecycle contract is invalid",
				Message: err.Error(), Location: "scenario-validation/v1.DurableValidationRunService",
				Remediation: "Implement prompt idempotent Start, Get, explicit Abort, and server-owned Wait with a valid terminal lifecycle response.",
			})
		}
	}
}

func (s *Service) validateTargetDeclaration(ctx context.Context, report *Report, descriptor providerdescriptor.Descriptor) {
	declared := descriptor.Targets.EffectiveKinds()
	for _, kind := range declared {
		if _, ok := validTargetKinds[kind]; !ok {
			report.add(Finding{Code: CodeDeclaredKindsUnsupported, Severity: SeverityError, Title: "Provider declares an unsupported target kind", Message: fmt.Sprintf("targets.kinds contains %q, which this Test Genie target model does not support.", kind), Location: providerdescriptor.RelPath + ":targets.kinds", Remediation: "Declare only one of the eight repository target kinds."})
		}
	}
	if descriptor.MaturitySpec != nil {
		for _, capability := range descriptor.MaturitySpec.Capabilities {
			if len(capability.AppliesTo) == 0 {
				continue
			}
			for _, kind := range capability.AppliesTo {
				covered := false
				for _, mapping := range descriptor.MaturitySpec.Findings {
					if mapping.CapabilityID == capability.ID {
						covered = true
						break
					}
				}
				if !covered {
					report.add(Finding{Code: CodeCapabilityCoverageGap, Severity: SeverityError, Title: "Capability has no finding coverage for a target kind", Message: fmt.Sprintf("capability %q declares appliesTo=%q but no finding maps to that capability.", capability.ID, kind), Location: providerdescriptor.RelPath + ":maturity.capabilities", Remediation: "Map at least one emitted finding to every declared capability target kind, or remove the appliesTo entry."})
				}
			}
		}
	}
	if s.RepoRoot != "" {
		if contract, err := repocontract.LoadDefault(s.RepoRoot); err == nil {
			targets, err := contract.EnumerateTargets(s.RepoRoot)
			counts := map[string]int{}
			if err == nil {
				for _, target := range targets {
					counts[string(target.Kind)]++
				}
			}
			for _, kind := range declared {
				if counts[kind] == 0 {
					report.add(Finding{Code: CodeTargetGlobUnresolvable, Severity: SeverityError, Title: "Provider target kind resolves to no repository targets", Message: fmt.Sprintf("targets.kinds includes %q, but repo-contract enumeration found no targets of that kind.", kind), Location: ".vrooli/repo-contract.json:targets", Remediation: "Fix the target roots/marker or stop declaring an empty target kind."})
				}
			}
			s.validateExecutionRunners(ctx, report, targets, declared)
		}
	}
}

// validateRunningSpecVersion compares the maturity spec version the running
// provider reports against the version in its on-disk descriptor.
//
// This closes the one drift the descriptor-only checks cannot see. A provider
// binary that predates its descriptor keeps answering, keeps passing readiness,
// and emits finding codes the descriptor never declared. Those codes then
// resolve through the descriptor's `fallback` mapping, so a stale binary does
// not fail loudly — it silently scores a *different* capability than the one
// the finding belongs to. Observed live on 2026-08-03: a storage-manager build
// emitting STORAGE_ACCOUNTABILITY_NOT_GOVERNED, a code absent from the entire
// repository, while the descriptor declared STORAGE_ACCOUNTABILITY_UNGOVERNED.
//
// An empty version on either side means the provider or descriptor predates
// versioned specs, which is not drift.
func (s *Service) validateRunningSpecVersion(report *Report, descriptor providerdescriptor.Descriptor, runningVersion string) {
	if descriptor.MaturitySpec == nil {
		return
	}
	declaredVersion := strings.TrimSpace(descriptor.MaturitySpec.Version)
	runningVersion = strings.TrimSpace(runningVersion)
	if declaredVersion == "" || runningVersion == "" || declaredVersion == runningVersion {
		return
	}
	report.add(Finding{
		Code:     CodeRunningSpecVersionStale,
		Severity: SeverityError,
		Title:    "Running provider serves a different maturity spec than its descriptor",
		Message: fmt.Sprintf(
			"DescribeProvider reports maturity spec version %q but %s declares %q. The running binary predates its descriptor, so it can emit finding codes the descriptor does not declare — those resolve through the fallback mapping and silently score the wrong capability.",
			runningVersion, providerdescriptor.RelPath, declaredVersion),
		Location:    "scenario-validation/v1.DescribeProvider.spec_version",
		Remediation: "Rebuild and restart the provider so the running binary matches its descriptor.",
	})
}

// validateExecutionRunners is deliberately limited to executable repository
// kinds. Documentation, teams, resources, and safeguards can be validated by
// provider-specific static checks; package/control-plane targets must have a
// registered language runner or conformance reports a typed gap.
func (s *Service) validateExecutionRunners(ctx context.Context, report *Report, targets []repocontract.Target, declared []string) {
	for _, kind := range declared {
		if kind != "package" && kind != "control-plane" {
			continue
		}
		for _, target := range targets {
			if string(target.Kind) != kind {
				continue
			}
			language, proven := s.codeFactsLanguage(ctx, filepath.Join(s.RepoRoot, filepath.FromSlash(target.Root)))
			if !proven {
				// A provider outage is not evidence that a repository target has no
				// runner. Unit/quality providers report the degraded code-facts
				// condition during the actual target run.
				continue
			}
			if _, err := targetexecution.ForLanguage(language); err != nil {
				report.add(Finding{
					Code:        CodeExecutionRunnerMissing,
					Severity:    SeverityError,
					Title:       "Executable target has no registered runner",
					Message:     fmt.Sprintf("%s:%s was classified as %s, but Test Genie has no runner for that language: %v", kind, target.ID, language, err),
					Location:    target.Root,
					Remediation: "Add a deterministic Go or TypeScript runner, or classify the target as unsupported instead of silently skipping it.",
				})
			}
		}
	}
}

func (s *Service) codeFactsLanguage(ctx context.Context, root string) (string, bool) {
	baseURL, err := discovery.NewResolver(discovery.ResolverConfig{}).ResolveScenarioURLDefault(ctx, "code-facts")
	if err != nil || strings.TrimSpace(baseURL) == "" {
		return "", false
	}
	client := factsconnect.NewCodeFactsServiceClient(http.DefaultClient, baseURL)
	resp, err := client.DescribeCodeFacts(ctx, connect.NewRequest(&factsv1.DescribeCodeFactsRequest{
		Target:  &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_PATH, Path: root},
		Include: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_PARSE_UNITS}, UseCache: true,
	}))
	if err != nil || resp == nil || resp.Msg == nil {
		return "", false
	}
	for _, unit := range resp.Msg.GetParseUnits() {
		if unit.GetStatus() == factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN && strings.TrimSpace(unit.GetLanguage()) != "" {
			return strings.ToLower(strings.TrimSpace(unit.GetLanguage())), true
		}
	}
	return "", false
}

var validTargetKinds = map[string]struct{}{
	"scenario": {}, "resource": {}, "tool": {}, "safeguard": {}, "team": {}, "package": {}, "control-plane": {}, "docs": {}, "project": {},
}

func conformanceFindings(response *scenariovalidationv1.ValidateScenarioResponse, descriptor providerdescriptor.Descriptor) []Finding {
	if response == nil || response.GetAssessment() == nil {
		return nil
	}
	declared := map[string]struct{}{}
	for _, kind := range descriptor.Targets.EffectiveKinds() {
		declared[kind] = struct{}{}
	}
	var findings []Finding
	for _, item := range response.GetAssessment().GetFindings() {
		subject := item.GetSubject()
		if subject == nil || subject.GetKind() == commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_UNSPECIFIED {
			continue
		}
		kind := validationTargetKindName(subject.GetKind())
		if _, ok := declared[kind]; ok {
			continue
		}
		findings = append(findings, Finding{Code: CodeSubjectOutsideDeclaredKinds, Severity: SeverityError, Title: "Finding subject is outside provider coverage", Message: fmt.Sprintf("finding %q attributes its result to target kind %q, but the provider declares %v.", item.GetCode(), kind, descriptor.Targets.EffectiveKinds()), Location: "common.v1.AssessmentFinding.subject", Remediation: "Declare the subject kind in targets.kinds or emit the finding only for a declared target."})
	}
	return findings
}

func validationTargetKindName(kind commonv1.ValidationTargetKind) string {
	switch kind {
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO:
		return "scenario"
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_RESOURCE:
		return "resource"
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_TOOL:
		return "tool"
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SAFEGUARD:
		return "safeguard"
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_TEAM:
		return "team"
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PACKAGE:
		return "package"
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_CONTROL_PLANE:
		return "control-plane"
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_DOCS:
		return "docs"
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PROJECT:
		return "project"
	default:
		return "unspecified"
	}
}

func targetKindEnum(kind string) commonv1.ValidationTargetKind {
	switch kind {
	case "scenario":
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO
	case "resource":
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_RESOURCE
	case "tool":
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_TOOL
	case "safeguard":
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SAFEGUARD
	case "team":
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_TEAM
	case "package":
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PACKAGE
	case "control-plane":
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_CONTROL_PLANE
	case "docs":
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_DOCS
	case "project":
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PROJECT
	default:
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_UNSPECIFIED
	}
}

// DurableConformanceProbe validates the provider-neutral durable lifecycle.
type DurableConformanceProbe func(context.Context, string, string, time.Duration) error

// DefaultDurableConformanceProbe exercises one provider-owned lifecycle using
// only the shared protocol. Abort makes the probe bounded and leaves no
// potentially expensive validation work running after conformance returns.
func DefaultDurableConformanceProbe(ctx context.Context, provider, target string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = defaultProbeTimeout
	}
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, provider)
	if err != nil {
		return fmt.Errorf("resolve %s URL: %w", provider, err)
	}
	client := scenariovalidationconnect.NewDurableValidationRunServiceClient(&http.Client{Timeout: timeout}, strings.TrimRight(strings.TrimSpace(baseURL), "/"))
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	key := "provider-conformance:" + strings.TrimSpace(target)
	start, err := client.StartValidationRun(probeCtx, connect.NewRequest(&scenariovalidationv1.StartValidationRunRequest{Scenario: target, IdempotencyKey: key}))
	if err != nil || start == nil || start.Msg == nil || start.Msg.GetRun() == nil || start.Msg.GetRun().GetRunId() == "" {
		return fmt.Errorf("StartValidationRun did not return a run: %w", err)
	}
	runID := start.Msg.GetRun().GetRunId()
	replay, err := client.StartValidationRun(probeCtx, connect.NewRequest(&scenariovalidationv1.StartValidationRunRequest{Scenario: target, IdempotencyKey: key}))
	if err != nil || replay == nil || replay.Msg == nil || replay.Msg.GetRun().GetRunId() != runID {
		return fmt.Errorf("StartValidationRun replay did not return the original run %q: %w", runID, err)
	}
	got, err := client.GetValidationRun(probeCtx, connect.NewRequest(&scenariovalidationv1.GetValidationRunRequest{RunId: runID}))
	if err != nil || got == nil || got.Msg == nil || got.Msg.GetRun().GetRunId() != runID {
		return fmt.Errorf("GetValidationRun did not reattach to %q: %w", runID, err)
	}
	if _, err := client.AbortValidationRun(probeCtx, connect.NewRequest(&scenariovalidationv1.AbortValidationRunRequest{RunId: runID, Reason: "provider conformance lifecycle probe"})); err != nil {
		return fmt.Errorf("AbortValidationRun %q: %w", runID, err)
	}
	waited, err := client.WaitValidationRun(probeCtx, connect.NewRequest(&scenariovalidationv1.WaitValidationRunRequest{RunId: runID}))
	if err != nil || waited == nil || waited.Msg == nil || waited.Msg.GetRun() == nil || !terminalState(waited.Msg.GetRun().GetState()) {
		return fmt.Errorf("WaitValidationRun did not return a terminal run %q: %w", runID, err)
	}
	return nil
}

func terminalState(state scenariovalidationv1.ValidationRunState) bool {
	return state == scenariovalidationv1.ValidationRunState_VALIDATION_RUN_STATE_SUCCEEDED ||
		state == scenariovalidationv1.ValidationRunState_VALIDATION_RUN_STATE_FAILED ||
		state == scenariovalidationv1.ValidationRunState_VALIDATION_RUN_STATE_CANCELED ||
		state == scenariovalidationv1.ValidationRunState_VALIDATION_RUN_STATE_RECOVERY_FAILED
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
