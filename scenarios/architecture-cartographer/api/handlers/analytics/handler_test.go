package analytics_test

import (
	"context"
	"testing"

	analyticsh "architecture-cartographer/handlers/analytics"
	"architecture-cartographer/internal/analytics"
	"architecture-cartographer/internal/analytics/mocks"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	analyticsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/analytics"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/analytics/analytics_v1connect"
)

func TestHandler_ListEvents_FiltersKinds(t *testing.T) {
	svc := &mocks.FakeService{Events: []analytics.Event{
		{ID: "e1", Kind: analytics.EventKindConflictDetected, Scenario: "demo"},
	}}
	h := analyticsh.NewHandler(svc)
	resp, err := h.ListEvents(context.Background(), connect.NewRequest(&analyticsv1.ListEventsRequest{
		Scenario: "demo",
		Kinds:    []analyticsv1.EventKind{analyticsv1.EventKind_EVENT_KIND_CONFLICT_DETECTED},
	}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetEvents(), 1)
	require.Equal(t, analyticsv1.EventKind_EVENT_KIND_CONFLICT_DETECTED, resp.Msg.GetEvents()[0].GetKind())
	require.Equal(t, int64(1), svc.ListEventsCalls.Load())
}

func TestHandler_GetStats_RejectsMissingScenario(t *testing.T) {
	h := analyticsh.NewHandler(&mocks.FakeService{})
	_, err := h.GetStats(context.Background(), connect.NewRequest(&analyticsv1.GetStatsRequest{}))
	require.Error(t, err)
	var ce *connect.Error
	require.ErrorAs(t, err, &ce)
	require.Equal(t, connect.CodeInvalidArgument, ce.Code())
}

func TestHandler_GetStats_PassthroughIncludingSuppression(t *testing.T) {
	svc := &mocks.FakeService{Summary: analytics.StatsSummary{
		Scenario:                     "demo",
		ConflictsDetected:            7,
		VerdictSuccessRate:           0,
		VerdictSuccessRateSuppressed: true,
		VerdictObservationCount:      3,
	}}
	h := analyticsh.NewHandler(svc)
	resp, err := h.GetStats(context.Background(), connect.NewRequest(&analyticsv1.GetStatsRequest{Scenario: "demo"}))
	require.NoError(t, err)
	stats := resp.Msg.GetStats()
	require.Equal(t, int64(7), stats.GetConflictsDetected())
	require.True(t, stats.GetVerdictSuccessRateSuppressed(), "suppression must round-trip")
}

func TestHandler_RecordOverride_HappyPath(t *testing.T) {
	svc := &mocks.FakeService{}
	h := analyticsh.NewHandler(svc)
	resp, err := h.RecordOverride(context.Background(), connect.NewRequest(&analyticsv1.RecordOverrideRequest{
		Scenario:      "demo",
		ChunkId:       "chunk:f1",
		VerdictDomain: "graph",
		ChosenDomain:  "signals",
		Note:          "test",
	}))
	require.NoError(t, err)
	require.False(t, resp.Msg.GetDryRun())
	require.Equal(t, "demo", resp.Msg.GetOverride().GetScenario())
	require.Equal(t, int64(1), svc.RecordOverrideCalls.Load())
}

func TestHandler_RecordOverride_DryRunDoesNotPersist(t *testing.T) {
	svc := &mocks.FakeService{}
	h := analyticsh.NewHandler(svc)
	resp, err := h.RecordOverride(context.Background(), connect.NewRequest(&analyticsv1.RecordOverrideRequest{
		Scenario: "demo", ChunkId: "chunk", VerdictDomain: "a", ChosenDomain: "b",
		DryRun: true,
	}))
	require.NoError(t, err)
	require.True(t, resp.Msg.GetDryRun())
	require.Equal(t, int64(0), svc.RecordOverrideCalls.Load(), "dry-run must not call service")
}

func TestHandler_RecordOverride_HeaderDryRun(t *testing.T) {
	svc := &mocks.FakeService{}
	h := analyticsh.NewHandler(svc)
	req := connect.NewRequest(&analyticsv1.RecordOverrideRequest{
		Scenario: "demo", ChunkId: "chunk", VerdictDomain: "a", ChosenDomain: "b",
	})
	req.Header().Set("X-Dry-Run", "true")
	resp, err := h.RecordOverride(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.Msg.GetDryRun())
	require.Equal(t, int64(0), svc.RecordOverrideCalls.Load())
}

func TestHandler_ListPlacements_PassesOutcomes(t *testing.T) {
	svc := &mocks.FakeService{}
	h := analyticsh.NewHandler(svc)
	_, err := h.ListPlacements(context.Background(), connect.NewRequest(&analyticsv1.ListPlacementsRequest{
		Scenario: "demo", Outcomes: []string{"auto_placed"},
	}))
	require.NoError(t, err)
	require.Equal(t, int64(1), svc.ListPlacementsCalls.Load())
}

func TestHandler_InterfaceSatisfied(t *testing.T) {
	var _ analytics_v1connect.AnalyticsServiceHandler = (*analyticsh.Handler)(nil)
}
