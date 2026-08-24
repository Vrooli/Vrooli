package cliutil

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
)

var (
	lookPathFn              = exec.LookPath
	execCommandContextFn    = exec.CommandContext
	sqliteLookPathFn        = exec.LookPath
	sqliteExecCommandFn     = exec.CommandContext
	peerRecordLookupFn      = lookupPeerRecord
	runtimeRegistryLookupFn = lookupRuntimeRegistry
)

// Port lookups shell out to the vrooli CLI, which is expensive: a cold Go
// binary start plus a host-wide listener snapshot. Long-lived servers resolve
// the same scenario port on every request, so an uncached detector turns
// steady request traffic into a fork storm.
//
// This cache is the single process-wide owner of `vrooli scenario port`.
// api-core's discovery.Resolver routes through it too, so a lookup performed
// for one of them is reused by the other rather than each paying its own fork.
// Before that, two independent implementations of this one operation existed
// and only one of them was cached.
var (
	portCacheMu sync.Mutex
	portCache   = map[string]*portCacheEntry{}

	portCacheNow = time.Now
)

// portLookupTimeout bounds a lookup whose caller supplied no deadline.
const portLookupTimeout = 5 * time.Second

// Default staleness tolerances for callers that express no preference. A
// resolved port is stable for the lifetime of a running scenario, so it may be
// held a while. An unresolved one is re-checked promptly because it usually
// means the scenario is still starting.
const (
	defaultPortMaxAge         = 60 * time.Second
	defaultPortNegativeMaxAge = 3 * time.Second
)

// PortCachePolicy states how stale a cached lookup a caller is willing to
// accept. Freshness is a property of the caller, not of the entry: a resolver
// that must notice a restarted scenario within seconds and a CLI helper that
// can hold an address for a minute share the same underlying process, but
// neither imposes its tolerance on the other. This is what lets one cache serve
// both without either giving up its guarantee.
type PortCachePolicy struct {
	// MaxAge bounds reuse of a successful lookup. Zero disables reuse.
	MaxAge time.Duration
	// NegativeMaxAge bounds reuse of a failed lookup. Zero disables reuse.
	NegativeMaxAge time.Duration
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

// Resolved reports whether the lookup produced a usable port.
func (o ScenarioPortOutcome) Resolved() bool {
	return o.Err == nil && o.Port != ""
}

type portCacheEntry struct {
	mu       sync.Mutex
	outcome  ScenarioPortOutcome
	resolved time.Time
}

// resetPortDetectorCache drops every memoized lookup. Tests use it so a
// replaced execCommandContextFn is actually consulted.
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
		entry = &portCacheEntry{}
		portCache[key] = entry
	}
	return entry
}

// LookupScenarioPort runs `vrooli scenario port <target> <portVar>` at most once
// per policy window and at most once across concurrent callers for the same
// key. Holding the per-key lock across the command is deliberate: a burst of N
// callers for one scenario must cost one process, not N.
func LookupScenarioPort(ctx context.Context, target, portVar string, policy PortCachePolicy) ScenarioPortOutcome {
	entry := portCacheEntryFor(target + "\x00" + portVar)
	entry.mu.Lock()
	defer entry.mu.Unlock()

	if !entry.resolved.IsZero() {
		maxAge := policy.MaxAge
		if !entry.outcome.Resolved() {
			maxAge = policy.NegativeMaxAge
		}
		if maxAge > 0 && portCacheNow().Sub(entry.resolved) < maxAge {
			return entry.outcome
		}
	}

	outcome := peerRecordLookupFn(target, portVar)
	if !outcome.Resolved() {
		outcome = runtimeRegistryLookupFn(ctx, target, portVar)
	}
	if !outcome.Resolved() {
		outcome = portLookupRunner(ctx, target, portVar)
	}

	// A cancelled or expired context describes the caller's deadline, not the
	// target's state. Caching it would let one caller's cancellation deny an
	// unrelated caller for the whole negative window.
	if ctx.Err() != nil {
		return outcome
	}

	entry.outcome, entry.resolved = outcome, portCacheNow()
	return outcome
}

