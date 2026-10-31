package shortcuts

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	shortcutsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/shortcuts"
)

// Deps wires the seams the Connect shortcuts handler needs.
type Deps struct {
	Service Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect-RPC handler implementing
// ShortcutsServiceHandler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

// ErrInvalidArgument is the sentinel the Service implementation returns
// for caller-visible validation failures (unknown scope, blank name,
// shortcut entry missing label/command). Mapped to CodeInvalidArgument.
var ErrInvalidArgument = errors.New("invalid argument")
var ErrConflict = errors.New("preferences conflict")

func (h *connectHandler) GetAgentPreferences(ctx context.Context, _ *connect.Request[shortcutsv1.GetAgentPreferencesRequest]) (*connect.Response[shortcutsv1.GetAgentPreferencesResponse], error) {
	p, err := h.deps.Service.AgentPreferences(ctx)
	if err != nil {
		h.deps.Logger.Printf("shortcuts.GetAgentPreferences: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("could not read agent preferences"))
	}
	return connect.NewResponse(&shortcutsv1.GetAgentPreferencesResponse{Preferences: preferencesToProto(p)}), nil
}

func (h *connectHandler) UpdateAgentPreferences(ctx context.Context, req *connect.Request[shortcutsv1.UpdateAgentPreferencesRequest]) (*connect.Response[shortcutsv1.UpdateAgentPreferencesResponse], error) {
	p, err := h.deps.Service.UpdateAgentPreferences(ctx, AgentPreferences{ProfileID: req.Msg.GetProfileId(), ProfileInstance: req.Msg.GetProfileInstance(), OrderedAgentIDs: req.Msg.GetOrderedAgentIds(), HiddenAgentIDs: req.Msg.GetHiddenAgentIds()}, req.Msg.GetExpectedRevision())
	if err != nil {
		if errors.Is(err, ErrConflict) {
			return nil, connect.NewError(connect.CodeAborted, err)
		}
		if errors.Is(err, ErrInvalidArgument) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		h.deps.Logger.Printf("shortcuts.UpdateAgentPreferences: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("could not save agent preferences"))
	}
	return connect.NewResponse(&shortcutsv1.UpdateAgentPreferencesResponse{Preferences: preferencesToProto(p)}), nil
}

func preferencesToProto(p AgentPreferences) *shortcutsv1.AgentPreferences {
	return &shortcutsv1.AgentPreferences{ProfileId: p.ProfileID, ProfileInstance: p.ProfileInstance, OrderedAgentIds: p.OrderedAgentIDs, HiddenAgentIds: p.HiddenAgentIDs, Revision: p.Revision}
}

func (h *connectHandler) GetEffective(ctx context.Context, _ *connect.Request[shortcutsv1.GetEffectiveRequest]) (*connect.Response[shortcutsv1.GetEffectiveResponse], error) {
	out := h.deps.Service.Effective(ctx)
	return connect.NewResponse(&shortcutsv1.GetEffectiveResponse{
		Shortcuts:   shortcutsToProto(out.Shortcuts),
		ProfileId:   out.ProfileID,
		Scope:       out.Scope,
		ProfileName: out.Name,
	}), nil
}

func (h *connectHandler) ListProfiles(ctx context.Context, _ *connect.Request[shortcutsv1.ListProfilesRequest]) (*connect.Response[shortcutsv1.ListProfilesResponse], error) {
	profiles := h.deps.Service.List(ctx)
	pp := make([]*shortcutsv1.Profile, 0, len(profiles))
	for _, p := range profiles {
		pp = append(pp, profileToProto(p))
	}
	return connect.NewResponse(&shortcutsv1.ListProfilesResponse{Profiles: pp}), nil
}

func (h *connectHandler) UpsertProfile(ctx context.Context, req *connect.Request[shortcutsv1.UpsertProfileRequest]) (*connect.Response[shortcutsv1.UpsertProfileResponse], error) {
	in := UpsertRequest{
		ID:        req.Msg.GetId(),
		Scope:     req.Msg.GetScope(),
		Name:      req.Msg.GetName(),
		Shortcuts: shortcutsFromProto(req.Msg.GetShortcuts()),
	}
	p, err := h.deps.Service.Upsert(ctx, in)
	if err != nil {
		if errors.Is(err, ErrInvalidArgument) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		h.deps.Logger.Printf("shortcuts.UpsertProfile: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&shortcutsv1.UpsertProfileResponse{Profile: profileToProto(p)}), nil
}

func (h *connectHandler) DeleteProfile(ctx context.Context, req *connect.Request[shortcutsv1.DeleteProfileRequest]) (*connect.Response[shortcutsv1.DeleteProfileResponse], error) {
	h.deps.Service.Delete(ctx, req.Msg.GetId())
	return connect.NewResponse(&shortcutsv1.DeleteProfileResponse{}), nil
}

func shortcutsToProto(in []Shortcut) []*shortcutsv1.Shortcut {
	out := make([]*shortcutsv1.Shortcut, 0, len(in))
	for _, s := range in {
		out = append(out, &shortcutsv1.Shortcut{
			Label:       s.Label,
			Command:     s.Command,
			Description: s.Description,
			AgentId:     s.AgentID,
		})
	}
	return out
}

func shortcutsFromProto(in []*shortcutsv1.Shortcut) []Shortcut {
	out := make([]Shortcut, 0, len(in))
	for _, s := range in {
		out = append(out, Shortcut{
			Label:       s.GetLabel(),
			Command:     s.GetCommand(),
			Description: s.GetDescription(),
			AgentID:     s.GetAgentId(),
		})
	}
	return out
}

func profileToProto(p Profile) *shortcutsv1.Profile {
	return &shortcutsv1.Profile{
		Id:        p.ID,
		Scope:     p.Scope,
		Name:      p.Name,
		Shortcuts: shortcutsToProto(p.Shortcuts),
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}
