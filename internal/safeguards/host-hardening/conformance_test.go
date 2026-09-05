package hosthardening

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestConformance(t *testing.T) {
	policy := resolvePolicy(nil)
	hostreqkittest.RunSuite(t, hostreqkittest.Case{
		NewHandler:         func() hostreqkit.Handler { return newTestHandler() },
		Name:               "host_hardening",
		Kind:               hostreqspec.KindSafeguard,
		SupportedPlatforms: []string{"linux"},
		ManifestDefaults: &hostreqkittest.ManifestDefaultsCase{
			Load: func() (map[string]hostreqkittest.ManifestProperty, error) {
				return hostreqkittest.LoadManifestProperties("safeguard.json")
			},
			Required: []string{"oops_policy", "softlockup_policy", "hung_task_timeout_secs"},
			Expected: map[string]any{
				"oops_policy":            policy.OopsPolicy,
				"softlockup_policy":      policy.SoftlockupPolicy,
				"hung_task_timeout_secs": float64(policy.HungTaskTimeout),
			},
		},
		Checks: []string{"name_and_kind", "inspect_manual_requirement", "inspect_unsupported_platform", "inspect_no_sysctl_not_applicable", "defaults_match_manifest"},
	})
}
