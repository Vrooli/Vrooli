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
	logger    *log.Logger
	validator *providerconformance.Service
	spec      *assessment.Spec
	faultSpec *assessment.Spec
	repoRoot  string
	faultRun  func(context.Context, string) (string, error)
	describer assessment.Describer
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
	return &Service{logger: logger, validator: validator, spec: spec, faultSpec: faultSpec, repoRoot: repoRoot, faultRun: runDBMFaultInjection, describer: describer}
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
	if req.GetScenario() != "data-backup-manager" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("fault-injection only validates data-backup-manager"))
	}
	if !req.GetIncludeExecution() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("fault-injection requires execution mode"))
	}
	if s.faultSpec == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fault-injection maturity spec is unavailable"))
	}
	root, err := s.resolveDBMTarget(req.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	runner := s.faultRun
	if runner == nil {
		runner = runDBMFaultInjection
	}
	collector := metrics.Start()
	output, runErr := runner(ctx, root)
	receiptEvidence, receiptErr := extractFixtureReceipts(output)
	if runErr == nil && receiptErr != nil {
		runErr = receiptErr
	}
	findings := []assessment.Finding(nil)
	status := scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED
	if runErr != nil {
		status = scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED
		findings = append(findings, assessment.Finding{Code: "dbm_fault_injection.execution_failed", Severity: "SEVERITY_ERROR", Title: "Simulated recovery fault injection failed", Message: runErr.Error(), Location: "scenarios/data-backup-manager/scripts/db6-fault-injection.sh", Remediation: "Inspect the Test Genie phase evidence and repair the failing owner contract."})
	}
	maturity, err := assessment.BuildProtoAssessment(assessment.BuildInput{Scenario: "data-backup-manager", Spec: *s.faultSpec, Findings: findings})
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build fault-injection maturity assessment: %w", err))
	}
	detail, err := structpb.NewStruct(map[string]any{"phase": "fault-injection", "outcome": status.String(), "evidence_kind": "fixture", "native_qualification": false, "capability_receipts": receiptEvidence, "output": boundedOutput(output, 12000)})
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp, err := assessment.BuildValidationResponse("data-backup-manager", maturity, detail, collector.Stop(), assessment.WithValidationStatus(status))
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

func runDBMFaultInjection(ctx context.Context, scenarioRoot string) (string, error) {
	cmd := exec.CommandContext(ctx, "bash", filepath.Join(scenarioRoot, "scripts", "db6-fault-injection.sh"))
	cmd.Dir = scenarioRoot
	output, err := cmd.CombinedOutput()
	return string(output), err
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
