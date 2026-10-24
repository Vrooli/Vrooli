package readiness

import (
	"testing"
)

func TestGoogleReadinessHasSixIndependentRungs(t *testing.T) {
	readiness := GoogleReadiness(false, false, false, true, false, false)
	if err := readiness.Validate(); err != nil {
		t.Fatal(err)
	}
	if readiness.Rungs[3].State != RungReady || readiness.Rungs[0].State != RungUnavailable {
		t.Fatalf("unexpected readiness: %#v", readiness)
	}
	if readiness.Rungs[1].Obligation == "" {
		t.Fatal("developer-verification rung omitted its dated obligation")
	}
	if readiness.Rungs[3].Obligation == "" {
		t.Fatal("target-api rung omitted its dated obligation")
	}
}
