package cliutil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/repo-contract-go/cliinvoke"
)

var (
	lookPathFn              = exec.LookPath
	userHomeFn              = os.UserHomeDir
	runVrooliFn             = cliinvoke.Run
	sqliteLookPathFn        = exec.LookPath
	sqliteExecCommandFn     = exec.CommandContext
	peerRecordLookupFn      = lookupPeerRecord
	peerOwnerPIDAliveFn     = isPIDRunning
	runtimeRegistryLookupFn = lookupRuntimeRegistry
	desktopPortLookupFn     = lookupDesktopPort
)

// DOC: docs/reference/port-resolution.md
// Port lookup uses the lifecycle peer record and runtime registry before
// falling back to the vrooli CLI. The shared cache prevents repeated work for
// long-lived callers while retaining a short retry window for confirmed
// stopped targets. Transient lookup failures are never cached.
//
// This cache is the process-wide owner of the fallback command. api-core's
// discovery.Resolver uses the same ladder and cache, so local authorities are
// tried before either caller pays for a CLI process.
var (
	portCacheMu sync.Mutex
	portCache   = map[string]*portCacheEntry{}

	portCacheNow = time.Now
)

// portLookupTimeout bounds a lookup whose caller supplied no deadline.
const portLookupTimeout = 5 * time.Second

// Default staleness tolerances for callers that express no preference. Keep
// successful addresses fresh across lifecycle restarts for every direct
// cliutil consumer; callers that need another tolerance must opt into it.
// Confirmed stopped targets use a short retry window; other failures are
// retried immediately.
const (
	defaultPortMaxAge         = 2 * time.Second
	defaultPortNegativeMaxAge = 500 * time.Millisecond
)

// PortCachePolicy states how stale a cached lookup a caller is willing to
// accept. Freshness is a property of the caller. The default is two seconds to
// match the shared Go and Node scenario resolvers; callers may opt into a
// different bound when their contract requires it.
type PortCachePolicy struct {
	// MaxAge bounds reuse of a successful lookup. Zero disables reuse.
	MaxAge time.Duration
	// NegativeMaxAge bounds reuse of a confirmed stopped-target lookup. Other
	// failures are never cached. Zero disables negative reuse.
	NegativeMaxAge time.Duration
	// OnCacheLookup observes this call's shared-cache decision. It receives true
	// for a reusable entry and false when the lookup must evaluate authorities.
	// It is called synchronously after the per-key gate and at most once.
	OnCacheLookup func(hit bool)
}

// DefaultPortCachePolicy is used by the CLI-facing detectors.
var DefaultPortCachePolicy = PortCachePolicy{
	MaxAge:         defaultPortMaxAge,
	NegativeMaxAge: defaultPortNegativeMaxAge,
}

// ScenarioPortOutcome is one classified result of `vrooli scenario port`.
// Output is retained so callers that need to distinguish "not running" from
// "no such scenario" can classify it without re-running the command.
type ScenarioPortOutcome struct {
	Port   string
	Output string
	Err    error
}

const ScenarioPortCodeNotRunning = "scenario_not_running"

// ScenarioPortError carries the CLI's machine-readable port failure class.
type ScenarioPortError struct {
	Code    string
	Message string
}

