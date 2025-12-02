package bundleruntime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"scenario-to-desktop-runtime/manifest"
)

// exitCode extracts the exit code from an error, if available.
func exitCode(err error) *int {
	if err == nil {
		code := 0
		return &code
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		code := ee.ExitCode()
		return &code
	}
	return nil
}

// serviceProcess tracks a running service process.
type serviceProcess struct {
	proc     Process
	logPath  string
	logFile  File // log file handle for closing
	service  manifest.Service
	started  time.Time
	cancel   context.CancelFunc
	stopping bool
}

// launchServices starts all services in dependency order.
func (s *Supervisor) launchServices(ctx context.Context) error {
	order, err := topoSort(s.opts.Manifest.Services)
	if err != nil {
		return fmt.Errorf("dependency sort: %w", err)
	}

	for _, id := range order {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		svc := findService(s.opts.Manifest.Services, id)
		if svc == nil {
			continue
		}

		// Ensure dependencies are ready before starting.
		if err := s.waitForDependencies(ctx, svc); err != nil {
			s.setStatus(svc.ID, ServiceStatus{Ready: false, Message: err.Error()})
			_ = s.recordTelemetry("service_blocked", map[string]interface{}{
				"service_id": svc.ID,
				"reason":     err.Error(),
			})
			continue
		}

		if err := s.startService(ctx, *svc); err != nil {
			s.setStatus(svc.ID, ServiceStatus{Ready: false, Message: err.Error()})
			_ = s.recordTelemetry("service_start_failed", map[string]interface{}{
				"service_id": svc.ID,
				"error":      err.Error(),
			})
			continue
		}
	}
	return nil
}

// startService launches a single service and sets up monitoring.
func (s *Supervisor) startService(ctx context.Context, svc manifest.Service) error {
	if s.opts.DryRun {
		s.setStatus(svc.ID, ServiceStatus{Ready: true, Message: "dry-run"})
		return nil
	}

	bin, ok := s.opts.Manifest.ResolveBinary(svc)
	if !ok {
		return fmt.Errorf("resolve binary for service %s", svc.ID)
	}

	cmdPath := manifest.ResolvePath(s.opts.BundlePath, bin.Path)

	// Prepare directories.
	if err := s.prepareServiceDirs(svc); err != nil {
		return err
	}

	// Build environment.
	envMap, err := s.renderEnvMap(svc, bin)
	if err != nil {
		return err
	}

	// Apply secrets, GPU settings, and Playwright conventions.
	if err := s.applySecrets(envMap, svc); err != nil {
		return err
	}
	if err := s.applyPlaywrightConventions(svc, envMap); err != nil {
		return err
	}
	if err := s.applyGPURequirement(envMap, svc); err != nil {
		return err
	}

	// Verify assets.
	if err := s.ensureAssets(svc); err != nil {
		return err
	}

	// Run migrations.
	if err := s.runMigrations(ctx, svc, bin, envMap); err != nil {
		return err
	}

	// Create the command context.
	cmdCtx, cancel := context.WithCancel(ctx)
	args := s.renderArgs(bin.Args)

	// Determine working directory.
	workDir := s.opts.BundlePath
	if bin.CWD != "" {
		workDir = manifest.ResolvePath(s.opts.BundlePath, bin.CWD)
	}

	// Set up logging.
	logWriter, logPath, err := s.logWriter(svc)
	if err != nil {
		cancel()
		return err
	}
	defer func() {
		if err != nil && logWriter != nil {
			_ = logWriter.Close()
		}
	}()

	// Start the process using the injected ProcessRunner.
	proc, err := s.procRunner.Start(cmdCtx, cmdPath, args, envMapToList(envMap), workDir, logWriter, logWriter)
	if err != nil {
		cancel()
		return fmt.Errorf("start %s: %w", svc.ID, err)
	}

	svcProc := &serviceProcess{
		proc:    proc,
		logPath: logPath,
		logFile: logWriter,
		service: svc,
		started: s.clock.Now(),
		cancel:  cancel,
	}
	s.setProc(svc.ID, svcProc)
	s.setStatus(svc.ID, ServiceStatus{Ready: false, Message: "starting"})
	_ = s.recordTelemetry("service_start", map[string]interface{}{"service_id": svc.ID})

	// Monitor readiness in background.
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.waitForReadiness(cmdCtx, svc); err != nil {
			s.setStatus(svc.ID, ServiceStatus{Ready: false, Message: err.Error()})
			_ = s.recordTelemetry("service_not_ready", map[string]interface{}{
				"service_id": svc.ID,
				"error":      err.Error(),
			})
		} else {
			s.setStatus(svc.ID, ServiceStatus{Ready: true, Message: "ready"})
			_ = s.recordTelemetry("service_ready", map[string]interface{}{"service_id": svc.ID})
		}
	}()

	// Monitor for unexpected exits.
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		err := proc.Wait()
		code := exitCode(err)
		msg := "stopped"
		if err != nil {
			msg = err.Error()
		}
		s.setStatus(svc.ID, ServiceStatus{Ready: false, Message: msg, ExitCode: code})
		_ = s.recordTelemetry("service_exit", map[string]interface{}{
			"service_id": svc.ID,
			"exit_code":  code,
			"error":      msg,
		})
		// Close log file when process exits.
		if svcProc.logFile != nil {
			_ = svcProc.logFile.Close()
		}
	}()

	return nil
}

