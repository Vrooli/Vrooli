// Package discovery provides target-aware runtime helpers for resolving scenario addresses.
// DOC: docs/concepts/REACH-AND-CONFIGURATION.md
package discovery

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vrooli/api-core/demand"
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

	// NegativeCacheTTL bounds reuse of a confirmed stopped-target result. Other
	// failures are transient or caller-specific and are never cached. Zero uses
	// the default; set it negative to retry stopped targets on every lookup.
	NegativeCacheTTL time.Duration

	// Now supplies time for deterministic cache tests. Nil uses time.Now.
	Now func() time.Time
}

// DemandOptions opts one URL resolution into a bounded demand lease. The
// default resolver remains read-only; callers must explicitly provide a
// client and consumer identity so background probes cannot accidentally keep
// scenarios alive.
type DemandOptions struct {
	Client demand.LeaseClient
	// Starter enables opt-in startup when the target is stopped. It remains
	// separate from Client so lease-only callers stay read-only.
	Starter    demand.ScenarioStarter
	AutoStart  bool
	ConsumerID string
	Kind       string
	RequestID  string
	Variant    string
	Metadata   string
	TTL        time.Duration
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
	staticBaseErr error
	remoteBaseURL string
	cacheTTL      time.Duration
	negativeTTL   time.Duration
	sharedLookup  bool
	customClock   bool
	now           func() time.Time
	cacheMu       sync.Mutex
	cache         map[string]*cachedPort
}

type resolverCacheObservationKey struct{}
type resolverCacheSampleKey struct{}

type resolverCacheSample struct{ once sync.Once }

// ResolverCacheObservation counts one cache decision per API Resolver
// operation performed with a context returned by ObserveResolverCache. The
// default production Resolver uses cliutil's shared cache; injected resolvers
// report their private cache. Counters are safe for concurrent fan-out and
// isolated from unrelated requests.
type ResolverCacheObservation struct {
	hits   atomic.Int64
	misses atomic.Int64
}

// ObserveResolverCache attaches query-scoped cache accounting to ctx. On the
// default production path, a miss means local discovery authorities will be
// evaluated; it does not imply that a CLI process will be launched.
func ObserveResolverCache(ctx context.Context) (context.Context, *ResolverCacheObservation) {
	observation := &ResolverCacheObservation{}
	return context.WithValue(ctx, resolverCacheObservationKey{}, observation), observation
}

// Snapshot returns the counts recorded so far for this observation.
func (o *ResolverCacheObservation) Snapshot() (hits, misses int64) {
	if o == nil {
		return 0, 0
	}
	return o.hits.Load(), o.misses.Load()
}

func observeResolverCache(ctx context.Context, hit bool) {
	observation, _ := ctx.Value(resolverCacheObservationKey{}).(*ResolverCacheObservation)
	if observation == nil {
		return
	}
	if hit {
		observation.hits.Add(1)
	} else {
		observation.misses.Add(1)
	}
}

func beginResolverCacheSample(ctx context.Context) context.Context {
	if _, ok := ctx.Value(resolverCacheObservationKey{}).(*ResolverCacheObservation); !ok {
		return ctx
	}
	return context.WithValue(ctx, resolverCacheSampleKey{}, &resolverCacheSample{})
}

// cachedPort holds one key's resolution and serializes lookups without
// preventing waiters from honoring their own contexts. err is non-nil for a
// cached negative result.
type cachedPort struct {
	lookupGate chan struct{}
	port       int
	err        *Error
	resolvedAt time.Time
}

const defaultPortKey = "API_PORT"

// defaultResolverCacheTTL amortizes fan-out resolution for one query while
// bounding the stale-address window after a scenario restart.
const defaultResolverCacheTTL = 2 * time.Second

// defaultResolverNegativeCacheTTL suppresses repeated recovery against a
// confirmed stopped scenario while keeping the retry window short.
const defaultResolverNegativeCacheTTL = 500 * time.Millisecond

// ErrorKind identifies the class of discovery failure.
type ErrorKind string

