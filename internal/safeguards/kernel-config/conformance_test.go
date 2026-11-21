package kernelconfig

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
)

func TestConformance(t *testing.T) {
	hostreqkittest.RunSafeguardSuite(t, "kernel_config", newTestHandler, []string{"linux"},
		"name_and_kind",
		"inspect_manual_requirement",
		"inspect_unsupported_platform",
		"inspect_no_sysctl_not_applicable",
		"apply_unsupported_returns_early",
		"apply_not_applicable_returns_early",
		"apply_manual_returns_early",
		"apply_already_applied_skips",
		"apply_dry_run",
	)
}
