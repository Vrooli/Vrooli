package kdumptools

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestConformance(t *testing.T) {
	hostreqkittest.RunSuite(t, hostreqkittest.Case{
		NewHandler:         newHandler,
		Name:               "kdump-tools",
		Kind:               hostreqspec.KindTool,
		SupportedPlatforms: []string{"linux"},
		InstallCommand:     "apt-get install kdump-tools",
		Checks:             []string{"name_and_kind"},
	})
}
