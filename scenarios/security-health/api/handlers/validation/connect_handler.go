// Package validation hosts the shared ScenarioValidationService handler for
// security-health. The handler delegates real work to the internal validation
// Service and returns the common scenario-validation response shape.
package validation

import (
	"context"
	"fmt"
	"log"

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

// Deps wires the seams the Connect validation handler needs.
type Deps struct {
	Logger       *log.Logger
	Validator    Validator
	MaturitySpec *assessment.Spec
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

func buildMaturityAssessment(rep validation.Report, spec *assessment.Spec) (*commonv1.MaturityAssessment, error) {
	if spec == nil {
		return nil, nil
	}
	findings := make([]assessment.Finding, 0, len(rep.Findings))
	for _, f := range rep.Findings {
		findings = append(findings, assessment.Finding{
			Code:        f.RuleID,
			Severity:    severityToken(f.Severity),
			Title:       f.Title,
			Message:     f.Description,
			Location:    f.FilePath,
			Remediation: f.Remediation,
			Phase:       spec.Phase,
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
