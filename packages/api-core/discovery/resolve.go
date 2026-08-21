// Package discovery provides runtime helpers for resolving scenario ports.
package discovery

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

// CommandRunner executes a command and returns combined stdout/stderr.
type CommandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

// CommandScopeResolver maps one remote argv verb to the concrete catalog scope
// derived from its owning CLI manifest. The bool is false for an unknown or
// ambiguous verb.
type CommandScopeResolver func(command string) (scope string, ok bool)

// ResolverConfig configures a Resolver.
type ResolverConfig struct {
	// VrooliPath is the CLI binary to invoke. Defaults to "vrooli".
	VrooliPath string

	// CommandRunner overrides command execution (useful for tests).
	CommandRunner CommandRunner

	// Host defaults to "localhost" when building URLs.
	Host string

	// Scheme defaults to "http" when building URLs.
	Scheme string

	// StaticBaseURL bypasses CLI discovery entirely and uses this URL for all
	// resolutions. Useful for testing with httptest.Server. When set, the
	// scenario slug is ignored and this URL is returned directly.
	// Example: "http://127.0.0.1:12345"
	StaticBaseURL string

	// CacheTTL bounds how long a successful runtime address is reused. A short
	// cache removes one CLI process per provider leaf while preserving dynamic
	// port correctness; a failed lookup invalidates the entry immediately.
	// Zero uses the default two-second TTL. Set it negative to disable caching.
	CacheTTL time.Duration

	// NegativeCacheTTL bounds how long a failed lookup is reused. Without it a
	// stopped scenario costs one CLI fork per caller per attempt, which is the
	// exact shape of a fork storm. Kept much shorter than CacheTTL so a
	// scenario that has just started is not held down. Zero uses the default;
	// set it negative to re-fork on every failure.
	NegativeCacheTTL time.Duration

	// Now supplies time for deterministic cache tests. Nil uses time.Now.
	Now func() time.Time

	// TargetResolver and Relay provide the optional node transport. Leaving
	// both nil preserves the local resolver and its exact CLI behavior.
	TargetResolver TargetResolver
	Relay          RelayTransport
	CommandScope   CommandScopeResolver
}

// Resolver resolves scenario ports by shelling out to the Vrooli CLI. Successful
// addresses are cached briefly and failed lookups invalidate the entry, so a
// restarted scenario is never pinned to a stale address for longer than the
// configured TTL.
// If configured with a static base URL, it bypasses CLI discovery entirely.
type Resolver struct {
	vrooliPath     string
	runner         CommandRunner
	host           string
	scheme         string
	staticBaseURL  string // When set, bypasses CLI discovery
	targetResolver TargetResolver
	relay          RelayTransport
	commandScope   CommandScopeResolver
	cacheTTL       time.Duration
	negativeTTL    time.Duration
	now            func() time.Time
	cacheMu        sync.Mutex
	cache          map[string]*cachedPort
	cacheHits      int64
	cacheMisses    int64
}

// cachedPort holds one key's resolution plus the lock that serializes lookups
// for that key. err is non-nil for a cached negative result.
type cachedPort struct {
	mu         sync.Mutex
	port       int
	err        *Error
	resolvedAt time.Time
}

const defaultPortKey = "API_PORT"

// defaultResolverCacheTTL amortizes fan-out resolution for one query while
// bounding the stale-address window after a scenario restart.
const defaultResolverCacheTTL = 2 * time.Second

// defaultResolverNegativeCacheTTL suppresses a fork stampede against a stopped
// scenario while keeping the recovery window short enough that a scenario which
// just came up is picked up promptly.
const defaultResolverNegativeCacheTTL = 500 * time.Millisecond

// ErrorKind identifies the class of discovery failure.
type ErrorKind string

const (
	ErrInvalidInput               ErrorKind = "invalid_input"
	ErrVrooliNotFound             ErrorKind = "vrooli_not_found"
	ErrScenarioNotRunning         ErrorKind = "scenario_not_running"
	ErrTimeout                    ErrorKind = "timeout"
	ErrInvalidPort                ErrorKind = "invalid_port"
	ErrCommandFailed              ErrorKind = "command_failed"
	ErrNodeOffline                ErrorKind = "node_offline"
	ErrNodeOutOfScope             ErrorKind = "node_out_of_scope"
	ErrNodeUnpaired               ErrorKind = "node_unpaired_or_revoked"
	ErrRemoteTransportUnavailable ErrorKind = "remote_transport_unavailable"
	ErrRemoteCallFailed           ErrorKind = "remote_call_failed"
)

