package authoring

import (
	"context"
	"log"

	internalauthoring "plan-manager/internal/authoring"

	"connectrpc.com/connect"

	authoringv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/authoring"
)

// Deps wires the seams the Connect authoring handler needs.
type Deps struct {
	Service internalauthoring.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the AuthoringService Connect handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) StartSession(ctx context.Context, req *connect.Request[authoringv1.StartSessionRequest]) (*connect.Response[authoringv1.StartSessionResponse], error) {
	sess, err := h.deps.Service.StartSession(ctx, req.Msg.GetTitle(), req.Msg.GetSlug(), req.Msg.GetTemplateId())
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	return connect.NewResponse(&authoringv1.StartSessionResponse{Session: sessionToProto(sess)}), nil
}

func (h *connectHandler) GetSection(ctx context.Context, req *connect.Request[authoringv1.GetSectionRequest]) (*connect.Response[authoringv1.GetSectionResponse], error) {
	sec, err := h.deps.Service.GetSection(ctx, req.Msg.GetSessionId(), internalauthoring.SectionKey(req.Msg.GetSectionKey()))
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	return connect.NewResponse(&authoringv1.GetSectionResponse{Section: sectionToProto(sec)}), nil
}

func (h *connectHandler) SubmitSection(ctx context.Context, req *connect.Request[authoringv1.SubmitSectionRequest]) (*connect.Response[authoringv1.SubmitSectionResponse], error) {
	sess, violations, err := h.deps.Service.SubmitSection(ctx, req.Msg.GetSessionId(), internalauthoring.SectionKey(req.Msg.GetSectionKey()), req.Msg.GetContent())
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	return connect.NewResponse(&authoringv1.SubmitSectionResponse{
		Session:    sessionToProto(sess),
		Violations: violationsToProto(violations),
	}), nil
}

func (h *connectHandler) Next(ctx context.Context, req *connect.Request[authoringv1.NextRequest]) (*connect.Response[authoringv1.NextResponse], error) {
	sec, complete, err := h.deps.Service.Next(ctx, req.Msg.GetSessionId())
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	resp := &authoringv1.NextResponse{Complete: complete}
	if !complete {
		resp.Section = sectionToProto(sec)
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ValidateStructure(ctx context.Context, req *connect.Request[authoringv1.ValidateStructureRequest]) (*connect.Response[authoringv1.ValidateStructureResponse], error) {
	valid, violations, err := h.deps.Service.ValidateStructure(ctx, req.Msg.GetSessionId())
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	return connect.NewResponse(&authoringv1.ValidateStructureResponse{
		Valid:      valid,
		Violations: violationsToProto(violations),
	}), nil
}

func (h *connectHandler) Autofill(ctx context.Context, req *connect.Request[authoringv1.AutofillRequest]) (*connect.Response[authoringv1.AutofillResponse], error) {
	sess, results, err := h.deps.Service.Autofill(ctx, req.Msg.GetSessionId(), autofillSourcesFromProto(req.Msg.GetSources()))
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	return connect.NewResponse(&authoringv1.AutofillResponse{
		Session: sessionToProto(sess),
		Results: autofillResultsToProto(results),
	}), nil
}

func (h *connectHandler) Finalize(ctx context.Context, req *connect.Request[authoringv1.FinalizeRequest]) (*connect.Response[authoringv1.FinalizeResponse], error) {
	plan, err := h.deps.Service.Finalize(ctx, req.Msg.GetSessionId())
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	return connect.NewResponse(&authoringv1.FinalizeResponse{Plan: planToProto(plan)}), nil
}
