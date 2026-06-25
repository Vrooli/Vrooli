package exposure

import (
	"context"
	"testing"

	"tunnel-manager/internal/testutil/mocks"
)

// fixedPorts is a tiny PortResolver returning a single scenario's UI port.
type fixedPorts struct {
	port int
	err  error
}

func (f fixedPorts) UIPort(context.Context, string) (int, error) { return f.port, f.err }

func TestEnsureRunning_SkipsStartWhenPortListening(t *testing.T) {
	cmd := &mocks.FakeCmdRunner{}
	runner := &CLIRunner{
		Runner: cmd.Run,
		Ports:  fixedPorts{port: 21242},
		Dial:   func(context.Context, int) bool { return true }, // already serving
	}
	if err := runner.EnsureRunning(context.Background(), "react-component-library"); err != nil {
		t.Fatalf("EnsureRunning: %v", err)
	}
	if cmd.CallCount() != 0 {
		t.Fatalf("expected zero `scenario start` shells for an already-running scenario, got %d", cmd.CallCount())
	}
}

func TestEnsureRunning_StartsWhenNotListening(t *testing.T) {
	cmd := &mocks.FakeCmdRunner{}
	runner := &CLIRunner{
		Runner: cmd.Run,
		Ports:  fixedPorts{port: 21242},
		Dial:   func(context.Context, int) bool { return false }, // cold
	}
	if err := runner.EnsureRunning(context.Background(), "react-component-library"); err != nil {
		t.Fatalf("EnsureRunning: %v", err)
	}
	if cmd.CallCount() != 1 {
		t.Fatalf("expected one `scenario start` shell for a cold scenario, got %d", cmd.CallCount())
	}
	got := cmd.Calls[0]
	if got.Name != "vrooli" || len(got.Args) < 3 || got.Args[0] != "scenario" || got.Args[1] != "start" {
		t.Fatalf("unexpected argv: %s %v", got.Name, got.Args)
	}
}

func TestEnsureRunning_RangedScenarioFallsBackToStart(t *testing.T) {
	cmd := &mocks.FakeCmdRunner{}
	// No fixed UI port (ranged scenario): resolver errors, so the fast-path is
	// disabled and EnsureRunning falls back to the idempotent start.
	runner := &CLIRunner{
		Runner: cmd.Run,
		Ports:  fixedPorts{err: ErrPortUnresolved{Scenario: "ranged", Reason: "no fixed UI port"}},
		Dial:   func(context.Context, int) bool { t.Fatal("dial must not run when port unresolved"); return false },
	}
	if err := runner.EnsureRunning(context.Background(), "ranged"); err != nil {
		t.Fatalf("EnsureRunning: %v", err)
	}
	if cmd.CallCount() != 1 {
		t.Fatalf("expected fallback start, got %d calls", cmd.CallCount())
	}
}
