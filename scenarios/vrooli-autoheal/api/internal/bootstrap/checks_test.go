// Package bootstrap tests
// [REQ:HEALTH-REGISTRY-001]
package bootstrap

import (
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
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
		// Infrastructure checks
		"infra-network",
		"infra-dns",
		"infra-docker",
		"infra-cloudflared",
		"infra-rdp",
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

func TestConfiguredInfrastructureTargets(t *testing.T) {
	t.Setenv(NetworkTargetEnv, "resolver.test:53")
	t.Setenv(DNSDomainEnv, "resolver.test")

	networkTarget := configuredInfrastructureValue(NetworkTargetEnv)
	dnsDomain := configuredInfrastructureValue(DNSDomainEnv)
	if networkTarget == "" || dnsDomain == "" {
		t.Fatal("configured infrastructure targets must be non-empty")
	}
	if !containsColon(networkTarget) {
		t.Errorf("network target %q should be in host:port format", networkTarget)
	}
}

func containsColon(s string) bool {
	for _, c := range s {
		if c == ':' {
			return true
		}
	}
	return false
}
