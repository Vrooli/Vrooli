// Package infra provides infrastructure health checks
// Tests for decision boundary functions
// [REQ:INFRA-RDP-001] [REQ:INFRA-CLOUDFLARED-001]
package infra

import (
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

// TestSelectCloudflaredVerifyMethod tests cloudflared verification method selection
func TestSelectCloudflaredVerifyMethod(t *testing.T) {
	tests := []struct {
		name     string
		caps     *platform.Capabilities
		expected CloudflaredVerifyCapability
	}{
		{
			name: "systemd available",
			caps: &platform.Capabilities{
				Platform:        platform.Linux,
				SupportsSystemd: true,
			},
			expected: CanVerifyViaSystemd,
		},
		{
			name: "linux without systemd",
			caps: &platform.Capabilities{
				Platform:        platform.Linux,
				SupportsSystemd: false,
			},
			expected: CannotVerifyRunning,
		},
		{
			name: "macos no systemd",
			caps: &platform.Capabilities{
				Platform:        platform.MacOS,
				SupportsLaunchd: true,
			},
			expected: CannotVerifyRunning, // Launchd not supported yet
		},
		{
			name: "windows",
			caps: &platform.Capabilities{
				Platform:           platform.Windows,
				SupportsWindowsSvc: true,
			},
			expected: CannotVerifyRunning, // Windows service check not implemented yet
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SelectCloudflaredVerifyMethod(tt.caps)
			if result != tt.expected {
				t.Errorf("SelectCloudflaredVerifyMethod() = %v, want %v",
					result, tt.expected)
			}
		})
	}
}

// TestCloudflaredInstallStateConstants verifies the enum values are distinct
func TestCloudflaredInstallStateConstants(t *testing.T) {
	if CloudflaredNotInstalled == CloudflaredInstalled {
		t.Error("CloudflaredNotInstalled should not equal CloudflaredInstalled")
	}
}

// TestCloudflaredVerifyCapabilityConstants verifies the enum values are distinct
func TestCloudflaredVerifyCapabilityConstants(t *testing.T) {
	if CannotVerifyRunning == CanVerifyViaSystemd {
		t.Error("CannotVerifyRunning should not equal CanVerifyViaSystemd")
	}
}
