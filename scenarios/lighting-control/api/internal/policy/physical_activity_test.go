package policy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/policy"
)

func TestPhysicalActivityLowerBoundDoesNotBecomeAnExactLogicalCount(t *testing.T) {
	now := time.Date(2026, 11, 11, 18, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name     string
		lower    float64
		stamp    time.Time
		effect   string
		fallback bool
	}{
		{"one physical executor with unknown others", 1, now, "slow-pulse", false},
		{"proven busy lower bound", 5, now, "gentle-pulse", false},
		{"partial zero is unknown", 0, now, "", true},
		{"positive missing source time", 1, time.Time{}, "", true},
		{"positive stale source time", 1, now.Add(-time.Minute), "", true},
		{"positive future source time", 1, now.Add(time.Minute), "", true},
		{"positive future within metrics clock tolerance", 1, now.Add(time.Second), "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			engine := testEngine(policyv1.PresetKind_PRESET_KIND_WORK)
			source := SourceInput{Name: "agent-manager", State: policyv1.EvidenceState_EVIDENCE_STATE_DEGRADED, ObservedAt: now, SourceTime: test.stamp, Reason: "one scope cannot be resolved", Measurements: map[string]float64{"observed_executing_runs": test.lower, "logical_queued_runs": 100, "active_runs": 999}}
			_, out, err := engine.Evaluate(State{}, Inputs{Now: now, AgentManager: source})
			require.NoError(t, err)
			require.Equal(t, test.effect, out.EffectId)
			require.Equal(t, test.fallback, out.StaticFallback)
			if !test.fallback {
				require.Equal(t, policyv1.PresetKind_PRESET_KIND_WORK, out.Preset)
				require.Equal(t, policyv1.EvidenceState_EVIDENCE_STATE_DEGRADED, out.Evidence[0].State)
				require.Contains(t, out.Reason, "total coverage remains unknown")
			}
		})
	}
}

func TestQueuedRecordsNeverEstablishWorkAndCompleteZeroIsKnownIdle(t *testing.T) {
	now := time.Date(2026, 11, 11, 18, 0, 0, 0, time.UTC)
	engine := testEngine(policyv1.PresetKind_PRESET_KIND_WORK)
	source := fresh("agent-manager", now, map[string]float64{"active_runs": 0, "logical_queued_runs": 100})
	_, out, err := engine.Evaluate(State{}, Inputs{Now: now, AgentManager: source})
	require.NoError(t, err)
	require.Empty(t, out.EffectId)
	require.False(t, out.StaticFallback)
	require.Contains(t, out.Reason, "no executing work")
}

func TestCombinedKeepsIndependentHealthWhileActivityCoverageIsPartial(t *testing.T) {
	now := time.Date(2026, 11, 11, 18, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		lower    float64
		effect   string
		fallback bool
	}{{1, "slow-pulse", false}, {5, "gentle-pulse", false}, {0, "", true}} {
		engine := testEngine(policyv1.PresetKind_PRESET_KIND_COMBINED)
		source := SourceInput{Name: "agent-manager", State: policyv1.EvidenceState_EVIDENCE_STATE_DEGRADED, SourceTime: now, ObservedAt: now, Measurements: map[string]float64{"observed_executing_runs": test.lower, "logical_queued_runs": 100}, Reason: "unknown peer scope"}
		_, out, err := engine.Evaluate(State{}, Inputs{Now: now, AgentManager: source, InfrastructureHealth: []SourceInput{fresh("host", now, map[string]float64{"health_level": 1})}})
		require.NoError(t, err)
		require.Equal(t, test.effect, out.EffectId)
		require.Equal(t, test.fallback, out.StaticFallback)
		require.Equal(t, engine.Config.HealthPalette.Healthy, out.Color)
	}
}

func TestExactActivityRequiresCurrentOwnerTime(t *testing.T) {
	now := time.Date(2026, 11, 11, 18, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name      string
		stamp     time.Time
		wantPulse bool
	}{
		{"exact current boundary", now, true},
		{"future one nanosecond", now.Add(time.Nanosecond), false},
		{"future one second", now.Add(time.Second), false},
		{"missing owner time despite current receipt", time.Time{}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := SourceInput{Name: "agent-manager", State: policyv1.EvidenceState_EVIDENCE_STATE_FRESH, SourceTime: test.stamp, ObservedAt: now, Measurements: map[string]float64{"active_runs": 1}}
			_, out, err := testEngine(policyv1.PresetKind_PRESET_KIND_WORK).Evaluate(State{}, Inputs{Now: now, AgentManager: source})
			require.NoError(t, err)
			require.Equal(t, test.wantPulse, out.EffectId != "")
			require.Equal(t, !test.wantPulse, out.StaticFallback)
		})
	}
}
