// Package watchdog tests for installer functionality
// [REQ:WATCH-INSTALL-001]
package watchdog

import (
	"context"
	"runtime"
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
