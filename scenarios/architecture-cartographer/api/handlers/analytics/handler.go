// Package analytics is the Connect-RPC surface for the analytics
// domain. Translates between proto wire types and internal/analytics
// domain types.
package analytics

import (
	"context"
	"errors"
	"strings"
	"time"

	"architecture-cartographer/internal/analytics"

	"connectrpc.com/connect"
	analyticsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/analytics"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/analytics/analytics_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Handler implements analytics_v1connect.AnalyticsServiceHandler.
type Handler struct {
	analytics_v1connect.UnimplementedAnalyticsServiceHandler
	svc analytics.Service
}

// NewHandler constructs the Connect handler.
func NewHandler(svc analytics.Service) *Handler { return &Handler{svc: svc} }

var _ analytics_v1connect.AnalyticsServiceHandler = (*Handler)(nil)

func (h *Handler) ListEvents(ctx context.Context, req *connect.Request[analyticsv1.ListEventsRequest]) (*connect.Response[analyticsv1.ListEventsResponse], error) {
	filter := analytics.EventFilter{
		Scenario:  strings.TrimSpace(req.Msg.GetScenario()),
		PageSize:  int(req.Msg.GetPageSize()),
		PageToken: req.Msg.GetPageToken(),
	}
	if since := req.Msg.GetSince(); since != nil {
		filter.Since = since.AsTime()
	}
	for _, k := range req.Msg.GetKinds() {
		filter.Kinds = append(filter.Kinds, protoToEventKind(k))
	}
	page, err := h.svc.ListEvents(ctx, filter)
	if err != nil {
		return nil, connect.NewError(analytics.ErrorToConnectCode(err), err)
	}
	out := &analyticsv1.ListEventsResponse{NextPageToken: page.NextPageToken}
	for _, e := range page.Events {
		out.Events = append(out.Events, eventToProto(e))
	}
	return connect.NewResponse(out), nil
}

func (h *Handler) GetStats(ctx context.Context, req *connect.Request[analyticsv1.GetStatsRequest]) (*connect.Response[analyticsv1.GetStatsResponse], error) {
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	s, err := h.svc.Stats(ctx, scenario)
	if err != nil {
		return nil, connect.NewError(analytics.ErrorToConnectCode(err), err)
	}
	return connect.NewResponse(&analyticsv1.GetStatsResponse{Stats: statsToProto(s)}), nil
}

func (h *Handler) ListPlacements(ctx context.Context, req *connect.Request[analyticsv1.ListPlacementsRequest]) (*connect.Response[analyticsv1.ListPlacementsResponse], error) {
	filter := analytics.PlacementFilter{
		Scenario:  strings.TrimSpace(req.Msg.GetScenario()),
		Outcomes:  append([]string(nil), req.Msg.GetOutcomes()...),
		PageSize:  int(req.Msg.GetPageSize()),
		PageToken: req.Msg.GetPageToken(),
	}
	page, err := h.svc.ListPlacements(ctx, filter)
	if err != nil {
		return nil, connect.NewError(analytics.ErrorToConnectCode(err), err)
	}
	out := &analyticsv1.ListPlacementsResponse{NextPageToken: page.NextPageToken}
	for _, p := range page.Placements {
		out.Placements = append(out.Placements, placementToProto(p))
	}
	return connect.NewResponse(out), nil
}

func (h *Handler) RecordOverride(ctx context.Context, req *connect.Request[analyticsv1.RecordOverrideRequest]) (*connect.Response[analyticsv1.RecordOverrideResponse], error) {
	if req.Msg.GetDryRun() || req.Header().Get("X-Dry-Run") == "true" {
		// Preview shape — return the override the service would have
		// recorded without touching storage.
		ov := analytics.Override{
			Scenario:       strings.TrimSpace(req.Msg.GetScenario()),
			ChunkID:        req.Msg.GetChunkId(),
			VerdictDomain:  req.Msg.GetVerdictDomain(),
			ChosenDomain:   req.Msg.GetChosenDomain(),
			Note:           req.Msg.GetNote(),
			VerdictEventID: req.Msg.GetVerdictEventId(),
			RecordedAt:     time.Time{},
		}
		return connect.NewResponse(&analyticsv1.RecordOverrideResponse{
			Override: overrideToProto(ov),
			DryRun:   true,
		}), nil
	}
	saved, err := h.svc.RecordOverride(ctx, analytics.Override{
		Scenario:       req.Msg.GetScenario(),
		ChunkID:        req.Msg.GetChunkId(),
		VerdictDomain:  req.Msg.GetVerdictDomain(),
		ChosenDomain:   req.Msg.GetChosenDomain(),
		Note:           req.Msg.GetNote(),
		VerdictEventID: req.Msg.GetVerdictEventId(),
	})
	if err != nil {
		return nil, connect.NewError(analytics.ErrorToConnectCode(err), err)
	}
	return connect.NewResponse(&analyticsv1.RecordOverrideResponse{
		Override: overrideToProto(saved),
	}), nil
}

// -------------------------- proto<->domain --------------------------

func eventToProto(e analytics.Event) *analyticsv1.Event {
	out := &analyticsv1.Event{
		Id:              e.ID,
		Kind:            eventKindToProto(e.Kind),
		Scenario:        e.Scenario,
		Domain:          e.Domain,
		ConflictId:      e.ConflictID,
		ChunkId:         e.ChunkID,
		PlanId:          e.PlanID,
		RunId:           e.RunID,
		CorrectsEventId: e.CorrectsEventID,
		Payload:         append([]byte(nil), e.Payload...),
		Actor:           e.Actor,
	}
	if !e.RecordedAt.IsZero() {
		out.RecordedAt = timestamppb.New(e.RecordedAt)
	}
	return out
}

func placementToProto(p analytics.Placement) *analyticsv1.Placement {
	out := &analyticsv1.Placement{
		Id:        p.ID,
		Scenario:  p.Scenario,
		ChunkId:   p.ChunkID,
		ChunkPath: p.ChunkPath,
		Outcome:   p.Outcome,
		AutoActed: p.AutoActed,
	}
	if !p.RecordedAt.IsZero() {
		out.RecordedAt = timestamppb.New(p.RecordedAt)
	}
	return out
}

func overrideToProto(o analytics.Override) *analyticsv1.Override {
	out := &analyticsv1.Override{
		Id:             o.ID,
		Scenario:       o.Scenario,
		ChunkId:        o.ChunkID,
		VerdictDomain:  o.VerdictDomain,
		ChosenDomain:   o.ChosenDomain,
		Note:           o.Note,
		VerdictEventId: o.VerdictEventID,
	}
	if !o.RecordedAt.IsZero() {
		out.RecordedAt = timestamppb.New(o.RecordedAt)
	}
	return out
}

func statsToProto(s analytics.StatsSummary) *analyticsv1.StatsSummary {
	return &analyticsv1.StatsSummary{
		Scenario:                     s.Scenario,
		ConflictsDetected:            s.ConflictsDetected,
		ConflictsResolved:            s.ConflictsResolved,
		ConflictsForceResolved:       s.ConflictsForceResolved,
		PlacementsAuto:               s.PlacementsAuto,
		PlacementsSuggest:            s.PlacementsSuggest,
		Overrides:                    s.Overrides,
		VerdictSuccessRate:           s.VerdictSuccessRate,
		VerdictSuccessRateSuppressed: s.VerdictSuccessRateSuppressed,
		VerdictObservationCount:      s.VerdictObservationCount,
	}
}

func eventKindToProto(k analytics.EventKind) analyticsv1.EventKind {
	switch k {
	case analytics.EventKindConflictDetected:
		return analyticsv1.EventKind_EVENT_KIND_CONFLICT_DETECTED
	case analytics.EventKindConflictAssigned:
		return analyticsv1.EventKind_EVENT_KIND_CONFLICT_ASSIGNED
	case analytics.EventKindConflictResolved:
		return analyticsv1.EventKind_EVENT_KIND_CONFLICT_RESOLVED
	case analytics.EventKindConflictReopened:
		return analyticsv1.EventKind_EVENT_KIND_CONFLICT_REOPENED
	case analytics.EventKindConflictForceResolved:
		return analyticsv1.EventKind_EVENT_KIND_CONFLICT_FORCE_RESOLVED
	case analytics.EventKindVerdictProduced:
		return analyticsv1.EventKind_EVENT_KIND_VERDICT_PRODUCED
	case analytics.EventKindPlacementAuto:
		return analyticsv1.EventKind_EVENT_KIND_PLACEMENT_AUTO
	case analytics.EventKindPlacementSuggest:
		return analyticsv1.EventKind_EVENT_KIND_PLACEMENT_SUGGEST
	case analytics.EventKindOverrideRecorded:
		return analyticsv1.EventKind_EVENT_KIND_OVERRIDE_RECORDED
	case analytics.EventKindApplyPlanned:
		return analyticsv1.EventKind_EVENT_KIND_APPLY_PLANNED
	case analytics.EventKindApplyRan:
		return analyticsv1.EventKind_EVENT_KIND_APPLY_RAN
	case analytics.EventKindApplyBuildGreen:
		return analyticsv1.EventKind_EVENT_KIND_APPLY_BUILD_GREEN
	case analytics.EventKindApplyBuildRed:
		return analyticsv1.EventKind_EVENT_KIND_APPLY_BUILD_RED
	case analytics.EventKindApplyReverted:
		return analyticsv1.EventKind_EVENT_KIND_APPLY_REVERTED
	default:
		return analyticsv1.EventKind_EVENT_KIND_UNSPECIFIED
	}
}

func protoToEventKind(k analyticsv1.EventKind) analytics.EventKind {
	switch k {
	case analyticsv1.EventKind_EVENT_KIND_CONFLICT_DETECTED:
		return analytics.EventKindConflictDetected
	case analyticsv1.EventKind_EVENT_KIND_CONFLICT_ASSIGNED:
		return analytics.EventKindConflictAssigned
	case analyticsv1.EventKind_EVENT_KIND_CONFLICT_RESOLVED:
		return analytics.EventKindConflictResolved
	case analyticsv1.EventKind_EVENT_KIND_CONFLICT_REOPENED:
		return analytics.EventKindConflictReopened
	case analyticsv1.EventKind_EVENT_KIND_CONFLICT_FORCE_RESOLVED:
		return analytics.EventKindConflictForceResolved
	case analyticsv1.EventKind_EVENT_KIND_VERDICT_PRODUCED:
		return analytics.EventKindVerdictProduced
	case analyticsv1.EventKind_EVENT_KIND_PLACEMENT_AUTO:
		return analytics.EventKindPlacementAuto
	case analyticsv1.EventKind_EVENT_KIND_PLACEMENT_SUGGEST:
		return analytics.EventKindPlacementSuggest
	case analyticsv1.EventKind_EVENT_KIND_OVERRIDE_RECORDED:
		return analytics.EventKindOverrideRecorded
	case analyticsv1.EventKind_EVENT_KIND_APPLY_PLANNED:
		return analytics.EventKindApplyPlanned
	case analyticsv1.EventKind_EVENT_KIND_APPLY_RAN:
		return analytics.EventKindApplyRan
	case analyticsv1.EventKind_EVENT_KIND_APPLY_BUILD_GREEN:
		return analytics.EventKindApplyBuildGreen
	case analyticsv1.EventKind_EVENT_KIND_APPLY_BUILD_RED:
		return analytics.EventKindApplyBuildRed
	case analyticsv1.EventKind_EVENT_KIND_APPLY_REVERTED:
		return analytics.EventKindApplyReverted
	default:
		return analytics.EventKind("")
	}
}
