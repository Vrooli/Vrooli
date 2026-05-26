package eligibility

import "testing"

func TestDecide_NoViolations_Routed(t *testing.T) {
	s := &ViolationSummary{Total: 0}
	e := decide(s)
	if !e.Routed {
		t.Fatalf("expected Routed, got fallback; violations=%v", e.Violations)
	}
}

func TestDecide_HighSeverityRoutedDrivers_Fallback(t *testing.T) {
	s := &ViolationSummary{
		Total: 1,
		TopViolations: []ViolationExcerpt{
			{Severity: "high", RuleID: RuleRoutedDrivers, FilePath: "api/db.go", LineNumber: 3},
		},
	}
	e := decide(s)
	if e.Routed {
		t.Fatalf("expected fallback for high-sev routed_database_drivers")
	}
	if len(e.Violations) != 1 || e.Violations[0].RuleID != RuleRoutedDrivers {
		t.Fatalf("expected single routed_database_drivers excerpt, got %+v", e.Violations)
	}
}

func TestDecide_MediumHandleCapture_Fallback(t *testing.T) {
	s := &ViolationSummary{
		Total: 1,
		TopViolations: []ViolationExcerpt{
			{Severity: "medium", RuleID: RuleRoutedHandleCapture, FilePath: "api/server.go"},
		},
	}
	if decide(s).Routed {
		t.Fatalf("expected fallback for medium handle-capture")
	}
}

func TestDecide_HighDatabaseBackoff_Fallback(t *testing.T) {
	s := &ViolationSummary{
		Total: 1,
		TopViolations: []ViolationExcerpt{
			{Severity: "high", RuleID: RuleDatabaseBackoff, FilePath: "api/main.go"},
		},
	}
	if decide(s).Routed {
		t.Fatalf("expected fallback for high database_backoff (raw sql.Open)")
	}
}

// Regression: scenario-auditor groups the three routing rules under a
// single public RuleID (`routed-test-db-v1`) and exposes the originating
// rule name via the `source` field. The decision must key off `source`
// (via CanonicalRuleID) so consolidation can't silently route everything.
func TestDecide_ConsolidatedRuleID_UsesSourceForMatching(t *testing.T) {
	s := &ViolationSummary{
		Total: 1,
		TopViolations: []ViolationExcerpt{
			{
				Severity: "medium",
				RuleID:   "routed-test-db-v1",
				Source:   RuleRoutedHandleCapture,
				FilePath: "api/internal/eventlog/repository.go",
			},
		},
	}
	e := decide(s)
	if e.Routed {
		t.Fatalf("expected fallback when source=routed_database_handle_capture even under consolidated rule_id")
	}
	if len(e.Violations) != 1 {
		t.Fatalf("expected the violation to be carried into the excerpt list; got %+v", e.Violations)
	}
}

func TestDecide_LowHandleCapture_StaysRouted(t *testing.T) {
	// Low severity from handle-capture should NOT disqualify; only
	// medium-or-higher does.
	s := &ViolationSummary{
		Total: 1,
		TopViolations: []ViolationExcerpt{
			{Severity: "low", RuleID: RuleRoutedHandleCapture},
		},
	}
	if !decide(s).Routed {
		t.Fatalf("expected routed (low handle-capture is below medium threshold)")
	}
}

func TestDecide_UnrelatedRule_StaysRouted(t *testing.T) {
	s := &ViolationSummary{
		Total: 5,
		TopViolations: []ViolationExcerpt{
			{Severity: "high", RuleID: "some_other_rule"},
		},
	}
	if !decide(s).Routed {
		t.Fatalf("expected routed (unrelated rule)")
	}
}

func TestDecide_FallsBackOnByRuleSummary(t *testing.T) {
	// TopViolations is empty (e.g. trimmed by limit) but ByRule has the
	// routing rule with non-zero count. decide() must catch it.
	s := &ViolationSummary{
		Total: 0,
		ByRule: []RuleCount{
			{RuleID: RuleRoutedDrivers, Count: 1, Severity: "high"},
		},
	}
	if decide(s).Routed {
		t.Fatalf("expected fallback (ByRule reports routed_database_drivers)")
	}
}