// stopServices stops all services in reverse dependency order.
func (s *Supervisor) stopServices(ctx context.Context) {
	order, err := topoSort(s.opts.Manifest.Services)
	if err != nil {
		return
	}

	// Stop in reverse dependency order.
	for i := len(order) - 1; i >= 0; i-- {
		id := order[i]
		proc := s.getProc(id)
		if proc == nil {
			continue
		}

		proc.stopping = true
		if proc.cancel != nil {
			proc.cancel()
		}
		if proc.proc != nil {
			s.gracefulStop(ctx, proc)
		}
	}
}

// gracefulStop attempts graceful shutdown, then forceful kill.
func (s *Supervisor) gracefulStop(ctx context.Context, proc *serviceProcess) {
	_ = proc.proc.Signal(Interrupt)

	waitCh := make(chan error, 1)
	go func() { waitCh <- proc.proc.Wait() }()

	select {
	case <-ctx.Done():
		_ = proc.proc.Kill()
	case <-waitCh:
		// Process exited normally.
	case <-s.clock.After(3 * time.Second):
		_ = proc.proc.Kill()
	}
}

// prepareServiceDirs creates required directories for a service.
func (s *Supervisor) prepareServiceDirs(svc manifest.Service) error {
	for _, dir := range svc.DataDirs {
		path := manifest.ResolvePath(s.appData, dir)
		if err := s.fs.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("create data dir for %s: %w", svc.ID, err)
		}
	}
	if svc.LogDir != "" {
		logPath := manifest.ResolvePath(s.appData, svc.LogDir)
		if err := s.fs.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
			return fmt.Errorf("prepare log dir: %w", err)
		}
		// Touch the log file to ensure it exists.
		f, err := s.fs.OpenFile(logPath, fileCreateAppend, 0o644)
		if err != nil {
			return fmt.Errorf("prepare log file: %w", err)
		}
		_ = f.Close()
	}
	return nil
}

// fileCreateAppend is the flag combination for os.O_CREATE|os.O_WRONLY|os.O_APPEND
var fileCreateAppend = os.O_CREATE | os.O_WRONLY | os.O_APPEND

// logWriter creates a log file writer for a service.
func (s *Supervisor) logWriter(svc manifest.Service) (File, string, error) {
	if svc.LogDir == "" {
		return nil, "", nil
	}
	logPath := manifest.ResolvePath(s.appData, svc.LogDir)
	f, err := s.fs.OpenFile(logPath, fileCreateAppend, 0o644)
	if err != nil {
		return nil, "", fmt.Errorf("open log file: %w", err)
	}
	return f, logPath, nil
}

// startServicesAsync initiates service startup in a goroutine.
// Only starts services if not already started and required secrets are available.
func (s *Supervisor) startServicesAsync() {
	s.mu.Lock()
	if s.servicesStarted {
		s.mu.Unlock()
		return
	}
	s.servicesStarted = true
	ctx := s.runtimeCtx
	s.mu.Unlock()

	if ctx == nil {
		ctx = context.Background()
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.launchServices(ctx); err != nil {
			_ = s.recordTelemetry("runtime_error", map[string]interface{}{"error": err.Error()})
		}
	}()
}