const (
	ErrInvalidInput       ErrorKind = "invalid_input"
	ErrVrooliNotFound     ErrorKind = "vrooli_not_found"
	ErrScenarioNotRunning ErrorKind = "scenario_not_running"
	ErrTimeout            ErrorKind = "timeout"
	ErrInvalidPort        ErrorKind = "invalid_port"
	ErrInvalidBaseURL     ErrorKind = "invalid_base_url"
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
	staticBaseURL := strings.TrimSpace(cfg.StaticBaseURL)
	var staticBaseErr error
	if staticBaseURL != "" {
		staticBaseURL, staticBaseErr = cliutil.NormalizeScenarioAPIBaseURL(staticBaseURL)
		if staticBaseErr != nil {
			staticBaseURL = strings.TrimSpace(cfg.StaticBaseURL)
		}
	}
	return &Resolver{
		vrooliPath:    vrooliPath,
		runner:        runner,
		host:          host,
		scheme:        scheme,
		staticBaseURL: staticBaseURL,
		staticBaseErr: staticBaseErr,
		remoteBaseURL: strings.TrimRight(cfg.RemoteBaseURL, "/"),
		cacheTTL:      cacheTTL,
		negativeTTL:   negativeTTL,
		// Only an unconfigured resolver may use the shared seam; an injected
		// runner or binary path is an explicit request for this resolver's own
		// execution path.
		sharedLookup: cfg.CommandRunner == nil && cfg.VrooliPath == "",
		customClock:  cfg.Now != nil,
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

// NewResolverWithOptionalStaticOverride uses a configured base URL for default
// local resolution when it is non-empty, then falls back to the process-wide
// local discovery resolver. Bind the result to one logical scenario target.
// Explicit non-local Bridge targets remain Bridge-routed and bypass this static
// URL override.
func NewResolverWithOptionalStaticOverride(baseURL string) *Resolver {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL != "" {
		return NewStaticResolver(baseURL)
	}
	return DefaultResolver()
}

// ResolveScenarioPort resolves a scenario's port through the local discovery
// ladder: lifecycle peer record, runtime registry, then the control-plane CLI.
//
// Successful lookups are cached for ResolverConfig.CacheTTL. Confirmed
// stopped-target results use the shorter ResolverConfig.NegativeCacheTTL;
// invalid input, transient command/runtime failures, and context timeouts are
// never cached. Concurrent callers for one key collapse onto a single
// invocation.
//
// If the resolver was created with a static base URL, the port is extracted
// from that URL instead of invoking the CLI.
func (r *Resolver) ResolveScenarioPort(ctx context.Context, scenarioSlug, portKey string) (int, error) {
	// Static mode: extract port from the configured URL
	if r.staticBaseURL != "" {
		if r.staticBaseErr != nil {
			return 0, r.invalidStaticBaseError(scenarioSlug, portKey)
		}
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
	if r.sharedLookup && !r.customClock {
		ctx = beginResolverCacheSample(ctx)
		port, derr := r.lookupPortWithFallback(ctx, scenarioSlug, portKey)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return 0, &Error{Kind: ErrTimeout, Scenario: scenarioSlug, PortKey: portKey, Err: ctxErr}
		}
		if derr != nil {
			return 0, derr
		}
		return port, nil
	}
	cacheKey := scenarioSlug + "\x00" + portKey
	return r.resolvePortCached(ctx, scenarioSlug, portKey, cacheKey)
}

// resolvePortCached collapses concurrent lookups for one key onto a single
// ladder evaluation and reuses successful results and confirmed stopped-target
// results for their respective TTLs. The per-key gate preserves that sharing
// while allowing each waiter to honor its own context.
func (r *Resolver) resolvePortCached(ctx context.Context, scenarioSlug, portKey, cacheKey string) (int, error) {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return 0, &Error{Kind: ErrTimeout, Scenario: scenarioSlug, PortKey: portKey, Err: ctxErr}
	}
	if r.cacheTTL < 0 {
		observeResolverCache(ctx, false)
		port, derr := r.lookupPortWithFallback(ctx, scenarioSlug, portKey)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return 0, &Error{Kind: ErrTimeout, Scenario: scenarioSlug, PortKey: portKey, Err: ctxErr}
		}
		if derr != nil {
			return 0, derr
		}
		return port, nil
	}

	entry := r.entryFor(cacheKey)
	select {
	case entry.lookupGate <- struct{}{}:
		defer func() { <-entry.lookupGate }()
	case <-ctx.Done():
		return 0, &Error{Kind: ErrTimeout, Scenario: scenarioSlug, PortKey: portKey, Err: ctx.Err()}
	}
	if err := ctx.Err(); err != nil {
		return 0, &Error{Kind: ErrTimeout, Scenario: scenarioSlug, PortKey: portKey, Err: err}
	}

	if !entry.resolvedAt.IsZero() {
		ttl := r.cacheTTL
		if entry.err != nil {
			ttl = r.negativeTTL
		}
		if r.now().Sub(entry.resolvedAt) < ttl {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return 0, &Error{Kind: ErrTimeout, Scenario: scenarioSlug, PortKey: portKey, Err: ctxErr}
			}
			observeResolverCache(ctx, true)
			if entry.err != nil {
				return 0, entry.err
			}
			return entry.port, nil
		}
	}
	observeResolverCache(ctx, false)

	port, derr := r.lookupPortWithFallback(ctx, scenarioSlug, portKey)
	if ctxErr := ctx.Err(); ctxErr != nil {
		entry.port, entry.err, entry.resolvedAt = 0, nil, time.Time{}
		return 0, &Error{Kind: ErrTimeout, Scenario: scenarioSlug, PortKey: portKey, Err: ctxErr}
	}

	if derr != nil {
		entry.port, entry.err, entry.resolvedAt = 0, nil, time.Time{}
		if derr.Kind == ErrScenarioNotRunning && r.negativeTTL > 0 {
			entry.err, entry.resolvedAt = derr, r.now()
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			entry.port, entry.err, entry.resolvedAt = 0, nil, time.Time{}
			return 0, &Error{Kind: ErrTimeout, Scenario: scenarioSlug, PortKey: portKey, Err: ctxErr}
		}
		return 0, derr
	}

	entry.port, entry.err, entry.resolvedAt = port, nil, r.now()
	if ctxErr := ctx.Err(); ctxErr != nil {
		entry.port, entry.err, entry.resolvedAt = 0, nil, time.Time{}
		return 0, &Error{Kind: ErrTimeout, Scenario: scenarioSlug, PortKey: portKey, Err: ctxErr}
	}
	return port, nil
}

