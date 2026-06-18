package signals_test

import (
	"context"
	"testing"

	signalsh "architecture-cartographer/handlers/signals"
	"architecture-cartographer/internal/signals"
	"architecture-cartographer/internal/signals/mocks"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/graph"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/shared"
	signalsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/signals"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/signals/signals_v1connect"
)

func sampleVerdict() signals.Verdict {
	return signals.Verdict{
		ChunkID:   "chunk:f1",
		ChunkPath: "a/x.go",
		Tier:      signals.TierAutoPlace,
		TopDomain: "graph", TopValue: 0.92,
		RunnerUpDomain: "signals", RunnerUpValue: 0.31,
		DomainValues: []signals.DomainValue{{Domain: "graph", Value: 0.92}},
		Scores: []signals.Score{
			{Signal: "path-token", Domain: "graph", Value: 0.95, Reason: "exact dir match", Evidence: []signals.Evidence{{Kind: "path_token", Summary: "internal/graph"}}},
		},
	}
}

func TestHandler_ScoreChunk_TranslatesVerdict(t *testing.T) {
	svc := &mocks.FakeService{Verdict: sampleVerdict()}
	h := signalsh.NewHandler(svc)

	resp, err := h.ScoreChunk(context.Background(), connect.NewRequest(&signalsv1.ScoreChunkRequest{
		Scenario: "demo",
		Chunk:    &graphv1.Chunk{Id: "chunk:f1", FileId: "f1", Path: "a/x.go"},
	}))
	require.NoError(t, err)
	v := resp.Msg.GetVerdict()
	require.Equal(t, "chunk:f1", v.GetChunkId())
	require.Equal(t, sharedv1.Tier_TIER_AUTO_PLACE, v.GetTier())
	require.Equal(t, "graph", v.GetTopDomain())
	require.Len(t, v.GetScores(), 1)
	require.Equal(t, int64(1), svc.ScoreCalls.Load())
}

func TestHandler_ScoreChunk_RejectsMissingScenario(t *testing.T) {
	h := signalsh.NewHandler(&mocks.FakeService{})
	_, err := h.ScoreChunk(context.Background(), connect.NewRequest(&signalsv1.ScoreChunkRequest{
		Chunk: &graphv1.Chunk{Id: "x"},
	}))
	require.Error(t, err)
	var ce *connect.Error
	require.ErrorAs(t, err, &ce)
	require.Equal(t, connect.CodeInvalidArgument, ce.Code())
}

func TestHandler_ScoreChunk_RejectsNoChunkOrFileID(t *testing.T) {
	h := signalsh.NewHandler(&mocks.FakeService{})
	_, err := h.ScoreChunk(context.Background(), connect.NewRequest(&signalsv1.ScoreChunkRequest{Scenario: "demo"}))
	require.Error(t, err)
	var ce *connect.Error
	require.ErrorAs(t, err, &ce)
	require.Equal(t, connect.CodeInvalidArgument, ce.Code())
}

func TestHandler_ExplainVerdict_Delegates(t *testing.T) {
	svc := &mocks.FakeService{Verdict: sampleVerdict()}
	h := signalsh.NewHandler(svc)
	_, err := h.ExplainVerdict(context.Background(), connect.NewRequest(&signalsv1.ExplainVerdictRequest{
		Scenario: "demo", FileId: "f1",
	}))
	require.NoError(t, err)
	require.Equal(t, int64(1), svc.ExplainCalls.Load())
}

func TestHandler_ListSignals(t *testing.T) {
	svc := &mocks.FakeService{Signals: []signals.SignalDescriptor{
		{Name: "path-token", DefaultWeight: 1.5, Stability: "stable"},
		{Name: "import-cluster", DefaultWeight: 1.0, Disabled: true, DisabledReason: "unavailable"},
	}}
	h := signalsh.NewHandler(svc)
	resp, err := h.ListSignals(context.Background(), connect.NewRequest(&signalsv1.ListSignalsRequest{Scenario: "demo"}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetSignals(), 2)
	require.True(t, resp.Msg.GetSignals()[1].GetDisabled())
}

func TestHandler_InterfaceSatisfied(t *testing.T) {
	var _ signals_v1connect.SignalsServiceHandler = (*signalsh.Handler)(nil)
}
