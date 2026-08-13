package validation

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/anypb"

	internalvalidation "unit-health/internal/validation"

	"github.com/vrooli/api-core/metrics"
	"github.com/vrooli/maturity-go/assessment"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation/validation_v1connect"
)

// Deps are the handler's collaborators.
type Deps struct {
	Service      *internalvalidation.Service
	Logger       *log.Logger
	MaturitySpec *assessment.Spec
	// Environment is the host CaptureEnvironment captured once at module init
	// (os/arch/cpu/mem/present-GPUs). nil is safe — the metrics collector
	// backfills os/arch/num_cpu from the stdlib.
	Environment *commonv1.CaptureEnvironment
}

// Handler implements the generated ValidationServiceHandler.
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

// ValidateScenario discovers, plans, executes, and analyzes the target's tests.
func (h *Handler) ValidateScenario(ctx context.Context, req *connect.Request[validationv1.ValidateScenarioRequest]) (*connect.Response[validationv1.ValidateScenarioResponse], error) {
	if req.Msg.GetScenario() == "" && req.Msg.GetPath() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario or path is required"))
	}
	report, err := h.svc.Validate(ctx, internalvalidation.Request{
		Scenario:         req.Msg.GetScenario(),
		TargetKind:       "scenario",
		Path:             req.Msg.GetPath(),
		Workspaces:       req.Msg.GetWorkspaces(),
		IncludeExecution: req.Msg.GetIncludeExecution(),
		UseCache:         req.Msg.GetUseCache(),
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

// SharedHandler adapts Unit Health's rich validation RPC to the shared
// ScenarioValidationService contract consumed by Test Genie.
type SharedHandler struct {
	handler *Handler
}

func NewSharedHandler(handler *Handler) *SharedHandler {
	return &SharedHandler{handler: handler}
}

func (h *SharedHandler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	if h == nil || h.handler == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("unit validation handler not wired"))
	}
	collector := metrics.Start(metrics.WithEnvironment(h.handler.env))
	native, err := h.handler.ValidateScenario(internalvalidation.WithMetrics(ctx, collector), connect.NewRequest(&validationv1.ValidateScenarioRequest{
		Scenario:         req.Msg.GetScenario(),
		Path:             req.Msg.GetPath(),
		IncludeExecution: req.Msg.GetIncludeExecution(),
		UseCache:         true,
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

func (h *SharedHandler) ValidateTarget(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateTargetRequest]) (*connect.Response[scenariovalidationv1.ValidateTargetResponse], error) {
	target := req.Msg.GetTarget()
	if target == nil || target.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("target is required"))
	}
	path := req.Msg.GetPath()
	if path == "" {
		path = target.GetRoot()
	}
	targetKind := validationTargetKindString(target.GetKind())
	report, err := h.handler.svc.Validate(ctx, internalvalidation.Request{
		Scenario:         target.GetId(),
		TargetKind:       targetKind,
		Path:             path,
		IncludeExecution: req.Msg.GetIncludeExecution(),
		UseCache:         true,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	native, err := responseToProto(report, h.handler.spec)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build target validation response: %w", err))
	}
	detail, err := anypb.New(native)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("pack target validation response: %w", err))
	}
	return connect.NewResponse(&scenariovalidationv1.ValidateTargetResponse{
		Target:       target,
		Status:       validationStatusFromString(native.GetStatus()),
		Assessment:   native.GetAssessment(),
		NativeDetail: detail,
	}), nil
}

func validationStatusFromString(status string) scenariovalidationv1.ValidationStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "passed":
		return scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED
	case "failed", "failing":
		return scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED
	case "degraded":
		return scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_DEGRADED
	case "error":
		return scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_ERROR
	default:
		return scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_UNSPECIFIED
	}
}

