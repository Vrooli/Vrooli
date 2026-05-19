package targetruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"test-genie/internal/shared"
)

// Needs declares which target services a phase plan requires.
type Needs struct {
	UI  bool
	API bool
}

// URLs contains the runtime endpoints resolved for a target scenario.
type URLs struct {
	UI  string
	API string
}

// Lease records whether Test Genie started the target scenario for this run.
type Lease struct {
	URLs    URLs
	Started bool
}

type (
	CommandRunner func(ctx context.Context, dir string, env map[string]string, logWriter io.Writer, name string, args ...string) error
	PortProbe     func(ctx context.Context, port int) bool
	PIDProbe      func(pid int) bool
)

// Manager owns lifecycle operations for the scenario under test.
type Manager struct {
	Name         string
	ScenarioDir  string
	Home         string
	StartTimeout time.Duration
	PollInterval time.Duration

	runCommand CommandRunner
	portOpen   PortProbe
	pidAlive   PIDProbe
}

// New creates a target runtime manager for one scenario.
func New(name, scenarioDir string) *Manager {
	home, _ := os.UserHomeDir()
	return &Manager{
		Name:         strings.TrimSpace(name),
		ScenarioDir:  filepath.Clean(strings.TrimSpace(scenarioDir)),
		Home:         home,
		StartTimeout: 2 * time.Minute,
		PollInterval: 500 * time.Millisecond,
		runCommand:   defaultCommandRunner,
		portOpen:     isPortOpen,
		pidAlive:     isPIDAlive,
	}
}

// WithCommandRunner overrides command execution for tests.
func (m *Manager) WithCommandRunner(run CommandRunner) *Manager {
	if run != nil {
		m.runCommand = run
	}
	return m
}

// WithHome overrides the Vrooli home directory for tests.
func (m *Manager) WithHome(home string) *Manager {
	if strings.TrimSpace(home) != "" {
		m.Home = filepath.Clean(home)
	}
	return m
}

// WithProbes overrides runtime probes for tests.
func (m *Manager) WithProbes(portOpen PortProbe, pidAlive PIDProbe) *Manager {
	if portOpen != nil {
		m.portOpen = portOpen
	}
	if pidAlive != nil {
		m.pidAlive = pidAlive
	}
	return m
}

// EnsureRunning starts the target scenario when the requested runtime URLs are
// not already available. If it starts the target, callers must Cleanup the lease.
func (m *Manager) EnsureRunning(ctx context.Context, needs Needs, logWriter io.Writer) (Lease, error) {
	if !needs.UI && !needs.API {
		return Lease{}, nil
	}
	if err := m.validate(); err != nil {
		return Lease{}, err
	}

	if urls, ok := m.resolveURLs(ctx, needs); ok {
		return Lease{URLs: urls}, nil
	}

	shared.LogStep(logWriter, "starting target scenario %s", m.Name)
	if err := m.runLifecycle(ctx, nil, logWriter, "start"); err != nil {
		return Lease{}, fmt.Errorf("start target scenario %s: %w", m.Name, err)
	}

	urls, err := m.waitForURLs(ctx, needs)
	if err != nil {
		return Lease{}, err
	}
	return Lease{URLs: urls, Started: true}, nil
}

// Cleanup stops the target scenario only when Test Genie started it.
func (m *Manager) Cleanup(ctx context.Context, lease Lease, logWriter io.Writer) error {
	if !lease.Started {
		return nil
	}
	shared.LogStep(logWriter, "stopping target scenario %s", m.Name)
	return m.runLifecycle(ctx, nil, logWriter, "stop")
}

// RestartWithEnv restarts the target scenario with temporary environment
// overrides, usually for playbooks isolation resources.
func (m *Manager) RestartWithEnv(ctx context.Context, env map[string]string, logWriter io.Writer) error {
	if err := m.validate(); err != nil {
		return err
	}
	shared.LogStep(logWriter, "restarting target scenario %s with isolated resources", m.Name)
	return m.runLifecycle(ctx, env, logWriter, "restart")
}

