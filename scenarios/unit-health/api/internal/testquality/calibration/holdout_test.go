package calibration

import (
	"testing"
	"unit-health/internal/testquality"
)

func TestCompareHoldoutKeepsMissingAsUnknown(t *testing.T) {
	labels := []testquality.HoldoutLabel{{TestIdentity: "t1", SourceIdentity: "s", RuleVersion: "r1", Label: "behavioral", Reviewer: "human", ReviewedAt: "2026-09-08"}}
	r, err := CompareHoldout(labels, nil, "r1")
	if err != nil {
		t.Fatal(err)
	}
	if r.Denominator != 1 || r.Unknown != 1 || r.PromotionAllowed {
		t.Fatalf("unexpected report: %+v", r)
	}
}

func TestCompareHoldoutReportsConfusionAndNeverAutoPromotes(t *testing.T) {
	labels := make([]testquality.HoldoutLabel, 5)
	obs := make([]testquality.SampledObservation, 5)
	for i := range labels {
		id := string(rune('a' + i))
		labels[i] = testquality.HoldoutLabel{TestIdentity: id, SourceIdentity: "s", RuleVersion: "r1", Label: "behavioral", Reviewer: "human", ReviewedAt: "2026-09-08"}
		obs[i] = testquality.SampledObservation{TestIdentity: id, SourceIdentity: "s", Label: "behavioral", Status: "observed"}
	}
	r, err := CompareHoldout(labels, obs, "r1")
	if err != nil {
		t.Fatal(err)
	}
	if r.Compared != 5 || r.Confusion["behavioral"]["behavioral"] != 5 || r.PromotionAllowed {
		t.Fatalf("unexpected report: %+v", r)
	}
	if r.PromotionReason == "" {
		t.Fatal("promotion rationale missing")
	}
}
