package ports

import (
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/portspec"
	"github.com/vrooli/vrooli/internal/process"
	resourceenv "github.com/vrooli/vrooli/internal/resources/env"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func defaultHostBootID(ctx context.Context) (string, error) {
	snap, err := hostsession.DefaultProvider{}.Current(ctx, "")
	if err != nil {
		return "", err
	}
	return snap.BootID, nil
}

const (
	mutationLockTimeout     = 2 * time.Second
	mutationLockRetry       = 10 * time.Millisecond
	mutationLockStaleWindow = 30 * time.Second
	defaultClaimTTL         = 5 * time.Minute
)

var (
	isTCPPortInUseFn             = isTCPPortInUse
	inspectPortListenersFn       = network.InspectPortListeners
	readProcessEnvironmentPortFn = process.ReadEnvironment
)

type Manager struct {
	Root          string
	Home          string
	stateDir      string // resolved once from the runtime_home authority
	Now           func() time.Time
	ResourcePorts map[string]int
}

type Lock struct {
	Scenario  string
	PID       int
	Timestamp time.Time
	Port      int
	Path      string
}

type Environment struct {
	AllocatedPorts map[string]int
	EnvVars        map[string]string
	RuntimeClaims  map[string]scenarioruntime.PortClaim
	IsRunning      bool
	Message        string
}

type RuntimeClaimStore interface {
	scenarioruntime.PortClaimRepository
	scenarioruntime.CleanupRepository
	scenarioruntime.ProcessRefRepository
	scenarioruntime.EventRepository
	GetInstance(ctx context.Context, instanceID string) (scenarioruntime.Instance, error)
	StopLease(ctx context.Context, instanceID string, generation int64, reason string) (scenarioruntime.Instance, error)
}

// PortPreempter is the lifecycle hook the ports package calls to gracefully
// stop a still-live scenario that holds a fixed port we need. It is a small
// interface so that ports never imports lifecycle (and lifecycle's runner is
// the only production implementation).
type PortPreempter interface {
	StopScenario(ctx context.Context, scenarioName string) error
}

// CurrentBootIDFunc returns the current host boot ID. Injected so tests can
// simulate a post-reboot machine without touching /proc.
type CurrentBootIDFunc func(ctx context.Context) string

type RuntimeClaimOptions struct {
	Enabled    bool
	Context    context.Context
	Store      RuntimeClaimStore
	InstanceID string
	TTL        time.Duration

	// Preempter, when non-nil, is called to gracefully stop a live vrooli
	// scenario that holds a fixed port we need. Nil disables live preemption
	// (stale-claim preemption still works without it).
	Preempter PortPreempter

	// CurrentBootID returns the current host boot identifier so the
	// stale-claim classifier can recognize cross-boot leftover state.
	// Defaults to hostsession-derived value when unset.
	CurrentBootID CurrentBootIDFunc

	// PIDIsRunning reports whether the given pid is live. Defaults to
	// process.IsPIDRunning when unset; tests can override.
	PIDIsRunning func(pid int) bool
}

func NewManager(root, home string) (*Manager, error) {
	registry, err := resourceenv.LoadPortRegistry(root)
	if err != nil {
		return nil, err
	}
	cleanHome := filepath.Clean(home)
	stateDir, err := process.ScenarioStateDir(cleanHome)
	if err != nil {
		return nil, err
	}
	return &Manager{
		Root:          filepath.Clean(root),
		Home:          cleanHome,
		stateDir:      stateDir,
		Now:           time.Now,
		ResourcePorts: registry.ResourcePorts,
	}, nil
}

func (m *Manager) StateDir() string {
	return m.stateDir
}

func (m *Manager) lockPath(port int) string {
	return filepath.Join(m.StateDir(), fmt.Sprintf(".port_%d.lock", port))
}

func (m *Manager) mutationLockPath(port int) string {
	return filepath.Join(m.StateDir(), fmt.Sprintf(".port_%d.guard", port))
}

func (m *Manager) EnsureStateDir() error {
	_, err := config.EnsureOwnedDir(m.StateDir())
	return err
}

func (m *Manager) ReadLock(port int) (Lock, bool, error) {
	return m.readLockFile(m.lockPath(port), port)
}

func (m *Manager) readLockFile(path string, port int) (Lock, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Lock{}, false, nil
		}
		return Lock{}, false, err
	}

	raw := strings.TrimSpace(string(data))
	if raw == "" {
		return Lock{Port: port, Path: path}, true, nil
	}

	parts := strings.Split(raw, ":")
	lock := Lock{Port: port, Path: path}
	if len(parts) > 0 {
		lock.Scenario = strings.TrimSpace(parts[0])
	}
	if len(parts) > 1 {
		lock.PID, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
	}
	if len(parts) > 2 {
		if seconds, err := strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 64); err == nil {
			lock.Timestamp = time.Unix(seconds, 0).UTC()
		}
	}
	return lock, true, nil
}

