package themes

import (
	"context"
	"log"

	"connectrpc.com/connect"

	"react-component-library/internal/themes"

	themesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/themes"
)

type Deps struct {
	Service themes.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ListBuiltinThemes(ctx context.Context, _ *connect.Request[themesv1.ListBuiltinThemesRequest]) (*connect.Response[themesv1.ListBuiltinThemesResponse], error) {
	out, err := h.deps.Service.ListBuiltins(ctx)
	if err != nil {
		h.deps.Logger.Printf("themes.ListBuiltinThemes: %v", err)
		return nil, themes.ToConnectError(err)
	}
	resp := &themesv1.ListBuiltinThemesResponse{Themes: make([]*themesv1.Theme, 0, len(out))}
	for _, t := range out {
		resp.Themes = append(resp.Themes, themeToProto(t))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetBuiltinTheme(ctx context.Context, req *connect.Request[themesv1.GetBuiltinThemeRequest]) (*connect.Response[themesv1.GetBuiltinThemeResponse], error) {
	t, err := h.deps.Service.GetBuiltin(ctx, req.Msg.Id)
	if err != nil {
		connectErr := themes.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("themes.GetBuiltinTheme(%q): %v", req.Msg.Id, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&themesv1.GetBuiltinThemeResponse{Theme: themeToProto(t)}), nil
}

func (h *connectHandler) GetThemeFromScenario(ctx context.Context, req *connect.Request[themesv1.GetThemeFromScenarioRequest]) (*connect.Response[themesv1.GetThemeFromScenarioResponse], error) {
	t, err := h.deps.Service.ResolveFromScenario(ctx, req.Msg.ScenarioId)
	if err != nil {
		connectErr := themes.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("themes.GetThemeFromScenario(%q): %v", req.Msg.ScenarioId, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&themesv1.GetThemeFromScenarioResponse{Theme: themeToProto(t)}), nil
}

func themeToProto(t themes.Theme) *themesv1.Theme {
	tokens := make(map[string]string, len(t.Tokens))
	for k, v := range t.Tokens {
		tokens[k] = v
	}
	return &themesv1.Theme{
		Id:     t.ID,
		Name:   t.Name,
		Tokens: tokens,
		Source: t.Source,
	}
}
