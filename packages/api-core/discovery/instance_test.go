package discovery

import (
	"context"
	"errors"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestResolveScenarioPortRoutesToShadowWhenShadowed(t *testing.T) {
	t.Setenv(cliutil.EnvShadowScenarios, "agent-manager")

	var gotTarget string
	runner := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		// args: scenario port <target> API_PORT
		gotTarget = args[2]
		return []byte("19001\n"), nil
	}

	resolver := NewResolver(ResolverConfig{CommandRunner: runner})
	port, err := resolver.ResolveScenarioPort(context.Background(), "agent-manager", "API_PORT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotTarget != "agent-manager@shadow" {
		t.Fatalf("expected shadow target, got %q", gotTarget)
	}
	if port != 19001 {
		t.Fatalf("expected 19001, got %d", port)
	}
}

func TestResolveScenarioPortLiveUnaffectedForUnshadowed(t *testing.T) {
	t.Setenv(cliutil.EnvShadowScenarios, "agent-manager")

	var gotTarget string
	runner := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		gotTarget = args[2]
		return []byte("12345\n"), nil
	}

	resolver := NewResolver(ResolverConfig{CommandRunner: runner})
	if _, err := resolver.ResolveScenarioPort(context.Background(), "my-scenario", "API_PORT"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotTarget != "my-scenario" {
		t.Fatalf("expected bare target for unshadowed scenario, got %q", gotTarget)
	}
}

func TestResolveScenarioPortFallsBackToLiveWhenShadowNotRunning(t *testing.T) {
	t.Setenv(cliutil.EnvShadowScenarios, "swarm-manager")
	cliutil.ResetShadowFallbackWarning("swarm-manager")

	var targets []string
	runner := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		target := args[2]
		targets = append(targets, target)
		if target == "swarm-manager@shadow" {
			return []byte("scenario not running"), errors.New("exit status 1")
		}
		return []byte("20002\n"), nil
	}

	resolver := NewResolver(ResolverConfig{CommandRunner: runner})
	port, err := resolver.ResolveScenarioPort(context.Background(), "swarm-manager", "API_PORT")
	if err != nil {
		t.Fatalf("unexpected error after fallback: %v", err)
	}
	if len(targets) != 2 || targets[0] != "swarm-manager@shadow" || targets[1] != "swarm-manager" {
		t.Fatalf("expected shadow-then-live lookups, got %v", targets)
	}
	if port != 20002 {
		t.Fatalf("expected live fallback port 20002, got %d", port)
	}
}

func TestResolveScenarioPortShadowOtherErrorDoesNotFallBack(t *testing.T) {
	t.Setenv(cliutil.EnvShadowScenarios, "swarm-manager")

	var calls int
	runner := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		calls++
		// A non-"not running" failure must surface, not silently retry live.
		return []byte("boom"), errors.New("exit status 2")
	}

	resolver := NewResolver(ResolverConfig{CommandRunner: runner})
	_, err := resolver.ResolveScenarioPort(context.Background(), "swarm-manager", "API_PORT")
	var discoveryErr *Error
	if !errors.As(err, &discoveryErr) || discoveryErr.Kind != ErrCommandFailed {
		t.Fatalf("expected ErrCommandFailed, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected no live fallback for non-not-running error, got %d calls", calls)
	}
}
