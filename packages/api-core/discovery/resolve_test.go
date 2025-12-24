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

// ============================================================================
// Static Resolver Tests
// ============================================================================

func TestNewStaticResolver(t *testing.T) {
	t.Parallel()

	resolver := NewStaticResolver("http://127.0.0.1:12345")
	url, err := resolver.ResolveScenarioURLDefault(context.Background(), "any-scenario")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "http://127.0.0.1:12345" {
		t.Fatalf("expected static URL, got %q", url)
	}
}

func TestStaticResolverBypassesCLI(t *testing.T) {
	t.Parallel()

	cliCalled := false
	resolver := NewResolver(ResolverConfig{
		StaticBaseURL: "http://localhost:9999",
		CommandRunner: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			cliCalled = true
			return []byte("8080"), nil
		},
	})

	url, err := resolver.ResolveScenarioURL(context.Background(), "my-scenario", "API_PORT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "http://localhost:9999" {
		t.Fatalf("expected static URL, got %q", url)
	}
	if cliCalled {
		t.Fatal("CLI should not be called in static mode")
	}
}

func TestStaticResolverExtractsPort(t *testing.T) {
	t.Parallel()

	resolver := NewStaticResolver("http://127.0.0.1:12345")
	port, err := resolver.ResolveScenarioPort(context.Background(), "any-scenario", "API_PORT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port != 12345 {
		t.Fatalf("expected port 12345, got %d", port)
	}
}

func TestStaticResolverDefaultPorts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		url          string
		expectedPort int
	}{
		{"http://localhost", 80},
		{"https://localhost", 443},
	}

	for _, tc := range tests {
		resolver := NewStaticResolver(tc.url)
		port, err := resolver.ResolveScenarioPort(context.Background(), "my-scenario", "API_PORT")
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", tc.url, err)
		}
		if port != tc.expectedPort {
			t.Fatalf("expected port %d for %q, got %d", tc.expectedPort, tc.url, port)
		}
	}
}

func TestStaticResolverTrimsTrailingSlash(t *testing.T) {
	t.Parallel()

	resolver := NewStaticResolver("http://localhost:8080/")
	url, err := resolver.ResolveScenarioURLDefault(context.Background(), "my-scenario")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "http://localhost:8080" {
		t.Fatalf("expected trailing slash trimmed, got %q", url)
	}
}

func TestStaticResolverIgnoresScenarioSlug(t *testing.T) {
	t.Parallel()

	resolver := NewStaticResolver("http://test-server:5555")

	// Empty slug should work in static mode
	url, err := resolver.ResolveScenarioURL(context.Background(), "", "API_PORT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "http://test-server:5555" {
		t.Fatalf("expected static URL, got %q", url)
	}
}

func TestStaticResolverUnknownSchemeNoPort(t *testing.T) {
	t.Parallel()

	resolver := NewStaticResolver("ftp://localhost")
	_, err := resolver.ResolveScenarioPort(context.Background(), "my-scenario", "API_PORT")

	var discoveryErr *Error
	if !errors.As(err, &discoveryErr) || discoveryErr.Kind != ErrInvalidPort {
		t.Fatalf("expected ErrInvalidPort for unknown scheme without port, got %v", err)
	}
}
