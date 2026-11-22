// Package intents exposes editorial date/daypart intentions. This surface never
// reserves operator time or schedules provider work.
package intents

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	internalartifacts "content-desk/internal/artifacts"
	internalintents "content-desk/internal/intents"
	"content-desk/internal/module"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	intentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/intents"
	intentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/intents/intents_v1connect"
)

type handler struct{ service internalintents.Service }
type revisionSource struct{ repo internalartifacts.Repository }

func (s revisionSource) CurrentRevision(ctx context.Context, draftID string) (internalintents.CurrentRevision, error) {
	r, err := s.repo.GetCurrentRevision(ctx, draftID)
	if err != nil {
		return internalintents.CurrentRevision{}, err
	}
	return internalintents.CurrentRevision{DraftID: r.DraftID, Body: r.Body, RevisionID: r.RevisionID}, nil
}

var _ intentconnect.PublicationIntentServiceHandler = handler{}

func (h handler) ListPublicationIntents(ctx context.Context, req *connect.Request[intentv1.ListPublicationIntentsRequest]) (*connect.Response[intentv1.ListPublicationIntentsResponse], error) {
	items, err := h.service.List(ctx, req.Msg.DraftId, req.Msg.Destination)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &intentv1.ListPublicationIntentsResponse{}
	for _, item := range items {
		out.Intents = append(out.Intents, message(item))
	}
	return connect.NewResponse(out), nil
}
func (h handler) SetPublicationIntent(ctx context.Context, req *connect.Request[intentv1.SetPublicationIntentRequest]) (*connect.Response[intentv1.SetPublicationIntentResponse], error) {
	m := req.Msg
	item, err := h.service.Set(ctx, internalintents.SetInput{DraftID: m.DraftId, Destination: m.Destination, DesiredLocalDate: m.DesiredLocalDate, TimeZone: m.TimeZone, Daypart: m.Daypart, ExpectedRevisionID: m.ExpectedRevisionId, ExpectedBody: m.ExpectedBody, ExpectedIntentVersion: m.ExpectedIntentVersion, IdempotencyKey: m.IdempotencyKey})
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("publication intention not saved: %w", err))
	}
	return connect.NewResponse(&intentv1.SetPublicationIntentResponse{Intent: message(item)}), nil
}
func message(x internalintents.PublicationIntent) *intentv1.PublicationIntent {
	return &intentv1.PublicationIntent{Id: x.ID, DraftId: x.DraftID, Version: x.Version, SourceRevisionId: x.SourceRevisionID, BodySha256: x.BodySHA256, Destination: x.Destination, DesiredLocalDate: x.DesiredLocalDate, TimeZone: x.TimeZone, Daypart: x.Daypart, Status: x.Status, CreatedAt: x.CreatedAt.Format(time.RFC3339Nano)}
}
func Module(db *database.RoutedDB) module.Module {
	path, h := intentconnect.NewPublicationIntentServiceHandler(handler{service: internalintents.NewService(db, revisionSource{repo: internalartifacts.NewSQLiteRepository(db)})})
	return module.Module{Name: "intents", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h}) }, Endpoints: Endpoints}
}
func Schema() string { return internalintents.Schema() }

var Endpoints = []module.EndpointDescriptor{
	{ID: "intent_list", Path: intentconnect.PublicationIntentServiceListPublicationIntentsProcedure, Method: "POST", Summary: "List desired date and daypart intentions", Category: "plan"},
	{ID: "intent_set", Path: intentconnect.PublicationIntentServiceSetPublicationIntentProcedure, Method: "POST", Summary: "Record revision-guarded editorial timing intention", Category: "plan"},
}
