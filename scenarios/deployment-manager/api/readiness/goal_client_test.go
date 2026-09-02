package readiness

import (
	"context"
	"testing"
)

func TestGoalClientRequiresResolver(t *testing.T) {
	_, _, err := (&GoalClient{}).Open(context.Background(), GoalSpec{Name: "readiness/demo/abc"})
	if err == nil {
		t.Fatal("expected unconfigured client refusal")
	}
}
