package sources

import (
	"context"
	"fmt"
	"math"
	"strings"

	"connectrpc.com/connect"
	conditionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/condition"
	coveragev1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/coverage"
	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/policy"
	"lighting-control/internal/policy"
)

type conditionReader interface {
	GetCondition(context.Context, *connect.Request[conditionv1.GetConditionRequest]) (*connect.Response[conditionv1.GetConditionResponse], error)
}

func (s *Sources) readHostHealth(ctx context.Context) []policy.SourceInput {
	unavailable := policy.SourceInput{Name: "infrastructure-manager/substrate", State: policyv1.EvidenceState_EVIDENCE_STATE_UNAVAILABLE, ObservedAt: s.now().UTC()}
	if s.newConditionReader == nil {
		unavailable.Reason = "host-health reader is not configured"
		return []policy.SourceInput{unavailable}
	}
	base, err := s.resolve(ctx, "infrastructure-manager")
	if err != nil {
		unavailable.Reason = fmt.Sprintf("resolve host-health owner: %v", err)
		return []policy.SourceInput{unavailable}
	}
	client := s.newConditionReader(s.client, strings.TrimRight(base, "/"))
	response, err := client.GetCondition(ctx, connect.NewRequest(&conditionv1.GetConditionRequest{Projection: coveragev1.Projection_PROJECTION_SUBSTRATE}))
	if err != nil {
		unavailable.Reason = fmt.Sprintf("read host substrate condition: %v", err)
		return []policy.SourceInput{unavailable}
	}
	var evidence []policy.SourceInput
	for _, reading := range response.Msg.GetReadings() {
		if reading.GetOutOfScope() || !strings.HasPrefix(reading.GetCellRef(), "substrate/") {
			continue
		}
		input := policy.SourceInput{
			Name:       "infrastructure-manager/" + reading.GetCellRef(),
			ObservedAt: s.now().UTC(), SourceTime: timeFromProto(reading.GetObservedAt()),
			State:  policyv1.EvidenceState_EVIDENCE_STATE_DEGRADED,
			Reason: reading.GetUnavailableReason(),
		}
		if input.Reason == "" {
			input.Reason = reading.GetTrustVerdict().String() + "; " + reading.GetBandVerdict().String() + "; " + reading.GetBandExplanation()
		}
		switch reading.GetTrustVerdict() {
		case conditionv1.TrustVerdict_TRUST_VERDICT_GHOST, conditionv1.TrustVerdict_TRUST_VERDICT_SHELVED, conditionv1.TrustVerdict_TRUST_VERDICT_SATURATED:
			input.Excluded = true
		}
		if reading.GetTrustVerdict() == conditionv1.TrustVerdict_TRUST_VERDICT_VALID && !input.SourceTime.IsZero() {
			switch reading.GetBandVerdict() {
			case conditionv1.BandVerdict_BAND_VERDICT_IN_BAND:
				input.Measurements = map[string]float64{"health_level": 1}
				input.State = policyv1.EvidenceState_EVIDENCE_STATE_FRESH
			case conditionv1.BandVerdict_BAND_VERDICT_OUT_OF_BAND:
				level := float64(2)
				// The owner defines ordered severity 0=OK, 1=WARNING,
				// 2=CRITICAL. Other units retain its out-of-band warning.
				if reading.GetUnit() == "severity" && reading.GetValue() == 2 {
					level = 3
				}
				if !math.IsNaN(reading.GetValue()) && !math.IsInf(reading.GetValue(), 0) {
					input.Measurements = map[string]float64{"health_level": level}
					input.State = policyv1.EvidenceState_EVIDENCE_STATE_FRESH
				}
			}
		}
		evidence = append(evidence, input)
	}
	if len(evidence) == 0 {
		unavailable.Reason = "host-health owner returned no in-scope substrate readings"
		return []policy.SourceInput{unavailable}
	}
	return evidence
}
