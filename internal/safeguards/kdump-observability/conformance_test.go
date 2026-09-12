package kdumpobservability

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestConformance(t *testing.T) {
	hostreqkittest.RunSuite(t, hostreqkittest.Case{
		NewHandler:         func() hostreqkit.Handler { return NewHandler(testManifest()) },
		Name:               "kdump_observability",
		Kind:               hostreqspec.KindSafeguard,
		SupportedPlatforms: []string{"linux"},
		ManifestDefaults: &hostreqkittest.ManifestDefaultsCase{
			Load: func() (map[string]hostreqkittest.ManifestProperty, error) {
				return hostreqkittest.LoadManifestProperties("safeguard.json")
			},
			Required: []string{"retain_vmcores"},
			Expected: map[string]any{"retain_vmcores": float64(defaultRetainVmcores)},
		},
		Checks: []string{"name_and_kind", "inspect_unsupported_platform", "defaults_match_manifest"},
	})
}
