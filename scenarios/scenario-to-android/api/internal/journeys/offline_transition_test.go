package journeys

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

type offlineFixtureDevice struct {
	*scenarioInstallFakeDevice
	acquiredTarget string
	released       bool
	attachRoutes   []string
	attachErr      error
	recoveryErr    error
	offlineErr     error
	terminalState  *InstalledPackageState
	logStarts      int
}

func (f *offlineFixtureDevice) Acquire(_ context.Context, target, _ string, _ time.Duration) (Lease, error) {
	f.acquiredTarget = target
	return Lease{ID: "offline-lease", DeviceID: target, Token: "offline-token"}, nil
}

func (f *offlineFixtureDevice) Release(context.Context, Lease) error {
	f.released = true
	return nil
}

func (f *offlineFixtureDevice) AttachWebView(context.Context, Lease, string) (WebViewAttachment, error) {
	return WebViewAttachment{}, errors.New("unrouted attachment must never run")
}

func (f *offlineFixtureDevice) AttachWebViewWithTransport(_ context.Context, lease Lease, pkg, transport string) (WebViewAttachment, error) {
	f.attachRoutes = append(f.attachRoutes, transport)
	if lease.DeviceID != f.acquiredTarget || pkg != "com.example.generated" {
		return WebViewAttachment{}, errors.New("attachment lost device/package identity")
	}
	return WebViewAttachment{CDPEndpoint: "http://127.0.0.1:9222", RendererID: "fixture-renderer"}, f.attachErr
}

func (f *offlineFixtureDevice) Execute(ctx context.Context, lease Lease, action string, args map[string]string) (ActionResult, error) {
	result, err := f.scenarioInstallFakeDevice.Execute(ctx, lease, action, args)
	if action == "network" {
		if args["target"] == "offline" {
			return result, f.offlineErr
		}
		if args["target"] == "online" {
			return result, f.recoveryErr
		}
	}
	return result, err
}

func (f *offlineFixtureDevice) ReadPackageState(ctx context.Context, lease Lease, pkg string) (InstalledPackageState, error) {
	state, err := f.scenarioInstallFakeDevice.ReadPackageState(ctx, lease, pkg)
	if f.packageReads > 1 && f.terminalState != nil {
		return *f.terminalState, err
	}
	return state, err
}

func (f *offlineFixtureDevice) StartLogCapture(context.Context, Lease) error {
	f.logStarts++
	return nil
}

func (f *offlineFixtureDevice) StopLogCapture(context.Context, Lease) (LogCaptureArtifact, error) {
	return LogCaptureArtifact{}, nil
}

func offlineFixtureRequest(t *testing.T) deliveryramp.DriverRequest {
	request := appInstallRequest(t, "verify")
	request.Plan = InstalledAppOfflineTransitionPlan()
	request.Cell.Target.ID = "android-exact-emulator"
	request.Cell.Target.Capabilities = append(request.Cell.Target.Capabilities, "network-control")
	return request
}

func newOfflineFixtureDevice() *offlineFixtureDevice {
	return &offlineFixtureDevice{scenarioInstallFakeDevice: &scenarioInstallFakeDevice{packageState: InstalledPackageState{Installed: true, Version: "1.2.3", VersionCode: 12}}}
}

func requireFixturePreserved(t *testing.T, device *offlineFixtureDevice) {
	t.Helper()
	require.NotContains(t, device.actions, "install")
	require.NotContains(t, device.actions, "install-artifact")
	require.NotContains(t, device.actions, "uninstall")
	require.Zero(t, device.startedRecording)
	require.Zero(t, device.logStarts)
	require.True(t, device.released)
	require.Equal(t, "off", device.state.ScreenState)
	require.Equal(t, "locked", device.state.LockState)
}

func fixtureNetworkTargets(device *offlineFixtureDevice) []string {
	var targets []string
	for index, action := range device.actions {
		if action == "network" {
			targets = append(targets, device.calls[index]["target"])
		}
	}
	return targets
}

