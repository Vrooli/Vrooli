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
	Logger       *log.Logger
	Validator    Validator
	MaturitySpec *assessment.Spec
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
	collector := metrics.Start(metrics.WithEnvironment(h.deps.Environment))
	report, err := h.deps.Validator.ValidateScenario(manifestvalidation.WithMetrics(ctx, collector), scenario)
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