// Error provides structured details about discovery failures.
type Error struct {
	Kind     ErrorKind
	Scenario string
	PortKey  string
	Node     string
	Output   string
	Err      error
}

func (e *Error) Error() string {
	parts := []string{
		"api-core discovery",
		string(e.Kind),
		fmt.Sprintf("scenario=%q", e.Scenario),
		fmt.Sprintf("port=%q", e.PortKey),
	}
	if e.Node != "" {
		parts = append(parts, fmt.Sprintf("node=%q", e.Node))
	}
	if e.Output != "" {
		parts = append(parts, fmt.Sprintf("output=%q", e.Output))
	}
	if e.Err != nil {
		parts = append(parts, fmt.Sprintf("err=%v", e.Err))
	}
	return strings.Join(parts, " ")
}

func (e *Error) Unwrap() error {
	return e.Err
}

// IsScenarioNotRunning reports whether the error indicates a stopped scenario.
func IsScenarioNotRunning(err error) bool {
	var target *Error
	return errors.As(err, &target) && target.Kind == ErrScenarioNotRunning
}

// NewResolver constructs a Resolver with defaults applied.
func NewResolver(cfg ResolverConfig) *Resolver {
	vrooliPath := cfg.VrooliPath
	if vrooliPath == "" {
		vrooliPath = "vrooli"
	}
	runner := cfg.CommandRunner
	if runner == nil {
		runner = defaultRunner
	}
	host := cfg.Host
	if host == "" {
		host = "localhost"
	}
	scheme := cfg.Scheme
	if scheme == "" {
		scheme = "http"
	}
	cacheTTL := cfg.CacheTTL
	if cacheTTL == 0 {
		cacheTTL = defaultResolverCacheTTL
	}
	negativeTTL := cfg.NegativeCacheTTL
	if negativeTTL == 0 {
		negativeTTL = defaultResolverNegativeCacheTTL
	}
	if negativeTTL < 0 {
		negativeTTL = 0
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &Resolver{
		vrooliPath:     vrooliPath,
		runner:         runner,
		host:           host,
		scheme:         scheme,
		staticBaseURL:  strings.TrimRight(cfg.StaticBaseURL, "/"),
		cacheTTL:       cacheTTL,
		negativeTTL:    negativeTTL,
		now:            now,
		cache:          make(map[string]*cachedPort),
		targetResolver: cfg.TargetResolver,
		relay:          cfg.Relay,
		commandScope:   cfg.CommandScope,
	}
}

// NewStaticResolver creates a Resolver that bypasses CLI discovery and always
// returns the provided base URL. This is useful for testing with httptest.Server.
//
// Example:
//
//	server := httptest.NewServer(handler)
//	defer server.Close()
//	resolver := discovery.NewStaticResolver(server.URL)
//	client := NewClient(resolver, server.Client())
func NewStaticResolver(baseURL string) *Resolver {
	return NewResolver(ResolverConfig{
		StaticBaseURL: baseURL,
	})
}

// ResolveScenarioPort resolves a scenario's port by calling:
// `vrooli scenario port <slug> <portKey>`.
//
// Successful lookups are cached for ResolverConfig.CacheTTL and failures for the
// shorter ResolverConfig.NegativeCacheTTL, so neither a running nor a stopped
// scenario costs a process per caller. Concurrent callers for one key collapse
// onto a single invocation. A context timeout is never cached, since it
// describes the caller's deadline rather than the target's state.
//
// If the resolver was created with a static base URL, the port is extracted
// from that URL instead of invoking the CLI.
func (r *Resolver) ResolveScenarioPort(ctx context.Context, scenarioSlug, portKey string) (int, error) {
	// Static mode: extract port from the configured URL
	if r.staticBaseURL != "" {
		return r.extractPortFromStaticURL(scenarioSlug, portKey)
	}

	if scenarioSlug == "" {
		return 0, &Error{
			Kind:     ErrInvalidInput,
			Scenario: scenarioSlug,
			PortKey:  portKey,
			Err:      errors.New("scenario slug is required"),
		}
	}
	if portKey == "" {
		portKey = defaultPortKey
	}
	cacheKey := scenarioSlug + "\x00" + portKey
	return r.resolvePortCached(ctx, scenarioSlug, portKey, cacheKey)
}

// resolvePortCached collapses concurrent lookups for one key onto a single CLI
// invocation and reuses both successful and failed results for their respective
// TTLs. Holding the per-key lock across the lookup is deliberate: a burst of N
// callers for the same scenario must cost one fork, not N. Before this, every
// caller forked `vrooli scenario port` because the only cache lived on a
// Resolver that the package-level wrappers rebuilt per call.
func (r *Resolver) resolvePortCached(ctx context.Context, scenarioSlug, portKey, cacheKey string) (int, error) {
	if r.cacheTTL < 0 {
		port, derr := r.lookupPortWithFallback(ctx, scenarioSlug, portKey)
		if derr != nil {
			return 0, derr
		}
		return port, nil
	}

	entry := r.entryFor(cacheKey)
	entry.mu.Lock()
	defer entry.mu.Unlock()

	if !entry.resolvedAt.IsZero() {
		ttl := r.cacheTTL
		if entry.err != nil {
			ttl = r.negativeTTL
		}
		if r.now().Sub(entry.resolvedAt) < ttl {
			r.countHit()
			if entry.err != nil {
				return 0, entry.err
			}
			return entry.port, nil
		}
	}
	r.countMiss()

	port, derr := r.lookupPortWithFallback(ctx, scenarioSlug, portKey)

	// A timeout reflects the caller's deadline, not the target's state. Caching
	// it would let one caller's cancellation deny an unrelated caller, so this
	// result is returned without being recorded.
	if derr != nil && derr.Kind == ErrTimeout {
		entry.resolvedAt = time.Time{}
		return 0, derr
	}

	entry.port, entry.err, entry.resolvedAt = port, derr, r.now()
	if derr != nil {
		return 0, derr
	}
	return port, nil
}

// lookupPortWithFallback performs instance routing (Case B): when the target
// scenario is ambiently shadowed (VROOLI_SHADOW_SCENARIOS), address its
// "@shadow" record. If that non-live lookup reports the scenario isn't running
// — the engagement may have been torn down — warn once and fall back to the
// live instance. Never silent.
func (r *Resolver) lookupPortWithFallback(ctx context.Context, scenarioSlug, portKey string) (int, *Error) {
	target := cliutil.ResolveShadowTarget(scenarioSlug)
	port, derr := r.lookupPort(ctx, scenarioSlug, target, portKey)
	if derr != nil && cliutil.IsNonLiveTarget(target) && derr.Kind == ErrScenarioNotRunning {
		cliutil.WarnShadowFallback(scenarioSlug)
		port, derr = r.lookupPort(ctx, scenarioSlug, scenarioSlug, portKey)
	}
	return port, derr
}

func (r *Resolver) entryFor(key string) *cachedPort {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()
	entry, ok := r.cache[key]
	if !ok {
		entry = &cachedPort{}
		r.cache[key] = entry
	}
	return entry
}

func (r *Resolver) countHit() {
	r.cacheMu.Lock()
	r.cacheHits++
	r.cacheMu.Unlock()
}

func (r *Resolver) countMiss() {
	r.cacheMu.Lock()
	r.cacheMisses++
	r.cacheMu.Unlock()
}

// CacheStats returns cumulative successful cache lookups and misses. The
// counters are resolver-local: callers can take a before/after sample around
// one operation without exposing cache contents or corpus data.
func (r *Resolver) CacheStats() (hits, misses int64) {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()
	return r.cacheHits, r.cacheMisses
}

// lookupPort shells `vrooli scenario port <target> <portKey>` and classifies the
// result. reportSlug is the user-facing scenario name recorded on any Error
// (which may differ from target when routing to a variant record).
func (r *Resolver) lookupPort(ctx context.Context, reportSlug, target, portKey string) (int, *Error) {
	output, err := r.runner(ctx, r.vrooliPath, "scenario", "port", target, portKey)
	text := strings.TrimSpace(string(output))
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return 0, &Error{
				Kind:     ErrTimeout,
				Scenario: reportSlug,
				PortKey:  portKey,
				Output:   text,
				Err:      ctxErr,
			}
		}
		if errors.Is(err, exec.ErrNotFound) {
			return 0, &Error{
				Kind:     ErrVrooliNotFound,
				Scenario: reportSlug,
				PortKey:  portKey,
				Output:   text,
				Err:      err,
			}
		}
		lower := strings.ToLower(text)
		if strings.Contains(lower, "not running") || strings.Contains(lower, "not started") || strings.Contains(lower, "no running runtime ports") || strings.Contains(lower, "no runtime ports found") {
			return 0, &Error{
				Kind:     ErrScenarioNotRunning,
				Scenario: reportSlug,
				PortKey:  portKey,
				Output:   text,
				Err:      err,
			}
		}
		return 0, &Error{
			Kind:     ErrCommandFailed,
			Scenario: reportSlug,
			PortKey:  portKey,
			Output:   text,
			Err:      err,
		}
	}

	port, parseErr := strconv.Atoi(text)
	if parseErr != nil || port <= 0 {
		return 0, &Error{
			Kind:     ErrInvalidPort,
			Scenario: reportSlug,
			PortKey:  portKey,
			Output:   text,
			Err:      parseErr,
		}
	}

	return port, nil
}

