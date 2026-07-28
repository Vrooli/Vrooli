// Package artifacts mounts the draft-lifecycle Connect surface.
package artifacts

import (
	"context"

	"connectrpc.com/connect"
	internalartifacts "content-desk/internal/artifacts"
	"content-desk/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	artifactsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/artifacts"
	artifactsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/artifacts/artifacts_v1connect"
)

type handler struct{ repo internalartifacts.Repository }

var _ artifactsconnect.ArtifactsServiceHandler = handler{}

func (h handler) ListDrafts(ctx context.Context, _ *connect.Request[artifactsv1.ListDraftsRequest]) (*connect.Response[artifactsv1.ListDraftsResponse], error) {
	drafts, err := h.repo.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &artifactsv1.ListDraftsResponse{}
	for _, draft := range drafts {
		response.Drafts = append(response.Drafts, draftMessage(draft))
	}
	return connect.NewResponse(response), nil
}

func (h handler) CreateDraft(ctx context.Context, request *connect.Request[artifactsv1.CreateDraftRequest]) (*connect.Response[artifactsv1.CreateDraftResponse], error) {
	draft, err := h.repo.Create(ctx, internalartifacts.Draft{CampaignID: request.Msg.CampaignId, PostTypeID: request.Msg.PostTypeId, Body: request.Msg.Body, Channel: request.Msg.Channel, Format: request.Msg.Format, Lane: request.Msg.Lane, SKU: request.Msg.Sku})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&artifactsv1.CreateDraftResponse{Draft: draftMessage(draft)}), nil
}

func (h handler) UpdateDraftBody(ctx context.Context, request *connect.Request[artifactsv1.UpdateDraftBodyRequest]) (*connect.Response[artifactsv1.UpdateDraftBodyResponse], error) {
	draft, err := h.repo.UpdateBody(ctx, request.Msg.Id, request.Msg.Body)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&artifactsv1.UpdateDraftBodyResponse{Draft: draftMessage(draft)}), nil
}

func (h handler) PublishDraft(ctx context.Context, request *connect.Request[artifactsv1.PublishDraftRequest]) (*connect.Response[artifactsv1.PublishDraftResponse], error) {
	draft, recordID, err := h.repo.Publish(ctx, request.Msg.Id, internalartifacts.PublishInput{Audience: request.Msg.Audience, PublishedURL: request.Msg.PublishedUrl, PlatformPostID: request.Msg.PlatformPostId, SeriesID: request.Msg.SeriesId, PriorPublishID: request.Msg.PriorPublishId})
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&artifactsv1.PublishDraftResponse{Draft: draftMessage(draft), PublishRecordId: recordID}), nil
}

func (h handler) TransitionDraft(ctx context.Context, request *connect.Request[artifactsv1.TransitionDraftRequest]) (*connect.Response[artifactsv1.TransitionDraftResponse], error) {
	draft, err := h.repo.Transition(ctx, request.Msg.Id, internalartifacts.DraftEvent(request.Msg.Event))
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&artifactsv1.TransitionDraftResponse{Draft: draftMessage(draft)}), nil
}

func (h handler) ApproveDraft(ctx context.Context, request *connect.Request[artifactsv1.ApproveDraftRequest]) (*connect.Response[artifactsv1.ApproveDraftResponse], error) {
	draft, err := h.repo.Approve(ctx, request.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&artifactsv1.ApproveDraftResponse{Draft: draftMessage(draft)}), nil
}

func draftMessage(draft internalartifacts.Draft) *artifactsv1.Draft {
	return &artifactsv1.Draft{Id: draft.ID, CampaignId: draft.CampaignID, Status: string(draft.Status), PostTypeId: draft.PostTypeID, Body: draft.Body, Channel: draft.Channel, Format: draft.Format, Lane: draft.Lane, Sku: draft.SKU}
}

func Module(db *database.RoutedDB) module.Module {
	path, h := artifactsconnect.NewArtifactsServiceHandler(handler{repo: internalartifacts.NewSQLiteRepository(db)})
	return module.Module{Name: "artifacts", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h}) }, Endpoints: Endpoints}
}
func Schema() string { return internalartifacts.Schema() }

var Endpoints = []module.EndpointDescriptor{
	{ID: "artifacts_approve", Path: artifactsconnect.ArtifactsServiceApproveDraftProcedure, Method: "POST", Summary: "Approve draft after all gates pass", Category: "artifacts"},
	{ID: "artifacts_create", Path: artifactsconnect.ArtifactsServiceCreateDraftProcedure, Method: "POST", Summary: "Create draft", Category: "artifacts"},
	{ID: "artifacts_list", Path: artifactsconnect.ArtifactsServiceListDraftsProcedure, Method: "POST", Summary: "List drafts", Category: "artifacts"},
	{ID: "artifacts_transition", Path: artifactsconnect.ArtifactsServiceTransitionDraftProcedure, Method: "POST", Summary: "Transition draft", Category: "artifacts"},
	{ID: "artifacts_update_body", Path: artifactsconnect.ArtifactsServiceUpdateDraftBodyProcedure, Method: "POST", Summary: "Create attributed draft revision", Category: "artifacts"},
	{ID: "artifacts_publish", Path: artifactsconnect.ArtifactsServicePublishDraftProcedure, Method: "POST", Summary: "Publish an approved draft and append ledger record", Category: "artifacts"},
}