func (m *Manager) RemoveLock(port int) error {
	return m.withMutationLock(port, func() error {
		return m.removeLockUnlocked(port)
	})
}

// RemoveScenarioLocks deletes any leftover `.port_<port>.lock` files owned by
// scenarioName plus the legacy `<scenario>.json` state file. Allocation does
// not consult these files for ownership; the cleanup exists to keep the state
// directory tidy until older installs have stopped writing them.
func (m *Manager) RemoveScenarioLocks(scenarioName string) error {
	locks, err := m.LocksForScenario(scenarioName)
	if err != nil {
		return err
	}
	for _, lock := range locks {
		if err := m.removeLockIfMatches(lock); err != nil {
			return err
		}
	}
	stateFile := filepath.Join(m.StateDir(), scenarioName+".json")
	if err := os.Remove(stateFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (m *Manager) LocksForScenario(scenarioName string) ([]Lock, error) {
	pattern := filepath.Join(m.StateDir(), ".port_*.lock")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	locks := make([]Lock, 0, len(files))
	for _, file := range files {
		port, err := lockPortFromPath(file)
		if err != nil {
			continue
		}
		lock, exists, err := m.ReadLock(port)
		if err != nil || !exists {
			continue
		}
		if lock.Scenario == scenarioName {
			locks = append(locks, lock)
		}
	}
	sort.Slice(locks, func(i, j int) bool { return locks[i].Port < locks[j].Port })
	return locks, nil
}

func lockPortFromPath(path string) (int, error) {
	name := strings.TrimSuffix(filepath.Base(path), ".lock")
	name = strings.TrimPrefix(name, ".port_")
	return strconv.Atoi(name)
}

type mutationGuard struct {
	PID       int
	Timestamp time.Time
}

func (m *Manager) withMutationLock(port int, fn func() error) error {
	if err := m.EnsureStateDir(); err != nil {
		return err
	}
	release, err := m.acquireMutationLock(port)
	if err != nil {
		return err
	}
	defer release()
	return fn()
}

func (m *Manager) acquireMutationLock(port int) (func(), error) {
	path := m.mutationLockPath(port)
	deadline := time.Now().Add(mutationLockTimeout)
	payload := []byte(fmt.Sprintf("%d:%d\n", os.Getpid(), m.Now().Unix()))

	for {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			if _, writeErr := file.Write(payload); writeErr != nil {
				_ = file.Close()
				_ = os.Remove(path)
				return nil, writeErr
			}
			if closeErr := file.Close(); closeErr != nil {
				_ = os.Remove(path)
				return nil, closeErr
			}
			_ = config.ChownToInvokingUser(path)
			return func() {
				_ = os.Remove(path)
			}, nil
		}
		if !os.IsExist(err) {
			return nil, err
		}

		guard, exists, readErr := m.readMutationGuard(path)
		if readErr == nil && !exists {
			continue
		}
		if readErr == nil && exists {
			age := m.Now().UTC().Sub(guard.Timestamp)
			if (guard.PID > 0 && !process.IsPIDRunning(guard.PID)) || age > mutationLockStaleWindow {
				_ = os.Remove(path)
				continue
			}
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for port %d mutation lock", port)
		}
		time.Sleep(mutationLockRetry)
	}
}

func (m *Manager) readMutationGuard(path string) (mutationGuard, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return mutationGuard{}, false, nil
		}
		return mutationGuard{}, false, err
	}
	parts := strings.Split(strings.TrimSpace(string(data)), ":")
	guard := mutationGuard{}
	if len(parts) > 0 {
		guard.PID, _ = strconv.Atoi(strings.TrimSpace(parts[0]))
	}
	if len(parts) > 1 {
		if seconds, parseErr := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64); parseErr == nil {
			guard.Timestamp = time.Unix(seconds, 0).UTC()
		}
	}
	return guard, true, nil
}

