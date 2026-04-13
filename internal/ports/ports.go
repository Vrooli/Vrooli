package ports

import (
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

	"github.com/vrooli/vrooli/internal/process"
	resourceenv "github.com/vrooli/vrooli/internal/resources/env"
	"github.com/vrooli/vrooli/internal/scenario"
)

const (
	staleLockWindow         = 5 * time.Minute
	mutationLockTimeout     = 2 * time.Second
	mutationLockRetry       = 10 * time.Millisecond
	mutationLockStaleWindow = 30 * time.Second
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
	IsRunning      bool
	Message        string
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

func (m *Manager) WriteLock(port int, scenarioName string, pid int) error {
	return m.withMutationLock(port, func() error {
		return m.writeLockUnlocked(port, scenarioName, pid)
	})
}

func (m *Manager) claimLock(port int, scenarioName string, pid int) error {
	return m.withMutationLock(port, func() error {
		lock, exists, err := m.ReadLock(port)
		if err != nil {
			return err
		}
		if exists && lock.Scenario != "" && lock.Scenario != scenarioName {
			return fmt.Errorf("port %d locked by scenario %q", port, lock.Scenario)
		}
		return m.writeLockUnlocked(port, scenarioName, pid)
	})
}

func (m *Manager) RemoveLock(port int) error {
	return m.withMutationLock(port, func() error {
		return m.removeLockUnlocked(port)
	})
}

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

func (m *Manager) writeLockUnlocked(port int, scenarioName string, pid int) error {
	if err := m.EnsureStateDir(); err != nil {
		return err
	}
	content := []byte(fmt.Sprintf("%s:%d:%d\n", scenarioName, pid, m.Now().Unix()))
	return writeFileAtomically(m.lockPath(port), content, 0o644)
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
	live := process.LiveRecords(records)

	allocated, envVars, err := m.allocateScenario(item.Slug, item.Manifest, live)
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
		IsRunning:      len(live) > 0,
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

func (m *Manager) allocateScenario(scenarioName string, manifest scenario.ServiceManifest, records []process.Record) (map[string]int, map[string]string, error) {
	allocated := make(map[string]int)
	envVars := make(map[string]string)

	for _, portSummary := range manifest.SortedPorts() {
		port, err := m.allocatePortDefinition(scenarioName, records, portSummary)
		if err != nil {
			return nil, nil, err
		}
		if port <= 0 {
			continue
		}
		allocated[portSummary.Name] = port
		envVars[portSummary.EnvVar] = strconv.Itoa(port)
	}

	return allocated, envVars, nil
}

func (m *Manager) allocatePortDefinition(scenarioName string, records []process.Record, portSummary scenario.PortSummary) (int, error) {
	if portSummary.FixedPort != nil {
		port := *portSummary.FixedPort
		ownerPID, err := m.ensurePortClaimed(port, scenarioName, records)
		if err != nil {
			return 0, fmt.Errorf("fixed port %d for %s unavailable: %w", port, portSummary.Name, err)
		}
		if err := m.claimLock(port, scenarioName, ownerPID); err != nil {
			return 0, err
		}
		return port, nil
	}

	if portSummary.Range == "" {
		return 0, nil
	}

	start, end, err := parseRange(portSummary.Range)
	if err != nil {
		return 0, fmt.Errorf("parse range for %s: %w", portSummary.Name, err)
	}
	if end < start {
		return 0, fmt.Errorf("invalid range %q", portSummary.Range)
	}

	size := end - start + 1
	offset := int(crc32.ChecksumIEEE([]byte(scenarioName+"_"+portSummary.Name)) % uint32(size))
	for attempt := 0; attempt < size; attempt++ {
		port := start + ((offset + attempt) % size)
		ownerPID, err := m.ensurePortClaimed(port, scenarioName, records)
		if err != nil {
			continue
		}
		if err := m.claimLock(port, scenarioName, ownerPID); err != nil {
			continue
		}
		return port, nil
	}

	return 0, fmt.Errorf("no available ports in range %s for %s", portSummary.Range, portSummary.Name)
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

func (m *Manager) ensurePortClaimed(port int, scenarioName string, records []process.Record) (int, error) {
	if reservedByResource(m.ResourcePorts, port) {
		return 0, fmt.Errorf("reserved for resource")
	}

	lock, exists, err := m.ReadLock(port)
	if err != nil {
		return 0, err
	}
	if exists {
		switch {
		case lock.Scenario == scenarioName && lock.PID > 0 && process.IsPIDRunning(lock.PID):
			return lock.PID, nil
		case lock.Scenario == scenarioName:
			_ = m.removeLockIfMatches(lock)
		case lock.PID > 0 && process.IsPIDRunning(lock.PID):
			return 0, fmt.Errorf("locked by scenario %q", lock.Scenario)
		// Keep a short hold window for dead foreign owners so parallel restarts do not
		// immediately race each other into reclaiming the same port.
		case !lock.Timestamp.IsZero() && m.Now().UTC().Sub(lock.Timestamp) < staleLockWindow:
			return 0, fmt.Errorf("recent stale lock held by scenario %q", lock.Scenario)
		default:
			_ = m.removeLockIfMatches(lock)
		}
	}

	if pid := runtimeOwnerPID(records, port); pid > 0 {
		if err := m.WriteLock(port, scenarioName, pid); err != nil {
			return 0, fmt.Errorf("repair lock for port %d: %w", port, err)
		}
		return pid, nil
	}

	inUse, err := isTCPPortInUse(port)
	if err != nil {
		return 0, err
	}
	if inUse {
		return 0, errors.New("port already in use")
	}

	return os.Getpid(), nil
}

func reservedByResource(resourcePorts map[string]int, port int) bool {
	for _, reserved := range resourcePorts {
		if reserved == port {
			return true
		}
	}
	return false
}

func runtimeOwnerPID(records []process.Record, port int) int {
	for _, record := range process.LiveRecords(records) {
		if record.Port == port && record.PID > 0 {
			return record.PID
		}
	}
	return 0
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
