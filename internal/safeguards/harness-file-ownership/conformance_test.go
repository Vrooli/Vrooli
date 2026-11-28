package harnessfileownership

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
)

func TestConformance(t *testing.T) {
	hostreqkittest.RunSafeguardSuite(t, "harness_file_ownership", func() hostreqkit.Handler {
		return NewHandler(hostreqkit.SafeguardManifest{Name: "harness_file_ownership", Handler: "harness_file_ownership"})
	}, []string{"linux", "darwin"},
		"name_and_kind",
		"inspect_manual_requirement",
		"apply_unsupported_returns_early",
		"apply_not_applicable_returns_early",
		"apply_manual_returns_early",
		"apply_already_applied_skips",
	)
}