func (m *Manager) removeLockUnlocked(port int) error {
	if err := os.Remove(m.lockPath(port)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (m *Manager) removeLockIfMatches(expected Lock) error {
	return m.withMutationLock(expected.Port, func() error {
		current, exists, err := m.ReadLock(expected.Port)
		if err != nil {
			return err
		}
		if !exists {
			return nil
		}
		if current.Scenario != expected.Scenario || current.PID != expected.PID || !current.Timestamp.Equal(expected.Timestamp) {
			return nil
		}
		return m.removeLockUnlocked(expected.Port)
	})
}

func writeFileAtomically(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	file, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := file.Name()
	cleanup := func() {
		_ = os.Remove(tmpPath)
	}
	if err := file.Chmod(perm); err != nil {
		_ = file.Close()
		cleanup()
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		cleanup()
		return err
	}
	if err := file.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		cleanup()
		return err
	}
	// A sudo'd write would otherwise leave this root-owned in the operator's home.
	return config.ChownToInvokingUser(path)
}

func (m *Manager) CleanStaleLocks() error {
	if err := m.EnsureStateDir(); err != nil {
		return err
	}
	pattern := filepath.Join(m.StateDir(), ".port_*.lock")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	for _, file := range files {
		port, err := lockPortFromPath(file)
		if err != nil {
			continue
		}
		lock, exists, err := m.ReadLock(port)
		if err != nil || !exists {
			continue
		}
		if lock.PID > 0 && process.IsPIDRunning(lock.PID) {
			continue
		}
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (m *Manager) BuildEnvironment(item scenario.Scenario, records []process.Record) (Environment, error) {
	return m.BuildEnvironmentWithRuntimeClaims(item, records, RuntimeClaimOptions{})
}

func (m *Manager) BuildEnvironmentWithRuntimeClaims(item scenario.Scenario, _ []process.Record, claimOptions RuntimeClaimOptions) (Environment, error) {
	instanceKey := scenarioruntime.InstanceKey{Scenario: item.Slug, Variant: item.Variant}.Normalize()
	ns := instanceKey.Namespace()

	allocated, envVars, runtimeClaims, err := m.allocateScenario(instanceKey, item.Manifest, claimOptions)
	if err != nil {
		return Environment{}, err
	}

	resourceEnv, err := m.loadResourceEnvironment(instanceKey, item.Manifest)
	if err != nil {
		return Environment{}, err
	}
	for key, value := range resourceEnv {
		envVars[key] = value
	}

	scenarioVars := map[string]string{
		"SCENARIO_NAME":       item.Slug,
		"SCENARIO_MODE":       "true",
		"SCENARIO_PATH":       item.Path,
		"SCENARIO_DATA_DIR":   filepath.Join(item.Path, "data"),
		"VROOLI_SCENARIO":     item.Slug,
		"VROOLI_SCENARIO_DIR": item.Path,
	}
	// VROOLI_SCENARIO stays the bare slug; the variant-aware namespace is carried
	// in its own env vars (VROOLI_VARIANT + VROOLI_STORAGE_NAMESPACE) from the
	// InstanceKey SSOT, so scenarios derive shadow-isolated storage namespaces
	// from the environment instead of hardcoding their slug. See §1a / P5.
	for key, value := range ns.EnvVars {
		scenarioVars[key] = value
	}
	for key, value := range scenarioVars {
		envVars[key] = value
	}

	expandedManifestEnv := make(map[string]string, len(item.Manifest.Environment))
	for key, value := range item.Manifest.Environment {
		expandedManifestEnv[key] = expandTemplate(value, envVars)
	}
	for key, value := range expandedManifestEnv {
		envVars[key] = value
	}

	return Environment{
		AllocatedPorts: allocated,
		EnvVars:        envVars,
		RuntimeClaims:  runtimeClaims,
		Message:        "allocated ports for scenario",
	}, nil
}

func (m *Manager) BuildProjectEnvironment(item scenario.Scenario) (Environment, error) {
	allocated := make(map[string]int)
	envVars := make(map[string]string)

	for _, portSummary := range item.Manifest.SortedPorts() {
		if portSummary.EnvVar == "" {
			continue
		}
		port := 0
		if override, err := strconv.Atoi(strings.TrimSpace(os.Getenv(portSummary.EnvVar))); err == nil && override > 0 {
			port = override
		} else if portSummary.FixedPort != nil {
			port = *portSummary.FixedPort
		}
		if port <= 0 {
			continue
		}
		allocated[portSummary.Name] = port
		envVars[portSummary.EnvVar] = strconv.Itoa(port)
	}

	resourceEnv, err := m.loadResourceEnvironment(scenarioruntime.InstanceKey{Scenario: item.Slug, Variant: item.Variant}.Normalize(), item.Manifest)
	if err != nil {
		return Environment{}, err
	}
	for key, value := range resourceEnv {
		envVars[key] = value
	}

	expandedManifestEnv := make(map[string]string, len(item.Manifest.Environment))
	for key, value := range item.Manifest.Environment {
		expandedManifestEnv[key] = expandTemplate(value, envVars)
	}
	for key, value := range expandedManifestEnv {
		envVars[key] = value
	}

	return Environment{
		AllocatedPorts: allocated,
		EnvVars:        envVars,
		Message:        "resolved fixed ports for project lifecycle",
	}, nil
}

func (m *Manager) allocateScenario(key scenarioruntime.InstanceKey, manifest scenario.ServiceManifest, claimOptions RuntimeClaimOptions) (map[string]int, map[string]string, map[string]scenarioruntime.PortClaim, error) {
	allocated := make(map[string]int)
	envVars := make(map[string]string)
	runtimeClaims := make(map[string]scenarioruntime.PortClaim)
	newRuntimeClaims := make(map[string]scenarioruntime.PortClaim)

	for _, portSummary := range manifest.SortedPorts() {
		allocation, err := m.allocatePortDefinition(key, portSummary, claimOptions)
		if err != nil {
			m.releaseNewRuntimeClaims(runtimeClaimContext(claimOptions), claimOptions.Store, key.Scenario, newRuntimeClaims)
			return nil, nil, nil, err
		}
		port := allocation.port
		if port <= 0 {
			continue
		}
		allocated[portSummary.Name] = port
		envVars[portSummary.EnvVar] = strconv.Itoa(port)
		if allocation.runtimeClaim.ClaimID != "" {
			runtimeClaims[portSummary.Name] = allocation.runtimeClaim
			if allocation.newClaim {
				newRuntimeClaims[portSummary.Name] = allocation.runtimeClaim
			}
		}
	}

	return allocated, envVars, runtimeClaims, nil
}

type portAllocation struct {
	port         int
	runtimeClaim scenarioruntime.PortClaim
	newClaim     bool
}

func (m *Manager) allocatePortDefinition(key scenarioruntime.InstanceKey, portSummary scenario.PortSummary, claimOptions RuntimeClaimOptions) (portAllocation, error) {
	if portSummary.FixedPort != nil {
		port := *portSummary.FixedPort
		// Fixed ports are a live-only privilege (§1a / P1): a constant port can
		// only be honored by one instance, and that is reserved for live.
		if key.IsLive() {
			claim, claimed, err := m.acquireRuntimePortClaim(key, portSummary, port, claimOptions, true)
			if err != nil {
				return portAllocation{}, fmt.Errorf("fixed port %d for %s unavailable: %w", port, portSummary.Name, err)
			}
			if err := m.ensurePortBindable(port, key.Scenario); err != nil {
				if claimed {
					_, _ = claimOptions.Store.ReleasePortClaim(runtimeClaimContext(claimOptions), claim.ClaimID)
				}
				return portAllocation{}, fmt.Errorf("fixed port %d for %s unavailable: %w", port, portSummary.Name, err)
			}
			return portAllocation{port: port, runtimeClaim: claim, newClaim: claimed}, nil
		}
		// A non-live variant cannot take the constant (it belongs to live), but
		// it must NOT be left without a port — skipping would leave e.g. a
		// shadow's UI unstartable and the scenario permanently degraded. Fall
		// back to a dynamically-allocated port in the SAME canonical band as the
		// fixed port (a UI fixed port -> the UI band, an API fixed port -> the
		// API band, etc. — see internal/portspec), so the variant keeps a
		// role-appropriate port. The live fixed value itself is excluded so the
		// variant can never collide with (or be preempted by) live.
		bandStart, bandEnd := fallbackBandForFixedPort(portSummary)
		return m.allocateFromBand(key, portSummary, claimOptions, bandStart, bandEnd, port)
	}

	if portSummary.Range == "" {
		return portAllocation{}, nil
	}

	start, end, err := parseRange(portSummary.Range)
	if err != nil {
		return portAllocation{}, fmt.Errorf("parse range for %s: %w", portSummary.Name, err)
	}
	if end < start {
		return portAllocation{}, fmt.Errorf("invalid range %q", portSummary.Range)
	}
	return m.allocateFromBand(key, portSummary, claimOptions, start, end, -1)
}

// allocateFromBand deterministically picks a free, bindable port in [start,end].
// The first-choice port is seeded from the variant-aware PortSeed (bare slug for
// live — so live ports never shift — "scenario@variant" otherwise) so different
// variants prefer different ports. avoid (>= 0) is never selected; it keeps a
// non-live variant off the live fixed port whose band it is borrowing.
func (m *Manager) allocateFromBand(key scenarioruntime.InstanceKey, portSummary scenario.PortSummary, claimOptions RuntimeClaimOptions, start, end, avoid int) (portAllocation, error) {
	if end < start {
		return portAllocation{}, fmt.Errorf("invalid port band %d-%d for %s", start, end, portSummary.Name)
	}
	ns := key.Namespace()
	size := end - start + 1
	offset := int(crc32.ChecksumIEEE([]byte(ns.PortSeed+"_"+portSummary.Name)) % uint32(size))
	for attempt := 0; attempt < size; attempt++ {
		port := start + ((offset + attempt) % size)
		if port == avoid {
			continue
		}
		claim, claimed, err := m.acquireRuntimePortClaim(key, portSummary, port, claimOptions, false)
		if err != nil {
			continue
		}
		if err := m.ensurePortBindable(port, key.Scenario); err != nil {
			if claimed {
				_, _ = claimOptions.Store.ReleasePortClaim(runtimeClaimContext(claimOptions), claim.ClaimID)
			}
			continue
		}
		return portAllocation{port: port, runtimeClaim: claim, newClaim: claimed}, nil
	}
	return portAllocation{}, fmt.Errorf("no available ports in band %d-%d for %s", start, end, portSummary.Name)
}

// fallbackBandForFixedPort returns the canonical port band a non-live variant
// should borrow when the live instance owns a fixed port. The role declared by
// the env var / name is the primary signal (API_PORT -> API band, UI_PORT -> UI
// band, ... — the scenario's stated intent); a fixed port whose name is
// role-ambiguous falls back to the canonical band its own value sits in (e.g. a
// constant of 21241 -> the UI band), and finally to the reserved headroom band.
func fallbackBandForFixedPort(portSummary scenario.PortSummary) (int, int) {
	if role := roleFromPortName(portSummary.EnvVar, portSummary.Name); role != portspec.RoleUnknown {
		return bandRangeForRole(role)
	}
	fixed := 0
	if portSummary.FixedPort != nil {
		fixed = *portSummary.FixedPort
	}
	if role, ok := portspec.CanonicalBand(fixed); ok {
		return bandRangeForRole(role)
	}
	return portspec.ReservedHeadroomStart, portspec.ReservedHeadroomEnd
}

func bandRangeForRole(role portspec.CanonicalRole) (int, int) {
	switch role {
	case portspec.RoleAPI:
		return portspec.APIRangeStart, portspec.APIRangeEnd
	case portspec.RoleUI:
		return portspec.UIRangeStart, portspec.UIRangeEnd
	case portspec.RoleWS:
		return portspec.WSRangeStart, portspec.WSRangeEnd
	default:
		return portspec.ReservedHeadroomStart, portspec.ReservedHeadroomEnd
	}
}

func roleFromPortName(envVar, name string) portspec.CanonicalRole {
	s := strings.ToLower(envVar + " " + name)
	switch {
	case strings.Contains(s, "ws") || strings.Contains(s, "websocket"):
		return portspec.RoleWS
	case strings.Contains(s, "ui"):
		return portspec.RoleUI
	case strings.Contains(s, "api"):
		return portspec.RoleAPI
	default:
		return portspec.RoleUnknown
	}
}

func (m *Manager) acquireRuntimePortClaim(key scenarioruntime.InstanceKey, portSummary scenario.PortSummary, port int, options RuntimeClaimOptions, fixed bool) (scenarioruntime.PortClaim, bool, error) {
	if !runtimeClaimsEnabled(options) {
		return scenarioruntime.PortClaim{}, false, nil
	}
	ctx := runtimeClaimContext(options)
	if err := expireReservedRuntimeClaims(ctx, options.Store, m.Now().UTC()); err != nil {
		return scenarioruntime.PortClaim{}, false, err
	}
	existing, ok, err := findExistingRuntimeClaim(ctx, options.Store, options.InstanceID, portSummary.Name, port)
	if err != nil {
		return scenarioruntime.PortClaim{}, false, err
	}
	if ok {
		return existing, false, nil
	}
	ttl := options.TTL
	if ttl <= 0 {
		ttl = defaultClaimTTL
	}
	expiresAt := m.Now().UTC().Add(ttl)
	newClaim := func() (scenarioruntime.PortClaim, error) {
		return options.Store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
			InstanceID: options.InstanceID,
			Scenario:   key.Scenario,
			Variant:    key.Variant,
			PortName:   portSummary.Name,
			EnvVar:     portSummary.EnvVar,
			Port:       port,
			BindHost:   "127.0.0.1",
			URL:        runtimePortURL(portSummary.Name, port),
			Status:     scenarioruntime.ClaimStatusReserved,
			ExpiresAt:  &expiresAt,
		})
	}
	claim, err := newClaim()
	if err == nil {
		return claim, true, nil
	}
	if !errors.Is(err, scenarioruntime.ErrActiveClaimConflict) {
		return scenarioruntime.PortClaim{}, false, err
	}
	// Conflict path. Range allocators retry on the next port and never
	// preempt — fixed-port callers are the only ones that should disturb
	// other instances' state.
	if !fixed {
		return scenarioruntime.PortClaim{}, false, fmt.Errorf("active registry claim already owns port %d", port)
	}
	preempted, kind, conflictScenario, perr := m.preemptFixedPortConflict(ctx, options, key, port)
	if perr != nil {
		return scenarioruntime.PortClaim{}, false, fmt.Errorf("active registry claim already owns port %d: preemption failed: %w", port, perr)
	}
	if !preempted {
		return scenarioruntime.PortClaim{}, false, fmt.Errorf("active registry claim already owns port %d", port)
	}
	claim, err = newClaim()
	if err != nil {
		if errors.Is(err, scenarioruntime.ErrActiveClaimConflict) {
			return scenarioruntime.PortClaim{}, false, fmt.Errorf("active registry claim already owns port %d (preempted %s claim from %q, still conflicting)", port, kind, conflictScenario)
		}
		return scenarioruntime.PortClaim{}, false, err
	}
	m.recordPortPreemptedEvent(ctx, options, key.Scenario, port, kind, conflictScenario)
	return claim, true, nil
}

// preemptFixedPortConflict resolves an active registry claim that's blocking
// a fixed-port acquire. It returns (preempted, kind, conflictingScenario, error).
// `kind` is one of "stale" / "live" / "" (when no claim found).
//
// Stale conflicts are finalized in place via FinalizeStuckInstance. Live
// conflicts are stopped via the lifecycle Preempter callback when one is
// configured; without a Preempter, a live conflict is not preempted and the
// caller surfaces the original ErrActiveClaimConflict.
func (m *Manager) preemptFixedPortConflict(ctx context.Context, options RuntimeClaimOptions, key scenarioruntime.InstanceKey, port int) (bool, string, string, error) {
	conflict, ok, err := findActiveClaimOnPort(ctx, options.Store, port)
	if err != nil {
		return false, "", "", err
	}
	if !ok {
		// No registry row holds the port — the conflict is either a race
		// with another in-flight acquire, or transient. Let the caller
		// surface the original conflict error.
		return false, "", "", nil
	}
	// Never preempt a different variant. Fixed ports are live-only, so the
	// requester here is always live and may only reclaim its own live-variant
	// stale/leftover claim; a claim owned by another variant is left untouched
	// (defense in depth — the allocator already skips fixed ports for non-live).
	conflictVariant := scenarioruntime.InstanceKey{Scenario: conflict.Scenario, Variant: conflict.Variant}.Normalize().Variant
	if conflictVariant != key.Variant {
		return false, "live", conflict.Scenario, nil
	}
	instance, err := options.Store.GetInstance(ctx, conflict.InstanceID)
	if err != nil && !errors.Is(err, scenarioruntime.ErrNotFound) {
		return false, "", conflict.Scenario, err
	}
	if errors.Is(err, scenarioruntime.ErrNotFound) {
		// Claim row points at a deleted instance — definitionally stale.
		// Just release the orphan claim.
		_, relErr := options.Store.ReleasePortClaim(ctx, conflict.ClaimID)
		if relErr != nil && !errors.Is(relErr, scenarioruntime.ErrNotFound) {
			return false, "stale", conflict.Scenario, relErr
		}
		return true, "stale", conflict.Scenario, nil
	}

	currentBootID := resolveCurrentBootID(ctx, options)
	pidIsRunning := options.PIDIsRunning
	if pidIsRunning == nil {
		pidIsRunning = process.IsPIDRunning
	}
	now := m.Now().UTC()

	if trigger, stale := classifyStuckInstance(instance, currentBootID, pidIsRunning, now); stale {
		if err := scenarioruntime.FinalizeStuckInstance(ctx, options.Store, instance, trigger, now); err != nil {
			return false, "stale", instance.Scenario, err
		}
		return true, "stale", instance.Scenario, nil
	}

	// Live conflict — only preempt via the lifecycle hook.
	if options.Preempter == nil {
		return false, "live", instance.Scenario, nil
	}
	if err := options.Preempter.StopScenario(ctx, instance.Scenario); err != nil {
		return false, "live", instance.Scenario, err
	}
	return true, "live", instance.Scenario, nil
}

// findActiveClaimOnPort returns the active (reserved or bound) registry claim
// that currently owns the given port, if any. There can be at most one due to
// the unique active-port index on runtime_port_claims.
func findActiveClaimOnPort(ctx context.Context, store RuntimeClaimStore, port int) (scenarioruntime.PortClaim, bool, error) {
	claims, err := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{
		Statuses: scenarioruntime.ActivePortClaimStatuses(),
	})
	if err != nil {
		return scenarioruntime.PortClaim{}, false, err
	}
	for _, c := range claims {
		if c.Port == port {
			return c, true, nil
		}
	}
	return scenarioruntime.PortClaim{}, false, nil
}