func (e *ScenarioPortError) Error() string {
	if e == nil {
		return "scenario port lookup failed"
	}
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

func IsScenarioPortNotRunning(err error) bool {
	var target *ScenarioPortError
	return errors.As(err, &target) && target.Code == ScenarioPortCodeNotRunning
}

// Resolved reports whether the lookup produced a usable port.
func (o ScenarioPortOutcome) Resolved() bool {
	return o.Err == nil && o.Port != ""
}

type portCacheEntry struct {
	lookupGate chan struct{}
	outcome    ScenarioPortOutcome
	resolved   time.Time
}

// resetPortDetectorCache drops every memoized lookup. Tests use it so a
// replaced runVrooliFn is actually consulted.
func resetPortDetectorCache() {
	portCacheMu.Lock()
	defer portCacheMu.Unlock()
	portCache = map[string]*portCacheEntry{}
}

// SetPortCacheNowForTest overrides the cache clock so tests can advance time
// without sleeping. Passing nil restores time.Now. Production must not call it.
func SetPortCacheNowForTest(now func() time.Time) {
	portCacheMu.Lock()
	defer portCacheMu.Unlock()
	if now == nil {
		now = time.Now
	}
	portCacheNow = now
}

func portCacheEntryFor(key string) *portCacheEntry {
	portCacheMu.Lock()
	defer portCacheMu.Unlock()
	entry, ok := portCache[key]
	if !ok {
		entry = &portCacheEntry{lookupGate: make(chan struct{}, 1)}
		portCache[key] = entry
	}
	return entry
}

// LookupScenarioPort resolves <target>'s <portVar> through three rungs: the
// lifecycle peer record, the runtime registry, and finally `vrooli scenario
// port`. Each rung is attempted only when the preceding rung misses. One
// caller deadline (or the default five-second bound when absent) covers the
// registry and CLI rungs, including a wait for another caller's in-flight
// evaluation. Results are cached per policy window and concurrent callers for
// one key share one evaluation.
func LookupScenarioPort(ctx context.Context, target, portVar string, policy PortCachePolicy) ScenarioPortOutcome {
	lookupCtx, cancel := boundedLookupContext(ctx)
	defer cancel()
	if err := lookupCtx.Err(); err != nil {
		return ScenarioPortOutcome{Err: err}
	}

	entry := portCacheEntryFor(target + "\x00" + portVar)
	select {
	case entry.lookupGate <- struct{}{}:
		defer func() { <-entry.lookupGate }()
		if err := lookupCtx.Err(); err != nil {
			return ScenarioPortOutcome{Err: err}
		}
	case <-lookupCtx.Done():
		return ScenarioPortOutcome{Err: lookupCtx.Err()}
	}

	if !entry.resolved.IsZero() {
		maxAge := policy.MaxAge
		if !entry.outcome.Resolved() {
			maxAge = policy.NegativeMaxAge
		}
		if maxAge > 0 && portCacheNow().Sub(entry.resolved) < maxAge {
			if policy.OnCacheLookup != nil {
				policy.OnCacheLookup(true)
			}
			return outcomeAfterLookupContext(entry.outcome, lookupCtx)
		}
	}
	if err := lookupCtx.Err(); err != nil {
		return ScenarioPortOutcome{Err: err}
	}
	if policy.OnCacheLookup != nil {
		policy.OnCacheLookup(false)
	}
	if os.Getenv("VROOLI_DESKTOP_MODE") == "true" {
		outcome := desktopPortLookupFn(lookupCtx, target, portVar)
		outcome = outcomeAfterLookupContext(outcome, lookupCtx)
		if lookupCtx.Err() == nil && outcome.Resolved() {
			entry.outcome, entry.resolved = outcome, portCacheNow()
		}
		if err := lookupCtx.Err(); err != nil {
			entry.outcome, entry.resolved = ScenarioPortOutcome{}, time.Time{}
			return outcomeAfterLookupContext(outcome, lookupCtx)
		}
		return outcome
	}

	outcome := peerRecordLookupFn(target, portVar)
	if !outcome.Resolved() {
		if err := lookupCtx.Err(); err != nil {
			return outcomeAfterLookupContext(outcome, lookupCtx)
		}
	}
	if !outcome.Resolved() {
		outcome = runtimeRegistryLookupFn(lookupCtx, target, portVar)
	}
	if !outcome.Resolved() && lookupCtx.Err() == nil {
		outcome = portLookupRunner(lookupCtx, target, portVar)
	}
	outcome = outcomeAfterLookupContext(outcome, lookupCtx)

	// A cancelled or expired context describes the caller's deadline, not the
	// target's state. Caching it would let one caller's cancellation deny an
	// unrelated caller for the whole negative window.
	if lookupCtx.Err() != nil {
		return outcome
	}
	if outcome.Resolved() || scenarioNotRunningOutcome(outcome) {
		entry.outcome, entry.resolved = outcome, portCacheNow()
	} else {
		entry.resolved = time.Time{}
	}
	if err := lookupCtx.Err(); err != nil {
		entry.outcome, entry.resolved = ScenarioPortOutcome{}, time.Time{}
		return outcomeAfterLookupContext(outcome, lookupCtx)
	}
	return outcome
}

// InvalidateScenarioPortCache clears one target/port outcome after acquiring
// the same per-key gate used by lookups. The caller deadline also bounds time
// waiting for an in-flight lookup, so a startup recovery cannot hang here.
func InvalidateScenarioPortCache(ctx context.Context, target, portVar string) error {
	lookupCtx, cancel := boundedLookupContext(ctx)
	defer cancel()
	if err := lookupCtx.Err(); err != nil {
		return err
	}
	entry := portCacheEntryFor(target + "\x00" + portVar)
	select {
	case entry.lookupGate <- struct{}{}:
		defer func() { <-entry.lookupGate }()
	case <-lookupCtx.Done():
		return lookupCtx.Err()
	}
	if err := lookupCtx.Err(); err != nil {
		return err
	}
	entry.outcome = ScenarioPortOutcome{}
	entry.resolved = time.Time{}
	return nil
}

func outcomeAfterLookupContext(outcome ScenarioPortOutcome, ctx context.Context) ScenarioPortOutcome {
	if err := ctx.Err(); err != nil {
		outcome.Port = ""
		outcome.Err = err
	}
	return outcome
}

func scenarioNotRunningOutcome(outcome ScenarioPortOutcome) bool {
	return IsScenarioPortNotRunning(outcome.Err)
}

// lookupRuntimeRegistry is the second local authority after peer records.
// The CLI package intentionally keeps this fallback behind the sqlite3 CLI so
// it does not introduce a CGO or driver dependency into every CLI consumer;
// the lifecycle-owned registry remains read-only and the subprocess is only
// reached after the cheap peer-record miss.
func lookupRuntimeRegistry(ctx context.Context, target, portVar string) ScenarioPortOutcome {
	dbPath, err := runtimeHomeEntryPath(repocontract.HomeKeyRuntimeDB)
	if err != nil {
		return ScenarioPortOutcome{Err: err}
	}
	if _, err := os.Stat(dbPath); err != nil {
		return ScenarioPortOutcome{Err: err}
	}
	scenario, variant := target, "live"
	if base, requested, ok := strings.Cut(target, "@"); ok {
		scenario, variant = base, requested
	}
	quote := func(value string) string { return "'" + strings.ReplaceAll(value, "'", "''") + "'" }
	query := fmt.Sprintf("SELECT rpc.port FROM runtime_instances ri JOIN runtime_port_claims rpc ON rpc.instance_id = ri.instance_id WHERE ri.scenario = %s AND ri.variant = %s AND ri.status = 'running' AND rpc.port_name = %s AND rpc.status = 'bound' ORDER BY ri.generation DESC LIMIT 1;", quote(scenario), quote(variant), quote(normalizePortClaimKey(portVar)))
	sqlite, err := sqliteLookPathFn("sqlite3")
	if err != nil || strings.TrimSpace(sqlite) == "" {
		return ScenarioPortOutcome{Err: err}
	}
	cmd := sqliteExecCommandFn(ctx, sqlite, "-readonly", "-noheader", dbPath, query)
	output, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(output))
	if err != nil {
		return ScenarioPortOutcome{Output: text, Err: err}
	}
	port := sanitizePortOutput(text)
	if port == "" {
		return ScenarioPortOutcome{Output: text, Err: os.ErrNotExist}
	}
	return ScenarioPortOutcome{Port: port, Output: text}
}

