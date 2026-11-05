package catalog

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/music-library/v1/catalog"
	svc "github.com/vrooli/vrooli/packages/proto/gen/go/music-library/v1/catalog/catalog_v1connect"
	"music-library/internal/catalog"
	"music-library/internal/module"
)

type handler struct{ repo catalog.Repository }

func Module(db *database.RoutedDB) module.Module {
	h := &handler{repo: catalog.NewStore(db)}
	path, server := svc.NewCatalogServiceHandler(h)
	return module.Module{Name: "catalog", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: server}) }, Endpoints: Endpoints}
}
func Schema() string { return catalog.Schema() }
func mapError(err error) error {
	if errors.Is(err, catalog.ErrInvalid) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	if errors.Is(err, catalog.ErrNotFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
func domain(t *pb.Track) catalog.Track {
	if t == nil {
		return catalog.Track{}
	}
	uses := make([]catalog.Use, 0, len(t.Uses))
	for _, u := range t.Uses {
		if u != nil {
			uses = append(uses, catalog.Use{Project: u.Project, Revision: u.Revision, Status: u.Status})
		}
	}
	return catalog.Track{SHA256: t.Sha256, Title: t.Title, SourcePath: t.SourcePath, TakeID: t.TakeId, ReceiptPath: t.ReceiptPath, Tags: t.Tags, Approval: t.Approval, Feedback: t.Feedback, Uses: uses}
}
func proto(t catalog.Track) *pb.Track {
	uses := make([]*pb.TrackUse, 0, len(t.Uses))
	for _, u := range t.Uses {
		uses = append(uses, &pb.TrackUse{Project: u.Project, Revision: u.Revision, Status: u.Status})
	}
	return &pb.Track{Sha256: t.SHA256, Title: t.Title, SourcePath: t.SourcePath, TakeId: t.TakeID, ReceiptPath: t.ReceiptPath, Tags: t.Tags, Approval: t.Approval, Feedback: t.Feedback, Uses: uses}
}
func (h *handler) PutTrack(ctx context.Context, r *connect.Request[pb.PutTrackRequest]) (*connect.Response[pb.PutTrackResponse], error) {
	t, e := h.repo.Put(ctx, domain(r.Msg.Track))
	if e != nil {
		return nil, mapError(e)
	}
	return connect.NewResponse(&pb.PutTrackResponse{Track: proto(t)}), nil
}
func (h *handler) ListTracks(ctx context.Context, r *connect.Request[pb.ListTracksRequest]) (*connect.Response[pb.ListTracksResponse], error) {
	ts, e := h.repo.List(ctx, catalog.Filter{Tag: r.Msg.Tag, UnusedOnly: r.Msg.UnusedOnly, Project: r.Msg.Project})
	if e != nil {
		return nil, mapError(e)
	}
	out := make([]*pb.Track, 0, len(ts))
	for _, t := range ts {
		out = append(out, proto(t))
	}
	return connect.NewResponse(&pb.ListTracksResponse{Tracks: out}), nil
}
func (h *handler) RecordUse(ctx context.Context, r *connect.Request[pb.RecordUseRequest]) (*connect.Response[pb.RecordUseResponse], error) {
	u := r.Msg.Use
	if u == nil {
		return nil, mapError(catalog.ErrInvalid)
	}
	t, e := h.repo.RecordUse(ctx, r.Msg.Sha256, catalog.Use{Project: u.Project, Revision: u.Revision, Status: u.Status})
	if e != nil {
		return nil, mapError(e)
	}
	return connect.NewResponse(&pb.RecordUseResponse{Track: proto(t)}), nil
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "catalog_put", Path: svc.CatalogServicePutTrackProcedure, Method: "POST", Summary: "Save a catalog track", Description: "Store a content-identified track, tags and explicit listener feedback.", Category: "catalog"},
	{ID: "catalog_list", Path: svc.CatalogServiceListTracksProcedure, Method: "POST", Summary: "List catalog tracks", Description: "Filter tracks by tag, project and accepted-use state.", Category: "catalog"},
	{ID: "catalog_use", Path: svc.CatalogServiceRecordUseProcedure, Method: "POST", Summary: "Record track use", Description: "Associate a track with a project revision as draft or accepted.", Category: "catalog"},
}
