package pstorenative

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestConformance(t *testing.T) {
	hostreqkittest.RunSuite(t, hostreqkittest.Case{NewHandler: func() hostreqkit.Handler { return newHandler() }, Name: "pstore_native", Kind: hostreqspec.KindSafeguard, SupportedPlatforms: []string{"linux"}, Checks: []string{"name_and_kind", "inspect_unsupported_platform", "inspect_manual_requirement", "apply_already_applied_skips", "apply_dry_run"}})
}