// classifyStuckInstance decides whether a vrooli-owned conflicting instance
// is stale enough that the ports preemption path can finalize it in place
// without consulting the lifecycle. It is more conservative than the
// maintenance reaper because it runs on any active claim — not just on
// status=stopping. The rules:
//
//   - status NOT in {starting, running} → stale (the instance is already
//     past its lifecycle; the active claim is leftover state),
//   - boot mismatch → stale (machine rebooted; pid space changed),
//   - owner_pid set but dead → stale,
//   - heartbeat deadline elapsed → stale,
//   - otherwise → live (owner_pid missing alone is *not* enough; many
//     legitimate in-flight startups have not recorded a pid yet, and
//     preempting them would be a self-inflicted denial of service).
func classifyStuckInstance(instance scenarioruntime.Instance, currentBootID string, pidIsRunning func(int) bool, now time.Time) (string, bool) {
	if currentBootID != "" && instance.HostBootID != "" && instance.HostBootID != currentBootID {
		return "boot_id_mismatch", true
	}
	if !scenarioruntime.IsActiveInstanceStatus(instance.Status) {
		return "non_active_status:" + instance.Status, true
	}
	if instance.OwnerPID != nil && *instance.OwnerPID > 0 && !pidIsRunning(*instance.OwnerPID) {
		return "owner_pid_dead", true
	}
	// Heartbeat-only staleness is no longer sufficient: scenarios with
	// long-running setup/develop phases (e.g. web-console's UI build at
	// 4400+ vite modules) routinely run past the 30s heartbeat TTL while
	// the owning process is still alive and making forward progress.
	// Preempting in that window kills in-flight builds and surfaces as
	// "generation is stale" / bind UNIQUE constraint errors. Require an
	// additional corroborating signal — owner_pid known-dead, status
	// non-active, or boot mismatch — before classifying as stale. A
	// missing owner_pid combined with an expired heartbeat is treated as
	// stale because the supervisor is supposed to populate owner_pid
	// once it has heartbeated at least once.
	if instance.HeartbeatDeadlineAt != nil && !instance.HeartbeatDeadlineAt.After(now) {
		if instance.OwnerPID == nil || *instance.OwnerPID <= 0 {
			return "heartbeat_expired_no_owner", true
		}
	}
	return "", false
}

