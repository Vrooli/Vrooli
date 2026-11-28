package runtimesupervisor

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/envkit-go"
	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/repo-contract-go/cliinvoke"
	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/setpoint"
	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/internal/tuning"
)

func (s *Service) ensureOpened(ctx context.Context) error {
	if s.store != nil {
		return nil
	}
	hostProvider := s.cfg.HostProvider
	if hostProvider == nil {
		hostProvider = hostsession.DefaultProvider{}
	}
	host, err := hostProvider.Current(ctx, s.cfg.HomeDir)
	if err != nil {
		return fmt.Errorf("resolve host session: %w", err)
	}
	s.host = host
	factory := s.cfg.StoreFactory
	if factory == nil {
		factory = func(ctx context.Context, cfg scenarioruntime.Config) (Store, error) {
			return scenarioruntime.NewSQLiteStore(ctx, cfg)
		}
		s.ownsStore = true
	}
	store, err := factory(ctx, scenarioruntime.Config{HomeDir: s.cfg.HomeDir, DBPath: s.cfg.DBPath, Clock: s.cfg.Clock})
	if err != nil {
		return fmt.Errorf("open runtime registry: %w", err)
	}
	s.store = store
	return nil
}

func (s *Service) stopSession(ctx context.Context, status string, reason string) (scenarioruntime.SupervisorSession, error) {
	if s.store == nil || s.session.SupervisorID == "" {
		return scenarioruntime.SupervisorSession{}, nil
	}
	return s.store.StopSupervisorSession(ctx, s.session.SupervisorID, status, reason)
}

func (s *Service) now() time.Time {
	if s.cfg.Clock == nil {
		return time.Now().UTC()
	}
	return s.cfg.Clock.Now().UTC()
}

func (s *Service) pidRunning(pid int) bool {
	if s.cfg.PIDRunning != nil {
		return s.cfg.PIDRunning(pid)
	}
	return process.IsPIDRunning(pid)
}

// captureListenerSnapshotFn seams the per-tick snapshot capture so tests can
// pin the real tickPortListener branch (evidence conversion, capture-after-
// store-reads ordering) without reading the live host's TCP table.
var captureListenerSnapshotFn = network.CaptureTCPListenerSnapshot

// tickPortListener returns the per-tick listener evidence source: the
// configured override when present (tests), otherwise a closure over ONE
// freshly captured TCPListenerSnapshot shared by every port queried this
// tick.
func (s *Service) tickPortListener() PortListenerFunc {
	if s.cfg.PortListener != nil {
		return s.cfg.PortListener
	}
	snapshot := captureListenerSnapshotFn()
	return func(port int) scenarioruntime.ListenerEvidence {
		state := snapshot.Listening(port)
		if !state.Known {
			return scenarioruntime.ListenerEvidence{Known: false}
		}
		evidence := scenarioruntime.ListenerEvidence{Known: true, Listening: state.Listening}
		// Label provenance: listener_process_label previously carried the
		// per-PID `ps` command string; it now comes from /proc/<pid>/cmdline
		// (linux), the lsof comm name (darwin), or the process image path
		// (windows). Same information, new source; downstream only displays
		// it.
		if len(state.Listeners) > 0 {
			pid := state.Listeners[0].PID
			evidence.PID = &pid
			evidence.ProcessLabel = state.Listeners[0].Label
		}
		return evidence
	}
}

func (s *Service) executeHealthProbes(ctx context.Context, instanceIDs []string, instances map[string]scenarioruntime.Instance, claims map[string][]scenarioruntime.PortClaim) (int, error) {
	if len(instanceIDs) == 0 {
		return 0, nil
	}
	workers := normalizeMaxHealthConcurrency(s.cfg.MaxHealthConcurrency)
	if workers > len(instanceIDs) {
		workers = len(instanceIDs)
	}
	jobs := make(chan string)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	var completed int
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for instanceID := range jobs {
				if ctx.Err() != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = ctx.Err()
					}
					mu.Unlock()
					continue
				}
				instance, ok := instances[instanceID]
				if !ok {
					continue
				}
				snapshot := s.probeHealth(ctx, instance, claims[instanceID])
				if _, err := s.store.UpsertHealthSnapshot(ctx, snapshot); err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = fmt.Errorf("upsert health snapshot for %s: %w", instanceID, err)
					}
					mu.Unlock()
					continue
				}
				mu.Lock()
				completed++
				mu.Unlock()
			}
		}()
	}
	for _, instanceID := range instanceIDs {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return completed, ctx.Err()
		case jobs <- instanceID:
		}
	}
	close(jobs)
	wg.Wait()
	if firstErr != nil {
		return completed, firstErr
	}
	return completed, nil
}