// lookupPeerRecord is the cheap, lifecycle-published address source. A record
// is accepted only while its owner is alive; stale files therefore cannot pin a
// caller to a dead process or a reused port.
func lookupPeerRecord(target, portVar string) ScenarioPortOutcome {
	requestedScenario, requestedVariant := SplitInstance(target)
	if requestedScenario == "" || strings.ContainsAny(requestedScenario, `/\\`) || strings.ContainsAny(requestedVariant, `/\\`) {
		return ScenarioPortOutcome{Err: fmt.Errorf("invalid peer record target")}
	}
	requestedVariant = normalizeVariant(requestedVariant)
	target = InstanceSlug(requestedScenario, requestedVariant)
	root, err := runtimeHomePath()
	if err != nil {
		return ScenarioPortOutcome{Err: err}
	}
	payload, err := os.ReadFile(filepath.Join(root, "peers", target+".json"))
	if err != nil {
		return ScenarioPortOutcome{Err: err}
	}
	var record struct {
		SchemaVersion int            `json:"schema_version"`
		Scenario      string         `json:"scenario"`
		Instance      string         `json:"instance"`
		Version       string         `json:"version"`
		Provides      []string       `json:"provides"`
		OwnerPID      int            `json:"owner_pid"`
		Ports         map[string]int `json:"ports"`
	}
	if err := json.Unmarshal(payload, &record); err != nil {
		return ScenarioPortOutcome{Err: err}
	}
	recordVariant := normalizeVariant(record.Instance) // Empty is the pre-variant live record shape.
	if record.SchemaVersion != 1 || record.Scenario != requestedScenario || recordVariant != requestedVariant || !peerOwnerPIDAliveFn(record.OwnerPID) {
		return ScenarioPortOutcome{Err: os.ErrNotExist}
	}
	port, ok := record.Ports[normalizePortClaimKey(portVar)]
	if !ok || port < 1 || port > 65535 {
		return ScenarioPortOutcome{Err: os.ErrNotExist}
	}
	if record.Provides != nil && !slices.Contains(record.Provides, normalizePortClaimKey(portVar)) {
		return ScenarioPortOutcome{Err: os.ErrNotExist}
	}
	return ScenarioPortOutcome{Port: strconv.Itoa(port)}
}

