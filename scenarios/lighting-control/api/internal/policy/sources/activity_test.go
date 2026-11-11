package sources

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/policy"
	"google.golang.org/protobuf/types/known/timestamppb"
	"lighting-control/internal/policy"
)

type activityServer struct {
	apiconnect.UnimplementedAgentManagerServiceHandler
	response *apipb.GetRuntimeActivityResponse
	err      error
}

func (a activityServer) GetRuntimeActivity(context.Context, *connect.Request[apipb.GetRuntimeActivityRequest]) (*connect.Response[apipb.GetRuntimeActivityResponse], error) {
	if a.err != nil {
		return nil, a.err
	}
	return connect.NewResponse(a.response), nil
}
func TestTypedPhysicalActivityReadNeverInfersExecutingFromQueueOrReceipt(t *testing.T) {
	now := time.Date(2026, 11, 11, 18, 0, 0, 0, time.UTC)
	zero, one := uint32(0), uint32(1)
	for _, test := range []struct {
		name     string
		snapshot *apipb.GetRuntimeActivityResponse
		err      error
		fresh    bool
		exact    bool
		lower    float64
	}{
		{"complete physical zero despite queued work", &apipb.GetRuntimeActivityResponse{ObservedAt: timestamppb.New(now), Complete: true, ExecutingRuns: &zero, LogicalQueuedRuns: 100}, nil, true, true, 0},
		{"positive partial", &apipb.GetRuntimeActivityResponse{ObservedAt: timestamppb.New(now), ObservedExecutingRuns: 1, UnknownReasons: []string{"unresolved"}}, nil, false, false, 1},
		{"partial zero", &apipb.GetRuntimeActivityResponse{ObservedAt: timestamppb.New(now), LogicalQueuedRuns: 100}, nil, false, false, 0},
		{"no timestamp", &apipb.GetRuntimeActivityResponse{Complete: true, ExecutingRuns: &one, ObservedExecutingRuns: 1}, nil, false, false, 1},
		{"contradictory complete", &apipb.GetRuntimeActivityResponse{ObservedAt: timestamppb.New(now), Complete: true, ExecutingRuns: &one}, nil, false, false, 0},
		{"failed owner", nil, connect.NewError(connect.CodeUnavailable, errors.New("physical read failed")), false, false, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			path, handler := apiconnect.NewAgentManagerServiceHandler(activityServer{response: test.snapshot, err: test.err})
			mux := http.NewServeMux()
			mux.Handle(path, handler)
			server := httptest.NewServer(mux)
			defer server.Close()
			source := New()
			source.now = func() time.Time { return now }
			source.resolve = func(context.Context, string) (string, error) { return server.URL, nil }
			got := source.readAgentManager(t.Context())
			require.Equal(t, test.fresh, got.State == policyv1.EvidenceState_EVIDENCE_STATE_FRESH)
			_, exact := got.Measurements["active_runs"]
			require.Equal(t, test.exact, exact)
			if test.err == nil {
				require.Equal(t, test.lower, got.Measurements["observed_executing_runs"])
				if test.snapshot.ObservedAt != nil {
					require.Equal(t, now, got.SourceTime)
				}
			} else {
				require.Empty(t, got.Measurements)
			}
		})
	}
}

func TestTypedActivityOwnerTimeAndPartialEvidenceDriveWorkTruthfully(t *testing.T) {
	now := time.Date(2026, 11, 11, 18, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name       string
		lower      uint32
		stamp      *timestamppb.Timestamp
		wantEffect string
		fallback   bool
	}{
		{"physical positive", 1, timestamppb.New(now), "slow-pulse", false},
		{"proven busy", 5, timestamppb.New(now), "gentle-pulse", false},
		{"partial zero", 0, timestamppb.New(now), "", true},
		{"stale source despite new receipt", 1, timestamppb.New(now.Add(-time.Hour)), "", true},
		{"future source", 1, timestamppb.New(now.Add(time.Hour)), "", true},
		{"future source within generic clock tolerance", 1, timestamppb.New(now.Add(time.Second)), "", true},
		{"missing source", 1, nil, "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			path, handler := apiconnect.NewAgentManagerServiceHandler(activityServer{response: &apipb.GetRuntimeActivityResponse{ObservedAt: test.stamp, ObservedExecutingRuns: test.lower, LogicalQueuedRuns: 999, UnknownReasons: []string{"unknown peer"}}})
			mux := http.NewServeMux()
			mux.Handle(path, handler)
			server := httptest.NewServer(mux)
			defer server.Close()
			source := New()
			source.resolve = func(context.Context, string) (string, error) { return server.URL, nil }
			source.now = func() time.Time { return now }
			input := source.readAgentManager(t.Context())
			require.NotContains(t, input.Measurements, "active_runs")
			config := policy.DefaultConfig()
			config.DefaultPreset = policyv1.PresetKind_PRESET_KIND_WORK
			config.ActivityEffects.ActiveEffectId = "slow-pulse"
			config.ActivityEffects.BusyEffectId = "gentle-pulse"
			_, effective, err := (policy.Engine{Config: config}).Evaluate(policy.State{}, policy.Inputs{Now: now, AgentManager: input})
			require.NoError(t, err)
			require.Equal(t, test.wantEffect, effective.EffectId)
			require.Equal(t, test.fallback, effective.StaticFallback)
		})
	}
}