func (s *Service) probeHealth(ctx context.Context, instance scenarioruntime.Instance, claims []scenarioruntime.PortClaim) scenarioruntime.HealthSnapshot {
	if s.cfg.HealthProber != nil {
		return s.cfg.HealthProber(ctx, instance, claims)
	}
	now := s.now()
	root := strings.TrimSpace(instance.ScopePath)
	if root == "" {
		return scenarioruntime.HealthSnapshot{
			InstanceID: instance.InstanceID,
			Scenario:   instance.Scenario,
			Status:     scenarioruntime.HealthStatusUnknown,
			CheckedAt:  &now,
			Error:      "runtime instance has no scope path for manifest health probing",
		}
	}
	item, err := scenario.Load(root, instance.Scenario, scenario.SandboxEnvFromEnv())
	if err != nil {
		return scenarioruntime.HealthSnapshot{
			InstanceID: instance.InstanceID,
			Scenario:   instance.Scenario,
			Status:     scenarioruntime.HealthStatusUnknown,
			CheckedAt:  &now,
			Error:      fmt.Sprintf("load scenario manifest: %v", err),
		}
	}
	// Health snapshots are shared by lifecycle reuse and status readers. Probe
	// the same owner-declared component profile as the running instance; using
	// the full manifest here would reintroduce checks for inactive ports (for
	// example UI_PORT on an API-only lab instance) on every supervisor tick.
	item, err = item.SelectProfile(instance.Variant)
	if err != nil {
		return scenarioruntime.HealthSnapshot{
			InstanceID: instance.InstanceID,
			Scenario:   instance.Scenario,
			Status:     scenarioruntime.HealthStatusUnknown,
			CheckedAt:  &now,
			Error:      fmt.Sprintf("select scenario profile for health probing: %v", err),
		}
	}
	return scenarioruntime.HealthProbe{Clock: s.cfg.Clock, AllowLegacyHTTP: true}.Probe(ctx, scenarioruntime.HealthProbeInput{
		InstanceID:            instance.InstanceID,
		Scenario:              instance.Scenario,
		HealthConfig:          item.Manifest.HealthConfig(),
		Ports:                 healthPortsFromClaims(item.Manifest, claims),
		ExpectedBuildIdentity: instance.BuildIdentity,
	})
}

func healthPortsFromClaims(manifest scenario.ServiceManifest, claims []scenarioruntime.PortClaim) map[string]int {
	ports := make(map[string]int)
	for _, claim := range claims {
		if claim.Port <= 0 {
			continue
		}
		envVar := strings.TrimSpace(claim.EnvVar)
		if envVar == "" {
			envVar = manifest.PortEnvVar(claim.PortName)
		}
		if envVar != "" {
			ports[envVar] = claim.Port
		}
	}
	return ports
}

func normalizeRenewInterval(v time.Duration) time.Duration {
	if v <= 0 {
		return DefaultRenewInterval
	}
	return v
}

func normalizeLeaseTTL(v time.Duration) time.Duration {
	if v <= 0 {
		return DefaultLeaseTTL
	}
	return v
}

func normalizeHealthInterval(v time.Duration) time.Duration {
	if v <= 0 {
		return DefaultHealthInterval
	}
	return v
}

func normalizeMaxHealthConcurrency(v int) int {
	if v <= 0 {
		return DefaultMaxHealthConcurrency
	}
	return v
}

func normalizeBatchSize(v int) int {
	if v <= 0 {
		return DefaultBatchSize
	}
	return v
}

