package administration

import "testing"

func TestRemoteProfileProductionUsesControlPlaneEnvironmentFirst(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	t.Setenv("VROOLI_ENVIRONMENT", "production")
	if !isRemoteProfileProductionEnvironment() {
		t.Fatal("LPBS local development default must not mask production remote-profile policy")
	}
	t.Setenv("VROOLI_ENVIRONMENT", "")
	if isRemoteProfileProductionEnvironment() {
		t.Fatal("scenario environment fallback should be development")
	}
}

func TestValidateRemoteProfileEnvironment(t *testing.T) {
	for _, testCase := range []struct{ raw, want string }{
		{"production", "production"},
		{" PRODUCTION ", "production"},
		{"development", "development"},
		{"", "development"},
		{"unexpected", "development"},
	} {
		if got := validateRemoteProfileEnvironment(testCase.raw); got != testCase.want {
			t.Errorf("validateRemoteProfileEnvironment(%q) = %q, want %q", testCase.raw, got, testCase.want)
		}
	}
}
