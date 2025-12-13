package bundleruntime

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"scenario-to-desktop-runtime/assets"
	"scenario-to-desktop-runtime/deps"
	"scenario-to-desktop-runtime/manifest"
	"scenario-to-desktop-runtime/strutil"
)

// =============================================================================
// Asset Management (uses cached verifier)
// =============================================================================

// ensureAssets verifies all required assets for a service exist and are valid.
func (s *Supervisor) ensureAssets(svc manifest.Service) error {
	return s.assetVerifier.EnsureAssets(svc)
}

// =============================================================================
// Playwright Conventions (delegates to assets package)
// =============================================================================

// applyPlaywrightConventions sets up environment variables for Playwright-based services.
func (s *Supervisor) applyPlaywrightConventions(svc manifest.Service, env map[string]string) error {
	cfg := assets.PlaywrightConfig{
		BundlePath: s.opts.BundlePath,
		FS:         s.fs,
		EnvReader:  s.envReader,
		Ports:      s.portAllocator,
		Telemetry:  s.telemetry,
	}
	return assets.ApplyPlaywrightConventions(cfg, svc, env)
}

// =============================================================================
// GPU Requirement (uses cached applier)
// =============================================================================

// applyGPURequirement enforces GPU requirements for a service.
func (s *Supervisor) applyGPURequirement(env map[string]string, svc manifest.Service) error {
	return s.gpuApplier.Apply(env, svc)
}

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
	order, err := deps.TopoSort(s.opts.Manifest.Services)
	if err != nil {
		return fmt.Errorf("dependency sort: %w", err)
	}

	for _, id := range order {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		svc := deps.FindService(s.opts.Manifest.Services, id)
		if svc == nil {
			continue
		}

		// Ensure dependencies are ready before starting.
		if err := s.healthChecker.WaitForDependencies(ctx, svc); err != nil {
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

	// Serve static UI bundles without requiring an executable.
	if strings.EqualFold(svc.Type, "ui-bundle") {
		return s.startUIBundleService(ctx, svc)
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
	proc, err := s.procRunner.Start(cmdCtx, cmdPath, args, strutil.EnvMapToList(envMap), workDir, logWriter, logWriter)
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
		if err := s.healthChecker.WaitForReadiness(cmdCtx, svc.ID); err != nil {
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

// startUIBundleService serves UI assets from the bundle using an embedded static server.
func (s *Supervisor) startUIBundleService(ctx context.Context, svc manifest.Service) error {
	if err := s.prepareServiceDirs(svc); err != nil {
		return err
	}

	// Resolve port (default to "ui", otherwise first requested port)
	portName := "ui"
	if svc.Health.PortName != "" {
		portName = svc.Health.PortName
	} else if svc.Readiness.PortName != "" {
		portName = svc.Readiness.PortName
	} else if svc.Ports != nil && len(svc.Ports.Requested) > 0 {
		portName = svc.Ports.Requested[0].Name
	}
	port, err := s.portAllocator.Resolve(svc.ID, portName)
	if err != nil {
		return fmt.Errorf("allocate port for %s: %w", svc.ID, err)
	}

	// Determine asset root from first asset entry.
	if len(svc.Assets) == 0 || svc.Assets[0].Path == "" {
		return fmt.Errorf("ui-bundle %s has no assets defined", svc.ID)
	}
	assetRoot := filepath.Dir(svc.Assets[0].Path)
	if assetRoot == "" || assetRoot == "." {
		assetRoot = filepath.Dir(svc.Assets[0].Path)
	}
	serveRoot := manifest.ResolvePath(s.opts.BundlePath, assetRoot)

	// Verify assets exist.
	if err := s.ensureAssets(svc); err != nil {
		return err
	}

	// Try to resolve API service port for proxying /api and /ws.
	apiPort := s.resolveAPIPort()
	var apiProxy *httputil.ReverseProxy
	if apiPort > 0 {
		target, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", apiPort))
		apiProxy = httputil.NewSingleHostReverseProxy(target)
	}

	logWriter, logPath, err := s.logWriter(svc)
	if err != nil {
		return err
	}

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return fmt.Errorf("listen on %d: %w", port, err)
	}

	// SPA-friendly file server: serve files when they exist; otherwise fallback to index.html for SPA routes.
	fileServer := http.FileServer(http.Dir(serveRoot))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Proxy API/WS to the backend service if available.
		if apiProxy != nil && (strings.HasPrefix(r.URL.Path, "/api") || strings.HasPrefix(strings.ToLower(r.URL.Path), "/ws")) {
			apiProxy.ServeHTTP(w, r)
			return
		}

		path := filepath.Join(serveRoot, filepath.Clean(r.URL.Path))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}
		// Fallback to index.html for SPA routes only when index exists.
		indexPath := filepath.Join(serveRoot, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			http.ServeFile(w, r, indexPath)
			return
		}
		http.NotFound(w, r)
	})

	server := &http.Server{Handler: handler}
	serverCtx, cancel := context.WithCancel(ctx)

	// Track service for shutdown.
	svcProc := &serviceProcess{
		proc:    nil, // managed in-process
		logPath: logPath,
		logFile: logWriter,
		service: svc,
		started: s.clock.Now(),
		cancel:  cancel,
	}
	s.setProc(svc.ID, svcProc)

	// Start serving.
	go func() {
		_ = server.Serve(ln) // server shutdown errors are ignored here
	}()

	// Shutdown on context cancel.
	go func() {
		<-serverCtx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	s.setStatus(svc.ID, ServiceStatus{Ready: true, Message: fmt.Sprintf("listening on %d", port)})
	_ = s.recordTelemetry("service_ready", map[string]interface{}{
		"service_id": svc.ID,
		"port":       port,
		"type":       "ui-bundle",
	})

	return nil
}

// resolveAPIPort attempts to find a service that exposes an "api" port and returns its allocated value.
func (s *Supervisor) resolveAPIPort() int {
	ports := s.portAllocator.Map()

	// Prefer a service with ID containing "-api".
	for svcID, entries := range ports {
		if strings.Contains(strings.ToLower(svcID), "-api") {
			if port, ok := entries["api"]; ok {
				return port
			}
		}
	}

	// Otherwise, return the first service that exposes "api".
	for _, entries := range ports {
		if port, ok := entries["api"]; ok {
			return port
		}
	}
	return 0
}

// stopServices stops all services in reverse dependency order.
func (s *Supervisor) stopServices(ctx context.Context) {
	order, err := deps.TopoSort(s.opts.Manifest.Services)
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

// LogWriter implements migrations.LogProvider.
func (s *Supervisor) LogWriter(svc manifest.Service) (File, string, error) {
	return s.logWriter(svc)
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