func resolveCurrentBootID(ctx context.Context, options RuntimeClaimOptions) string {
	if options.CurrentBootID != nil {
		return options.CurrentBootID(ctx)
	}
	host, err := defaultHostBootID(ctx)
	if err != nil {
		return ""
	}
	return host
}

// recordPortPreemptedEvent writes a port_preempted runtime event for forensics.
// Best-effort: errors are intentionally swallowed because preemption already
// succeeded and the acquire path should not fail solely due to an event-log
// write failure.
func (m *Manager) recordPortPreemptedEvent(ctx context.Context, options RuntimeClaimOptions, requestingScenario string, port int, kind, conflictingScenario string) {
	details := fmt.Sprintf(`{"port":%d,"requesting_scenario":%q,"conflicting_scenario":%q,"conflict_kind":%q}`,
		port, requestingScenario, conflictingScenario, kind)
	_, _ = options.Store.RecordEvent(ctx, scenarioruntime.Event{
		InstanceID:  options.InstanceID,
		Scenario:    requestingScenario,
		EventType:   "port_preempted",
		DetailsJSON: details,
	})
}

func runtimeClaimsEnabled(options RuntimeClaimOptions) bool {
	return options.Enabled && options.Store != nil && strings.TrimSpace(options.InstanceID) != ""
}

