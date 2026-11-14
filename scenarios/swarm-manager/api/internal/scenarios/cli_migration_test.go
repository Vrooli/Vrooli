package scenarios

import (
	"context"
	"errors"
	"strings"
	"testing"

	vroolicli "github.com/vrooli/vrooli-cli-go"
)

// fakeRunner is a vroolicli.Runner that returns canned output and records the
// args it was invoked with.
type fakeRunner struct {
	out     []byte
	err     error
	gotArgs []string
}

func (f *fakeRunner) Run(_ context.Context, _ string, args ...string) ([]byte, error) {
	f.gotArgs = args
	return f.out, f.err
}

func (f *fakeRunner) RunCombined(_ context.Context, _ string, args ...string) ([]byte, error) {
	f.gotArgs = args
	return f.out, f.err
}

func clientWith(f *fakeRunner) *vroolicli.Client {
	return vroolicli.New(vroolicli.WithRunner(f))
}

func TestCLIProviderListDecodesTypedContract(t *testing.T) {
	runner := &fakeRunner{out: []byte(`{"success":true,"scenarios":[
		{"name":"alpha","description":"A","path":"/s/alpha","version":"1.0","status":"running","tags":["x"]},
		{"name":"","path":"/s/ghost"},
		{"name":"beta","path":"/s/beta","status":"stopped"}
	]}`)}
	p := &CLIProvider{client: clientWith(runner), includePorts: true}

	got, err := p.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	// The nameless entry is skipped.
	if len(got) != 2 {
		t.Fatalf("want 2 scenarios, got %d: %+v", len(got), got)
	}
	if got[0].Name != "alpha" || got[0].Description != "A" || got[0].Status != "running" || got[0].Version != "1.0" {
		t.Errorf("alpha mapped wrong: %+v", got[0])
	}
	// includePorts must request port data.
	if !containsArg(runner.gotArgs, "--include-ports") {
		t.Errorf("expected --include-ports in args, got %v", runner.gotArgs)
	}
}

// TestHealthProbeReadsRealContract is the regression guard for the bug the proto
// migration surfaced: the prior probe parsed a `scenario_data`/`diagnostics`
// shape the CLI never emits, so every scenario reported unhealthy. The typed
// contract reads scenario.status / scenario.health_status directly.
func TestHealthProbeReadsRealContract(t *testing.T) {
	t.Run("running+healthy => healthy", func(t *testing.T) {
		runner := &fakeRunner{out: []byte(`{"success":true,"scenario":{"name":"em","status":"running","health_status":"healthy"}}`)}
		c := &CLIHealthChecker{client: clientWith(runner)}

		snap, err := c.Check(context.Background(), "em")
		if err != nil {
			t.Fatalf("Check: %v", err)
		}
		if !snap.Healthy {
			t.Fatalf("expected healthy, got unhealthy: %q", snap.Details)
		}
		if snap.ScenarioStatus != "running" || snap.HealthStatus != "healthy" {
			t.Errorf("fields mapped wrong: %+v", snap)
		}
		if snap.Details != "scenario is healthy" {
			t.Errorf("details = %q", snap.Details)
		}
	})

	t.Run("stopped+null health => unhealthy with real reasons", func(t *testing.T) {
		runner := &fakeRunner{out: []byte(`{"success":true,"scenario":{"name":"em","status":"stopped","health_status":null,"health_error":"port closed"}}`)}
		c := &CLIHealthChecker{client: clientWith(runner)}

		snap, err := c.Check(context.Background(), "em")
		if err != nil {
			t.Fatalf("Check: %v", err)
		}
		if snap.Healthy {
			t.Fatal("expected unhealthy")
		}
		for _, want := range []string{"scenario status=stopped", "health status=none", "port closed"} {
			if !strings.Contains(snap.Details, want) {
				t.Errorf("details %q missing %q", snap.Details, want)
			}
		}
	})

	t.Run("empty name => name required", func(t *testing.T) {
		c := &CLIHealthChecker{client: clientWith(&fakeRunner{})}
		if _, err := c.Check(context.Background(), "  "); !errors.Is(err, errScenarioNameRequired) {
			t.Fatalf("want errScenarioNameRequired, got %v", err)
		}
	})
}

func TestCLILifecycleInvokesVerb(t *testing.T) {
	runner := &fakeRunner{out: []byte("ok")}
	lc := &CLILifecycle{
		client:         clientWith(runner),
		startTimeout:   defaultStartTimeout,
		stopTimeout:    defaultStopTimeout,
		restartTimeout: defaultRestartTimeout,
	}

	if err := lc.Restart(context.Background(), "em"); err != nil {
		t.Fatalf("Restart: %v", err)
	}
	if !containsArg(runner.gotArgs, "scenario") || !containsArg(runner.gotArgs, "restart") || !containsArg(runner.gotArgs, "em") {
		t.Errorf("unexpected args: %v", runner.gotArgs)
	}

	if err := lc.Start(context.Background(), "   "); !errors.Is(err, errScenarioNameRequired) {
		t.Errorf("want errScenarioNameRequired for blank name, got %v", err)
	}
}

func containsArg(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}
