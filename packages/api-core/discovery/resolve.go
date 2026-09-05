// Package discovery provides target-aware runtime helpers for resolving scenario addresses.
// DOC: docs/concepts/REACH-AND-CONFIGURATION.md
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

	"github.com/vrooli/api-core/nodereach"
	"github.com/vrooli/cli-core/cliutil"
)

// CommandRunner executes a command and returns combined stdout/stderr.
type CommandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

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

	// RemoteBaseURL is the Bridge base URL used for target-aware resolutions.
	// Production normally discovers Bridge by scenario name; tests and embedded
	// callers may provide an explicit endpoint.
	RemoteBaseURL string

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
}

// Resolver resolves scenario addresses through the local discovery ladder.
// The ladder tries a lifecycle peer record, the runtime registry, and then the
// control-plane CLI fallback. Successful addresses are cached briefly and
// failed lookups expire quickly, so restarts are not pinned to stale addresses.
// A static base URL bypasses discovery entirely.
type Resolver struct {
	vrooliPath    string
	runner        CommandRunner
	host          string
	scheme        string
	staticBaseURL string // When set, bypasses CLI discovery
	remoteBaseURL string
	cacheTTL      time.Duration
	negativeTTL   time.Duration
	sharedLookup  bool
	now           func() time.Time
	cacheMu       sync.Mutex
	cache         map[string]*cachedPort
	cacheHits     int64
	cacheMisses   int64
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
	ErrInvalidInput       ErrorKind = "invalid_input"
	ErrVrooliNotFound     ErrorKind = "vrooli_not_found"
	ErrScenarioNotRunning ErrorKind = "scenario_not_running"
	ErrTimeout            ErrorKind = "timeout"
	ErrInvalidPort        ErrorKind = "invalid_port"
	ErrCommandFailed      ErrorKind = "command_failed"
)

// Error provides structured details about discovery failures.
type Error struct {
	Kind     ErrorKind
	Scenario string
	PortKey  string
	Command  string
	Node     string
	Output   string
	Err      error
}