func (m *Manager) releaseNewRuntimeClaims(ctx context.Context, store RuntimeClaimStore, _ string, claims map[string]scenarioruntime.PortClaim) {
	if store == nil || len(claims) == 0 {
		return
	}
	for _, claim := range claims {
		if claim.ClaimID == "" {
			continue
		}
		_, _ = store.ReleasePortClaim(ctx, claim.ClaimID)
	}
}

func runtimeClaimContext(options RuntimeClaimOptions) context.Context {
	if options.Context != nil {
		return options.Context
	}
	return context.Background()
}

func findExistingRuntimeClaim(ctx context.Context, store RuntimeClaimStore, instanceID string, portName string, port int) (scenarioruntime.PortClaim, bool, error) {
	claims, err := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{
		InstanceID: instanceID,
		Statuses:   scenarioruntime.ActivePortClaimStatuses(),
	})
	if err != nil {
		return scenarioruntime.PortClaim{}, false, err
	}
	for _, claim := range claims {
		if claim.PortName == portName && claim.Port == port {
			return claim, true, nil
		}
	}
	return scenarioruntime.PortClaim{}, false, nil
}

func expireReservedRuntimeClaims(ctx context.Context, store RuntimeClaimStore, at time.Time) error {
	claims, err := expiredReservedRuntimeClaims(ctx, store, at)
	if err != nil {
		return err
	}
	for _, claim := range claims {
		if _, err := store.ExpirePortClaim(ctx, claim.ClaimID); err != nil {
			return err
		}
	}
	return nil
}

