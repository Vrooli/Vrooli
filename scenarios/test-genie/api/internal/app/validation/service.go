// Package validation hosts the Connect-RPC ScenarioValidationService handler
// that exposes Test Genie's provider-conformance judgment: it validates target
// scenarios that declare themselves as Test Genie phase providers through
// .vrooli/test-genie.json. It is a thin transport wrapper over
// internal/providerconformance.
package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/metrics"
	"github.com/vrooli/maturity-go/assessment"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"google.golang.org/protobuf/types/known/structpb"

	"test-genie/internal/providerconformance"
	"test-genie/internal/selfhealth"
)

// Service implements scenariovalidationconnect.ScenarioValidationServiceHandler
// for the provider-conformance phase.
type Service struct {
	logger         *log.Logger
	validator      *providerconformance.Service
	spec           *assessment.Spec
	faultSpec      *assessment.Spec
	repoRoot       string
	faultRun       func(context.Context, string) (string, error)
	matrixFaultRun func(context.Context, string) (string, error)
	describer      assessment.Describer
}

// NewService builds the provider-conformance validation service. repoRoot is
// the repository root; the maturity spec is Test Genie's own descriptor-embedded
// provider-conformance spec. The live contract probe uses the same Connect
// probe as the fleet conformance scan.
func NewService(logger *log.Logger, repoRoot string, spec *assessment.Spec) *Service {
	if logger == nil {
		logger = log.Default()
	}
	validator := providerconformance.New(repoRoot)
	validator.Probe = selfhealth.DefaultConformanceProbe
	validator.DurableProbe = providerconformance.DefaultDurableConformanceProbe
	// Test Genie answers the same readiness contract it asks of others. A load
	// failure yields the zero Describer, which reports Unimplemented.
	describer, _ := assessment.LoadDescriber(filepath.Join(repoRoot, "scenarios", "test-genie"))
	faultSpec, faultErr := assessment.LoadSpecForPhase(filepath.Join(repoRoot, "scenarios", "test-genie"), "fault-injection")
	if faultErr != nil {
		logger.Printf("validation: fault-injection maturity spec unavailable: %v", faultErr)
	}
	return &Service{logger: logger, validator: validator, spec: spec, faultSpec: faultSpec, repoRoot: repoRoot, faultRun: runDBMFaultInjection, matrixFaultRun: runGeneratedFaultMatrix, describer: describer}
}

// NewServiceForTest builds a Service with an injected validator seam.
func NewServiceForTest(logger *log.Logger, validator *providerconformance.Service, spec *assessment.Spec) *Service {
	if logger == nil {
		logger = log.Default()
	}
	return &Service{logger: logger, validator: validator, spec: spec}
}

func (s *Service) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	if req != nil && req.Msg != nil && req.Msg.GetPhase() == "fault-injection" {
		return s.validateFaultInjection(ctx, req.Msg)
	}
	collector := metrics.Start()
	if s.spec == nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("provider-conformance maturity spec is unavailable"))
	}
	report, err := s.validator.ValidateScenario(ctx, req.Msg.GetScenario(), req.Msg.GetPath())
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	maturity, err := providerconformance.BuildMaturityAssessment(report.Scenario, report.Findings, *s.spec)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build maturity assessment: %w", err))
	}
	native, err := nativeDetail(report)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build native detail: %w", err))
	}
	resp, err := assessment.BuildValidationResponse(report.Scenario, maturity, native, collector.Stop())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build shared validation response: %w", err))
	}
	return connect.NewResponse(resp), nil
}

