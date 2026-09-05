// Package world mounts WorldService: world config, per-scene layouts and the
// server-streamed swarm feed.
package world

import (
	"context"
	"errors"
	"net/http"
	"time"

	"connectrpc.com/connect"
	worldv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/world"
	worldconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/world/world_v1connect"

	domain "prompt-manager/internal/world"
)

type connectHandler struct {
	worldconnect.UnimplementedWorldServiceHandler
	store *domain.Store
	hub   *domain.Hub
}

// NewConnectMount returns the Connect path and handler for WorldService.
func NewConnectMount(store *domain.Store, hub *domain.Hub) (string, http.Handler) {
	return worldconnect.NewWorldServiceHandler(&connectHandler{store: store, hub: hub})
}

func configToProto(c domain.Config) *worldv1.WorldConfig {
	return &worldv1.WorldConfig{
		Scene:           c.Scene,
		QualityProfile:  c.QualityProfile,
		QualityAuto:     c.QualityAuto,
		PeriodMode:      c.PeriodMode,
		TwoDMode:        c.TwoDMode,
		ShowDiagnostics: c.ShowDiagnostics,
		Scale:           c.Scale,
		UpdatedAt:       c.UpdatedAt,
	}
}

func configFromProto(p *worldv1.WorldConfig) domain.Config {
	if p == nil {
		return domain.Config{}
	}
	return domain.Config{
		Scene:           p.GetScene(),
		QualityProfile:  p.GetQualityProfile(),
		QualityAuto:     p.GetQualityAuto(),
		PeriodMode:      p.GetPeriodMode(),
		TwoDMode:        p.GetTwoDMode(),
		ShowDiagnostics: p.GetShowDiagnostics(),
		Scale:           p.GetScale(),
	}
}

func layoutToProto(l domain.Layout) *worldv1.WorldLayout {
	out := &worldv1.WorldLayout{Scene: l.Scene, UpdatedAt: l.UpdatedAt}
	for _, o := range l.Overrides {
		po := &worldv1.LayoutOverride{PlaceId: o.PlaceID, Removed: o.Removed}
		if o.Position != nil {
			po.Position = &worldv1.Vec2{X: o.Position.X, Z: o.Position.Z}
		}
		if o.Rotation != nil {
			r := *o.Rotation
			po.Rotation = &r
		}
		out.Overrides = append(out.Overrides, po)
	}
	for _, d := range l.Decor {
		out.Decor = append(out.Decor, &worldv1.DecorAddition{Id: d.ID, PropId: d.PropID, Position: &worldv1.Vec2{X: d.Position.X, Z: d.Position.Z}, Rotation: d.Rotation, Scale: d.Scale})
	}
	return out
}

func layoutFromProto(p *worldv1.WorldLayout) domain.Layout {
	out := domain.Layout{Overrides: []domain.Override{}, Decor: []domain.Decor{}}
	if p == nil {
		return out
	}
	out.Scene = p.GetScene()
	for _, o := range p.GetOverrides() {
		ov := domain.Override{PlaceID: o.GetPlaceId(), Removed: o.GetRemoved()}
		if o.Position != nil {
			ov.Position = &domain.Vec2{X: o.GetPosition().GetX(), Z: o.GetPosition().GetZ()}
		}
		if o.Rotation != nil {
			r := o.GetRotation()
			ov.Rotation = &r
		}
		out.Overrides = append(out.Overrides, ov)
	}
	for _, d := range p.GetDecor() {
		out.Decor = append(out.Decor, domain.Decor{ID: d.GetId(), PropID: d.GetPropId(), Position: domain.Vec2{X: d.GetPosition().GetX(), Z: d.GetPosition().GetZ()}, Rotation: d.GetRotation(), Scale: d.GetScale()})
	}
	return out
}

