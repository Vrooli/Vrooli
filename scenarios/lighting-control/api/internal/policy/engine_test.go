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
	}, HealthPalette: &policyv1.HealthPalette{Healthy: &controlv1.Color{Green: 180}, Attention: &controlv1.Color{Red: 220, Green: 150}, Failure: &controlv1.Color{Red: 220}}, ActivityEffects: &policyv1.ActivityEffects{ActiveEffectId: "slow-pulse", BusyEffectId: "gentle-pulse", BusyAgentThreshold: 5}}}
}

func fresh(name string, now time.Time, m map[string]float64) SourceInput {
	return SourceInput{Name: name, State: policyv1.EvidenceState_EVIDENCE_STATE_FRESH, ObservedAt: now, SourceTime: now, Measurements: m}
}

func eval(t *testing.T, e Engine, s State, now time.Time, cpu float64, runs float64, health float64) (State, *policyv1.EffectivePolicy) {
	t.Helper()
	si := fresh("system-monitor", now, map[string]float64{"cpu_usage_percent": cpu, "gpu_usage_percent": cpu - 5, "health_failed": health})
	ai := fresh("agent-manager", now, map[string]float64{"active_runs": runs})
	ns, out, err := e.Evaluate(s, Inputs{Now: now, SystemMonitor: si, SystemHealth: si, InfrastructureHealth: []SourceInput{fresh("infrastructure-manager/substrate/SB1", now, map[string]float64{"health_level": 1 + 2*health})}, AgentManager: ai})
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

func TestLoadUsesFreshUtilizationWhenMemoryHealthDetailIsMissing(t *testing.T) {
	e := testEngine(policyv1.PresetKind_PRESET_KIND_LOAD)
	now := time.Date(2026, 11, 10, 17, 0, 0, 0, time.UTC)
	load := fresh("system-monitor", now, map[string]float64{"cpu_usage_percent": 87})
	load.Reason = "CPU/GPU snapshot is fresh; health detail unavailable"
	_, out, err := e.Evaluate(State{}, Inputs{Now: now, SystemMonitor: load})
	require.NoError(t, err)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_LOAD, out.Preset)
	require.False(t, out.StaticFallback)
}

func TestUnhealthyRequiresExplicitSignalAndEvidenceStatesFallback(t *testing.T) {
	e := testEngine(policyv1.PresetKind_PRESET_KIND_HEALTH)
	now := time.Now().UTC()
	sys := fresh("system-monitor", now, map[string]float64{"cpu_percent": 99})
	_, out, err := e.Evaluate(State{}, Inputs{Now: now, SystemHealth: sys})
	require.NoError(t, err)
	require.True(t, out.StaticFallback)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_STATIC, out.Preset)
	require.Contains(t, out.Reason, "health evidence")
	sys.Measurements["health_failed"] = 1
	_, out, err = e.Evaluate(State{}, Inputs{Now: now, SystemHealth: sys})
	require.NoError(t, err)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_HEALTH, out.Preset)
	require.False(t, out.StaticFallback)
	require.Equal(t, uint32(220), out.Color.Red)
	sys.Measurements["health_failed"] = 0
	sys.Measurements = map[string]float64{"cpu_usage_percent": 99, "memory_pressure_level": 2}
	_, out, err = e.Evaluate(State{}, Inputs{Now: now, SystemHealth: sys})
	require.NoError(t, err)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_HEALTH, out.Preset)
	require.Equal(t, uint32(220), out.Color.Red)
	require.False(t, out.StaticFallback) // elevated pressure is attention, not failure.
	sys.Measurements["memory_pressure_level"] = 3
	_, out, err = e.Evaluate(State{}, Inputs{Now: now, SystemHealth: sys})
	require.NoError(t, err)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_HEALTH, out.Preset)
	require.Equal(t, uint32(220), out.Color.Red)
	sys.State = policyv1.EvidenceState_EVIDENCE_STATE_DEGRADED
	_, out, err = e.Evaluate(State{}, Inputs{Now: now, SystemHealth: sys})
	require.NoError(t, err)
	require.True(t, out.StaticFallback)
	require.Equal(t, policyv1.EvidenceState_EVIDENCE_STATE_DEGRADED, out.Evidence[0].State)
}

