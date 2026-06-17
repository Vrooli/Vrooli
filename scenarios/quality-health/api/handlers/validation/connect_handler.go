// Package validation exposes quality-health through the shared
// ScenarioValidationService contract consumed by Test Genie.
package validation

import (
	"context"
	"errors"
	"fmt"
	"log"

	"connectrpc.com/connect"

	auditH "quality-health/handlers/audit"
	internalaudit "quality-health/internal/audit"

	"github.com/vrooli/maturity-go/assessment"
	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/quality-health/v1/audit"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// Auditor is the subset of quality-health's audit service needed by the
// shared validation adapter.
type Auditor interface {
	Audit(context.Context, internalaudit.Request) (internalaudit.Response, error)
}

// Deps wires the shared validation handler.
type Deps struct {
	Auditor      Auditor
	Logger       *log.Logger
	MaturitySpec *assessment.Spec
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
	if h.deps.Auditor == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("quality validation auditor not wired"))
	}
	if req.Msg.GetScenario() == "" && req.Msg.GetPath() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario or path is required"))
	}
	report, err := h.deps.Auditor.Audit(ctx, internalaudit.Request{
		Scenario:                req.Msg.GetScenario(),
		Path:                    req.Msg.GetPath(),
		IncludeCommandExecution: req.Msg.GetIncludeExecution(),
		UseCache:                true,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	native, err := auditResponseToProto(report, h.deps.MaturitySpec)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build quality native detail: %w", err))
	}
	resp, err := assessment.BuildValidationResponse(native.GetScenario(), native.GetAssessment(), native, statusOverride(native)...)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build shared validation response: %w", err))
	}
	return connect.NewResponse(resp), nil
}

func auditResponseToProto(in internalaudit.Response, spec *assessment.Spec) (*auditv1.AuditQualityResponse, error) {
	return auditH.ResponseToProto(in, spec)
}

func statusOverride(resp *auditv1.AuditQualityResponse) []assessment.ValidationResponseOption {
	switch resp.GetStatus() {
	case "degraded":
		return []assessment.ValidationResponseOption{assessment.WithValidationStatus(scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_DEGRADED)}
	case "error":
		return []assessment.ValidationResponseOption{assessment.WithValidationStatus(scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_ERROR)}
	default:
		return nil
	}
}
