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

func (h *connectHandler) GetEffective(_ context.Context, _ *connect.Request[shortcutsv1.GetEffectiveRequest]) (*connect.Response[shortcutsv1.GetEffectiveResponse], error) {
	out := h.deps.Service.Effective()
	return connect.NewResponse(&shortcutsv1.GetEffectiveResponse{
		Shortcuts: shortcutsToProto(out),
	}), nil
}

func (h *connectHandler) ListProfiles(_ context.Context, _ *connect.Request[shortcutsv1.ListProfilesRequest]) (*connect.Response[shortcutsv1.ListProfilesResponse], error) {
	profiles := h.deps.Service.List()
	pp := make([]*shortcutsv1.Profile, 0, len(profiles))
	for _, p := range profiles {
		pp = append(pp, profileToProto(p))
	}
	return connect.NewResponse(&shortcutsv1.ListProfilesResponse{Profiles: pp}), nil
}

func (h *connectHandler) UpsertProfile(_ context.Context, req *connect.Request[shortcutsv1.UpsertProfileRequest]) (*connect.Response[shortcutsv1.UpsertProfileResponse], error) {
	in := UpsertRequest{
		ID:        req.Msg.GetId(),
		Scope:     req.Msg.GetScope(),
		Name:      req.Msg.GetName(),
		Shortcuts: shortcutsFromProto(req.Msg.GetShortcuts()),
	}
	p, err := h.deps.Service.Upsert(in)
	if err != nil {
		if errors.Is(err, ErrInvalidArgument) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		h.deps.Logger.Printf("shortcuts.UpsertProfile: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&shortcutsv1.UpsertProfileResponse{Profile: profileToProto(p)}), nil
}

func (h *connectHandler) DeleteProfile(_ context.Context, req *connect.Request[shortcutsv1.DeleteProfileRequest]) (*connect.Response[shortcutsv1.DeleteProfileResponse], error) {
	h.deps.Service.Delete(req.Msg.GetId())
	return connect.NewResponse(&shortcutsv1.DeleteProfileResponse{}), nil
}

func shortcutsToProto(in []Shortcut) []*shortcutsv1.Shortcut {
	out := make([]*shortcutsv1.Shortcut, 0, len(in))
	for _, s := range in {
		out = append(out, &shortcutsv1.Shortcut{
			Label:       s.Label,
			Command:     s.Command,
			Description: s.Description,
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
