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
func TestDetectCollectsFreshViews(t *testing.T) {
	// Note: Because of sync.Once, we can only test that Detect returns non-nil
	// and returns the same value on repeated calls
	caps1 := Detect()
	caps2 := Detect()

	if caps1 == nil {
		t.Fatal("Detect() returned nil")
	}

	if caps2 == nil {
		t.Fatal("second Detect() returned nil")
	}
	if caps1 == caps2 {
		t.Error("Detect() should return a fresh view on subsequent calls")
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

// TestSystemdDetection verifies systemd detection logic
func TestSystemdDetection(t *testing.T) {
	hasSystemd := detectSystemd()

	// On non-Linux, should be false
	if runtime.GOOS != "linux" && hasSystemd {
		t.Errorf("detectSystemd() = true on non-Linux platform %s", runtime.GOOS)
	}

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

// TestCapabilitiesStructure verifies Capabilities struct fields are properly set
func TestCapabilitiesStructure(t *testing.T) {
	caps := &Capabilities{
		Platform:            Linux,
		SupportsRDP:         true,
		SupportsSystemd:     true,
		SupportsLaunchd:     false,
		SupportsWindowsSvc:  false,
		IsHeadlessServer:    true,
		HasDocker:           true,
		IsWSL:               false,
		SupportsCloudflared: true,
	}

	if caps.Platform != Linux {
		t.Errorf("Platform = %v, want linux", caps.Platform)
	}
	if !caps.SupportsRDP {
		t.Error("SupportsRDP should be true")
	}
	if !caps.SupportsSystemd {
		t.Error("SupportsSystemd should be true")
	}
	if caps.SupportsLaunchd {
		t.Error("SupportsLaunchd should be false")
	}
	if caps.SupportsWindowsSvc {
		t.Error("SupportsWindowsSvc should be false")
	}
	if !caps.IsHeadlessServer {
		t.Error("IsHeadlessServer should be true")
	}
	if !caps.HasDocker {
		t.Error("HasDocker should be true")
	}
	if caps.IsWSL {
		t.Error("IsWSL should be false")
	}
	if !caps.SupportsCloudflared {
		t.Error("SupportsCloudflared should be true")
	}
}
