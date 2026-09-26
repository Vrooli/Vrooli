package summarize

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

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	summv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"
	summconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize/summarize_v1connect"

	testutil "github.com/vrooli/cli-core/cliapptest"
)

type fakeSvc struct {
	summconnect.UnimplementedSummarizeServiceHandler
	fn func(*summv1.SummarizeRequest) (*summv1.SummarizeResponse, error)
}

func (f *fakeSvc) Summarize(_ context.Context, req *connect.Request[summv1.SummarizeRequest]) (*connect.Response[summv1.SummarizeResponse], error) {
	resp, err := f.fn(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func mountSummarize(t *testing.T, svc summconnect.SummarizeServiceHandler) *cliapp.ScenarioApp {
	t.Helper()
	path, h := summconnect.NewSummarizeServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, h)
	return testutil.NewTestApp(t, mux)
}

// Happy path: default level "moderate" is filled in when --level is
// omitted; result line includes provider trace + body.
func TestTextDefaultsLevel(t *testing.T) {
	app := mountSummarize(t, &fakeSvc{
		fn: func(req *summv1.SummarizeRequest) (*summv1.SummarizeResponse, error) {
			require.Equal(t, summv1.SummarizeLevel_SUMMARIZE_LEVEL_MODERATE, req.GetLevel())
			require.Equal(t, "hello world", req.GetText())
			return &summv1.SummarizeResponse{
				Text: "summary", ProviderTier: commonv1.ProviderTier_PROVIDER_TIER_LOCAL, ProviderId: "ollama", LatencyMs: 42,
			}, nil
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "text"}, {Name: "level"}}}
	ctx, buf := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"text": "hello world"},
	})
	require.NoError(t, h.text(ctx))
	out := buf.String()
	require.Contains(t, out, "[local/ollama, 42ms] summary")
}

// Error path: provider returns Unavailable — handler surfaces wrapped
// "summarize" error.
func TestTextProviderUnavailable(t *testing.T) {
	app := mountSummarize(t, &fakeSvc{
		fn: func(_ *summv1.SummarizeRequest) (*summv1.SummarizeResponse, error) {
			return nil, connect.NewError(connect.CodeUnavailable, errors.New("ollama down"))
		},
	})
	h := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "text"}, {Name: "level"}}}
	ctx, _ := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"text": "x", "level": "heavy"},
	})
	err := h.text(ctx)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "summarize"), "want operation tag, got %q", err.Error())
}
