package runtime

import (
	"testing"
	"time"
)

func TestFleetSchedulerOptionsFromLookup(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want fleetSchedulerOptions
	}{
		{
			name: "default off with documented defaults",
			want: fleetSchedulerOptions{
				preset:           "comprehensive",
				interval:         6 * time.Hour,
				maxConcurrent:    1,
				maxRunsPerCycle:  5,
				stalenessHorizon: 7 * 24 * time.Hour,
			},
		},
		{
			name: "configured values",
			env: map[string]string{
				"TEST_GENIE_FLEET_SCHEDULER_ENABLED":           "true",
				"TEST_GENIE_FLEET_SCHEDULER_PRESET":            "quick",
				"TEST_GENIE_FLEET_SCHEDULER_INTERVAL":          "2h",
				"TEST_GENIE_FLEET_SCHEDULER_START_JITTER":      "30",
				"TEST_GENIE_FLEET_SCHEDULER_MAX_CONCURRENT":    "3",
				"TEST_GENIE_FLEET_SCHEDULER_MAX_PER_CYCLE":     "7",
				"TEST_GENIE_FLEET_SCHEDULER_CYCLE_BUDGET":      "90s",
				"TEST_GENIE_FLEET_SCHEDULER_STALENESS_HORIZON": "48h",
			},
			want: fleetSchedulerOptions{
				enabled:          true,
				preset:           "quick",
				interval:         2 * time.Hour,
				initialJitter:    30 * time.Second,
				maxConcurrent:    3,
				maxRunsPerCycle:  7,
				cycleBudget:      90 * time.Second,
				stalenessHorizon: 48 * time.Hour,
			},
		},
		{
			name: "invalid values fall back without enabling",
			env: map[string]string{
				"TEST_GENIE_FLEET_SCHEDULER_ENABLED":           "sometimes",
				"TEST_GENIE_FLEET_SCHEDULER_PRESET":            "",
				"TEST_GENIE_FLEET_SCHEDULER_INTERVAL":          "soon",
				"TEST_GENIE_FLEET_SCHEDULER_START_JITTER":      "-1s",
				"TEST_GENIE_FLEET_SCHEDULER_MAX_CONCURRENT":    "0",
				"TEST_GENIE_FLEET_SCHEDULER_MAX_PER_CYCLE":     "-2",
				"TEST_GENIE_FLEET_SCHEDULER_CYCLE_BUDGET":      "n/a",
				"TEST_GENIE_FLEET_SCHEDULER_STALENESS_HORIZON": "bad",
			},
			want: fleetSchedulerOptions{
				preset:           "comprehensive",
				interval:         6 * time.Hour,
				maxConcurrent:    1,
				maxRunsPerCycle:  5,
				stalenessHorizon: 7 * 24 * time.Hour,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fleetSchedulerOptionsFromLookup(func(key string) string { return tt.env[key] })
			if got != tt.want {
				t.Fatalf("options = %#v, want %#v", got, tt.want)
			}
		})
	}
}
