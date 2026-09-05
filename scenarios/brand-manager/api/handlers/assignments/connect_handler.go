package assignments

import (
	"context"
	"log"

	"brand-manager/internal/assignments"

	"connectrpc.com/connect"

	assignmentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/assignments"
)

// Deps wires the seams the Connect assignments handler needs.
type Deps struct {
	Service assignments.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect-RPC assignments handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ListAssignments(ctx context.Context, req *connect.Request[assignmentsv1.ListAssignmentsRequest]) (*connect.Response[assignmentsv1.ListAssignmentsResponse], error) {
	results, err := h.deps.Service.List(ctx, req.Msg.GetBrandId())
	if err != nil {
		h.deps.Logger.Printf("assignments.ListAssignments: %v", err)
		return nil, assignments.ToConnectError(err)
	}
	resp := &assignmentsv1.ListAssignmentsResponse{Assignments: make([]*assignmentsv1.Assignment, 0, len(results))}
	for _, a := range results {
		resp.Assignments = append(resp.Assignments, assignmentToProto(a))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) AssignBrand(ctx context.Context, req *connect.Request[assignmentsv1.AssignBrandRequest]) (*connect.Response[assignmentsv1.AssignBrandResponse], error) {
	assigned, err := h.deps.Service.Assign(ctx, assignments.AssignInput{
		BrandID:      req.Msg.GetBrandId(),
		ScenarioName: req.Msg.GetScenarioName(),
		Elements:     req.Msg.GetElements(),
	})
	if err != nil {
		return nil, h.translate("assignments.AssignBrand", err)
	}
	return connect.NewResponse(&assignmentsv1.AssignBrandResponse{Assignment: assignmentToProto(assigned)}), nil
}

func (h *connectHandler) GetScenarioStatus(ctx context.Context, req *connect.Request[assignmentsv1.GetScenarioStatusRequest]) (*connect.Response[assignmentsv1.GetScenarioStatusResponse], error) {
	status, err := h.deps.Service.ScenarioStatus(ctx, req.Msg.GetScenarioName())
	if err != nil {
		return nil, h.translate("assignments.GetScenarioStatus", err)
	}
	return connect.NewResponse(&assignmentsv1.GetScenarioStatusResponse{Status: statusToProto(status)}), nil
}

func (h *connectHandler) UnassignScenario(ctx context.Context, req *connect.Request[assignmentsv1.UnassignScenarioRequest]) (*connect.Response[assignmentsv1.UnassignScenarioResponse], error) {
	if err := h.deps.Service.Unassign(ctx, req.Msg.GetScenarioName()); err != nil {
		return nil, h.translate("assignments.UnassignScenario", err)
	}
	return connect.NewResponse(&assignmentsv1.UnassignScenarioResponse{}), nil
}

// translate maps a domain error to a Connect error, logging only genuine
// internal failures (never the client-fault 4xx-equivalent codes).
func (h *connectHandler) translate(op string, err error) error {
	connectErr := assignments.ToConnectError(err)
	if connect.CodeOf(connectErr) == connect.CodeInternal {
		h.deps.Logger.Printf("%s: %v", op, err)
	}
	return connectErr
}