func (s *Service) validateFaultInjection(ctx context.Context, req *scenariovalidationv1.ValidateScenarioRequest) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	scenario := req.GetScenario()
	if !req.GetIncludeExecution() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("fault-injection requires execution mode"))
	}
	if s.faultSpec == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fault-injection maturity spec is unavailable"))
	}
	root, err := s.resolveFaultTarget(scenario, req.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	isDBM := pathExists(filepath.Join(root, "api", "internal", "destinationreadiness", "recovery.go"))
	isMatrix := pathExists(filepath.Join(root, ".vrooli")) && pathExists(filepath.Join(root, "kernel", "tests", "fault_matrix.py"))
	if !isDBM && !isMatrix {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("fault-injection does not apply to %q", scenario))
	}
	runner := s.faultRun
	if isMatrix {
		runner = s.matrixFaultRun
	}
	if runner == nil {
		if isMatrix {
			runner = runGeneratedFaultMatrix
		} else {
			runner = runDBMFaultInjection
		}
	}
	collector := metrics.Start()
	output, runErr := runner(ctx, root)
	var evidence any
	if isDBM {
		receiptEvidence, receiptErr := extractFixtureReceipts(output)
		evidence = receiptEvidence
		if runErr == nil && receiptErr != nil {
			runErr = receiptErr
		}
	} else {
		matrixEvidence, matrixErr := extractProgramRuntimeMatrix(output)
		evidence = matrixEvidence
		if runErr == nil && matrixErr != nil {
			runErr = matrixErr
		}
	}
	findings := []assessment.Finding(nil)
	status := scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED
	if runErr != nil {
		status = scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED
		code := "dbm_fault_injection.execution_failed"
		location := "scenarios/data-backup-manager/scripts/db6-fault-injection.sh"
		title := "Simulated recovery fault injection failed"
		if isMatrix {
			code = "child_envelope_fault_matrix.execution_failed"
			location, _ = filepath.Rel(s.repoRoot, filepath.Join(root, "kernel", "tests", "fault_matrix.py"))
			title = "Child-envelope fault matrix failed"
		}
		findings = append(findings, assessment.Finding{Code: code, Severity: "SEVERITY_ERROR", Title: title, Message: runErr.Error(), Location: location, Remediation: "Inspect the generated Test Genie phase evidence and repair the failing owner contract."})
	}
	maturity, err := assessment.BuildProtoAssessment(assessment.BuildInput{Scenario: scenario, Spec: *s.faultSpec, Findings: findings})
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build fault-injection maturity assessment: %w", err))
	}
	evidenceKind, evidenceField := "fixture", "capability_receipts"
	if isMatrix {
		evidenceKind, evidenceField = "generated_fixture", "fault_matrix"
	}
	detailValues := map[string]any{"scenario": scenario, "phase": "fault-injection", "outcome": status.String(), "evidence_kind": evidenceKind, "native_qualification": false, evidenceField: evidence, "output": boundedOutput(output, 12000)}
	detail, err := structpb.NewStruct(detailValues)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp, err := assessment.BuildValidationResponse(scenario, maturity, detail, collector.Stop(), assessment.WithValidationStatus(status))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build fault-injection response: %w", err))
	}
	return connect.NewResponse(resp), nil
}

func (s *Service) resolveDBMTarget(requestPath string) (string, error) {
	root, err := filepath.Abs(filepath.Join(s.repoRoot, "scenarios", "data-backup-manager"))
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(requestPath) == "" {
		return root, nil
	}
	resolved, err := filepath.Abs(requestPath)
	if err != nil {
		return "", err
	}
	if resolved != root {
		return "", fmt.Errorf("fault-injection target path must resolve to the repository DBM scenario")
	}
	return root, nil
}

func (s *Service) resolveFaultTarget(scenario, requestPath string) (string, error) {
	if scenario == "data-backup-manager" {
		return s.resolveDBMTarget(requestPath)
	}
	if scenario == "" || filepath.Base(scenario) != scenario || scenario == "." {
		return "", fmt.Errorf("fault-injection target scenario is invalid")
	}
	root, err := filepath.Abs(filepath.Join(s.repoRoot, "scenarios", scenario))
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(requestPath) == "" {
		return root, nil
	}
	resolved, err := filepath.Abs(requestPath)
	if err != nil {
		return "", err
	}
	if resolved != root {
		return "", fmt.Errorf("fault-injection target path must resolve to the repository scenario")
	}
	return root, nil
}

