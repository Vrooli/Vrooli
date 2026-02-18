// Package watchdog tests for installer functionality
// [REQ:WATCH-INSTALL-001]
package watchdog

import (
	"context"
	"os/user"
	"runtime"
	"strings"
	"testing"
	"time"

	"vrooli-autoheal/internal/platform"
)

func TestInstallOptions(t *testing.T) {
	// Test that InstallOptions struct is correctly initialized
	opts := InstallOptions{
		UseSystemService: false,
		EnableLingering:  true,
	}

	if opts.UseSystemService {
		t.Error("expected UseSystemService to be false")
	}
	if !opts.EnableLingering {
		t.Error("expected EnableLingering to be true")
	}
}

func TestInstallResult(t *testing.T) {
	// Test InstallResult struct
	result := InstallResult{
		Success:       true,
		Message:       "Test message",
		ServicePath:   "/test/path",
		NeedsLinger:   false,
		LingerCommand: "",
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}
	if result.Message != "Test message" {
		t.Errorf("expected Message='Test message', got '%s'", result.Message)
	}
}

func TestUninstallResult(t *testing.T) {
	// Test UninstallResult struct
	result := UninstallResult{
		Success: true,
		Message: "Uninstalled successfully",
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}
}

func TestInstallStatus(t *testing.T) {
	// Test InstallStatus struct
	status := InstallStatus{
		Installed:        true,
		Enabled:          true,
		Running:          true,
		BootProtected:    true,
		ServicePath:      "/etc/systemd/system/test.service",
		WatchdogType:     "systemd",
		CanInstall:       true,
		NeedsLinger:      false,
		ProtectionLevel:  "full",
		LastChecked:      time.Now().UTC().Format(time.RFC3339),
		RecommendedSetup: "user",
	}

	if !status.BootProtected {
		t.Error("expected BootProtected to be true")
	}
	if status.ProtectionLevel != "full" {
		t.Errorf("expected ProtectionLevel='full', got '%s'", status.ProtectionLevel)
	}
}

func TestInstall_UnsupportedPlatform(t *testing.T) {
	// Test install on unsupported platform
	plat := &platform.Capabilities{
		Platform: "unsupported",
	}

	d := NewDetector(plat)
	ctx := context.Background()
	opts := InstallOptions{}

	result := d.Install(ctx, opts)

	if result.Success {
		t.Error("expected install to fail on unsupported platform")
	}
	if result.Error == "" {
		t.Error("expected error message for unsupported platform")
	}
}

func TestUninstall_UnsupportedPlatform(t *testing.T) {
	// Test uninstall on unsupported platform
	plat := &platform.Capabilities{
		Platform: "unsupported",
	}

	d := NewDetector(plat)
	ctx := context.Background()

	result := d.Uninstall(ctx)

	if result.Success {
		t.Error("expected uninstall to fail on unsupported platform")
	}
}

func TestEnableLingering_NonLinux(t *testing.T) {
	// Test EnableLingering on non-Linux platform
	plat := &platform.Capabilities{
		Platform: "macos",
	}

	d := NewDetector(plat)
	ctx := context.Background()

	result := d.EnableLingering(ctx)

	if result.Success {
		t.Error("expected EnableLingering to fail on non-Linux")
	}
	if result.Error != "not linux" {
		t.Errorf("expected error='not linux', got '%s'", result.Error)
	}
}

func TestGetInstallStatus(t *testing.T) {
	plat := &platform.Capabilities{
		Platform:        "linux",
		SupportsSystemd: true,
	}

	d := NewDetector(plat)
	status := d.GetInstallStatus()

	if status == nil {
		t.Fatal("expected non-nil status")
	}

	// Verify required fields are populated
	if status.LastChecked == "" {
		t.Error("expected LastChecked to be populated")
	}
	if status.RecommendedSetup == "" {
		t.Error("expected RecommendedSetup to be populated")
	}
}

func TestGetRecommendedSetup(t *testing.T) {
	tests := []struct {
		platformType platform.Type
		expected     string
	}{
		{platform.Linux, "user"},
		{platform.MacOS, "user"},
		{platform.Windows, "system"},
		{platform.Other, ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.platformType), func(t *testing.T) {
			plat := &platform.Capabilities{Platform: tt.platformType}
			d := NewDetector(plat)

			result := d.getRecommendedSetup()
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// Integration test that runs on actual platform
func TestInstall_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// This test only runs on the current platform
	plat := platform.Detect()
	d := NewDetector(plat)

	// Don't actually install in tests, just verify the methods don't panic
	status := d.GetInstallStatus()
	t.Logf("Current platform: %s", plat.Platform)
	t.Logf("Can install: %v", status.CanInstall)
	t.Logf("Installed: %v", status.Installed)
	t.Logf("Protection level: %s", status.ProtectionLevel)
}

func TestInstallLinux_NoSystemd(t *testing.T) {
	plat := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: false,
	}

	d := NewDetector(plat)
	ctx := context.Background()
	opts := InstallOptions{}

	result := d.Install(ctx, opts)

	if result.Success {
		t.Error("expected install to fail when systemd not available")
	}
	if result.Error != "systemd not supported" {
		t.Errorf("expected error='systemd not supported', got '%s'", result.Error)
	}
}

func TestInstallMacOS_NoLaunchd(t *testing.T) {
	plat := &platform.Capabilities{
		Platform:        platform.MacOS,
		SupportsLaunchd: false,
	}

	d := NewDetector(plat)
	ctx := context.Background()
	opts := InstallOptions{}

	result := d.Install(ctx, opts)

	if result.Success {
		t.Error("expected install to fail when launchd not available")
	}
	if result.Error != "launchd not supported" {
		t.Errorf("expected error='launchd not supported', got '%s'", result.Error)
	}
}

func TestInstallWindows_NotOnWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	plat := &platform.Capabilities{
		Platform:           platform.Windows,
		SupportsWindowsSvc: true,
	}

	d := NewDetector(plat)
	ctx := context.Background()
	opts := InstallOptions{}

	result := d.Install(ctx, opts)

	if result.Success {
		t.Error("expected install to fail when not on Windows")
	}
	if result.Error != "not windows" {
		t.Errorf("expected error='not windows', got '%s'", result.Error)
	}
}

