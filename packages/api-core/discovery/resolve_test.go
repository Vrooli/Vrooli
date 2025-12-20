package discovery

import (
	"context"
	"errors"
	"os/exec"
	"testing"
)

func TestResolveScenarioPortCallsCLIEachTime(t *testing.T) {
	t.Parallel()

	callCount := 0
	runner := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		callCount++
		if name != "vrooli" {
			t.Fatalf("expected vrooli, got %q", name)
		}
		expected := []string{"scenario", "port", "my-scenario", "API_PORT"}
		if len(args) != len(expected) {
			t.Fatalf("unexpected args: %v", args)
		}
		for i, arg := range expected {
			if args[i] != arg {
				t.Fatalf("unexpected args: %v", args)
			}
		}
		return []byte("12345\n"), nil
	}

	resolver := NewResolver(ResolverConfig{CommandRunner: runner})
	if _, err := resolver.ResolveScenarioPort(context.Background(), "my-scenario", "API_PORT"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := resolver.ResolveScenarioPort(context.Background(), "my-scenario", "API_PORT"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 calls, got %d", callCount)
	}
}

func TestResolveScenarioPortInvalidOutput(t *testing.T) {
	t.Parallel()

	resolver := NewResolver(ResolverConfig{
		CommandRunner: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("nope"), nil
		},
	})

	_, err := resolver.ResolveScenarioPort(context.Background(), "my-scenario", "API_PORT")
	var discoveryErr *Error
	if !errors.As(err, &discoveryErr) || discoveryErr.Kind != ErrInvalidPort {
		t.Fatalf("expected ErrInvalidPort, got %v", err)
	}
}

func TestResolveScenarioPortDefaultsPortKey(t *testing.T) {
	t.Parallel()

	runner := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if len(args) != 4 || args[3] != "API_PORT" {
			t.Fatalf("expected API_PORT default, got args: %v", args)
		}
		return []byte("4444"), nil
	}

	resolver := NewResolver(ResolverConfig{CommandRunner: runner})
	port, err := resolver.ResolveScenarioPort(context.Background(), "my-scenario", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port != 4444 {
		t.Fatalf("unexpected port: %d", port)
	}
}

func TestResolveScenarioPortNotRunning(t *testing.T) {
	t.Parallel()

	resolver := NewResolver(ResolverConfig{
		CommandRunner: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("scenario not running"), errors.New("exit 1")
		},
	})

	_, err := resolver.ResolveScenarioPort(context.Background(), "my-scenario", "API_PORT")
	var discoveryErr *Error
	if !errors.As(err, &discoveryErr) || discoveryErr.Kind != ErrScenarioNotRunning {
		t.Fatalf("expected ErrScenarioNotRunning, got %v", err)
	}
}

func TestResolveScenarioPortVrooliMissing(t *testing.T) {
	t.Parallel()

	resolver := NewResolver(ResolverConfig{
		CommandRunner: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, exec.ErrNotFound
		},
	})

	_, err := resolver.ResolveScenarioPort(context.Background(), "my-scenario", "API_PORT")
	var discoveryErr *Error
	if !errors.As(err, &discoveryErr) || discoveryErr.Kind != ErrVrooliNotFound {
		t.Fatalf("expected ErrVrooliNotFound, got %v", err)
	}
}

func TestResolveScenarioURLDefaults(t *testing.T) {
	t.Parallel()

	resolver := NewResolver(ResolverConfig{
		CommandRunner: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("8080"), nil
		},
	})

	url, err := resolver.ResolveScenarioURL(context.Background(), "my-scenario", "API_PORT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "http://localhost:8080" {
		t.Fatalf("unexpected url: %q", url)
	}
}

func TestResolveScenarioURLOverrides(t *testing.T) {
	t.Parallel()

	resolver := NewResolver(ResolverConfig{
		Scheme: "https",
		Host:   "127.0.0.1",
		CommandRunner: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("443"), nil
		},
	})

	url, err := resolver.ResolveScenarioURL(context.Background(), "my-scenario", "API_PORT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://127.0.0.1:443" {
		t.Fatalf("unexpected url: %q", url)
	}
}

func TestResolveScenarioURLDefaultConvenience(t *testing.T) {
	t.Parallel()

	runner := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if len(args) != 4 || args[3] != "API_PORT" {
			t.Fatalf("expected API_PORT default, got args: %v", args)
		}
		return []byte("9090"), nil
	}

	url, err := NewResolver(ResolverConfig{CommandRunner: runner}).
		ResolveScenarioURLDefault(context.Background(), "my-scenario")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "http://localhost:9090" {
		t.Fatalf("unexpected url: %q", url)
	}
}