func (e *Error) Error() string {
	parts := []string{
		"api-core discovery",
		string(e.Kind),
		fmt.Sprintf("scenario=%q", e.Scenario),
	}
	if e.Command != "" {
		parts = append(parts, fmt.Sprintf("command=%q", e.Command))
	} else {
		parts = append(parts, fmt.Sprintf("port=%q", e.PortKey))
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
		vrooliPath:    vrooliPath,
		runner:        runner,
		host:          host,
		scheme:        scheme,
		staticBaseURL: strings.TrimRight(cfg.StaticBaseURL, "/"),
		remoteBaseURL: strings.TrimRight(cfg.RemoteBaseURL, "/"),
		cacheTTL:      cacheTTL,
		negativeTTL:   negativeTTL,
		// Only an unconfigured resolver may use the shared seam; an injected
		// runner or binary path is an explicit request for this resolver's own
		// execution path.
		sharedLookup: cfg.CommandRunner == nil && cfg.VrooliPath == "",
		now:          now,
		cache:        make(map[string]*cachedPort),
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

// ResolveScenarioPort resolves a scenario's port through the local discovery
// ladder: lifecycle peer record, runtime registry, then the control-plane CLI.
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

// resolvePortCached collapses concurrent lookups for one key onto a single
// ladder evaluation and reuses successful and failed results for their TTLs.
// Holding the per-key lock across the lookup is deliberate: a burst of N
// callers for one scenario must cost one ladder evaluation, not N.
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
	port, derr := r.lookupPortViaLadder(ctx, scenarioSlug, target, portKey)
	if derr != nil && cliutil.IsNonLiveTarget(target) && derr.Kind == ErrScenarioNotRunning {
		cliutil.WarnShadowFallback(scenarioSlug)
		port, derr = r.lookupPortViaLadder(ctx, scenarioSlug, scenarioSlug, portKey)
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

// lookupPortViaLadder evaluates the local authorities and the control-plane
// fallback, then classifies the result. reportSlug is the user-facing scenario
// name recorded on any Error (which may differ from target for a variant).
// runLookup obtains one raw port reading. In production it routes through
// cliutil's process-wide cache, which is the single owner of
// `vrooli scenario port` — so a lookup performed for a CLI helper and one
// performed here cost one process between them rather than one each. The
// Resolver's own staleness tolerance travels with the request as a
// PortCachePolicy, so sharing the cache never lengthens the window in which
// this Resolver may hand back the address of a restarted scenario.
//
// A Resolver configured with an explicit CommandRunner or VrooliPath keeps its
// own path: that injection is the test seam, and honoring it is what lets a
// test drive this code without a real CLI on PATH.
//
// It returns the text to parse as a port, the full output for classification,
// and the execution error.
func (r *Resolver) runLookup(ctx context.Context, target, portKey string) (portText, output string, err error) {
	if r.sharedLookup {
		outcome := cliutil.LookupScenarioPort(ctx, target, portKey, cliutil.PortCachePolicy{
			MaxAge:         nonNegative(r.cacheTTL),
			NegativeMaxAge: nonNegative(r.negativeTTL),
		})
		return outcome.Port, outcome.Output, outcome.Err
	}
	raw, runErr := r.runner(ctx, r.vrooliPath, "scenario", "port", target, portKey)
	text := strings.TrimSpace(string(raw))
	return text, text, runErr
}

func nonNegative(d time.Duration) time.Duration {
	if d < 0 {
		return 0
	}
	return d
}

func (r *Resolver) lookupPortViaLadder(ctx context.Context, reportSlug, target, portKey string) (int, *Error) {
	portText, text, err := r.runLookup(ctx, target, portKey)
	if err != nil {
		// A deadline may come from the caller's context or from the bound the
		// shared lookup applies when the caller supplied none. Both are
		// timeouts; classifying the latter as a generic command failure would
		// send an operator hunting a CLI bug instead of a hung lookup.
		if ctxErr := ctx.Err(); ctxErr != nil || errors.Is(err, context.DeadlineExceeded) {
			if ctxErr == nil {
				ctxErr = err
			}
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

	port, parseErr := strconv.Atoi(portText)
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

// Target identifies the machine on which a scenario should be reached.
// An empty NodeID means the local machine.
type Target struct {
	NodeID string
}

type resolveOptions struct {
	target Target
}

// WithTarget selects the machine for a scenario URL resolution. Existing
// callers that omit the option retain local discovery behavior.
func WithTarget(target Target) ResolveOption {
	return func(options *resolveOptions) { options.target = target }
}

// ResolveOption customizes a target-aware scenario URL resolution.
type ResolveOption func(*resolveOptions)

// ResolveScenarioURL resolves a scenario's port and returns a URL using the
// resolver's scheme and host. WithTarget selects a registered Bridge node;
// local resolution remains the peer/registry/CLI ladder.
//
// If the resolver was created with a static base URL, that URL is returned
// directly without invoking the CLI.
func (r *Resolver) ResolveScenarioURL(ctx context.Context, scenarioSlug, portKey string) (string, error) {
	return r.resolveScenarioURL(ctx, scenarioSlug, portKey, Target{})
}

// ResolveScenarioURLWithOptions is the option-shaped companion for callers
// that need target selection without changing the long-standing resolver
// interface implemented by Bridge and other consumers.
func (r *Resolver) ResolveScenarioURLWithOptions(ctx context.Context, scenarioSlug, portKey string, options ...ResolveOption) (string, error) {
	resolved := resolveOptions{}
	for _, option := range options {
		if option != nil {
			option(&resolved)
		}
	}
	return r.resolveScenarioURL(ctx, scenarioSlug, portKey, resolved.target)
}

// ResolveScenarioURLForTarget resolves a scenario URL for an explicit target
// while preserving ResolveScenarioURL's established interface.
func (r *Resolver) ResolveScenarioURLForTarget(ctx context.Context, scenarioSlug, portKey string, target Target) (string, error) {
	return r.resolveScenarioURL(ctx, scenarioSlug, portKey, target)
}

func (r *Resolver) resolveScenarioURL(ctx context.Context, scenarioSlug, portKey string, target Target) (string, error) {
	resolved := resolveOptions{}
	resolved.target = target
	// Static mode: return the configured URL directly
	if r.staticBaseURL != "" && resolved.target.NodeID == "" {
		return r.staticBaseURL, nil
	}
	if resolved.target.NodeID != "" {
		return r.resolveRemoteScenarioURL(ctx, scenarioSlug, resolved.target.NodeID)
	}

	port, err := r.ResolveScenarioPort(ctx, scenarioSlug, portKey)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s://%s:%d", r.scheme, r.host, port), nil
}

func (r *Resolver) resolveRemoteScenarioURL(ctx context.Context, scenarioSlug, nodeID string) (string, error) {
	bridgeURL := r.remoteBaseURL
	if bridgeURL == "" {
		var err error
		bridgeURL, err = nodereach.New(nodereach.Config{}).ResolveURL(ctx)
		if err != nil {
			return "", err
		}
	}
	if strings.TrimSpace(scenarioSlug) == "" || strings.TrimSpace(nodeID) == "" {
		return "", &Error{Kind: ErrInvalidInput, Scenario: scenarioSlug, Err: errors.New("scenario and node target are required")}
	}
	return strings.TrimRight(bridgeURL, "/") + "/api/v1/targets/" + url.PathEscape(nodeID) + "/scenarios/" + url.PathEscape(scenarioSlug), nil
}

// ResolveScenarioPortDefault resolves the standard API port for a scenario.
func (r *Resolver) ResolveScenarioPortDefault(ctx context.Context, scenarioSlug string) (int, error) {
	return r.ResolveScenarioPort(ctx, scenarioSlug, defaultPortKey)
}

// ResolveScenarioURLDefault resolves the standard API URL for a scenario.
func (r *Resolver) ResolveScenarioURLDefault(ctx context.Context, scenarioSlug string) (string, error) {
	return r.ResolveScenarioURL(ctx, scenarioSlug, defaultPortKey)
}

// ResolveScenarioURLDefaultForTarget resolves the standard API URL for an
// explicit target while preserving the existing default method interface.
func (r *Resolver) ResolveScenarioURLDefaultForTarget(ctx context.Context, scenarioSlug string, target Target) (string, error) {
	return r.ResolveScenarioURLForTarget(ctx, scenarioSlug, defaultPortKey, target)
}

// ResolveScenarioURLDefaultWithOptions resolves the standard API URL with
// optional target selection.
func (r *Resolver) ResolveScenarioURLDefaultWithOptions(ctx context.Context, scenarioSlug string, options ...ResolveOption) (string, error) {
	return r.ResolveScenarioURLWithOptions(ctx, scenarioSlug, defaultPortKey, options...)
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

// ResolveScenarioURLDefaultForTarget is the convenience wrapper for an
// explicit target.
func ResolveScenarioURLDefaultForTarget(ctx context.Context, scenarioSlug string, target Target) (string, error) {
	return DefaultResolver().ResolveScenarioURLDefaultForTarget(ctx, scenarioSlug, target)
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
