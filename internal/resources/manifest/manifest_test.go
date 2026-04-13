package manifest

import (
	"strings"
	"testing"
)

func TestValidateRejectsMissingRequiredFields(t *testing.T) {
	err := Validate(ResourceManifest{})
	if err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateRejectsInvalidLegacyAdapter(t *testing.T) {
	err := Validate(ResourceManifest{
		Name:            "redis",
		Driver:          "legacy-adapter",
		PortabilityTier: "partial",
		LegacyAdapter: ResourceLegacyAdapter{
			Owner:            "fixture",
			DecisionDeadline: "2026-12-31",
			FinalDisposition: "invalid",
			LegacyCLIPath:    "resources/redis/cli.sh",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "final_disposition") {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateAcceptsExternalCLIManifest(t *testing.T) {
	err := Validate(ResourceManifest{
		Name:            "redis",
		Driver:          "external-cli",
		Binary:          "redis-server",
		PortabilityTier: "full",
		Platforms:       ResourcePlatforms{Linux: "supported", MacOS: "partial", Windows: "unsupported"},
	})
	if err != nil {
		t.Fatalf("Validate(): %v", err)
	}
}

func TestSupportForCurrentPlatformUsesMappedOSNames(t *testing.T) {
	value := ResourcePlatforms{Linux: "supported", MacOS: "partial", Windows: "unsupported"}.SupportForCurrentPlatform()
	if value == "" {
		t.Fatal("expected support state for current platform")
	}
}
