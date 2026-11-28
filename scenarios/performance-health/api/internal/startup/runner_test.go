package startup

import (
	"context"
	"errors"
	"testing"
	"time"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

type fakeStatus struct {
	item *cliv1.ScenarioStatusItem
	err  error
}

func (f fakeStatus) ScenarioStatus(context.Context, string) (*cliv1.ScenarioStatusSingle, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &cliv1.ScenarioStatusSingle{Scenario: f.item}, nil
}

// [REQ:PH-STARTUP-001] The real CLIRunner restarts via the injected restart hook
// and times time-to-healthy from the status seam (no real lifecycle calls).
func TestCLIRunnerMeasuresTimeToHealthy(t *testing.T) {
	restarted := false
	r := &CLIRunner{
		Status:       fakeStatus{item: &cliv1.ScenarioStatusItem{Status: "running"}},
		Restart:      func(context.Context, string) error { restarted = true; return nil },
		PollInterval: time.Millisecond,
	}
	m, err := r.Measure(context.Background(), "demo", 2*time.Second)
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	if !restarted {
		t.Fatal("expected restart hook to be invoked")
	}
	if !m.Healthy {
		t.Fatalf("expected healthy measurement, got %+v", m)
	}
	if m.Metrics == nil {
		t.Fatal("expected a resource-envelope metrics collection")
	}
}

// [REQ:PH-STARTUP-001] A restart failure surfaces as an error and a noted,
// unhealthy measurement.
func TestCLIRunnerRestartFailure(t *testing.T) {
	r := &CLIRunner{
		Status:  fakeStatus{},
		Restart: func(context.Context, string) error { return errors.New("boom") },
	}
	m, err := r.Measure(context.Background(), "demo", time.Second)
	if err == nil {
		t.Fatal("expected restart failure to surface")
	}
	if m.Healthy {
		t.Fatal("a failed restart must not report healthy")
	}
}

// [REQ:PH-STARTUP-001] An empty scenario is rejected before any restart.
func TestCLIRunnerRequiresScenario(t *testing.T) {
	r := &CLIRunner{}
	if _, err := r.Measure(context.Background(), "  ", time.Second); err == nil {
		t.Fatal("expected empty scenario to be rejected")
	}
}
