package validation

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"connectrpc.com/connect"

	"structure-health/internal/autofix"
	"structure-health/internal/portswitch"
	internalvalidation "structure-health/internal/validation"

	"github.com/vrooli/api-core/metrics"
	"github.com/vrooli/maturity-go/assessment"
	autofixcore "github.com/vrooli/maturity-go/autofix"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/validation/validation_v1connect"
)

// Deps are the handler's collaborators.
type Deps struct {
	Service      *internalvalidation.Service
	Logger       *log.Logger
	MaturitySpec *assessment.Spec
	// Environment is the host CaptureEnvironment captured once at module init.
	// nil is safe — the metrics collector backfills os/arch/num_cpu.
	Environment *commonv1.CaptureEnvironment
}

// Handler implements the generated native ValidationServiceHandler.
type Handler struct {
	validationconnect.UnimplementedValidationServiceHandler
	svc    *internalvalidation.Service
	logger *log.Logger
	spec   *assessment.Spec
	env    *commonv1.CaptureEnvironment
}

// NewHandlerWithDeps builds a Handler, defaulting nil collaborators.
func NewHandlerWithDeps(deps Deps) *Handler {
	if deps.Logger == nil {
		deps.Logger = log.Default()
	}
	if deps.Service == nil {
		deps.Service = internalvalidation.New()
	}
	return &Handler{svc: deps.Service, logger: deps.Logger, spec: deps.MaturitySpec, env: deps.Environment}
}

var _ validationconnect.ValidationServiceHandler = (*Handler)(nil)

// ValidateScenario reconciles the target scenario's structure and returns the
// rich native response (the shared assessment is mirrored inside it).
func (h *Handler) ValidateScenario(ctx context.Context, req *connect.Request[validationv1.ValidateScenarioRequest]) (*connect.Response[validationv1.ValidateScenarioResponse], error) {
	if req.Msg.GetScenario() == "" && req.Msg.GetPath() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario or path is required"))
	}
	report, err := h.svc.Validate(ctx, internalvalidation.Request{
		Scenario:         req.Msg.GetScenario(),
		Path:             req.Msg.GetPath(),
		IncludeExecution: req.Msg.GetIncludeExecution(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	resp, err := responseToProto(report, h.spec)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build maturity assessment: %w", err))
	}
	return connect.NewResponse(resp), nil
}

// PreviewFixConfig reports the deterministic structure/service.json fixes
// structure-health could apply, without writing anything.
func (h *Handler) PreviewFixConfig(ctx context.Context, req *connect.Request[validationv1.FixConfigRequest]) (*connect.Response[validationv1.FixConfigResponse], error) {
	if req.Msg.GetScenario() == "" && req.Msg.GetPath() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario or path is required"))
	}
	scenario, candidates, err := h.svc.PreviewFix(ctx, req.Msg.GetScenario(), req.Msg.GetPath(), req.Msg.GetRuleIds())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(fixResponseToProto(scenario, false, candidates)), nil
}

// ApplyFixConfig applies the deterministic fixes (format-preserving service.json
// edits / skeleton file creation) and reports what changed.
func (h *Handler) ApplyFixConfig(ctx context.Context, req *connect.Request[validationv1.FixConfigRequest]) (*connect.Response[validationv1.FixConfigResponse], error) {
	if req.Msg.GetScenario() == "" && req.Msg.GetPath() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario or path is required"))
	}
	scenario, candidates, err := h.svc.ApplyFix(ctx, req.Msg.GetScenario(), req.Msg.GetPath(), req.Msg.GetRuleIds())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(fixResponseToProto(scenario, true, candidates)), nil
}

// AssignFixedPort switches a scenario's port from a canonical range to a free
// in-band fixed port so it can be exposed as a scenario route.
func (h *Handler) AssignFixedPort(ctx context.Context, req *connect.Request[validationv1.PortSwitchRequest]) (*connect.Response[validationv1.PortSwitchResponse], error) {
	if req.Msg.GetScenario() == "" && req.Msg.GetPath() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario or path is required"))
	}
	res, err := h.svc.AssignFixedPort(ctx, req.Msg.GetScenario(), req.Msg.GetPath(), req.Msg.GetPortName(), req.Msg.GetApply())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(portSwitchToProto(res)), nil
}

// ReleaseFixedPort reverts a scenario's port back to the canonical range.
func (h *Handler) ReleaseFixedPort(ctx context.Context, req *connect.Request[validationv1.PortSwitchRequest]) (*connect.Response[validationv1.PortSwitchResponse], error) {
	if req.Msg.GetScenario() == "" && req.Msg.GetPath() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario or path is required"))
	}
	res, err := h.svc.ReleaseFixedPort(ctx, req.Msg.GetScenario(), req.Msg.GetPath(), req.Msg.GetPortName(), req.Msg.GetApply())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(portSwitchToProto(res)), nil
}

