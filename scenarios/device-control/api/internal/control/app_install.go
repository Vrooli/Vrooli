package control

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"device-control/internal/androidartifact"
	"device-control/strategy"
)

// InstallAppRequest is the "try my app on the phone" check: install a local APK,
// prove it by package state, and optionally open it and capture the screen. The
// caller's request is the operator's explicit install confirmation.
type InstallAppRequest struct {
	Device     string `json:"device"`
	Actor      string `json:"actor"`
	APK        string `json:"apk"`
	Open       bool   `json:"open,omitempty"`
	Screenshot bool   `json:"screenshot,omitempty"`
}

type InstallAppResult struct {
	Status            string   `json:"status"` // installed_verified | failed
	DeviceID          string   `json:"device_id"`
	Package           string   `json:"package,omitempty"`
	VersionName       string   `json:"version_name,omitempty"`
	VersionCode       int64    `json:"version_code,omitempty"`
	Detail            string   `json:"detail,omitempty"`
	LaunchStatus      string   `json:"launch_status,omitempty"`
	ForegroundPackage string   `json:"foreground_package,omitempty"`
	EvidenceIDs       []string `json:"evidence_ids,omitempty"`
	NextActions       []string `json:"next_actions,omitempty"`
}

const maxInstallAPKBytes = 300 << 20

// apkInspector reads the package identity from the APK itself, never from the
// caller, so the verified package is the one actually installed.
func (s *Service) apkIdentity(ctx context.Context, path string) (androidartifact.APKIdentity, error) {
	s.mu.Lock()
	inspector := s.apkInspector
	s.mu.Unlock()
	if inspector == nil {
		inspector = androidartifact.AAPTInspector{ADBPath: "adb"}
	}
	return inspector.Inspect(ctx, path)
}

func validateAPKPath(path string) error {
	if !filepath.IsAbs(path) || !strings.EqualFold(filepath.Ext(path), ".apk") {
		return fmt.Errorf("apk must be an absolute path to a .apk file")
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("apk %q is not a readable regular file", path)
	}
	if info.Size() == 0 || info.Size() > maxInstallAPKBytes {
		return fmt.Errorf("apk size must be between 1 byte and %d MiB", maxInstallAPKBytes>>20)
	}
	return nil
}

// InstallApp installs, verifies, and optionally opens and screenshots. Failures
// are results with next actions, not errors, so a caller always sees the cause.
func (s *Service) InstallApp(ctx context.Context, request InstallAppRequest) (InstallAppResult, error) {
	result := InstallAppResult{Status: "failed"}
	if err := validateAPKPath(request.APK); err != nil {
		result.Detail = err.Error()
		return result, err
	}
	device, err := s.resolveExactOrUniqueDevice(ctx, request.Device)
	if err != nil {
		result.Detail = err.Error()
		return result, err
	}
	result.DeviceID = device.ID
	identity, err := s.apkIdentity(ctx, request.APK)
	if err != nil {
		result.Detail = "read APK identity: " + err.Error()
		return result, nil
	}
	result.Package, result.VersionName, result.VersionCode = identity.PackageName, identity.VersionName, identity.VersionCode
	actor := firstNonEmpty(strings.TrimSpace(request.Actor), "app-install")
	s.prepareScreen(ctx, device, CaptureRequest{Device: device.ID, Actor: actor})
	if installErr := s.installAndVerify(ctx, device, actor, request.APK, identity, &result); installErr != nil {
		result.Detail = installErr.Error()
		result.NextActions = s.launchGapNextActions(ctx, device)
		return result, nil
	}
	result.Status = "installed_verified"
	if request.Open || request.Screenshot {
		launch, _ := s.LaunchApp(ctx, LaunchAppRequest{Device: device.ID, App: identity.PackageName, Actor: actor})
		result.LaunchStatus, result.ForegroundPackage = launch.Status, launch.ForegroundPackage
		result.NextActions = append(result.NextActions, launch.NextActions...)
		if launch.Status != "foreground_verified" {
			result.Detail = firstNonEmpty(launch.Detail, "the app did not reach the foreground")
		}
	}
	if request.Screenshot {
		capture, _ := s.Capture(ctx, CaptureRequest{Device: device.ID, Kind: "screenshot", Actor: actor})
		for _, reference := range capture.Evidence {
			result.EvidenceIDs = append(result.EvidenceIDs, reference.ID)
		}
		result.NextActions = append(result.NextActions, capture.NextActions...)
	}
	return result, nil
}

// installAndVerify runs install and package-state under one lease. The device
// transport that declares app lifecycle also performs the install.
func (s *Service) installAndVerify(ctx context.Context, device Device, actor, apk string, identity androidartifact.APKIdentity, result *InstallAppResult) (err error) {
	lease, err := s.AcquireContext(ctx, device.ID, actor, 5*time.Minute)
	if err != nil {
		return err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if _, releaseErr := s.ReleaseContext(cleanup, lease.ID); releaseErr != nil && err == nil {
			err = fmt.Errorf("install finished but lease release failed: %w", releaseErr)
		}
	}()
	target, err := s.resolveLifecycleTarget(ctx, device.ID, lease.LeaseToken, "install")
	if err != nil {
		return err
	}
	actuator, ok := target.adapter.(strategy.InputActuator)
	if !ok {
		return &strategy.UnsupportedCapabilityError{Capability: strategy.CapAppLifecycle, Operation: "install"}
	}
	if err = s.ValidateLease(ctx, device.ID, lease.LeaseToken); err != nil {
		return err
	}
	installErr := actuator.Actuate(ctx, strategy.Actuation{Action: "install", Package: identity.PackageName, Value: apk})
	state, stateErr := s.runLifecycle(ctx, target, AppLifecycleOperation{Actor: actor, LeaseToken: lease.LeaseToken, Confirmed: true, AppLifecycleRequest: strategy.AppLifecycleRequest{Operation: "package-state", Package: identity.PackageName}})
	if installErr != nil {
		return fmt.Errorf("install failed: %w", installErr)
	}
	if stateErr != nil || !state.Result.Installed {
		return fmt.Errorf("install was accepted but package-state does not show %s installed", identity.PackageName)
	}
	if identity.VersionCode > 0 && state.Result.VersionCode > 0 && state.Result.VersionCode != identity.VersionCode {
		return fmt.Errorf("installed version code %d differs from the APK's %d", state.Result.VersionCode, identity.VersionCode)
	}
	return nil
}
