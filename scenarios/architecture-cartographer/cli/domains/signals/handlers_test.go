package signals

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/shared"
	signalsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/signals"
	signalsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/signals/signals_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "github.com/vrooli/cli-core/cliapptest"
)

type fakeService struct {
	signalsconnect.UnimplementedSignalsServiceHandler

	mu          sync.Mutex
	scoreReqs   []*signalsv1.ScoreChunkRequest
	scoreResp   *signalsv1.ScoreChunkResponse
	explainResp *signalsv1.ExplainVerdictResponse
	listResp    *signalsv1.ListSignalsResponse
}

func (s *fakeService) ScoreChunk(_ context.Context, req *connect.Request[signalsv1.ScoreChunkRequest]) (*connect.Response[signalsv1.ScoreChunkResponse], error) {
	s.mu.Lock()
	s.scoreReqs = append(s.scoreReqs, req.Msg)
	s.mu.Unlock()
	return connect.NewResponse(s.scoreResp), nil
}

func (s *fakeService) ExplainVerdict(_ context.Context, _ *connect.Request[signalsv1.ExplainVerdictRequest]) (*connect.Response[signalsv1.ExplainVerdictResponse], error) {
	return connect.NewResponse(s.explainResp), nil
}

func (s *fakeService) ListSignals(_ context.Context, _ *connect.Request[signalsv1.ListSignalsRequest]) (*connect.Response[signalsv1.ListSignalsResponse], error) {
	return connect.NewResponse(s.listResp), nil
}

func connectAPI(t *testing.T, svc *fakeService) http.Handler {
	t.Helper()
	path, handler := signalsconnect.NewSignalsServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func sampleVerdict() *sharedv1.Verdict {
	return &sharedv1.Verdict{
		ChunkId:        "f-1",
		ChunkPath:      "api/internal/graph/x.go",
		Tier:           sharedv1.Tier_TIER_AUTO_PLACE,
		TopDomain:      "graph",
		TopValue:       0.91,
		RunnerUpDomain: "manifest",
		RunnerUpValue:  0.12,
		DomainValues:   []*sharedv1.DomainValue{{Domain: "graph", Value: 0.91}},
		Scores: []*sharedv1.Score{
			{Signal: "path-token", Domain: "graph", Value: 0.8, Reason: "matched 'graph'", Evidence: []*sharedv1.Evidence{
				{Kind: "path_token", Summary: "token graph in path", Weight: 0.8, Locator: "api/internal/graph"},
			}},
		},
	}
}

func TestScore_RendersVerdictWithoutEvidence(t *testing.T) {
	svc := &fakeService{scoreResp: &signalsv1.ScoreChunkResponse{Verdict: sampleVerdict()}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, scoreSchema(), cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"scenario": "demo", "file": "file:f-1"},
	})

	require.NoError(t, h.score(ctx))
	require.Len(t, svc.scoreReqs, 1)
	require.Equal(t, "file:f-1", svc.scoreReqs[0].GetFileId())
	body := out.String()
	require.Contains(t, body, "tier=auto_place")
	require.Contains(t, body, "top=graph")
	require.Contains(t, body, "signal path-token")
	require.NotContains(t, body, "evidence[path_token]")
}

func TestExplain_RendersEvidence(t *testing.T) {
	svc := &fakeService{explainResp: &signalsv1.ExplainVerdictResponse{Verdict: sampleVerdict()}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, scoreSchema(), cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"scenario": "demo", "file": "file:f-1"},
	})

	require.NoError(t, h.explain(ctx))
	require.Contains(t, out.String(), "evidence[path_token]")
}

func TestList_RendersDisabledState(t *testing.T) {
	svc := &fakeService{listResp: &signalsv1.ListSignalsResponse{
		Signals: []*signalsv1.SignalDescriptor{
			{Name: "git-co-edit", DefaultWeight: 0.5, Stability: "beta", Disabled: true, DisabledReason: "git unavailable"},
		},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "scenario"}},
	}, cliapptest.TestRunContextOptions{})

	require.NoError(t, h.list(ctx))
	body := out.String()
	require.Contains(t, body, "git-co-edit")
	require.Contains(t, body, "disabled (git unavailable)")
}

func scoreSchema() cliapp.ArgSchema {
	return cliapp.ArgSchema{
		Positionals: []cliapp.Positional{
			{Name: "scenario", Required: true},
			{Name: "file", Required: true},
		},
	}
}
