package usage

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	usagev1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/usage"
	usageconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/usage/usage_v1connect"

	"audio-tools/cli/internal/testutil"
)

type fakeSvc struct {
	usageconnect.UnimplementedUsageServiceHandler
	list    func(*usagev1.ListRecentRequest) ([]*usagev1.UsageRow, error)
	summary func() (*usagev1.Summary, error)
}

func (f *fakeSvc) ListRecent(_ context.Context, req *connect.Request[usagev1.ListRecentRequest]) (*connect.Response[usagev1.ListRecentResponse], error) {
	rows, err := f.list(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&usagev1.ListRecentResponse{Rows: rows}), nil
}

func (f *fakeSvc) GetSummary(_ context.Context, _ *connect.Request[usagev1.GetSummaryRequest]) (*connect.Response[usagev1.GetSummaryResponse], error) {
	s, err := f.summary()
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&usagev1.GetSummaryResponse{Summary: s}), nil
}

func mountUsage(t *testing.T, svc usageconnect.UsageServiceHandler) *cliapp.ScenarioApp {
	t.Helper()
	path, h := usageconnect.NewUsageServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, h)
	return testutil.NewTestApp(t, mux)
}

// Happy path: list renders one line per row including provider trace.
func TestListWithRows(t *testing.T) {
	app := mountUsage(t, &fakeSvc{
		list: func(req *usagev1.ListRecentRequest) ([]*usagev1.UsageRow, error) {
			require.EqualValues(t, 86400, req.GetSinceSeconds())
			require.EqualValues(t, 50, req.GetLimit())
			return []*usagev1.UsageRow{
				{EmittedAt: "2026-05-16T00:00:00Z", Capability: "stt", Operation: "transcribe", ProviderTier: "local", ProviderId: "whisper", LatencyMs: 200},
			}, nil
		},
	})
	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	require.NoError(t, h.list(ctx))
	out := buf.String()
	require.Contains(t, out, "transcribe")
	require.Contains(t, out, "local/whisper")
}

// Happy path: empty list prints the "(no usage rows)" sentinel.
func TestListEmpty(t *testing.T) {
	app := mountUsage(t, &fakeSvc{
		list: func(_ *usagev1.ListRecentRequest) ([]*usagev1.UsageRow, error) { return nil, nil },
	})
	h := newHandlers(app)
	ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	require.NoError(t, h.list(ctx))
	require.Contains(t, buf.String(), "(no usage rows)")
}

// Error path: summary RPC fails — handler surfaces wrapped error.
func TestSummaryError(t *testing.T) {
	app := mountUsage(t, &fakeSvc{
		summary: func() (*usagev1.Summary, error) {
			return nil, connect.NewError(connect.CodeInternal, errors.New("db down"))
		},
	})
	h := newHandlers(app)
	ctx, _ := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	err := h.summary(ctx)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "usage summary"), "want operation tag, got %q", err.Error())
}
