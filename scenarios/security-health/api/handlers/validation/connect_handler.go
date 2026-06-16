// Package validation hosts the Connect-RPC handler for security-health's
// ValidationService. The handler delegates real work to the internal
// validation Service and maps its domain types onto the proto Finding/Severity
// shape. It is the producer half of the test-genie `security` phase: the CLI
// renders this RPC's output as --json, and test-genie maps each Finding to
// FINDING_SOURCE_SECURITY.
package validation

import (
	"context"
	"fmt"
	"log"

	"connectrpc.com/connect"

	"security-health/internal/validation"

	"github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1/validation"
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
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler returns a handler satisfying the generated
// ValidationServiceHandler interface.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ValidateScenario(ctx context.Context, req *connect.Request[validationv1.ValidateScenarioRequest]) (*connect.Response[validationv1.ValidateScenarioResponse], error) {
	if h.deps.Validator == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("validation.ValidateScenario: validator not wired"))
	}
	report, err := h.deps.Validator.ValidateScenario(ctx, req.Msg.GetScenario())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	maturityAssessment, err := buildMaturityAssessment(report, h.deps.MaturitySpec)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build maturity assessment: %w", err))
	}
	resp := &validationv1.ValidateScenarioResponse{
		Scenario:        report.Scenario,
		Passed:          report.Passed,
		Findings:        findingsToProto(report.Findings),
		SkippedScanners: report.SkippedScanners,
		Assessment:      maturityAssessment,
		Summary: &validationv1.Summary{
			Errors:   int32(report.Summary.Errors),
			Warnings: int32(report.Summary.Warnings),
			Infos:    int32(report.Summary.Infos),
		},
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
			Severity:    severityToProto(f.Severity).String(),
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

func findingsToProto(in []validation.Finding) []*validationv1.Finding {
	out := make([]*validationv1.Finding, 0, len(in))
	for _, f := range in {
		out = append(out, &validationv1.Finding{
			RuleId:      f.RuleID,
			Severity:    severityToProto(f.Severity),
			Title:       f.Title,
			Description: f.Description,
			Remediation: f.Remediation,
			FilePath:    f.FilePath,
			Scanner:     f.Scanner,
		})
	}
	return out
}

func severityToProto(s validation.Severity) validationv1.Severity {
	switch s {
	case validation.SeverityError:
		return validationv1.Severity_SEVERITY_ERROR
	case validation.SeverityWarning:
		return validationv1.Severity_SEVERITY_WARNING
	case validation.SeverityInfo:
		return validationv1.Severity_SEVERITY_INFO
	default:
		return validationv1.Severity_SEVERITY_UNSPECIFIED
	}
}
