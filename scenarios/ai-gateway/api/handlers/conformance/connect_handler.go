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
	internalrouting "ai-gateway/internal/routing"
)

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
	path, err := projectValidationPath(target, req.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeUnimplemented, err)
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

func projectValidationPath(target *commonv1.ValidationTarget, path string) (string, error) {
	if target.GetKind().String() != "VALIDATION_TARGET_KIND_PROJECT" {
		return "", errors.New("ai-gateway target validation is only defined for project targets")
	}
	if path == "" {
		path = target.GetRoot()
	}
	return path, nil
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
	collector := metrics.Start()
	readiness, readinessErr := h.readModelEvalReadiness(ctx)
	findings := modelEvalReadinessFindings(readiness, readinessErr)
	execution := h.executeModelEval(ctx, req.Msg.GetIncludeExecution())
	findings = append(findings, execution.findings...)
	maturity, err := assessment.BuildProtoAssessment(assessment.BuildInput{
		Scenario: target.GetId(), Spec: *spec, Findings: findings, Target: target,
	})
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build model-eval readiness assessment: %w", err))
	}
	status := modelEvalValidationStatus(readiness, readinessErr, execution.failed)
	native, err := modelEvalNativeDetail(readiness, readinessErr, execution)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, err)
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

func (h *connectHandler) readModelEvalReadiness(ctx context.Context) (internalconformance.ModelEvalReadiness, error) {
	roles, roleErr := internalconformance.LoadModelEvalRoles(h.deps.ModelPolicyPath)
	taskRoles, taskRoleErr := internalconformance.LoadGatewayTaskRoles(h.deps.TaskRoleCatalogPath)
	consumers, consumerErr := h.loadModelEvalConsumers(ctx)
	runtimeModels, runtimeModelsErr := h.loadModelEvalRuntimeModels(ctx)
	readiness := internalconformance.ModelEvalReadiness{}
	readinessErr := firstReadinessError(roleErr, taskRoleErr, consumerErr, runtimeModelsErr)
	if readinessErr == nil {
		readiness, readinessErr = internalconformance.ReadModelEvalReadiness(ctx, h.deps.ModelEvalSource, internalconformance.ModelEvalRoleDenominators{PolicyRoles: roles, TaskRoles: taskRoles, Consumers: consumers})
	}
	if readinessErr == nil {
		for _, item := range runtimeModels {
			readiness.ObservedRuntimeModels = append(readiness.ObservedRuntimeModels, internalconformance.ModelEvalRuntimeModel{
				Scenario: item.Scenario, Role: item.Role, Provider: item.Provider, Locality: item.Locality,
				Model: item.Model, Status: item.Status, SampleCount: item.SampleCount, LatestObservedAt: item.LatestObservedAt,
			})
		}
	}
	return readiness, readinessErr
}

