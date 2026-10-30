package wiring

import (
	"strings"
	"testing"

	"agent-manager/internal/runsignal"
	"agent-manager/internal/usageattribution"
)

// A problem with no measured time files no time sentence; measured time
// is worded by its kind.
func TestFrictionCostPhraseOmitsUnmeasuredTime(t *testing.T) {
	if got := frictionCostPhrase(usageattribution.Problem{}); got != "" {
		t.Errorf("unmeasured = %q, want empty", got)
	}
	if got := frictionCostPhrase(usageattribution.Problem{WallClockMS: 90_000, TimeKind: runsignal.TimeIdle}); !strings.Contains(got, "idle") {
		t.Errorf("idle = %q", got)
	}
	if got := frictionCostPhrase(usageattribution.Problem{WallClockMS: 90_000}); !strings.Contains(got, "redundant calls") {
		t.Errorf("redundant = %q", got)
	}
}