func TestInstalledOfflineTransitionPassesBothBASStatesAndKeepsPackage(t *testing.T) {
	device, bas := newOfflineFixtureDevice(), &countingBAS{}
	request := offlineFixtureRequest(t)
	result, err := (Driver{Devices: device, BAS: bas, DeviceTransport: "usb"}).Execute(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, deliveryramp.DispositionPass, result.Disposition)
	require.Equal(t, request.Cell.Target.ID, device.acquiredTarget)
	require.Equal(t, []string{"offline", "online"}, fixtureNetworkTargets(device))
	require.Equal(t, []string{"usb", "usb"}, device.attachRoutes)
	require.Len(t, bas.preflightCalls, 2)
	require.Len(t, bas.requests, 2)
	require.Equal(t, "offline_transition/web-flow-offline", bas.requests[0].StepID)
	require.Equal(t, "offline_transition/web-flow-online", bas.requests[1].StepID)
	for _, basRequest := range bas.requests {
		require.Equal(t, request.RunID, basRequest.RunID)
		require.Equal(t, "offline-lease", basRequest.IsolationLeaseID)
		require.Equal(t, request.Artifact.ImmutableRef, basRequest.Artifact.ImmutableRef)
	}
	for _, step := range result.Steps {
		if step.Action == "bas-flow" && step.ChapterID == "offline_transition" {
			require.Equal(t, "observed", step.AssertionStatus)
		}
	}
	require.Equal(t, 2, device.packageReads, "preflight and terminal identity must both be read")
	require.Equal(t, "device-cleanup/package-identity", result.Steps[len(result.Steps)-2].ID)
	require.Equal(t, "observed", result.Steps[len(result.Steps)-2].AssertionStatus)
	requireFixturePreserved(t, device)
}

func TestInstalledOfflineTransitionRefusesUnsafePreconditionsBeforeMutation(t *testing.T) {
	tests := []struct {
		name  string
		alter func(*deliveryramp.DriverRequest, *offlineFixtureDevice, *countingBAS, *Driver)
	}{
		{"wireless", func(_ *deliveryramp.DriverRequest, _ *offlineFixtureDevice, _ *countingBAS, driver *Driver) {
			driver.DeviceTransport = "wireless"
		}},
		{"missing exact target", func(r *deliveryramp.DriverRequest, _ *offlineFixtureDevice, _ *countingBAS, _ *Driver) {
			r.Cell.Target.ID = ""
		}},
		{"wrong fixture", func(r *deliveryramp.DriverRequest, _ *offlineFixtureDevice, _ *countingBAS, _ *Driver) {
			r.Artifact.Metadata["scenario_name"] = "user-app"
		}},
		{"digest mismatch", func(r *deliveryramp.DriverRequest, _ *offlineFixtureDevice, _ *countingBAS, _ *Driver) {
			r.Artifact.Checksum = "sha256:wrong"
		}},
		{"no network control", func(r *deliveryramp.DriverRequest, _ *offlineFixtureDevice, _ *countingBAS, _ *Driver) {
			r.Cell.Target.Capabilities = []string{"screen-control", "app-lifecycle"}
		}},
		{"absent package", func(_ *deliveryramp.DriverRequest, d *offlineFixtureDevice, _ *countingBAS, _ *Driver) {
			d.packageState.Installed = false
		}},
		{"unknown version code", func(_ *deliveryramp.DriverRequest, d *offlineFixtureDevice, _ *countingBAS, _ *Driver) {
			d.packageState.VersionCode = 0
		}},
		{"different version code", func(_ *deliveryramp.DriverRequest, d *offlineFixtureDevice, _ *countingBAS, _ *Driver) {
			d.packageState.VersionCode = 13
		}},
		{"different version name", func(_ *deliveryramp.DriverRequest, d *offlineFixtureDevice, _ *countingBAS, _ *Driver) {
			d.packageState.Version = "other"
		}},
		{"package state unavailable", func(_ *deliveryramp.DriverRequest, d *offlineFixtureDevice, _ *countingBAS, _ *Driver) {
			d.packageStateErr = errors.New("readback unavailable")
		}},
		{"flow unregistered", func(_ *deliveryramp.DriverRequest, _ *offlineFixtureDevice, b *countingBAS, _ *Driver) {
			b.preflightErr = errors.New("flow is not registered")
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			device, bas := newOfflineFixtureDevice(), &countingBAS{}
			request := offlineFixtureRequest(t)
			driver := Driver{Devices: device, BAS: bas, DeviceTransport: "usb"}
			test.alter(&request, device, bas, &driver)
			result, err := driver.Execute(context.Background(), request)
			require.True(t, err != nil || result.Disposition == deliveryramp.DispositionFailed)
			require.Empty(t, device.actions, "precondition refusal must precede all mutations")
			require.Zero(t, bas.calls)
			require.Zero(t, device.startedRecording)
		})
	}
}

