package emergencywatchdog

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestConformance(t *testing.T) {
	hostreqkittest.RunSuite(t, hostreqkittest.Case{
		NewHandler:         func() hostreqkit.Handler { return NewHandler(testManifest()) },
		Name:               "emergency_watchdog",
		Kind:               hostreqspec.KindSafeguard,
		SupportedPlatforms: []string{"linux"},
		ManifestDefaults: &hostreqkittest.ManifestDefaultsCase{
			Load: func() (map[string]hostreqkittest.ManifestProperty, error) {
				return hostreqkittest.LoadManifestProperties("safeguard.json")
			},
			Required:  []string{"setpoint_path"},
			Forbidden: []string{"disk_floor_mb", "unit_threshold_seconds", "cpu_pressure_avg10"},
		},
		Checks: []string{"name_and_kind", "inspect_unsupported_platform", "defaults_match_manifest"},
	})
}
