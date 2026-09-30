package readiness

import (
	"context"
	"testing"
	"time"
)

type fakeGoals struct{ spec GoalSpec }

func (f *fakeGoals) Open(_ context.Context, spec GoalSpec) (string, bool, error) {
	f.spec = spec
	return "goal-1", false, nil
}

type fakeAnchor struct{ releaseID, goal, commit string }

func (f *fakeAnchor) SetReadinessApproval(_ context.Context, releaseID, goal, commit string) error {
	f.releaseID, f.goal, f.commit = releaseID, goal, commit
	return nil
}

func TestTriggeredSpecNamesFiredTrigger(t *testing.T) {
	now := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	spec, fired, err := TriggeredSpec(GoalSpec{Name: "readiness/demo/abc"}, []TriggerInput{{Kind: TriggerPriceChange, PreviousValue: "10", CurrentValue: "12"}}, now)
	if err != nil || !fired || spec.Trigger != string(TriggerPriceChange) {
		t.Fatalf("spec=%+v fired=%t err=%v", spec, fired, err)
	}
}

func TestTriggeredSpecSupportsVoluntaryReview(t *testing.T) {
	spec, fired, err := TriggeredSpec(GoalSpec{Name: "readiness/demo/abc"}, nil, time.Now())
	if err != nil || fired || spec.Trigger != "" {
		t.Fatalf("spec=%+v fired=%t err=%v", spec, fired, err)
	}
}

func TestCoordinatorOpenTriggeredNamesTheFiredTrigger(t *testing.T) {
	goals := &fakeGoals{}
	anchor := &fakeAnchor{}
	c := &Coordinator{Goals: goals, Anchor: anchor}
	goal, _, fired, err := c.OpenTriggered(context.Background(), "release-1", GoalSpec{Name: "readiness/demo/abc"}, []TriggerInput{{Kind: TriggerMonetizationChange, PreviousValue: "old", CurrentValue: "new"}}, time.Now())
	if err != nil || !fired || goal == "" {
		t.Fatalf("OpenTriggered = %q, fired=%t, err=%v", goal, fired, err)
	}
	if goals.spec.Trigger != string(TriggerMonetizationChange) {
		t.Fatalf("trigger = %q", goals.spec.Trigger)
	}
}
