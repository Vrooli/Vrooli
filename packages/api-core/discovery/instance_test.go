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

// TestResolveScenarioPortVariantDependencyNeverFallsBackToLive covers a
// presentation instance: a dependency it follows at its own variant must fail
// loudly when that instance is not running. Falling back to live would answer
// with the operator's real data — the leak the follow list exists to prevent.
func TestResolveScenarioPortVariantDependencyNeverFallsBackToLive(t *testing.T) {
	t.Setenv(cliutil.EnvInstanceVariant, "presentation")
	t.Setenv(cliutil.EnvVariantDependencies, "vrooli-bridge, audio-tools")
	cliutil.ResetShadowFallbackWarning("vrooli-bridge")

	var targets []string
	runner := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		target := args[2]
		targets = append(targets, target)
		if target == "vrooli-bridge@presentation" {
			return []byte("scenario not running"), errors.New("exit status 1")
		}
		return []byte("20002\n"), nil
	}

	resolver := NewResolver(ResolverConfig{CommandRunner: runner})
	_, err := resolver.ResolveScenarioPort(context.Background(), "vrooli-bridge", "API_PORT")
	if err == nil {
		t.Fatal("variant dependency fell back to live instead of failing closed")
	}
	var discoveryErr *Error
	if !errors.As(err, &discoveryErr) || discoveryErr.Kind != ErrScenarioNotRunning {
		t.Fatalf("expected ErrScenarioNotRunning, got %v", err)
	}
	if len(targets) != 1 || targets[0] != "vrooli-bridge@presentation" {
		t.Fatalf("expected exactly one variant-targeted lookup, got %v", targets)
	}
}

// TestResolveScenarioPortUnlistedDependencyStaysLive proves the follow list is
// opt-in per dependency: anything not named still resolves live.
func TestResolveScenarioPortUnlistedDependencyStaysLive(t *testing.T) {
	t.Setenv(cliutil.EnvInstanceVariant, "presentation")
	t.Setenv(cliutil.EnvVariantDependencies, "vrooli-bridge")

	var targets []string
	runner := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		targets = append(targets, args[2])
		return []byte("20003\n"), nil
	}

	resolver := NewResolver(ResolverConfig{CommandRunner: runner})
	port, err := resolver.ResolveScenarioPort(context.Background(), "integration-hub", "API_PORT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port != 20003 || len(targets) != 1 || targets[0] != "integration-hub" {
		t.Fatalf("unlisted dependency did not resolve live: port=%d targets=%v", port, targets)
	}
}
