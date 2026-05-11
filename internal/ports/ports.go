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

	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/process"
	resourceenv "github.com/vrooli/vrooli/internal/resources/env"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

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
}

type RuntimeClaimOptions struct {
	Enabled    bool
	Context    context.Context
	Store      RuntimeClaimStore
	InstanceID string
	TTL        time.Duration
}

func NewManager(root, home string) (*Manager, error) {
	registry, err := resourceenv.LoadPortRegistry(root)
	if err != nil {
		return nil, err
	}
	return &Manager{
		Root:          filepath.Clean(root),
		Home:          filepath.Clean(home),
		Now:           time.Now,
		ResourcePorts: registry.ResourcePorts,
	}, nil
}

func (m *Manager) StateDir() string {
	return process.ScenarioStateDir(m.Home)
}

func (m *Manager) lockPath(port int) string {
	return filepath.Join(m.StateDir(), fmt.Sprintf(".port_%d.lock", port))
}

func (m *Manager) mutationLockPath(port int) string {
	return filepath.Join(m.StateDir(), fmt.Sprintf(".port_%d.guard", port))
}

func (m *Manager) EnsureStateDir() error {
	return os.MkdirAll(m.StateDir(), 0o755)
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
	return nil
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
	allocated, envVars, runtimeClaims, err := m.allocateScenario(item.Slug, item.Manifest, claimOptions)
	if err != nil {
		return Environment{}, err
	}

	resourceEnv, err := m.loadResourceEnvironment(item.Slug, item.Manifest)
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

	resourceEnv, err := m.loadResourceEnvironment(item.Slug, item.Manifest)
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

func (m *Manager) allocateScenario(scenarioName string, manifest scenario.ServiceManifest, claimOptions RuntimeClaimOptions) (map[string]int, map[string]string, map[string]scenarioruntime.PortClaim, error) {
	allocated := make(map[string]int)
	envVars := make(map[string]string)
	runtimeClaims := make(map[string]scenarioruntime.PortClaim)
	newRuntimeClaims := make(map[string]scenarioruntime.PortClaim)

	for _, portSummary := range manifest.SortedPorts() {
		allocation, err := m.allocatePortDefinition(scenarioName, portSummary, claimOptions)
		if err != nil {
			m.releaseNewRuntimeClaims(runtimeClaimContext(claimOptions), claimOptions.Store, scenarioName, newRuntimeClaims)
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

func (m *Manager) allocatePortDefinition(scenarioName string, portSummary scenario.PortSummary, claimOptions RuntimeClaimOptions) (portAllocation, error) {
	if portSummary.FixedPort != nil {
		port := *portSummary.FixedPort
		claim, claimed, err := m.acquireRuntimePortClaim(scenarioName, portSummary, port, claimOptions)
		if err != nil {
			return portAllocation{}, fmt.Errorf("fixed port %d for %s unavailable: %w", port, portSummary.Name, err)
		}
		if err := m.ensurePortBindable(port, scenarioName); err != nil {
			if claimed {
				_, _ = claimOptions.Store.ReleasePortClaim(runtimeClaimContext(claimOptions), claim.ClaimID)
			}
			return portAllocation{}, fmt.Errorf("fixed port %d for %s unavailable: %w", port, portSummary.Name, err)
		}
		return portAllocation{port: port, runtimeClaim: claim, newClaim: claimed}, nil
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

	size := end - start + 1
	offset := int(crc32.ChecksumIEEE([]byte(scenarioName+"_"+portSummary.Name)) % uint32(size))
	for attempt := 0; attempt < size; attempt++ {
		port := start + ((offset + attempt) % size)
		claim, claimed, err := m.acquireRuntimePortClaim(scenarioName, portSummary, port, claimOptions)
		if err != nil {
			continue
		}
		if err := m.ensurePortBindable(port, scenarioName); err != nil {
			if claimed {
				_, _ = claimOptions.Store.ReleasePortClaim(runtimeClaimContext(claimOptions), claim.ClaimID)
			}
			continue
		}
		return portAllocation{port: port, runtimeClaim: claim, newClaim: claimed}, nil
	}

	return portAllocation{}, fmt.Errorf("no available ports in range %s for %s", portSummary.Range, portSummary.Name)
}

func (m *Manager) acquireRuntimePortClaim(scenarioName string, portSummary scenario.PortSummary, port int, options RuntimeClaimOptions) (scenarioruntime.PortClaim, bool, error) {
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
	claim, err := options.Store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
		InstanceID: options.InstanceID,
		Scenario:   scenarioName,
		PortName:   portSummary.Name,
		EnvVar:     portSummary.EnvVar,
		Port:       port,
		BindHost:   "127.0.0.1",
		URL:        runtimePortURL(portSummary.Name, port),
		Status:     scenarioruntime.ClaimStatusReserved,
		ExpiresAt:  &expiresAt,
	})
	if err != nil {
		if errors.Is(err, scenarioruntime.ErrActiveClaimConflict) {
			return scenarioruntime.PortClaim{}, false, fmt.Errorf("active registry claim already owns port %d", port)
		}
		return scenarioruntime.PortClaim{}, false, err
	}
	return claim, true, nil
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

func (m *Manager) loadResourceEnvironment(scenarioName string, manifest scenario.ServiceManifest) (map[string]string, error) {
	resolution, err := resourceenv.ResolveScenario(m.Root, m.Home, scenarioName, manifest)
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