func TestCombinedPriorityAndSourceFreshness(t *testing.T) {
	e := testEngine(policyv1.PresetKind_PRESET_KIND_COMBINED)
	now := time.Now().UTC()
	s := State{}
	s, out := eval(t, e, s, now, 95, 7, 1)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_COMBINED, out.Preset)
	require.Equal(t, "gentle-pulse", out.EffectId)
	require.Equal(t, uint32(220), out.Color.Red)
	s, out = eval(t, e, s, now.Add(10*time.Second), 95, 3, 0)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_COMBINED, out.Preset)
	require.Equal(t, "slow-pulse", out.EffectId)
	require.Equal(t, uint32(180), out.Color.Green)
	s, out = eval(t, e, s, now.Add(16*time.Second), 95, 0, 0)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_COMBINED, out.Preset)
	require.Empty(t, out.EffectId) // No active agents means a steady health color.
	a := fresh("agent-manager", now.Add(40*time.Second), map[string]float64{"active_runs": 2})
	a.SourceTime = now.Add(9 * time.Second)
	_, out, err := e.Evaluate(s, Inputs{Now: now.Add(40 * time.Second), SystemMonitor: fresh("system-monitor", now.Add(40*time.Second), map[string]float64{"cpu_percent": 40}), AgentManager: a})
	require.NoError(t, err)
	require.True(t, out.StaticFallback)
	require.Equal(t, policyv1.EvidenceState_EVIDENCE_STATE_STALE, out.Evidence[1].State)
}

func TestHealthAndActivityPresetsUseConfigurableSignals(t *testing.T) {
	e := testEngine(policyv1.PresetKind_PRESET_KIND_HEALTH)
	now := time.Now().UTC()
	_, healthy := eval(t, e, State{}, now, 98, 0, 0)
	require.Equal(t, uint32(180), healthy.Color.Green) // utilization alone does not change health color.
	e.Config.DefaultPreset = policyv1.PresetKind_PRESET_KIND_WORK
	_, work := eval(t, e, State{}, now, 20, 2, 0)
	require.Equal(t, "slow-pulse", work.EffectId)
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

func TestManualDynamicPresetStillUsesFreshSignalsAndExpires(t *testing.T) {
	e := Engine{Config: DefaultConfig()}
	now := time.Now().UTC()
	state, err := e.SetManual(State{}, policyv1.PresetKind_PRESET_KIND_COMBINED, now.Add(time.Minute), now)
	require.NoError(t, err)
	state, result := eval(t, e, state, now, 99, 2, 1)
	require.True(t, result.ManualPriority)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_COMBINED, result.Preset)
	require.Equal(t, e.Config.HealthPalette.Failure, result.Color)
	require.Equal(t, "Breathing", result.EffectId)
	_, idle := eval(t, e, state, now.Add(time.Second), 99, 0, 0)
	require.Equal(t, e.Config.HealthPalette.Healthy, idle.Color)
	require.Empty(t, idle.EffectId, "idle work must remove the preset's default breathing effect")
	_, stale, err := e.Evaluate(state, Inputs{Now: now.Add(30 * time.Second)})
	require.NoError(t, err)
	require.True(t, stale.ManualPriority)
	require.True(t, stale.StaticFallback)
	require.Empty(t, stale.EffectId)
	require.Equal(t, e.Config.StaleFallbackColor, stale.Color)
	_, expired := eval(t, e, state, now.Add(time.Minute), 99, 2, 1)
	require.False(t, expired.ManualPriority)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_OFF, expired.Preset)
}

func TestCombinedKeepsFreshHealthWhenActivityIsStale(t *testing.T) {
	e := Engine{Config: DefaultConfig()}
	e.Config.DefaultPreset = policyv1.PresetKind_PRESET_KIND_COMBINED
	now := time.Now().UTC()
	_, result, err := e.Evaluate(State{}, Inputs{
		Now:          now,
		SystemHealth: fresh("system-monitor/health", now, map[string]float64{"memory_pressure_level": 3}),
		AgentManager: fresh("agent-manager", now.Add(-time.Hour), map[string]float64{"active_runs": 7}),
	})
	require.NoError(t, err)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_COMBINED, result.Preset)
	require.Equal(t, e.Config.HealthPalette.Failure, result.Color)
	require.Empty(t, result.EffectId)
	require.True(t, result.StaticFallback)
	require.Equal(t, policyv1.EvidenceState_EVIDENCE_STATE_STALE, result.Evidence[1].State)
}