func EnvConfig() Config {
	// The pressure bars come from the setpoint (internal/setpoint, the one
	// reader): memory PSI from its own cell, CPU PSI from the cell the
	// watchdog and autoheal grade. The environment overrides the file, not a
	// constant. A present file that fails validation falls back to the
	// compiled bars and is logged so a broken bar file is visible.
	cwd, _ := os.Getwd()
	bars, err := setpoint.Resolve(os.Environ(), cwd)
	if err != nil {
		bars = setpoint.Fallback()
		fmt.Fprintf(os.Stderr, "runtime supervisor: setpoint unreadable, using %s: %v\n", bars.Path, err)
	}
	pressureThreshold := floatEnv("VROOLI_RUNTIME_PRESSURE_SOME_AVG10", bars.Max(setpoint.CellMemoryPSI, 0))
	cpuPressureThreshold := floatEnv("VROOLI_RUNTIME_PRESSURE_CPU_SOME_AVG10", bars.Max(setpoint.CellCPUPressure, 0))
	return Config{
		RenewInterval:        durationEnv("VROOLI_RUNTIME_SUPERVISOR_RENEW_INTERVAL", DefaultRenewInterval),
		LeaseTTL:             durationEnv("VROOLI_RUNTIME_SUPERVISOR_LEASE_TTL", DefaultLeaseTTL),
		HealthInterval:       durationEnv("VROOLI_RUNTIME_SUPERVISOR_HEALTH_INTERVAL", DefaultHealthInterval),
		MaxHealthConcurrency: intEnv("VROOLI_RUNTIME_SUPERVISOR_MAX_HEALTH_CONCURRENCY", DefaultMaxHealthConcurrency),
		BatchSize:            intEnv("VROOLI_RUNTIME_SUPERVISOR_BATCH_SIZE", DefaultBatchSize),
		RecoveryQuietPeriod:  durationEnv("VROOLI_RUNTIME_RECOVERY_QUIET_PERIOD", DefaultRecoveryQuietPeriod),
		RecoveryCooldown:     durationEnv("VROOLI_RUNTIME_RECOVERY_COOLDOWN", DefaultRecoveryCooldown),
		RecoveryConcurrency:  intEnv("VROOLI_RUNTIME_RECOVERY_CONCURRENCY", DefaultRecoveryConcurrency),
		PressureSomeAvg10:    pressureThreshold,
		PressureProvider:     NewHostPressureProviderWithCPU(pressureThreshold, cpuPressureThreshold),
	}
}

func ModeFromEnv() string {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(ModeEnv))) {
	case ModeOff:
		return ModeOff
	case ModeOn:
		return ModeOn
	case ModeAuto:
		return ModeAuto
	default:
		return ModeAuto
	}
}

func EnsureRunning(ctx context.Context, cfg Config) error {
	mode := ModeFromEnv()
	if mode == ModeOff {
		return nil
	}
	svc := New(cfg)
	report, err := svc.Status(ctx)
	_ = svc.Close()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if cfg.Clock != nil {
		now = cfg.Clock.Now().UTC()
	}
	if report.Status == scenarioruntime.SupervisorStatusRunning && report.HeartbeatDeadlineAt.After(now) {
		return nil
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	// Prefer the installed service. Launching the supervisor ourselves makes it
	// a child of whatever ran this command, and on Linux it then lives in that
	// caller's cgroup: a supervisor started from a scenario's own service (or a
	// terminal multiplexer pane) dies whenever that unrelated thing restarts.
	// The service manager places it correctly and keeps restarting it.
	if started, err := platform.StartInstalledService(platform.ServiceInstallOptions{HomeDir: cfg.HomeDir, User: true}); err != nil {
		// Fall through to the direct launch: an unsupervised supervisor is
		// still better than none, and the failure is reported where the caller
		// can log it.
		fmt.Fprintf(orDiscard(cfg.Stderr), "warning: could not start the installed runtime supervisor service (%v); launching it directly\n", err)
	} else if started {
		return nil
	}

	exe, err := supervisorExecutable(cfg.HomeDir, cfg.Executable)
	if err != nil {
		return err
	}
	// A Go test binary is not a control-plane executable. Starting it with the
	// supervisor argv makes it run its test suite again; when those tests call
	// EnsureRunning this creates a detached recursive test storm. This matters
	// in particular when the installed service is unavailable, because the
	// fallback otherwise uses os.Executable().
	// This is a detached daemon launch, not a run-and-wait invocation, so it
	// starts the process and releases it instead of going through
	// cliinvoke.Run; the argv still comes from the shared catalog so the
	// invoker registry test parses exactly what runs here.
	cmd := shell.NewCommand(exe, cliinvoke.RuntimeSupervisorRun()...)
	cmd.Env = supervisorCommandEnv(os.Environ(), cfg.HomeDir)
	// The spawned supervisor outlives this process, so inheriting the caller's
	// streams is worse than useless: writes land on the stderr of a CLI that
	// has already exited, and io.Discard loses the reason it died. Give it the
	// same log file the installed service writes to, opened as a real *os.File
	// so the child inherits the descriptor instead of depending on a copying
	// goroutine that dies with us.
	logFile, logErr := openSupervisorLog(cfg.HomeDir)
	if logErr != nil && cfg.Stderr != nil {
		fmt.Fprintf(cfg.Stderr, "warning: runtime supervisor log unavailable (%v); its output will be discarded\n", logErr)
	}
	if logFile != nil {
		defer logFile.Close()
	}
	cmd.Stdout = firstNonNilWriter(cfg.Stdout, logFile)
	cmd.Stderr = firstNonNilWriter(cfg.Stderr, logFile)
	if err := platform.ConfigureCommand(cmd, platform.ProcessOptions{Detached: true}); err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start runtime supervisor: %w", err)
	}
	return cmd.Process.Release()
}

