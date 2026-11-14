package crashkernelreserve

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestConformance(t *testing.T) {
	hostreqkittest.RunSuite(t, hostreqkittest.Case{
		NewHandler:         func() hostreqkit.Handler { return newHandler() },
		Name:               "crashkernel_reserve",
		Kind:               hostreqspec.KindSafeguard,
		SupportedPlatforms: []string{"linux"},
		Checks:             []string{"name_and_kind", "inspect_manual_requirement", "inspect_unsupported_platform", "apply_already_applied_skips", "apply_unsupported_returns_early", "apply_manual_returns_early", "apply_not_applicable_returns_early"},
		PackageChecks: []hostreqkittest.PackageCheck{{
			Name: "apply_updates_grub_config_without_running_update_grub",
			Run:  checkApplyUpdatesGrubConfigWithoutRunningUpdateGrub,
		}},
	})
}