func firstReadinessError(errorsInOrder ...error) error {
	for _, err := range errorsInOrder {
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *connectHandler) loadModelEvalConsumers(ctx context.Context) ([]string, error) {
	if h.deps.ModelEvalConsumers == nil {
		return nil, errors.New("route-ledger consumer inventory is not configured")
	}
	return h.deps.ModelEvalConsumers.ModelEvalConsumers(ctx)
}

func (h *connectHandler) loadModelEvalRuntimeModels(ctx context.Context) ([]internalrouting.ModelEvalRouteModel, error) {
	if h.deps.ModelEvalRuntimeModels == nil {
		return nil, errors.New("route-ledger runtime-model inventory is not configured")
	}
	return h.deps.ModelEvalRuntimeModels(ctx)
}

func modelEvalValidationStatus(readiness internalconformance.ModelEvalReadiness, readinessErr error, executionFailed bool) scenariovalidationv1.ValidationStatus {
	if executionFailed || readinessErr == nil && modelEvalReadinessFailed(readiness) {
		return scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED
	}
	return scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_DEGRADED
}

func modelEvalReadinessFailed(readiness internalconformance.ModelEvalReadiness) bool {
	return retrievalEvidenceFailed(readiness) || coverageFailed(readiness.RoleCoverage) ||
		coverageFailed(readiness.TaskRoleCoverage) || coverageFailed(readiness.ConsumerCoverage)
}

func retrievalEvidenceFailed(readiness internalconformance.ModelEvalReadiness) bool {
	return len(readiness.MissingDenominatorIDs) > 0 || len(readiness.InvalidDenominatorIDs) > 0 ||
		readiness.RegisteredSuites < readiness.RequiredRetrievalSuites ||
		len(readiness.MissingCorpusValidationIDs) > 0 || len(readiness.InvalidCorpusValidationIDs) > 0
}

func coverageFailed(coverage []internalconformance.RoleCoverage) bool {
	for _, role := range coverage {
		if role.Deficient {
			return true
		}
	}
	return false
}

func modelEvalReadinessFindings(readiness internalconformance.ModelEvalReadiness, readinessErr error) []assessment.Finding {
	findings := make([]assessment.Finding, 0, 4)
	if readinessErr != nil {
		return append(findings, assessment.Finding{
			Code: "model_eval.readiness_source_unavailable", Severity: "SEVERITY_ERROR",
			Title: "Model evaluation evidence source unavailable", Phase: "model-eval",
			Message:     "Could not read authoritative role policy or Search Hub accepted-suite evidence: " + readinessErr.Error(),
			Location:    "ai-gateway model-eval provider",
			Remediation: "Restore the resource policy and Search Hub EvalService endpoints; rerun model-eval readiness.",
		})
	}
	findings = appendRetrievalReadinessFinding(findings, readiness)
	if len(readiness.ObservedRuntimeModels) == 0 {
		findings = append(findings, assessment.Finding{
			Code: "model_eval.runtime_model_inventory_empty", Severity: "SEVERITY_ERROR",
			Title: "No effective runtime model observations found", Phase: "model-eval",
			Message:     "The complete AI Gateway route ledger contains no selected runtime model identity.",
			Location:    "AI Gateway route_events.selected_model",
			Remediation: "Record selected runtime model identity on routed calls and rerun the model-eval census.",
		})
	}
	findings = appendCoverageReadinessFindings(findings, readiness)
	return findings
}

type modelEvalExecution struct {
	state                 string
	liveInferenceExecuted bool
	failed                bool
	receipt               *ModelEvalExecutionReceipt
	findings              []assessment.Finding
}

func modelEvalNativeDetail(readiness internalconformance.ModelEvalReadiness, readinessErr error, execution modelEvalExecution) (*structpb.Struct, error) {
	readinessDetail, err := readinessJSONDetail(readiness)
	if err != nil {
		return nil, err
	}
	native, err := structpb.NewStruct(map[string]any{
		"phase": "model-eval", "readiness": readinessDetail, "source_error": errorText(readinessErr),
		"live_inference_executed": execution.liveInferenceExecuted,
		"execution_state":         modelEvalExecutionState(readiness, readinessErr, execution.state),
	})
	if err != nil {
		return nil, fmt.Errorf("encode model-eval readiness detail: %w", err)
	}
	if execution.receipt != nil {
		receiptDetail, err := jsonObject(execution.receipt)
		if err != nil {
			return nil, err
		}
		native.Fields["execution_receipt"], err = structpb.NewValue(receiptDetail)
		if err != nil {
			return nil, errors.New("build metadata-only model-eval execution receipt")
		}
	}
	return native, nil
}

func readinessJSONDetail(readiness internalconformance.ModelEvalReadiness) (map[string]any, error) {
	data, err := json.Marshal(readiness)
	if err != nil {
		return nil, fmt.Errorf("encode model-eval readiness: %w", err)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("decode model-eval readiness: %w", err)
	}
	return result, nil
}

func jsonObject(value any) (map[string]any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, errors.New("encode metadata-only model-eval execution receipt")
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, errors.New("decode metadata-only model-eval execution receipt")
	}
	return result, nil
}

func (h *connectHandler) executeModelEval(ctx context.Context, requested bool) modelEvalExecution {
	result := modelEvalExecution{state: "not_requested"}
	if !requested {
		result.findings = append(result.findings, assessment.Finding{
			Code: "model_eval.operator_inference_admission_required", Severity: "SEVERITY_ERROR",
			Title: "Live evaluation awaits operator admission", Phase: "model-eval",
			Message:     "Readiness inspection does not dispatch inference; the separate execution phase is waiting for operator admission.",
			Location:    "LM2 inference admission",
			Remediation: "Obtain the separate operator-routed LM2 inference admission before executing any candidate model.",
		})
		return result
	}
	result.state = "blocked_owner_executor_unavailable"
	if h.deps.ModelEvalExecutor == nil {
		result.failed = true
		result.findings = append(result.findings, unavailableModelEvalExecutorFinding())
		return result
	}
	receipt, err := h.deps.ModelEvalExecutor.ExecuteAgentManagerModelEval(ctx)
	result.receipt = &receipt
	result.liveInferenceExecuted = receipt.Succeeded+receipt.Failed > 0
	if err != nil {
		result.failed = true
		result.state = "owner_refused_or_failed"
		result.findings = append(result.findings, failedModelEvalExecutionFinding())
		return result
	}
	result.state = "terminal_receipt_recorded"
	return result
}

