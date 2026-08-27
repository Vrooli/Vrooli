package kdumpobservability

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestConformance(t *testing.T) {
	hostreqkittest.RunSuite(t, hostreqkittest.Case{NewHandler: func() hostreqkit.Handler { return NewHandler(testManifest()) }, Name: "kdump_observability", Kind: hostreqspec.KindSafeguard, SupportedPlatforms: []string{"linux"}, Checks: []string{"name_and_kind", "inspect_unsupported_platform"}})
}
