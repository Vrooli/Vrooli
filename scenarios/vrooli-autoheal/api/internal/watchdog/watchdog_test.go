// Package watchdog tests
// [REQ:WATCH-DETECT-001]
package watchdog

import (
	"runtime"
	"testing"

	"vrooli-autoheal/internal/platform"
)

func TestNewDetector(t *testing.T) {
	plat := &platform.Capabilities{
		Platform:        "linux",
		SupportsSystemd: true,
	}

	d := NewDetector(plat)
	if d == nil {
		t.Fatal("expected non-nil detector")
	}
	if d.platform != plat {
		t.Error("expected platform to be set")
	}
}

func TestDetect_Linux(t *testing.T) {
	plat := &platform.Capabilities{
		Platform:        "linux",
		SupportsSystemd: true,
	}

	d := NewDetector(plat)
	status := d.Detect()

	if status == nil {
		t.Fatal("expected non-nil status")
	}

	// Loop should be running (we're calling from the API)
	if !status.LoopRunning {
		t.Error("expected LoopRunning to be true")
	}

	// Should be able to install on Linux with systemd
	if !status.CanInstall {
		t.Error("expected CanInstall to be true for Linux with systemd")
	}

	// WatchdogType should be systemd
	if status.WatchdogType != WatchdogTypeSystemd {
		t.Errorf("expected WatchdogType=%s, got=%s", WatchdogTypeSystemd, status.WatchdogType)
	}
}

func TestDetect_LinuxNoSystemd(t *testing.T) {
	plat := &platform.Capabilities{
		Platform:        "linux",
		SupportsSystemd: false,
	}

	d := NewDetector(plat)
	status := d.Detect()

	if status.CanInstall {
		t.Error("expected CanInstall to be false when systemd not available")
	}

	if status.LastError == "" {
		t.Error("expected LastError to be set when systemd not available")
	}
}

func TestDetect_MacOS(t *testing.T) {
	plat := &platform.Capabilities{
		Platform:        "macos",
		SupportsLaunchd: true,
	}

	d := NewDetector(plat)
	status := d.Detect()

	if status.WatchdogType != WatchdogTypeLaunchd {
		t.Errorf("expected WatchdogType=%s, got=%s", WatchdogTypeLaunchd, status.WatchdogType)
	}

	if !status.CanInstall {
		t.Error("expected CanInstall to be true for macOS with launchd")
	}
}

func TestDetect_Windows(t *testing.T) {
	plat := &platform.Capabilities{
		Platform:           "windows",
		SupportsWindowsSvc: true,
	}

	d := NewDetector(plat)
	status := d.Detect()

	// On non-Windows systems, Windows detection returns early with an error
	// and WatchdogType will be empty. On actual Windows, it would be WatchdogTypeWindows.
	if runtime.GOOS == "windows" {
		if status.WatchdogType != WatchdogTypeWindows {
			t.Errorf("expected WatchdogType=%s, got=%s", WatchdogTypeWindows, status.WatchdogType)
		}
	} else {
		// On non-Windows, the detection is skipped and LastError is set
		if status.LastError == "" {
			t.Error("expected LastError to be set when detecting Windows on non-Windows host")
		}
	}
}

func TestDetect_UnsupportedPlatform(t *testing.T) {
	plat := &platform.Capabilities{
		Platform: "other",
	}

	d := NewDetector(plat)
	status := d.Detect()

	if status.CanInstall {
		t.Error("expected CanInstall to be false for unsupported platform")
	}

	if status.LastError == "" {
		t.Error("expected LastError to be set for unsupported platform")
	}
}

func TestGetCached(t *testing.T) {
	plat := &platform.Capabilities{
		Platform:        "linux",
		SupportsSystemd: true,
	}

	d := NewDetector(plat)

	// First call should trigger detection
	status1 := d.GetCached()
	if status1 == nil {
		t.Fatal("expected non-nil status from GetCached")
	}

	// Second call should return cached value
	status2 := d.GetCached()
	if status2 == nil {
		t.Fatal("expected non-nil status from second GetCached")
	}
}

func TestCalculateProtectionLevel(t *testing.T) {
	plat := &platform.Capabilities{Platform: "linux", SupportsSystemd: true}
	d := NewDetector(plat)

	tests := []struct {
		name     string
		status   Status
		expected ProtectionLevel
	}{
		{
			name: "full protection",
			status: Status{
				LoopRunning:       true,
				WatchdogInstalled: true,
				WatchdogEnabled:   true,
				WatchdogRunning:   true,
			},
			expected: ProtectionFull,
		},
		{
			name: "partial protection - loop only",
			status: Status{
				LoopRunning:       true,
				WatchdogInstalled: false,
			},
			expected: ProtectionPartial,
		},
		{
			name: "partial protection - installed but not running",
			status: Status{
				LoopRunning:       true,
				WatchdogInstalled: true,
				WatchdogEnabled:   true,
				WatchdogRunning:   false,
			},
			expected: ProtectionPartial,
		},
		{
			name: "no protection",
			status: Status{
				LoopRunning:       false,
				WatchdogInstalled: false,
			},
			expected: ProtectionNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.calculateProtectionLevel(&tt.status)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetServiceTemplate_Linux(t *testing.T) {
	plat := &platform.Capabilities{Platform: "linux", SupportsSystemd: true}
	d := NewDetector(plat)

	template, err := d.GetServiceTemplate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if template == "" {
		t.Error("expected non-empty template")
	}

	// Check for key systemd directives
	if !contains(template, "[Unit]") {
		t.Error("expected [Unit] section in systemd template")
	}
	if !contains(template, "[Service]") {
		t.Error("expected [Service] section in systemd template")
	}
	if !contains(template, "Restart=always") {
		t.Error("expected Restart=always in systemd template")
	}
}

func TestGetServiceTemplate_MacOS(t *testing.T) {
	plat := &platform.Capabilities{Platform: "macos", SupportsLaunchd: true}
	d := NewDetector(plat)

	template, err := d.GetServiceTemplate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !contains(template, "com.vrooli.autoheal") {
		t.Error("expected label in launchd template")
	}
	if !contains(template, "<key>KeepAlive</key>") {
		t.Error("expected KeepAlive in launchd template")
	}
}

func TestGetServiceTemplate_Windows(t *testing.T) {
	plat := &platform.Capabilities{Platform: "windows", SupportsWindowsSvc: true}
	d := NewDetector(plat)

	template, err := d.GetServiceTemplate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !contains(template, "BootTrigger") {
		t.Error("expected BootTrigger in Windows task template")
	}
	if !contains(template, "RestartOnFailure") {
		t.Error("expected RestartOnFailure in Windows task template")
	}
}

func TestGetServiceTemplate_Unsupported(t *testing.T) {
	plat := &platform.Capabilities{Platform: "other"}
	d := NewDetector(plat)

	_, err := d.GetServiceTemplate()
	if err == nil {
		t.Error("expected error for unsupported platform")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
