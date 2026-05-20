// Package validation hosts the Connect-RPC handler for cli-health's
// ValidationService. The handler delegates real work to the
// manifestvalidation service and maps its domain types onto the proto
// Finding/Severity shape.
package validation

import (
	"context"
	"fmt"
	"log"
	"strings"

	"connectrpc.com/connect"

	"cli-health/internal/services/manifestvalidation"

	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli-health/v1/validation"
)

// Deps wires the seams the Connect validation handler needs.
type Deps struct {
	Logger    *log.Logger
	Validator Validator
	// ReservedNames are non-scenario CLI names that should be rejected with
	// InvalidArgument rather than fed to the scenario validator. Sourced from
	// the aisearch ExternalCLIs config so vrooli (and any future ExternalCLI)
	// can't be misinterpreted as a scenario.
	ReservedNames []string
}

// Validator is the slice of manifestvalidation.Service the handler exercises.
// Stays an interface so handler tests can stub it without spinning up buf.
type Validator interface {
	ValidateScenario(ctx context.Context, scenario string) (manifestvalidation.Report, error)
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler returns a handler that satisfies the generated
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
	scenario := req.Msg.GetScenario()
	for _, reserved := range h.deps.ReservedNames {
		if strings.EqualFold(strings.TrimSpace(reserved), strings.TrimSpace(scenario)) {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%s is not a scenario; only scenario CLIs are validated", scenario))
		}
	}
	report, err := h.deps.Validator.ValidateScenario(ctx, scenario)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	resp := &validationv1.ValidateScenarioResponse{
		Scenario: report.Scenario,
		Passed:   report.Passed,
		Findings: findingsToProto(report.Findings),
		Summary: &validationv1.Summary{
			Errors:   int32(report.Summary.Errors),
			Warnings: int32(report.Summary.Warnings),
			Infos:    int32(report.Summary.Infos),
		},
	}
	return connect.NewResponse(resp), nil
}

func findingsToProto(in []manifestvalidation.Finding) []*validationv1.Finding {
	out := make([]*validationv1.Finding, 0, len(in))
	for _, f := range in {
		out = append(out, &validationv1.Finding{
			Severity:   severityToProto(f.Severity),
			Code:       f.Code,
			Location:   f.Location,
			Message:    f.Message,
			Suggestion: f.Suggestion,
		})
	}
	return out
}

func severityToProto(s manifestvalidation.Severity) validationv1.Severity {
	switch s {
	case manifestvalidation.SeverityError:
		return validationv1.Severity_SEVERITY_ERROR
	case manifestvalidation.SeverityWarning:
		return validationv1.Severity_SEVERITY_WARNING
	case manifestvalidation.SeverityInfo:
		return validationv1.Severity_SEVERITY_INFO
	default:
		return validationv1.Severity_SEVERITY_UNSPECIFIED
	}
}