// runtimeHomePath resolves the same runtime-home authority for peer records
// and the runtime registry. VROOLI_HOME is the explicit runtime-home override;
// otherwise the repo-contract default is resolved from the OS user home.
func runtimeHomePath() (string, error) {
	if configured := strings.TrimSpace(os.Getenv("VROOLI_HOME")); configured != "" {
		return filepath.Clean(configured), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return repocontract.VrooliUserRoot(home)
}

func runtimeHomeEntryPath(key string) (string, error) {
	if configured := strings.TrimSpace(os.Getenv("VROOLI_HOME")); configured != "" {
		return repocontract.RuntimeHomeEntryPathAtRoot(configured, key)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return repocontract.RuntimeHomeEntryPath(home, key)
}

// normalizePortClaimKey maps the public environment spelling to the claim
// vocabulary used by peer records and runtime_port_claims. The CLI fallback
// still receives the caller's original spelling because it accepts env-var
// names, while local authorities store the shorter claim name.
func normalizePortClaimKey(portVar string) string {
	key := strings.ToLower(strings.TrimSpace(portVar))
	key = strings.TrimSuffix(key, "_port")
	if key == "api" || key == "ui" {
		return key
	}
	return strings.TrimSpace(portVar)
}

// portLookupRunner is the indirection SetPortLookupRunner replaces. Production
// always uses runScenarioPortCommand.
var portLookupRunner = runScenarioPortCommand

// SetPortLookupRunner replaces the port-lookup implementation and returns a
// function restoring the previous one, dropping the cache on both ends so a
// replacement is actually consulted.
//
// This seam is exported because the cache it fronts is now shared across
// packages: api-core's discovery resolver routes through it, and its tests must
// be able to drive it without a real vrooli binary on PATH. Production code must
// not call this.
func SetPortLookupRunner(fn func(ctx context.Context, target, portVar string) ScenarioPortOutcome) func() {
	previous := portLookupRunner
	if fn == nil {
		fn = runScenarioPortCommand
	}
	portLookupRunner = fn
	resetPortDetectorCache()
	return func() {
		portLookupRunner = previous
		resetPortDetectorCache()
	}
}

// resolveVrooliBinary finds the CLI through the one invoker seam; a miss
// falls back to the bare name so exec reports the PATH failure itself.
func resolveVrooliBinary() string {
	home, _ := userHomeFn()
	path, err := cliinvoke.Resolve(cliinvoke.ResolveOptions{UserHome: home, LookPath: lookPathFn})
	if err != nil {
		return "vrooli"
	}
	return path
}

// runScenarioPortCommand is the sole place this repository shells out after
// peer-record and runtime-registry discovery miss.
func runScenarioPortCommand(ctx context.Context, target, portVar string) ScenarioPortOutcome {
	res := runVrooliFn(ctx, cliinvoke.Invocation{Binary: resolveVrooliBinary(), Args: cliinvoke.ScenarioPortJSON(target, portVar)})
	text := strings.TrimSpace(string(res.Combined()))
	if err := res.Error(); err != nil {
		if failure := scenarioPortFailureFromJSON(text); failure != nil {
			return ScenarioPortOutcome{Output: text, Err: failure}
		}
		return ScenarioPortOutcome{Output: text, Err: err}
	}
	return parseScenarioPortCommandOutput(text)
}

func parseScenarioPortCommandOutput(text string) ScenarioPortOutcome {
	var payload struct {
		Success   *bool  `json:"success"`
		Port      int    `json:"port"`
		Error     string `json:"error"`
		ErrorCode string `json:"error_code"`
	}
	if json.Unmarshal([]byte(text), &payload) == nil && payload.Success != nil {
		if *payload.Success && payload.Port > 0 && payload.Port <= 65535 {
			return ScenarioPortOutcome{Port: strconv.Itoa(payload.Port), Output: text}
		}
		if payload.ErrorCode == ScenarioPortCodeNotRunning {
			return ScenarioPortOutcome{Output: text, Err: &ScenarioPortError{Code: payload.ErrorCode, Message: payload.Error}}
		}
		return ScenarioPortOutcome{Output: text, Err: errors.New("CLI returned an unsuccessful scenario port response")}
	}
	return ScenarioPortOutcome{Port: sanitizePortOutput(text), Output: text}
}

func scenarioPortFailureFromJSON(text string) *ScenarioPortError {
	var payload struct {
		ErrorCode string `json:"error_code"`
		Error     string `json:"error"`
	}
	if json.Unmarshal([]byte(text), &payload) != nil {
		return nil
	}
	if payload.ErrorCode != ScenarioPortCodeNotRunning {
		return nil
	}
	return &ScenarioPortError{Code: payload.ErrorCode, Message: payload.Error}
}

// boundedLookupContext honors a caller's deadline when it has one and otherwise
// applies the default bound, so a caller is never blocked indefinitely by a
// hung CLI nor has its own shorter deadline ignored.
func boundedLookupContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, portLookupTimeout)
}

