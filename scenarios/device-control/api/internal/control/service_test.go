package control

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"device-control/internal/conformance"
	devicedomain "device-control/internal/devices"
	"device-control/strategy"
	"device-control/strategy/fakes"
	strategyregistry "device-control/strategy/registry"
	"github.com/stretchr/testify/require"
	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	_ "modernc.org/sqlite"
)

func testService(t *testing.T) (*Service, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:control-test-"+t.Name()+"?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	fake := fakes.New("fake", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot)
	svc, err := NewWithDB(strategyregistry.New(fake), db)
	require.NoError(t, err)
	return svc, db
}

func TestLeaseAndAuditSurviveServiceReconstruction(t *testing.T) {
	svc, db := testService(t)
	session, err := svc.Acquire("fake", "operator", 0)
	require.NoError(t, err)
	require.NotEmpty(t, session.LeaseToken)
	require.NoError(t, func() error { _, err := svc.Kill(session.ID, "test"); return err }())

	reloaded, err := NewWithDB(strategyregistry.New(), db)
	require.NoError(t, err)
	sessions := reloaded.ListSessions()
	require.Len(t, sessions, 1)
	require.Equal(t, "killed", sessions[0].State)
}

func TestAgentRefusesWithoutSkillAndPromotesPassingRun(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.StartAgent(context.Background(), "observe the screen", "fake", "operator", false)
	require.ErrorContains(t, err, "prompt-manager device-control skill is unavailable")

	run, err := svc.StartAgent(context.Background(), "observe the screen", "fake", "operator", true)
	require.NoError(t, err)
	require.Equal(t, "completed", run.State)
	promoted, err := svc.PromoteAgent(run.ID)
	require.NoError(t, err)
	require.Equal(t, "promoted", promoted.State)
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

func TestAndroidConformanceUnavailableResultEmitsPhysicalTargetVerdict(t *testing.T) {
	svc, _ := testService(t)
	svc.devices.Upsert(devicedomain.Record{ID: "fake", Serial: "serial-1", HostNodeID: "host-1", Kind: "physical"})
	result, err := svc.RunAndroidConformance(context.Background(), conformance.Fixture{ID: conformance.FixtureID, PackageName: "com.example.hello", APKPath: "/missing/hello.apk", DeepLink: "hello://home"}, "fake", "operator", "")
	require.NoError(t, err)
	require.Equal(t, "unavailable", result.Disposition)
	require.NotNil(t, result.Verdict)
	require.Equal(t, commonv1.DeviceKind_DEVICE_KIND_PHYSICAL, result.Verdict.Target.DeviceKind)
	require.Contains(t, result.Verdict.Detail, "device_id=fake")
	require.Contains(t, result.Verdict.Detail, "serial=serial-1")
	require.Contains(t, result.Verdict.Detail, "host_node_id=host-1")
}

func TestAndroidConformanceRefreshesBridgeInventoryBeforeLease(t *testing.T) {
	reader := staticAttachedReader{devices: []AttachedDevice{{
		ID: "bridge-android-1", Name: "Galaxy", HostNodeID: "host-1", Kind: "android",
		Transport: "usb", Serial: "serial-1", OSVersion: "13", TrustState: "trusted", Reachability: "reachable",
	}}}
	svc := NewWithAttached(strategyregistry.New(), reader)
	result, err := svc.RunAndroidConformance(context.Background(), conformance.Fixture{
		ID: conformance.FixtureID, PackageName: "com.example.hello", APKPath: "/missing/hello.apk", DeepLink: "hello://home",
	}, "bridge-android-1", "operator", "")
	require.NoError(t, err)
	require.Equal(t, "unavailable", result.Disposition)
	require.Contains(t, result.Reason, "fixture APK is unavailable")
	require.Equal(t, "serial-1", result.Serial)
	require.Equal(t, "host-1", result.HostNodeID)
}

func TestAndroidConformanceRejectsNonPhysicalDevice(t *testing.T) {
	svc, _ := testService(t)
	svc.devices.Upsert(devicedomain.Record{ID: "emulator-1", Kind: "emulator", Serial: "emulator-1", HostNodeID: "host-1"})
	result, err := svc.RunAndroidConformance(context.Background(), conformance.Fixture{ID: conformance.FixtureID, PackageName: "com.example.hello", APKPath: "/missing/hello.apk", DeepLink: "hello://home"}, "emulator-1", "operator", "")
	require.NoError(t, err)
	require.Equal(t, "unavailable", result.Disposition)
	require.Contains(t, result.Reason, "requires a physical device")
	require.Equal(t, commonv1.DeviceKind_DEVICE_KIND_PHYSICAL, result.Verdict.Target.DeviceKind)
}

func TestAndroidConformanceReportsCapabilityGapAsUnavailable(t *testing.T) {
	svc, _ := testService(t)
	apk, err := os.CreateTemp(t.TempDir(), "hello-mobile-*.apk")
	require.NoError(t, err)
	require.NoError(t, apk.Close())
	result, err := svc.RunAndroidConformance(context.Background(), conformance.Fixture{ID: conformance.FixtureID, PackageName: "com.example.hello", APKPath: apk.Name(), DeepLink: "hello://home"}, "fake", "operator", "")
	require.NoError(t, err)
	require.Equal(t, "failed", result.Disposition)
	require.Len(t, result.Chapters, 12)
	require.Equal(t, "unavailable", result.Chapters[0].Disposition)
	require.Contains(t, result.Chapters[0].Message, "requires")
	for _, chapter := range result.Chapters {
		require.NotEqual(t, "not_run", chapter.Disposition, "chapter %s must have a terminal disposition", chapter.ID)
	}
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

func TestBridgeAttachmentMergesWithLocalPhysicalIdentity(t *testing.T) { // [REQ:DVC-P0-003]
	adapter := &enumeratingFake{
		Strategy: fakes.New("fake-android", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot),
		devices:  []strategy.Device{{ID: "android-stable", Serial: "serial-1", Model: "Pixel", OSVersion: "13", StrategyID: "fake-android", Transport: "usb", Health: strategy.StatusAvailable}},
	}
	reader := staticAttachedReader{devices: []AttachedDevice{{ID: "android-stable", Name: "Pixel", HostNodeID: "swarminator", Kind: "android", Transport: "usb", Serial: "serial-1", OSVersion: "13", TrustState: "trusted", Reachability: "reachable"}}}
	svc := NewWithAttached(strategyregistry.New(adapter), reader)
	devices := svc.Devices(context.Background())
	require.Len(t, devices, 1)
	require.Equal(t, "android-stable", devices[0].ID)
	require.Equal(t, "swarminator", devices[0].HostNodeID)
	require.Equal(t, "fake-android", devices[0].StrategyID)
	require.Equal(t, strategy.StatusAvailable, devices[0].Status)
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
}

func TestBridgeOnlyAttachmentRemainsListedWhenBridgeGoesOffline(t *testing.T) {
	reader := &sequenceAttachedReader{responses: [][]AttachedDevice{{{
		ID: "bridge-android-1", Name: "Galaxy", HostNodeID: "host-1", Kind: "android",
		Transport: "usb", Serial: "serial-1", OSVersion: "13", TrustState: "trusted", Reachability: "reachable",
	}}, nil}}
	svc := NewWithAttached(strategyregistry.New(), reader)
	first := svc.Devices(context.Background())
	require.Len(t, first, 1)
	require.Equal(t, strategy.StatusAvailable, first[0].Status)

	second := svc.Devices(context.Background())
	require.Len(t, second, 1)
	require.Equal(t, "bridge-android-1", second[0].ID)
	require.Equal(t, "physical", second[0].Kind)
	require.Equal(t, strategy.HealthUnreachable, second[0].Status)
	require.Contains(t, second[0].HealthReason, "host-1")
}

func TestFlowTransportDefaultsToUSBAndWirelessMustBeExplicit(t *testing.T) { // [REQ:DVC-P0-011]
	svc, _ := testService(t)
	svc.devices.Upsert(devicedomain.Record{ID: "wireless-device", Kind: "physical", Serial: "serial-1", StrategyID: "fake", Transport: "wireless"})
	promoted := fakes.New("fake", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot)
	svc.transportStrategies["wireless-device"] = promoted

	usb, ok := svc.strategyForFlow("wireless-device", "")
	require.True(t, ok)
	require.NotSame(t, promoted, usb)

	wireless, ok := svc.strategyForFlow("wireless-device", "wireless")
	require.True(t, ok)
	require.Same(t, promoted, wireless)

	_, ok = svc.strategyForFlow("wireless-device", "bluetooth")
	require.False(t, ok)
}

func TestWirelessTransportStateSurvivesServiceReconstruction(t *testing.T) { // [REQ:DVC-P0-011]
	svc, db := testService(t)
	first := newPersistentWirelessStrategy()
	svc.registry = strategyregistry.New(first)
	svc.devices.Upsert(devicedomain.Record{ID: "wireless-device", Kind: "physical", Serial: "serial-1", StrategyID: first.ID(), Transport: "usb"})

	device, err := svc.PromoteWireless(context.Background(), "wireless-device")
	require.NoError(t, err)
	require.Equal(t, "wireless", device.Transport)

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

func TestRequestStorageUsesRoutedDatabaseAndFileRoots(t *testing.T) { // [REQ:DVC-P0-011]
	primaryPath := filepath.Join(t.TempDir(), "primary.db")
	routed, err := coredb.Open(context.Background(), coredb.Config{
		Driver:       coredb.DriverSQLite,
		DSN:          primaryPath,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = routed.Close() })

	primaryRoots := storage.Paths{DataDir: t.TempDir()}
	roots := filerouting.New(primaryRoots)
	fake := fakes.New("fake", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot)
	svc, err := NewWithDB(strategyregistry.New(fake), routed, roots)
	require.NoError(t, err)

	leaseID := "device-control-routing-test"
	testCtx := coredb.WithTestMode(context.Background())
	testDBPath := filepath.Join(t.TempDir(), "test.db")
	require.NoError(t, routed.InstallTestPool(context.Background(), testDBPath, leaseID, time.Minute))
	testRoots := storage.Paths{DataDir: t.TempDir()}
	require.NoError(t, roots.InstallTestRoots(testRoots, leaseID, time.Minute))
	t.Cleanup(func() { _ = routed.ClearTestPool(leaseID) })
	t.Cleanup(func() { _ = roots.ClearTestRoots(leaseID) })

	_, err = routed.ExecContext(testCtx, `CREATE TABLE device_control_sessions (
 id TEXT PRIMARY KEY, device_id TEXT NOT NULL, actor TEXT NOT NULL,
 state TEXT NOT NULL, lease_token TEXT NOT NULL DEFAULT '', kill_reason TEXT NOT NULL DEFAULT '',
 expires_at TEXT NOT NULL, created_at TEXT NOT NULL
 )`)
	require.NoError(t, err)

	session, err := svc.AcquireContext(testCtx, "fake", "routing-test", time.Minute)
	require.NoError(t, err)
	require.Len(t, svc.ListSessionsContext(testCtx), 1)
	require.Empty(t, svc.ListSessions(), "test-mode session must not leak into the primary database")

	require.NoError(t, svc.persistArtifact(testCtx, "routed-artifact", []byte("redacted"), "log"))
	artifact, kind, err := svc.ArtifactContext(testCtx, "routed-artifact")
	require.NoError(t, err)
	require.Equal(t, "log", kind)
	require.Equal(t, []byte("redacted"), artifact)
	require.Equal(t, int64(1), roots.LeaseStats().TestRootWrites)
	_, err = os.Stat(filepath.Join(testRoots.DataDir, "evidence", "routed-artifact.bin"))
	require.NoError(t, err)

	_, _ = svc.ReleaseContext(testCtx, session.ID)
}

func TestRunRetainsRedactedCaptureAndTapCoordinates(t *testing.T) { // [REQ:DVC-P0-008]
	svc, _ := testService(t)
	result, err := svc.Run(context.Background(), Flow{ID: "flow-1", Steps: []Step{
		{ID: "capture", Kind: "observe", RequiredCapabilities: []string{strategy.CapScreenshot}},
		{ID: "tap", Kind: "tap", Target: "12,34", RequiredCapabilities: []string{strategy.CapInput}},
	}}, "fake", "operator")
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	require.Len(t, result.Evidence, 1)
	require.NotEmpty(t, result.Evidence[0].SHA256)
	require.Greater(t, result.Evidence[0].SizeBytes, int64(0))
	artifactPath := svc.artifacts[result.Evidence[0].ID]
	contents, err := os.ReadFile(artifactPath)
	require.NoError(t, err)
	require.Len(t, contents, int(result.Evidence[0].SizeBytes))
	fake, ok := svc.registry.Get("fake")
	require.True(t, ok)
	actuator := fake.(*fakes.Strategy)
	calls := actuator.Calls()
	require.Len(t, calls, 1)
	require.InDelta(t, 12, calls[0].Pointer.X, 0)
	require.InDelta(t, 34, calls[0].Pointer.Y, 0)
}

func TestRunRejectsUnredactedCaptureWithoutActor(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.Run(context.Background(), Flow{AllowUnredactedCapture: true, Steps: []Step{{ID: "capture", Kind: "observe"}}}, "fake", "")
	require.ErrorContains(t, err, "requires an actor")
}

func TestUnredactedCaptureIsAuditedWithActor(t *testing.T) {
	svc, _ := testService(t)
	result, err := svc.Run(context.Background(), Flow{AllowUnredactedCapture: true, Steps: []Step{{ID: "capture", Kind: "observe"}}}, "fake", "owner-1")
	require.NoError(t, err)
	require.Len(t, result.Evidence, 1)
	require.True(t, result.Evidence[0].OptedOut)
	records := svc.Audit()
	require.NotEmpty(t, records)
	require.Equal(t, "owner-1", records[0].Actor)
	require.True(t, records[0].RedactionOptedOut)
}

func TestRunReusesHeldLeaseTokenWithoutConflict(t *testing.T) { // [REQ:DVC-P0-004]
	svc, _ := testService(t)
	session, err := svc.Acquire("fake", "operator", time.Minute)
	require.NoError(t, err)
	result, err := svc.RunWithLease(context.Background(), Flow{Steps: []Step{{ID: "wait", Kind: "wait"}}}, "fake", "operator", session.LeaseToken)
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	require.Len(t, svc.ListSessions(), 1)
	require.Equal(t, "held", svc.ListSessions()[0].State)
	_, err = svc.Release(session.ID)
	require.NoError(t, err)
}

func TestFrameDifferenceFailsWhenActuationIsSuppressed(t *testing.T) { // [REQ:DVC-P0-008]
	svc, _ := testService(t)
	result, err := svc.Run(context.Background(), Flow{SuppressActuation: true, Steps: []Step{
		{ID: "before", Kind: "observe", RequiredCapabilities: []string{strategy.CapScreenshot}},
		{ID: "actuate", Kind: "key", Target: "KEYCODE_APP_SWITCH", RequiredCapabilities: []string{strategy.CapInput}},
		{ID: "after", Kind: "observe", RequiredCapabilities: []string{strategy.CapScreenshot}},
		{ID: "changed", Kind: "assert-frame-different"},
	}}, "fake", "operator")
	require.NoError(t, err)
	require.Equal(t, "failed", result.Disposition)
	require.Contains(t, result.Chapters[len(result.Chapters)-1].Message, "identical")
}

func TestDeviceDisconnectReleasesLeaseAndRetainsPriorEvidence(t *testing.T) { // [REQ:DVC-P0-004]
	svc, _ := testService(t)
	fake, _ := svc.registry.Get("fake")
	fake.(*fakes.Strategy).ActuateErr = &strategy.AvailabilityError{Reason: "adb transport disappeared", NextAction: "Reconnect the device over USB."}
	result, err := svc.Run(context.Background(), Flow{Steps: []Step{
		{ID: "before", Kind: "observe", RequiredCapabilities: []string{strategy.CapScreenshot}},
		{ID: "actuate", Kind: "key", Target: "ENTER", RequiredCapabilities: []string{strategy.CapInput}},
	}}, "fake", "operator")
	require.NoError(t, err)
	require.Equal(t, "device_disconnected", result.Disposition)
	require.True(t, result.Incomplete)
	require.Equal(t, "actuate", result.DisconnectStep)
	require.Len(t, result.Evidence, 1)
	require.Empty(t, svc.ListLiveSessions())
}

func TestListLiveSessionsExcludesFinishedLeases(t *testing.T) {
	svc, _ := testService(t)
	session, err := svc.Acquire("fake", "operator", time.Minute)
	require.NoError(t, err)
	require.Len(t, svc.ListLiveSessions(), 1)
	_, err = svc.Release(session.ID)
	require.NoError(t, err)
	require.Empty(t, svc.ListLiveSessions())
	require.Len(t, svc.ListSessions(), 1)
}

func TestKillStopsAnInFlightFlow(t *testing.T) { // [REQ:DVC-P0-009]
	svc, _ := testService(t)
	session, err := svc.Acquire("fake", "operator", time.Minute)
	require.NoError(t, err)
	finished := make(chan RunResult, 1)
	go func() {
		result, _ := svc.RunWithLease(context.Background(), Flow{Steps: []Step{{ID: "wait", Kind: "wait", TimeoutMS: 5000, Arguments: map[string]any{"settle_ms": float64(5000)}}}}, "fake", "operator", session.LeaseToken)
		finished <- result
	}()
	time.Sleep(25 * time.Millisecond)
	_, err = svc.Kill(session.ID, "operator requested kill")
	require.NoError(t, err)
	result := <-finished
	require.Equal(t, "cancelled", result.Disposition)
	require.NotEmpty(t, result.Chapters)
}

type failingAttachedReader struct{}

func (failingAttachedReader) List(context.Context) ([]AttachedDevice, error) {
	return nil, context.DeadlineExceeded
}

type staticAttachedReader struct{ devices []AttachedDevice }

type persistentWirelessStrategy struct {
	*fakes.Strategy
	endpoint string
}

func newPersistentWirelessStrategy() *persistentWirelessStrategy {
	return &persistentWirelessStrategy{Strategy: fakes.New("persistent-wireless", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot)}
}

func (s *persistentWirelessStrategy) ForDevice(string) strategy.Strategy { return s }

func (s *persistentWirelessStrategy) PromoteWireless(context.Context) error {
	s.endpoint = "192.168.1.42:5555"
	return nil
}

func (s *persistentWirelessStrategy) WirelessEndpoint() string { return s.endpoint }

func (s *persistentWirelessStrategy) RestoreWireless(endpoint string) strategy.Strategy {
	return &persistentWirelessStrategy{Strategy: s.Strategy, endpoint: endpoint}
}

func (r staticAttachedReader) List(context.Context) ([]AttachedDevice, error) { return r.devices, nil }

type sequenceAttachedReader struct {
	responses [][]AttachedDevice
	index     int
}

func (r *sequenceAttachedReader) List(context.Context) ([]AttachedDevice, error) {
	if r.index >= len(r.responses) {
		return nil, nil
	}
	response := r.responses[r.index]
	r.index++
	return response, nil
}

type enumeratingFake struct {
	*fakes.Strategy
	devices []strategy.Device
}

func (f *enumeratingFake) Enumerate(context.Context) ([]strategy.Device, error) {
	return f.devices, nil
}
