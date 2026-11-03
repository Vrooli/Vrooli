package control

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	devicedomain "device-control/internal/devices"
	"device-control/strategy"
	"device-control/strategy/fakes"
	strategyregistry "device-control/strategy/registry"
	"github.com/stretchr/testify/require"
)

type effectRouteCountingStrategy struct {
	*fakes.Strategy
	describes atomic.Int32
}

func (s *effectRouteCountingStrategy) Describe(ctx context.Context) (strategy.Declaration, error) {
	s.describes.Add(1)
	return s.Strategy.Describe(ctx)
}

type effectRouteScreenStrategy struct {
	*effectRouteCountingStrategy
	screenChanges atomic.Int32
}

func (s *effectRouteScreenStrategy) ReadState(context.Context) (strategy.DeviceState, error) {
	return strategy.DeviceState{ScreenState: "off", LockState: "locked"}, nil
}

func (s *effectRouteScreenStrategy) SetScreenState(context.Context, string) error {
	s.screenChanges.Add(1)
	return nil
}

func bridgeRouteGapRecord(id, strategyID string, capabilities ...string) devicedomain.Record {
	now := time.Now().UTC()
	capabilitiesByName := make(map[string]strategy.Capability, len(capabilities))
	for _, name := range capabilities {
		capabilitiesByName[name] = strategy.Capability{Name: name, Status: strategy.StatusUnavailable, Reason: currentRouteObservationGap}
	}
	return devicedomain.Record{
		ID: id, Kind: "physical", StrategyID: strategyID, Transport: "wireless", HostNodeID: "bridge-node",
		Status: strategy.StatusUnavailable, Health: "degraded", HealthReason: currentRouteObservationGap,
		ObservedAt: now, Transports: []strategy.DeviceTransport{{
			StrategyID: strategyID, Name: "wireless", Health: strategy.StatusUnavailable,
			HealthReason: currentRouteObservationGap, ObservedAt: now, Capabilities: capabilitiesByName,
		}},
	}
}

func TestUnavailableBridgeRouteCannotExecuteFlow(t *testing.T) {
	fake := &effectRouteCountingStrategy{Strategy: fakes.New("fake-wireless", strategy.StatusAvailable, strategy.CapInput, strategy.CapAppLifecycle)}
	svc := New(strategyregistry.New(fake))
	svc.devices.UpsertIdentity(bridgeRouteGapRecord("bridge-flow", fake.ID(), strategy.CapInput))

	_, err := svc.Run(context.Background(), Flow{Transport: "wireless", Steps: []Step{{ID: "home", Kind: "key", Target: "HOME"}}}, "bridge-flow", "operator")
	require.ErrorContains(t, err, "no current actionable transport observation from Device Control")
	require.Zero(t, fake.describes.Load(), "flow must fail before the selected adapter is queried")
	require.Empty(t, fake.Calls(), "flow must fail before dispatch")
}

func TestUnavailableBridgeRouteCannotExecuteAppLifecycle(t *testing.T) {
	fake := &effectRouteCountingStrategy{Strategy: fakes.New("fake-wireless", strategy.StatusAvailable, strategy.CapAppLifecycle)}
	svc := New(strategyregistry.New(fake))
	svc.devices.UpsertIdentity(bridgeRouteGapRecord("bridge-app", fake.ID(), strategy.CapAppLifecycle))
	lease, err := svc.Acquire("bridge-app", "operator", time.Minute)
	require.NoError(t, err)

	_, err = svc.ExecuteAppLifecycle(context.Background(), "bridge-app", AppLifecycleOperation{
		Actor: "operator", LeaseToken: lease.LeaseToken, Confirmed: true,
		AppLifecycleRequest: strategy.AppLifecycleRequest{Operation: "launch", Package: "com.example.editor"},
	})
	require.ErrorContains(t, err, "no current actionable transport observation from Device Control")
	require.Zero(t, fake.describes.Load(), "app lifecycle must fail before the selected adapter is queried")
	require.Empty(t, fake.LifecycleCalls(), "app lifecycle must fail before dispatch")
}