func supervisorExecutable(homeDir, requested string) (string, error) {
	executable, _, err := ExecutablePath(homeDir, requested)
	if err != nil {
		return "", fmt.Errorf("resolve vrooli executable for runtime supervisor: %w", err)
	}
	if isGoTestBinary(executable) {
		return "", fmt.Errorf("refuse to launch runtime supervisor from Go test binary %q", executable)
	}
	return executable, nil
}

func isGoTestBinary(path string) bool {
	// Normalize the alternate separator so the predicate remains testable and
	// correct when a Windows-style path is supplied by a cross-platform caller.
	name := strings.ToLower(filepath.Base(strings.ReplaceAll(strings.TrimSpace(path), "\\", "/")))
	return strings.HasSuffix(name, ".test") || strings.HasSuffix(name, ".test.exe")
}

// orDiscard keeps optional diagnostics from panicking on an unset writer.
func orDiscard(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}

// openSupervisorLog appends to the supervisor's log file, creating the log
// directory when needed. A nil file (with the error) means the caller falls
// back to discarding output rather than failing the start.
func openSupervisorLog(homeDir string) (*os.File, error) {
	path, err := LogPath(homeDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), tuning.PermDir); err != nil {
		return nil, fmt.Errorf("create runtime supervisor log dir: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, tuning.PermFile)
	if err != nil {
		return nil, fmt.Errorf("open runtime supervisor log: %w", err)
	}
	return file, nil
}

// firstNonNilWriter prefers an explicitly configured writer, falls back to the
// log file, and discards only when neither exists. The typed-nil check matters:
// a nil *os.File in an io.Writer is not a nil interface.
func firstNonNilWriter(configured io.Writer, logFile *os.File) io.Writer {
	if configured != nil {
		return configured
	}
	if logFile != nil {
		return logFile
	}
	return io.Discard
}

func supervisorCommandEnv(env []string, home string) []string {
	overlay := envkit.Env{ModeEnv + "=" + ModeOn}
	if strings.TrimSpace(home) != "" {
		overlay = append(overlay, "HOME="+home)
	}
	return envkit.WithOverlay(envkit.Env(env), envkit.SameScenario, overlay)
}

func durationEnv(name string, fallback time.Duration) time.Duration {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func intEnv(name string, fallback int) int {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func floatEnv(name string, fallback float64) float64 {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

// runningUnderNativeUnit reports whether the native service manager started
// this process: systemd sets INVOCATION_ID for every unit it runs, and the
// rendered supervisor unit sets VROOLI_RUNTIME_SUPERVISOR=on.
func runningUnderNativeUnit() bool {
	return os.Getenv("INVOCATION_ID") != "" && os.Getenv("VROOLI_RUNTIME_SUPERVISOR") == "on"
}
