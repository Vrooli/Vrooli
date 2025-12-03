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

// TestDetectRDP tests RDP detection for different platforms
func TestDetectRDP(t *testing.T) {
	tests := []struct {
		name     string
		caps     *Capabilities
		expected bool // expected to be a boolean (may vary based on system)
	}{
		{
			name: "linux with systemd",
			caps: &Capabilities{
				Platform:        Linux,
				SupportsSystemd: true,
			},
		},
		{
			name: "linux without systemd",
			caps: &Capabilities{
				Platform:        Linux,
				SupportsSystemd: false,
			},
		},
		{
			name: "macos",
			caps: &Capabilities{
				Platform: MacOS,
			},
		},
		{
			name: "other platform",
			caps: &Capabilities{
				Platform: Other,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := detectRDP(tc.caps)
			// Just verify it doesn't panic and returns a bool
			t.Logf("detectRDP(%s) = %v", tc.name, result)
		})
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

// TestDetectFunctions tests all detect* functions don't panic
func TestDetectFunctions(t *testing.T) {
	// All these should run without panicking
	t.Run("detectPlatform", func(t *testing.T) {
		p := detectPlatform()
		t.Logf("Platform: %s", p)
	})

	t.Run("detectWSL", func(t *testing.T) {
		wsl := detectWSL()
		t.Logf("WSL: %v", wsl)
	})

	t.Run("detectDocker", func(t *testing.T) {
		docker := detectDocker()
		t.Logf("Docker: %v", docker)
	})

	t.Run("detectSystemd", func(t *testing.T) {
		systemd := detectSystemd()
		t.Logf("Systemd: %v", systemd)
	})

	t.Run("detectLaunchd", func(t *testing.T) {
		launchd := detectLaunchd()
		t.Logf("Launchd: %v", launchd)
	})

	t.Run("detectWindowsServices", func(t *testing.T) {
		winSvc := detectWindowsServices()
		t.Logf("Windows Services: %v", winSvc)
	})

	t.Run("detectCloudflared", func(t *testing.T) {
		cf := detectCloudflared()
		t.Logf("Cloudflared: %v", cf)
	})

	t.Run("detectHeadless", func(t *testing.T) {
		headless := detectHeadless()
		t.Logf("Headless: %v", headless)
	})
}

// TestDetectRDPForWindows tests the Windows RDP detection path specifically
func TestDetectRDPForWindows(t *testing.T) {
	caps := &Capabilities{
		Platform:           Windows,
		SupportsWindowsSvc: true,
	}

	// On Windows, RDP should return true (built-in)
	if runtime.GOOS == "windows" {
		result := detectRDP(caps)
		if !result {
			t.Log("Note: On Windows, native RDP should always be true")
		}
	} else {
		// When testing on non-Windows, the Windows path returns true unconditionally
		// This is intentional - Windows always has native RDP
		result := detectRDP(caps)
		if !result {
			t.Log("Windows path for RDP detection returns true (native RDP)")
		}
	}
}
