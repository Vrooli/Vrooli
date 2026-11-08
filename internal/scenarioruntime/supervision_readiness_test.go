package scenarioruntime

import (
	"testing"
	"time"
)

func TestHealthProbeDueRefreshesNotReadySnapshotsWithoutSteadyStateStorm(t *testing.T) {
	now := time.Date(2026, 11, 6, 20, 10, 35, 0, time.UTC)
	ready, starting := true, false
	for _, tc := range []struct {
		name      string
		readiness *bool
		age       time.Duration
		interval  time.Duration
		want      bool
	}{
		{name: "starting sample expires quickly", readiness: &starting, age: 2 * time.Second, interval: 45 * time.Second, want: true},
		{name: "same starting sample is fresh", readiness: &starting, age: 0, interval: 45 * time.Second},
		{name: "ready sample retains steady cadence", readiness: &ready, age: 2 * time.Second, interval: 45 * time.Second},
		{name: "disabled probing stays disabled", readiness: &starting, age: time.Minute, interval: 0},
		{name: "future observation is refreshed", readiness: &ready, age: -time.Minute, interval: 45 * time.Second, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checked := now.Add(-tc.age)
			if got := HealthProbeDue(HealthSnapshot{Readiness: tc.readiness, CheckedAt: &checked}, now, tc.interval); got != tc.want {
				t.Fatalf("due=%t want=%t", got, tc.want)
			}
		})
	}
}