// Restore restarts the target scenario with the normal environment.
func (m *Manager) Restore(ctx context.Context, logWriter io.Writer) error {
	if err := m.validate(); err != nil {
		return err
	}
	shared.LogStep(logWriter, "restoring target scenario %s runtime", m.Name)
	return m.runLifecycle(ctx, nil, logWriter, "restart")
}

func (m *Manager) validate() error {
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("target scenario name is required")
	}
	if strings.TrimSpace(m.Home) == "" {
		return fmt.Errorf("vrooli home directory is required")
	}
	return nil
}

func (m *Manager) runLifecycle(ctx context.Context, env map[string]string, logWriter io.Writer, action string) error {
	args := []string{"scenario", action, m.Name, "--clean-stale"}
	if action == "stop" {
		args = []string{"scenario", "stop", m.Name}
	}
	if action != "stop" && strings.TrimSpace(m.ScenarioDir) != "" {
		args = append(args, "--path", m.ScenarioDir)
	}
	return m.runCommand(ctx, "", env, logWriter, "vrooli", args...)
}

func (m *Manager) waitForURLs(ctx context.Context, needs Needs) (URLs, error) {
	timeout := m.StartTimeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	interval := m.PollInterval
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		urls, ok := m.resolveURLs(waitCtx, needs)
		if ok {
			return urls, nil
		}
		select {
		case <-waitCtx.Done():
			return URLs{}, fmt.Errorf("timeout waiting for target scenario %s runtime URLs: %w", m.Name, waitCtx.Err())
		case <-ticker.C:
		}
	}
}

func (m *Manager) resolveURLs(ctx context.Context, needs Needs) (URLs, bool) {
	records, err := m.readRecords()
	if err != nil {
		return URLs{}, false
	}

	urls := URLs{}
	for _, record := range records {
		if record.Port <= 0 || !m.pidAlive(record.PID) || !m.portOpen(ctx, record.Port) {
			continue
		}
		step := strings.ToLower(record.Step)
		if urls.UI == "" && strings.Contains(step, "ui") {
			urls.UI = fmt.Sprintf("http://127.0.0.1:%d", record.Port)
		}
		if urls.API == "" && strings.Contains(step, "api") {
			urls.API = fmt.Sprintf("http://127.0.0.1:%d", record.Port)
		}
	}
	if needs.UI && urls.UI == "" {
		return URLs{}, false
	}
	if needs.API && urls.API == "" {
		return URLs{}, false
	}
	return urls, true
}

type processRecord struct {
	PID  int    `json:"pid"`
	Port int    `json:"port"`
	Step string `json:"step"`
}

func (m *Manager) readRecords() ([]processRecord, error) {
	processDir := filepath.Join(m.Home, ".vrooli", "processes", "scenarios", m.Name)
	files, err := filepath.Glob(filepath.Join(processDir, "*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	records := make([]processRecord, 0, len(files))
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		var record processRecord
		if err := json.Unmarshal(data, &record); err != nil {
			return nil, err
		}
		if record.Step == "" {
			record.Step = strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		}
		records = append(records, record)
	}
	return records, nil
}

func defaultCommandRunner(ctx context.Context, dir string, env map[string]string, logWriter io.Writer, name string, args ...string) error {
	cmdArgs := args
	if name == "vrooli" && (len(args) == 0 || args[0] != "--no-stale-check") {
		cmdArgs = append([]string{"--no-stale-check"}, args...)
	}
	cmd := exec.CommandContext(ctx, name, cmdArgs...)
	if strings.TrimSpace(dir) != "" {
		cmd.Dir = dir
	}
	cmd.Env = commandEnvironment(env)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	return cmd.Run()
}

func commandEnvironment(overrides map[string]string) []string {
	blocked := map[string]struct{}{
		"API_PORT": {},
		"UI_PORT":  {},
	}
	result := make([]string, 0, len(os.Environ())+len(overrides))
	for _, item := range os.Environ() {
		key, _, ok := strings.Cut(item, "=")
		if ok {
			if _, skip := blocked[key]; skip {
				continue
			}
		}
		result = append(result, item)
	}
	for key, value := range overrides {
		result = append(result, key+"="+value)
	}
	return result
}

func isPortOpen(ctx context.Context, port int) bool {
	dialer := net.Dialer{Timeout: 500 * time.Millisecond}
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func isPIDAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}
