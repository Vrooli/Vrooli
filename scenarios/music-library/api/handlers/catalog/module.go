package catalog

import (
	"context"
	"errors"
	"time"

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
	if errors.Is(err, catalog.ErrUnavailable) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	if errors.Is(err, catalog.ErrReservation) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
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
			uses = append(uses, catalog.Use{Project: u.Project, Revision: u.Revision, Status: u.Status, Purpose: u.Purpose})
		}
	}
	priority := int(t.BankPriority)
	if priority == 0 {
		priority = 100
	}
	return catalog.Track{SHA256: t.Sha256, Title: t.Title, SourcePath: t.SourcePath, TakeID: t.TakeId, ReceiptPath: t.ReceiptPath, Tags: t.Tags, Purposes: t.Purposes, BankPriority: priority, Approval: t.Approval, Feedback: t.Feedback, Uses: uses}
}
func proto(t catalog.Track) *pb.Track {
	uses := make([]*pb.TrackUse, 0, len(t.Uses))
	for _, u := range t.Uses {
		uses = append(uses, &pb.TrackUse{Project: u.Project, Revision: u.Revision, Status: u.Status, Purpose: u.Purpose})
	}
	return &pb.Track{Sha256: t.SHA256, Title: t.Title, SourcePath: t.SourcePath, TakeId: t.TakeID, ReceiptPath: t.ReceiptPath, Tags: t.Tags, Purposes: t.Purposes, BankPriority: int32(t.BankPriority), BankState: t.BankState, Approval: t.Approval, Feedback: t.Feedback, Uses: uses}
}
func (h *handler) PutTrack(ctx context.Context, r *connect.Request[pb.PutTrackRequest]) (*connect.Response[pb.PutTrackResponse], error) {
	t, e := h.repo.Put(ctx, domain(r.Msg.Track))
	if e != nil {
		return nil, mapError(e)
	}
	return connect.NewResponse(&pb.PutTrackResponse{Track: proto(t)}), nil
}
func (h *handler) ListTracks(ctx context.Context, r *connect.Request[pb.ListTracksRequest]) (*connect.Response[pb.ListTracksResponse], error) {
	ts, e := h.repo.List(ctx, catalog.Filter{Tag: r.Msg.Tag, Purpose: r.Msg.Purpose, UnusedOnly: r.Msg.UnusedOnly, Project: r.Msg.Project})
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
	t, e := h.repo.RecordUse(ctx, r.Msg.Sha256, catalog.Use{Project: u.Project, Revision: u.Revision, Status: u.Status, Purpose: u.Purpose})
	if e != nil {
		return nil, mapError(e)
	}
	return connect.NewResponse(&pb.RecordUseResponse{Track: proto(t)}), nil
}

func (h *handler) ReserveTrack(ctx context.Context, r *connect.Request[pb.ReserveTrackRequest]) (*connect.Response[pb.ReserveTrackResponse], error) {
	input := r.Msg
	reservation, track, err := h.repo.Reserve(ctx, catalog.ReserveRequest{SHA256: input.Sha256, Purpose: input.Purpose, Project: input.Project, Revision: input.Revision, Consumer: input.Consumer, LeaseSeconds: int(input.LeaseSeconds)})
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&pb.ReserveTrackResponse{Track: proto(track), ReservationId: reservation.ID, ExpiresAt: reservation.ExpiresAt.Format(time.RFC3339Nano)}), nil
}

func (h *handler) ReleaseReservation(ctx context.Context, r *connect.Request[pb.ReleaseReservationRequest]) (*connect.Response[pb.ReleaseReservationResponse], error) {
	track, err := h.repo.Release(ctx, r.Msg.ReservationId, r.Msg.Consumer)
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&pb.ReleaseReservationResponse{Track: proto(track)}), nil
}

func (h *handler) CommitReservation(ctx context.Context, r *connect.Request[pb.CommitReservationRequest]) (*connect.Response[pb.CommitReservationResponse], error) {
	track, use, err := h.repo.CommitReservation(ctx, r.Msg.ReservationId, r.Msg.Consumer, r.Msg.Status)
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&pb.CommitReservationResponse{Track: proto(track), Use: &pb.TrackUse{Project: use.Project, Revision: use.Revision, Status: use.Status, Purpose: use.Purpose}}), nil
}

func (h *handler) RollbackReservation(ctx context.Context, r *connect.Request[pb.RollbackReservationRequest]) (*connect.Response[pb.RollbackReservationResponse], error) {
	track, err := h.repo.RollbackReservation(ctx, r.Msg.ReservationId, r.Msg.Consumer)
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&pb.RollbackReservationResponse{Track: proto(track)}), nil
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "catalog_put", Path: svc.CatalogServicePutTrackProcedure, Method: "POST", Summary: "Save a catalog track", Description: "Store a content-identified track, tags and explicit listener feedback.", Category: "catalog"},
	{ID: "catalog_list", Path: svc.CatalogServiceListTracksProcedure, Method: "POST", Summary: "List catalog tracks", Description: "Filter tracks by tag, project and accepted-use state.", Category: "catalog"},
	{ID: "catalog_use", Path: svc.CatalogServiceRecordUseProcedure, Method: "POST", Summary: "Record track use", Description: "Associate a track with a project revision as draft or accepted.", Category: "catalog"},
	{ID: "catalog_reserve", Path: svc.CatalogServiceReserveTrackProcedure, Method: "POST", Summary: "Reserve an approved bank track", Description: "Atomically reserve a liked track with the requested purpose for one consumer and film revision. Supply a SHA-256 to intentionally reuse a known track.", Category: "catalog"},
	{ID: "catalog_release_reservation", Path: svc.CatalogServiceReleaseReservationProcedure, Method: "POST", Summary: "Release a bank reservation", Description: "Release a reservation owned by the specified consumer so another project can select the track.", Category: "catalog"},
	{ID: "catalog_commit_reservation", Path: svc.CatalogServiceCommitReservationProcedure, Method: "POST", Summary: "Commit a bank reservation", Description: "Atomically bind a reserved track to its project revision as draft or accepted and retain its use purpose.", Category: "catalog"},
	{ID: "catalog_rollback_reservation", Path: svc.CatalogServiceRollbackReservationProcedure, Method: "POST", Summary: "Roll back a draft bank use", Description: "Release a committed draft when its film is abandoned; accepted uses remain immutable and delivered refill events receive a cancellation request.", Category: "catalog"},
}