// lookupRuntimeRegistry is the second local authority after peer records.
// The CLI package intentionally keeps this fallback behind the sqlite3 CLI so
// it does not introduce a CGO or driver dependency into every CLI consumer;
// the lifecycle-owned registry remains read-only and the subprocess is only
// reached after the cheap peer-record miss.
func lookupRuntimeRegistry(ctx context.Context, target, portVar string) ScenarioPortOutcome {
	home, err := os.UserHomeDir()
	if err != nil {
		return ScenarioPortOutcome{Err: err}
	}
	dbPath, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyRuntimeDB)
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
	query := fmt.Sprintf("SELECT rpc.port FROM runtime_instances ri JOIN runtime_port_claims rpc ON rpc.instance_id = ri.instance_id WHERE ri.scenario = %s AND ri.variant = %s AND ri.status = 'running' AND rpc.port_name = %s AND rpc.status = 'bound' ORDER BY ri.generation DESC LIMIT 1;", quote(scenario), quote(variant), quote(portVar))
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
	home, err := os.UserHomeDir()
	if err != nil {
		return ScenarioPortOutcome{Err: err}
	}
	root, err := repocontract.VrooliUserRoot(home)
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
		OwnerPID      int            `json:"owner_pid"`
		Ports         map[string]int `json:"ports"`
	}
	if err := json.Unmarshal(payload, &record); err != nil {
		return ScenarioPortOutcome{Err: err}
	}
	if record.SchemaVersion != 1 || record.Scenario != target || !isPIDRunning(record.OwnerPID) {
		return ScenarioPortOutcome{Err: os.ErrNotExist}
	}
	port, ok := record.Ports[portVar]
	if !ok || port <= 0 {
		return ScenarioPortOutcome{Err: os.ErrNotExist}
	}
	return ScenarioPortOutcome{Port: strconv.Itoa(port)}
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

// runScenarioPortCommand is the sole place this repository shells out for a
// scenario port.
func runScenarioPortCommand(ctx context.Context, target, portVar string) ScenarioPortOutcome {
	ctx, cancel := boundedLookupContext(ctx)
	defer cancel()

	argv0 := "vrooli"
	if resolved, err := lookPathFn("vrooli"); err == nil && strings.TrimSpace(resolved) != "" {
		argv0 = resolved
	}
	cmd := execCommandContextFn(ctx, argv0, "--no-stale-check", "--json", "scenario", "port", target, portVar)
	output, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(output))
	if err != nil {
		return ScenarioPortOutcome{Output: text, Err: err}
	}
	return ScenarioPortOutcome{Port: sanitizePortOutput(text), Output: text}
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
// If that non-live lookup yields nothing, it warns once and falls back to the
// live instance — never silent. For an unshadowed scenario the target is the
// bare name and behavior is unchanged.
func DetectPortFromVrooli(scenarioName, portVar string) func() string {
	return func() string {
		target := ResolveShadowTarget(scenarioName)
		port := detectPortForTarget(target, portVar)
		if port == "" && IsNonLiveTarget(target) {
			WarnShadowFallback(scenarioName)
			port = detectPortForTarget(BareScenarioName(scenarioName), portVar)
		}
		return port
	}
}

func detectPortForTarget(target, portVar string) string {
	return LookupScenarioPort(context.Background(), target, portVar, DefaultPortCachePolicy).Port
}

// DetectScenarioRuntimeStatus returns a detector for Vrooli's lifecycle state.
// Unlike a port lookup, the status command distinguishes an intentionally
// stopped scenario from an unknown or merely unconfigured local API. Failures
// intentionally return "" so callers can retain conservative generic guidance.
func DetectScenarioRuntimeStatus(scenarioName string) func() string {
	return func() string {
		target := ResolveShadowTarget(scenarioName)
		status := detectRuntimeStatusForTarget(target)
		if status == "" && IsNonLiveTarget(target) {
			WarnShadowFallback(scenarioName)
			status = detectRuntimeStatusForTarget(BareScenarioName(scenarioName))
		}
		return status
	}
}

func detectRuntimeStatusForTarget(target string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	argv0 := "vrooli"
	if resolved, err := lookPathFn("vrooli"); err == nil && strings.TrimSpace(resolved) != "" {
		argv0 = resolved
	}
	cmd := execCommandContextFn(ctx, argv0, "--no-stale-check", "--json", "scenario", "status", target)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return runtimeStatusFromJSON(string(output))
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
	re := regexp.MustCompile(`\b(\d{2,5})\b`)
	match := re.FindString(trimmed)
	return strings.TrimSpace(match)
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
