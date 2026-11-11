package control_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/scheduletest"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
	control "lighting-control/internal/control"
)

// Public discovery is what clients use to construct the effect selector. An SDK
// descriptor that never reaches this response is not a usable capability.
func TestAdversarialBPublicDiscoveryPreservesNativeEffects(t *testing.T) {
	b := &fakeBackend{protocol: 6, controllers: []control.SDKController{{Serial: "review-b-effects", LEDCount: 4, Modes: []control.SDKMode{{Name: "Static"}, {Name: "Breathing"}}, Zones: []control.SDKZone{{Name: "Strip", LEDCount: 4, Static: true}}}}}
	s, _ := newService(t, b, scheduletest.New(time.Now()))
	_, cs, err := s.ListControllers(context.Background())
	require.NoError(t, err)
	require.Len(t, cs, 1)
	require.Len(t, cs[0].Zones, 1)
	require.Contains(t, cs[0].Zones[0].SupportedEffects, "Breathing", "fresh physical effect must reach the public discovery response")
}

func TestAdversarialBConfigurationWriteFailureCannotReachHardware(t *testing.T) {
	ctx := context.Background()
	b := &configuredBackend{fakeBackend: fakeBackend{protocol: 6, controllers: []control.SDKController{{Serial: "review-b-storage", Zones: []control.SDKZone{{Name: "Header", LEDMax: 500, Flags: 2, Static: true}}}}}}
	s, db := newService(t, b, scheduletest.New(time.Now()))
	_, cs, err := s.ListControllers(ctx)
	require.NoError(t, err)
	id, zone := cs[0].ControllerId, cs[0].Zones[0].ZoneId
	_, err = s.Enroll(ctx, id, true)
	require.NoError(t, err)
	_, _, err = s.SetManualOff(ctx, true, nil, nil)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TRIGGER reject_configuration BEFORE INSERT ON lighting_zone_configuration BEGIN SELECT RAISE(ABORT, 'fixture storage unavailable'); END`)
	require.NoError(t, err)
	_, _, err = s.ConfigureZone(ctx, id, zone, 24)
	require.ErrorContains(t, err, "fixture storage unavailable")
	require.Empty(t, b.lengths, "no physical write without durable intent")
	_, cs, err = s.ListControllers(ctx)
	require.NoError(t, err)
	require.Zero(t, cs[0].Zones[0].ConfiguredLedCount)
	override, err := control.NewSQLiteRepository(db, nil).GetManualOverride(ctx)
	require.NoError(t, err)
	require.True(t, override.Active)
	require.True(t, override.AllEnrolled)
	require.Equal(t, controlv1.BackendState_BACKEND_STATE_READY, s.BackendStatus(ctx).State)
}

func TestAdversarialBDiscoveryDropsEffectsWhenDeviceLosesCapability(t *testing.T) {
	b := &fakeBackend{protocol: 6, controllers: []control.SDKController{{Serial: "review-b-effects-return", Zones: []control.SDKZone{{Name: "Header", LEDCount: 2, Static: true, Modes: []control.SDKMode{{Name: "Breathing"}}}}}}}
	s, _ := newService(t, b, scheduletest.New(time.Now()))
	_, cs, err := s.ListControllers(context.Background())
	require.NoError(t, err)
	require.Contains(t, cs[0].Zones[0].SupportedEffects, "Breathing")
	b.controllers[0].Zones[0].Modes = nil
	_, cs, err = s.ListControllers(context.Background())
	require.NoError(t, err)
	require.Empty(t, cs[0].Zones[0].SupportedEffects, "stale effects must not survive fresh capability loss")
}
