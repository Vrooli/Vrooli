package landing

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	"landing-page-business-suite-api/internal/experimentation"
	"landing-page-business-suite-api/internal/presentation"
)

// PresentationAdminHandler is mounted only behind administrator middleware.
// Preview shares the finite public renderer contract, but not its public route.
type PresentationAdminHandler struct{ store *experimentation.ConfigStore }

func NewPresentationAdminHandler(store *experimentation.ConfigStore) PresentationAdminHandler {
	return PresentationAdminHandler{store: store}
}

func (h PresentationAdminHandler) GetPresentation(ctx context.Context, request *connect.Request[lpbsv1.GetPresentationRequest]) (*connect.Response[lpbsv1.PresentationEditorResponse], error) {
	state, err := h.store.GetPresentationState(ctx, request.Msg.GetVariantSlug())
	if err != nil {
		return nil, presentationConnectError(err)
	}
	revision := request.Msg.GetRevision()
	if revision == "" {
		revision = state.DraftRevision
	}
	return h.editorResponse(ctx, request.Msg.GetVariantSlug(), revision, state)
}

func (h PresentationAdminHandler) SaveDraft(ctx context.Context, request *connect.Request[lpbsv1.SavePresentationDraftRequest]) (*connect.Response[lpbsv1.PresentationEditorResponse], error) {
	document, err := PresentationDocumentFromProto(request.Msg.GetDocument())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid presentation document: %w", err))
	}
	state, err := h.store.SavePresentationDraft(ctx, request.Msg.GetVariantSlug(), document, request.Msg.GetExpectedGeneration())
	if err != nil {
		return nil, presentationConnectError(err)
	}
	return h.editorResponse(ctx, request.Msg.GetVariantSlug(), state.DraftRevision, state)
}

func (h PresentationAdminHandler) Publish(ctx context.Context, request *connect.Request[lpbsv1.ActivatePresentationRequest]) (*connect.Response[lpbsv1.PresentationEditorResponse], error) {
	state, err := h.store.PublishPresentation(ctx, request.Msg.GetVariantSlug(), request.Msg.GetRevision(), request.Msg.GetExpectedGeneration())
	if err != nil {
		return nil, presentationConnectError(err)
	}
	return h.editorResponse(ctx, request.Msg.GetVariantSlug(), state.ActiveRevision, state)
}

func (h PresentationAdminHandler) Rollback(ctx context.Context, request *connect.Request[lpbsv1.ActivatePresentationRequest]) (*connect.Response[lpbsv1.PresentationEditorResponse], error) {
	state, err := h.store.RollbackPresentation(ctx, request.Msg.GetVariantSlug(), request.Msg.GetRevision(), request.Msg.GetExpectedGeneration())
	if err != nil {
		return nil, presentationConnectError(err)
	}
	return h.editorResponse(ctx, request.Msg.GetVariantSlug(), state.ActiveRevision, state)
}

func (h PresentationAdminHandler) Preview(ctx context.Context, request *connect.Request[lpbsv1.PreviewPresentationRequest]) (*connect.Response[lpbsv1.PreviewPresentationResponse], error) {
	loaded, err := h.store.GetPresentationRevision(ctx, request.Msg.GetVariantSlug(), request.Msg.GetRevision())
	if err != nil {
		return nil, presentationConnectError(err)
	}
	value, err := presentation.Resolve(loaded.Document, presentation.ResolveRequest{
		Route: request.Msg.GetRoute(), Locale: request.Msg.GetLocale(), Variant: request.Msg.GetVariantSlug(),
		PreviewRevision: loaded.Revision, PreviewAuthorized: true,
	})
	if err != nil {
		return nil, presentationConnectError(err)
	}
	wire, err := ResolvedPresentationProto(value)
	if err != nil {
		return nil, presentationConnectError(err)
	}
	response := connect.NewResponse(&lpbsv1.PreviewPresentationResponse{Presentation: wire})
	response.Header().Set("Cache-Control", "private, no-store")
	response.Header().Set("X-Robots-Tag", "noindex, nofollow")
	return response, nil
}

func (h PresentationAdminHandler) editorResponse(ctx context.Context, variant, revision string, state *experimentation.PresentationState) (*connect.Response[lpbsv1.PresentationEditorResponse], error) {
	message := &lpbsv1.PresentationEditorResponse{VariantSlug: variant, Revision: revision, State: &lpbsv1.PresentationRevisionState{
		Generation: state.Generation, DraftRevision: state.DraftRevision, ActiveRevision: state.ActiveRevision, PublishedRevisions: append([]string{}, state.PublishedRevisions...),
	}}
	if revision != "" {
		loaded, err := h.store.GetPresentationRevision(ctx, variant, revision)
		if err != nil {
			return nil, presentationConnectError(err)
		}
		document, err := PresentationDocumentProto(loaded.Document)
		if err != nil {
			return nil, presentationConnectError(err)
		}
		message.Document = document
	}
	response := connect.NewResponse(message)
	response.Header().Set("Cache-Control", "private, no-store")
	return response, nil
}

func presentationConnectError(err error) error {
	var invalid *presentation.ValidationError
	switch {
	case errors.Is(err, context.Canceled):
		return connect.NewError(connect.CodeCanceled, errors.New("presentation request canceled"))
	case errors.Is(err, context.DeadlineExceeded):
		return connect.NewError(connect.CodeDeadlineExceeded, errors.New("presentation request timed out"))
	case errors.Is(err, experimentation.ErrPresentationConflict):
		return connect.NewError(connect.CodeAborted, errors.New("presentation changed; reload before saving or publishing"))
	case errors.Is(err, experimentation.ErrPresentationNotFound), errors.Is(err, presentation.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, errors.New("presentation not found"))
	case errors.Is(err, experimentation.ErrPresentationUnqualified):
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("presentation references are not qualified for publication"))
	case errors.Is(err, experimentation.ErrPresentationInvalid):
		return connect.NewError(connect.CodeInvalidArgument, errors.New("invalid presentation variant or revision"))
	case errors.Is(err, presentation.ErrPreviewUnauthorized):
		return connect.NewError(connect.CodePermissionDenied, errors.New("presentation preview requires administrator access"))
	case errors.As(err, &invalid):
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	default:
		// Filesystem and owner errors can contain private paths or evidence.
		return connect.NewError(connect.CodeUnavailable, errors.New("presentation storage or resolution is unavailable"))
	}
}

func RegisterPresentationAdminRoutes(router *mux.Router, store *experimentation.ConfigStore, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	if requireAdmin == nil {
		panic("presentation administration requires authentication middleware")
	}
	_, handler := lpbsconnect.NewProductPresentationAdminServiceHandler(NewPresentationAdminHandler(store))
	protected := requireAdmin(http.HandlerFunc(handler.ServeHTTP))
	for _, procedure := range []string{
		lpbsconnect.ProductPresentationAdminServiceGetPresentationProcedure,
		lpbsconnect.ProductPresentationAdminServiceSaveDraftProcedure,
		lpbsconnect.ProductPresentationAdminServicePreviewProcedure,
		lpbsconnect.ProductPresentationAdminServicePublishProcedure,
		lpbsconnect.ProductPresentationAdminServiceRollbackProcedure,
	} {
		router.Handle(procedure, protected).Methods(http.MethodPost)
	}
}

var _ lpbsconnect.ProductPresentationAdminServiceHandler = PresentationAdminHandler{}
