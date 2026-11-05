package handoffrules

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	handoffrulesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/handoffrules"

	hrdomain "web-console/internal/handoffrules"
)

// Deps wires the seams the Connect capture-rule handler needs.
type Deps struct {
	Service Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect-RPC handler implementing
// HandoffRulesServiceHandler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

// mapError maps the domain sentinel onto a Connect code. A blank name, a
// blank pattern, and an unknown source are caller mistakes: the operator
// authoring a rule needs to see which field is wrong, not a server error.
func (h *connectHandler) mapError(op string, err error) error {
	if errors.Is(err, hrdomain.ErrInvalidRule) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	h.deps.Logger.Printf("handoffrules.%s: %v", op, err)
	return connect.NewError(connect.CodeInternal, err)
}

func (h *connectHandler) ListRules(ctx context.Context, _ *connect.Request[handoffrulesv1.ListRulesRequest]) (*connect.Response[handoffrulesv1.ListRulesResponse], error) {
	rules, err := h.deps.Service.List(ctx)
	if err != nil {
		return nil, h.mapError("ListRules", err)
	}
	out := make([]*handoffrulesv1.Rule, 0, len(rules))
	for _, r := range rules {
		out = append(out, ruleToProto(r))
	}
	return connect.NewResponse(&handoffrulesv1.ListRulesResponse{Rules: out}), nil
}

func (h *connectHandler) UpsertRule(ctx context.Context, req *connect.Request[handoffrulesv1.UpsertRuleRequest]) (*connect.Response[handoffrulesv1.UpsertRuleResponse], error) {
	in := UpsertRequest{
		ID:        req.Msg.GetId(),
		Name:      req.Msg.GetName(),
		Enabled:   req.Msg.GetEnabled(),
		Source:    req.Msg.GetSource(),
		Pattern:   req.Msg.GetPattern(),
		Surfaces:  req.Msg.GetSurfaces(),
		SortOrder: int(req.Msg.GetSortOrder()),
	}
	r, err := h.deps.Service.Upsert(ctx, in)
	if err != nil {
		return nil, h.mapError("UpsertRule", err)
	}
	return connect.NewResponse(&handoffrulesv1.UpsertRuleResponse{Rule: ruleToProto(r)}), nil
}

func (h *connectHandler) DeleteRule(ctx context.Context, req *connect.Request[handoffrulesv1.DeleteRuleRequest]) (*connect.Response[handoffrulesv1.DeleteRuleResponse], error) {
	if _, err := h.deps.Service.Delete(ctx, req.Msg.GetId()); err != nil {
		return nil, h.mapError("DeleteRule", err)
	}
	return connect.NewResponse(&handoffrulesv1.DeleteRuleResponse{}), nil
}

func ruleToProto(r Rule) *handoffrulesv1.Rule {
	return &handoffrulesv1.Rule{
		Id:        r.ID,
		Name:      r.Name,
		Enabled:   r.Enabled,
		Source:    r.Source,
		Pattern:   r.Pattern,
		Surfaces:  r.Surfaces,
		SortOrder: int32(r.SortOrder),
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}
