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

	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/assets"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/deps"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
	resourceplan "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/resources"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/strutil"
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

		if skip, reason := shouldSkipService(*svc); skip {
			status := ServiceStatus{Ready: false, Skipped: true}
			if reason != "" {
				status.Message = fmt.Sprintf("skipped: %s", reason)
			} else {
				status.Message = "skipped: on-demand service"
			}
			s.setStatus(svc.ID, status)
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

func shouldSkipService(svc manifest.Service) (bool, string) {
	if len(svc.Metadata) == 0 {
		return false, ""
	}
	runMode, ok := svc.Metadata["run_mode"].(string)
	if !ok || !strings.EqualFold(runMode, "on_demand") {
		return false, ""
	}
	if reason, ok := svc.Metadata["skip_reason"].(string); ok && strings.TrimSpace(reason) != "" {
		return true, reason
	}
	return true, ""
}

// startService launches a single service and sets up monitoring.
func (s *Supervisor) startService(ctx context.Context, svc manifest.Service) error {
	if s.opts.DryRun {
		s.setStatus(svc.ID, ServiceStatus{Ready: true, Message: "dry-run"})
		return nil
	}

	if strings.EqualFold(svc.Type, "ui-bundle") {
		return s.startUIBundleService(ctx, svc)
	}

	bin, ok := s.opts.Manifest.ResolveBinary(svc)
	if !ok {
		return fmt.Errorf("resolve binary for service %s", svc.ID)
	}

	envMap, err := s.prepareServiceEnv(ctx, svc, bin)
	if err != nil {
		return err
	}

	cmdPath := manifest.ResolvePath(s.opts.BundlePath, bin.Path)
	cmdCtx, cancel := context.WithCancel(ctx)
	args := s.renderArgs(bin.Args)

	workDir := s.opts.BundlePath
	if bin.CWD != "" {
		workDir = manifest.ResolvePath(s.opts.BundlePath, bin.CWD)
	}

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

	s.monitorReadiness(cmdCtx, svc.ID, nil)
	s.monitorExit(svcProc, proc)

	return nil
}

// prepareServiceEnv handles all pre-launch preparation: dirs, env, secrets, assets, migrations.
func (s *Supervisor) prepareServiceEnv(ctx context.Context, svc manifest.Service, bin manifest.Binary) (map[string]string, error) {
	if err := s.prepareServiceDirs(svc); err != nil {
		return nil, err
	}

	envMap, err := s.renderEnvMap(svc, bin)
	if err != nil {
		return nil, err
	}
	if s.resourceServer != nil {
		for key, value := range s.resourceServer.Environment() {
			if existing, exists := envMap[key]; exists && existing != value {
				return nil, fmt.Errorf("shared managed resource environment conflicts with service %s variable %s", svc.ID, key)
			}
			envMap[key] = value
		}
	}

	if err := s.applySecrets(envMap, svc); err != nil {
		return nil, err
	}
	if err := s.applyPlaywrightConventions(svc, envMap); err != nil {
		return nil, err
	}
	if err := s.applyGPURequirement(envMap, svc); err != nil {
		return nil, err
	}

	if err := s.ensureAssets(svc); err != nil {
		return nil, err
	}

	if err := s.runMigrations(ctx, svc, bin, envMap); err != nil {
		return nil, err
	}

	return envMap, nil
}

// monitorReadiness watches for service readiness in a background goroutine.
// extraFields are appended to telemetry events.
func (s *Supervisor) monitorReadiness(ctx context.Context, svcID string, extraFields map[string]interface{}) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.healthChecker.WaitForReadiness(ctx, svcID); err != nil {
			s.setStatus(svcID, ServiceStatus{Ready: false, Message: err.Error()})
			fields := map[string]interface{}{"service_id": svcID, "error": err.Error()}
			for k, v := range extraFields {
				fields[k] = v
			}
			_ = s.recordTelemetry("service_not_ready", fields)
			return
		}
		s.setStatus(svcID, ServiceStatus{Ready: true, Message: "ready"})
		fields := map[string]interface{}{"service_id": svcID}
		for k, v := range extraFields {
			fields[k] = v
		}
		_ = s.recordTelemetry("service_ready", fields)
	}()
}

// monitorExit watches for unexpected process exits in a background goroutine.
func (s *Supervisor) monitorExit(svcProc *serviceProcess, proc Process) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		err := proc.Wait()
		code := exitCode(err)
		msg := "stopped"
		if err != nil {
			msg = err.Error()
		}
		s.setStatus(svcProc.service.ID, ServiceStatus{Ready: false, Message: msg, ExitCode: code})
		_ = s.recordTelemetry("service_exit", map[string]interface{}{
			"service_id": svcProc.service.ID,
			"exit_code":  code,
			"error":      msg,
		})
		if svcProc.logFile != nil {
			_ = svcProc.logFile.Close()
		}
	}()
}