// ResolveScenarioURL resolves a scenario's port and returns a URL
// using the resolver's scheme and host.
//
// If the resolver was created with a static base URL, that URL is returned
// directly without invoking the CLI.
func (r *Resolver) ResolveScenarioURL(ctx context.Context, scenarioSlug, portKey string) (string, error) {
	// Static mode: return the configured URL directly
	if r.staticBaseURL != "" {
		return r.staticBaseURL, nil
	}

	port, err := r.ResolveScenarioPort(ctx, scenarioSlug, portKey)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s://%s:%d", r.scheme, r.host, port), nil
}

// ResolveScenarioPortDefault resolves the standard API port for a scenario.
func (r *Resolver) ResolveScenarioPortDefault(ctx context.Context, scenarioSlug string) (int, error) {
	return r.ResolveScenarioPort(ctx, scenarioSlug, defaultPortKey)
}

// ResolveScenarioURLDefault resolves the standard API URL for a scenario.
func (r *Resolver) ResolveScenarioURLDefault(ctx context.Context, scenarioSlug string) (string, error) {
	return r.ResolveScenarioURL(ctx, scenarioSlug, defaultPortKey)
}

// sharedResolver backs the package-level convenience wrappers. It must be a
// process-wide singleton: the Resolver's cache is a field, so constructing a
// Resolver per call — as these wrappers previously did — guaranteed a cache miss
// and forked `vrooli scenario port` on every single call. With 130+ callsites,
// several of them on request paths, that turned inbound HTTP volume directly
// into process-creation volume.
var (
	sharedResolverOnce sync.Once
	sharedResolver     *Resolver
)