func validationTargetKindString(kind commonv1.ValidationTargetKind) string {
	switch kind {
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO:
		return "scenario"
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PACKAGE:
		return "package"
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_CONTROL_PLANE:
		return "control-plane"
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_RESOURCE:
		return "resource"
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_TOOL:
		return "tool"
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SAFEGUARD:
		return "safeguard"
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_TEAM:
		return "team"
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_DOCS:
		return "docs"
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PROJECT:
		return "project"
	default:
		return "unspecified"
	}
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
	errCount, warnCount, infoCount := findingCounts(in.Findings)
	maturityAssessment, err := buildMaturityAssessment(in, spec)
	if err != nil {
		return nil, err
	}
	out := &validationv1.ValidateScenarioResponse{
		RunId:          in.RunID,
		Status:         in.Status,
		Summary:        in.Summary,
		Scenario:       in.Scenario,
		TargetKind:     in.TargetKind,
		TargetPath:     in.TargetPath,
		DegradedReason: in.DegradedReason,
		Plan:           planToProto(in.Plan),
		Maturity: &validationv1.MaturitySummary{
			Rung:      int32(in.Maturity.Rung),
			Label:     in.Maturity.Label,
			Rationale: in.Maturity.Rationale,
		},
		Counts: &validationv1.ValidationCounts{
			Errors:             int32(errCount),
			Warnings:           int32(warnCount),
			Infos:              int32(infoCount),
			Surfaces:           int32(len(in.Surfaces)),
			Workspaces:         int32(len(in.Workspaces)),
			CoverageTargets:    int32(len(in.Coverage)),
			SuppressedFindings: int32(len(in.SuppressedFindings)),
		},
		NextSteps:  in.NextSteps,
		Assessment: maturityAssessment,
	}
	for _, s := range in.Surfaces {
		out.Surfaces = append(out.Surfaces, surfaceToProto(s))
	}
	for _, w := range in.Workspaces {
		out.Workspaces = append(out.Workspaces, workspaceToProto(w))
	}
	for _, r := range in.CommandResults {
		out.CommandResults = append(out.CommandResults, commandToProto(r))
	}
	for _, c := range in.Coverage {
		out.Coverage = append(out.Coverage, coverageToProto(c))
	}
	for _, p := range in.ProjectionChecks {
		out.ProjectionChecks = append(out.ProjectionChecks, projectionCheckToProto(p))
	}
	for _, f := range in.Findings {
		out.Findings = append(out.Findings, findingToProto(f))
	}
	for _, f := range in.SuppressedFindings {
		out.SuppressedFindings = append(out.SuppressedFindings, findingToProto(f))
	}
	for _, d := range in.Diagnostics {
		out.Diagnostics = append(out.Diagnostics, diagnosticToProto(d))
	}
	for _, a := range in.Artifacts {
		out.Artifacts = append(out.Artifacts, artifactToProto(a))
	}
	return out, nil
}

func artifactToProto(in internalvalidation.Artifact) *validationv1.Artifact {
	return &validationv1.Artifact{Label: in.Label, Kind: in.Kind, Reference: in.Reference}
}

func buildMaturityAssessment(in internalvalidation.Response, spec *assessment.Spec) (*commonv1.MaturityAssessment, error) {
	if spec == nil {
		return nil, fmt.Errorf("maturity spec is required")
	}
	findings := make([]assessment.Finding, 0, len(in.Findings))
	for _, f := range in.Findings {
		findings = append(findings, assessment.Finding{
			Code:        f.Code,
			Severity:    severityToAssessment(f.Severity),
			Title:       f.Code,
			Message:     f.Message,
			Location:    f.FilePath,
			Remediation: f.Remediation,
			Source:      findingSource(f.Code, spec),
			Phase:       spec.Phase,
		})
	}
	return assessment.BuildProtoAssessment(assessment.BuildInput{
		Scenario: in.Scenario,
		Spec:     *spec,
		Findings: findings,
	})
}

