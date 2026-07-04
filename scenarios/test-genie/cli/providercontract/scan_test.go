package providercontract

import (
	"testing"
	"time"

	"test-genie/internal/selfhealth"
)

// The conformance-scan core (scanProvider, probing, scoring, hard-violation
// rules) now lives in test-genie/internal/selfhealth and is tested there
// (conformance_test.go). These tests cover the thin CLI wrapper: arg parsing and
// the report-shape mapping that preserves the stable snake_case JSON contract.

func TestParseScanArgsDefaults(t *testing.T) {
	args, err := ParseScanArgs([]string{"scan"})
	if err != nil {
		t.Fatalf("ParseScanArgs: %v", err)
	}
	if args.Target != selfhealth.DefaultScanTarget {
		t.Fatalf("default target = %q, want %q", args.Target, selfhealth.DefaultScanTarget)
	}
	if args.Timeout != time.Minute {
		t.Fatalf("default timeout = %v, want 1m", args.Timeout)
	}
	if args.JSON {
		t.Fatal("JSON should default to false")
	}
}

func TestParseScanArgsOverrides(t *testing.T) {
	args, err := ParseScanArgs([]string{"scan", "branding", "--json", "--target", "proto-health", "--timeout", "5s"})
	if err != nil {
		t.Fatalf("ParseScanArgs: %v", err)
	}
	if !args.JSON || args.Target != "proto-health" || args.Timeout != 5*time.Second || args.Subject != "branding" {
		t.Fatalf("unexpected parsed args: %+v", args)
	}
}

func TestParseScanArgsRejectsNonScan(t *testing.T) {
	if _, err := ParseScanArgs([]string{"check"}); err == nil {
		t.Fatal("expected error for non-scan subcommand")
	}
}

func TestParseScanArgsRejectsUnknownSubject(t *testing.T) {
	if _, err := ParseScanArgs([]string{"scan", "not-a-phase"}); err == nil {
		t.Fatal("expected error for unknown scan subject")
	}
}

func TestParseScanArgsRejectsExtraSubjects(t *testing.T) {
	if _, err := ParseScanArgs([]string{"scan", "branding", "contracts"}); err == nil {
		t.Fatal("expected error for extra scan subjects")
	}
}

func TestHardViolationDelegatesToSSOT(t *testing.T) {
	hard := func(r ProviderReport) bool {
		return selfhealth.IsHardViolation(r.SpecValid, r.Reachable, r.ContractValid, r.IdentityOK, r.MetricsAdopted)
	}
	// spec-invalid is always hard.
	if !hard(ProviderReport{SpecValid: false}) {
		t.Fatal("spec-invalid must be a hard violation")
	}
	// reachable + contract-invalid is hard.
	if !hard(ProviderReport{SpecValid: true, Reachable: true, ContractValid: false}) {
		t.Fatal("reachable contract-invalid must be hard")
	}
	// unreachable with valid spec is not hard (environmental, never gated on metrics).
	if hard(ProviderReport{SpecValid: true, Reachable: false}) {
		t.Fatal("unreachable with valid spec must not be hard")
	}
	// reachable + fully valid contract/identity but metrics dropped is now hard
	// (Plan 3 Part B flip — metrics_adopted is no longer advisory).
	if !hard(ProviderReport{SpecValid: true, Reachable: true, ContractValid: true, IdentityOK: true, MetricsAdopted: false}) {
		t.Fatal("reachable provider that dropped metrics must be a hard violation")
	}
	// reachable + fully adopted is clean.
	if hard(ProviderReport{SpecValid: true, Reachable: true, ContractValid: true, IdentityOK: true, MetricsAdopted: true}) {
		t.Fatal("full adoption must not be a hard violation")
	}
}
