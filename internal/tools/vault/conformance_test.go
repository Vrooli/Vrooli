package vault

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestConformance(t *testing.T) {
	hostreqkittest.RunSuite(t, hostreqkittest.Case{
		NewHandler:         newTestHandler,
		Name:               testManifest.Name,
		Kind:               hostreqspec.KindTool,
		SupportedPlatforms: testManifest.Platforms,
		InstallCommand:     "apt-get install vault",
		Checks: []string{
			"name_and_kind",
			"inspect_manual_requirement",
			"apply_manual_returns_early",
			"apply_unsupported_returns_early",
		},
	})
}
