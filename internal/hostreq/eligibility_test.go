package hostreq

import (
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestEvaluateEligibility(t *testing.T) {
	base := ResolvedRequirement{Name: "fixture", Kind: KindTool, Required: true}
	for _, tc := range []struct {
		name     string
		bundling hostreqspec.Bundling
		required bool
		present  bool
		want     EligibilityVerdict
	}{
		{"prohibited", hostreqspec.BundlingProhibited, true, true, EligibilityIneligible},
		{"required host absent", hostreqspec.BundlingHostRequired, true, false, EligibilityIneligible},
		{"optional host absent", hostreqspec.BundlingHostRequired, false, false, EligibilityDegraded},
		{"host present", hostreqspec.BundlingHostRequired, true, true, EligibilityEligible},
		{"vendor artifact", hostreqspec.BundlingVendorable, true, true, EligibilityEligible},
		{"vendor artifact missing", hostreqspec.BundlingVendorable, true, false, EligibilityIneligible},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requirement := base
			requirement.Required = tc.required
			requirement.Bundling = tc.bundling
			result := EvaluateEligibility(requirement, TierDesktop, "windows", tc.present)
			if result.Verdict != tc.want || !strings.Contains(result.Reason, "fixture") {
				t.Fatalf("EvaluateEligibility() = %+v, want %q and named requirement", result, tc.want)
			}
		})
	}
}
