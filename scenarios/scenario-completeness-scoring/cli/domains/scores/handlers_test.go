package scores

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	scoringv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-completeness-scoring/v1/scoring"
)

type scoreServiceStub struct {
	getRequest    *scoringv1.GetScoreRequest
	trendRequest  *scoringv1.GetScoreTrendRequest
	listRequest   *scoringv1.ListScoresRequest
	getResponse   *connect.Response[scoringv1.GetScoreResponse]
	trendResponse *connect.Response[scoringv1.GetScoreTrendResponse]
	listResponse  *connect.Response[scoringv1.ListScoresResponse]
	getError      error
	trendError    error
	listError     error
}

type failingScoreWriter struct{ err error }

func (w failingScoreWriter) Write([]byte) (int, error) { return 0, w.err }

func (s *scoreServiceStub) GetScore(_ context.Context, req *connect.Request[scoringv1.GetScoreRequest]) (*connect.Response[scoringv1.GetScoreResponse], error) {
	s.getRequest = req.Msg
	return s.getResponse, s.getError
}

func (s *scoreServiceStub) GetScoreTrend(_ context.Context, req *connect.Request[scoringv1.GetScoreTrendRequest]) (*connect.Response[scoringv1.GetScoreTrendResponse], error) {
	s.trendRequest = req.Msg
	return s.trendResponse, s.trendError
}

func (s *scoreServiceStub) ListScores(_ context.Context, req *connect.Request[scoringv1.ListScoresRequest]) (*connect.Response[scoringv1.ListScoresResponse], error) {
	s.listRequest = req.Msg
	return s.listResponse, s.listError
}

func scoreRunContext(t *testing.T, name string, args []string, stdout *bytes.Buffer) cliapp.RunContext {
	t.Helper()
	bindings := map[string]func(cliapp.RunContext) error{
		"ScoreService.GetScore":      func(cliapp.RunContext) error { return nil },
		"ScoreService.GetScoreTrend": func(cliapp.RunContext) error { return nil },
		"ScoreService.ListScores":    func(cliapp.RunContext) error { return nil },
	}
	group, err := cliapp.LoadFromManifest(cliapptest.ReadManifest(t), GroupName, bindings)
	require.NoError(t, err)
	command := cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, name)
	ctx, err := cliapptest.NewTestRunContextFromArgs(command.Args, args, nil, stdout, &bytes.Buffer{})
	require.NoError(t, err)
	return ctx
}

func TestScoreHandlersBuildRequestsFromManifestArguments(t *testing.T) {
	client := &scoreServiceStub{
		getResponse:   connect.NewResponse(&scoringv1.GetScoreResponse{Scenario: "cli-health"}),
		trendResponse: connect.NewResponse(&scoringv1.GetScoreTrendResponse{Scenario: "cli-health"}),
		listResponse:  connect.NewResponse(&scoringv1.ListScoresResponse{}),
	}
	h := &handlers{client: client}

	var out bytes.Buffer
	getCtx := scoreRunContext(t, "get", []string{"cli-health"}, &out)
	require.NoError(t, h.get(getCtx))
	require.Equal(t, "cli-health", client.getRequest.GetScenario())
	require.Equal(t, FormatReport(client.getResponse.Msg), out.String())

	out.Reset()
	trendCtx := scoreRunContext(t, "trend", []string{"cli-health", "--limit", "9"}, &out)
	require.NoError(t, h.trend(trendCtx))
	require.Equal(t, "cli-health", client.trendRequest.GetScenario())
	require.EqualValues(t, 9, client.trendRequest.GetLimit())
	require.Equal(t, FormatTrend(client.trendResponse.Msg), out.String())

	out.Reset()
	listCtx := scoreRunContext(t, "list", []string{"--sort", "priority", "--order", "asc", "--page_size", "40", "--page-token", "next", "--rung", "R3", "--min-score", "20", "--max-score", "89", "--category", "utility", "--recompute"}, &out)
	require.NoError(t, h.list(listCtx))
	request := client.listRequest
	require.Equal(t, scoringv1.ScoreSortBy_SCORE_SORT_BY_PRIORITY, request.GetSortBy())
	require.Equal(t, scoringv1.SortOrder_SORT_ORDER_ASC, request.GetOrder())
	require.EqualValues(t, 40, request.GetPageSize())
	require.Equal(t, "next", request.GetPageToken())
	require.Equal(t, "R3", request.GetRung())
	require.Equal(t, int32(20), request.GetMinScore())
	require.Equal(t, int32(89), request.GetMaxScore())
	require.Equal(t, "utility", request.GetCategory())
	require.True(t, request.GetRecompute())
	require.Equal(t, FormatList(client.listResponse.Msg), out.String())
}

