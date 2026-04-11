package ports

import (
	"bufio"
	"errors"
	"fmt"
	"hash/crc32"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
)

const staleLockWindow = 5 * time.Minute

var resourcePortPattern = regexp.MustCompile(`\["([^"]+)"\]="([0-9]+)"`)

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
	resourcePorts, err := loadResourcePorts(root)
	if err != nil {
		return nil, err
	}
	return &Manager{
		Root:          filepath.Clean(root),
		Home:          filepath.Clean(home),
		Now:           time.Now,
		ResourcePorts: resourcePorts,
	}, nil
}

func loadResourcePorts(root string) (map[string]int, error) {
	path := filepath.Join(root, "scripts", "resources", "port_registry.sh")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	ports := make(map[string]int)
	for _, match := range resourcePortPattern.FindAllStringSubmatch(string(data), -1) {
		port, err := strconv.Atoi(match[2])
		if err != nil {
			continue
		}
		ports[match[1]] = port
	}
	return ports, nil
}

func (m *Manager) StateDir() string {
	return process.ScenarioStateDir(m.Home)
}

func (m *Manager) lockPath(port int) string {
	return filepath.Join(m.StateDir(), fmt.Sprintf(".port_%d.lock", port))
}

func (m *Manager) EnsureStateDir() error {
	return os.MkdirAll(m.StateDir(), 0o755)
}

func (m *Manager) ReadLock(port int) (Lock, bool, error) {
	path := m.lockPath(port)
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
	if err := m.EnsureStateDir(); err != nil {
		return err
	}
	content := fmt.Sprintf("%s:%d:%d\n", scenarioName, pid, m.Now().Unix())
	return os.WriteFile(m.lockPath(port), []byte(content), 0o644)
}

func (m *Manager) claimLock(port int, scenarioName string, pid int) error {
	if err := m.EnsureStateDir(); err != nil {
		return err
	}
	path := m.lockPath(port)

	if lock, exists, err := m.ReadLock(port); err != nil {
		return err
	} else if exists {
		if lock.Scenario != "" && lock.Scenario != scenarioName {
			return fmt.Errorf("port %d locked by scenario %q", port, lock.Scenario)
		}
		return m.WriteLock(port, scenarioName, pid)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if os.IsExist(err) {
			lock, exists, readErr := m.ReadLock(port)
			if readErr != nil {
				return readErr
			}
			if exists && (lock.Scenario == "" || lock.Scenario == scenarioName) {
				return m.WriteLock(port, scenarioName, pid)
			}
			return fmt.Errorf("port %d locked by scenario %q", port, lock.Scenario)
		}
		return err
	}
	defer file.Close()

	if _, err := fmt.Fprintf(file, "%s:%d:%d\n", scenarioName, pid, m.Now().Unix()); err != nil {
		return err
	}
	return nil
}

func (m *Manager) RemoveLock(port int) error {
	if err := os.Remove(m.lockPath(port)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (m *Manager) RemoveScenarioLocks(scenarioName string) error {
	locks, err := m.LocksForScenario(scenarioName)
	if err != nil {
		return err
	}
	for _, lock := range locks {
		if err := m.RemoveLock(lock.Port); err != nil {
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

	for resourceName, dep := range item.Manifest.Dependencies.Resources {
		if !dep.Enabled && !dep.Required {
			continue
		}
		if port, ok := m.ResourcePorts[resourceName]; ok {
			envVars[strings.ToUpper(strings.ReplaceAll(resourceName, "-", "_"))+"_PORT"] = strconv.Itoa(port)
		}
	}

	resourceEnv, err := m.loadResourceEnvironment(item.Manifest)
	if err != nil {
		return Environment{}, err
	}
	for key, value := range resourceEnv {
		envVars[key] = value
	}

	m.applyPostgresOverride(item.Slug, item.Manifest, envVars)

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
			_ = m.RemoveLock(port)
		case lock.PID > 0 && process.IsPIDRunning(lock.PID):
			return 0, fmt.Errorf("locked by scenario %q", lock.Scenario)
		case !lock.Timestamp.IsZero() && m.Now().UTC().Sub(lock.Timestamp) < staleLockWindow:
			return 0, fmt.Errorf("recent stale lock held by scenario %q", lock.Scenario)
		default:
			_ = m.RemoveLock(port)
		}
	}

	if pid := runtimeOwnerPID(records, port); pid > 0 {
		_ = m.WriteLock(port, scenarioName, pid)
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

func (m *Manager) loadResourceEnvironment(manifest scenario.ServiceManifest) (map[string]string, error) {
	env := make(map[string]string)
	for resourceName, dep := range manifest.Dependencies.Resources {
		if !dep.Enabled && !dep.Required {
			continue
		}

		loaded, err := m.loadResourceExports(resourceName)
		if err != nil {
			return nil, err
		}
		for key, value := range loaded {
			env[key] = value
		}
	}
	return env, nil
}

func (m *Manager) loadResourceExports(resourceName string) (map[string]string, error) {
	configDir := filepath.Join(m.Root, "resources", resourceName, "config")
	exportsFile := filepath.Join(configDir, "exports.sh")
	if _, err := os.Stat(exportsFile); os.IsNotExist(err) {
		exportsFile = filepath.Join(configDir, "defaults.sh")
	}
	if _, err := os.Stat(exportsFile); os.IsNotExist(err) {
		return map[string]string{}, nil
	} else if err != nil {
		return nil, err
	}

	prefix := strings.ToUpper(strings.ReplaceAll(resourceName, "-", "_")) + "_"
	script := fmt.Sprintf(`
set -e
export APP_ROOT=%s
export VROOLI_ROOT=%s
export HOME=%s
export DEBUG=false
if [ -f %s ]; then
  source %s >/dev/null 2>&1 || true
fi
env | sort
`, shellQuote(m.Root), shellQuote(m.Root), shellQuote(m.Home), shellQuote(exportsFile), shellQuote(exportsFile))

	cmd := exec.Command("bash", "-lc", script)
	cmd.Dir = m.Root
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return map[string]string{}, nil
		}
		return nil, err
	}

	result := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		result[parts[0]] = parts[1]
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func (m *Manager) applyPostgresOverride(scenarioName string, manifest scenario.ServiceManifest, envVars map[string]string) {
	dep, ok := manifest.Dependencies.Resources["postgres"]
	if !ok || (!dep.Enabled && !dep.Required) {
		return
	}

	dbName := strings.TrimSpace(dep.Database)
	if dbName == "" {
		dbName = "vrooli_" + strings.ReplaceAll(scenarioName, "-", "_")
	}

	host := defaultString(envVars["POSTGRES_HOST"], "localhost")
	port := defaultString(envVars["POSTGRES_PORT"], "5433")
	user := defaultString(envVars["POSTGRES_USER"], "vrooli")
	password := envVars["POSTGRES_PASSWORD"]
	sslmode := defaultString(envVars["POSTGRES_SSLMODE"], "disable")

	url := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, password, host, port, dbName, sslmode)
	envVars["POSTGRES_DB"] = dbName
	envVars["POSTGRES_URL"] = url
	envVars["DATABASE_URL"] = url
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
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
