// Package validation hosts the shared ScenarioValidationService handler for
// security-health. The handler delegates real work to the internal validation
// Service and returns the common scenario-validation response shape.
package validation

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"

	"security-health/internal/validation"

	"github.com/vrooli/api-core/metrics"
	"github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// Validator is the slice of internal/validation.Service the handler exercises.
// An interface so handler tests can stub it without running real scanners.
type Validator interface {
	ValidateScenario(ctx context.Context, scenario string) (validation.Report, error)
}

type TargetValidator interface {
	ValidateTarget(ctx context.Context, kind validation.ValidationTargetKind, path string) (validation.Report, error)
}

type Fixer interface {
	PreviewFix(ctx context.Context, scenario, path string, ruleIDs []string) (string, []validation.SecurityHeaderFixCandidate, []string, error)
	ApplyFix(ctx context.Context, scenario, path string, ruleIDs []string) (string, []validation.SecurityHeaderFixCandidate, []string, error)
}

// Deps wires the seams the Connect validation handler needs.
type Deps struct {
	Logger          *log.Logger
	Validator       Validator
	TargetValidator TargetValidator
	RepoRoot        string
	Fixer           Fixer
	MaturitySpec    *assessment.Spec
	// Environment is the host CaptureEnvironment captured once at module init
	// (os/arch/cpu/mem/present-GPUs). nil is safe — the metrics collector
	// backfills os/arch/num_cpu from the stdlib.
	Environment *commonv1.CaptureEnvironment
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	if d.Fixer == nil {
		if fixer, ok := d.Validator.(Fixer); ok {
			d.Fixer = fixer
		}
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	if h.deps.Validator == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("validation.ValidateScenario: validator not wired"))
	}
	collector := metrics.Start(metrics.WithEnvironment(h.deps.Environment))
	report, err := h.deps.Validator.ValidateScenario(validation.WithMetrics(ctx, collector), req.Msg.GetScenario())
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	maturityAssessment, err := buildMaturityAssessment(report, h.deps.MaturitySpec)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build maturity assessment: %w", err))
	}
	execMetrics := collector.Stop()
	resp, err := assessment.BuildValidationResponse(report.Scenario, maturityAssessment, nil, execMetrics)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build shared validation response: %w", err))
	}
	return connect.NewResponse(resp), nil
}

// ValidateTarget exposes Security Health's path-first validator to shared
// validation consumers. The control-plane identity always resolves to the
// configured repository root; callers cannot substitute an arbitrary path for
// that privileged repository-wide scan.
func (h *connectHandler) ValidateTarget(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateTargetRequest]) (*connect.Response[scenariovalidationv1.ValidateTargetResponse], error) {
	if h.deps.TargetValidator == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("validation.ValidateTarget: target validator not wired"))
	}
	target := req.Msg.GetTarget()
	if target == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("validation target is required"))
	}

	var kind validation.ValidationTargetKind
	var path string
	switch target.GetKind() {
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO:
		kind = validation.ValidationTargetScenario
		if strings.TrimSpace(target.GetId()) == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("scenario target id is required"))
		}
		path = strings.TrimSpace(req.Msg.GetPath())
		if path == "" {
			path = strings.TrimSpace(target.GetRoot())
		}
		if path == "" {
			path = filepath.Join(h.deps.RepoRoot, "scenarios", target.GetId())
		}
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_CONTROL_PLANE:
		kind = validation.ValidationTargetControlPlane
		path = h.deps.RepoRoot
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("security-health does not validate target kind %s", target.GetKind().String()))
	}

	collector := metrics.Start(metrics.WithEnvironment(h.deps.Environment))
	report, err := h.deps.TargetValidator.ValidateTarget(validation.WithMetrics(ctx, collector), kind, path)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	maturityAssessment, err := buildMaturityAssessment(report, h.deps.MaturitySpec)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build maturity assessment: %w", err))
	}
	execMetrics := collector.Stop()
	responseOptions := []assessment.ValidationResponseOption(nil)
	if report.Passed {
		responseOptions = append(responseOptions, assessment.WithValidationStatus(scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED))
	}
	legacy, err := assessment.BuildValidationResponse(report.Scenario, maturityAssessment, nil, execMetrics, responseOptions...)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build shared target response: %w", err))
	}
	return connect.NewResponse(&scenariovalidationv1.ValidateTargetResponse{
		Target: target, Status: legacy.GetStatus(), Assessment: legacy.GetAssessment(), NativeDetail: legacy.GetNativeDetail(), Metrics: legacy.GetMetrics(),
	}), nil
}