func TestScoreHandlersUseSharedJSONRenderer(t *testing.T) {
	response := &scoringv1.GetScoreResponse{Scenario: "cli-health"}
	client := &scoreServiceStub{getResponse: connect.NewResponse(response)}
	h := &handlers{client: client}
	var out bytes.Buffer
	ctx, err := cliapptest.NewTestRunContextFromArgs(
		cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true}}},
		[]string{"cli-health", "--json"}, nil, &out, &bytes.Buffer{},
	)
	require.NoError(t, err)
	require.NoError(t, h.get(ctx))
	var got map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	require.Equal(t, "cli-health", got["scenario"])
	require.NotContains(t, out.String(), "MATURITY")
}

func TestRenderScoreResponseDoesNotFormatHumanOutputForJSON(t *testing.T) {
	response := &scoringv1.GetScoreResponse{Scenario: "cli-health"}
	var out bytes.Buffer
	ctx := cliapptest.NewTestRunContext(cliapptest.TestRunContextOptions{JSON: true, Stdout: &out})
	humanCalled := false
	err := renderScoreResponse(ctx, response, func(*scoringv1.GetScoreResponse) string {
		humanCalled = true
		return "human report"
	})
	require.NoError(t, err)
	require.False(t, humanCalled)
	var got map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	require.Equal(t, "cli-health", got["scenario"])
}

func TestRenderScoreResponsePropagatesWriterErrors(t *testing.T) {
	want := errors.New("closed output")
	ctx := cliapptest.NewTestRunContext(cliapptest.TestRunContextOptions{Stdout: failingScoreWriter{err: want}})
	err := renderScoreResponse(ctx, &scoringv1.GetScoreResponse{}, func(*scoringv1.GetScoreResponse) string { return "report" })
	require.ErrorIs(t, err, want)
}

func TestScoreHandlersPreserveAPIAndNilResponseErrors(t *testing.T) {
	apiFailure := errors.New("service unavailable")
	cases := []struct {
		name    string
		command string
		invoke  func(*handlers, cliapp.RunContext) error
		client  *scoreServiceStub
		args    []string
		want    string
	}{
		{"get api error", "get", (*handlers).get, &scoreServiceStub{getError: apiFailure}, []string{"cli-health"}, "get score"},
		{"trend api error", "trend", (*handlers).trend, &scoreServiceStub{trendError: apiFailure}, []string{"cli-health"}, "get score trend"},
		{"list api error", "list", (*handlers).list, &scoreServiceStub{listError: apiFailure}, nil, "list scores"},
		{"get nil response", "get", (*handlers).get, &scoreServiceStub{}, []string{"cli-health"}, "server returned no score response"},
		{"trend nil response", "trend", (*handlers).trend, &scoreServiceStub{}, []string{"cli-health"}, "server returned no score trend response"},
		{"list nil response", "list", (*handlers).list, &scoreServiceStub{}, nil, "server returned no score list response"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := &handlers{client: tc.client}
			ctx := scoreRunContext(t, tc.command, tc.args, &bytes.Buffer{})
			err := tc.invoke(h, ctx)
			require.Error(t, err)
			require.True(t, strings.Contains(err.Error(), tc.want), "error = %v, want %q", err, tc.want)
		})
	}
}
