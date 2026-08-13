package settings

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "github.com/vrooli/cli-core/cliapptest"

	settingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/settings"
	settingsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/settings/settings_v1connect"
)

type fakeService struct {
	mu        sync.Mutex
	stored    *settingsv1.Settings
	updates   []*settingsv1.UpdateSettingsRequest
	getErr    error
	updateErr error
}

func (f *fakeService) GetSettings(_ context.Context, _ *connect.Request[settingsv1.GetSettingsRequest]) (*connect.Response[settingsv1.GetSettingsResponse], error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.stored == nil {
		f.stored = &settingsv1.Settings{Theme: settingsv1.Theme_THEME_SYSTEM, FontScale: settingsv1.FontScale_FONT_SCALE_MD}
	}
	return connect.NewResponse(&settingsv1.GetSettingsResponse{Settings: f.stored}), nil
}

func (f *fakeService) UpdateSettings(_ context.Context, req *connect.Request[settingsv1.UpdateSettingsRequest]) (*connect.Response[settingsv1.UpdateSettingsResponse], error) {
	f.mu.Lock()
	f.updates = append(f.updates, req.Msg)
	f.mu.Unlock()
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	f.stored = req.Msg.Settings
	return connect.NewResponse(&settingsv1.UpdateSettingsResponse{Settings: f.stored}), nil
}

func connectAPI(t *testing.T, svc *fakeService) http.Handler {
	t.Helper()
	path, handler := settingsconnect.NewSettingsServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func TestSettingsGet_RendersResults(t *testing.T) {
	core := clitest.NewTestApp(t, connectAPI(t, &fakeService{}))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})

	require.NoError(t, h.get(ctx))
	require.Contains(t, out.String(), "theme")
	require.Contains(t, out.String(), "system")
}

func TestSettingsSet_AppliesMask(t *testing.T) {
	svc := &fakeService{}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "pair", Required: true, Repeated: true}}}
	ctx, _ := cliapptest.NewCapturedRunContext(core, schema, cliapptest.TestRunContextOptions{Repeated: map[string][]string{"pair": {"theme=dark"}}})

	require.NoError(t, h.set(ctx))
	require.Len(t, svc.updates, 1)
	require.Equal(t, []string{"theme"}, svc.updates[0].UpdateMask.Paths)
	require.Equal(t, settingsv1.Theme_THEME_DARK, svc.updates[0].Settings.Theme)
}

func TestSettingsSet_UnknownKeyReturnsError(t *testing.T) {
	core := clitest.NewTestApp(t, connectAPI(t, &fakeService{}))
	h := newHandlers(core)
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "pair", Required: true, Repeated: true}}}
	ctx, _ := cliapptest.NewCapturedRunContext(core, schema, cliapptest.TestRunContextOptions{Repeated: map[string][]string{"pair": {"bogus=value"}}})

	err := h.set(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown setting")
}