func buildMaturityAssessment(rep validation.Report, spec *assessment.Spec) (*commonv1.MaturityAssessment, error) {
	if spec == nil {
		return nil, nil
	}
	findings := make([]assessment.Finding, 0, len(rep.Findings))
	for _, f := range rep.Findings {
		findings = append(findings, assessment.Finding{
			Code:             f.RuleID,
			Severity:         severityToken(f.Severity),
			Title:            f.Title,
			Message:          f.Description,
			Location:         f.FilePath,
			Remediation:      f.Remediation,
			Phase:            spec.Phase,
			AutofixAvailable: f.FixPreviewable,
			FixClass:         string(f.FixClass),
		})
	}
	return assessment.BuildProtoAssessment(assessment.BuildInput{
		Scenario: rep.Scenario,
		Spec:     *spec,
		Findings: findings,
	})
}

func severityToken(s validation.Severity) string {
	switch s {
	case validation.SeverityError:
		return "SEVERITY_ERROR"
	case validation.SeverityWarning:
		return "SEVERITY_WARNING"
	case validation.SeverityInfo:
		return "SEVERITY_INFO"
	default:
		return "SEVERITY_UNSPECIFIED"
	}
}

// PreviewFix reports deterministic Security Health remediations without
// writing. Today that covers only the low-risk generated-Go API security
// headers middleware shape; other security findings remain manual by design.
func (h *connectHandler) PreviewFix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return h.fix(ctx, req, false)
}

// ApplyFix writes deterministic Security Health remediations selected by
// PreviewFix. Callers must opt into this RPC explicitly.
func (h *connectHandler) ApplyFix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return h.fix(ctx, req, true)
}

func (h *connectHandler) fix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest], apply bool) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	if h.deps.Fixer == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("security validation fixer not wired"))
	}
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("fix request is required"))
	}
	if strings.TrimSpace(req.Msg.GetScenario()) == "" && strings.TrimSpace(req.Msg.GetPath()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario or path is required"))
	}
	var (
		scenario   string
		candidates []validation.SecurityHeaderFixCandidate
		messages   []string
		err        error
	)
	if apply {
		scenario, candidates, messages, err = h.deps.Fixer.ApplyFix(ctx, req.Msg.GetScenario(), req.Msg.GetPath(), req.Msg.GetRuleIds())
	} else {
		scenario, candidates, messages, err = h.deps.Fixer.PreviewFix(ctx, req.Msg.GetScenario(), req.Msg.GetPath(), req.Msg.GetRuleIds())
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(securityFixResponse(scenario, apply, candidates, messages)), nil
}

func securityFixResponse(scenario string, applied bool, candidates []validation.SecurityHeaderFixCandidate, messages []string) *scenariovalidationv1.FixResponse {
	out := &scenariovalidationv1.FixResponse{
		Scenario: scenario,
		Applied:  applied,
		Messages: messages,
	}
	for _, c := range candidates {
		out.Candidates = append(out.Candidates, &scenariovalidationv1.FixCandidate{
			RuleId:      c.RuleID,
			FilePath:    c.FilePath,
			Description: c.Description,
			Before:      c.Before,
			After:       c.After,
			Applied:     c.Applied,
		})
	}
	return out
}
