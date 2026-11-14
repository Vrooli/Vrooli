package conformance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/metrics"
	"github.com/vrooli/maturity-go/assessment"
	conformancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/conformance"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/types/known/structpb"

	internalconformance "ai-gateway/internal/conformance"
)

type Deps struct {
	Scanner             *internalconformance.Scanner
	MaturitySpec        *assessment.Spec
	ScenarioDir         string
	ModelPolicyPath     string
	TaskRoleCatalogPath string
	ModelEvalSource     internalconformance.ModelEvalReadinessSource
}

type connectHandler struct {
	scanner *internalconformance.Scanner
	deps    Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Scanner == nil {
		d.Scanner = internalconformance.NewScanner()
	}
	return &connectHandler{scanner: d.Scanner, deps: d}
}

func (h *connectHandler) ScanScenario(ctx context.Context, req *connect.Request[conformancev1.ScanScenarioRequest]) (*connect.Response[conformancev1.ScanScenarioResponse], error) {
	report, err := h.scanner.Scan(ctx, internalconformance.ScanRequest{
		Scenario: req.Msg.GetScenario(),
		Path:     req.Msg.GetPath(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&conformancev1.ScanScenarioResponse{
		Scenario:        report.Scenario,
		MaturityLevel:   report.MaturityLevel,
		Findings:        report.Findings,
		Recommendations: report.Recommendations,
	}), nil
}

func (h *connectHandler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	if req.Msg.GetPhase() == "model-eval" {
		target := &commonv1.ValidationTarget{
			Id: req.Msg.GetScenario(), Kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO,
			Root: req.Msg.GetPath(),
		}
		result, err := h.validateModelEvalTarget(ctx, connect.NewRequest(&scenariovalidationv1.ValidateTargetRequest{
			Target: target, Path: req.Msg.GetPath(), Phase: req.Msg.GetPhase(),
			IncludeExecution: req.Msg.GetIncludeExecution(), CapabilitySubset: req.Msg.GetCapabilitySubset(),
			RetainedEvidenceSets: req.Msg.GetRetainedEvidenceSets(),
		}), target)
		if err != nil {
			return nil, err
		}
		return connect.NewResponse(&scenariovalidationv1.ValidateScenarioResponse{
			Scenario: req.Msg.GetScenario(), Status: result.Msg.GetStatus(), Assessment: result.Msg.GetAssessment(),
			NativeDetail: result.Msg.GetNativeDetail(), Metrics: result.Msg.GetMetrics(),
			FailureClassification: result.Msg.GetFailureClassification(),
		}), nil
	}
	collector := metrics.Start()
	report, err := h.scanner.Scan(ctx, internalconformance.ScanRequest{
		Scenario: req.Msg.GetScenario(),
		Path:     req.Msg.GetPath(),
	})
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if h.maturitySpec() == nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, errors.New("maturity spec is required"))
	}
	maturity, err := internalconformance.BuildMaturityAssessment(report, *h.maturitySpec())
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

func (h *connectHandler) ValidateTarget(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateTargetRequest]) (*connect.Response[scenariovalidationv1.ValidateTargetResponse], error) {
	target := req.Msg.GetTarget()
	if target == nil || target.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("target is required"))
	}
	if req.Msg.GetPhase() == "model-eval" {
		return h.validateModelEvalTarget(ctx, req, target)
	}
	path := req.Msg.GetPath()
	if path == "" {
		path = target.GetRoot()
	}
	if target.GetKind().String() != "VALIDATION_TARGET_KIND_PROJECT" {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("ai-gateway target validation is only defined for project targets"))
	}
	collector := metrics.Start()
	report, err := h.scanner.ScanProject(ctx, path)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if h.maturitySpec() == nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, errors.New("maturity spec is required"))
	}
	maturity, err := internalconformance.BuildMaturityAssessment(report, *h.maturitySpec())
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build target maturity assessment: %w", err))
	}
	native, err := nativeDetail(report)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	shared, err := assessment.BuildValidationResponse(report.Scenario, maturity, native, collector.Stop())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build target validation response: %w", err))
	}
	return connect.NewResponse(&scenariovalidationv1.ValidateTargetResponse{Target: target, Status: shared.GetStatus(), Assessment: shared.GetAssessment(), NativeDetail: shared.GetNativeDetail(), Metrics: shared.GetMetrics()}), nil
}