func TestUnavailableBridgeRouteCannotDirectActuate(t *testing.T) {
	fake := &effectRouteCountingStrategy{Strategy: fakes.New("fake-wireless", strategy.StatusAvailable, strategy.CapInput)}
	svc := New(strategyregistry.New(fake))
	svc.devices.UpsertIdentity(bridgeRouteGapRecord("bridge-direct", fake.ID(), strategy.CapInput))
	lease, err := svc.Acquire("bridge-direct", "operator", time.Minute)
	require.NoError(t, err)

	_, err = svc.ActuateDevice(context.Background(), "bridge-direct", "operator", lease.LeaseToken, DirectActuation{Transport: "wireless", Key: "HOME"})
	require.ErrorContains(t, err, "no current actionable transport observation from Device Control")
	require.Zero(t, fake.describes.Load(), "direct actuation must fail before the selected adapter is queried")
	require.Empty(t, fake.Calls(), "direct actuation must fail before dispatch")
}

func TestUnavailableBridgeRouteCannotChangeScreenState(t *testing.T) {
	fake := &effectRouteScreenStrategy{effectRouteCountingStrategy: &effectRouteCountingStrategy{Strategy: fakes.New("fake-wireless", strategy.StatusAvailable, strategy.CapScreenControl)}}
	svc := New(strategyregistry.New(fake))
	svc.devices.UpsertIdentity(bridgeRouteGapRecord("bridge-screen", fake.ID(), strategy.CapScreenControl))

	_, err := svc.Screen(context.Background(), ScreenRequest{Device: "bridge-screen", Operation: "wake", Actor: "operator"})
	require.Error(t, err)
	require.Zero(t, fake.describes.Load(), "screen action must fail before the selected adapter is queried")
	require.Zero(t, fake.screenChanges.Load(), "screen action must fail before dispatch")
}

func TestUnavailableBridgeRouteCannotCapture(t *testing.T) {
	fake := &effectRouteCountingStrategy{Strategy: fakes.New("fake-wireless", strategy.StatusAvailable, strategy.CapScreenshot)}
	svc := New(strategyregistry.New(fake))
	svc.devices.UpsertIdentity(bridgeRouteGapRecord("bridge-capture", fake.ID(), strategy.CapScreenshot))

	_, err := svc.Capture(context.Background(), CaptureRequest{Device: "bridge-capture", Kind: "screenshot", Actor: "operator"})
	require.NoError(t, err, "an unavailable capture route is an honest capability gap")
	require.Zero(t, fake.describes.Load(), "capture must fail before the selected adapter is queried")
}

func TestUnavailableBridgeRouteCannotUnlock(t *testing.T) {
	fake := &effectRouteCountingStrategy{Strategy: fakes.New("fake-wireless", strategy.StatusAvailable)}
	svc := New(strategyregistry.New(fake))
	svc.devices.UpsertIdentity(bridgeRouteGapRecord("bridge-unlock", fake.ID()))
	lease, err := svc.Acquire("bridge-unlock", "operator", time.Minute)
	require.NoError(t, err)

	_, err = svc.UnlockDevice(context.Background(), "profile-unneeded", "bridge-unlock", "operator", lease.LeaseToken)
	require.ErrorContains(t, err, "no current actionable transport observation from Device Control")
	require.Zero(t, fake.describes.Load(), "unlock must fail before the selected adapter is queried")
}

func TestRequireCurrentEffectRouteRejectsStaleProfile(t *testing.T) {
	fake := fakes.New("fake-usb", strategy.StatusAvailable, strategy.CapInput)
	svc := New(strategyregistry.New(fake))
	observedAt := time.Now().UTC().Add(-currentRouteObservationMaxAge - time.Second)
	svc.devices.UpsertIdentity(devicedomain.Record{
		ID: "stale-device", StrategyID: fake.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable,
		ObservedAt: observedAt, Transports: []strategy.DeviceTransport{{
			Name: "usb", StrategyID: fake.ID(), Health: strategy.StatusAvailable, ObservedAt: observedAt,
			Capabilities: map[string]strategy.Capability{strategy.CapInput: {Name: strategy.CapInput, Status: strategy.StatusAvailable}},
		}},
	})

	_, err := svc.requireCurrentEffectRoute("stale-device", "usb")
	require.ErrorContains(t, err, "observation is stale")
}
