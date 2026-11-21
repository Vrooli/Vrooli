package guidance

import (
	"context"
	"errors"
	"log"

	guidancesvc "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/guidance"

	"connectrpc.com/connect"
	guidancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/guidance"
)

type Deps struct {
	Service *guidancesvc.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	if d.Service == nil {
		d.Service = guidancesvc.NewService(nil)
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) NextGate(ctx context.Context, req *connect.Request[guidancev1.NextGateRequest]) (*connect.Response[guidancev1.NextGateResponse], error) {
	result, err := h.deps.Service.NextGate(ctx, req.Msg.Scenario)
	if err != nil {
		h.deps.Logger.Printf("guidance.NextGate(%q): %v", req.Msg.Scenario, err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("get next orientation gate"))
	}
	return connect.NewResponse(resultToProto(result)), nil
}

func resultToProto(result guidancesvc.NextGateResult) *guidancev1.NextGateResponse {
	out := &guidancev1.NextGateResponse{
		Scenario:         result.Scenario,
		Finalized:        result.Finalized,
		Complete:         result.Complete,
		FinalizeRequired: result.FinalizeRequired,
		Completed:        result.Completed,
		Required:         result.Required,
		Message:          result.Message,
	}
	if result.Gate != nil {
		out.Gate = gateToProto(*result.Gate)
	}
	return out
}

func gateToProto(gate guidancesvc.Gate) *guidancev1.OrientationGate {
	out := &guidancev1.OrientationGate{
		Id:          gate.ID,
		Title:       gate.Title,
		Description: gate.Description,
		Required:    gate.Required,
		Complete:    gate.Complete,
		Docs:        gate.Docs,
		Remediation: gate.Remediation,
	}
	for _, check := range gate.Checks {
		out.Checks = append(out.Checks, &guidancev1.OrientationCheck{
			Kind:     check.Kind,
			Label:    check.Label,
			Passed:   check.Passed,
			Skipped:  check.Skipped,
			Optional: check.Optional,
			Message:  check.Message,
		})
	}
	return out
}
