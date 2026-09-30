package lifecycle

import "testing"

func TestPlanStartDecisionTable(t *testing.T) {
	cases := []struct {
		name       string
		in         startPlanInput
		want       startDecision
		wantReason string
	}{
		{
			name: "authoritative healthy fresh reuses",
			in:   startPlanInput{RegistryPresent: true, RegistryAuthoritative: true, Healthy: true},
			want: decisionReuseRunning,
		},
		{
			name:       "authoritative unhealthy restarts",
			in:         startPlanInput{RegistryPresent: true, RegistryAuthoritative: true},
			want:       decisionStopThenStart,
			wantReason: "unhealthy",
		},
		{
			name:       "authoritative healthy but stale restarts",
			in:         startPlanInput{RegistryPresent: true, RegistryAuthoritative: true, Healthy: true, FreshnessStale: true},
			want:       decisionStopThenStart,
			wantReason: "stale",
		},
		{
			name:       "authoritative unhealthy and stale restarts",
			in:         startPlanInput{RegistryPresent: true, RegistryAuthoritative: true, FreshnessStale: true},
			want:       decisionStopThenStart,
			wantReason: "unhealthy; stale",
		},
		{
			name:       "present but non-authoritative cleans up first",
			in:         startPlanInput{RegistryPresent: true},
			want:       decisionStopThenStart,
			wantReason: "stale registry instance",
		},
		{
			name: "absent starts fresh",
			in:   startPlanInput{},
			want: decisionFreshStart,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan := planStart(tc.in)
			if plan.Decision != tc.want {
				t.Fatalf("decision = %v, want %v", plan.Decision, tc.want)
			}
			if plan.RestartReason != tc.wantReason {
				t.Fatalf("reason = %q, want %q", plan.RestartReason, tc.wantReason)
			}
		})
	}
}