func runDBMFaultInjection(ctx context.Context, scenarioRoot string) (string, error) {
	cmd := exec.CommandContext(ctx, "bash", filepath.Join(scenarioRoot, "scripts", "db6-fault-injection.sh"))
	cmd.Dir = scenarioRoot
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func runGeneratedFaultMatrix(ctx context.Context, scenarioRoot string) (string, error) {
	cmd := exec.CommandContext(ctx, "python3", filepath.Join(scenarioRoot, "kernel", "tests", "fault_matrix.py"))
	cmd.Dir = scenarioRoot
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func extractProgramRuntimeMatrix(output string) (map[string]any, error) {
	var payload map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &payload); err != nil {
		return nil, fmt.Errorf("generated fault matrix did not emit valid JSON")
	}
	if payload["evidence_kind"] != "generated_fixture" || payload["native_qualification"] != false {
		return nil, fmt.Errorf("generated fault matrix must remain explicitly fixture-only")
	}
	programs, programsOK := payload["program_count"].(float64)
	rows, rowsOK := payload["row_count"].(float64)
	passed, passedOK := payload["passed_rows"].(float64)
	failed, failedOK := payload["failed_rows"].(float64)
	faults, faultsOK := payload["fault_classes"].([]any)
	matrix, matrixOK := payload["matrix"].([]any)
	if !programsOK || !rowsOK || !passedOK || !failedOK || !faultsOK || !matrixOK || programs <= 0 || programs != float64(int(programs)) || len(faults) != 5 || rows != programs*float64(len(faults)) || rows != passed || failed != 0 || len(matrix) != int(rows) {
		return nil, fmt.Errorf("generated fault matrix is incomplete or has failed rows")
	}
	seenPrograms, seenFaults, seenPairs := map[string]struct{}{}, map[string]struct{}{}, map[string]struct{}{}
	expectedFaults := map[string]struct{}{}
	for _, value := range faults {
		fault, ok := value.(string)
		if !ok || fault == "" {
			return nil, fmt.Errorf("generated fault matrix has an invalid fault class")
		}
		expectedFaults[fault] = struct{}{}
	}
	if len(expectedFaults) != len(faults) {
		return nil, fmt.Errorf("generated fault matrix fault classes are duplicated")
	}
	expectedStatuses := map[string]string{"unavailable": "unavailable", "degraded": "partial", "healthy-empty": "failed", "malformed": "failed", "partial": "partial"}
	expectedBehaviors := map[string]string{
		"unavailable":   "propagate_child_status_without_retry",
		"degraded":      "propagate_child_status_without_retry",
		"partial":       "propagate_child_status_without_retry",
		"healthy-empty": "refuse_empty_child_result",
		"malformed":     "refuse_malformed_child_envelope",
	}
	for _, value := range matrix {
		row, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("generated fault matrix contains an invalid row")
		}
		digest, digestOK := row["contract_digest"].(string)
		if row["passed"] != true || !digestOK || !strings.HasPrefix(digest, "sha256:") {
			return nil, fmt.Errorf("generated fault matrix contains an invalid row")
		}
		capabilities, capabilitiesOK := row["capability_requirements"].([]any)
		fallback, fallbackOK := row["observable_fallback"].(map[string]any)
		if !capabilitiesOK || len(capabilities) == 0 || !fallbackOK || fallback["status"] == "" || fallback["behavior"] == "" {
			return nil, fmt.Errorf("generated fault matrix row omits capability requirements or observable fallback")
		}
		for _, capability := range capabilities {
			item, ok := capability.(map[string]any)
			if !ok || !((stringField(item, "binding") != "" && stringField(item, "effect") != "") || stringField(item, "library_program") != "") {
				return nil, fmt.Errorf("generated fault matrix row has an invalid capability requirement")
			}
		}
		program, programOK := row["program"].(string)
		fault, faultOK := row["fault"].(string)
		if !programOK || !faultOK || program == "" || fault == "" {
			return nil, fmt.Errorf("generated fault matrix row has no program or fault identity")
		}
		if _, expected := expectedFaults[fault]; !expected {
			return nil, fmt.Errorf("generated fault matrix row names an undeclared fault class")
		}
		if status, ok := expectedStatuses[fault]; !ok || stringField(row, "observed_status") != status || stringField(row, "expected_status") != status || stringField(fallback, "status") != status || stringField(fallback, "behavior") != expectedBehaviors[fault] {
			return nil, fmt.Errorf("generated fault matrix row has an unexpected fallback for %q", fault)
		}
		seenPrograms[program] = struct{}{}
		seenFaults[fault] = struct{}{}
		pair := program + "\x00" + fault
		if _, duplicate := seenPairs[pair]; duplicate {
			return nil, fmt.Errorf("generated fault matrix repeats a program/fault row")
		}
		seenPairs[pair] = struct{}{}
	}
	if len(seenPrograms) != int(programs) || len(seenFaults) != len(faults) || len(seenPairs) != int(rows) {
		return nil, fmt.Errorf("generated fault matrix identities do not match its summary")
	}
	if err := validateProgramSourceExecution(payload["program_source_execution"], programs, expectedFaults, seenPrograms); err != nil {
		return nil, err
	}
	return payload, nil
}

