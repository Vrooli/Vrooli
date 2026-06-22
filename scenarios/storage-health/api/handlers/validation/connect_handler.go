// Package validation hosts the shared ScenarioValidationService handler for
// storage-health. The handler delegates real work to the internal validation
// Service and returns the common scenario-validation response shape that
// test-genie's delegated `storage` phase consumes.
package validation

import (
	"context"
	"fmt"
	"log"

	"connectrpc.com/connect"

	"storage-health/internal/autofix"
	"storage-health/internal/validation"

	"github.com/vrooli/api-core/metrics"
	"github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// Validator is the slice of internal/validation.Service the handler exercises.
// An interface so handler tests can stub it without running real analyzers.
type Validator interface {
	ValidateScenario(ctx context.Context, scenario string) (validation.Report, error)
}

// Deps wires the seams the Connect validation handler needs.
type Deps struct {
	Logger       *log.Logger
	Validator    Validator
	MaturitySpec *assessment.Spec
	// RepoRoot is the resolved repository root. PreviewFix/ApplyFix resolve a
	// request's scenario to scenarios/<scenario> beneath it so the deterministic
	// autofix registry operates on the right tree. Empty disables the Fix RPCs.
	RepoRoot string
	// Environment is the host CaptureEnvironment captured once at module init.
	// nil is safe — the metrics collector backfills os/arch/num_cpu.
	Environment *commonv1.CaptureEnvironment
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the shared validation handler.
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
		return nil, fmt.Errorf("maturity spec is required")
	}
	findings := make([]assessment.Finding, 0, len(rep.Findings))
	for _, f := range rep.Findings {
		// A finding is auto-fixable when the autofix registry has a fixer for its
		// Code. The registry is the single source of truth for what storage-health
		// can deterministically remediate, so the assessment's AutofixableCount
		// reflects exactly the registry's coverage.
		findings = append(findings, assessment.Finding{
			Code:             f.Code,
			Severity:         f.Severity.Token(),
			Title:            f.Title,
			Message:          f.Message,
			Location:         f.Location,
			Remediation:      f.Remediation,
			Phase:            spec.Phase,
			AutofixAvailable: f.AutofixAvailable || autofix.CoveredCodes[f.Code],
		})
	}
	return assessment.BuildProtoAssessment(assessment.BuildInput{
		Scenario: rep.Scenario,
		Spec:     *spec,
		Findings: findings,
	})
}