func TestStaleFallbackUsesItsOwnColorAndHonorsGlobalCap(t *testing.T) {
	e := testEngine(policyv1.PresetKind_PRESET_KIND_HEALTH)
	e.Config.BrightnessCapPercent = 3
	_, result, err := e.Evaluate(State{}, Inputs{Now: time.Now().UTC()})
	require.NoError(t, err)
	require.Equal(t, e.Config.StaleFallbackColor, result.Color, "an unrelated configured Static preset must not replace stale fallback")
	require.EqualValues(t, 3, result.BrightnessPercent)
	require.Empty(t, result.EffectId)
}

func TestHealthUsesMostSevereExplicitSignal(t *testing.T) {
	level, known := healthClass(map[string]float64{"health_failed": 0, "memory_pressure_level": 3})
	require.True(t, known)
	require.Equal(t, "failure", level)
}

func TestDisabledPresetCannotRemainSelectedButOffAlwaysWorks(t *testing.T) {
	e := Engine{Config: DefaultConfig()}
	now := time.Now().UTC()
	e.Config.DefaultPreset = policyv1.PresetKind_PRESET_KIND_COMBINED
	for _, preset := range e.Config.Presets {
		if preset.Kind == policyv1.PresetKind_PRESET_KIND_COMBINED || preset.Kind == policyv1.PresetKind_PRESET_KIND_OFF {
			preset.Enabled = false
		}
	}
	_, err := e.SetManual(State{}, policyv1.PresetKind_PRESET_KIND_COMBINED, time.Time{}, now)
	require.ErrorIs(t, err, ErrInvalidManualRequest)
	_, fallback := eval(t, e, State{}, now, 99, 7, 1)
	require.True(t, fallback.StaticFallback)
	require.Empty(t, fallback.EffectId)
	state, err := e.SetManual(State{}, policyv1.PresetKind_PRESET_KIND_OFF, time.Time{}, now)
	require.NoError(t, err)
	_, off := eval(t, e, state, now, 99, 7, 1)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_OFF, off.Preset)
	require.Zero(t, off.BrightnessPercent)
}

func TestMachineHealthRequiresHostEvidenceAndKeepsMemoryIndependent(t *testing.T) {
	now := time.Now().UTC()
	e := Engine{Config: DefaultConfig()}
	e.Config.DefaultPreset = policyv1.PresetKind_PRESET_KIND_HEALTH
	memory := fresh("system-monitor/health", now, map[string]float64{"memory_pressure_level": 1})
	_, unavailable, err := e.Evaluate(State{}, Inputs{Now: now, SystemHealth: memory})
	require.NoError(t, err)
	require.True(t, unavailable.StaticFallback, "normal memory alone cannot establish host health")
	host := fresh("infrastructure-manager/substrate/SB1", now, map[string]float64{"health_level": 1})
	_, healthy, err := e.Evaluate(State{}, Inputs{Now: now, SystemHealth: memory, InfrastructureHealth: []SourceInput{host}})
	require.NoError(t, err)
	require.False(t, healthy.StaticFallback)
	require.Equal(t, e.Config.HealthPalette.Healthy, healthy.Color)
	host.Measurements["health_level"] = 3
	_, failed, err := e.Evaluate(State{}, Inputs{Now: now, SystemHealth: memory, InfrastructureHealth: []SourceInput{host}})
	require.NoError(t, err)
	require.Equal(t, e.Config.HealthPalette.Failure, failed.Color)
	host.SourceTime = now.Add(-time.Hour)
	_, stale, err := e.Evaluate(State{}, Inputs{Now: now, SystemHealth: memory, InfrastructureHealth: []SourceInput{host}})
	require.NoError(t, err)
	require.True(t, stale.StaticFallback)
	require.Equal(t, policyv1.EvidenceState_EVIDENCE_STATE_STALE, stale.Evidence[0].State)
	memory.Measurements["memory_pressure_level"] = 3
	_, pressureFailure, err := e.Evaluate(State{}, Inputs{Now: now, SystemHealth: memory, InfrastructureHealth: []SourceInput{host}})
	require.NoError(t, err)
	require.Equal(t, e.Config.HealthPalette.Failure, pressureFailure.Color, "a measured critical pressure verdict remains actionable")
}