func validateProgramSourceExecution(value any, expectedPrograms float64, expectedFaults, expectedProgramIDs map[string]struct{}) error {
	execution, ok := value.(map[string]any)
	if !ok || execution["execution_kind"] != "program_runtime_session_with_injected_library_handles" || execution["native_qualification"] != false {
		return fmt.Errorf("program source fault execution must identify its injected runtime mode and remain fixture-only")
	}
	programs, programsOK := execution["program_count"].(float64)
	rows, rowsOK := execution["row_count"].(float64)
	passed, passedOK := execution["passed_rows"].(float64)
	failed, failedOK := execution["failed_rows"].(float64)
	faults, faultsOK := execution["fault_classes"].([]any)
	matrix, matrixOK := execution["matrix"].([]any)
	if !programsOK || !rowsOK || !passedOK || !failedOK || !faultsOK || !matrixOK || programs != expectedPrograms || rows != programs*float64(len(expectedFaults)) || passed != rows || failed != 0 || len(matrix) != int(rows) || len(faults) != len(expectedFaults) {
		return fmt.Errorf("program source fault execution is incomplete or has failed rows")
	}
	seenFaults := map[string]struct{}{}
	for _, value := range faults {
		fault, ok := value.(string)
		if !ok {
			return fmt.Errorf("program source fault execution has an invalid fault class")
		}
		if _, expected := expectedFaults[fault]; !expected {
			return fmt.Errorf("program source fault execution names an undeclared fault class")
		}
		seenFaults[fault] = struct{}{}
	}
	if len(seenFaults) != len(expectedFaults) {
		return fmt.Errorf("program source fault execution omits a fault class")
	}
	seenPrograms, seenPairs := map[string]struct{}{}, map[string]struct{}{}
	for _, value := range matrix {
		row, ok := value.(map[string]any)
		if !ok || row["passed"] != true || row["native_qualification"] != false || row["execution_kind"] != execution["execution_kind"] {
			return fmt.Errorf("program source fault execution contains an invalid row")
		}
		program, programOK := row["program"].(string)
		fault, faultOK := row["fault"].(string)
		dependencies, dependenciesOK := row["injected_dependencies"].([]any)
		children, childrenOK := row["observed_child_envelopes"].([]any)
		if !programOK || !faultOK || program == "" || fault == "" || !dependenciesOK || len(dependencies) == 0 || !childrenOK || len(children) == 0 || stringField(row, "parent_status") == "" || stringField(row, "parent_phase") == "" {
			return fmt.Errorf("program source fault execution row omits an observed parent or child path")
		}
		if row["no_retry"] != true {
			return fmt.Errorf("program source fault execution retried an injected child")
		}
		fallback, fallbackOK := row["observable_fallback"].(map[string]any)
		capabilities, capabilitiesOK := row["capability_requirements"].([]any)
		if !fallbackOK || fallback["observed"] != true || stringField(fallback, "behavior") == "" || stringField(fallback, "status") != stringField(row, "parent_status") || stringField(fallback, "phase") != stringField(row, "parent_phase") || !capabilitiesOK || len(capabilities) == 0 {
			return fmt.Errorf("program source fault row omits the caller's observable fallback or capability requirements")
		}
		parentStatus := stringField(row, "parent_status")
		parentPhase := stringField(row, "parent_phase")
		if !stringListContains(row["parent_status_enum"], parentStatus) || !stringListContains(row["parent_phase_enum"], parentPhase) {
			return fmt.Errorf("program source fault row returned a parent status or phase outside its contract")
		}
		fallbackErrors, _ := fallback["error_classes"].([]any)
		fallbackMarkers, _ := fallback["signal_markers"].([]any)
		if parentStatus == "ok" && len(fallbackErrors) == 0 && len(fallbackMarkers) == 0 {
			return fmt.Errorf("successful parent status hides an injected failure without explicit fallback evidence")
		}
		if _, expected := expectedFaults[fault]; !expected {
			return fmt.Errorf("program source fault execution contains an undeclared fault")
		}
		seenPrograms[program] = struct{}{}
		pair := program + "\x00" + fault
		if _, duplicate := seenPairs[pair]; duplicate {
			return fmt.Errorf("program source fault execution repeats a program/fault row")
		}
		seenPairs[pair] = struct{}{}
		expectedStatuses, expectedStatusesOK := row["expected_status_by_dependency"].(map[string]any)
		if !expectedStatusesOK || len(expectedStatuses) == 0 {
			return fmt.Errorf("program source fault execution row omits expected child status")
		}
		injected := map[string]struct{}{}
		libraryCapabilities := map[string]struct{}{}
		for _, capabilityValue := range capabilities {
			capability, ok := capabilityValue.(map[string]any)
			if !ok || !((stringField(capability, "binding") != "" && stringField(capability, "effect") != "") || stringField(capability, "library_program") != "") {
				return fmt.Errorf("program source fault row has an invalid capability requirement")
			}
			if program := stringField(capability, "library_program"); program != "" {
				libraryCapabilities[program] = struct{}{}
			}
		}
		for _, dependencyValue := range dependencies {
			dependency, ok := dependencyValue.(string)
			if !ok || dependency == "" {
				return fmt.Errorf("program source fault execution row has an invalid injected dependency")
			}
			if _, duplicate := injected[dependency]; duplicate {
				return fmt.Errorf("program source fault execution retried the same injected dependency")
			}
			if _, declared := libraryCapabilities[dependency]; !declared {
				return fmt.Errorf("program source fault execution dependency is absent from its declared capability requirements")
			}
			injected[dependency] = struct{}{}
		}
		observed := map[string]struct{}{}
		for _, childValue := range children {
			child, ok := childValue.(map[string]any)
			if !ok || stringField(child, "dependency") == "" || stringField(child, "status") == "" || stringField(child, "status") == "exception" {
				return fmt.Errorf("program source fault execution row has an invalid observed child envelope")
			}
			dependency := stringField(child, "dependency")
			if _, found := injected[dependency]; !found || stringField(child, "status") != stringField(expectedStatuses, dependency) {
				return fmt.Errorf("program source fault execution row differs from its expected child status")
			}
			if _, duplicate := observed[dependency]; duplicate {
				return fmt.Errorf("program source fault execution observed a repeated child call")
			}
			observed[dependency] = struct{}{}
		}
		if len(observed) != len(injected) {
			return fmt.Errorf("program source fault execution omits an injected dependency result")
		}
	}
	if len(seenPrograms) != int(programs) || len(seenPairs) != int(rows) || len(expectedProgramIDs) != len(seenPrograms) {
		return fmt.Errorf("program source fault execution identities do not match its summary")
	}
	for program := range expectedProgramIDs {
		if _, exists := seenPrograms[program]; !exists {
			return fmt.Errorf("program source fault execution does not match the helper matrix program set")
		}
	}
	return nil
}

