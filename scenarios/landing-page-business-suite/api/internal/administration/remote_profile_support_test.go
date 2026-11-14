package administration

import "testing"

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
