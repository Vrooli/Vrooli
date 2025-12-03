// Package platform tests
// [REQ:PLAT-DETECT-001] [REQ:PLAT-DETECT-002] [REQ:PLAT-DETECT-003] [REQ:PLAT-DETECT-004]
package platform

import (
	"runtime"
	"testing"
)

// TestDetectPlatform verifies platform detection returns valid type
// [REQ:PLAT-DETECT-001]
func TestDetectPlatform(t *testing.T) {
	platform := detectPlatform()

	validPlatforms := map[Type]bool{
		Linux:   true,
		Windows: true,
		MacOS:   true,
		Other:   true,
	}

	if !validPlatforms[platform] {
		t.Errorf("detectPlatform() returned invalid platform: %v", platform)
	}

	// Verify platform matches GOOS
	expected := map[string]Type{
		"linux":   Linux,
		"darwin":  MacOS,
		"windows": Windows,
	}

	if exp, ok := expected[runtime.GOOS]; ok {
		if platform != exp {
			t.Errorf("detectPlatform() = %v, want %v for GOOS=%s", platform, exp, runtime.GOOS)
		}
	} else if platform != Other {
		t.Errorf("detectPlatform() = %v, want Other for unknown GOOS=%s", platform, runtime.GOOS)
	}
}

// TestDetectCapabilities verifies capabilities detection returns valid struct
// [REQ:PLAT-DETECT-002]
func TestDetectCapabilities(t *testing.T) {
	caps := detect()

	if caps == nil {
		t.Fatal("detect() returned nil")
	}

	// Verify platform is set
	if caps.Platform == "" {
		t.Error("Platform field is empty")
	}

	// Platform-specific validation
	switch caps.Platform {
	case Linux:
		// Linux should not support launchd or Windows services
		if caps.SupportsLaunchd {
			t.Error("Linux should not support launchd")
		}
		if caps.SupportsWindowsSvc {
			t.Error("Linux should not support Windows services")
		}

	case MacOS:
		// macOS should not support systemd or Windows services
		if caps.SupportsSystemd {
			t.Error("macOS should not support systemd")
		}
		if caps.SupportsWindowsSvc {
			t.Error("macOS should not support Windows services")
		}

	case Windows:
		// Windows should not support systemd or launchd
		if caps.SupportsSystemd {
			t.Error("Windows should not support systemd")
		}
		if caps.SupportsLaunchd {
			t.Error("Windows should not support launchd")
		}
	}
}

// TestDetectCached verifies caching works correctly
// [REQ:PLAT-DETECT-003]
func TestDetectCached(t *testing.T) {
	// Note: Because of sync.Once, we can only test that Detect returns non-nil
	// and returns the same value on repeated calls
	caps1 := Detect()
	caps2 := Detect()

	if caps1 == nil {
		t.Fatal("Detect() returned nil")
	}

	if caps1 != caps2 {
		t.Error("Detect() should return cached value on subsequent calls")
	}
}

// TestWSLDetection verifies WSL detection logic
// [REQ:PLAT-DETECT-004]
func TestWSLDetection(t *testing.T) {
	// WSL detection should only potentially return true on Linux
	if runtime.GOOS != "linux" {
		isWSL := detectWSL()
		if isWSL {
			t.Errorf("detectWSL() = true on non-Linux platform %s", runtime.GOOS)
		}
	}
	// On Linux, we can't assert the result since it depends on environment
}

// TestDockerDetection verifies Docker detection doesn't panic
func TestDockerDetection(t *testing.T) {
	// Just verify it doesn't panic
	hasDocker := detectDocker()
	t.Logf("Docker available: %v", hasDocker)
}

// TestSystemdDetection verifies systemd detection logic
func TestSystemdDetection(t *testing.T) {
	hasSystemd := detectSystemd()

	// On non-Linux, should be false
	if runtime.GOOS != "linux" && hasSystemd {
		t.Errorf("detectSystemd() = true on non-Linux platform %s", runtime.GOOS)
	}

	t.Logf("Systemd available: %v", hasSystemd)
}

// TestLaunchdDetection verifies launchd detection logic
func TestLaunchdDetection(t *testing.T) {
	hasLaunchd := detectLaunchd()

	// On non-macOS, should be false
	if runtime.GOOS != "darwin" && hasLaunchd {
		t.Errorf("detectLaunchd() = true on non-macOS platform %s", runtime.GOOS)
	}

	t.Logf("Launchd available: %v", hasLaunchd)
}

// TestWindowsServicesDetection verifies Windows services detection logic
func TestWindowsServicesDetection(t *testing.T) {
	hasSvc := detectWindowsServices()

	// On non-Windows, should be false
	if runtime.GOOS != "windows" && hasSvc {
		t.Errorf("detectWindowsServices() = true on non-Windows platform %s", runtime.GOOS)
	}

	t.Logf("Windows services available: %v", hasSvc)
}

// TestCloudflaredDetection verifies cloudflared detection doesn't panic
func TestCloudflaredDetection(t *testing.T) {
	// Just verify it doesn't panic
	hasCF := detectCloudflared()
	t.Logf("Cloudflared available: %v", hasCF)
}

// TestHeadlessDetection verifies headless detection doesn't panic
func TestHeadlessDetection(t *testing.T) {
	// Just verify it doesn't panic
	isHeadless := detectHeadless()
	t.Logf("Headless server: %v", isHeadless)
}

// TestPlatformType verifies Type constants are correct
func TestPlatformType(t *testing.T) {
	tests := []struct {
		platform Type
		expected string
	}{
		{Linux, "linux"},
		{Windows, "windows"},
		{MacOS, "macos"},
		{Other, "other"},
	}

	for _, tc := range tests {
		if string(tc.platform) != tc.expected {
			t.Errorf("Type %v = %q, want %q", tc.platform, tc.platform, tc.expected)
		}
	}
}
