package control

import (
	"context"
	"database/sql"
	"testing"
	"time"

	devicedomain "device-control/internal/devices"
	identitydomain "device-control/internal/identity"
	"device-control/strategy"
	"device-control/strategy/fakes"
	strategyregistry "device-control/strategy/registry"
	"github.com/stretchr/testify/require"
)

func TestVerifiedReconnectRefreshesReachabilityAndCapabilities(t *testing.T) {
	adapter := newPersistentWirelessStrategy()
	adapter.endpoint = "192.168.1.42:5555"
	svc := New(strategyregistry.New(adapter))
	svc.devices.UpsertIdentity(devicedomain.Record{ID: "wireless-device", Kind: "physical", Serial: "serial-1", StrategyID: adapter.ID(), Transport: "wireless", ObservedAt: time.Now().UTC().Add(-time.Hour)})
	svc.transportStrategies["wireless-device"] = adapter
	before := time.Now().UTC()
	device, err := svc.ReconnectWireless(context.Background(), " wireless-device ")
	require.NoError(t, err)
	require.Equal(t, strategy.StatusAvailable, device.Health)
	require.False(t, device.ObservedAt.Before(before))
	require.Equal(t, "192.168.1.43:37123", device.Endpoint)
	require.NotEmpty(t, device.Capabilities)
}

