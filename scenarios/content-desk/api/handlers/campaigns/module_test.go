package campaigns

import (
	"strings"
	"testing"
)

func TestLaunchAssetReadinessTiers(t *testing.T) {
	cases := []struct {
		name                                        string
		scenario                                    string
		slots, approved, readyForReview, inProgress int
		wantStatus, wantDetail                      string
	}{
		{name: "no slots", scenario: "web-console", slots: 0, wantStatus: "failed", wantDetail: "no launch slots reported"},
		{name: "approved outranks reviewable", scenario: "web-console", slots: 2, approved: 3, readyForReview: 1, wantStatus: "passed", wantDetail: "3 approved/published"},
		{name: "ready for review is visible", scenario: "web-console", slots: 2, readyForReview: 4, inProgress: 2, wantStatus: "passed", wantDetail: "4 ready-for-review"},
		{name: "in flight only is not yet available", scenario: "web-console", slots: 2, inProgress: 2, wantStatus: "failed", wantDetail: "no reviewable launch assets"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, detail := launchAssetReadiness(tc.scenario, tc.slots, tc.approved, tc.readyForReview, tc.inProgress)
			if status != tc.wantStatus {
				t.Fatalf("status = %q, want %q", status, tc.wantStatus)
			}
			if !strings.Contains(detail, tc.wantDetail) {
				t.Fatalf("detail = %q, want it to contain %q", detail, tc.wantDetail)
			}
		})
	}
}
