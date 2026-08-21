package discovery

import (
	"context"
	"errors"
	"os/exec"
	"sync"
	"testing"
	"time"
)

func TestResolveScenarioPortCachesWithinTTL(t *testing.T) {
	callCount := 0
	now := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
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

	resolver := NewResolver(ResolverConfig{CommandRunner: runner, CacheTTL: time.Second, Now: func() time.Time { return now }})
	if _, err := resolver.ResolveScenarioPort(context.Background(), "my-scenario", "API_PORT"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := resolver.ResolveScenarioPort(context.Background(), "my-scenario", "API_PORT"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 call within TTL, got %d", callCount)
	}
	now = now.Add(2 * time.Second)
	if _, err := resolver.ResolveScenarioPort(context.Background(), "my-scenario", "API_PORT"); err != nil {
		t.Fatalf("unexpected error after TTL: %v", err)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 calls after TTL, got %d", callCount)
	}
}

func TestResolveScenarioPortInvalidatesCacheOnFailure(t *testing.T) {
	responses := []struct {
		out []byte
		err error
	}{
		{out: []byte("1234"), err: nil},
		{out: []byte("stopped"), err: errors.New("stopped")},
		{out: []byte("5678"), err: nil},
	}
	calls := 0
	runner := func(context.Context, string, ...string) ([]byte, error) {
		response := responses[calls]
		calls++
		return response.out, response.err
	}
	resolver := NewResolver(ResolverConfig{CommandRunner: runner, CacheTTL: -time.Second})
	if _, err := resolver.ResolveScenarioPort(context.Background(), "my-scenario", "API_PORT"); err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.ResolveScenarioPort(context.Background(), "my-scenario", "API_PORT"); err == nil {
		t.Fatal("expected failed refresh")
	}
	port, err := resolver.ResolveScenarioPort(context.Background(), "my-scenario", "API_PORT")
	if err != nil || port != 5678 {
		t.Fatalf("port=%d err=%v, want 5678 and nil", port, err)
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

func TestResolveScenarioPortNoRuntimePortsIsNotRunning(t *testing.T) {
	t.Parallel()

	resolver := NewResolver(ResolverConfig{
		CommandRunner: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("Runtime error: no running runtime ports found for scenario \"my-scenario\""), errors.New("exit status 1")
		},
	})

	_, err := resolver.ResolveScenarioPort(context.Background(), "my-scenario", "API_PORT")
	if !IsScenarioNotRunning(err) {
		t.Fatalf("expected no-runtime-ports output to classify as stopped, got %v", err)
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

// TestPackageWrappersShareOneResolver is the test whose absence let the fork
// storm ship. The Resolver always had a working cache, but the package-level
// wrappers rebuilt the Resolver per call, so the cache could never hit. Asserting
// cache behavior on a hand-built Resolver passes either way; the defect is only
// visible through the public entry point.
func TestPackageWrappersShareOneResolver(t *testing.T) {
	if DefaultResolver() != DefaultResolver() {
		t.Fatal("DefaultResolver returned distinct instances; the package cache cannot hit")
	}
}

func TestPackageWrapperCacheActuallyHits(t *testing.T) {
	resolver := DefaultResolver()
	calls := 0
	restore := resolver.runner
	resolver.runner = func(context.Context, string, ...string) ([]byte, error) {
		calls++
		return []byte("4321\n"), nil
	}
	t.Cleanup(func() { resolver.runner = restore })

	hitsBefore, _ := resolver.CacheStats()
	for i := 0; i < 5; i++ {
		if _, err := ResolveScenarioPortDefault(context.Background(), "wrapper-cache-scenario"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	hitsAfter, _ := resolver.CacheStats()

	if calls != 1 {
		t.Fatalf("5 wrapper calls forked the CLI %d times, want 1", calls)
	}
	if hitsAfter <= hitsBefore {
		t.Fatalf("cache hits did not increase (%d -> %d)", hitsBefore, hitsAfter)
	}
}

// TestConcurrentLookupsCollapseToOneFork pins the per-key lock. A burst of
// callers for one scenario must cost one process, not one process each.
func TestConcurrentLookupsCollapseToOneFork(t *testing.T) {
	var mu sync.Mutex
	calls := 0
	resolver := NewResolver(ResolverConfig{
		CacheTTL: time.Minute,
		CommandRunner: func(context.Context, string, ...string) ([]byte, error) {
			mu.Lock()
			calls++
			mu.Unlock()
			time.Sleep(20 * time.Millisecond) // widen the window a stampede would exploit
			return []byte("9876\n"), nil
		},
	})

	const callers = 64
	var wg sync.WaitGroup
	wg.Add(callers)
	for i := 0; i < callers; i++ {
		go func() {
			defer wg.Done()
			if port, err := resolver.ResolveScenarioPortDefault(context.Background(), "burst"); err != nil || port != 9876 {
				t.Errorf("port=%d err=%v, want 9876 and nil", port, err)
			}
		}()
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if calls != 1 {
		t.Fatalf("%d concurrent callers forked %d times, want 1", callers, calls)
	}
}

// TestFailedLookupIsNegativeCached covers the other half of the storm: a stopped
// scenario previously cost one fork per caller per attempt, forever.
func TestFailedLookupIsNegativeCached(t *testing.T) {
	now := time.Now()
	calls := 0
	resolver := NewResolver(ResolverConfig{
		CacheTTL:         time.Minute,
		NegativeCacheTTL: 500 * time.Millisecond,
		Now:              func() time.Time { return now },
		CommandRunner: func(context.Context, string, ...string) ([]byte, error) {
			calls++
			return []byte("scenario is not running"), errors.New("exit status 1")
		},
	})

	for i := 0; i < 4; i++ {
		if _, err := resolver.ResolveScenarioPortDefault(context.Background(), "stopped"); err == nil {
			t.Fatal("expected a discovery error")
		}
	}
	if calls != 1 {
		t.Fatalf("4 failing calls forked %d times, want 1 within the negative TTL", calls)
	}

	// Past the negative TTL the scenario gets another chance — a scenario that
	// has since started must not stay pinned to its failure.
	now = now.Add(time.Second)
	if _, err := resolver.ResolveScenarioPortDefault(context.Background(), "stopped"); err == nil {
		t.Fatal("expected a discovery error")
	}
	if calls != 2 {
		t.Fatalf("calls=%d, want 2 after the negative TTL expired", calls)
	}
}

// TestTimeoutIsNotCached guards the one result that must never be shared: a
// context deadline belongs to its caller, not to the target scenario.
func TestTimeoutIsNotCached(t *testing.T) {
	calls := 0
	resolver := NewResolver(ResolverConfig{
		CacheTTL:         time.Minute,
		NegativeCacheTTL: time.Minute,
		CommandRunner: func(ctx context.Context, _ string, _ ...string) ([]byte, error) {
			calls++
			<-ctx.Done()
			return nil, ctx.Err()
		},
	})

	for i := 0; i < 2; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		_, err := resolver.ResolveScenarioPortDefault(ctx, "slow")
		cancel()
		if err == nil {
			t.Fatal("expected a timeout error")
		}
	}
	if calls != 2 {
		t.Fatalf("calls=%d, want 2: a cached timeout would deny an unrelated caller", calls)
	}
}
