package policy

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/policy"
	"google.golang.org/protobuf/proto"
	internalpolicy "lighting-control/internal/policy"
)

type previewInputs struct{ inputs internalpolicy.Inputs }

func (p *previewInputs) Read(context.Context) internalpolicy.Inputs { return p.inputs }
func loadInputs(now time.Time) internalpolicy.Inputs {
	return internalpolicy.Inputs{Now: now, SystemMonitor: internalpolicy.SourceInput{Name: "system-monitor", State: policyv1.EvidenceState_EVIDENCE_STATE_FRESH, ObservedAt: now, SourceTime: now, Measurements: map[string]float64{"cpu_percent": 95}}}
}

func TestGetDefaultPreviewDoesNotInitializePolicy(t *testing.T) {
	now := time.Date(2026, 11, 11, 15, 0, 0, 0, time.UTC)
	store := &memoryPolicyStore{}
	service := NewService(store, &previewInputs{inputs: loadInputs(now)}, nil)
	service.now = func() time.Time { return now }
	config, view, err := service.Get(context.Background())
	require.NoError(t, err)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_OFF, config.DefaultPreset)
	require.Equal(t, policyv1.PresetKind_PRESET_KIND_OFF, view.Preset)
	require.Zero(t, store.saveCalls, "a public read must not initialize authoritative policy")
	require.Nil(t, store.config)
	config, err = service.Config(context.Background())
	require.NoError(t, err)
	require.NotNil(t, config)
	require.Zero(t, store.saveCalls)
}

func TestGetPreviewsLegacyExpiryPendingAndHysteresisWithoutEffects(t *testing.T) {
	now := time.Date(2026, 11, 11, 15, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name    string
		legacy  bool
		state   internalpolicy.State
		pending bool
	}{
		{name: "legacy normalization", legacy: true},
		{name: "load hysteresis and debounce", state: internalpolicy.State{ActivePreset: policyv1.PresetKind_PRESET_KIND_STATIC}},
		{name: "expired manual selection", state: internalpolicy.State{ManualPreset: policyv1.PresetKind_PRESET_KIND_OFF, ManualExpiry: now.Add(-time.Second), ManualSelectionPending: true}},
		{name: "live pending selection", state: internalpolicy.State{ManualPreset: policyv1.PresetKind_PRESET_KIND_STATIC, ManualExpiry: now.Add(time.Minute), ManualSelectionPending: true}, pending: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := internalpolicy.DefaultConfig()
			config.DefaultPreset = policyv1.PresetKind_PRESET_KIND_LOAD
			if tc.legacy {
				config.HealthPalette = nil
				config.ActivityEffects = nil
			}
			original := proto.Clone(config).(*policyv1.PolicyConfig)
			store := &memoryPolicyStore{config: config, state: tc.state}
			control := &selectionControl{offActive: true}
			source := &previewInputs{inputs: loadInputs(now)}
			service := NewService(store, source, control)
			at := now
			service.now = func() time.Time { return at }
			for _, offset := range []time.Duration{0, time.Second, 11 * time.Second} {
				at = now.Add(offset)
				source.inputs = loadInputs(at)
				normalized, view, err := service.Get(context.Background())
				require.NoError(t, err)
				require.NotNil(t, normalized.HealthPalette)
				require.NotNil(t, normalized.ActivityEffects)
				require.Equal(t, tc.pending, strings.Contains(view.Reason, "control selection pending"))
				require.True(t, proto.Equal(original, store.config), "read normalization changed the stored object")
				require.Equal(t, tc.state, store.state, "preview advanced expiry/hysteresis/pending state")
				require.Zero(t, store.saveCalls)
				require.Empty(t, control.calls, "read must not reconcile/select/clear/apply ownership")
				require.True(t, control.offActive)
			}
		})
	}
}

func TestBackgroundTransitionsAreIndependentOfPolicyReads(t *testing.T) {
	start := time.Date(2026, 11, 11, 15, 0, 0, 0, time.UTC)
	for _, pending := range []bool{false, true} {
		t.Run(map[bool]string{false: "hysteresis", true: "expired pending ownership"}[pending], func(t *testing.T) {
			config := internalpolicy.DefaultConfig()
			config.DefaultPreset = policyv1.PresetKind_PRESET_KIND_LOAD
			initial := internalpolicy.State{ActivePreset: policyv1.PresetKind_PRESET_KIND_STATIC}
			if pending {
				initial.ManualPreset = policyv1.PresetKind_PRESET_KIND_OFF
				initial.ManualExpiry = start.Add(3 * time.Second)
				initial.ManualSelectionPending = true
			}
			leftStore := &memoryPolicyStore{config: proto.Clone(config).(*policyv1.PolicyConfig), state: initial}
			rightStore := &memoryPolicyStore{config: proto.Clone(config).(*policyv1.PolicyConfig), state: initial}
			leftControl, rightControl := &selectionControl{offActive: true}, &selectionControl{offActive: true}
			leftSource, rightSource := &previewInputs{}, &previewInputs{}
			left, right := NewService(leftStore, leftSource, leftControl), NewService(rightStore, rightSource, rightControl)
			at := start
			left.now = func() time.Time { return at }
			right.now = left.now
			for _, offset := range []time.Duration{5 * time.Second, 11 * time.Second, 20 * time.Second} {
				at = start.Add(offset - 2*time.Second)
				leftSource.inputs = loadInputs(at)
				_, _, err := left.Get(context.Background())
				require.NoError(t, err)
				require.Equal(t, rightStore.state, leftStore.state, "UI read advanced state before the background tick")
				require.Equal(t, rightStore.saveCalls, leftStore.saveCalls)
				at = start.Add(offset)
				leftSource.inputs = loadInputs(at)
				rightSource.inputs = loadInputs(at)
				left.reconcileTick(context.Background())
				right.reconcileTick(context.Background())
				require.Equal(t, rightStore.state, leftStore.state, "foreground reads changed background transition cadence")
				require.Equal(t, rightStore.saveCalls, leftStore.saveCalls)
				require.Equal(t, rightControl.calls, leftControl.calls)
				require.Equal(t, rightControl.offActive, leftControl.offActive)
				require.True(t, proto.Equal(right.lastView, left.lastView))
			}
		})
	}
}
