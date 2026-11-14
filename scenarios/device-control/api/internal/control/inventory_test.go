package control

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	devicedomain "device-control/internal/devices"
	"device-control/strategy"
	"device-control/strategy/fakes"
	strategyregistry "device-control/strategy/registry"
	"github.com/stretchr/testify/require"
)

func TestInventoryEnumeratesIndependentStrategiesConcurrently(t *testing.T) {
	ready := make(chan struct{})
	slow := &coordinatedInventoryProbe{Strategy: fakes.New("a-slow", strategy.StatusAvailable), ready: ready, waits: true}
	fast := &coordinatedInventoryProbe{Strategy: fakes.New("z-fast", strategy.StatusAvailable), ready: ready}
	svc := New(strategyregistry.New(slow, fast))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	devices := svc.Devices(ctx)
	ids := make([]string, 0, len(devices))
	for _, device := range devices {
		ids = append(ids, device.ID)
	}
	require.ElementsMatch(t, []string{"a-slow-device", "z-fast-device"}, ids)
}

func TestInventoryProcessesReadyDeviceBeforeOtherDiscoveryFinishes(t *testing.T) {
	ready := make(chan struct{})
	early := &earlyScopedInventoryProbe{Strategy: fakes.New("a-early", strategy.StatusAvailable), ready: ready}
	late := &coordinatedInventoryProbe{Strategy: fakes.New("z-late", strategy.StatusAvailable), ready: ready, waits: true}
	svc := New(strategyregistry.New(early, late))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	devices := svc.Devices(ctx)
	ids := make([]string, 0, len(devices))
	for _, device := range devices {
		ids = append(ids, device.ID)
	}
	require.ElementsMatch(t, []string{"early-device", "z-late-device"}, ids)
}

func TestInventoryProbesDevicesIndependently(t *testing.T) {
	base := fakes.New("scoped", strategy.StatusUnavailable, strategy.CapScreenshot)
	base.Declaration.Status = strategy.StatusAvailable
	probe := &coordinatedScopedProbe{Strategy: base, ready: make(chan struct{})}
	svc := New(strategyregistry.New(probe))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	devices := svc.Devices(ctx)
	require.Len(t, devices, 2)
	for _, device := range devices {
		found := false
		for _, capability := range device.Capabilities {
			if capability.Name == strategy.CapScreenshot {
				found = capability.Status == strategy.StatusAvailable
			}
		}
		require.True(t, found, "device %s did not retain its own screenshot capability", device.ID)
	}
}

func TestInventoryDoesNotBorrowCapabilitiesAfterDeviceProbeFailure(t *testing.T) {
	probe := &failingScopedInventoryProbe{Strategy: fakes.New("scoped", strategy.StatusAvailable, strategy.CapScreenshot)}
	svc := New(strategyregistry.New(probe))
	devices := svc.Devices(context.Background())
	capabilities := make(map[string]string)
	for _, device := range devices {
		for _, capability := range device.Capabilities {
			if capability.Name == strategy.CapScreenshot {
				capabilities[device.ID] = capability.Status
			}
		}
	}
	require.Equal(t, strategy.StatusUnavailable, capabilities["failed-device"])
	require.Equal(t, strategy.StatusAvailable, capabilities["healthy-device"])
}

