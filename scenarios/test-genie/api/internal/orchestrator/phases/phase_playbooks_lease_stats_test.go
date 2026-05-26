package phases

import (
	"errors"
	"strings"
	"testing"

	"test-genie/internal/playbooks/config"
)

func TestApplyLeaseStatsResult_GreenStaysGreen(t *testing.T) {
	report := &RunReport{}
	applyLeaseStatsResult(report, 5, 0, config.Default())

	if report.Err != nil {
		t.Fatalf("expected no error, got %v", report.Err)
	}
	if report.FailureClassification != "" {
		t.Fatalf("expected no failure classification, got %q", report.FailureClassification)
	}
	if len(report.Observations) != 0 {
		t.Fatalf("expected no observations, got %d", len(report.Observations))
	}
}

func TestApplyLeaseStatsResult_EmptyPoolHardFails(t *testing.T) {
	report := &RunReport{}
	applyLeaseStatsResult(report, 0, 0, config.Default())

	if report.Err == nil {
		t.Fatal("expected hard failure for empty test pool")
	}
	if !strings.Contains(report.Err.Error(), "0 test-mode requests") {
		t.Errorf("error message missing diagnostic: %v", report.Err)
	}
	if report.FailureClassification != FailureClassMisconfiguration {
		t.Errorf("FailureClassification = %q, want %q", report.FailureClassification, FailureClassMisconfiguration)
	}
	if !strings.Contains(report.Remediation, "allow_empty_test_pool") {
		t.Errorf("Remediation missing opt-out hint: %q", report.Remediation)
	}
}

func TestApplyLeaseStatsResult_BypassHardFails(t *testing.T) {
	report := &RunReport{}
	applyLeaseStatsResult(report, 7, 3, config.Default())

	if report.Err == nil {
		t.Fatal("expected hard failure for primary-bypass")
	}
	if !strings.Contains(report.Err.Error(), "routed bypass detected") {
		t.Errorf("error message missing diagnostic: %v", report.Err)
	}
	if report.FailureClassification != FailureClassMisconfiguration {
		t.Errorf("FailureClassification = %q, want %q", report.FailureClassification, FailureClassMisconfiguration)
	}
}

func TestApplyLeaseStatsResult_EmptyPoolOptOut(t *testing.T) {
	cfg := config.Default()
	cfg.AllowEmptyTestPool = true

	report := &RunReport{}
	applyLeaseStatsResult(report, 0, 0, cfg)

	if report.Err != nil {
		t.Fatalf("expected no error when opted out, got %v", report.Err)
	}
	if len(report.Observations) != 1 {
		t.Fatalf("expected 1 warning observation, got %d", len(report.Observations))
	}
	if !strings.Contains(report.Observations[0].Text, "0 test-mode requests") {
		t.Errorf("observation missing diagnostic: %+v", report.Observations[0])
	}
	if report.Observations[0].Prefix != "WARNING" {
		t.Errorf("observation Prefix = %q, want WARNING", report.Observations[0].Prefix)
	}
}

func TestApplyLeaseStatsResult_BypassOptOut(t *testing.T) {
	cfg := config.Default()
	cfg.AllowEmptyTestPool = true

	report := &RunReport{}
	applyLeaseStatsResult(report, 5, 2, cfg)

	if report.Err != nil {
		t.Fatalf("expected no error when opted out, got %v", report.Err)
	}
	if len(report.Observations) != 1 {
		t.Fatalf("expected 1 warning observation, got %d", len(report.Observations))
	}
	if !strings.Contains(report.Observations[0].Text, "routed bypass detected") {
		t.Errorf("observation missing diagnostic: %+v", report.Observations[0])
	}
}

func TestApplyLeaseStatsResult_PreservesExistingError(t *testing.T) {
	original := errors.New("playbook step failed")
	report := &RunReport{Err: original, FailureClassification: FailureClassSystem}

	applyLeaseStatsResult(report, 0, 4, config.Default())

	if !errors.Is(report.Err, original) {
		t.Fatalf("expected original error preserved, got %v", report.Err)
	}
	if report.FailureClassification != FailureClassSystem {
		t.Errorf("FailureClassification overwritten: got %q", report.FailureClassification)
	}
	// Lease-stats issues should still surface as observations.
	if len(report.Observations) != 2 {
		t.Fatalf("expected 2 warning observations (bypass + empty), got %d", len(report.Observations))
	}
}

func TestApplyLeaseStatsResult_BothIssuesHardFail(t *testing.T) {
	report := &RunReport{}
	applyLeaseStatsResult(report, 0, 3, config.Default())

	// Bypass is evaluated first, so it owns Err; empty-pool surfaces as a
	// warning observation since report.Err is now non-nil.
	if report.Err == nil {
		t.Fatal("expected hard failure")
	}
	if !strings.Contains(report.Err.Error(), "routed bypass detected") {
		t.Errorf("Err should be the bypass diagnostic, got: %v", report.Err)
	}
	if len(report.Observations) != 1 {
		t.Fatalf("expected the empty-pool diagnostic as an observation, got %d", len(report.Observations))
	}
	if !strings.Contains(report.Observations[0].Text, "0 test-mode requests") {
		t.Errorf("expected empty-pool observation, got: %+v", report.Observations[0])
	}
}

func TestApplyLeaseStatsResult_NilConfigDefaultsToStrict(t *testing.T) {
	report := &RunReport{}
	applyLeaseStatsResult(report, 0, 0, nil)

	if report.Err == nil {
		t.Fatal("expected hard failure even with nil config (strict by default)")
	}
}