// DetectPortFromVrooli returns a detector that asks vrooli for the port of a
// scenario. The detector is instance-aware: it resolves the shadow-aware target
// (explicit --instance override or ambient VROOLI_SHADOW_SCENARIOS) at call time,
// so when the named scenario is shadowed the lookup addresses "<name>@shadow".
// An ambient shadow selection may fall back to live only after a typed
// scenario_not_running result. Explicit instance selections, transient and
// unclassified errors, and variant dependencies stay pinned. For an
// unshadowed scenario the target is the bare name and behavior is unchanged.
func DetectPortFromVrooli(scenarioName, portVar string) func() string {
	return func() string {
		target := ResolveShadowTarget(scenarioName)
		outcome := LookupScenarioPort(context.Background(), target, portVar, DefaultPortCachePolicy)
		if ShouldAutomaticallyFallbackToLiveInstance(scenarioName, target, IsScenarioPortNotRunning(outcome.Err)) {
			WarnShadowFallback(scenarioName)
			outcome = LookupScenarioPort(context.Background(), BareScenarioName(scenarioName), portVar, DefaultPortCachePolicy)
		}
		return outcome.Port
	}
}

// DetectScenarioRuntimeStatus returns the lifecycle state for the selected
// scenario instance. Status is already meaningful for a stopped scenario, so
// an empty/error result must stay scoped to that target instead of silently
// substituting the live instance.
func DetectScenarioRuntimeStatus(scenarioName string) func() string {
	return func() string {
		target := ResolveShadowTarget(scenarioName)
		return detectRuntimeStatusForTarget(target)
	}
}

func detectRuntimeStatusForTarget(target string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res := runVrooliFn(ctx, cliinvoke.Invocation{Binary: resolveVrooliBinary(), Args: cliinvoke.ScenarioStatusJSON(target)})
	if res.Class != cliinvoke.OK {
		return ""
	}
	return runtimeStatusFromJSON(string(res.Combined()))
}

func runtimeStatusFromJSON(output string) string {
	var payload struct {
		Scenario struct {
			Status string `json:"status"`
		} `json:"scenario"`
		Runtime struct {
			Status string `json:"status"`
		} `json:"runtime"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		return ""
	}
	if status := strings.TrimSpace(payload.Scenario.Status); status != "" {
		return status
	}
	return strings.TrimSpace(payload.Runtime.Status)
}

func sanitizePortOutput(output string) string {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return ""
	}
	if port := portFromJSON(trimmed); port != "" {
		return port
	}
	// Human-facing lifecycle output commonly contains an ISO date. Never let
	// its year (or month/day) become a plausible-looking port.
	if regexp.MustCompile(`\b\d{4}[-/]\d{1,2}[-/]\d{1,2}\b`).MatchString(trimmed) {
		return ""
	}
	re := regexp.MustCompile(`\b(\d{2,5})\b`)
	match := re.FindString(trimmed)
	if match == "" {
		return ""
	}
	port, err := strconv.Atoi(match)
	if err != nil || port < 1 || port > 65535 {
		return ""
	}
	return match
}

func portFromJSON(output string) string {
	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		return ""
	}
	switch v := payload["port"].(type) {
	case float64:
		if v > 0 {
			return strconv.Itoa(int(v))
		}
	case string:
		return strings.TrimSpace(v)
	}
	return ""
}