func unavailableModelEvalExecutorFinding() assessment.Finding {
	return assessment.Finding{
		Code: "model_eval.owner_executor_unavailable", Severity: "SEVERITY_ERROR",
		Title: "Agent Manager model-eval owner is unavailable", Phase: "model-eval",
		Message:     "The source-selecting Agent Manager execution owner is not configured; no live inference was attempted.",
		Location:    "Agent Manager approval-gated model-eval endpoint",
		Remediation: "Restore the authorized Agent Manager owner client and rerun the model-eval phase.",
	}
}

func failedModelEvalExecutionFinding() assessment.Finding {
	return assessment.Finding{
		Code: "model_eval.owner_execution_refused_or_failed", Severity: "SEVERITY_ERROR",
		Title: "Agent Manager model-eval batch did not complete", Phase: "model-eval",
		Message:     "The owner returned a refusal or failed terminal outcome; see its metadata-only receipt.",
		Location:    "Agent Manager approval-gated model-eval endpoint",
		Remediation: "Resolve the registered approval or the failed admission gate, then rerun the bounded batch.",
	}
}

func appendRetrievalReadinessFinding(findings []assessment.Finding, readiness internalconformance.ModelEvalReadiness) []assessment.Finding {
	if !retrievalEvidenceFailed(readiness) {
		return findings
	}
	return append(findings, assessment.Finding{
		Code: "model_eval.retrieval_denominators_incomplete", Severity: "SEVERITY_ERROR",
		Title: "Fixed retrieval denominators are incomplete", Phase: "model-eval",
		Message: fmt.Sprintf("Search Hub has %d registered suites; required floor is %d. Missing snapshots: %v. Invalid snapshots: %v. Missing corpus-validation receipts: %v. Invalid corpus-validation receipts: %v. Stale expected document cases remain in their fixed denominators and count as misses.",
			readiness.RegisteredSuites, readiness.RequiredRetrievalSuites,
			readiness.MissingDenominatorIDs, readiness.InvalidDenominatorIDs,
			readiness.MissingCorpusValidationIDs, readiness.InvalidCorpusValidationIDs),
		Location:    "search-hub EvalService fixed-denominator snapshots and corpus-validation receipts",
		Remediation: "Independently review each suite's positive cases, freeze a valid immutable denominator, then create a current provider-owned corpus-validation receipt. Do not remove stale expected documents; the evaluator counts them as misses.",
	})
}

func appendCoverageReadinessFindings(findings []assessment.Finding, readiness internalconformance.ModelEvalReadiness) []assessment.Finding {
	findings = appendCoverageFinding(findings, readiness.RoleCoverage, "model_eval.real_caller_coverage_incomplete", "Real caller coverage is incomplete", "Accepted source-linked caller cases per policy role: ", "Search Hub fixed denominator caller-audit cases", "Acquire authenticated Agent Manager caller events, bind successful Gateway receipts, independently review labels, and meet the role floor.")
	findings = appendCoverageFinding(findings, readiness.TaskRoleCoverage, "model_eval.gateway_task_role_coverage_incomplete", "AI Gateway task-role coverage is incomplete", "Accepted source-linked caller cases per AI Gateway task role: ", "AI Gateway inference-role-catalog caller cases", "Acquire and independently review real caller cases for every task role in the AI Gateway inference-role catalog.")
	findings = appendCoverageFinding(findings, readiness.ConsumerCoverage, "model_eval.consumer_coverage_incomplete", "Real caller consumer coverage is incomplete", "Accepted source-linked caller cases per observed route-ledger consumer: ", "Search Hub fixed denominator caller-audit cases and AI Gateway route ledger", "Independently review real caller cases for each observed consumer scenario until each reaches the caller floor.")
	return findings
}

func appendCoverageFinding(findings []assessment.Finding, coverage []internalconformance.RoleCoverage, code, title, prefix, location, remediation string) []assessment.Finding {
	deficient := make([]string, 0)
	for _, item := range coverage {
		if item.Deficient {
			deficient = append(deficient, fmt.Sprintf("%s=%d/%d", item.Role, item.Cases, item.Required))
		}
	}
	if len(deficient) == 0 {
		return findings
	}
	return append(findings, assessment.Finding{
		Code: code, Severity: "SEVERITY_ERROR", Title: title, Phase: "model-eval",
		Message: prefix + strings.Join(deficient, ", "), Location: location, Remediation: remediation,
	})
}

func modelEvalExecutionState(readiness internalconformance.ModelEvalReadiness, readinessErr error, executionState string) string {
	if executionState != "not_requested" {
		return executionState
	}
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
