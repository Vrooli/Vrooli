// Package infra provides infrastructure health checks
// Tests for decision boundary functions
// [REQ:INFRA-RDP-001] [REQ:INFRA-CLOUDFLARED-001]
package infra

import (
	"testing"

	"vrooli-autoheal/internal/platform"
)

// TestSelectRDPService tests the RDP service selection decision function
func TestSelectRDPService(t *testing.T) {
	tests := []struct {
		name           string
		caps           *platform.Capabilities
		expectedName   string
		expectedCheck  bool
	}{
		{
			name: "linux with systemd",
			caps: &platform.Capabilities{
				Platform:        platform.Linux,
				SupportsSystemd: true,
			},
			expectedName:  "xrdp",
			expectedCheck: true,
		},
		{
			name: "linux without systemd",
			caps: &platform.Capabilities{
				Platform:        platform.Linux,
				SupportsSystemd: false,
			},
			expectedName:  "xrdp",
			expectedCheck: false, // Can't check without service manager
		},
		{
			name: "windows",
			caps: &platform.Capabilities{
				Platform: platform.Windows,
			},
			expectedName:  "TermService",
			expectedCheck: true,
		},
		{
			name: "macos",
			caps: &platform.Capabilities{
				Platform: platform.MacOS,
			},
			expectedName:  "",
			expectedCheck: false,
		},
		{
			name: "other platform",
			caps: &platform.Capabilities{
				Platform: platform.Other,
			},
			expectedName:  "",
			expectedCheck: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := SelectRDPService(tt.caps)
			if info.ServiceName != tt.expectedName {
				t.Errorf("SelectRDPService() ServiceName = %q, want %q",
					info.ServiceName, tt.expectedName)
			}
			if info.Checkable != tt.expectedCheck {
				t.Errorf("SelectRDPService() Checkable = %v, want %v",
					info.Checkable, tt.expectedCheck)
			}
		})
	}
}

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
				Platform:       platform.MacOS,
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