func TestWirelessDebuggingFlowUsesTypedControllerAndSuppressesActuation(t *testing.T) {
	fixture := &wirelessDebugFlowFixture{Strategy: fakes.New("debug-device", strategy.StatusAvailable, strategy.CapWirelessDebugging)}
	svc := New(strategyregistry.New(fixture))
	svc.devices.Upsert(devicedomain.Record{ID: "phone", StrategyID: fixture.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	flow := Flow{Transport: "usb", Steps: []Step{{ID: "enable", Kind: "wireless-debugging", Target: "enable"}}}
	result, err := svc.Run(context.Background(), flow, "phone", "test")
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	require.Equal(t, []bool{true}, fixture.changes)
	flow.SuppressActuation = true
	flow.Steps[0].Target = "disable"
	result, err = svc.Run(context.Background(), flow, "phone", "test")
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	require.Equal(t, []bool{true}, fixture.changes)
	flow.Steps[0].Target = "toggle"
	result, err = svc.Run(context.Background(), flow, "phone", "test")
	require.NoError(t, err)
	require.Equal(t, "capability_gap", result.Disposition)
}

func TestWirelessDebuggingFlowCanAssertItsOutcome(t *testing.T) {
	fixture := &wirelessDebugFlowFixture{Strategy: fakes.New("debug-device", strategy.StatusAvailable, strategy.CapWirelessDebugging)}
	svc := New(strategyregistry.New(fixture))
	svc.devices.Upsert(devicedomain.Record{ID: "phone", StrategyID: fixture.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	flow := Flow{Transport: "usb", Steps: []Step{{ID: "enable", Kind: "wireless-debugging", Target: "enable"}, {ID: "verify", Kind: "wireless-debugging-assert", Target: "enable"}}}
	require.NoError(t, ValidatePromotionCandidate(flow))
	result, err := svc.Run(context.Background(), flow, "phone", "test")
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	flow.Steps[1].Target = "disable"
	result, err = svc.Run(context.Background(), flow, "phone", "test")
	require.NoError(t, err)
	require.Equal(t, "failed", result.Disposition)
}

func TestWirelessDebuggingPreflightExplainsUnsupportedMode(t *testing.T) {
	fixture := fakes.New("debug-device", strategy.StatusAvailable, strategy.CapWirelessDebugging)
	fixture.Declaration.Capabilities[strategy.CapWirelessDebugging] = strategy.Capability{Name: strategy.CapWirelessDebugging, Status: strategy.StatusUnavailable, NextAction: "Use legacy wireless promotion on older Android versions"}
	service := New(strategyregistry.New(fixture))
	service.devices.Upsert(devicedomain.Record{ID: "phone", StrategyID: fixture.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	result, err := service.Run(context.Background(), Flow{Transport: "usb", Steps: []Step{{ID: "enable", Kind: "wireless-debugging", Target: "enable"}}}, "phone", "test")
	require.NoError(t, err)
	require.Equal(t, "capability_gap", result.Disposition)
	require.Contains(t, result.Chapters[0].Message, "Use legacy wireless promotion")
}

func TestFlowTransportResolvesWirelessToEndpointBoundStrategy(t *testing.T) { // [REQ:DVC-P0-011]
	// A wireless device — a USB-promoted phone or a directly-onboarded network
	// TV/box — resolves an unselected transport to its verified endpoint-bound
	// strategy, never the ambient base adapter (DVC-P0-011: "refuses stale or
	// ambient endpoints"). A USB device still defaults to its base transport.
	// Explicit selection is honored; unknown transports are refused.
	svc, _ := testService(t)
	svc.devices.Upsert(devicedomain.Record{ID: "wireless-device", Kind: "physical", Serial: "serial-1", StrategyID: "fake", Transport: "wireless"})
	endpointBound := fakes.New("fake", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot)
	svc.transportStrategies["wireless-device"] = endpointBound

	resolved, ok := svc.strategyForFlow("wireless-device", "")
	require.True(t, ok)
	require.Same(t, endpointBound, resolved, "unselected transport must route to the endpoint-bound strategy, not the ambient base")

	wireless, ok := svc.strategyForFlow("wireless-device", "wireless")
	require.True(t, ok)
	require.Same(t, endpointBound, wireless)

	_, ok = svc.strategyForFlow("wireless-device", "bluetooth")
	require.False(t, ok)

	// A USB device with no endpoint-bound transport keeps the base adapter as
	// its default and does not silently escalate to a wireless strategy.
	svc.devices.Upsert(devicedomain.Record{ID: "usb-device", Kind: "physical", Serial: "serial-2", StrategyID: "fake", Transport: "usb"})
	base, ok := svc.strategyForFlow("usb-device", "")
	require.True(t, ok)
	require.NotSame(t, endpointBound, base)
}

func TestDefaultWirelessFlowReportsAndUsesWirelessTransport(t *testing.T) {
	base := fakes.New("fake", strategy.StatusAvailable, strategy.CapInput)
	wireless := fakes.New("fake-wireless", strategy.StatusAvailable, strategy.CapInput)
	svc := New(strategyregistry.New(base))
	svc.devices.Upsert(devicedomain.Record{ID: "phone", Kind: "physical", Serial: "serial-1", StrategyID: base.ID(), Transport: "wireless", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	svc.transportStrategies["phone"] = wireless
	result, err := svc.Run(context.Background(), Flow{Steps: []Step{{ID: "home", Kind: "key", Target: "HOME"}}}, "phone", "test")
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	require.Equal(t, "wireless", result.Binding.Transport)
	require.Len(t, wireless.Calls(), 1)
	require.Empty(t, base.Calls())
	require.Equal(t, "wireless", svc.Audit()[0].Transport)
}

func TestImplicitWirelessFlowFallsBackToVerifiedSerialUSBOnlyBeforeActuation(t *testing.T) {
	base := &serialScopedFlowFixture{Strategy: fakes.New("android-adb", strategy.StatusAvailable, strategy.CapInput)}
	wireless := fakes.New("wireless", strategy.StatusUnavailable, strategy.CapInput)
	svc := New(strategyregistry.New(base))
	svc.devices.Upsert(devicedomain.Record{ID: "phone", Kind: "physical", Serial: "serial-1", StrategyID: base.ID(), Transport: "wireless", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Transports: []strategy.DeviceTransport{{Name: "wireless", StrategyID: base.ID()}, {Name: "usb", StrategyID: base.ID()}}})
	svc.transportStrategies["phone"] = wireless
	flow := Flow{Steps: []Step{{ID: "home", Kind: "key", Target: "HOME"}}}
	require.True(t, svc.Validate(context.Background(), flow, "phone").Runnable, "validation and execution must select the same available transport")
	result, err := svc.Run(context.Background(), flow, "phone", "test")
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	require.Equal(t, "usb", result.Binding.Transport)
	require.Len(t, base.Calls(), 1)
	require.Empty(t, wireless.Calls())
	require.Equal(t, "usb", svc.Audit()[0].Transport)

	flow.Transport = "wireless"
	require.False(t, svc.Validate(context.Background(), flow, "phone").Runnable)
	result, err = svc.Run(context.Background(), flow, "phone", "test")
	require.NoError(t, err)
	require.Equal(t, "capability_gap", result.Disposition)
	require.Len(t, base.Calls(), 1, "an explicit wireless request must not silently use USB")
}

func TestValidateUsesPromotedWirelessDeviceStrategy(t *testing.T) {
	db, err := sql.Open("sqlite", "file:control-test-wireless-validate-"+t.Name()+"?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	generic := fakes.New("fake", strategy.StatusUnavailable, strategy.CapScreenshot)
	svc, err := NewWithDB(strategyregistry.New(generic), db)
	require.NoError(t, err)
	svc.devices.Upsert(devicedomain.Record{ID: "wireless-device", Kind: "physical", Serial: "serial-1", StrategyID: "fake", Transport: "wireless"})
	svc.transportStrategies["wireless-device"] = fakes.New("fake-wireless", strategy.StatusAvailable, strategy.CapScreenshot)

	report := svc.Validate(context.Background(), Flow{
		Transport: "wireless",
		Steps:     []Step{{ID: "observe", Kind: "observe", RequiredCapabilities: []string{strategy.CapScreenshot}}},
	}, "wireless-device")

	require.True(t, report.Runnable)
	require.Empty(t, report.Gaps)
}

func TestWirelessTransportStateSurvivesServiceReconstruction(t *testing.T) { // [REQ:DVC-P0-011]
	svc, db := testService(t)
	first := newPersistentWirelessStrategy()
	svc.registry = strategyregistry.New(first)
	svc.devices.Upsert(devicedomain.Record{ID: "wireless-device", Kind: "physical", Serial: "serial-1", StrategyID: first.ID(), Transport: "usb"})

	device, err := svc.PromoteWireless(context.Background(), "wireless-device")
	require.NoError(t, err)
	require.Equal(t, "wireless", device.Transport)
	require.Equal(t, "192.168.1.42:5555", device.Endpoint)

	reloadedStrategy := newPersistentWirelessStrategy()
	reloaded, err := NewWithDB(strategyregistry.New(reloadedStrategy), db)
	require.NoError(t, err)
	record, ok := reloaded.devices.Get("wireless-device")
	require.True(t, ok)
	require.Equal(t, "wireless", record.Transport)
	restored, ok := reloaded.transportStrategies["wireless-device"]
	require.True(t, ok)
	provider, ok := restored.(interface{ WirelessEndpoint() string })
	require.True(t, ok)
	require.Equal(t, "192.168.1.42:5555", provider.WirelessEndpoint())
}

func TestObservedTransportProfilesSurviveServiceReconstruction(t *testing.T) {
	svc, db := testService(t)
	first := &enumeratingFake{Strategy: fakes.New("android-adb", strategy.StatusAvailable, strategy.CapInput), devices: []strategy.Device{{
		ID: "tv-usb", Serial: "tv-serial", IdentityKey: "tv-serial", IdentityKind: string(identitydomain.ADBSerial), Model: "Living room TV", StrategyID: "android-adb", Transport: "usb", Endpoint: "usb", Health: strategy.StatusAvailable,
	}}}
	second := &enumeratingFake{Strategy: fakes.New("android-tv-remote", strategy.StatusAvailable, strategy.CapMedia), devices: []strategy.Device{{
		ID: "tv-remote", Serial: "tv-serial", IdentityKey: "tv-serial", IdentityKind: string(identitydomain.ADBSerial), Model: "Living room TV", StrategyID: "android-tv-remote", Transport: "mdns", Endpoint: "tv.local:6466", Health: strategy.StatusAvailable,
	}}}
	svc.registry = strategyregistry.New(first, second)

	devices := svc.Devices(context.Background())
	require.Len(t, devices, 1)
	require.Len(t, devices[0].Transports, 2)

	reloaded, err := NewWithDB(strategyregistry.New(), db)
	require.NoError(t, err)
	restored, ok := reloaded.devices.Get("tv-usb")
	require.True(t, ok)
	require.Len(t, restored.Transports, 2)
	require.ElementsMatch(t, []string{"android-adb", "android-tv-remote"}, []string{restored.Transports[0].StrategyID, restored.Transports[1].StrategyID})
	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM device_control_transport_profiles WHERE device_id = ?`, "tv-usb").Scan(&count))
	require.Equal(t, 2, count)
}

func TestIdentityMergeSplitPreservesClaimsAndAuditReachability(t *testing.T) {
	svc, db := testService(t)
	claimA := identitydomain.IdentityClaim{Kind: identitydomain.CastID, Value: "cast-a", StrategyID: "google-cast", Evidence: "observed"}
	claimB := identitydomain.IdentityClaim{Kind: identitydomain.CastID, Value: "cast-b", StrategyID: "google-cast", Evidence: "observed"}
	svc.devices.UpsertIdentity(devicedomain.Record{ID: "tv-canonical", Kind: "physical", IdentityKey: "cast-a", IdentityKind: string(identitydomain.CastID), Claims: []identitydomain.IdentityClaim{claimA}, StrategyID: "google-cast", Transport: "cast", Endpoint: "192.168.1.10:8009"})
	svc.devices.UpsertIdentity(devicedomain.Record{ID: "tv-member", Kind: "physical", IdentityKey: "cast-b", IdentityKind: string(identitydomain.CastID), Claims: []identitydomain.IdentityClaim{claimB}, StrategyID: "android-tv-remote", Transport: "mdns", Endpoint: "192.168.1.10:6466"})
	_, err := db.Exec(`INSERT INTO device_control_audits (id, actor, device_id, lease_id, verb, outcome, created_at, redaction_verified) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, "audit-before-merge", "operator", "tv-member", "lease", "key", "passed", time.Now().UTC().Format(time.RFC3339Nano), 1)
	require.NoError(t, err)

	merged, err := svc.MergeDevices(context.Background(), "tv-canonical", "tv-member", "cast-id=cast-a")
	require.NoError(t, err)
	require.Equal(t, "tv-canonical", merged.ID)
	require.Len(t, merged.Claims, 3, "the owner assertion and both observed claims remain readable")
	require.Len(t, svc.AuditForDevice(context.Background(), "tv-canonical"), 1)

	split, err := svc.SplitDevice(context.Background(), "tv-canonical")
	require.NoError(t, err)
	require.Len(t, split, 2)
	require.Len(t, svc.AuditForDevice(context.Background(), "tv-canonical"), 1)
	require.Len(t, svc.AuditForDevice(context.Background(), "tv-member"), 1)

	reloaded, err := NewWithDB(strategyregistry.New(), db)
	require.NoError(t, err)
	require.Len(t, reloaded.AuditForDevice(context.Background(), "tv-canonical"), 1)
	require.Len(t, reloaded.AuditForDevice(context.Background(), "tv-member"), 1)
}

func TestFlowsAndAuditsSelectBothGenericTransportProfiles(t *testing.T) {
	minimum, maximum := 0.0, 100.0
	first := fakes.NewPropertyOnly("hub-rest", strategy.PropertyDescriptor{Name: "brightness", ValueType: "number", Writable: true, Minimum: &minimum, Maximum: &maximum}, 10.0)
	second := fakes.NewPropertyOnly("hub-mdns", strategy.PropertyDescriptor{Name: "brightness", ValueType: "number", Writable: true, Minimum: &minimum, Maximum: &maximum}, 20.0)
	svc, _ := testService(t)
	svc.registry = strategyregistry.New(first, second)
	claim := identitydomain.IdentityClaim{Kind: identitydomain.ADBSerial, Value: "hub-serial", Evidence: "observed"}
	svc.devices.UpsertIdentity(devicedomain.Record{ID: "hub-device", Kind: "physical", Serial: "hub-serial", Claims: []identitydomain.IdentityClaim{claim}, StrategyID: first.ID(), Transport: "rest", Endpoint: "ha.example", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	svc.devices.UpsertIdentity(devicedomain.Record{ID: "hub-device-mdns", Kind: "physical", Serial: "hub-serial", Claims: []identitydomain.IdentityClaim{claim}, StrategyID: second.ID(), Transport: "mdns", Endpoint: "tv.local:6466", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})

	for _, test := range []struct {
		transport string
		value     float64
	}{
		{transport: "rest", value: 30},
		{transport: "mdns", value: 40},
	} {
		result, err := svc.Run(context.Background(), Flow{Transport: test.transport, Steps: []Step{{ID: "set-brightness", Kind: "property-set", RequiredCapabilities: []string{strategy.CapProperty}, Arguments: map[string]any{"name": "brightness", "value": test.value}}}}, "hub-device", "operator")
		require.NoError(t, err)
		require.Equal(t, "passed", result.Disposition)
	}
	audits := svc.Audit()
	require.Len(t, audits, 2)
	require.ElementsMatch(t, []string{"rest", "mdns"}, []string{audits[0].Transport, audits[1].Transport})
}

func TestFlowPreflightRecoversKnownWirelessBeforeSendingAction(t *testing.T) {
	for _, validateOnly := range []bool{false, true} {
		fixture := &recoveringWirelessStateFixture{Strategy: fakes.New("android-adb", strategy.StatusAvailable, strategy.CapInput)}
		svc := New(strategyregistry.New(fixture))
		svc.devices.Upsert(devicedomain.Record{ID: "phone", Kind: "physical", Serial: "serial-1", StrategyID: fixture.ID(), Transport: "wireless", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Transports: []strategy.DeviceTransport{{Name: "wireless", StrategyID: fixture.ID()}}})
		svc.transportStrategies["phone"] = fixture
		flow := Flow{Steps: []Step{{ID: "down", Kind: "key", Target: "DPAD_DOWN"}}}
		if validateOnly {
			require.True(t, svc.Validate(context.Background(), flow, "phone").Runnable)
			require.Empty(t, fixture.Calls())
		} else {
			result, err := svc.Run(context.Background(), flow, "phone", "test")
			require.NoError(t, err)
			require.Equal(t, "passed", result.Disposition)
			require.Equal(t, "wireless", result.Binding.Transport)
			require.Len(t, fixture.Calls(), 1)
		}
		require.Equal(t, 1, fixture.reconnects)
	}
}

func TestFlowCapabilityGapDoesNotReconnectHealthyWireless(t *testing.T) {
	fixture := &recoveringWirelessStateFixture{Strategy: fakes.New("android-adb", strategy.StatusAvailable, strategy.CapInput), online: true}
	svc := New(strategyregistry.New(fixture))
	svc.devices.Upsert(devicedomain.Record{ID: "phone", Kind: "physical", Serial: "serial-1", StrategyID: fixture.ID(), Transport: "wireless", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Transports: []strategy.DeviceTransport{{Name: "wireless", StrategyID: fixture.ID()}}})
	svc.transportStrategies["phone"] = fixture
	flow := Flow{Steps: []Step{{ID: "settings", Kind: "semantic-target", Target: "Settings"}}}

	report := svc.Validate(context.Background(), flow, "phone")
	require.False(t, report.Runnable)
	require.Equal(t, 0, fixture.reconnects, "a healthy transport cannot gain semantic-tree support by reconnecting")

	result, err := svc.Run(context.Background(), flow, "phone", "test")
	require.NoError(t, err)
	require.Equal(t, "capability_gap", result.Disposition)
	require.Equal(t, 0, fixture.reconnects)
	require.Empty(t, fixture.Calls(), "capability preflight must not actuate")
}

func TestInventoryLockHonorsCanceledReconnectContext(t *testing.T) {
	svc := New(strategyregistry.New(fakes.New("android-adb", strategy.StatusAvailable)))
	release, err := svc.lockInventory(context.Background())
	require.NoError(t, err)
	defer release()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = svc.ReconnectWireless(ctx, "phone")
	require.ErrorIs(t, err, context.Canceled)
}

func TestWirelessInventoryUsesRestoredAdapterCapabilities(t *testing.T) {
	svc, db := testService(t)
	first := newPersistentWirelessStrategy()
	svc.registry = strategyregistry.New(first)
	svc.devices.Upsert(devicedomain.Record{ID: "wireless-device", Kind: "physical", Serial: "serial-1", StrategyID: first.ID(), Transport: "usb"})
	_, err := svc.PromoteWireless(context.Background(), "wireless-device")
	require.NoError(t, err)

	reloadedStrategy := newPersistentWirelessStrategy()
	reloaded, err := NewWithDB(strategyregistry.New(reloadedStrategy), db)
	require.NoError(t, err)

	devices := reloaded.Devices(context.Background())
	require.Len(t, devices, 1)
	require.Equal(t, strategy.StatusAvailable, devices[0].Status)
	for _, name := range []string{strategy.CapInput, strategy.CapScreenshot} {
		var found strategy.Capability
		for _, capability := range devices[0].Capabilities {
			if capability.Name == name {
				found = capability
				break
			}
		}
		require.Equal(t, strategy.StatusAvailable, found.Status, "wireless capability %s", name)
	}
}

func TestWirelessReconnectPersistsRotatedEndpoint(t *testing.T) {
	svc, db := testService(t)
	first := newPersistentWirelessStrategy()
	svc.registry = strategyregistry.New(first)
	svc.devices.Upsert(devicedomain.Record{ID: "wireless-device", Kind: "physical", Serial: "serial-1", StrategyID: first.ID(), Transport: "usb"})
	_, err := svc.PromoteWireless(context.Background(), "wireless-device")
	require.NoError(t, err)

	reloaded, err := NewWithDB(strategyregistry.New(newPersistentWirelessStrategy()), db)
	require.NoError(t, err)
	device, err := reloaded.ReconnectWireless(context.Background(), "wireless-device")
	require.NoError(t, err)
	require.Equal(t, strategy.StatusAvailable, device.Status)
	restored, ok := reloaded.transportStrategies["wireless-device"].(interface{ WirelessEndpoint() string })
	require.True(t, ok)
	require.Equal(t, "192.168.1.43:37123", restored.WirelessEndpoint())

	state := reloaded.transportStates["wireless-device"]
	require.Equal(t, "192.168.1.43:37123", state.Endpoint)
}

func TestWirelessPromotionCanRecoverPreviouslyPromotedDeviceViaKnownUSB(t *testing.T) {
	svc, _ := testService(t)
	base := newPersistentWirelessStrategy()
	svc.registry = strategyregistry.New(base)
	svc.devices.Upsert(devicedomain.Record{ID: "wireless-device", Kind: "physical", Serial: "serial-1", StrategyID: base.ID(), Transport: "wireless", Transports: []strategy.DeviceTransport{{Name: "usb", StrategyID: base.ID()}}})
	device, err := svc.PromoteWireless(context.Background(), "wireless-device")
	require.NoError(t, err)
	require.Equal(t, "wireless", device.Transport)
	require.Equal(t, "192.168.1.42:5555", device.Endpoint)
}