func portSwitchToProto(r portswitch.Result) *validationv1.PortSwitchResponse {
	return &validationv1.PortSwitchResponse{
		Scenario:     r.Scenario,
		PortName:     r.PortName,
		PreviousPort: int32(r.PreviousPort),
		AssignedPort: int32(r.AssignedPort),
		Changed:      r.Changed,
		Applied:      r.Applied,
		Before:       r.Before,
		After:        r.After,
		Message:      r.Message,
	}
}

func fixResponseToProto(scenario string, applied bool, candidates []autofix.Candidate) *validationv1.FixConfigResponse {
	out := &validationv1.FixConfigResponse{Scenario: scenario, Applied: applied}
	for _, c := range candidates {
		out.Candidates = append(out.Candidates, &validationv1.AutofixCandidate{
			RuleId:      c.RuleID,
			FilePath:    c.FilePath,
			Description: c.Description,
			Before:      c.Before,
			After:       c.After,
			Applied:     c.Applied,
		})
	}
	if len(candidates) == 0 {
		out.Messages = append(out.Messages, "No auto-fixable structure findings are available.")
	}
	return out
}

// SharedHandler adapts structure-health's native validation RPC to the shared
// ScenarioValidationService contract consumed by Test Genie.
type SharedHandler struct {
	handler *Handler
}

// NewSharedHandler wraps a native Handler in the shared adapter.
func NewSharedHandler(handler *Handler) *SharedHandler {
	return &SharedHandler{handler: handler}
}

// ValidateScenario runs the native engine and returns the shared response with
// the native detail packed into native_detail and execution metrics attached.
func (h *SharedHandler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	if h == nil || h.handler == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("structure validation handler not wired"))
	}
	collector := metrics.Start(metrics.WithEnvironment(h.handler.env))
	native, err := h.handler.ValidateScenario(ctx, connect.NewRequest(&validationv1.ValidateScenarioRequest{
		Scenario:         req.Msg.GetScenario(),
		Path:             req.Msg.GetPath(),
		IncludeExecution: req.Msg.GetIncludeExecution(),
	}))
	if err != nil {
		collector.Stop()
		return nil, err
	}
	execMetrics := collector.Stop()
	resp, err := assessment.BuildValidationResponse(
		native.Msg.GetScenario(),
		native.Msg.GetAssessment(),
		native.Msg,
		execMetrics,
		statusOverride(native.Msg)...,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build shared validation response: %w", err))
	}
	return connect.NewResponse(resp), nil
}

// ValidateTarget is the explicit generalized target seam for all repository
// target kinds. The native engine and the shared assessment remain one path so
// CLI, fleet, and Test Genie consumers cannot drift.
func (h *SharedHandler) ValidateTarget(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateTargetRequest]) (*connect.Response[scenariovalidationv1.ValidateTargetResponse], error) {
	if h == nil || h.handler == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("structure validation handler not wired"))
	}
	target := req.Msg.GetTarget()
	if target == nil || target.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("target is required"))
	}
	targetKind := strings.TrimPrefix(strings.ToLower(target.GetKind().String()), "validation_target_kind_")
	targetKind = strings.ReplaceAll(targetKind, "_", "-")
	if targetKind == "unspecified" || targetKind == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("target kind is required"))
	}
	if targetKind != "scenario" && targetKind != "resource" && targetKind != "tool" && targetKind != "safeguard" && targetKind != "package" && targetKind != "control-plane" && targetKind != "docs" && targetKind != "team" && targetKind != "project" {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("structure-health does not support target kind %s", target.GetKind().String()))
	}
	collector := metrics.Start(metrics.WithEnvironment(h.handler.env))
	native, err := h.handler.svc.Validate(ctx, internalvalidation.Request{
		Scenario:         target.GetId(),
		Path:             req.Msg.GetPath(),
		TargetKind:       targetKind,
		TargetID:         target.GetId(),
		TargetRoot:       target.GetRoot(),
		IncludeExecution: req.Msg.GetIncludeExecution(),
	})
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	execMetrics := collector.Stop()
	nativeResp, err := responseToProto(native, h.handler.spec)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build native target response: %w", err))
	}
	sharedResp, err := assessment.BuildValidationResponse(
		nativeResp.GetScenario(),
		nativeResp.GetAssessment(),
		nativeResp,
		execMetrics,
		statusOverride(nativeResp)...,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build shared target response: %w", err))
	}
	return connect.NewResponse(&scenariovalidationv1.ValidateTargetResponse{
		Target:       target,
		Status:       sharedResp.GetStatus(),
		Assessment:   sharedResp.GetAssessment(),
		NativeDetail: sharedResp.GetNativeDetail(),
		Metrics:      sharedResp.GetMetrics(),
	}), nil
}

