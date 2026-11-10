package policy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/policy"
)

func testEngine(mode policyv1.PresetKind) Engine {
	return Engine{Config: &policyv1.PolicyConfig{DefaultPreset: mode, SourceMaxAgeSeconds: 30, LoadEnterPercent: 80, LoadExitPercent: 60, DebounceSeconds: 5, BrightnessCapPercent: 70, StaleFallbackColor: &controlv1.Color{Red: 10, Green: 20, Blue: 30}, StaleFallbackBrightnessPercent: 25, Presets: []*policyv1.Preset{
		{Kind: policyv1.PresetKind_PRESET_KIND_OFF, Enabled: true},
		{Kind: policyv1.PresetKind_PRESET_KIND_LOAD, Enabled: true, Color: &controlv1.Color{Red: 255}, BrightnessCapPercent: 90, EffectId: "pulse"},
		{Kind: policyv1.PresetKind_PRESET_KIND_WORK, Enabled: true, Color: &controlv1.Color{Green: 255}, BrightnessCapPercent: 50},
		{Kind: policyv1.PresetKind_PRESET_KIND_HEALTH, Enabled: true, Color: &controlv1.Color{Red: 255, Green: 40}, BrightnessCapPercent: 40},
		{Kind: policyv1.PresetKind_PRESET_KIND_COMBINED, Enabled: true},
		{Kind: policyv1.PresetKind_PRESET_KIND_STATIC, Enabled: true, Color: &controlv1.Color{Blue: 80}, BrightnessCapPercent: 35},
	}}}
}

func fresh(name string, now time.Time, m map[string]float64) SourceInput {
	return SourceInput{Name: name, State: policyv1.EvidenceState_EVIDENCE_STATE_FRESH, ObservedAt: now, SourceTime: now, Measurements: m}
}

func eval(t *testing.T, e Engine, s State, now time.Time, cpu float64, runs float64, health float64) (State, *policyv1.EffectivePolicy) {
	t.Helper()
	si := fresh("system-monitor", now, map[string]float64{"cpu_usage_percent": cpu, "gpu_usage_percent": cpu - 5, "health_failed": health})
	ai := fresh("agent-manager", now, map[string]float64{"active_runs": runs})
	ns, out, err := e.Evaluate(s, Inputs{Now: now, SystemMonitor: si, AgentManager: ai})
	require.NoError(t, err)
	return ns, out
}

func TestLoadPresetUsesHysteresisAndDebounce(t *testing.T) {
	e := testEngine(policyv1.PresetKind_PRESET_KIND_LOAD)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s, out := eval(t, e, State{}, now, 85, 0, 0)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_LOAD, out.Preset) // Initial state adopts first observed policy.
	s, out = eval(t, e, s, now.Add(time.Second), 79, 0, 0)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_LOAD, out.Preset) // hysteresis retains load above exit.
	s, out = eval(t, e, s, now.Add(2*time.Second), 55, 0, 0)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_LOAD, out.Preset)
	s, out = eval(t, e, s, now.Add(8*time.Second), 55, 0, 0)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_STATIC, out.Preset)
	_, out = eval(t, e, State{}, now.Add(9*time.Second), 90, 0, 0)
	require.LessOrEqual(t, out.BrightnessPercent, uint32(70))
	require.Equal(t, "pulse", out.EffectId)
}

