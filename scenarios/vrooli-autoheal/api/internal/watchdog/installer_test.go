package watchdog

import (
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

func TestInstallStatusRecommendedSetup(t *testing.T) {
	tests := map[string]string{"linux": "user", "macos": "user", "windows": "system", "other": ""}
	for platformName, expected := range tests {
		d := NewDetector(&platform.Capabilities{Platform: platform.Type(platformName)})
		if got := d.GetInstallStatus().RecommendedSetup; got != expected {
			t.Errorf("platform %q: recommended setup %q, want %q", platformName, got, expected)
		}
	}
}
