// Package activity exposes the append-only event log over Connect-RPC.
//
// The package owns no state of its own: internal/activity owns activity_events.
// The handler only maps proto request/response shapes onto that domain and
// translates domain validation errors into Connect codes.
package activity

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	activityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/activity"
	activityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/activity/activity_v1connect"

	domain "react-component-library/internal/activity"
	"react-component-library/internal/module"
)

// Service is the read surface the handler needs. The concrete implementation
// is domain.Store; tests inject a fake without touching a database.
type Service interface {
	List(ctx context.Context, filter domain.Filter, limit int) ([]domain.Event, error)
	Summary(ctx context.Context, days int) (domain.Summary, error)
}

// Handler adapts the activity domain onto the generated Connect interface.
type Handler struct{ service Service }

// NewHandler wires a service implementation.
func NewHandler(service Service) *Handler { return &Handler{service: service} }

// Module mounts ActivityService over the scenario database.
func Module(db domain.Database) module.Module {
	return ModuleFromService(domain.NewSQLiteStore(db))
}

// ModuleFromService is the constructor tests and alternate stores use.
func ModuleFromService(service Service) module.Module {
	path, handler := activityconnect.NewActivityServiceHandler(NewHandler(service))
	return module.Module{
		Name: "activity",
		Mount: func(router *mux.Router) {
			connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internal/activity.Schema for the modules registry.
func Schema() string { return domain.Schema() }

// ListActivity returns events newest-first.
func (h *Handler) ListActivity(ctx context.Context, req *connect.Request[activityv1.ListActivityRequest]) (*connect.Response[activityv1.ListActivityResponse], error) {
	if h == nil || h.service == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("activity service is not configured"))
	}
	events, err := h.service.List(ctx, domain.Filter{
		SubjectKind: strings.TrimSpace(req.Msg.GetSubjectKind()),
		SubjectID:   strings.TrimSpace(req.Msg.GetSubjectId()),
		Actor:       strings.TrimSpace(req.Msg.GetActor()),
		Since:       strings.TrimSpace(req.Msg.GetSince()),
	}, int(req.Msg.GetLimit()))
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*activityv1.ActivityEvent, 0, len(events))
	for _, event := range events {
		out = append(out, &activityv1.ActivityEvent{
			Id:          event.ID,
			Actor:       event.Actor,
			Action:      event.Action,
			SubjectKind: event.SubjectKind,
			SubjectId:   event.SubjectID,
			Summary:     event.Summary,
			BeforeJson:  event.BeforeJSON,
			AfterJson:   event.AfterJSON,
			At:          event.At,
			TraceId:     event.TraceID,
		})
	}
	return connect.NewResponse(&activityv1.ListActivityResponse{Events: out}), nil
}

// GetActivitySummary returns the bounded Stats activity rollup.
func (h *Handler) GetActivitySummary(ctx context.Context, req *connect.Request[activityv1.GetActivitySummaryRequest]) (*connect.Response[activityv1.GetActivitySummaryResponse], error) {
	if h == nil || h.service == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("activity service is not configured"))
	}
	summary, err := h.service.Summary(ctx, int(req.Msg.GetDays()))
	if err != nil {
		return nil, mapError(err)
	}
	byDay := make([]*activityv1.ActivityDay, 0, len(summary.ByDay))
	for _, day := range summary.ByDay {
		byDay = append(byDay, &activityv1.ActivityDay{Date: day.Date, Count: int32(day.Count)})
	}
	byActor := make([]*activityv1.ActivityActor, 0, len(summary.ByActor))
	for _, actor := range summary.ByActor {
		byActor = append(byActor, &activityv1.ActivityActor{Actor: actor.Actor, Count: int32(actor.Count)})
	}
	byAction := make([]*activityv1.ActivityAction, 0, len(summary.ByAction))
	for _, action := range summary.ByAction {
		byAction = append(byAction, &activityv1.ActivityAction{Action: action.Action, Count: int32(action.Count)})
	}
	return connect.NewResponse(&activityv1.GetActivitySummaryResponse{
		Total:    int32(summary.Total),
		ByDay:    byDay,
		ByActor:  byActor,
		ByAction: byAction,
		Empty:    summary.Empty,
	}), nil
}

// mapError keeps the wire contract to the two honest failure modes: bad input
// and a storage failure. Storage details stay in the log, not the response.
func mapError(err error) error {
	if errors.Is(err, domain.ErrInvalid) {
		message := strings.TrimPrefix(err.Error(), domain.ErrInvalid.Error()+": ")
		return connect.NewError(connect.CodeInvalidArgument, errors.New(message))
	}
	return connect.NewError(connect.CodeInternal, errors.New("activity storage failure"))
}

// Endpoints is the static surface consumed by `make endpoints`.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "activity_list",
		Path:        activityconnect.ActivityServiceListActivityProcedure,
		Method:      "POST",
		Summary:     "List activity events",
		Description: "Returns durable mutation events newest-first. Optional subject_kind, subject_id and actor filters narrow the log; since is an inclusive RFC3339 lower bound. limit defaults to 100 and is clamped to 500.",
		Category:    "activity",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"subject_kind": "string", "subject_id": "string", "actor": "string", "limit": "integer (default 100, max 500)", "since": "string (RFC3339, inclusive)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"events": "array<ActivityEvent>"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "since is not RFC3339"}},
	},
	{
		ID:          "activity_summary",
		Path:        activityconnect.ActivityServiceGetActivitySummaryProcedure,
		Method:      "POST",
		Summary:     "Get the activity summary",
		Description: "Returns total, by_day, by_actor and by_action counts for the last days (0 means 30, capped at 365). empty=true is a measured zero, not an unavailable source.",
		Category:    "activity",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"days": "integer (0 means 30, max 365)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"total": "integer", "by_day": "array<ActivityDay>", "by_actor": "array<ActivityActor>", "by_action": "array<ActivityAction>", "empty": "boolean"}},
	},
}