// lookupPortWithFallback performs instance routing (Case B): when the target
// scenario is ambiently shadowed (VROOLI_SHADOW_SCENARIOS), address its
// "@shadow" record. Ambient shadow selection may fall back on a typed
// not-running result; explicit selections and same-variant dependencies stay
// pinned. Transient failures never fall back, and a live fallback is warned.
func (r *Resolver) lookupPortWithFallback(ctx context.Context, scenarioSlug, portKey string) (int, *Error) {
	target := cliutil.ResolveShadowTarget(scenarioSlug)
	port, derr := r.lookupPortViaLadder(ctx, scenarioSlug, target, portKey)
	if derr == nil {
		return port, derr
	}
	if !cliutil.ShouldAutomaticallyFallbackToLiveInstance(scenarioSlug, target, derr.Kind == ErrScenarioNotRunning) {
		if derr.Kind == ErrScenarioNotRunning && cliutil.IsNonLiveTarget(target) && cliutil.IsVariantDependency(scenarioSlug) {
			derr.Output = fmt.Sprintf("%s is resolved at this instance's own variant %q and is not running; start it or remove it from %s (no live fallback)",
				cliutil.BareScenarioName(scenarioSlug), cliutil.OwnVariant(), cliutil.EnvVariantDependencies)
		}
		return 0, derr
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return 0, &Error{Kind: ErrTimeout, Scenario: scenarioSlug, PortKey: portKey, Err: ctxErr}
	}
	cliutil.WarnShadowFallback(scenarioSlug)
	liveTarget := cliutil.BareScenarioName(scenarioSlug)
	return r.lookupPortViaLadder(ctx, liveTarget, liveTarget, portKey)
}

