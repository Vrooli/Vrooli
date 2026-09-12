package analytics

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	analyticsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/analytics"
	analyticsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/analytics/analytics_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "github.com/vrooli/cli-core/cliapptest"
)

type fakeService struct {
	analyticsconnect.UnimplementedAnalyticsServiceHandler

	mu          sync.Mutex
	eventsReqs  []*analyticsv1.ListEventsRequest
	eventsResp  *analyticsv1.ListEventsResponse
	statsResp   *analyticsv1.GetStatsResponse
	placeResp   *analyticsv1.ListPlacementsResponse
	overrideRsp *analyticsv1.RecordOverrideResponse
}

func (s *fakeService) ListEvents(_ context.Context, req *connect.Request[analyticsv1.ListEventsRequest]) (*connect.Response[analyticsv1.ListEventsResponse], error) {
	s.mu.Lock()
	s.eventsReqs = append(s.eventsReqs, req.Msg)
	s.mu.Unlock()
	return connect.NewResponse(s.eventsResp), nil
}

func (s *fakeService) GetStats(_ context.Context, _ *connect.Request[analyticsv1.GetStatsRequest]) (*connect.Response[analyticsv1.GetStatsResponse], error) {
	return connect.NewResponse(s.statsResp), nil
}

func (s *fakeService) ListPlacements(_ context.Context, _ *connect.Request[analyticsv1.ListPlacementsRequest]) (*connect.Response[analyticsv1.ListPlacementsResponse], error) {
	return connect.NewResponse(s.placeResp), nil
}

func (s *fakeService) RecordOverride(_ context.Context, _ *connect.Request[analyticsv1.RecordOverrideRequest]) (*connect.Response[analyticsv1.RecordOverrideResponse], error) {
	return connect.NewResponse(s.overrideRsp), nil
}

func connectAPI(t *testing.T, svc *fakeService) http.Handler {
	t.Helper()
	path, handler := analyticsconnect.NewAnalyticsServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func TestEvents_ParsesKindFilter(t *testing.T) {
	svc := &fakeService{eventsResp: &analyticsv1.ListEventsResponse{}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, eventsSchema(), cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"scenario": "demo"},
		Flags:       map[string]string{"kind": "conflict_detected,override_recorded"},
	})

	require.NoError(t, h.events(ctx))
	require.Len(t, svc.eventsReqs, 1)
	require.Equal(t, []analyticsv1.EventKind{
		analyticsv1.EventKind_EVENT_KIND_CONFLICT_DETECTED,
		analyticsv1.EventKind_EVENT_KIND_OVERRIDE_RECORDED,
	}, svc.eventsReqs[0].GetKinds())
}

// TestStats_SuppressesRateBelowThreshold is the plan's required guard:
// when GetStats reports the verdict-success rate is suppressed, the CLI
// shows the explanatory message and NOT a fabricated percentage.
func TestStats_SuppressesRateBelowThreshold(t *testing.T) {
	svc := &fakeService{statsResp: &analyticsv1.GetStatsResponse{Stats: &analyticsv1.StatsSummary{
		Scenario:                     "demo",
		ConflictsDetected:            3,
		VerdictSuccessRate:           0,
		VerdictSuccessRateSuppressed: true,
		VerdictObservationCount:      2,
	}}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, scenarioSchema(), cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"scenario": "demo"},
	})

	require.NoError(t, h.stats(ctx))
	body := out.String()
	require.Contains(t, body, "suppressed")
	require.Contains(t, body, "below the N>=5 threshold")
	require.NotContains(t, body, "0.0%")
	require.NotContains(t, body, "%)")
}

func TestStats_ShowsRateWhenNotSuppressed(t *testing.T) {
	svc := &fakeService{statsResp: &analyticsv1.GetStatsResponse{Stats: &analyticsv1.StatsSummary{
		Scenario:                "demo",
		VerdictSuccessRate:      0.8,
		VerdictObservationCount: 10,
	}}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, scenarioSchema(), cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"scenario": "demo"},
	})

	require.NoError(t, h.stats(ctx))
	body := out.String()
	require.Contains(t, body, "80.0%")
	require.Contains(t, body, "N=10")
}

func TestOverrideRecord_DryRunRendersNotPersisted(t *testing.T) {
	svc := &fakeService{overrideRsp: &analyticsv1.RecordOverrideResponse{
		Override: &analyticsv1.Override{Id: "o-1", Scenario: "demo", ChunkId: "f-1", VerdictDomain: "graph", ChosenDomain: "manifest"},
		DryRun:   true,
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, overrideSchema(), cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"scenario": "demo"},
		Flags:       map[string]string{"chunk-id": "f-1", "verdict-domain": "graph", "chosen-domain": "manifest"},
	})

	require.NoError(t, h.overrideRecord(ctx))
	require.Contains(t, out.String(), "dry-run: no row persisted")
}

func eventsSchema() cliapp.ArgSchema {
	return cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "scenario", Required: true}},
		Flags:       []cliapp.Flag{{Name: "kind"}, {Name: "since"}, {Name: "page-size"}, {Name: "page-token"}},
	}
}

func scenarioSchema() cliapp.ArgSchema {
	return cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true}}}
}

func overrideSchema() cliapp.ArgSchema {
	return cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "scenario", Required: true}},
		Flags: []cliapp.Flag{
			{Name: "chunk-id", Required: true},
			{Name: "verdict-domain", Required: true},
			{Name: "chosen-domain", Required: true},
			{Name: "note"},
			{Name: "verdict-event-id"},
			{Name: "idempotency-key"},
		},
	}
}
