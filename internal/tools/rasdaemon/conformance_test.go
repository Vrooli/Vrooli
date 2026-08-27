package rasdaemon

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestConformance(t *testing.T) {
	hostreqkittest.RunSuite(t, hostreqkittest.Case{
		NewHandler:         newHandler,
		Name:               "rasdaemon",
		Kind:               hostreqspec.KindTool,
		SupportedPlatforms: []string{"linux"},
		InstallCommand:     "apt-get install rasdaemon",
		Checks:             []string{"name_and_kind"},
	})
}