func stringField(value map[string]any, key string) string {
	field, _ := value[key].(string)
	return field
}

func stringListContains(value any, wanted string) bool {
	items, ok := value.([]any)
	if !ok || wanted == "" {
		return false
	}
	for _, item := range items {
		if candidate, ok := item.(string); ok && candidate == wanted {
			return true
		}
	}
	return false
}

func extractFixtureReceipts(output string) (map[string]any, error) {
	const prefix = "DB6_FIXTURE_RECEIPTS="
	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, prefix)), &payload); err != nil {
			return nil, fmt.Errorf("DB6 fixture receipt evidence is invalid JSON")
		}
		if payload["evidence_kind"] != "fixture" || payload["native_qualification"] != false {
			return nil, fmt.Errorf("DB6 receipt evidence must remain explicitly fixture-only")
		}
		return payload, nil
	}
	return nil, fmt.Errorf("DB6 typed receipt evidence is missing")
}

func boundedOutput(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	return value[:maxBytes] + "\n[output truncated]"
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (s *Service) PreviewFix(context.Context, *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("provider-conformance fixes are manual: descriptor authorship requires provider-owner judgment"))
}

func (s *Service) ApplyFix(context.Context, *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("provider-conformance fixes are manual: descriptor authorship requires provider-owner judgment"))
}

func nativeDetail(report providerconformance.Report) (*structpb.Struct, error) {
	detail := map[string]any{
		"scenario": report.Scenario,
		"status":   report.Summary.Status(),
		"probed":   report.Probed,
		"summary": map[string]any{
			"errors":   report.Summary.Errors,
			"warnings": report.Summary.Warnings,
		},
	}
	if report.Phase != "" {
		detail["phase"] = report.Phase
	}
	if report.ProbeSkipReason != "" {
		detail["probeSkipReason"] = report.ProbeSkipReason
	}
	return structpb.NewStruct(detail)
}

// Served composes this service with the shared DescribeProvider implementation.
// Test Genie is itself a provider (the provider-conformance phase), so it must
// answer the same readiness contract it asks every other provider to answer.
func (s *Service) Served() scenariovalidationv1connect.ScenarioValidationServiceHandler {
	return assessment.Serve(s, s.describer)
}