func expiredReservedRuntimeClaims(ctx context.Context, store RuntimeClaimStore, at time.Time) ([]scenarioruntime.PortClaim, error) {
	claims, err := store.ListExpiredActivePortClaims(ctx, at)
	if err != nil {
		return nil, err
	}
	reserved := make([]scenarioruntime.PortClaim, 0, len(claims))
	for _, claim := range claims {
		if claim.Status != scenarioruntime.ClaimStatusReserved {
			continue
		}
		reserved = append(reserved, claim)
	}
	return reserved, nil
}

func runtimePortURL(portName string, port int) string {
	return scenarioruntime.LocalPortURL(portName, port)
}

func parseRange(value string) (int, int, error) {
	parts := strings.Split(strings.TrimSpace(value), "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected start-end range")
	}
	start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, err
	}
	end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, err
	}
	return start, end, nil
}

// ensurePortBindable confirms the OS port is not already held by a foreign
// listener. The registry claim acquired before this call is the ownership
// authority; this only adds a socket-bind safety check so a same-scenario
// restart can detect lingering external processes squatting on the port.
//
// Lock files are no longer consulted here. A leftover `.port_<port>.lock`
// file is a legacy artifact that the maintenance layer will clean up;
// allocation never treats it as ownership evidence.
func (m *Manager) ensurePortBindable(port int, scenarioName string) error {
	if reservedByResource(m.ResourcePorts, port) {
		return fmt.Errorf("reserved for resource")
	}
	inUse, err := isTCPPortInUseFn(port)
	if err != nil {
		return err
	}
	if !inUse {
		return nil
	}
	// A listener exists. If it belongs to this scenario, treat as a
	// recoverable restart-in-progress (the registry claim is already ours).
	// Anything else is a real conflict that the operator must resolve.
	detail, ok, descErr := describeVrooliPortConflict(port, scenarioName)
	if descErr != nil {
		return descErr
	}
	if ok {
		if strings.Contains(detail, fmt.Sprintf("scenario %q", scenarioName)) {
			return nil
		}
		return errors.New(detail)
	}
	return errors.New("port already in use")
}