func (r *Resolver) entryFor(key string) *cachedPort {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()
	entry, ok := r.cache[key]
	if !ok {
		entry = &cachedPort{lookupGate: make(chan struct{}, 1)}
		r.cache[key] = entry
	}
	return entry
}

func (r *Resolver) invalidateScenarioCache(ctx context.Context, scenarioSlug, portKey string) error {
	if r.cacheTTL < 0 {
		return nil
	}
	if r.sharedLookup {
		target := cliutil.ResolveShadowTarget(scenarioSlug)
		targets := []string{target}
		if cliutil.ShouldAutomaticallyFallbackToLiveInstance(scenarioSlug, target, true) {
			targets = append(targets, cliutil.BareScenarioName(scenarioSlug))
		}
		for _, cacheTarget := range targets {
			if err := cliutil.InvalidateScenarioPortCache(ctx, cacheTarget, portKey); err != nil {
				return &Error{Kind: ErrTimeout, Scenario: cacheTarget, PortKey: portKey, Err: err}
			}
		}
		if !r.customClock {
			return nil
		}
	}
	entry := r.entryFor(scenarioSlug + "\x00" + portKey)
	select {
	case entry.lookupGate <- struct{}{}:
	case <-ctx.Done():
		return &Error{Kind: ErrTimeout, Scenario: scenarioSlug, PortKey: portKey, Err: ctx.Err()}
	}
	defer func() { <-entry.lookupGate }()
	if ctxErr := ctx.Err(); ctxErr != nil {
		return &Error{Kind: ErrTimeout, Scenario: scenarioSlug, PortKey: portKey, Err: ctxErr}
	}
	entry.resolvedAt = time.Time{}
	entry.err = nil
	return nil
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
		policy := cliutil.PortCachePolicy{
			MaxAge:         nonNegative(r.cacheTTL),
			NegativeMaxAge: nonNegative(r.negativeTTL),
		}
		if sample, _ := ctx.Value(resolverCacheSampleKey{}).(*resolverCacheSample); sample != nil {
			policy.OnCacheLookup = func(hit bool) {
				sample.once.Do(func() { observeResolverCache(ctx, hit) })
			}
		}
		outcome := cliutil.LookupScenarioPort(ctx, target, portKey, policy)
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
	if ctxErr := ctx.Err(); ctxErr != nil {
		return 0, &Error{Kind: ErrTimeout, Scenario: reportSlug, PortKey: portKey, Err: ctxErr}
	}
	portText, text, err := r.runLookup(ctx, target, portKey)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return 0, &Error{Kind: ErrTimeout, Scenario: reportSlug, PortKey: portKey, Output: text, Err: ctxErr}
	}
	if cliutil.IsScenarioPortNotRunning(err) || cliPortErrorCode(text) == cliutil.ScenarioPortCodeNotRunning {
		return 0, &Error{Kind: ErrScenarioNotRunning, Scenario: reportSlug, PortKey: portKey, Output: text, Err: err}
	}
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

