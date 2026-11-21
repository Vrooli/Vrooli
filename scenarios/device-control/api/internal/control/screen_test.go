package control

import (
	"context"
	"fmt"
	"testing"

	devicedomain "device-control/internal/devices"
	identitydomain "device-control/internal/identity"
	"device-control/strategy"
	"device-control/strategy/fakes"
	strategyregistry "device-control/strategy/registry"
	"github.com/stretchr/testify/require"
)

func TestScreenOnConditionRequiresPositiveState(t *testing.T) {
	svc := &Service{}
	for _, tc := range []struct {
		state     string
		wantError bool
	}{{"on", false}, {"off", true}, {"", true}, {"unknown", true}} {
		t.Run(fmt.Sprintf("state_%q", tc.state), func(t *testing.T) {
			adapter := &screenConditionStrategy{Strategy: fakes.New("screen", strategy.StatusAvailable), state: strategy.DeviceState{ScreenState: tc.state}}
			err := svc.evaluateFlowCondition(context.Background(), adapter, Condition{Kind: "screen-on"}, nil, nil)
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestScreenOffConditionRequiresPositiveState(t *testing.T) {
	svc := &Service{}
	for _, tc := range []struct {
		state     string
		wantError bool
	}{{"off", false}, {"on", true}, {"", true}, {"unknown", true}} {
		t.Run(fmt.Sprintf("state_%q", tc.state), func(t *testing.T) {
			adapter := &screenConditionStrategy{Strategy: fakes.New("screen", strategy.StatusAvailable), state: strategy.DeviceState{ScreenState: tc.state}}
			err := svc.evaluateFlowCondition(context.Background(), adapter, Condition{Kind: "screen-off"}, nil, nil)
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestScreenshotDoesNotRunWhenScreenStateUnknown(t *testing.T) {
	adapter := &screenConditionStrategy{Strategy: fakes.New("screen-fixture", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot), state: strategy.DeviceState{ScreenState: "unknown", LockState: "unlocked"}}
	svc := New(strategyregistry.New(adapter))
	result, err := svc.Run(context.Background(), Flow{Steps: []Step{{ID: "capture", Kind: "screenshot"}}}, adapter.ID(), "test")
	require.NoError(t, err)
	require.NotEqual(t, "passed", result.Disposition)
	require.Empty(t, adapter.Calls())
}

func TestReadyHomeIsExplicitAndRequiresForegroundEvidence(t *testing.T) {
	for _, matches := range []bool{true, false} {
		t.Run(fmt.Sprintf("home_matches_%v", matches), func(t *testing.T) {
			base := &semanticScreenFixture{Strategy: fakes.New("home-fixture", strategy.StatusAvailable, strategy.CapScreenControl), state: strategy.DeviceState{ScreenState: "on", LockState: "unlocked", ForegroundPackage: "com.example.app"}}
			adapter := &homeScreenFixture{semanticScreenFixture: base, observed: "com.example.other"}
			if matches {
				adapter.observed = "com.example.home"
			}
			svc := New(strategyregistry.New(adapter))
			svc.devices.Upsert(devicedomain.Record{ID: "target", Kind: "physical", StrategyID: adapter.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
			_, err := svc.Screen(context.Background(), ScreenRequest{Device: "target", Operation: "ready"})
			require.NoError(t, err)
			require.Zero(t, adapter.calls)
			result, err := svc.Screen(context.Background(), ScreenRequest{Device: "target", Operation: "ready", Home: true})
			require.NoError(t, err)
			require.Equal(t, 1, adapter.calls)
			if matches {
				require.Equal(t, "verified", result.Status)
			} else {
				require.Equal(t, "unverified", result.Status)
			}
		})
	}
}

func TestReadyDismissesOnlyWhenVerifiedUnlocked(t *testing.T) {
	for _, secure := range []bool{false, true} {
		t.Run(fmt.Sprintf("secure_%v", secure), func(t *testing.T) {
			base := &semanticScreenFixture{Strategy: fakes.New("screen-fixture", strategy.StatusAvailable, strategy.CapScreenControl), state: strategy.DeviceState{ScreenState: "on", LockState: "locked"}}
			adapter := &dismissScreenFixture{semanticScreenFixture: base, secure: secure}
			svc := New(strategyregistry.New(adapter))
			svc.devices.Upsert(devicedomain.Record{ID: "target", Kind: "physical", StrategyID: adapter.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
			result, err := svc.Screen(context.Background(), ScreenRequest{Device: "target", Operation: "ready"})
			if secure {
				require.Error(t, err)
				require.Equal(t, "unavailable", result.Status)
			} else {
				require.NoError(t, err)
				require.Equal(t, "verified", result.Status)
			}
			require.Equal(t, 1, adapter.dismissals)
			require.Empty(t, svc.ListLiveSessions())
		})
	}
}

func TestSemanticScreenWaitsForDelayedSleepTransitionWithoutRepeatingCommand(t *testing.T) {
	fixture := &delayedScreenFixture{
		Strategy:  fakes.New("screen-delay", strategy.StatusAvailable, strategy.CapScreenControl),
		offAtRead: 8, // Initial state plus seven post-command reads: beyond the old 2s window.
	}
	svc := New(strategyregistry.New(fixture))
	svc.devices.Upsert(devicedomain.Record{ID: "target", StrategyID: fixture.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})

	result, err := svc.Screen(context.Background(), ScreenRequest{Device: "target", Operation: "sleep"})
	require.NoError(t, err)
	require.Equal(t, "verified", result.Status)
	require.Equal(t, "off", result.After.ScreenState)
	require.Equal(t, 8, fixture.reads)
	require.Equal(t, 1, fixture.calls, "screen sleep must not be repeated while waiting for the device to settle")
	require.Empty(t, svc.ListLiveSessions())
}

func TestScreenFlowWaitsForObservedTransitionWithoutRepeatingInput(t *testing.T) {
	fixture := &delayedScreenFixture{Strategy: fakes.New("screen-delay", strategy.StatusAvailable, strategy.CapScreenControl)}
	svc := New(strategyregistry.New(fixture))
	svc.devices.Upsert(devicedomain.Record{ID: "target", StrategyID: fixture.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	flow := Flow{Transport: "usb", Steps: []Step{{ID: "sleep", Kind: "screen", Target: "sleep", Postconditions: []Condition{{Kind: "screen-off"}}}}}
	result, err := svc.Run(context.Background(), flow, "target", "test")
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	require.Equal(t, 1, fixture.calls)
	require.Equal(t, 2, fixture.reads)
}

func TestSemanticScreenOperationsVerifyStateAndReleaseLease(t *testing.T) {
	adapter := &semanticScreenFixture{Strategy: fakes.New("screen-fixture", strategy.StatusAvailable, strategy.CapScreenControl), state: strategy.DeviceState{ScreenState: "off", LockState: "unlocked"}}
	svc := New(strategyregistry.New(adapter))
	svc.devices.Upsert(devicedomain.Record{ID: "target", Name: "TV", Kind: "physical", StrategyID: adapter.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	for _, tc := range []struct {
		op, want string
		calls    int
	}{{"wake", "on", 1}, {"ready", "on", 1}, {"sleep", "off", 2}} {
		result, err := svc.Screen(context.Background(), ScreenRequest{Device: "target", Operation: tc.op})
		require.NoError(t, err)
		require.Equal(t, "verified", result.Status)
		require.Equal(t, tc.want, result.After.ScreenState)
		require.Len(t, adapter.calls, tc.calls)
		require.Empty(t, svc.ListLiveSessions())
	}
	adapter.state = strategy.DeviceState{ScreenState: "on", LockState: "locked"}
	result, err := svc.Screen(context.Background(), ScreenRequest{Device: "target", Operation: "ready"})
	require.Error(t, err)
	require.Equal(t, "unavailable", result.Status)
	require.Empty(t, svc.ListLiveSessions())
}

func TestSemanticScreenRejectsUnverifiedResult(t *testing.T) {
	adapter := &semanticScreenFixture{Strategy: fakes.New("screen-fixture", strategy.StatusAvailable, strategy.CapScreenControl), state: strategy.DeviceState{ScreenState: "off"}, ignore: true}
	svc := New(strategyregistry.New(adapter))
	svc.devices.Upsert(devicedomain.Record{ID: "target", Kind: "physical", StrategyID: adapter.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	result, err := svc.Screen(context.Background(), ScreenRequest{Device: "target", Operation: "wake"})
	require.ErrorContains(t, err, "not verified")
	require.Equal(t, "unavailable", result.Status)
	require.Equal(t, []string{"on"}, adapter.calls)
	require.Empty(t, svc.ListLiveSessions())
}

func TestSemanticScreenRejectsAmbiguousNamesAndMissingCapability(t *testing.T) {
	adapter := &semanticScreenFixture{Strategy: fakes.New("screen-fixture", strategy.StatusAvailable, strategy.CapScreenControl), state: strategy.DeviceState{ScreenState: "off"}}
	svc := New(strategyregistry.New(adapter))
	for _, id := range []string{"one", "two"} {
		svc.devices.Upsert(devicedomain.Record{ID: id, Name: "Shared", Kind: "physical", StrategyID: adapter.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	}
	_, err := svc.Screen(context.Background(), ScreenRequest{Device: "Shared", Operation: "wake"})
	require.Error(t, err)
	require.Empty(t, adapter.calls)
	adapter.Declaration.Capabilities[strategy.CapScreenControl] = strategy.Capability{Name: strategy.CapScreenControl, Status: strategy.StatusUnavailable}
	_, err = svc.Screen(context.Background(), ScreenRequest{Device: "one", Operation: "wake"})
	require.ErrorContains(t, err, "no available")
	require.Empty(t, adapter.calls)
	require.Empty(t, svc.ListLiveSessions())
}

func TestReadDeviceStateMergesStateBearingTransportProperties(t *testing.T) {
	minimum, maximum := 0.0, 100.0
	first := fakes.NewPropertyOnly("state-rest", strategy.PropertyDescriptor{Name: "brightness", ValueType: "number", StateClass: strategy.StateBearing, Minimum: &minimum, Maximum: &maximum}, 30.0)
	second := fakes.NewPropertyOnly("state-mdns", strategy.PropertyDescriptor{Name: "volume", ValueType: "number", StateClass: strategy.StateBearing, Minimum: &minimum, Maximum: &maximum}, 0.7)
	svc := New(strategyregistry.New(first, second))
	claim := identitydomain.IdentityClaim{Kind: identitydomain.CastID, Value: "state-device", Evidence: "observed"}
	svc.devices.UpsertIdentity(devicedomain.Record{ID: "state-device", Kind: "physical", Claims: []identitydomain.IdentityClaim{claim}, IdentityKey: "state-device", StrategyID: first.ID(), Transport: "rest", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	svc.devices.UpsertIdentity(devicedomain.Record{ID: "state-device-mdns", Kind: "physical", Claims: []identitydomain.IdentityClaim{claim}, IdentityKey: "state-device", StrategyID: second.ID(), Transport: "mdns", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})

	state, err := svc.ReadDeviceState(context.Background(), "state-device")
	require.NoError(t, err)
	require.Equal(t, 30.0, state.Properties["brightness"].Value)
	require.Equal(t, 0.7, state.Properties["volume"].Value)
	require.Equal(t, "rest", state.Properties["brightness"].Transport)
	require.Equal(t, "mdns", state.Properties["volume"].Transport)
}

func TestReadDeviceStateUsesReachableSecondaryTransportForCoreState(t *testing.T) {
	minimum, maximum := 0.0, 100.0
	first := stateReaderOverride{PropertyOnly: fakes.NewPropertyOnly("state-rest", strategy.PropertyDescriptor{Name: "brightness", ValueType: "number", Minimum: &minimum, Maximum: &maximum}, 0), state: strategy.DeviceState{Unavailable: map[string]string{"screen_state": "unreachable", "lock_state": "unreachable", "foreground_package": "unreachable"}}}
	second := stateReaderOverride{PropertyOnly: fakes.NewPropertyOnly("state-mdns", strategy.PropertyDescriptor{Name: "volume", ValueType: "number", Minimum: &minimum, Maximum: &maximum}, 0), state: strategy.DeviceState{ScreenState: "on", LockState: "unlocked", ForegroundPackage: "com.example.home"}}
	svc := New(strategyregistry.New(first, second))
	claim := identitydomain.IdentityClaim{Kind: identitydomain.CastID, Value: "state-device", Evidence: "observed"}
	svc.devices.UpsertIdentity(devicedomain.Record{ID: "state-device", Kind: "physical", Claims: []identitydomain.IdentityClaim{claim}, IdentityKey: "state-device", StrategyID: first.ID(), Transport: "rest", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	svc.devices.UpsertIdentity(devicedomain.Record{ID: "state-device-mdns", Kind: "physical", Claims: []identitydomain.IdentityClaim{claim}, IdentityKey: "state-device", StrategyID: second.ID(), Transport: "mdns", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	state, err := svc.ReadDeviceState(context.Background(), "state-device")
	require.NoError(t, err)
	require.Equal(t, "on", state.ScreenState)
	require.Equal(t, "unlocked", state.LockState)
	require.Equal(t, "com.example.home", state.ForegroundPackage)
	require.NotContains(t, state.Unavailable, "screen_state")
	require.NotContains(t, state.Unavailable, "lock_state")
	require.NotContains(t, state.Unavailable, "foreground_package")
}

func TestReadDeviceStateRejectsTransportWithOnlyFailedProbes(t *testing.T) {
	fixture := stateReaderOverride{PropertyOnly: fakes.NewPropertyOnly("state-rest", strategy.PropertyDescriptor{Name: "brightness", ValueType: "number"}, 0), state: strategy.DeviceState{BatteryLevel: -1, DreamState: "unknown", Unavailable: map[string]string{"screen_state": "unreachable", "battery": "unreachable"}}}
	svc := New(strategyregistry.New(fixture))
	svc.devices.Upsert(devicedomain.Record{ID: "device", StrategyID: fixture.ID(), Transport: "rest", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	_, err := svc.ReadDeviceState(context.Background(), "device")
	require.ErrorContains(t, err, "all state probes unavailable")
}

func TestReadDeviceStateRecoversWirelessOnceWithoutReplayingActions(t *testing.T) {
	for _, shouldFail := range []bool{false, true} {
		fixture := &recoveringWirelessStateFixture{Strategy: fakes.New("android-adb", strategy.StatusAvailable, strategy.CapInput), fail: shouldFail}
		svc := New(strategyregistry.New(fixture))
		svc.devices.Upsert(devicedomain.Record{ID: "phone", Kind: "physical", Serial: "serial-1", StrategyID: fixture.ID(), Transport: "wireless", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Transports: []strategy.DeviceTransport{{Name: "wireless", StrategyID: fixture.ID()}}})
		svc.transportStrategies["phone"] = fixture
		state, err := svc.ReadDeviceState(context.Background(), "phone")
		if shouldFail {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, "on", state.ScreenState)
		}
		require.Equal(t, 1, fixture.reconnects)
		require.Empty(t, fixture.Calls())
	}
}

func TestReadDeviceStateMergesDreamStateFromSecondaryTransport(t *testing.T) {
	minimum, maximum := 0.0, 100.0
	first := stateReaderOverride{
		PropertyOnly: fakes.NewPropertyOnly("state-rest", strategy.PropertyDescriptor{Name: "brightness", ValueType: "number", StateClass: strategy.StateBearing, Minimum: &minimum, Maximum: &maximum}, 30.0),
		state:        strategy.DeviceState{DreamState: "unknown", Unavailable: map[string]string{"dream_state": "adb dumpsys dreams"}},
	}
	second := stateReaderOverride{
		PropertyOnly: fakes.NewPropertyOnly("state-adb", strategy.PropertyDescriptor{Name: "battery", ValueType: "number", StateClass: strategy.StateBearing, Minimum: &minimum, Maximum: &maximum}, 80.0),
		state:        strategy.DeviceState{DreamState: "active", DreamComponent: "com.dreamx/.Dream"},
	}
	svc := New(strategyregistry.New(first, second))
	claim := identitydomain.IdentityClaim{Kind: identitydomain.CastID, Value: "state-device", Evidence: "observed"}
	svc.devices.UpsertIdentity(devicedomain.Record{ID: "state-device", Kind: "physical", Claims: []identitydomain.IdentityClaim{claim}, IdentityKey: "state-device", StrategyID: first.ID(), Transport: "rest", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	svc.devices.UpsertIdentity(devicedomain.Record{ID: "state-device-adb", Kind: "physical", Claims: []identitydomain.IdentityClaim{claim}, IdentityKey: "state-device", StrategyID: second.ID(), Transport: "adb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})

	state, err := svc.ReadDeviceState(context.Background(), "state-device")
	require.NoError(t, err)
	require.Equal(t, "active", state.DreamState)
	require.Equal(t, "com.dreamx/.Dream", state.DreamComponent)
	require.NotContains(t, state.Unavailable, "dream_state")
}
