package agentsessioncontainment

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestConformance(t *testing.T) {
	hostreqkittest.RunSuite(t, hostreqkittest.Case{
		NewHandler:         newTestHandler,
		Name:               "agent_session_containment",
		Kind:               hostreqspec.KindSafeguard,
		SupportedPlatforms: []string{"linux"},
		Seams: func(t *testing.T) {
			f := newFixture(t)
			f.active = "inactive"
		},
		Checks: []string{
			"name_and_kind",
			"inspect_manual_requirement",
			"inspect_unsupported_platform",
			"apply_unsupported_returns_early",
			"apply_not_applicable_returns_early",
			"apply_manual_returns_early",
			"apply_already_applied_skips",
			"apply_dry_run",
			"inspect_reports_validator_verdict",
			"apply_reverifies",
		},
	})
}