func cliPortErrorCode(output string) string {
	var payload struct {
		ErrorCode string `json:"error_code"`
	}
	if json.Unmarshal([]byte(output), &payload) != nil {
		return ""
	}
	return payload.ErrorCode
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
		if r.staticBaseErr != nil {
			return "", r.invalidStaticBaseError(scenarioSlug, portKey)
		}
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

func (r *Resolver) invalidStaticBaseError(scenarioSlug, portKey string) *Error {
	return &Error{
		Kind:     ErrInvalidBaseURL,
		Scenario: scenarioSlug,
		PortKey:  portKey,
		Output:   r.staticBaseURL,
		Err:      r.staticBaseErr,
	}
}

func (r *Resolver) resolveRemoteScenarioURL(ctx context.Context, scenarioSlug, nodeID string) (string, error) {
	if strings.TrimSpace(scenarioSlug) == "" || strings.TrimSpace(nodeID) == "" {
		return "", &Error{Kind: ErrInvalidInput, Scenario: scenarioSlug, Err: errors.New("scenario and node target are required")}
	}
	return nodereach.New(nodereach.Config{BridgeURL: r.remoteBaseURL}).ScenarioURL(ctx, nodeID, scenarioSlug)
}

// ResolveScenarioPortDefault resolves the standard API port for a scenario.
func (r *Resolver) ResolveScenarioPortDefault(ctx context.Context, scenarioSlug string) (int, error) {
	return r.ResolveScenarioPort(ctx, scenarioSlug, defaultPortKey)
}

// ResolveScenarioURLDefault resolves the standard API URL for a scenario.
func (r *Resolver) ResolveScenarioURLDefault(ctx context.Context, scenarioSlug string) (string, error) {
	return r.ResolveScenarioURL(ctx, scenarioSlug, defaultPortKey)
}

// ResolveScenarioURLWithDemand resolves an address and acquires a lease for
// the target. With AutoStart and Starter, a stopped target is started under
// demand supervision after the lease is acquired, then resolved again. The
// returned lease must be released by the caller when the scope ends.
func (r *Resolver) ResolveScenarioURLWithDemand(ctx context.Context, scenarioSlug string, options DemandOptions) (string, demand.Lease, error) {
	if options.Client == nil {
		return "", demand.Lease{}, errors.New("demand client is required")
	}
	if strings.TrimSpace(options.ConsumerID) == "" {
		return "", demand.Lease{}, errors.New("demand consumer id is required")
	}
	if strings.TrimSpace(options.Kind) == "" {
		options.Kind = demand.KindDependency
	}
	lookupSlug := scenarioSlug
	if variant := strings.TrimSpace(options.Variant); variant != "" && variant != "live" {
		lookupSlug += "@" + variant
	}
	url, err := r.ResolveScenarioURLDefault(ctx, lookupSlug)
	if err != nil && (!options.AutoStart || options.Starter == nil || !IsScenarioNotRunning(err)) {
		return "", demand.Lease{}, err
	}
	if strings.TrimSpace(options.RequestID) == "" {
		// Independent invocations must never share a releasable hold. A
		// caller requiring retry identity supplies its own RequestID.
		options.RequestID = "resolve:" + rand.Text()
	}
	lease, err := options.Client.Acquire(ctx, demand.AcquireRequest{
		LeaseID:    demand.StableVariantLeaseID(options.ConsumerID, scenarioSlug, options.Variant, options.RequestID),
		Scenario:   scenarioSlug,
		Variant:    options.Variant,
		ConsumerID: options.ConsumerID,
		Kind:       options.Kind,
		RequestID:  options.RequestID,
		Metadata:   options.Metadata,
		TTL:        options.TTL,
	})
	if err != nil {
		return "", demand.Lease{}, fmt.Errorf("acquire demand lease for %s: %w", scenarioSlug, err)
	}
	if url == "" {
		if options.Starter == nil {
			releaseCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_, _ = options.Client.Release(releaseCtx, lease.LeaseID, "empty demand resolution")
			return "", demand.Lease{}, errors.New("scenario starter is required for empty resolution")
		}
		if err := options.Starter.StartScenario(ctx, scenarioSlug, options.Variant); err != nil {
			releaseCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_, _ = options.Client.Release(releaseCtx, lease.LeaseID, "demand startup failed")
			return "", demand.Lease{}, err
		}
		if err := r.invalidateScenarioCache(ctx, lookupSlug, defaultPortKey); err != nil {
			releaseCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_, _ = options.Client.Release(releaseCtx, lease.LeaseID, "demand startup invalidation failed")
			return "", demand.Lease{}, err
		}
		url, err = r.ResolveScenarioURLDefault(ctx, lookupSlug)
		if err != nil {
			releaseCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_, _ = options.Client.Release(releaseCtx, lease.LeaseID, "demand startup unresolved")
			return "", demand.Lease{}, err
		}
	}
	return url, lease, nil
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
