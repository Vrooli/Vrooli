package documents

import (
	"context"
	"errors"
	"time"

	domain "persona/internal/documents"

	"connectrpc.com/connect"
	documentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/documents"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type connectHandler struct{ service domain.Service }

func NewConnectHandler(service domain.Service) *connectHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) BindDocument(ctx context.Context, req *connect.Request[documentsv1.BindDocumentRequest]) (*connect.Response[documentsv1.BindDocumentResponse], error) {
	b, err := h.service.Bind(ctx, domain.BindingInput{PersonaID: req.Msg.GetPersonaId(), DocumentID: req.Msg.GetDocumentId(), DocumentKind: req.Msg.GetDocumentKind(), ValidUntil: timeFromProto(req.Msg.GetValidUntil())})
	if err != nil {
		return nil, documentError(err)
	}
	return connect.NewResponse(&documentsv1.BindDocumentResponse{Binding: toProto(b)}), nil
}

func (h *connectHandler) ListBindings(ctx context.Context, req *connect.Request[documentsv1.ListBindingsRequest]) (*connect.Response[documentsv1.ListBindingsResponse], error) {
	items, err := h.service.List(ctx, req.Msg.GetPersonaId())
	if err != nil {
		return nil, documentError(err)
	}
	out := &documentsv1.ListBindingsResponse{Bindings: make([]*documentsv1.DocumentBinding, 0, len(items))}
	for _, item := range items {
		out.Bindings = append(out.Bindings, toProto(item))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ReleaseIntoHandoff(ctx context.Context, req *connect.Request[documentsv1.ReleaseIntoHandoffRequest]) (*connect.Response[documentsv1.ReleaseIntoHandoffResponse], error) {
	release, err := h.service.ReleaseIntoHandoff(ctx, domain.ReleaseInput{PersonaID: req.Msg.GetPersonaId(), DocumentID: req.Msg.GetDocumentId(), HandoffID: req.Msg.GetHandoffId()})
	if err != nil {
		return nil, documentError(err)
	}
	return connect.NewResponse(&documentsv1.ReleaseIntoHandoffResponse{ReleaseId: release.ID, HandoffId: release.HandoffID, DocumentId: release.DocumentID, ReleasedAt: timestamppb.New(release.ReleasedAt)}), nil
}

func documentError(err error) error {
	code := connect.CodeInternal
	if errors.Is(err, domain.ErrMissingPersona) || errors.Is(err, domain.ErrMissingDocument) || errors.Is(err, domain.ErrMissingHandoff) {
		code = connect.CodeInvalidArgument
	}
	if errors.Is(err, domain.ErrDocumentAuthorityUnavailable) {
		code = connect.CodeUnavailable
	}
	if errors.Is(err, domain.ErrHandoffMismatch) {
		code = connect.CodePermissionDenied
	}
	if errors.Is(err, domain.ErrHandoffClosed) {
		code = connect.CodeFailedPrecondition
	}
	return connect.NewError(code, err)
}

func timeFromProto(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}

func toProto(b domain.Binding) *documentsv1.DocumentBinding {
	out := &documentsv1.DocumentBinding{Id: b.ID, PersonaId: b.PersonaID, DocumentId: b.DocumentID, DocumentKind: b.DocumentKind, CreatedAt: timestamppb.New(b.CreatedAt)}
	if !b.ValidUntil.IsZero() {
		out.ValidUntil = timestamppb.New(b.ValidUntil)
	}
	return out
}
