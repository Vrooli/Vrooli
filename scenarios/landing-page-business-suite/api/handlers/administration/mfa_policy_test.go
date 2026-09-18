package administration

import "testing"

// The live hole this closes: cloud execution exports only
// VROOLI_ENVIRONMENT, so a deployed production process read an empty
// LPBS_ENVIRONMENT, derived "MFA not required", and accepted administrators
// with no second factor.
func TestAdminMFARequiredDerivesFromEitherEnvironmentVariable(t *testing.T) {
	for name, testCase := range map[string]struct {
		lpbs, vrooli string
		want         bool
	}{
		"control plane production only": {"", "production", true},
		"scenario production only":      {"production", "", true},
		"prod shorthand":                {"", "prod", true},
		"development":                   {"development", "", false},
		"nothing declared":              {"", "", false},
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("ADMIN_REQUIRE_MFA", "")
			t.Setenv("LPBS_ENVIRONMENT", testCase.lpbs)
			t.Setenv("VROOLI_ENVIRONMENT", testCase.vrooli)

			if got := AdminMFARequired(); got != testCase.want {
				t.Errorf("AdminMFARequired() = %t, want %t", got, testCase.want)
			}
		})
	}
}

// An explicit setting still wins in both directions, so an operator can turn
// the requirement on before a deployment is production, or off deliberately.
func TestAdminMFARequiredHonoursAnExplicitSetting(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	t.Setenv("VROOLI_ENVIRONMENT", "")
	for _, value := range []string{"true", "1", "yes"} {
		t.Setenv("ADMIN_REQUIRE_MFA", value)
		if !AdminMFARequired() {
			t.Errorf("ADMIN_REQUIRE_MFA=%q must require a second factor", value)
		}
	}

	t.Setenv("LPBS_ENVIRONMENT", "production")
	for _, value := range []string{"false", "0", "no"} {
		t.Setenv("ADMIN_REQUIRE_MFA", value)
		if AdminMFARequired() {
			t.Errorf("ADMIN_REQUIRE_MFA=%q must not require a second factor", value)
		}
	}
}
