// Package validation hosts the shared ScenarioValidationService handler for
// cli-health. The handler delegates real work to the manifestvalidation service
// and returns the common scenario-validation response shape.
package validation

import (
	"context"
	"fmt"
	"log"
	"strings"

	"connectrpc.com/connect"

	"cli-health/internal/services/manifestvalidation"

	"github.com/vrooli/api-core/metrics"
	"github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// Deps wires the seams the Connect validation handler needs.
type Deps struct {
	Logger          *log.Logger
	Validator       Validator
	TargetValidator TargetValidator
	MaturitySpec    *assessment.Spec
	// ReservedNames are non-scenario CLI names that should be rejected with
	// InvalidArgument rather than fed to the scenario validator. Sourced from
	// the aisearch ExternalCLIs config so vrooli (and any future ExternalCLI)
	// can't be misinterpreted as a scenario.
	ReservedNames []string
	// Environment is the host CaptureEnvironment captured once at module init
	// (os/arch/cpu/mem/present-GPUs). nil is safe — the metrics collector
	// backfills os/arch/num_cpu from the stdlib.
	Environment *commonv1.CaptureEnvironment
}

// Validator is the slice of manifestvalidation.Service the handler exercises.
// Stays an interface so handler tests can stub it without spinning up buf.
type Validator interface {
	ValidateScenario(ctx context.Context, scenario string) (manifestvalidation.Report, error)
}

// TargetValidator is the generalized provider seam. Keeping it optional
// preserves the legacy scenario handler contract while allowing cli-health to
// own the repository project target instead of relying on the generic adapter.
type TargetValidator interface {
	ValidateTarget(ctx context.Context, target manifestvalidation.Target) (manifestvalidation.Report, error)
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler returns a handler that satisfies the generated shared
// ScenarioValidationServiceHandler interface.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	if h.deps.Validator == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("validation.ValidateScenario: validator not wired"))
	}
	scenario := req.Msg.GetScenario()
	for _, reserved := range h.deps.ReservedNames {
		if strings.EqualFold(strings.TrimSpace(reserved), strings.TrimSpace(scenario)) {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%s is not a scenario; only scenario CLIs are validated", scenario))
		}
	}
	// When the caller resolved an explicit scenario path (e.g. Test Genie running
	// deep template validation against a temp scenario outside the repo
	// scenarios/ tree), thread it so the manifest loader reads from that dir.
	validateCtx := manifestvalidation.WithScenarioPath(ctx, req.Msg.GetPath())
	// Opt into the runtime CLI probe when the caller requested execution. The
	// default (false) keeps the static-only contract path unchanged.
	validateCtx = manifestvalidation.WithIncludeExecution(validateCtx, req.Msg.GetIncludeExecution())
	collector := metrics.Start(metrics.WithEnvironment(h.deps.Environment))
	report, err := h.deps.Validator.ValidateScenario(manifestvalidation.WithMetrics(validateCtx, collector), scenario)
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

// ValidateTarget validates the repository project target and scenario targets
// through the same provider-owned report pipeline. Project targets are exempt
// from the reserved-scenario-name guard because "repo" is an ownership
// identity, not a scenario lookup.
func (h *connectHandler) ValidateTarget(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateTargetRequest]) (*connect.Response[scenariovalidationv1.ValidateTargetResponse], error) {
	if h.deps.Validator == nil && h.deps.TargetValidator == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("validation.ValidateTarget: validator not wired"))
	}
	target := req.Msg.GetTarget()
	if target == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("validation target is required"))
	}
	kind := target.GetKind()
	var targetKind manifestvalidation.TargetKind
	switch kind {
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PROJECT:
		targetKind = manifestvalidation.TargetKindProject
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_CONTROL_PLANE:
		targetKind = manifestvalidation.TargetKindControlPlane
	case commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO:
		targetKind = manifestvalidation.TargetKindScenario
		for _, reserved := range h.deps.ReservedNames {
			if strings.EqualFold(strings.TrimSpace(reserved), strings.TrimSpace(target.GetId())) {
				return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%s is not a scenario; only scenario CLIs are validated", target.GetId()))
			}
		}
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("cli-health does not validate target kind %s", kind.String()))
	}
	if targetKind == manifestvalidation.TargetKindScenario && strings.TrimSpace(target.GetId()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("scenario target id is required"))
	}
	validateCtx := manifestvalidation.WithScenarioPath(ctx, req.Msg.GetPath())
	validateCtx = manifestvalidation.WithIncludeExecution(validateCtx, req.Msg.GetIncludeExecution())
	collector := metrics.Start(metrics.WithEnvironment(h.deps.Environment))
	var report manifestvalidation.Report
	var err error
	if h.deps.TargetValidator != nil {
		report, err = h.deps.TargetValidator.ValidateTarget(manifestvalidation.WithMetrics(validateCtx, collector), manifestvalidation.Target{
			Kind: targetKind,
			ID:   target.GetId(),
			Root: target.GetRoot(),
		})
	} else {
		id := target.GetId()
		if targetKind == manifestvalidation.TargetKindProject {
			id = manifestvalidation.ProjectTargetID
		}
		report, err = h.deps.Validator.ValidateScenario(manifestvalidation.WithMetrics(validateCtx, collector), id)
	}
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
	return connect.NewResponse(&scenariovalidationv1.ValidateTargetResponse{
		Target:     target,
		Status:     assessment.DeriveValidationStatus(maturityAssessment),
		Assessment: maturityAssessment,
		Metrics:    execMetrics,
	}), nil
}

func buildMaturityAssessment(rep manifestvalidation.Report, spec *assessment.Spec) (*commonv1.MaturityAssessment, error) {
	if spec == nil {
		return nil, nil
	}
	findings := make([]assessment.Finding, 0, len(rep.Findings))
	for _, f := range rep.Findings {
		findings = append(findings, assessment.Finding{
			Code:        f.Code,
			Severity:    severityToken(f.Severity),
			Message:     f.Message,
			Location:    f.Location,
			Remediation: f.Suggestion,
			Phase:       spec.Phase,
		})
	}
	return assessment.BuildProtoAssessment(assessment.BuildInput{
		Scenario: rep.Scenario,
		Spec:     *spec,
		Findings: findings,
	})
}

func severityToken(s manifestvalidation.Severity) string {
	switch s {
	case manifestvalidation.SeverityError:
		return "SEVERITY_ERROR"
	case manifestvalidation.SeverityWarning:
		return "SEVERITY_WARNING"
	case manifestvalidation.SeverityInfo:
		return "SEVERITY_INFO"
	default:
		return "SEVERITY_UNSPECIFIED"
	}
}
