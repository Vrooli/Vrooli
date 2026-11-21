package clock

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestConformance(t *testing.T) {
	hostreqkittest.RunSuite(t, hostreqkittest.Case{NewHandler: func() hostreqkit.Handler { return newTestHandler() }, Name: "clock", Kind: hostreqspec.KindSafeguard, Checks: []string{"name_and_kind", "inspect_manual_requirement", "apply_unsupported_returns_early", "apply_already_applied_skips", "apply_dry_run"}})
}
