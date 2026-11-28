package generatedtreeownership

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
)

func TestConformance(t *testing.T) {
	hostreqkittest.RunSafeguardSuite(t, "generated_tree_ownership", func() hostreqkit.Handler {
		return NewHandler(hostreqkit.SafeguardManifest{Name: "generated_tree_ownership", Handler: "generated_tree_ownership"})
	}, []string{"linux", "darwin"},
		"name_and_kind",
		"inspect_manual_requirement",
		"apply_unsupported_returns_early",
		"apply_not_applicable_returns_early",
		"apply_manual_returns_early",
		"apply_already_applied_skips",
	)
}
