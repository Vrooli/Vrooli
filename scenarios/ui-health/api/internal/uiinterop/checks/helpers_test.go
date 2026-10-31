package checks

import (
	"testing"

	"ui-health/internal/uiinterop"
)

func TestIsTestFile(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		// Production files.
		{"api-internals.ts", false},
		{"connect.ts", false},
		{"App.tsx", false},
		{"useApi.ts", false},
		// Conventional test files.
		{"api.test.ts", true},
		{"Button.spec.tsx", true},
		{"handlers_test.go", true},
		{"foo_spec.ts", true},
		// Test-harness setup files (the resolveApiBase false-positive class).
		{"test-setup.ts", true},
		{"setup-tests.ts", true},
		{"setupTests.ts", true},
		{"vitest.setup.ts", true},
		{"jest.setup.ts", true},
		{"test-utils.ts", true},
	}
	for _, tc := range cases {
		if got := uiinterop.IsTestFile(tc.name); got != tc.want {
			t.Errorf("isTestFile(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}
