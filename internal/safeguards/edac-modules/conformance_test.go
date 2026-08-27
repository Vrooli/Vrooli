package edacmodules

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestConformance(t *testing.T) {
	hostreqkittest.RunSuite(t, hostreqkittest.Case{NewHandler: func() hostreqkit.Handler { return newHandler() }, Name: "edac_modules", Kind: hostreqspec.KindSafeguard, SupportedPlatforms: []string{"linux"}, Checks: []string{"name_and_kind", "inspect_manual_requirement", "inspect_unsupported_platform"}})
}