// PreviewFix exposes structure-health's deterministic fixes through the shared
// scenario-validation Fix RPC (dry-run). It reuses the native engine so the
// shared and native surfaces stay in lockstep.
func (h *SharedHandler) PreviewFix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	if h == nil || h.handler == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("structure validation handler not wired"))
	}
	if req.Msg.GetScenario() == "" && req.Msg.GetPath() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario or path is required"))
	}
	scenario, candidates, err := h.handler.svc.PreviewFix(ctx, req.Msg.GetScenario(), req.Msg.GetPath(), req.Msg.GetRuleIds())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(autofixcore.BuildFixResponse(firstNonEmpty(scenario, req.Msg.GetScenario()), false, candidates)), nil
}

// ApplyFix applies structure-health's deterministic fixes through the shared
// scenario-validation Fix RPC and reports what changed.
func (h *SharedHandler) ApplyFix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	if h == nil || h.handler == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("structure validation handler not wired"))
	}
	if req.Msg.GetScenario() == "" && req.Msg.GetPath() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario or path is required"))
	}
	scenario, candidates, err := h.handler.svc.ApplyFix(ctx, req.Msg.GetScenario(), req.Msg.GetPath(), req.Msg.GetRuleIds())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(autofixcore.BuildFixResponse(firstNonEmpty(scenario, req.Msg.GetScenario()), true, candidates)), nil
}

func statusOverride(resp *validationv1.ValidateScenarioResponse) []assessment.ValidationResponseOption {
	switch strings.ToLower(strings.TrimSpace(resp.GetStatus())) {
	case "degraded":
		return []assessment.ValidationResponseOption{assessment.WithValidationStatus(scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_DEGRADED)}
	case "error":
		return []assessment.ValidationResponseOption{assessment.WithValidationStatus(scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_ERROR)}
	default:
		return nil
	}
}

func responseToProto(in internalvalidation.Response, spec *assessment.Spec) (*validationv1.ValidateScenarioResponse, error) {
	maturityAssessment, err := buildMaturityAssessment(in, spec)
	if err != nil {
		return nil, err
	}
	out := &validationv1.ValidateScenarioResponse{
		RunId:          in.RunID,
		Status:         in.Status,
		Summary:        in.Summary,
		Scenario:       in.Scenario,
		TargetPath:     in.TargetPath,
		DegradedReason: in.DegradedReason,
		Profile:        profileToProto(in.Profile),
		NextSteps:      in.NextSteps,
		Assessment:     maturityAssessment,
	}
	for _, s := range in.Surfaces {
		out.Surfaces = append(out.Surfaces, surfaceToProto(s))
	}
	for _, f := range in.Findings {
		out.Findings = append(out.Findings, findingToProto(f))
	}
	return out, nil
}

func buildMaturityAssessment(in internalvalidation.Response, spec *assessment.Spec) (*commonv1.MaturityAssessment, error) {
	if spec == nil {
		return nil, fmt.Errorf("maturity spec is required")
	}
	findings := make([]assessment.Finding, 0, len(in.Findings))
	for _, f := range in.Findings {
		findings = append(findings, assessment.Finding{
			Code:             f.Code,
			Severity:         severityToAssessment(f.Severity),
			Title:            firstNonEmpty(f.Title, f.Code),
			Message:          f.Message,
			Location:         f.Location,
			Remediation:      f.Remediation,
			Source:           architecturev1.FindingSource_FINDING_SOURCE_STRUCTURE,
			Phase:            spec.Phase,
			AutofixAvailable: f.AutofixAvailable,
			FixClass:         f.FixClass,
		})
	}
	return assessment.BuildProtoAssessment(assessment.BuildInput{
		Scenario: in.Scenario,
		Spec:     *spec,
		Findings: findings,
	})
}

func severityToAssessment(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "error":
		return architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR.String()
	case "warning", "warn":
		return architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING.String()
	case "info":
		return architecturev1.FindingSeverity_FINDING_SEVERITY_INFO.String()
	default:
		return severity
	}
}

func profileToProto(in internalvalidation.DetectedProfile) *validationv1.DetectedProfile {
	return &validationv1.DetectedProfile{
		Id:              in.ID,
		BackendLanguage: in.BackendLanguage,
		UiFramework:     in.UIFramework,
		Recognized:      in.Recognized,
		Evidence:        in.Evidence,
	}
}

func surfaceToProto(in internalvalidation.SurfaceReconcile) *validationv1.SurfaceReconcile {
	return &validationv1.SurfaceReconcile{
		Surface:        in.Surface,
		Kind:           in.Kind,
		Declared:       in.Declared,
		Actual:         in.Actual,
		DeclaredDetail: in.DeclaredDetail,
		ActualDetail:   in.ActualDetail,
	}
}

func findingToProto(in internalvalidation.Finding) *validationv1.StructureFinding {
	return &validationv1.StructureFinding{
		Code:             in.Code,
		Severity:         in.Severity,
		Title:            in.Title,
		Message:          in.Message,
		Location:         in.Location,
		Remediation:      in.Remediation,
		Surface:          in.Surface,
		AutofixAvailable: in.AutofixAvailable,
		FixClass:         in.FixClass,
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