var kindToProto = map[domain.EventKind]worldv1.WorldEventKind{
	domain.KindSnapshot:           worldv1.WorldEventKind_WORLD_EVENT_KIND_SNAPSHOT,
	domain.KindRunStarted:         worldv1.WorldEventKind_WORLD_EVENT_KIND_RUN_STARTED,
	domain.KindRunFinished:        worldv1.WorldEventKind_WORLD_EVENT_KIND_RUN_FINISHED,
	domain.KindRunFailed:          worldv1.WorldEventKind_WORLD_EVENT_KIND_RUN_FAILED,
	domain.KindHeartbeatUpcoming:  worldv1.WorldEventKind_WORLD_EVENT_KIND_HEARTBEAT_UPCOMING,
	domain.KindHeartbeatCancelled: worldv1.WorldEventKind_WORLD_EVENT_KIND_HEARTBEAT_CANCELLED,
	domain.KindAgentMessage:       worldv1.WorldEventKind_WORLD_EVENT_KIND_AGENT_MESSAGE,
}

// EventToProto converts a feed event to its wire form.
func EventToProto(e domain.Event) *worldv1.WorldEvent {
	out := &worldv1.WorldEvent{
		Kind:    kindToProto[e.Kind],
		Seq:     e.Seq,
		At:      e.At.Format(time.RFC3339Nano),
		AgentId: e.AgentID,
		TeamId:  e.TeamID,
		RunId:   e.RunID,
		Message: e.Message,
	}
	if !e.ScheduledAt.IsZero() {
		out.ScheduledAt = e.ScheduledAt.Format(time.RFC3339)
	}
	for _, run := range e.ActiveRuns {
		out.ActiveRuns = append(out.ActiveRuns, &worldv1.ActiveRunSummary{TeamId: run.TeamID, AgentId: run.AgentID, RunId: run.RunID, StartedAt: run.StartedAt.Format(time.RFC3339)})
	}
	for _, up := range e.Upcoming {
		out.Upcoming = append(out.Upcoming, &worldv1.UpcomingHeartbeat{TeamId: up.TeamID, AgentId: up.AgentID, ScheduledAt: up.ScheduledAt.Format(time.RFC3339)})
	}
	return out
}

func (h *connectHandler) GetWorldConfig(_ context.Context, _ *connect.Request[worldv1.GetWorldConfigRequest]) (*connect.Response[worldv1.WorldConfig], error) {
	cfg, err := h.store.LoadConfig()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(configToProto(cfg)), nil
}

func (h *connectHandler) SetWorldConfig(_ context.Context, req *connect.Request[worldv1.SetWorldConfigRequest]) (*connect.Response[worldv1.WorldConfig], error) {
	saved, err := h.store.SaveConfig(configFromProto(req.Msg.GetConfig()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(configToProto(saved)), nil
}

func (h *connectHandler) GetLayout(_ context.Context, req *connect.Request[worldv1.GetLayoutRequest]) (*connect.Response[worldv1.WorldLayout], error) {
	layout, err := h.store.LoadLayout(req.Msg.GetScene())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(layoutToProto(layout)), nil
}

func (h *connectHandler) SetLayout(_ context.Context, req *connect.Request[worldv1.SetLayoutRequest]) (*connect.Response[worldv1.WorldLayout], error) {
	saved, err := h.store.SaveLayout(layoutFromProto(req.Msg.GetLayout()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(layoutToProto(saved)), nil
}

func (h *connectHandler) StreamWorldFeed(ctx context.Context, req *connect.Request[worldv1.StreamWorldFeedRequest], stream *connect.ServerStream[worldv1.WorldEvent]) error {
	if h.hub == nil {
		return connect.NewError(connect.CodeUnavailable, errors.New("world feed is not available"))
	}
	replay, live := h.hub.Subscribe(ctx, req.Msg.GetSinceSeq())
	if err := stream.Send(EventToProto(h.hub.Snapshot())); err != nil {
		return err
	}
	for _, event := range replay {
		if err := stream.Send(EventToProto(event)); err != nil {
			return err
		}
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-live:
			if !ok {
				return connect.NewError(connect.CodeResourceExhausted, errors.New("feed subscriber fell behind; reconnect with since_seq"))
			}
			if err := stream.Send(EventToProto(event)); err != nil {
				return err
			}
		}
	}
}
