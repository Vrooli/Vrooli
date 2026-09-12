package runtimesupervisorsafeguard

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestConformance(t *testing.T) {
	hostreqkittest.RunSuite(t, hostreqkittest.Case{
		NewHandler:         func() hostreqkit.Handler { return NewHandler(hostreqkit.SafeguardManifest{Name: manifestName}) },
		Name:               manifestName,
		Kind:               hostreqspec.KindSafeguard,
		SupportedPlatforms: []string{"linux", "macos"},
		Seams: func(t *testing.T) {
			seams(t, t.TempDir(), &fakeHost{show: activeShow()})
		},
		Checks: []string{"name_and_kind", "inspect_manual_requirement", "apply_unsupported_returns_early", "apply_manual_returns_early", "apply_already_applied_skips", "apply_dry_run", "inspect_reports_validator_verdict", "apply_reverifies"},
	})
}
