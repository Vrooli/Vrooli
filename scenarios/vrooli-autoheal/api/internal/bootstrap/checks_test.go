// Package bootstrap tests
// [REQ:HEALTH-REGISTRY-001]
package bootstrap

import (
	"testing"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

func TestRegisterDefaultChecks(t *testing.T) {
	registry := checks.NewRegistry(&platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
		HasDocker:       true,
	})

	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
		HasDocker:       true,
	}

	RegisterDefaultChecks(registry, caps)

	// Verify checks were registered
	checksList := registry.ListChecks()
	if len(checksList) == 0 {
		t.Fatal("RegisterDefaultChecks() did not register any checks")
	}

	// Verify expected check IDs are registered
	expectedIDs := []string{
		"infra-network",
		"infra-dns",
		"infra-docker",
		"infra-cloudflared",
		"infra-rdp",
		"resource-postgres",
		"resource-redis",
	}

	for _, expectedID := range expectedIDs {
		found := false
		for _, info := range checksList {
			if info.ID == expectedID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected check %q to be registered", expectedID)
		}
	}

	t.Logf("Registered %d checks", len(checksList))
}

func TestRegisterDefaultChecks_DifferentPlatforms(t *testing.T) {
	tests := []struct {
		name     string
		platform platform.Type
	}{
		{"Linux", platform.Linux},
		{"Windows", platform.Windows},
		{"MacOS", platform.MacOS},
		{"Other", platform.Other},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			caps := &platform.Capabilities{
				Platform: tc.platform,
			}
			registry := checks.NewRegistry(caps)

			// Should not panic
			RegisterDefaultChecks(registry, caps)

			checksList := registry.ListChecks()
			if len(checksList) < 5 {
				t.Errorf("Expected at least 5 checks registered, got %d", len(checksList))
			}
		})
	}
}

func TestDefaultConstants(t *testing.T) {
	// Verify default values are sensible
	if DefaultNetworkTarget == "" {
		t.Error("DefaultNetworkTarget should not be empty")
	}

	if DefaultDNSDomain == "" {
		t.Error("DefaultDNSDomain should not be empty")
	}

	// Verify network target is a valid host:port format
	if !containsColon(DefaultNetworkTarget) {
		t.Errorf("DefaultNetworkTarget %q should be in host:port format", DefaultNetworkTarget)
	}

	t.Logf("DefaultNetworkTarget: %s", DefaultNetworkTarget)
	t.Logf("DefaultDNSDomain: %s", DefaultDNSDomain)
}

func containsColon(s string) bool {
	for _, c := range s {
		if c == ':' {
			return true
		}
	}
	return false
}