func TestVerifyLoopBinaryExists_UsesProbeEnvironmentAndHome(t *testing.T) {
	plat := &platform.Capabilities{Platform: platform.Linux}
	probe := newFakeProbe()
	probe.userHomeDirPath = "/home/tester"
	probe.env["VROOLI_ROOT"] = "/custom/vrooli"
	probe.stats["/custom/vrooli/scenarios/vrooli-autoheal/cli/vrooli-autoheal-loop"] = nil

	d := detectorWithProbe(plat, probe)
	if err := d.verifyLoopBinaryExists(); err != nil {
		t.Fatalf("expected loop binary verification to pass, got error: %v", err)
	}
}

func TestVerifyLoopBinaryExists_MissingWindowsBinaryIncludesBuildHint(t *testing.T) {
	plat := &platform.Capabilities{Platform: platform.Windows}
	probe := newFakeProbe()
	probe.goosValue = "windows"
	probe.userHomeDirPath = "C:\\Users\\tester"

	d := detectorWithProbe(plat, probe)
	err := d.verifyLoopBinaryExists()
	if err == nil {
		t.Fatal("expected missing loop binary error")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "vrooli-autoheal-loop.exe") {
		t.Fatalf("expected windows binary path in error, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "go build -o vrooli-autoheal-loop.exe ./cli/loop") {
		t.Fatalf("expected windows build hint in error, got: %s", errMsg)
	}
}

func TestInstallWindows_NotOnWindows_UsesProbeGOOS(t *testing.T) {
	plat := &platform.Capabilities{
		Platform:           platform.Windows,
		SupportsWindowsSvc: true,
	}
	probe := newFakeProbe()
	probe.goosValue = "linux"
	d := detectorWithProbe(plat, probe)

	result := d.installWindows(context.Background(), InstallOptions{})
	if result.Success {
		t.Error("expected install to fail when probe GOOS is not windows")
	}
	if result.Error != "not windows" {
		t.Errorf("expected error='not windows', got '%s'", result.Error)
	}
}

func TestUninstallWindows_NotOnWindows_UsesProbeGOOS(t *testing.T) {
	plat := &platform.Capabilities{
		Platform:           platform.Windows,
		SupportsWindowsSvc: true,
	}
	probe := newFakeProbe()
	probe.goosValue = "linux"
	d := detectorWithProbe(plat, probe)

	result := d.uninstallWindows(context.Background())
	if result.Success {
		t.Error("expected uninstall to fail when probe GOOS is not windows")
	}
	if result.Error != "not windows" {
		t.Errorf("expected error='not windows', got '%s'", result.Error)
	}
}

func TestInstallLinux_UserService_UsesProbeSideEffectSeam(t *testing.T) {
	plat := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}
	probe := newFakeProbe()
	probe.goosValue = "linux"
	probe.userHomeDirPath = "/home/tester"
	probe.currentUserValue = &user.User{Username: "tester"}
	probe.env["VROOLI_ROOT"] = "/workspace/Vrooli"
	probe.stats["/workspace/Vrooli/scenarios/vrooli-autoheal/cli/vrooli-autoheal-loop"] = nil
	probe.stats["/var/lib/systemd/linger/tester"] = nil
	probe.commandOutputs[commandKey("systemctl", "--user", "daemon-reload")] = fakeCommandResult{}
	probe.commandOutputs[commandKey("systemctl", "--user", "enable", "vrooli-autoheal")] = fakeCommandResult{}
	probe.commandOutputs[commandKey("systemctl", "--user", "start", "vrooli-autoheal")] = fakeCommandResult{}

	d := detectorWithProbe(plat, probe)
	result := d.Install(context.Background(), InstallOptions{})
	if !result.Success {
		t.Fatalf("expected install success, got error: %s", result.Error)
	}
	servicePath := "/home/tester/.config/systemd/user/vrooli-autoheal.service"
	if result.ServicePath != servicePath {
		t.Fatalf("expected servicePath %q, got %q", servicePath, result.ServicePath)
	}
	if _, ok := probe.writtenFiles[servicePath]; !ok {
		t.Fatalf("expected service file to be written via probe seam at %s", servicePath)
	}
}

func TestUninstallLinux_UserService_UsesProbeSideEffectSeam(t *testing.T) {
	plat := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}
	probe := newFakeProbe()
	probe.userHomeDirPath = "/home/tester"
	servicePath := "/home/tester/.config/systemd/user/vrooli-autoheal.service"
	probe.stats[servicePath] = nil
	probe.commandRuns[commandKey("systemctl", "--user", "stop", "vrooli-autoheal")] = nil
	probe.commandRuns[commandKey("systemctl", "--user", "disable", "vrooli-autoheal")] = nil
	probe.commandRuns[commandKey("systemctl", "--user", "daemon-reload")] = nil

	d := detectorWithProbe(plat, probe)
	result := d.uninstallLinux(context.Background())
	if !result.Success {
		t.Fatalf("expected uninstall success, got error: %s", result.Error)
	}
	if !probe.removedFiles[servicePath] {
		t.Fatalf("expected user service file removal via probe seam for %s", servicePath)
	}
}
