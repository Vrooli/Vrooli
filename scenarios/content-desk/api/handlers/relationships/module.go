// Package relationships mounts the pinned content-relationship Connect surface.
package relationships

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"content-desk/internal/module"
	internal "content-desk/internal/relationships"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/relationships"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/relationships/relationships_v1connect"
)

type handler struct{ repo internal.Repository }

var _ connectv1.RelationshipsServiceHandler = handler{}

func (h handler) CreateRelation(ctx context.Context, req *connect.Request[v1.CreateRelationRequest]) (*connect.Response[v1.CreateRelationResponse], error) {
	if req.Msg.Relation == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, internal.ErrInvalid)
	}
	m := req.Msg.Relation
	item, err := h.repo.CreateRelation(ctx, internal.Relation{SourceDraftID: m.SourceDraftId, SourceRevisionID: m.SourceRevisionId, TargetDraftID: m.TargetDraftId, TargetRevisionID: m.TargetRevisionId, ProjectID: m.ProjectId, Kind: m.Kind, Primary: m.Primary})
	if err != nil {
		return nil, relationError(err)
	}
	return connect.NewResponse(&v1.CreateRelationResponse{Relation: relationMessage(item)}), nil
}
func (h handler) ListRelations(ctx context.Context, req *connect.Request[v1.ListRelationsRequest]) (*connect.Response[v1.ListRelationsResponse], error) {
	if req.Msg.DraftId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, internal.ErrInvalid)
	}
	items, err := h.repo.ListRelations(ctx, req.Msg.DraftId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ListRelationsResponse{}
	for _, x := range items {
		out.Relations = append(out.Relations, relationMessage(x))
	}
	return connect.NewResponse(out), nil
}
func (h handler) CreateCollection(ctx context.Context, req *connect.Request[v1.CreateCollectionRequest]) (*connect.Response[v1.CreateCollectionResponse], error) {
	c, err := h.repo.CreateCollection(ctx, req.Msg.ProjectId, req.Msg.Name)
	if err != nil {
		return nil, relationError(err)
	}
	return connect.NewResponse(&v1.CreateCollectionResponse{Collection: &v1.Collection{Id: c.ID, ProjectId: c.ProjectID, Name: c.Name}}), nil
}
func (h handler) ListCollections(ctx context.Context, req *connect.Request[v1.ListCollectionsRequest]) (*connect.Response[v1.ListCollectionsResponse], error) {
	items, err := h.repo.ListCollections(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, relationError(err)
	}
	out := &v1.ListCollectionsResponse{}
	for _, item := range items {
		out.Collections = append(out.Collections, &v1.Collection{Id: item.ID, ProjectId: item.ProjectID, Name: item.Name})
	}
	return connect.NewResponse(out), nil
}
func (h handler) AddCollectionItem(ctx context.Context, req *connect.Request[v1.AddCollectionItemRequest]) (*connect.Response[v1.AddCollectionItemResponse], error) {
	if req.Msg.Item == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, internal.ErrInvalid)
	}
	i := internal.CollectionItem{CollectionID: req.Msg.Item.CollectionId, DraftID: req.Msg.Item.DraftId, RevisionID: req.Msg.Item.RevisionId}
	if err := h.repo.AddCollectionItem(ctx, i); err != nil {
		return nil, relationError(err)
	}
	return connect.NewResponse(&v1.AddCollectionItemResponse{Item: itemMessage(i)}), nil
}
func (h handler) RemoveCollectionItem(ctx context.Context, req *connect.Request[v1.RemoveCollectionItemRequest]) (*connect.Response[v1.RemoveCollectionItemResponse], error) {
	if req.Msg.Item == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, internal.ErrInvalid)
	}
	i := internal.CollectionItem{CollectionID: req.Msg.Item.CollectionId, DraftID: req.Msg.Item.DraftId, RevisionID: req.Msg.Item.RevisionId}
	if err := h.repo.RemoveCollectionItem(ctx, i); err != nil {
		return nil, relationError(err)
	}
	return connect.NewResponse(&v1.RemoveCollectionItemResponse{Item: itemMessage(i)}), nil
}
func (h handler) ListCollectionItems(ctx context.Context, req *connect.Request[v1.ListCollectionItemsRequest]) (*connect.Response[v1.ListCollectionItemsResponse], error) {
	items, err := h.repo.ListCollectionItems(ctx, req.Msg.CollectionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ListCollectionItemsResponse{}
	for _, i := range items {
		out.Items = append(out.Items, itemMessage(i))
	}
	return connect.NewResponse(out), nil
}
func (h handler) PreviewImpact(ctx context.Context, req *connect.Request[v1.PreviewImpactRequest]) (*connect.Response[v1.PreviewImpactResponse], error) {
	i, err := h.repo.PreviewImpact(ctx, req.Msg.DraftId, req.Msg.Operation)
	if err != nil {
		return nil, relationError(err)
	}
	return connect.NewResponse(&v1.PreviewImpactResponse{DraftId: i.DraftID, Operation: i.Operation, Relationships: i.Relationships, CollectionMemberships: i.CollectionMemberships, Attachments: i.Attachments, HasDraftSlot: i.HasDraftSlot, HasApproval: i.HasApproval, PreviewOnly: true, CascadesApplied: false}), nil
}
func relationMessage(r internal.Relation) *v1.Relation {
	return &v1.Relation{Id: r.ID, SourceDraftId: r.SourceDraftID, SourceRevisionId: r.SourceRevisionID, TargetDraftId: r.TargetDraftID, TargetRevisionId: r.TargetRevisionID, ProjectId: r.ProjectID, Kind: r.Kind, Primary: r.Primary}
}
func itemMessage(i internal.CollectionItem) *v1.CollectionItem {
	return &v1.CollectionItem{CollectionId: i.CollectionID, DraftId: i.DraftID, RevisionId: i.RevisionID}
}
func relationError(err error) error {
	if errors.Is(err, internal.ErrInvalid) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	if errors.Is(err, internal.ErrNotFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	if errors.Is(err, internal.ErrConflict) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewError(connect.CodeInternal, fmt.Errorf("relationship operation: %w", err))
}
func Module(db *database.RoutedDB) module.Module {
	path, h := connectv1.NewRelationshipsServiceHandler(handler{repo: internal.NewSQLiteRepository(db)})
	return module.Module{Name: "relationships", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h}) }, Endpoints: Endpoints}
}
func Schema() string { return internal.Schema() }