func (h *connectHandler) validateModelEvalTarget(
	ctx context.Context,
	req *connect.Request[scenariovalidationv1.ValidateTargetRequest],
	target *commonv1.ValidationTarget,
) (*connect.Response[scenariovalidationv1.ValidateTargetResponse], error) {
	if target.GetKind() != commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO || target.GetId() != "ai-gateway" {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("model-eval validates the ai-gateway scenario target only"))
	}
	spec, err := assessment.LoadSpecForPhase(h.deps.ScenarioDir, "model-eval")
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("load model-eval maturity descriptor: %w", err))
	}
	roles, roleErr := internalconformance.LoadModelEvalRoles(h.deps.ModelPolicyPath)
	taskRoles, taskRoleErr := internalconformance.LoadGatewayTaskRoles(h.deps.TaskRoleCatalogPath)
	collector := metrics.Start()
	readiness := internalconformance.ModelEvalReadiness{}
	readinessErr := roleErr
	if readinessErr == nil {
		readinessErr = taskRoleErr
	}
	if readinessErr == nil {
		readiness, readinessErr = internalconformance.ReadModelEvalReadiness(ctx, h.deps.ModelEvalSource, internalconformance.ModelEvalRoleDenominators{PolicyRoles: roles, TaskRoles: taskRoles})
	}
	findings := make([]assessment.Finding, 0, 4)
	if readinessErr != nil {
		findings = append(findings, assessment.Finding{
			Code: "model_eval.readiness_source_unavailable", Severity: "SEVERITY_ERROR",
			Title: "Model evaluation evidence source unavailable", Phase: "model-eval",
			Message:     "Could not read authoritative role policy or Search Hub accepted-suite evidence: " + readinessErr.Error(),
			Location:    "ai-gateway model-eval provider",
			Remediation: "Restore the resource policy and Search Hub EvalService endpoints; rerun model-eval readiness.",
		})
	} else {
		if len(readiness.MissingDenominatorIDs) > 0 || len(readiness.InvalidDenominatorIDs) > 0 || readiness.RegisteredSuites < readiness.RequiredRetrievalSuites {
			findings = append(findings, assessment.Finding{
				Code: "model_eval.retrieval_denominators_incomplete", Severity: "SEVERITY_ERROR",
				Title: "Fixed retrieval denominators are incomplete", Phase: "model-eval",
				Message:     fmt.Sprintf("Search Hub has %d registered suites; required floor is %d. Missing snapshots: %v. Invalid snapshots: %v.", readiness.RegisteredSuites, readiness.RequiredRetrievalSuites, readiness.MissingDenominatorIDs, readiness.InvalidDenominatorIDs),
				Location:    "search-hub EvalService fixed-denominator snapshots",
				Remediation: "Independently review each suite's positive cases and freeze a valid immutable denominator through Search Hub.",
			})
		}
		var deficient []string
		for _, coverage := range readiness.RoleCoverage {
			if coverage.Deficient {
				deficient = append(deficient, fmt.Sprintf("%s=%d/%d", coverage.Role, coverage.Cases, coverage.Required))
			}
		}
		if len(deficient) > 0 {
			findings = append(findings, assessment.Finding{
				Code: "model_eval.real_caller_coverage_incomplete", Severity: "SEVERITY_ERROR",
				Title: "Real caller coverage is incomplete", Phase: "model-eval",
				Message:     "Accepted source-linked caller cases per policy role: " + strings.Join(deficient, ", "),
				Location:    "Search Hub fixed denominator caller-audit cases",
				Remediation: "Acquire authenticated Agent Manager caller events, bind successful Gateway receipts, independently review labels, and meet the role floor.",
			})
		}
		var deficientTaskRoles []string
		for _, coverage := range readiness.TaskRoleCoverage {
			if coverage.Deficient {
				deficientTaskRoles = append(deficientTaskRoles, fmt.Sprintf("%s=%d/%d", coverage.Role, coverage.Cases, coverage.Required))
			}
		}
		if len(deficientTaskRoles) > 0 {
			findings = append(findings, assessment.Finding{
				Code: "model_eval.gateway_task_role_coverage_incomplete", Severity: "SEVERITY_ERROR",
				Title: "AI Gateway task-role coverage is incomplete", Phase: "model-eval",
				Message:     "Accepted source-linked caller cases per AI Gateway task role: " + strings.Join(deficientTaskRoles, ", "),
				Location:    "AI Gateway inference-role-catalog caller cases",
				Remediation: "Acquire and independently review real caller cases for every task role in the AI Gateway inference-role catalog.",
			})
		}
		findings = append(findings, assessment.Finding{
			Code: "model_eval.operator_inference_admission_required", Severity: "SEVERITY_ERROR",
			Title: "Live evaluation awaits operator admission", Phase: "model-eval",
			Message:     "Readiness inspection never runs inference. The live model-eval execution phase remains waiting-operator.",
			Location:    "LM2 inference admission",
			Remediation: "Obtain the separate operator-routed LM2 inference admission before executing any candidate model.",
		})
	}
	maturity, err := assessment.BuildProtoAssessment(assessment.BuildInput{
		Scenario: target.GetId(), Spec: *spec, Findings: findings, Target: target,
	})
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build model-eval readiness assessment: %w", err))
	}
	status := scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_DEGRADED
	if readinessErr == nil && (len(readiness.MissingDenominatorIDs) > 0 || len(readiness.InvalidDenominatorIDs) > 0 || readiness.RegisteredSuites < readiness.RequiredRetrievalSuites) {
		status = scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED
	}
	if readinessErr == nil {
		for _, coverage := range readiness.RoleCoverage {
			if coverage.Deficient {
				status = scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED
				break
			}
		}
		for _, coverage := range readiness.TaskRoleCoverage {
			if coverage.Deficient {
				status = scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED
				break
			}
		}
	}
	readinessBytes, err := json.Marshal(readiness)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encode model-eval readiness: %w", err))
	}
	var readinessDetail map[string]any
	if err := json.Unmarshal(readinessBytes, &readinessDetail); err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("decode model-eval readiness: %w", err))
	}
	native, err := structpb.NewStruct(map[string]any{
		"phase": "model-eval", "readiness": readinessDetail, "source_error": errorText(readinessErr),
		"live_inference_executed": false, "execution_state": modelEvalExecutionState(readiness, readinessErr),
	})
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encode model-eval readiness detail: %w", err))
	}
	shared, err := assessment.BuildValidationResponse(target.GetId(), maturity, native, collector.Stop(), assessment.WithValidationStatus(status))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build model-eval response: %w", err))
	}
	shared.FailureClassification = "model_eval_evidence_incomplete"
	return connect.NewResponse(&scenariovalidationv1.ValidateTargetResponse{
		Target: target, Status: shared.GetStatus(), Assessment: shared.GetAssessment(),
		NativeDetail: shared.GetNativeDetail(), Metrics: shared.GetMetrics(), FailureClassification: shared.GetFailureClassification(),
	}), nil
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func modelEvalExecutionState(readiness internalconformance.ModelEvalReadiness, readinessErr error) string {
	if readinessErr != nil {
		return "blocked_evidence_source_unavailable"
	}
	if !readiness.Ready {
		return "blocked_accepted_suite_evidence_incomplete"
	}
	return "waiting_operator_inference_admission"
}

