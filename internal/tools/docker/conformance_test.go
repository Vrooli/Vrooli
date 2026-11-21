package docker

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestConformance(t *testing.T) {
	hostreqkittest.RunSuite(t, hostreqkittest.Case{
		NewHandler: func() hostreqkit.Handler { return NewHandler(testManifest()) },
		Name:       testManifest().Name,
		Kind:       hostreqspec.KindTool,
		Checks: []string{
			"name_and_kind",
			"inspect_manual_requirement",
		},
	})
}
