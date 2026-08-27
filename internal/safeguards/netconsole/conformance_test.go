package netconsole

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestConformance(t *testing.T) {
	hostreqkittest.RunSuite(t, hostreqkittest.Case{NewHandler: func() hostreqkit.Handler {
		return NewHandler(hostreqkit.SafeguardManifest{Name: "netconsole", Handler: "netconsole"})
	}, Name: "netconsole", Kind: hostreqspec.KindSafeguard, SupportedPlatforms: []string{"linux"}, Checks: []string{"name_and_kind", "inspect_manual_requirement", "inspect_unsupported_platform", "apply_unsupported_returns_early", "apply_already_applied_skips"}, PackageChecks: []hostreqkittest.PackageCheck{{Name: "apply_installs_and_loads_netconsole", Run: checkApplyInstallsAndLoadsNetconsole}}})
}