func describeVrooliPortConflict(port int, scenarioName string) (string, bool, error) {
	inspection, err := inspectPortListenersFn(port)
	if err != nil {
		return "", false, err
	}
	if !inspection.Inspection.Available {
		return "", false, nil
	}

	for _, listener := range inspection.Listeners {
		env, err := readProcessEnvironmentPortFn(listener.PID)
		if err != nil {
			continue
		}
		if !isVrooliManagedListener(env) {
			continue
		}
		ownerScenario := strings.TrimSpace(env["VROOLI_SCENARIO"])
		if ownerScenario == "" {
			return fmt.Sprintf("port already in use by Vrooli-managed listener (pid %d)", listener.PID), true, nil
		}
		if ownerScenario == scenarioName {
			return fmt.Sprintf("port already in use by existing Vrooli listener for scenario %q (pid %d)", ownerScenario, listener.PID), true, nil
		}
		return fmt.Sprintf("port already in use by Vrooli scenario %q (pid %d)", ownerScenario, listener.PID), true, nil
	}

	return "", false, nil
}

func isVrooliManagedListener(env map[string]string) bool {
	if strings.EqualFold(strings.TrimSpace(env["VROOLI_LIFECYCLE_MANAGED"]), "true") {
		return true
	}
	return strings.TrimSpace(env["VROOLI_SCENARIO"]) != ""
}

func reservedByResource(resourcePorts map[string]int, port int) bool {
	for _, reserved := range resourcePorts {
		if reserved == port {
			return true
		}
	}
	return false
}

func isTCPPortInUse(port int) (bool, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		var addrErr *net.OpError
		if errors.As(err, &addrErr) {
			return true, nil
		}
		if strings.Contains(strings.ToLower(err.Error()), "address already in use") {
			return true, nil
		}
		return false, err
	}
	_ = ln.Close()
	return false, nil
}

func (m *Manager) loadResourceEnvironment(key scenarioruntime.InstanceKey, manifest scenario.ServiceManifest) (map[string]string, error) {
	resolution, err := resourceenv.ResolveScenario(m.Root, m.Home, key.Scenario, key.Variant, manifest)
	if err != nil {
		return nil, err
	}
	return resolution.Values, nil
}

func expandTemplate(value string, env map[string]string) string {
	expanded := value

	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		expanded = strings.ReplaceAll(expanded, "${"+key+"}", env[key])
		expanded = strings.ReplaceAll(expanded, "$"+key, env[key])
	}
	return expanded
}
