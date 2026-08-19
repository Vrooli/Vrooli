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
	now            func() time.Time
	cacheMu        sync.Mutex
	cache          map[string]cachedPort
	cacheHits      int64
	cacheMisses    int64
}

type cachedPort struct {
	port      int
	expiresAt time.Time
}

const defaultPortKey = "API_PORT"

// defaultResolverCacheTTL amortizes fan-out resolution for one query while
// bounding the stale-address window after a scenario restart.
const defaultResolverCacheTTL = 2 * time.Second

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
		now:            now,
		cache:          make(map[string]cachedPort),
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
// Successful lookups are cached for ResolverConfig.CacheTTL. A failed lookup
// removes the cache entry before returning the structured discovery error.
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
	if port, ok := r.cached(cacheKey); ok {
		return port, nil
	}

	// Instance routing (Case B): when the target scenario is ambiently shadowed
	// (VROOLI_SHADOW_SCENARIOS), address its "@shadow" record. If that non-live
	// lookup reports the scenario isn't running — the engagement may have been
	// torn down — warn once and fall back to the live instance. Never silent.
	target := cliutil.ResolveShadowTarget(scenarioSlug)
	port, derr := r.lookupPort(ctx, scenarioSlug, target, portKey)
	if derr != nil && cliutil.IsNonLiveTarget(target) && derr.Kind == ErrScenarioNotRunning {
		cliutil.WarnShadowFallback(scenarioSlug)
		port, derr = r.lookupPort(ctx, scenarioSlug, scenarioSlug, portKey)
	}
	if derr != nil {
		r.invalidate(cacheKey)
		return 0, derr
	}
	r.store(cacheKey, port)
	return port, nil
}

func (r *Resolver) cached(key string) (int, bool) {
	if r.cacheTTL < 0 {
		return 0, false
	}
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()
	entry, ok := r.cache[key]
	if !ok {
		r.cacheMisses++
		return 0, false
	}
	if !r.now().Before(entry.expiresAt) {
		delete(r.cache, key)
		r.cacheMisses++
		return 0, false
	}
	r.cacheHits++
	return entry.port, true
}

// CacheStats returns cumulative successful cache lookups and misses. The
// counters are resolver-local: callers can take a before/after sample around
// one operation without exposing cache contents or corpus data.
func (r *Resolver) CacheStats() (hits, misses int64) {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()
	return r.cacheHits, r.cacheMisses
}

func (r *Resolver) store(key string, port int) {
	if r.cacheTTL < 0 {
		return
	}
	r.cacheMu.Lock()
	r.cache[key] = cachedPort{port: port, expiresAt: r.now().Add(r.cacheTTL)}
	r.cacheMu.Unlock()
}

func (r *Resolver) invalidate(key string) {
	r.cacheMu.Lock()
	delete(r.cache, key)
	r.cacheMu.Unlock()
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

// ResolveScenarioPort is a convenience wrapper using default config.
func ResolveScenarioPort(ctx context.Context, scenarioSlug, portKey string) (int, error) {
	return NewResolver(ResolverConfig{}).ResolveScenarioPort(ctx, scenarioSlug, portKey)
}

// ResolveScenarioURL is a convenience wrapper using default config.
func ResolveScenarioURL(ctx context.Context, scenarioSlug, portKey string) (string, error) {
	return NewResolver(ResolverConfig{}).ResolveScenarioURL(ctx, scenarioSlug, portKey)
}

// ResolveScenarioPortDefault is a convenience wrapper using the standard API port.
func ResolveScenarioPortDefault(ctx context.Context, scenarioSlug string) (int, error) {
	return NewResolver(ResolverConfig{}).ResolveScenarioPortDefault(ctx, scenarioSlug)
}

// ResolveScenarioURLDefault is a convenience wrapper using the standard API port.
func ResolveScenarioURLDefault(ctx context.Context, scenarioSlug string) (string, error) {
	return NewResolver(ResolverConfig{}).ResolveScenarioURLDefault(ctx, scenarioSlug)
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
