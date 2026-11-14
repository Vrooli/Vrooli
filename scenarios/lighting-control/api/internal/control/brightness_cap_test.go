package control_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/scheduletest"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/policy"
	control "lighting-control/internal/control"
	policy "lighting-control/internal/policy"
)

type unreadableBrightnessStore struct{}

func (unreadableBrightnessStore) Load(context.Context) (*policyv1.PolicyConfig, policy.State, error) {
	return nil, policy.State{}, errors.New("policy disk unreadable")
}
func (unreadableBrightnessStore) Save(context.Context, *policyv1.PolicyConfig, policy.State) error {
	return errors.New("unexpected save")
}

func TestUnreadableMaximumBrightnessFailsClosedButManualOffStillWrites(t *testing.T) {
	svc, _, backend, _, id, zone := enrolledPolicyFixture(t)
	svc.BindBrightnessPolicy(unreadableBrightnessStore{})
	result, err := svc.SetZoneState(t.Context(), staticState(id, zone, 255, 255, 255))
	require.NoError(t, err)
	require.Equal(t, controlv1.OperationState_OPERATION_STATE_DEFERRED, result.Operation)
	require.Contains(t, result.Reason, "policy disk unreadable")
	require.Empty(t, backend.writes)
	_, results, err := svc.SetManualOff(t.Context(), true, nil, nil)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, control.RGB{}, backend.writes[0].colors[0])
}

func TestEffectsRequireUsableBrightnessControlBeforeAnyModeWrite(t *testing.T) {
	for _, tc := range []struct {
		name            string
		flags, min, max uint32
		ok              bool
	}{
		{name: "absent"}, {name: "fixed", flags: 1 << 4, min: 10, max: 10},
		{name: "minimum exceeds cap", flags: 1 << 4, min: 50, max: 100},
		{name: "reversed", flags: 1 << 4, min: 100, max: 0},
		{name: "usable", flags: 1 << 4, max: 255, ok: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := &fakeBackend{protocol: 6, controllers: []control.SDKController{{Serial: "effect-limit", Zones: []control.SDKZone{{Name: "Strip", Static: false, LEDCount: 1, Modes: []control.SDKMode{{Name: "Breathing", SDKModeParameters: control.SDKModeParameters{Flags: tc.flags, BrightnessMin: tc.min, BrightnessMax: tc.max}}}}}}}}
			backend := &modeAwareBackend{fakeBackend: base}
			svc, _ := newService(t, backend, scheduletest.New(time.Now()))
			_, controllers, err := svc.ListControllers(t.Context())
			require.NoError(t, err)
			id, zone := controllers[0].ControllerId, controllers[0].Zones[0].ZoneId
			_, err = svc.Enroll(t.Context(), id, true)
			require.NoError(t, err)
			desired := &controlv1.ZoneState{ControllerId: id, ZoneId: zone, DesiredMode: &controlv1.ZoneState_Effect{Effect: &controlv1.EffectZoneState{EffectId: "Breathing", Color: &controlv1.Color{Red: 255}, BrightnessPercent: 90}}}
			result, err := svc.SetZoneState(t.Context(), desired)
			require.NoError(t, err)
			if tc.ok {
				require.Equal(t, uint32(35), result.Desired.GetEffect().BrightnessPercent)
				require.Len(t, backend.modes, 1)
				require.Equal(t, uint32(89), backend.modes[0].mode.Brightness)
			} else {
				require.NotNil(t, result.Desired.GetOff())
				require.Contains(t, result.Reason, "prior output may remain unchanged")
				require.Empty(t, backend.modes)
				require.Empty(t, backend.writes)
				require.Equal(t, controlv1.OperationState_OPERATION_STATE_UNSUPPORTED, result.Operation)
			}
		})
	}
}

func TestCapCommitWaitsForPriorEmissionAndSerializesReapply(t *testing.T) {
	base := &fakeBackend{protocol: 6, controllers: []control.SDKController{{Serial: "cap-order", Zones: []control.SDKZone{{Name: "Strip", LEDCount: 1, Static: true}}}}}
	backend := &blockingStableBackend{fakeBackend: base, entered: make(chan struct{}, 2), release: make(chan struct{}, 2)}
	svc, _ := newService(t, backend, scheduletest.New(time.Now()))
	_, controllers, err := svc.ListControllers(t.Context())
	require.NoError(t, err)
	id, zone := controllers[0].ControllerId, controllers[0].Zones[0].ZoneId
	_, err = svc.Enroll(t.Context(), id, true)
	require.NoError(t, err)
	done := make(chan error, 1)
	go func() { _, err := svc.SetZoneState(t.Context(), staticState(id, zone, 255, 0, 0)); done <- err }()
	<-backend.entered
	committed := make(chan struct{})
	changed := make(chan error, 1)
	go func() {
		changed <- svc.UpdateBrightnessPolicy(t.Context(), func() error { close(committed); return nil })
	}()
	select {
	case <-committed:
		t.Fatal("cap committed before prior emission completed")
	case <-time.After(25 * time.Millisecond):
	}
	backend.release <- struct{}{}
	require.NoError(t, <-done)
	<-committed
	<-backend.entered
	select {
	case err := <-changed:
		t.Fatalf("update completed before reapply: %v", err)
	default:
	}
	backend.release <- struct{}{}
	require.NoError(t, <-changed)
}

type brightnessStableBackend struct {
	*fakeBackend
	attempted []*controlv1.ZoneState
}

func (b *brightnessStableBackend) ApplyStableZone(_ context.Context, _, _ string, desired *controlv1.ZoneState) (control.SDKCommandResult, error) {
	b.attempted = append(b.attempted, desired)
	if desired.GetEffect() != nil {
		return control.SDKCommandResult{}, errors.Join(control.ErrUnsupported, control.ErrBrightnessLimit)
	}
	return control.SDKCommandResult{LocalWriteCompleted: true, BackendAcknowledged: true}, nil
}
func TestStableBrightnessCapabilityFailureEmitsTruthfulOffFallback(t *testing.T) {
	backend := &brightnessStableBackend{fakeBackend: &fakeBackend{protocol: 6, controllers: []control.SDKController{{Serial: "stable-cap-fallback", Zones: []control.SDKZone{{Name: "Strip", LEDCount: 1, Static: true}}}}}}
	svc, _ := newService(t, backend, scheduletest.New(time.Now()))
	_, controllers, err := svc.ListControllers(t.Context())
	require.NoError(t, err)
	id, zone := controllers[0].ControllerId, controllers[0].Zones[0].ZoneId
	_, err = svc.Enroll(t.Context(), id, true)
	require.NoError(t, err)
	desired := &controlv1.ZoneState{ControllerId: id, ZoneId: zone, DesiredMode: &controlv1.ZoneState_Effect{Effect: &controlv1.EffectZoneState{EffectId: "Breathing", Color: &controlv1.Color{Red: 255}, BrightnessPercent: 100}}}
	result, err := svc.SetZoneState(t.Context(), desired)
	require.NoError(t, err)
	require.Len(t, backend.attempted, 2)
	require.Equal(t, uint32(35), backend.attempted[0].GetEffect().BrightnessPercent)
	require.NotNil(t, backend.attempted[1].GetOff())
	require.NotNil(t, result.Desired.GetOff())
	require.True(t, result.LocalWriteCompleted)
	require.True(t, result.BackendAcknowledged)
	require.Contains(t, result.Reason, "static Off fallback")
	require.NotContains(t, result.Reason, "prior output may remain unchanged")
}
