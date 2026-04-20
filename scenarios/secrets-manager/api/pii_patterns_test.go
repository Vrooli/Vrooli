package main

import (
	"regexp"
	"strings"
	"testing"
)

func scanForPattern(t *testing.T, typ, input string) bool {
	t.Helper()
	for _, p := range piiVulnerabilityPatterns {
		if p.Type != typ {
			continue
		}
		re, err := regexp.Compile(p.Pattern)
		if err != nil {
			t.Fatalf("pattern %s failed to compile: %v", typ, err)
		}
		return re.FindStringIndex(input) != nil
	}
	t.Fatalf("pattern %s not found", typ)
	return false
}

func TestPIIPatterns_TruePositives(t *testing.T) {
	cases := []struct {
		typ   string
		input string
	}{
		{"pii_email", "const notifyAdmin = \"alice.smith@example.com\""},
		{"pii_phone_us", "contact: (415) 555-0100"},
		{"pii_phone_us", "number: 415-555-0123"},
		{"pii_ssn", "ssn = \"123-45-6789\""},
		{"pii_credit_card", "card := \"4111 1111 1111 1111\""},
		{"pii_ip_address", "host := \"192.168.1.42\""},
		{"pii_aws_key", "access := \"AKIAIOSFODNN7EXAMPLE\""},
		{"pii_home_dir", "cfg := \"/home/matthalloran8/data\""},
	}
	for _, tc := range cases {
		if !scanForPattern(t, tc.typ, tc.input) {
			t.Errorf("expected %s to match %q", tc.typ, tc.input)
		}
	}
}

func TestPIIPatterns_TrueNegatives(t *testing.T) {
	cases := []struct {
		typ   string
		input string
	}{
		// IPv4 pattern must not match a version like 1.2 (no 4 octets).
		{"pii_ip_address", "version: 1.2"},
		// AWS key pattern must not match a too-short token.
		{"pii_aws_key", "AKIASHORT"},
		// SSN pattern is distinct from a phone suffix.
		{"pii_ssn", "ext 4567"},
	}
	for _, tc := range cases {
		if scanForPattern(t, tc.typ, tc.input) {
			t.Errorf("expected %s NOT to match %q", tc.typ, tc.input)
		}
	}
}

func TestContextAwareFilter_ImportBlock(t *testing.T) {
	lines := []string{
		"package main",
		"",
		"import (",
		"\t\"github.com/example/alice@v1.2.3.4/bar\"",
		"\t\"example.com/alice@me\"",
		")",
		"",
		"var x = \"alice@example.com\"",
	}
	ctx := newFileScanContext("foo.go", lines)
	if !contextAwareFilter(ctx, 4, "pii_email") {
		t.Errorf("expected import-block line to be suppressed")
	}
	if !contextAwareFilter(ctx, 4, "pii_ip_address") {
		t.Errorf("expected import-block line to suppress IP matches")
	}
	if contextAwareFilter(ctx, 8, "pii_email") {
		t.Errorf("expected non-import line to NOT be suppressed")
	}
}

func TestContextAwareFilter_BuildTags(t *testing.T) {
	lines := []string{
		"//go:build linux && amd64",
		"// +build 1.2.3.4",
		"package main",
		"",
		"const ip = \"1.2.3.4\"",
	}
	ctx := newFileScanContext("tags.go", lines)
	if !contextAwareFilter(ctx, 1, "pii_ip_address") {
		t.Errorf("expected //go:build line to be suppressed")
	}
	if !contextAwareFilter(ctx, 2, "pii_ip_address") {
		t.Errorf("expected // +build line to be suppressed")
	}
	if contextAwareFilter(ctx, 5, "pii_ip_address") {
		t.Errorf("expected plain const line to not be suppressed")
	}
}

func TestContextAwareFilter_Lockfiles(t *testing.T) {
	lines := []string{"some-package 1.2.3:", "\tresolved \"https://registry/pkg/1.2.3.4\""}
	ctx := newFileScanContext("yarn.lock", lines)
	if !contextAwareFilter(ctx, 2, "pii_ip_address") {
		t.Errorf("expected yarn.lock line to be suppressed")
	}

	ctx = newFileScanContext("go.sum", []string{"github.com/foo v1.0.0 h1:abc"})
	if !contextAwareFilter(ctx, 1, "pii_email") {
		t.Errorf("expected go.sum to suppress all pii types")
	}
}

func TestContextAwareFilter_URLInComment(t *testing.T) {
	lines := []string{"// see https://docs.example.com/admin/1.2.3.4 for details"}
	ctx := newFileScanContext("notes.go", lines)
	if !contextAwareFilter(ctx, 1, "pii_ip_address") {
		t.Errorf("expected commented URL context to suppress IP")
	}
}

func TestContextAwareFilter_VersionPragmaLine(t *testing.T) {
	lines := []string{"Version: 1.2.3.4"}
	ctx := newFileScanContext("manifest.yaml", lines)
	if !contextAwareFilter(ctx, 1, "pii_ip_address") {
		t.Errorf("expected version pragma line to suppress IP")
	}
}

func TestPIITypeSet(t *testing.T) {
	if !isPIIType("pii_email") {
		t.Errorf("pii_email should be recognized as PII")
	}
	if isPIIType("sql_injection") {
		t.Errorf("sql_injection should not be PII")
	}
	// Smoke-check all defined types are in the set.
	for _, p := range piiVulnerabilityPatterns {
		if !strings.HasPrefix(p.Type, "pii_") {
			t.Errorf("PII pattern %s is missing pii_ prefix", p.Type)
		}
		if !isPIIType(p.Type) {
			t.Errorf("isPIIType missed %s", p.Type)
		}
	}
}