func TestInventoryProbesPromotedDevicesIndependently(t *testing.T) {
	base := fakes.New("promoted", strategy.StatusUnavailable, strategy.CapScreenshot)
	base.Declaration.Status = strategy.StatusAvailable
	svc := New(strategyregistry.New(base))
	var started atomic.Int32
	ready := make(chan struct{})
	for _, id := range []string{"phone", "tv"} {
		svc.devices.Upsert(devicedomain.Record{ID: id, Serial: id, StrategyID: "promoted", Transport: "wireless", Health: strategy.StatusAvailable})
		svc.transportStrategies[id] = &coordinatedPromotedProbe{Strategy: fakes.New("promoted", strategy.StatusAvailable, strategy.CapScreenshot), started: &started, ready: ready, deviceID: id}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	devices := svc.Devices(ctx)
	capabilities := map[string]string{}
	for _, device := range devices {
		for _, capability := range device.Capabilities {
			if capability.Name == strategy.CapScreenshot {
				capabilities[device.ID] = capability.Status
			}
		}
	}
	require.Equal(t, strategy.StatusAvailable, capabilities["phone"])
	require.Equal(t, strategy.StatusAvailable, capabilities["tv"])
}

func TestInventoryIsolatesPanickingEnumerator(t *testing.T) {
	broken := &panickingEnumerator{Strategy: fakes.New("broken", strategy.StatusAvailable)}
	healthy := &countingInventoryProbe{Strategy: fakes.New("healthy", strategy.StatusAvailable)}
	svc := New(strategyregistry.New(broken, healthy))
	devices := svc.Devices(context.Background())
	statuses := make(map[string]string)
	for _, device := range devices {
		statuses[device.ID] = device.Health
	}
	require.Equal(t, strategy.HealthUnreachable, statuses["unreachable:broken"])
	require.Equal(t, strategy.StatusAvailable, statuses["counted-device"])
}

func TestInventoryDoesNotEnumerateAfterCancellation(t *testing.T) {
	broken := &panickingEnumerator{Strategy: fakes.New("broken", strategy.StatusAvailable)}
	svc := New(strategyregistry.New(broken))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	results := svc.enumerateBaseStrategies(ctx, []strategy.Declaration{{StrategyID: "broken"}}, make(chan struct{}, 8))
	require.Len(t, results, 1)
	<-results[0].done
	require.ErrorIs(t, results[0].err, context.Canceled)
}

func TestInventoryDescribesBaseStrategyOnce(t *testing.T) {
	probe := &countingInventoryProbe{Strategy: fakes.New("counted", strategy.StatusAvailable, strategy.CapScreenshot)}
	svc := New(strategyregistry.New(probe))
	devices := svc.Devices(context.Background())
	require.Len(t, devices, 1)
	require.Equal(t, "counted-device", devices[0].ID)
	require.EqualValues(t, 1, probe.describes.Load())
}

func TestObserverProbeDoesNotBlockInventoryOrCachedReads(t *testing.T) {
	probe := &slowObserverProbe{countingInventoryProbe: &countingInventoryProbe{Strategy: fakes.New("counted", strategy.StatusAvailable)}, started: make(chan struct{}, 1), release: make(chan struct{})}
	svc := New(strategyregistry.New(probe))
	done := make(chan struct{})
	go func() { svc.Devices(context.Background()); close(done) }()
	<-probe.started
	read := make(chan struct{})
	go func() { svc.CachedDevices(); close(read) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		close(probe.release)
		t.Fatal("observer qualification held inventory open")
	}
	select {
	case <-read:
	case <-time.After(time.Second):
		close(probe.release)
		t.Fatal("observer qualification blocked cached device reads")
	}
	close(probe.release)
}

func TestCancelledObserverCannotRemoveReplacement(t *testing.T) {
	svc := New(strategyregistry.New())
	oldCtx, oldCancel := context.WithCancel(context.Background())
	old := &observerJob{cancel: oldCancel}
	_, replacementCancel := context.WithCancel(context.Background())
	defer replacementCancel()
	replacement := &observerJob{cancel: replacementCancel}
	svc.observerCancels["device"] = replacement
	oldCancel()
	svc.runDeviceObserver(oldCtx, devicedomain.Record{ID: "device"}, old)
	require.Same(t, replacement, svc.observerCancels["device"])
}

func TestForgottenDeviceObserverRemainsLifecycleBlockerUntilExit(t *testing.T) {
	probe := cancellationIgnoringDescribeStrategy{started: make(chan struct{}), release: make(chan struct{})}
	svc := New(strategyregistry.New(probe))
	svc.devices.UpsertIdentity(devicedomain.Record{ID: "blocked-device", Kind: "physical", StrategyID: "polling-state"})
	svc.mu.Lock()
	svc.startObserverLocked(devicedomain.Record{ID: "blocked-device", StrategyID: "polling-state"})
	job := svc.observerCancels["blocked-device"]
	svc.mu.Unlock()
	<-probe.started

	require.True(t, svc.ForgetDeviceContext(context.Background(), "blocked-device"))
	work, blockers, complete := svc.LifecycleInventory(context.Background())
	require.True(t, complete)
	require.NotEmpty(t, blockers, "cancellation does not prove the observer exited")
	require.Equal(t, "stopping", lifecyclePairingStatus(work, "observer/blocked-device"))

	close(probe.release)
	select {
	case <-job.done:
	case <-time.After(time.Second):
		t.Fatal("observer did not exit after its probe was released")
	}
	work, blockers, complete = svc.LifecycleInventory(context.Background())
	require.True(t, complete)
	require.Empty(t, blockers)
	require.Equal(t, "idle", lifecyclePairingStatus(work, "owner/observers"))
}

func TestObserverProbePanicReportsUnreachable(t *testing.T) {
	probe := &panickingObserverProbe{Strategy: fakes.New("panic-observer", strategy.StatusAvailable)}
	svc := New(strategyregistry.New(probe))
	subscription := svc.SubscribeStateChanges(2)
	defer subscription.Cancel()
	svc.mu.Lock()
	svc.startObserverLocked(devicedomain.Record{ID: "target", StrategyID: probe.ID()})
	svc.mu.Unlock()
	select {
	case event := <-subscription.Events:
		require.Equal(t, "target", event.DeviceID)
		require.Equal(t, "transport_health", event.Attribute)
		require.Equal(t, "unreachable", event.NewValue)
	case <-time.After(time.Second):
		t.Fatal("observer probe panic did not produce a health event")
	}
}

func TestSlowInventoryDoesNotBlockLeaseLifecycle(t *testing.T) {
	probe := &slowInventoryProbe{Strategy: fakes.New("slow-inventory", strategy.StatusAvailable), started: make(chan struct{}, 1), release: make(chan struct{})}
	target := fakes.New("lease-target", strategy.StatusAvailable, strategy.CapInput)
	svc := New(strategyregistry.New(probe, target))
	session, err := svc.Acquire("lease-target", "operator", time.Minute)
	require.NoError(t, err)
	inventoryDone := make(chan struct{})
	go func() { svc.Devices(context.Background()); close(inventoryDone) }()
	<-probe.started
	leaseResult := make(chan error, 1)
	go func() {
		if len(svc.ListLiveSessions()) != 1 {
			leaseResult <- fmt.Errorf("held lease missing")
			return
		}
		if err := svc.ValidateLease(context.Background(), "lease-target", session.LeaseToken); err != nil {
			leaseResult <- err
			return
		}
		if _, err := svc.Release(session.ID); err != nil {
			leaseResult <- err
			return
		}
		_, err := svc.Acquire("lease-target", "next", time.Minute)
		leaseResult <- err
	}()
	select {
	case err := <-leaseResult:
		close(probe.release)
		<-inventoryDone
		require.NoError(t, err)
	case <-time.After(time.Second):
		close(probe.release)
		<-inventoryDone
		t.Fatal("lease lifecycle waited for a slow inventory probe")
	}
}

func TestSlowInventoryDoesNotBlockCachedDeviceReads(t *testing.T) {
	probe := &slowInventoryProbe{Strategy: fakes.New("slow-inventory", strategy.StatusAvailable), started: make(chan struct{}, 1), release: make(chan struct{})}
	svc := New(strategyregistry.New(probe))
	done := make(chan struct{})
	go func() { svc.Devices(context.Background()); close(done) }()
	<-probe.started
	read := make(chan struct{})
	go func() { svc.CachedDevices(); close(read) }()
	select {
	case <-read:
		close(probe.release)
		<-done
	case <-time.After(time.Second):
		close(probe.release)
		<-done
		t.Fatal("cached device read waited for a slow inventory probe")
	}
}

func TestForgetWaitsForInventoryBeforeRemovingIdentity(t *testing.T) {
	probe := &slowInventoryProbe{Strategy: fakes.New("slow-inventory", strategy.StatusAvailable), started: make(chan struct{}, 1), release: make(chan struct{})}
	svc := New(strategyregistry.New(probe))
	svc.devices.Upsert(devicedomain.Record{ID: "target", Kind: "physical", StrategyID: "slow-inventory", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	inventoryDone := make(chan struct{})
	go func() { svc.Devices(context.Background()); close(inventoryDone) }()
	<-probe.started
	forgotten := make(chan bool, 1)
	go func() { forgotten <- svc.ForgetDevice("target") }()
	select {
	case <-forgotten:
		close(probe.release)
		<-inventoryDone
		t.Fatal("forget completed before the in-flight inventory")
	case <-time.After(50 * time.Millisecond):
	}
	close(probe.release)
	<-inventoryDone
	require.True(t, <-forgotten)
	_, found := svc.devices.Get("target")
	require.False(t, found)
}

func TestDescribeKnownStaleDeviceDoesNotDiscoverOtherTransports(t *testing.T) {
	svc := New(strategyregistry.New())
	old := time.Now().UTC().Add(-time.Hour)
	svc.devices.UpsertIdentity(devicedomain.Record{ID: "known-tv", Kind: "physical", Serial: "tv-serial", StrategyID: "android-adb", Transport: "wireless", ObservedAt: old})
	svc.attached = failingAttachedReader{}
	device, err := svc.DescribeDevice(context.Background(), " known-tv ")
	require.NoError(t, err)
	require.Equal(t, "known-tv", device.ID)
	require.Equal(t, old, device.ObservedAt)
}

func TestDeclaredPollingTransportStartsObservationWithoutCastSpecialCase(t *testing.T) {
	svc := New(strategyregistry.New(pollingStateStrategy{}))
	subscription := svc.SubscribeStateChanges(2)
	defer subscription.Cancel()
	svc.mu.Lock()
	svc.startObserverLocked(devicedomain.Record{ID: "poll-device", StrategyID: "polling-state", Transport: "polling-state", Status: strategy.StatusAvailable})
	svc.mu.Unlock()
	defer func() {
		svc.mu.Lock()
		if job := svc.observerCancels["poll-device"]; job != nil {
			job.cancel()
		}
		svc.mu.Unlock()
	}()

	select {
	case event := <-subscription.Events:
		require.Equal(t, "poll-device", event.DeviceID)
		require.Equal(t, "polling-state", event.Transport)
		require.Equal(t, "enabled", event.Attribute)
		require.Equal(t, true, event.NewValue)
	case <-time.After(time.Second):
		t.Fatal("declared polling transport did not emit state")
	}
}

func TestBridgeInventoryFailureIsExplicitlyDegraded(t *testing.T) {
	svc := NewWithAttached(strategyregistry.New(), failingAttachedReader{})
	devices := svc.Devices(context.Background())
	require.Len(t, devices, 1)
	require.Equal(t, "bridge host node is unavailable", devices[0].HealthReason)
	require.Equal(t, strategy.StatusUnavailable, devices[0].Status)
	require.NotNil(t, devices[0].Capabilities)
}

func TestDeviceCapabilitiesUseEmptyArrayWhenUnavailable(t *testing.T) {
	device := deviceFromRecord(devicedomain.Record{ID: "offline", Status: strategy.StatusUnavailable})
	require.NotNil(t, device.Capabilities)
	require.Empty(t, device.Capabilities)
}

func TestRetainedTransportIdentityReclassifiesUnambiguousKinds(t *testing.T) {
	emulator := deviceFromRecord(devicedomain.Record{ID: "emulator", Kind: "physical", StrategyID: "android-adb", Serial: "emulator-5554"})
	require.Equal(t, "emulator", emulator.Kind)
	require.Equal(t, "android", emulator.OnboardingKind)

	desktop := deviceFromRecord(devicedomain.Record{ID: "desktop", Kind: "physical", StrategyID: "host-desktop", Serial: "swarminator:2308"})
	require.Equal(t, "desktop", desktop.Kind)
}

func TestUSBProbeDistinguishesAndroidBusPresence(t *testing.T) {
	oldLookPath, oldUSBCommand := execLookPath, usbBusCommand
	t.Cleanup(func() { execLookPath, usbBusCommand = oldLookPath, oldUSBCommand })
	execLookPath = func(string) (string, error) { return "/usr/bin/tool", nil }
	usbBusCommand = func() ([]byte, error) {
		return []byte("Bus 001 Device 002: ID 04e8:6860 Samsung Electronics Co., Ltd\n"), nil
	}
	status, reason := usbBusProbe()
	require.Equal(t, "available", status)
	require.Contains(t, reason, "USB bus")

	usbBusCommand = func() ([]byte, error) { return []byte("Bus 001 Device 001: ID 1d6b:0002 Linux Foundation\n"), nil }
	status, reason = usbBusProbe()
	require.Equal(t, "unavailable", status)
	require.Contains(t, reason, "data-capable cable")
}

func TestCrossPlatformUSBInspectorsUseHostCommands(t *testing.T) { // [REQ:DVC-P0-003]
	oldHost, oldLookPath, oldProfiler, oldWindows := strategy.HostOS, execLookPath, systemProfilerCommand, windowsUSBCommand
	t.Cleanup(func() {
		strategy.HostOS, execLookPath, systemProfilerCommand, windowsUSBCommand = oldHost, oldLookPath, oldProfiler, oldWindows
	})
	execLookPath = func(name string) (string, error) { return "/usr/bin/" + name, nil }
	strategy.HostOS = "darwin"
	systemProfilerCommand = func() ([]byte, error) { return []byte("Vendor ID: 0x04e8\nProduct ID: 0x6860\n"), nil }
	status, reason := usbBusProbe()
	require.Equal(t, "available", status)
	require.NotContains(t, reason, "usbutils")
	strategy.HostOS = "windows"
	windowsUSBCommand = func() ([]byte, error) { return []byte("InstanceId : USB\\VID_04E8&PID_6860\nStatus : OK\n"), nil }
	status, reason = usbBusProbe()
	require.Equal(t, "available", status)
	require.NotContains(t, reason, "usbutils")
	strategy.HostOS = "plan9"
	status, reason = usbBusProbe()
	require.Equal(t, "cannot-inspect", status)
	require.Contains(t, reason, "plan9")
}

func TestUSBLinkStabilityReportsFlaps(t *testing.T) { // [REQ:DVC-P0-003]
	oldHost, oldLookPath, oldCommand := strategy.HostOS, execLookPath, usbBusCommand
	t.Cleanup(func() { strategy.HostOS, execLookPath, usbBusCommand = oldHost, oldLookPath, oldCommand })
	strategy.HostOS = "linux"
	execLookPath = func(string) (string, error) { return "/usr/bin/lsusb", nil }
	outputs := [][]byte{
		[]byte("Bus 001 Device 002: ID 04e8:6860 Samsung\n"),
		[]byte("Bus 001 Device 001: ID 1d6b:0002 Linux Foundation\n"),
		[]byte("Bus 001 Device 002: ID 04e8:6860 Samsung\n"),
	}
	usbBusCommand = func() ([]byte, error) { out := outputs[0]; outputs = outputs[1:]; return out, nil }
	result := sampleUSBLink()
	require.Equal(t, "available", result.Status)
	require.Equal(t, 2, result.FlapCount)
	require.Contains(t, result.Reason, "intermittent")
}

func TestAndroidOnboardingNamesExactSDKRepairCommand(t *testing.T) {
	old := execLookPath
	t.Cleanup(func() { execLookPath = old })
	execLookPath = func(string) (string, error) { return "", os.ErrNotExist }
	rungs := (&Service{}).Onboarding("android")
	for _, rung := range rungs {
		if rung["id"] == "android-sdk" {
			require.Contains(t, rung["next_action"], "vrooli resource install android-sdk")
			return
		}
	}
	t.Fatal("android-sdk rung not returned")
}

func TestBridgeFailureDoesNotAddPseudoDeviceBesidePhysicalInventory(t *testing.T) {
	adapter := &enumeratingFake{
		Strategy: fakes.New("fake-android", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot),
		devices:  []strategy.Device{{ID: "android-stable", Serial: "serial-1", Model: "Pixel", OSVersion: "13", StrategyID: "fake-android", Transport: "usb", Health: strategy.StatusAvailable}},
	}
	svc := NewWithAttached(strategyregistry.New(adapter), failingAttachedReader{})
	devices := svc.Devices(context.Background())
	require.Len(t, devices, 1)
	require.Equal(t, "android-stable", devices[0].ID)
}

func TestBridgeFailureAppearsInDiagnosticsBesidePhysicalInventory(t *testing.T) {
	adapter := &enumeratingFake{
		Strategy: fakes.New("fake-android", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot),
		devices:  []strategy.Device{{ID: "android-stable", Serial: "serial-1", Model: "Pixel", OSVersion: "13", StrategyID: "fake-android", Transport: "usb", Health: strategy.StatusAvailable}},
	}
	svc := NewWithAttached(strategyregistry.New(adapter), failingAttachedReader{})
	diagnostics := svc.DiagnosticDevices(context.Background())
	require.Len(t, diagnostics, 1)
	require.Equal(t, "bridge-inventory", diagnostics[0].ID)
	require.Equal(t, "bridge", diagnostics[0].Kind)
	require.Contains(t, diagnostics[0].HealthReason, "list Bridge attached devices")
}

func TestCachedDevicesReturnsStoredInventoryWithoutReprobing(t *testing.T) {
	adapter := &enumeratingFake{
		Strategy: fakes.New("fake-android", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot),
		devices:  []strategy.Device{{ID: "android-stable", Serial: "serial-1", Model: "Pixel", OSVersion: "13", StrategyID: "fake-android", Transport: "usb", Health: strategy.StatusAvailable}},
	}
	svc := New(strategyregistry.New(adapter))
	// Seed the same durable observation that a prior inventory pass would have
	// left behind. CachedDevices must not call the adapter again.
	svc.devices.Upsert(devicedomain.Record{ID: "android-stable", Kind: "physical", Serial: "serial-1", Model: "Pixel", StrategyID: "fake-android", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, ObservedAt: time.Now().UTC()})
	got := svc.CachedDevices()
	require.Len(t, got, 1)
	require.Equal(t, "android-stable", got[0].ID)
	require.Equal(t, "available", got[0].Health)
}

func TestADBInventoryClassifiesEmulatorSerialAsEmulator(t *testing.T) {
	adapter := &enumeratingFake{
		Strategy: fakes.New("fake-android", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot),
		devices:  []strategy.Device{{ID: "android-emulator", Serial: "emulator-5554", Model: "sdk_gphone", OSVersion: "16", StrategyID: "fake-android", Transport: "usb", Health: strategy.StatusAvailable}},
	}
	devices := NewWithAttached(strategyregistry.New(adapter), nil).Devices(context.Background())
	require.Len(t, devices, 1)
	require.Equal(t, "emulator", devices[0].Kind)
	require.Equal(t, "emulator-5554", devices[0].Serial)
}

func TestBridgeAttachmentMergesWithLocalPhysicalIdentity(t *testing.T) { // [REQ:DVC-P0-003]
	adapter := &enumeratingFake{
		Strategy: fakes.New("fake-android", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot),
		devices:  []strategy.Device{{ID: "android-stable", Serial: "serial-1", Model: "Pixel", OSVersion: "13", StrategyID: "fake-android", Transport: "usb", Health: strategy.StatusAvailable}},
	}
	reader := staticAttachedReader{devices: []AttachedDevice{{ID: "android-stable", Name: "Pixel", HostNodeID: "swarminator", Kind: "android", Transport: "usb", Serial: "serial-1", OSVersion: "13", TrustState: "trusted", Reachability: "reachable"}}}
	svc := NewWithAttached(strategyregistry.New(adapter), reader)
	svc.SetHostNodeID("swarminator")
	devices := svc.Devices(context.Background())
	require.Len(t, devices, 1)
	require.Equal(t, "android-stable", devices[0].ID)
	require.Equal(t, "swarminator", devices[0].HostNodeID)
	require.Equal(t, "fake-android", devices[0].StrategyID)
	require.Equal(t, strategy.StatusAvailable, devices[0].Status)
}

func TestLocallyEnumeratedDeviceDoesNotRequireBridgeAttachment(t *testing.T) {
	adapter := &enumeratingFake{
		Strategy: fakes.New("fake", strategy.StatusAvailable, strategy.CapInput),
		devices:  []strategy.Device{{ID: "local-tv", Model: "Living room TV", StrategyID: "fake", Transport: "mdns", Health: strategy.StatusAvailable}},
	}
	svc := NewWithAttached(strategyregistry.New(adapter), staticAttachedReader{})
	devices := svc.Devices(context.Background())
	require.Len(t, devices, 1)
	require.Empty(t, devices[0].HostNodeID)

	lease, err := svc.AcquireContext(context.Background(), "local-tv", "operator", time.Minute)
	require.NoError(t, err)
	require.NoError(t, svc.ValidateLease(context.Background(), "local-tv", lease.LeaseToken))
}

func TestBridgeOnlyAndroidAttachmentIsAPhysicalDevice(t *testing.T) {
	reader := staticAttachedReader{devices: []AttachedDevice{{
		ID: "bridge-android-1", Name: "Galaxy", HostNodeID: "host-1", Kind: "android",
		Transport: "usb", Serial: "serial-1", OSVersion: "13", TrustState: "trusted", Reachability: "reachable",
	}}}
	svc := NewWithAttached(strategyregistry.New(), reader)
	devices := svc.Devices(context.Background())
	require.Len(t, devices, 1)
	require.Equal(t, "physical", devices[0].Kind)
	require.Equal(t, "android-adb", devices[0].StrategyID)
	require.Equal(t, "Galaxy", devices[0].Model)
	require.Equal(t, strategy.StatusUnavailable, devices[0].Status)
	require.Equal(t, "degraded", devices[0].Health)
	require.Contains(t, devices[0].HealthReason, "no current Device Control transport observation")
	require.Empty(t, devices[0].Endpoint)
	for _, capability := range devices[0].Capabilities {
		require.Equal(t, strategy.StatusUnavailable, capability.Status)
	}
}

func TestBridgeReachabilityDoesNotRestoreStaleLocalDeviceReadiness(t *testing.T) {
	adapter := &enumeratingFake{
		Strategy: fakes.New("fake-android", strategy.StatusAvailable, strategy.CapInput),
		devices: []strategy.Device{{
			ID: "android-stable", Serial: "serial-1", Model: "Galaxy", StrategyID: "fake-android", Transport: "usb", Endpoint: "usb:1-2", Health: strategy.StatusAvailable, HealthReason: "current USB transport verified",
			Transports: []strategy.DeviceTransport{{StrategyID: "fake-android", Name: "USB", Endpoint: "usb:1-2", Endpoints: []string{"usb:1-2"}, Health: strategy.StatusAvailable, Capabilities: map[string]strategy.Capability{"input": {Name: "input", Status: strategy.StatusAvailable}}}},
		}},
	}
	reader := staticAttachedReader{devices: []AttachedDevice{{ID: "android-stable", Name: "Galaxy", HostNodeID: "online-host", Kind: "android", Transport: "usb", Serial: "serial-1", TrustState: "trusted", Reachability: "reachable"}}}
	svc := NewWithAttached(strategyregistry.New(adapter), reader)

	first := svc.Devices(context.Background())
	require.Len(t, first, 1)
	require.Equal(t, strategy.StatusAvailable, first[0].Status)
	require.Equal(t, "current USB transport verified", first[0].HealthReason)
	require.Len(t, first[0].Transports, 1)
	require.Equal(t, []string{"usb:1-2"}, first[0].Transports[0].Endpoints)
	require.Equal(t, strategy.StatusAvailable, first[0].Transports[0].Capabilities["input"].Status)

	adapter.devices = nil
	second := svc.Devices(context.Background())
	require.Len(t, second, 1)
	require.Equal(t, "android-stable", second[0].ID)
	require.Equal(t, "online-host", second[0].HostNodeID)
	require.Equal(t, strategy.StatusUnavailable, second[0].Status)
	require.Equal(t, "degraded", second[0].Health)
	require.Contains(t, second[0].HealthReason, "no current Device Control transport observation")
	require.NotContains(t, second[0].HealthReason, "disconnected")
	require.Empty(t, second[0].Endpoint)
	require.NotContains(t, second[0].HealthReason, "wireless ADB endpoint reachable")
	require.Len(t, second[0].Transports, 1)
	require.Equal(t, strategy.StatusUnavailable, second[0].Transports[0].Health)
	require.Empty(t, second[0].Transports[0].Endpoint)
	require.Empty(t, second[0].Transports[0].Endpoints)
	require.Equal(t, strategy.StatusUnavailable, second[0].Transports[0].Capabilities["input"].Status)
	require.Contains(t, second[0].Transports[0].Capabilities["input"].Reason, "no current Device Control transport observation")
}

func TestUnavailableBridgeRouteCannotDirectActuateWithImplicitOrExplicitTransport(t *testing.T) {
	fake := fakes.New("fake-wireless", strategy.StatusAvailable, strategy.CapInput)
	svc := New(strategyregistry.New(fake))
	svc.devices.UpsertIdentity(devicedomain.Record{
		ID: "bridge-stale", Kind: "physical", StrategyID: fake.ID(), Transport: "wireless",
		Status: strategy.StatusUnavailable, Health: "degraded", Endpoint: "192.0.2.44:5555",
		Capabilities: []strategy.Capability{{Name: strategy.CapInput, Status: strategy.StatusUnavailable}},
		Transports: []strategy.DeviceTransport{{
			Name: "wireless", StrategyID: fake.ID(), Endpoint: "192.0.2.44:5555", Health: strategy.StatusUnavailable,
			Capabilities: map[string]strategy.Capability{strategy.CapInput: {Name: strategy.CapInput, Status: strategy.StatusUnavailable}},
		}},
	})
	svc.mu.Lock()
	svc.transportStrategies["bridge-stale"] = fake
	svc.mu.Unlock()
	lease, err := svc.Acquire("bridge-stale", "operator", time.Minute)
	require.NoError(t, err)

	for _, request := range []DirectActuation{
		{Key: "DPAD_DOWN"},
		{Key: "DPAD_DOWN", Transport: "wireless"},
	} {
		_, err := svc.ActuateDevice(context.Background(), "bridge-stale", "operator", lease.LeaseToken, request)
		require.ErrorContains(t, err, "no current actionable transport observation")
		require.Empty(t, fake.Calls(), "stale persisted transport must not receive an input event")
	}
}

func TestDirectActuationCannotFallbackWhenProfileLacksRequestedCapability(t *testing.T) {
	fake := fakes.New("fake-usb", strategy.StatusAvailable, strategy.CapInput)
	svc := New(strategyregistry.New(fake))
	svc.devices.UpsertIdentity(devicedomain.Record{
		ID: "profile-gap", Kind: "physical", StrategyID: fake.ID(), Transport: "usb",
		Status: strategy.StatusAvailable, Health: strategy.StatusAvailable,
		Capabilities: []strategy.Capability{{Name: strategy.CapInput, Status: strategy.StatusUnavailable}},
		Transports: []strategy.DeviceTransport{{
			Name: "usb", StrategyID: fake.ID(), Health: strategy.StatusAvailable,
			Capabilities: map[string]strategy.Capability{strategy.CapInput: {Name: strategy.CapInput, Status: strategy.StatusUnavailable}},
		}},
	})
	lease, err := svc.Acquire("profile-gap", "operator", time.Minute)
	require.NoError(t, err)

	for _, tc := range []struct {
		name     string
		request  DirectActuation
		expected string
	}{
		{name: "implicit", request: DirectActuation{Key: "DPAD_DOWN"}, expected: "no available transport"},
		{name: "explicit", request: DirectActuation{Key: "DPAD_DOWN", Transport: "usb"}, expected: "does not currently prove capability"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.ActuateDevice(context.Background(), "profile-gap", "operator", lease.LeaseToken, tc.request)
			require.ErrorContains(t, err, tc.expected)
			require.Empty(t, fake.Calls(), "unsupported profile must not receive an input event")
		})
	}
}

func TestBridgeOnlyAttachmentRemainsListedWhenBridgeGoesOffline(t *testing.T) {
	reader := &sequenceAttachedReader{responses: [][]AttachedDevice{{{
		ID: "bridge-android-1", Name: "Galaxy", HostNodeID: "host-1", Kind: "android",
		Transport: "usb", Serial: "serial-1", OSVersion: "13", TrustState: "trusted", Reachability: "reachable",
	}}, nil}}
	svc := NewWithAttached(strategyregistry.New(), reader)
	first := svc.Devices(context.Background())
	require.Len(t, first, 1)
	require.Equal(t, strategy.StatusUnavailable, first[0].Status)

	second := svc.Devices(context.Background())
	require.Len(t, second, 1)
	require.Equal(t, "bridge-android-1", second[0].ID)
	require.Equal(t, "physical", second[0].Kind)
	require.Equal(t, strategy.HealthUnreachable, second[0].Status)
	require.Contains(t, second[0].HealthReason, "host-1")
}

func TestEnumerationFailureRemainsVisibleAsUnreachableDevice(t *testing.T) {
	failing := &failingEnumeratingFake{Strategy: fakes.New("lan-failure", strategy.StatusAvailable, strategy.CapInput), err: errors.New("multicast interface unavailable")}
	svc := New(strategyregistry.New(failing))
	devices := svc.Devices(context.Background())
	require.Len(t, devices, 1)
	require.Equal(t, "unreachable:lan-failure", devices[0].ID)
	require.Equal(t, strategy.HealthUnreachable, devices[0].Health)
	require.Contains(t, devices[0].HealthReason, "multicast interface unavailable")
}
