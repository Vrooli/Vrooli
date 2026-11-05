package evaluation_test

import (
	"testing"
	"web-search/internal/evaluation"
)

func TestOperationalMeasurementDoesNotUseFixtures(t *testing.T) {
	report := evaluation.MeasureOperational([]evaluation.Observation{obs("fixture", "m", 1, true, true)})
	if report.Status != "unknown" || report.EligibleAttempts != 0 {
		t.Fatalf("report = %+v", report)
	}
}

func TestOperationalMeasurementCountsOperatorObservations(t *testing.T) {
	o := obs("operator", "m", 1, true, true)
	o.Provenance = "operator"
	report := evaluation.MeasureOperational([]evaluation.Observation{o})
	if report.Status != "observed" || report.EligibleAttempts != 1 || report.VerifiedSuccesses != 1 {
		t.Fatalf("report = %+v", report)
	}
}
