package tcptuning

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
)

func TestConformance(t *testing.T) {
	hostreqkittest.RunSafeguardSuite(t, "tcp_tuning", newTestHandler, []string{"linux"},
		"name_and_kind",
		"inspect_manual_requirement",
		"inspect_unsupported_platform",
		"apply_unsupported_returns_early",
		"apply_manual_returns_early",
		"apply_already_applied_skips",
		"apply_dry_run",
	)
}
