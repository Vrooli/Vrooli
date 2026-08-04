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

// NoOp is the typed runtime implementation for repository targets that are
// source trees rather than deployable scenarios. It deliberately succeeds for
// lifecycle calls while retaining an operator-visible reason for the no-op.
type NoOp struct{ reason string }

func NewNoOp(reason string) *NoOp { return &NoOp{reason: strings.TrimSpace(reason)} }
func (n *NoOp) Reason() string {
	if n == nil || n.reason == "" {
		return "target has no deployable runtime"
	}
	return n.reason
}
func (n *NoOp) RestartWithEnv(context.Context, map[string]string, io.Writer) error { return nil }
func (n *NoOp) Restore(context.Context, io.Writer) error                           { return nil }

type (
	CommandRunner func(ctx context.Context, dir string, env map[string]string, logWriter io.Writer, name string, args ...string) error
	PortProbe     func(ctx context.Context, port int) bool
	PIDProbe      func(pid int) bool
)

const maxLifecycleDiagnosticBytes = 8 * 1024

// lifecycleDiagnostics retains the tail of lifecycle command output. Startup
// failures frequently put the actionable compiler or configuration error last;
// keeping that tail lets durable suite results name the failed contract instead
// of reducing it to an exit status.
type lifecycleDiagnostics struct {
	data      []byte
	truncated bool
}

func (d *lifecycleDiagnostics) Write(p []byte) (int, error) {
	written := len(p)
	if written == 0 {
		return 0, nil
	}
	if written >= maxLifecycleDiagnosticBytes {
		d.data = append(d.data[:0], p[written-maxLifecycleDiagnosticBytes:]...)
		d.truncated = true
		return written, nil
	}
	overflow := len(d.data) + written - maxLifecycleDiagnosticBytes
	if overflow > 0 {
		d.data = append(d.data[:0], d.data[overflow:]...)
		d.truncated = true
	}
	d.data = append(d.data, p...)
	return written, nil
}

func (d *lifecycleDiagnostics) String() string {
	detail := strings.TrimSpace(string(d.data))
	if detail == "" {
		return ""
	}
	if d.truncated {
		return "[last 8 KiB] " + detail
	}
	return detail
}

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

// LiveSurfaces reports the URLs of the target's currently-live surfaces. An
// empty field means that surface is not live. Unlike EnsureRunning, it never
// starts anything — it is a pure read of the lifecycle process records. The
// self-host guard uses it to reuse a live surface instead of clobbering the
// running process with `scenario start --clean-stale`.
func (m *Manager) LiveSurfaces(ctx context.Context) URLs {
	urls, _ := m.resolveURLs(ctx, Needs{})
	return urls
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
	var diagnostics lifecycleDiagnostics
	writer := io.Writer(&diagnostics)
	if logWriter != nil {
		writer = io.MultiWriter(logWriter, &diagnostics)
	}
	if err := m.runCommand(ctx, "", env, writer, "vrooli", args...); err != nil {
		if detail := diagnostics.String(); detail != "" {
			return fmt.Errorf("%w; lifecycle %s output: %s", err, action, detail)
		}
		return err
	}
	return nil
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