var Endpoints = []module.EndpointDescriptor{
	{ID: "relationships_create_relation", Path: connectv1.RelationshipsServiceCreateRelationProcedure, Method: "POST", Summary: "Create an exact-revision same-project content relation", Category: "relationships"},
	{ID: "relationships_list", Path: connectv1.RelationshipsServiceListRelationsProcedure, Method: "POST", Summary: "List pinned content relations", Category: "relationships"},
	{ID: "relationships_create_collection", Path: connectv1.RelationshipsServiceCreateCollectionProcedure, Method: "POST", Summary: "Create a non-owning saved collection", Category: "relationships"},
	{ID: "relationships_list_collections", Path: connectv1.RelationshipsServiceListCollectionsProcedure, Method: "POST", Summary: "List saved collections in one project", Category: "relationships"},
	{ID: "relationships_add_collection_item", Path: connectv1.RelationshipsServiceAddCollectionItemProcedure, Method: "POST", Summary: "Add a pinned revision to a saved collection", Category: "relationships"},
	{ID: "relationships_remove_collection_item", Path: connectv1.RelationshipsServiceRemoveCollectionItemProcedure, Method: "POST", Summary: "Remove one pinned revision from a saved collection", Category: "relationships"},
	{ID: "relationships_list_collection_items", Path: connectv1.RelationshipsServiceListCollectionItemsProcedure, Method: "POST", Summary: "List saved collection references", Category: "relationships"},
	{ID: "relationships_preview_impact", Path: connectv1.RelationshipsServicePreviewImpactProcedure, Method: "POST", Summary: "Preview detach, move, or archive impact without mutation", Category: "relationships"},
}
