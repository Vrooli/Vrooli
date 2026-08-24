package watchdog

import (
	"context"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

func TestCompatibilityInstallDoesNotMutateHost(t *testing.T) {
	d := NewDetector(&platform.Capabilities{Platform: "linux", SupportsSystemd: true})
	result := d.Install(context.Background(), InstallOptions{EnableLingering: true})
	if result.Success {
		t.Fatal("compatibility install must not report a scenario-owned mutation")
	}
	if !strings.Contains(result.Error, "vrooli setup") {
		t.Fatalf("expected setup guidance, got %q", result.Error)
	}
}

func TestCompatibilityUninstallDoesNotMutateHost(t *testing.T) {
	d := NewDetector(&platform.Capabilities{Platform: "linux", SupportsSystemd: true})
	result := d.Uninstall(context.Background())
	if result.Success {
		t.Fatal("compatibility uninstall must not report a scenario-owned mutation")
	}
	if !strings.Contains(result.Error, "unsupported") {
		t.Fatalf("expected unsupported guidance, got %q", result.Error)
	}
}

func TestCompatibilityLingeringDoesNotMutateHost(t *testing.T) {
	d := NewDetector(&platform.Capabilities{Platform: "linux", SupportsSystemd: true})
	result := d.EnableLingering(context.Background())
	if result.Success {
		t.Fatal("compatibility lingering must not report a scenario-owned mutation")
	}
	if !strings.Contains(result.Error, "vrooli setup") {
		t.Fatalf("expected setup guidance, got %q", result.Error)
	}
}

func TestInstallStatusRecommendedSetup(t *testing.T) {
	tests := map[string]string{"linux": "user", "macos": "user", "windows": "system", "other": ""}
	for platformName, expected := range tests {
		d := NewDetector(&platform.Capabilities{Platform: platform.Type(platformName)})
		if got := d.GetInstallStatus().RecommendedSetup; got != expected {
			t.Errorf("platform %q: recommended setup %q, want %q", platformName, got, expected)
		}
	}
}
