package sources

import (
	"context"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	conditionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/condition"
	coveragev1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/coverage"
	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/policy"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type conditionFunction func(context.Context, *connect.Request[conditionv1.GetConditionRequest]) (*connect.Response[conditionv1.GetConditionResponse], error)

func (f conditionFunction) GetCondition(ctx context.Context, req *connect.Request[conditionv1.GetConditionRequest]) (*connect.Response[conditionv1.GetConditionResponse], error) {
	return f(ctx, req)
}

func TestHostHealthUsesOwnerScopeTrustBandsAndOriginalTimestamp(t *testing.T) {
	now := time.Now().UTC()
	s := &Sources{
		now: func() time.Time { return now },
		resolve: func(_ context.Context, owner string) (string, error) {
			require.Equal(t, "infrastructure-manager", owner)
			return "http://owner", nil
		},
		newConditionReader: func(*http.Client, string) conditionReader {
			return conditionFunction(func(_ context.Context, req *connect.Request[conditionv1.GetConditionRequest]) (*connect.Response[conditionv1.GetConditionResponse], error) {
				require.Equal(t, coveragev1.Projection_PROJECTION_SUBSTRATE, req.Msg.Projection)
				return connect.NewResponse(&conditionv1.GetConditionResponse{Readings: []*conditionv1.Reading{
					{CellRef: "substrate/SB1", Unit: "severity", Value: 2, TrustVerdict: conditionv1.TrustVerdict_TRUST_VERDICT_VALID, BandVerdict: conditionv1.BandVerdict_BAND_VERDICT_OUT_OF_BAND, ObservedAt: timestamppb.New(now)},
					{CellRef: "substrate/SB2", Unit: "severity", Value: 2, TrustVerdict: conditionv1.TrustVerdict_TRUST_VERDICT_GHOST, BandVerdict: conditionv1.BandVerdict_BAND_VERDICT_OUT_OF_BAND, ObservedAt: timestamppb.New(now)},
					{CellRef: "availability/A1", Unit: "severity", Value: 2, TrustVerdict: conditionv1.TrustVerdict_TRUST_VERDICT_VALID, BandVerdict: conditionv1.BandVerdict_BAND_VERDICT_OUT_OF_BAND, ObservedAt: timestamppb.New(now)},
					{CellRef: "substrate/disabled", Unit: "severity", Value: 2, OutOfScope: true, TrustVerdict: conditionv1.TrustVerdict_TRUST_VERDICT_VALID, BandVerdict: conditionv1.BandVerdict_BAND_VERDICT_OUT_OF_BAND, ObservedAt: timestamppb.New(now)},
					{CellRef: "substrate/SB14", Unit: "percent", Value: 99, TrustVerdict: conditionv1.TrustVerdict_TRUST_VERDICT_VALID, BandVerdict: conditionv1.BandVerdict_BAND_VERDICT_IN_BAND, ObservedAt: timestamppb.New(now.Add(-time.Hour))},
				}}), nil
			})
		},
	}
	got := s.readHostHealth(context.Background())
	require.Len(t, got, 3)
	require.Equal(t, float64(3), got[0].Measurements["health_level"])
	require.Equal(t, policyv1.EvidenceState_EVIDENCE_STATE_DEGRADED, got[1].State)
	require.Empty(t, got[1].Measurements, "untrusted owner readings must not become fresh fault evidence")
	require.Equal(t, float64(1), got[2].Measurements["health_level"], "honor the owner's grade instead of applying new utilization thresholds")
	require.Equal(t, now.Add(-time.Hour), got[2].SourceTime)
}
