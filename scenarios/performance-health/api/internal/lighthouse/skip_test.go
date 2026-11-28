package lighthouse

import (
	"context"
	"testing"
)

// [REQ:PH-LH-002] Lighthouse SKIPS cleanly (never errors) when the runner is not
// wired (e.g. CLI absent / no UI URL).
func TestScoreSkipsWhenRunnerAbsent(t *testing.T) {
	svc := NewService(nil)
	res, err := svc.Score(context.Background(), "demo", "")
	if err != nil {
		t.Fatalf("Score should not error on skip: %v", err)
	}
	if res.Outcome != OutcomeSkipped {
		t.Fatalf("expected SKIPPED, got %#v", res)
	}
}