// startUIBundleService serves UI assets from the bundle using an embedded static server.
func (s *Supervisor) startUIBundleService(ctx context.Context, svc manifest.Service) error {
	if err := s.prepareServiceDirs(svc); err != nil {
		return err
	}

	port, err := s.resolveUIPort(svc)
	if err != nil {
		return err
	}

	serveRoot, err := s.resolveUIDistRoot(svc)
	if err != nil {
		return err
	}

	if err := s.ensureAssets(svc); err != nil {
		return err
	}

	logWriter, logPath, err := s.logWriter(svc)
	if err != nil {
		return err
	}

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return fmt.Errorf("listen on %d: %w", port, err)
	}

	handler := s.buildUIHandler(svc, serveRoot)
	server := &http.Server{Handler: handler}
	serverCtx, cancel := context.WithCancel(ctx)

	svcProc := &serviceProcess{
		proc:    nil, // managed in-process
		logPath: logPath,
		logFile: logWriter,
		service: svc,
		started: s.clock.Now(),
		cancel:  cancel,
	}
	s.setProc(svc.ID, svcProc)

	go func() {
		_ = server.Serve(ln)
	}()
	go func() {
		<-serverCtx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer shutdownCancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	s.setStatus(svc.ID, ServiceStatus{Ready: false, Message: "starting"})
	_ = s.recordTelemetry("service_start", map[string]interface{}{
		"service_id": svc.ID,
		"port":       port,
		"type":       "ui-bundle",
	})

	extraFields := map[string]interface{}{"port": port, "type": "ui-bundle"}
	s.monitorReadiness(serverCtx, svc.ID, extraFields)

	return nil
}

// resolveUIPort determines which port name to use for a UI bundle service.
func (s *Supervisor) resolveUIPort(svc manifest.Service) (int, error) {
	portName := "ui"
	switch {
	case svc.Health.PortName != "":
		portName = svc.Health.PortName
	case svc.Readiness.PortName != "":
		portName = svc.Readiness.PortName
	case svc.Ports != nil && len(svc.Ports.Requested) > 0:
		portName = svc.Ports.Requested[0].Name
	}
	port, err := s.portAllocator.Resolve(svc.ID, portName)
	if err != nil {
		return 0, fmt.Errorf("allocate port for %s: %w", svc.ID, err)
	}
	return port, nil
}

// buildUIHandler creates the HTTP handler for a UI bundle service with health, API proxy, and SPA fallback.
func (s *Supervisor) buildUIHandler(svc manifest.Service, serveRoot string) http.Handler {
	healthPath := svc.Health.Path
	if healthPath == "" {
		healthPath = "/health"
	}

	apiProxy := s.buildAPIProxy()
	fileServer := http.FileServer(http.Dir(serveRoot))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == healthPath {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"healthy"}`))
			return
		}

		if apiProxy != nil && (strings.HasPrefix(r.URL.Path, "/api") || strings.HasPrefix(strings.ToLower(r.URL.Path), "/ws")) {
			apiProxy.ServeHTTP(w, r)
			return
		}

		path := filepath.Join(serveRoot, filepath.Clean(r.URL.Path))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}
		indexPath := filepath.Join(serveRoot, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			http.ServeFile(w, r, indexPath)
			return
		}
		http.NotFound(w, r)
	})
}

// buildAPIProxy creates a reverse proxy to the API service, or nil if no API port is found.
func (s *Supervisor) buildAPIProxy() *httputil.ReverseProxy {
	apiPort := s.resolveAPIPort()
	if apiPort <= 0 {
		return nil
	}
	target, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", apiPort))
	return httputil.NewSingleHostReverseProxy(target)
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

// resolveUIDistRoot determines the directory to serve static files from for a ui-bundle service.
// It uses the following priority:
//  1. Explicit dist_root field in the service config
//  2. Automatic detection by finding index.html in the assets list
//  3. Error if neither works
//
// This approach ensures the serve root is always correct regardless of how assets are organized
// (e.g., assets in a subdirectory like "ui/dist/assets/").
func (s *Supervisor) resolveUIDistRoot(svc manifest.Service) (string, error) {
	// Priority 1: Explicit dist_root takes precedence
	if svc.DistRoot != "" {
		return manifest.ResolvePath(s.opts.BundlePath, svc.DistRoot), nil
	}

	// Priority 2: Find index.html in assets - its parent directory is the dist root
	for _, asset := range svc.Assets {
		if filepath.Base(asset.Path) == "index.html" {
			distRoot := filepath.Dir(asset.Path)
			return manifest.ResolvePath(s.opts.BundlePath, distRoot), nil
		}
	}

	// Priority 3: Error with clear guidance
	return "", fmt.Errorf(
		"ui-bundle %s: cannot determine dist root. "+
			"Either add 'dist_root' to the service config, or ensure index.html is in the assets list. "+
			"Assets found: %d",
		svc.ID, len(svc.Assets),
	)
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

// startBundledResources launches only the app-private server artifacts that
// were selected and verified by the immutable resource deployment plan. These
// processes are intentionally separate from manifest services: they have no
// user-facing launcher entry and cannot be replaced by host discovery.
func (s *Supervisor) startBundledResources(ctx context.Context) error {
	if s.resourcePlan == nil || s.opts.DryRun {
		return nil
	}
	supervisor := resourceplan.NewServiceSupervisor(s.opts.BundlePath, s.appData, s.opts.SharedResourceResolver)
	if err := supervisor.Start(ctx, s.resourcePlan); err != nil {
		return fmt.Errorf("start bundled managed resources: %w", err)
	}
	s.resourceServer = supervisor
	for resource, status := range supervisor.Statuses() {
		_ = s.recordTelemetry("managed_resource_start", map[string]interface{}{
			"resource": resource,
			"pid":      status.PID,
			"log_path": status.LogPath,
		})
	}
	return nil
}