func TestUnhealthyRequiresExplicitSignalAndEvidenceStatesFallback(t *testing.T) {
	e := testEngine(policyv1.PresetKind_PRESET_KIND_HEALTH)
	now := time.Now().UTC()
	sys := fresh("system-monitor", now, map[string]float64{"cpu_percent": 99})
	_, out, err := e.Evaluate(State{}, Inputs{Now: now, SystemMonitor: sys})
	require.NoError(t, err)
	require.True(t, out.StaticFallback)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_STATIC, out.Preset)
	require.Contains(t, out.Reason, "explicit health-failure")
	sys.Measurements["health_failed"] = 1
	_, out, err = e.Evaluate(State{}, Inputs{Now: now, SystemMonitor: sys})
	require.NoError(t, err)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_HEALTH, out.Preset)
	require.False(t, out.StaticFallback)
	sys.Measurements = map[string]float64{"cpu_usage_percent": 99, "memory_pressure_level": 2}
	_, out, err = e.Evaluate(State{}, Inputs{Now: now, SystemMonitor: sys})
	require.NoError(t, err)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_STATIC, out.Preset)
	require.False(t, out.StaticFallback) // elevated utilization/pressure is not health failure
	sys.Measurements["memory_pressure_level"] = 3
	_, out, err = e.Evaluate(State{}, Inputs{Now: now, SystemMonitor: sys})
	require.NoError(t, err)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_HEALTH, out.Preset)
	sys.State = policyv1.EvidenceState_EVIDENCE_STATE_DEGRADED
	_, out, err = e.Evaluate(State{}, Inputs{Now: now, SystemMonitor: sys})
	require.NoError(t, err)
	require.True(t, out.StaticFallback)
	require.Equal(t, policyv1.EvidenceState_EVIDENCE_STATE_DEGRADED, out.Evidence[0].State)
}

func TestCombinedPriorityAndSourceFreshness(t *testing.T) {
	e := testEngine(policyv1.PresetKind_PRESET_KIND_COMBINED)
	now := time.Now().UTC()
	s := State{}
	s, out := eval(t, e, s, now, 95, 7, 1)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_HEALTH, out.Preset)
	s, out = eval(t, e, s, now.Add(10*time.Second), 95, 7, 0)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_HEALTH, out.Preset) // debounce holds the prior choice
	s, out = eval(t, e, s, now.Add(16*time.Second), 95, 7, 0)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_LOAD, out.Preset)
	s, out = eval(t, e, s, now.Add(17*time.Second), 50, 3, 0)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_LOAD, out.Preset)
	s, out = eval(t, e, s, now.Add(23*time.Second), 50, 3, 0)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_WORK, out.Preset)
	a := fresh("agent-manager", now.Add(40*time.Second), map[string]float64{"active_runs": 2})
	a.SourceTime = now.Add(9 * time.Second)
	_, out, err := e.Evaluate(s, Inputs{Now: now.Add(40 * time.Second), SystemMonitor: fresh("system-monitor", now.Add(40*time.Second), map[string]float64{"cpu_percent": 40}), AgentManager: a})
	require.NoError(t, err)
	require.True(t, out.StaticFallback)
	require.Equal(t, policyv1.EvidenceState_EVIDENCE_STATE_STALE, out.Evidence[1].State)
}

func TestManualPriorityExpiryAndOff(t *testing.T) {
	e := testEngine(policyv1.PresetKind_PRESET_KIND_LOAD)
	now := time.Now().UTC()
	s, err := e.SetManual(State{}, policyv1.PresetKind_PRESET_KIND_OFF, now.Add(time.Minute), now)
	require.NoError(t, err)
	s, out := eval(t, e, s, now, 95, 0, 0)
	require.True(t, out.ManualPriority)
	require.Equal(t, uint32(0), out.BrightnessPercent)
	require.NotNil(t, out.RequestExpiresAt)
	s, out = eval(t, e, s, now.Add(2*time.Minute), 30, 0, 0)
	require.False(t, out.ManualPriority)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_STATIC, out.Preset)
}

func TestConfigValidationAndManualExpiryValidation(t *testing.T) {
	e := testEngine(policyv1.PresetKind_PRESET_KIND_LOAD)
	e.Config.LoadExitPercent = 90
	require.Error(t, e.Validate())
	e = testEngine(policyv1.PresetKind_PRESET_KIND_LOAD)
	_, err := e.SetManual(State{}, policyv1.PresetKind_PRESET_KIND_OFF, time.Now(), time.Now())
	require.Error(t, err)
}