func TestInstalledOfflineTransitionRecoversAfterAttachFailure(t *testing.T) {
	device := newOfflineFixtureDevice()
	failure := errors.New("USB WebView attachment failed")
	device.attachErr = failure
	result, err := (Driver{Devices: device, BAS: &countingBAS{}, DeviceTransport: "usb"}).Execute(context.Background(), offlineFixtureRequest(t))
	require.ErrorIs(t, err, failure)
	require.Equal(t, deliveryramp.DispositionFailed, result.Disposition)
	require.Equal(t, []string{"offline", "online"}, fixtureNetworkTargets(device))
	require.Equal(t, 2, device.packageReads)
	requireFixturePreserved(t, device)
}

func TestInstalledOfflineTransitionPreservesBASAndRecoveryErrors(t *testing.T) {
	device := newOfflineFixtureDevice()
	failure, recovery := errors.New("offline assertion failed"), errors.New("online recovery failed")
	device.recoveryErr = recovery
	result, err := (Driver{Devices: device, BAS: failingBAS{err: failure}, DeviceTransport: "usb"}).Execute(context.Background(), offlineFixtureRequest(t))
	require.ErrorIs(t, err, failure)
	require.ErrorIs(t, err, recovery)
	require.Equal(t, deliveryramp.DispositionFailed, result.Disposition)
	require.Equal(t, []string{"offline", "online"}, fixtureNetworkTargets(device))
	var recoveryStep *deliveryramp.JourneyStep
	for index := range result.Steps {
		if result.Steps[index].ID == "device-cleanup/network-online" {
			recoveryStep = &result.Steps[index]
		}
	}
	require.NotNil(t, recoveryStep)
	require.Equal(t, deliveryramp.StepFailed, recoveryStep.Disposition)
	require.Contains(t, recoveryStep.Error, recovery.Error())
	requireFixturePreserved(t, device)
}

func TestInstalledOfflineTransitionFailsUnknownTerminalPackageIdentity(t *testing.T) {
	device := newOfflineFixtureDevice()
	device.terminalState = &InstalledPackageState{Installed: true, Version: "1.2.3"}
	result, err := (Driver{Devices: device, BAS: &countingBAS{}, DeviceTransport: "usb"}).Execute(context.Background(), offlineFixtureRequest(t))
	require.ErrorContains(t, err, "terminal package readback failed")
	require.Equal(t, deliveryramp.DispositionFailed, result.Disposition)
	require.Equal(t, []string{"offline", "online"}, fixtureNetworkTargets(device))
	requireFixturePreserved(t, device)
}

func TestInstalledOfflineTransitionRecoversAfterUncertainOfflineAction(t *testing.T) {
	device := newOfflineFixtureDevice()
	device.offlineErr = context.DeadlineExceeded
	bas := &countingBAS{}
	result, err := (Driver{Devices: device, BAS: bas, DeviceTransport: "usb"}).Execute(context.Background(), offlineFixtureRequest(t))
	require.NoError(t, err, "failed device actions are retained as structured failures")
	require.Equal(t, deliveryramp.DispositionFailed, result.Disposition)
	require.Equal(t, []string{"offline", "online"}, fixtureNetworkTargets(device))
	require.Zero(t, bas.calls)
	requireFixturePreserved(t, device)
}
