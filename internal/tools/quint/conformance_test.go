package quint

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestConformance(t *testing.T) {
	hostreqkittest.RunSuite(t, hostreqkittest.Case{
		NewHandler:         newHandler,
		Name:               testManifest.Name,
		Kind:               hostreqspec.KindTool,
		SupportedPlatforms: testManifest.Platforms,
		InstallCommand:     "npm install @informalsystems/quint",
		Checks:             []string{"name_and_kind"},
	})
}
