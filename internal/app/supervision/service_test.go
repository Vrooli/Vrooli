package supervision

import (
	"testing"

	"github.com/vrooli/api-core/coreset"
)

func TestComputeAlwaysRetainsOperationalRecoveryPlane(t *testing.T) {
	report := Compute("", coreset.Authority{Seed: []string{"writer"}, TrustedBase: []string{"writer"}})

	for _, name := range coreset.RequiredOperationalScenarios {
		if !contains(report.Seed, name) || !contains(report.CoreSet, name) || !contains(report.TrustedBase, name) {
			t.Errorf("required operational scenario %q missing from report: %+v", name, report)
		}
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
