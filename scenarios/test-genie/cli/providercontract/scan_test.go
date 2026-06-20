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
	if args.Timeout != 30*time.Second {
		t.Fatalf("default timeout = %v, want 30s", args.Timeout)
	}
	if args.JSON {
		t.Fatal("JSON should default to false")
	}
}

func TestParseScanArgsOverrides(t *testing.T) {
	args, err := ParseScanArgs([]string{"scan", "--json", "--target", "proto-health", "--timeout", "5s"})
	if err != nil {
		t.Fatalf("ParseScanArgs: %v", err)
	}
	if !args.JSON || args.Target != "proto-health" || args.Timeout != 5*time.Second {
		t.Fatalf("unexpected parsed args: %+v", args)
	}
}

func TestParseScanArgsRejectsNonScan(t *testing.T) {
	if _, err := ParseScanArgs([]string{"check"}); err == nil {
		t.Fatal("expected error for non-scan subcommand")
	}
}

func TestHasHardViolationMapping(t *testing.T) {
	// spec-invalid is always hard.
	if !hasHardViolation(ProviderReport{SpecValid: false}) {
		t.Fatal("spec-invalid must be a hard violation")
	}
	// reachable + contract-invalid is hard.
	if !hasHardViolation(ProviderReport{SpecValid: true, Reachable: true, ContractValid: false}) {
		t.Fatal("reachable contract-invalid must be hard")
	}
	// unreachable with valid spec is not hard.
	if hasHardViolation(ProviderReport{SpecValid: true, Reachable: false}) {
		t.Fatal("unreachable with valid spec must not be hard")
	}
}
