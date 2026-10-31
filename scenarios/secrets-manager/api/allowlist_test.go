package main

import "testing"

func TestShouldExcludeFile_GlobAndType(t *testing.T) {
	rules := []AllowlistRule{
		{
			PathPattern:   "*_test.go",
			ExcludedTypes: []string{"pii_email", "pii_phone_us"},
			Enabled:       true,
		},
		{
			PathPattern:   "testdata/**",
			ExcludedTypes: []string{"*"},
			Enabled:       true,
		},
		{
			PathPattern:   "go.sum",
			ExcludedTypes: []string{"*"},
			Enabled:       true,
		},
		{
			PathPattern:   "disabled.txt",
			ExcludedTypes: []string{"*"},
			Enabled:       false,
		},
	}

	cases := []struct {
		name string
		path string
		typ  string
		want bool
	}{
		{"test file email excluded", "scanner_test.go", "pii_email", true},
		{"test file ssn not excluded", "scanner_test.go", "pii_ssn", false},
		{"testdata excludes all", "testdata/sample.json", "pii_ssn", true},
		{"go.sum excludes all", "go.sum", "pii_email", true},
		{"non-matching path", "handlers.go", "pii_email", false},
		{"disabled rule does not exclude", "disabled.txt", "pii_ssn", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldExcludeFile(rules, tc.path, tc.typ)
			if got != tc.want {
				t.Errorf("shouldExcludeFile(%q, %q) = %v, want %v", tc.path, tc.typ, got, tc.want)
			}
		})
	}
}

func TestShouldExcludeFile_AbsolutePathWithGlob(t *testing.T) {
	rules := []AllowlistRule{
		{PathPattern: "*_test.go", ExcludedTypes: []string{"*"}, Enabled: true},
	}
	if !shouldExcludeFile(rules, "/repo/pkg/foo_test.go", "pii_email") {
		t.Errorf("basename glob should match absolute paths")
	}
}

func TestValidateAllowlistRule(t *testing.T) {
	cases := []struct {
		name    string
		req     allowlistUpsertRequest
		wantErr bool
	}{
		{"valid", allowlistUpsertRequest{PathPattern: "*.go", ExcludedTypes: []string{"pii_email"}}, false},
		{"empty pattern", allowlistUpsertRequest{PathPattern: " ", ExcludedTypes: []string{"*"}}, true},
		{"too broad single star", allowlistUpsertRequest{PathPattern: "*", ExcludedTypes: []string{"*"}}, true},
		{"too broad double star", allowlistUpsertRequest{PathPattern: "**", ExcludedTypes: []string{"*"}}, true},
		{"missing types", allowlistUpsertRequest{PathPattern: "*.go"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAllowlistRule(tc.req)
			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestMatchesAnyAllowlistAllType(t *testing.T) {
	rules := []AllowlistRule{
		{PathPattern: "testdata/**", ExcludedTypes: []string{"*"}, Enabled: true},
		{PathPattern: "*_test.go", ExcludedTypes: []string{"pii_email"}, Enabled: true},
	}
	if !matchesAnyAllowlistAllType(rules, "testdata/fixture.json") {
		t.Errorf("testdata wildcard should produce a full skip")
	}
	if matchesAnyAllowlistAllType(rules, "handler_test.go") {
		t.Errorf("type-specific rule should not trigger full-file skip")
	}
}