// findingSource resolves the architecture finding source from the finding's
// declared dimension: coverage-dimension findings are coverage signals, all
// other Unit Health findings are test-standard signals.
func findingSource(code string, spec *assessment.Spec) architecturev1.FindingSource {
	if spec != nil {
		if mapping, ok := spec.Findings[code]; ok && mapping.Dimension == "coverage" {
			return architecturev1.FindingSource_FINDING_SOURCE_COVERAGE
		}
	}
	return architecturev1.FindingSource_FINDING_SOURCE_STANDARDS
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

func surfaceToProto(in internalvalidation.Surface) *validationv1.TestSurface {
	return &validationv1.TestSurface{Id: in.ID, Kind: in.Kind, Language: in.Language, Framework: in.Framework, RootPath: in.RootPath, PackageManager: in.PackageManager, Status: in.Status, Confidence: in.Confidence}
}

func workspaceToProto(in internalvalidation.Workspace) *validationv1.TestWorkspace {
	return &validationv1.TestWorkspace{Id: in.ID, Language: in.Language, RootPath: in.RootPath, Framework: in.Framework, CanonicalFramework: in.CanonicalFramework, TestCommand: in.TestCommand, CoverageCommand: in.CoverageCommand, PackageManager: in.PackageManager, Status: in.Status, DegradedReason: in.DegradedReason}
}

func planToProto(in internalvalidation.ExecutionPlan) *validationv1.ExecutionPlan {
	out := &validationv1.ExecutionPlan{Notes: in.Notes}
	for _, c := range in.Commands {
		out.Commands = append(out.Commands, &validationv1.PlannedCommand{WorkspaceId: c.WorkspaceID, Name: c.Name, Command: c.Command, WorkingDirectory: c.WorkingDirectory, TimeoutSeconds: int32(c.TimeoutSeconds)})
	}
	return out
}

func commandToProto(in internalvalidation.CommandResult) *validationv1.CommandResult {
	return &validationv1.CommandResult{Name: in.Name, Command: in.Command, WorkingDirectory: in.WorkingDirectory, Status: in.Status, ExitCode: int32(in.ExitCode), StdoutExcerpt: in.StdoutExcerpt, StderrExcerpt: in.StderrExcerpt, TimeoutSeconds: int32(in.TimeoutSeconds), FailureReason: in.FailureReason, FailureClass: in.FailureClass, DurationMs: in.DurationMS}
}

func coverageToProto(in internalvalidation.CoverageTarget) *validationv1.CoverageTarget {
	return &validationv1.CoverageTarget{Id: in.ID, Language: in.Language, SurfaceId: in.SurfaceID, FilePath: in.FilePath, CoveredLines: in.CoveredLines, TotalLines: in.TotalLines, CoveragePercent: in.CoveragePercent, Threshold: in.Threshold, Status: in.Status}
}

func projectionCheckToProto(in internalvalidation.ProjectionCheck) *validationv1.ProjectionCheck {
	return &validationv1.ProjectionCheck{Id: in.ID, WorkspaceId: in.WorkspaceID, SurfaceId: in.SurfaceID, Key: in.Key, Owner: in.Owner, FilePath: in.FilePath, PolicyValue: in.PolicyValue, NativeValue: in.NativeValue, Status: in.Status, Remediation: in.Remediation, FindingCode: in.FindingCode}
}

func findingToProto(in internalvalidation.Finding) *validationv1.ValidationFinding {
	return &validationv1.ValidationFinding{Id: in.ID, Scenario: in.Scenario, SurfaceId: in.SurfaceID, WorkspaceId: in.WorkspaceID, Language: in.Language, Framework: in.Framework, Code: in.Code, Category: in.Category, Severity: in.Severity, FilePath: in.FilePath, Symbol: in.Symbol, Message: in.Message, Evidence: in.Evidence, Expected: in.Expected, Observed: in.Observed, WhyItMatters: in.WhyItMatters, Remediation: in.Remediation, SourceCommand: in.SourceCommand, CreatedAt: in.CreatedAt}
}

func diagnosticToProto(in internalvalidation.Diagnostic) *validationv1.Diagnostic {
	return &validationv1.Diagnostic{Kind: in.Kind, WorkspaceId: in.WorkspaceID, Message: in.Message, Evidence: in.Evidence, Severity: in.Severity}
}

func findingCounts(findings []internalvalidation.Finding) (errCount, warnCount, infoCount int) {
	for _, f := range findings {
		switch strings.ToLower(f.Severity) {
		case "error":
			errCount++
		case "warning", "warn":
			warnCount++
		default:
			infoCount++
		}
	}
	return errCount, warnCount, infoCount
}