func (h *connectHandler) PreviewFix(_ context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return connect.NewResponse(&scenariovalidationv1.FixResponse{
		Scenario: req.Msg.GetScenario(),
		Applied:  false,
		Messages: []string{"AI conformance has no deterministic source rewrite fixers; use the migration guidance in ValidateScenario/ScanScenario."},
	}), nil
}

func (h *connectHandler) ApplyFix(_ context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return connect.NewResponse(&scenariovalidationv1.FixResponse{
		Scenario: req.Msg.GetScenario(),
		Applied:  false,
		Messages: []string{"AI conformance apply is intentionally a no-op until safe deterministic migrations exist."},
	}), nil
}

func (h *connectHandler) maturitySpec() *assessment.Spec {
	return h.deps.MaturitySpec
}

func nativeDetail(report internalconformance.ScanReport) (*structpb.Struct, error) {
	findings := make([]any, 0, len(report.Findings))
	for _, finding := range report.Findings {
		findings = append(findings, map[string]any{
			"rule_id":     finding.GetRuleId(),
			"severity":    finding.GetSeverity(),
			"path":        finding.GetPath(),
			"message":     finding.GetMessage(),
			"remediation": finding.GetRemediation(),
		})
	}
	recommendations := make([]any, 0, len(report.Recommendations))
	for _, rec := range report.Recommendations {
		recommendations = append(recommendations, rec)
	}
	return structpb.NewStruct(map[string]any{
		"scenario":        report.Scenario,
		"maturity_level":  report.MaturityLevel,
		"findings":        findings,
		"recommendations": recommendations,
	})
}
