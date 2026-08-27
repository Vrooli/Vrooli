package autohealwatchdog

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestConformance(t *testing.T) {
	hostreqkittest.RunSuite(t, hostreqkittest.Case{NewHandler: func() hostreqkit.Handler { return NewHandler(hostreqkit.SafeguardManifest{Name: "autoheal_watchdog"}) }, Name: "autoheal_watchdog", Kind: hostreqspec.KindSafeguard, SupportedPlatforms: []string{"linux", "macos", "windows"}, Checks: []string{"name_and_kind"}})
}
