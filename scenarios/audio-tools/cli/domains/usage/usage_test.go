package usage

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	"google.golang.org/protobuf/types/known/timestamppb"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	usagev1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/usage"
	usageconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/usage/usage_v1connect"

	testutil "github.com/vrooli/cli-core/cliapptest"
)

type fakeSvc struct {
	usageconnect.UnimplementedUsageServiceHandler
	list    func(*usagev1.ListRecentRequest) ([]*usagev1.UsageRow, error)
	summary func(*usagev1.GetSummaryRequest) (*usagev1.Summary, error)
}

func (f *fakeSvc) ListRecent(_ context.Context, req *connect.Request[usagev1.ListRecentRequest]) (*connect.Response[usagev1.ListRecentResponse], error) {
	rows, err := f.list(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&usagev1.ListRecentResponse{Rows: rows}), nil
}

func (f *fakeSvc) GetSummary(_ context.Context, req *connect.Request[usagev1.GetSummaryRequest]) (*connect.Response[usagev1.GetSummaryResponse], error) {
	s, err := f.summary(req.Msg)
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
				{EmittedAt: timestamppb.New(parseTime("2026-05-16T00:00:00Z")), Capability: "stt", Operation: "transcribe", ProviderTier: commonv1.ProviderTier_PROVIDER_TIER_LOCAL, ProviderId: "whisper", LatencyMs: 200},
			}, nil
		},
	})
	h := newHandlers(app)
	assertBothModes(t, app, h.list, "2026-05-16T00:00:00Z  stt        transcribe   local/whisper  200ms  err=\"\"\n")
}

// Happy path: empty list prints the "(no usage rows)" sentinel.
func TestListEmpty(t *testing.T) {
	app := mountUsage(t, &fakeSvc{
		list: func(_ *usagev1.ListRecentRequest) ([]*usagev1.UsageRow, error) { return nil, nil },
	})
	h := newHandlers(app)
	assertBothModes(t, app, h.list, "(no usage rows)\n")
}

// Error path: summary RPC fails — handler surfaces wrapped error.
func TestSummaryError(t *testing.T) {
	app := mountUsage(t, &fakeSvc{
		summary: func(*usagev1.GetSummaryRequest) (*usagev1.Summary, error) {
			return nil, connect.NewError(connect.CodeInternal, errors.New("db down"))
		},
	})
	h := newHandlers(app)
	ctx, _ := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	err := h.summary(ctx)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "usage summary"), "want operation tag, got %q", err.Error())
}

func TestSummaryOutputModes(t *testing.T) {
	app := mountUsage(t, &fakeSvc{
		summary: func(req *usagev1.GetSummaryRequest) (*usagev1.Summary, error) {
			require.EqualValues(t, 86400, req.GetSinceSeconds())
			return &usagev1.Summary{
				OperationsTotal: 3,
				CreditsTotal:    42,
				ErrorCount:      1,
				Distribution: []*usagev1.ProviderDistribution{{
					ProviderTier: commonv1.ProviderTier_PROVIDER_TIER_LOCAL,
					ProviderId:   "whisper", Count: 2, CreditsTotal: 30, AvgLatencyMs: 100.5,
				}},
			}, nil
		},
	})
	h := newHandlers(app)
	want := "Operations: 3\nCredits:    42\nErrors:     1\n\nProvider distribution:\n  local/whisper       count=2 credits=30 avg_ms=100.5\n"
	assertBothModes(t, app, h.summary, want)
}

func assertBothModes(t *testing.T, app *cliapp.ScenarioApp, run func(cliapp.RunContext) error, want string) {
	t.Helper()
	for _, jsonMode := range []bool{false, true} {
		ctx, buf := cliapptest.NewCapturedRunContext(app, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{JSON: jsonMode})
		require.NoError(t, run(ctx))
		require.Equal(t, want, buf.String())
	}
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}