// DefaultResolver returns the process-wide resolver used by the package-level
// convenience wrappers. Callers needing isolation (tests, alternate hosts,
// static base URLs) should construct their own via NewResolver.
func DefaultResolver() *Resolver {
	sharedResolverOnce.Do(func() {
		sharedResolver = NewResolver(ResolverConfig{})
	})
	return sharedResolver
}

// ResolveScenarioPort is a convenience wrapper using the shared resolver.
func ResolveScenarioPort(ctx context.Context, scenarioSlug, portKey string) (int, error) {
	return DefaultResolver().ResolveScenarioPort(ctx, scenarioSlug, portKey)
}

// ResolveScenarioURL is a convenience wrapper using the shared resolver.
func ResolveScenarioURL(ctx context.Context, scenarioSlug, portKey string) (string, error) {
	return DefaultResolver().ResolveScenarioURL(ctx, scenarioSlug, portKey)
}

// ResolveScenarioPortDefault is a convenience wrapper using the standard API port.
func ResolveScenarioPortDefault(ctx context.Context, scenarioSlug string) (int, error) {
	return DefaultResolver().ResolveScenarioPortDefault(ctx, scenarioSlug)
}

// ResolveScenarioURLDefault is a convenience wrapper using the standard API port.
func ResolveScenarioURLDefault(ctx context.Context, scenarioSlug string) (string, error) {
	return DefaultResolver().ResolveScenarioURLDefault(ctx, scenarioSlug)
}

func defaultRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

// extractPortFromStaticURL parses the port from the static base URL.
func (r *Resolver) extractPortFromStaticURL(scenarioSlug, portKey string) (int, error) {
	parsed, err := url.Parse(r.staticBaseURL)
	if err != nil {
		return 0, &Error{
			Kind:     ErrInvalidPort,
			Scenario: scenarioSlug,
			PortKey:  portKey,
			Output:   r.staticBaseURL,
			Err:      fmt.Errorf("parse static URL: %w", err),
		}
	}

	portStr := parsed.Port()
	if portStr == "" {
		// Use default ports for known schemes
		switch parsed.Scheme {
		case "http":
			return 80, nil
		case "https":
			return 443, nil
		default:
			return 0, &Error{
				Kind:     ErrInvalidPort,
				Scenario: scenarioSlug,
				PortKey:  portKey,
				Output:   r.staticBaseURL,
				Err:      errors.New("no port in static URL and unknown scheme"),
			}
		}
	}

	port, parseErr := strconv.Atoi(portStr)
	if parseErr != nil || port <= 0 {
		return 0, &Error{
			Kind:     ErrInvalidPort,
			Scenario: scenarioSlug,
			PortKey:  portKey,
			Output:   r.staticBaseURL,
			Err:      parseErr,
		}
	}

	return port, nil
}
