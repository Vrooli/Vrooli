package bundleruntime

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"scenario-to-desktop-runtime/manifest"
)

type Options struct {
	AppDataDir string
	Manifest   *manifest.Manifest
	BundlePath string
	DryRun     bool
}

type Supervisor struct {
	opts            Options
	authToken       string
	appData         string
	secretsPath     string
	telemetryPath   string
	migrationsPath  string
	portMap         map[string]map[string]int // service -> port name -> port
	serviceStatus   map[string]ServiceStatus
	server          *http.Server
	mu              sync.RWMutex
	started         bool
	procs           map[string]*serviceProcess
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	secrets         map[string]string
	servicesStarted bool
	runtimeCtx      context.Context
	migrations      migrationsState
}

type ServiceStatus struct {
	Ready    bool   `json:"ready"`
	Message  string `json:"message,omitempty"`
	ExitCode *int   `json:"exit_code,omitempty"`
}

type telemetryRecord struct {
	Timestamp time.Time              `json:"ts"`
	Event     string                 `json:"event"`
	ServiceID string                 `json:"service_id,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

type serviceProcess struct {
	cmd      *exec.Cmd
	logPath  string
	service  manifest.Service
	started  time.Time
	cancel   context.CancelFunc
	stopping bool
}

func NewSupervisor(opts Options) (*Supervisor, error) {
	if opts.Manifest == nil {
		return nil, errors.New("manifest is required")
	}

	appData := opts.AppDataDir
	if appData == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			return nil, fmt.Errorf("resolve app data dir: %w", err)
		}
		appData = filepath.Join(base, sanitizeAppName(opts.Manifest.App.Name))
	}

	s := &Supervisor{
		opts:          opts,
		appData:       appData,
		portMap:       make(map[string]map[string]int),
		serviceStatus: make(map[string]ServiceStatus),
		procs:         make(map[string]*serviceProcess),
	}

	return s, nil
}

func (s *Supervisor) Start(ctx context.Context) error {
	if err := os.MkdirAll(s.appData, 0o755); err != nil {
		return fmt.Errorf("create app data dir: %w", err)
	}

	// Prepare secrets + telemetry locations.
	s.secretsPath = filepath.Join(s.appData, "secrets.json")
	s.telemetryPath = manifest.ResolvePath(s.appData, s.opts.Manifest.Telemetry.File)
	s.migrationsPath = filepath.Join(s.appData, "migrations.json")
	if err := os.MkdirAll(filepath.Dir(s.telemetryPath), 0o755); err != nil {
		return fmt.Errorf("create telemetry dir: %w", err)
	}

	loadedSecrets, err := s.loadSecrets()
	if err != nil {
		return fmt.Errorf("load secrets: %w", err)
	}
	s.secrets = loadedSecrets
	migrations, err := s.loadMigrations()
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}
	s.migrations = migrations

	// Auth token.
	tokenPath := manifest.ResolvePath(s.appData, s.opts.Manifest.IPC.AuthTokenRel)
	token, err := loadOrCreateToken(tokenPath)
	if err != nil {
		return fmt.Errorf("load auth token: %w", err)
	}
	s.authToken = token

	// Allocate ports.
	if err := s.allocatePorts(); err != nil {
		return err
	}

	for _, svc := range s.opts.Manifest.Services {
		s.serviceStatus[svc.ID] = ServiceStatus{Ready: false, Message: "pending start"}
	}

	if err := s.recordTelemetry("runtime_start", nil); err != nil {
		return fmt.Errorf("write telemetry: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/readyz", s.handleReady)
	mux.HandleFunc("/ports", s.handlePorts)
	mux.HandleFunc("/logs/tail", s.handleLogs)
	mux.HandleFunc("/shutdown", s.handleShutdown)
	mux.HandleFunc("/secrets", s.handleSecrets)
	mux.HandleFunc("/telemetry", s.handleTelemetry)

	addr := fmt.Sprintf("%s:%d", s.opts.Manifest.IPC.Host, s.opts.Manifest.IPC.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: s.authMiddleware(mux),
	}
	s.server = server

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("start control API on %s: %w", addr, err)
	}

	s.started = true

	runtimeCtx, cancel := context.WithCancel(ctx)
	s.runtimeCtx = runtimeCtx
	s.cancel = cancel

	missing := s.missingRequiredSecrets()
	if len(missing) > 0 {
		msg := fmt.Sprintf("waiting for secrets: %s", strings.Join(missing, ", "))
		for _, svc := range s.opts.Manifest.Services {
			s.setStatus(svc.ID, ServiceStatus{Ready: false, Message: msg})
		}
		_ = s.recordTelemetry("secrets_missing", map[string]interface{}{"missing": missing})
	} else {
		s.startServicesAsync()
	}

	go func() {
		<-ctx.Done()
		_ = s.Shutdown(context.Background())
	}()

	go func() {
		if serveErr := server.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			_ = s.recordTelemetry("runtime_error", map[string]interface{}{"error": serveErr.Error()})
		}
	}()

	return nil
}

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

func (s *Supervisor) Shutdown(ctx context.Context) error {
	if !s.started {
		return nil
	}
	s.started = false
	if s.cancel != nil {
		s.cancel()
	}

	// Stop services in reverse dependency order.
	s.stopServices(ctx)
	s.wg.Wait()

	_ = s.recordTelemetry("runtime_shutdown", nil)
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// allocatePorts assigns port numbers per service and port name.
func (s *Supervisor) allocatePorts() error {
	defaultRange := PortRange{Min: 47000, Max: 48000}
	if s.opts.Manifest.Ports != nil && s.opts.Manifest.Ports.DefaultRange != nil {
		defaultRange = PortRange{
			Min: s.opts.Manifest.Ports.DefaultRange.Min,
			Max: s.opts.Manifest.Ports.DefaultRange.Max,
		}
	}

	reserved := map[int]bool{}
	if s.opts.Manifest.Ports != nil {
		for _, p := range s.opts.Manifest.Ports.Reserved {
			reserved[p] = true
		}
	}

	nextPort := defaultRange.Min

	for _, svc := range s.opts.Manifest.Services {
		if s.portMap[svc.ID] == nil {
			s.portMap[svc.ID] = make(map[string]int)
		}
		if svc.Ports == nil || len(svc.Ports.Requested) == 0 {
			continue
		}
		for _, req := range svc.Ports.Requested {
			rng := PortRange{
				Min: defaultRange.Min,
				Max: defaultRange.Max,
			}
			if req.Range.Min != 0 && req.Range.Max != 0 {
				rng = PortRange{Min: req.Range.Min, Max: req.Range.Max}
			}
			port, err := s.pickPort(rng, reserved, &nextPort)
			if err != nil {
				return fmt.Errorf("allocate port for %s:%s: %w", svc.ID, req.Name, err)
			}
			s.portMap[svc.ID][req.Name] = port
		}
	}

	return nil
}

func (s *Supervisor) pickPort(rng PortRange, reserved map[int]bool, next *int) (int, error) {
	min := rng.Min
	max := rng.Max
	if min == 0 || max == 0 || max < min {
		return 0, fmt.Errorf("invalid range %v", rng)
	}
	for p := min; p <= max; p++ {
		if reserved[p] {
			continue
		}
		if p < *next {
			continue
		}
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p))
		if err != nil {
			continue
		}
		ln.Close()
		*next = p + 1
		return p, nil
	}
	return 0, fmt.Errorf("no free port in %d-%d", min, max)
}

func (s *Supervisor) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "ok",
		"services": len(s.serviceStatus),
		"runtime":  runtime.Version(),
	})
}

func (s *Supervisor) handleReady(w http.ResponseWriter, r *http.Request) {
	allReady := true
	details := make(map[string]ServiceStatus)
	for id, st := range s.serviceStatus {
		details[id] = st
		if !st.Ready {
			allReady = false
		}
	}
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"ready":   allReady,
		"details": details,
	})
}

func (s *Supervisor) handlePorts(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"services": s.portMap,
	})
}

func (s *Supervisor) handleLogs(w http.ResponseWriter, r *http.Request) {
	serviceID := r.URL.Query().Get("serviceId")
	linesParam := r.URL.Query().Get("lines")
	lines := 200
	if linesParam != "" {
		if v, err := parsePositiveInt(linesParam); err == nil {
			lines = v
		}
	}

	var service manifest.Service
	found := false
	for _, svc := range s.opts.Manifest.Services {
		if svc.ID == serviceID {
			service = svc
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "unknown service", http.StatusBadRequest)
		return
	}

	if service.LogDir == "" {
		http.Error(w, "service has no log_dir", http.StatusBadRequest)
		return
	}

	logPath := manifest.ResolvePath(s.appData, service.LogDir)
	info, err := os.Stat(logPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("log path unavailable: %v", err), http.StatusBadRequest)
		return
	}
	if info.IsDir() {
		http.Error(w, "log_dir points to a directory; expected file path", http.StatusBadRequest)
		return
	}
	content, err := tailFile(logPath, lines)
	if err != nil {
		http.Error(w, fmt.Sprintf("tail logs: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write(content)
}

func (s *Supervisor) handleShutdown(w http.ResponseWriter, r *http.Request) {
	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = s.Shutdown(context.Background())
	}()
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "stopping"})
}

func (s *Supervisor) handleSecrets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var payload struct {
		Secrets map[string]string `json:"secrets"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if payload.Secrets == nil {
		payload.Secrets = map[string]string{}
	}

	merged := s.secretsCopy()
	for k, v := range payload.Secrets {
		merged[k] = v
	}
	missing := s.missingRequiredSecretsFrom(merged)
	if len(missing) > 0 {
		msg := fmt.Sprintf("missing required secrets: %s", strings.Join(missing, ", "))
		_ = s.recordTelemetry("secrets_missing", map[string]interface{}{"missing": missing})
		s.writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":   msg,
			"missing": missing,
		})
		for _, svc := range s.opts.Manifest.Services {
			needs := intersection(missing, svc.Secrets)
			if len(needs) == 0 {
				continue
			}
			s.setStatus(svc.ID, ServiceStatus{Ready: false, Message: msg})
		}
		return
	}

	if err := s.persistSecrets(merged); err != nil {
		http.Error(w, "persist secrets", http.StatusInternalServerError)
		return
	}

	s.mu.Lock()
	s.secrets = merged
	s.mu.Unlock()

	_ = s.recordTelemetry("secrets_updated", map[string]interface{}{"count": len(merged)})
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	if !s.servicesStarted {
		s.startServicesAsync()
	}
}

func (s *Supervisor) handleTelemetry(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"path":       s.telemetryPath,
		"upload_url": s.opts.Manifest.Telemetry.UploadTo,
	})
}

func (s *Supervisor) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Health can stay unauthenticated so Electron can gate on it.
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "missing auth", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token != s.authToken {
			http.Error(w, "invalid auth", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Supervisor) writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Supervisor) recordTelemetry(event string, details map[string]interface{}) error {
	rec := telemetryRecord{
		Timestamp: time.Now().UTC(),
		Event:     event,
		Details:   details,
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(s.telemetryPath), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(s.telemetryPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

// Utilities.

func (s *Supervisor) loadSecrets() (map[string]string, error) {
	out := map[string]string{}
	data, err := os.ReadFile(s.secretsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return out, nil
		}
		return nil, err
	}

	var wrapper struct {
		Secrets map[string]string `json:"secrets"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil || wrapper.Secrets == nil {
		var legacy map[string]string
		if err2 := json.Unmarshal(data, &legacy); err2 == nil {
			return legacy, nil
		}
		return nil, err
	}
	return wrapper.Secrets, nil
}

func (s *Supervisor) persistSecrets(secrets map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(s.secretsPath), 0o700); err != nil {
		return err
	}
	payload := struct {
		Secrets map[string]string `json:"secrets"`
	}{
		Secrets: secrets,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(s.secretsPath, data, 0o600)
}

func (s *Supervisor) secretsCopy() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.secrets))
	for k, v := range s.secrets {
		out[k] = v
	}
	return out
}

func (s *Supervisor) missingRequiredSecrets() []string {
	return s.missingRequiredSecretsFrom(s.secretsCopy())
}

func (s *Supervisor) missingRequiredSecretsFrom(secrets map[string]string) []string {
	var missing []string
	for _, sec := range s.opts.Manifest.Secrets {
		required := true
		if sec.Required != nil {
			required = *sec.Required
		}
		if !required {
			continue
		}
		val := strings.TrimSpace(secrets[sec.ID])
		if val == "" {
			missing = append(missing, sec.ID)
		}
	}
	return missing
}

func (s *Supervisor) findSecret(id string) *manifest.Secret {
	for i := range s.opts.Manifest.Secrets {
		if s.opts.Manifest.Secrets[i].ID == id {
			return &s.opts.Manifest.Secrets[i]
		}
	}
	return nil
}

// Migration handling.

type migrationsState struct {
	AppVersion string              `json:"app_version"`
	Applied    map[string][]string `json:"applied"`
}

func (s *Supervisor) loadMigrations() (migrationsState, error) {
	state := migrationsState{
		Applied: map[string][]string{},
	}
	data, err := os.ReadFile(s.migrationsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return state, nil
		}
		return state, err
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return state, err
	}
	if state.Applied == nil {
		state.Applied = map[string][]string{}
	}
	return state, nil
}

func (s *Supervisor) persistMigrations(state migrationsState) error {
	if err := os.MkdirAll(filepath.Dir(s.migrationsPath), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(s.migrationsPath, data, 0o600)
}

func loadOrCreateToken(path string) (string, error) {
	if data, err := os.ReadFile(path); err == nil && len(strings.TrimSpace(string(data))) > 0 {
		return strings.TrimSpace(string(data)), nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}

	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf)
	if err := os.WriteFile(path, []byte(token), 0o600); err != nil {
		return "", err
	}
	return token, nil
}

func sanitizeAppName(name string) string {
	out := strings.TrimSpace(name)
	if out == "" {
		return "desktop-app"
	}
	out = strings.ReplaceAll(out, " ", "-")
	out = strings.ToLower(out)
	return out
}

func parsePositiveInt(s string) (int, error) {
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, err
	}
	if v <= 0 {
		return 0, errors.New("must be positive")
	}
	return v, nil
}

func tailFile(path string, lines int) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(string(data), "\n")
	if lines >= len(parts) {
		return []byte(strings.Join(parts, "\n")), nil
	}
	return []byte(strings.Join(parts[len(parts)-lines:], "\n")), nil
}

// PortRange mirrors manifest.PortRange but keeps this package independent.
type PortRange struct {
	Min int
	Max int
}

// Service orchestration.

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
		// Ensure dependencies are ready before attempting start.
		if err := s.waitForDependencies(ctx, svc); err != nil {
			s.setStatus(svc.ID, ServiceStatus{Ready: false, Message: err.Error()})
			_ = s.recordTelemetry("service_blocked", map[string]interface{}{"service_id": svc.ID, "reason": err.Error()})
			continue
		}
		if err := s.startService(ctx, *svc); err != nil {
			s.setStatus(svc.ID, ServiceStatus{Ready: false, Message: err.Error()})
			_ = s.recordTelemetry("service_start_failed", map[string]interface{}{"service_id": svc.ID, "error": err.Error()})
			continue
		}
	}
	return nil
}

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

	if err := s.prepareServiceDirs(svc); err != nil {
		return err
	}

	envMap, err := s.renderEnvMap(svc, bin)
	if err != nil {
		return err
	}
	if err := s.applySecrets(envMap, svc); err != nil {
		return err
	}
	if err := s.applyPlaywrightConventions(svc, envMap); err != nil {
		return err
	}
	if err := s.ensureAssets(svc); err != nil {
		return err
	}
	if err := s.runMigrations(ctx, svc, bin, envMap); err != nil {
		return err
	}

	cmdCtx, cancel := context.WithCancel(ctx)
	args := s.renderArgs(bin.Args)
	cmd := exec.CommandContext(cmdCtx, cmdPath, args...)
	cmd.Env = envMapToList(envMap)
	if bin.CWD != "" {
		cmd.Dir = manifest.ResolvePath(s.opts.BundlePath, bin.CWD)
	} else {
		cmd.Dir = s.opts.BundlePath
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
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("start %s: %w", svc.ID, err)
	}

	proc := &serviceProcess{
		cmd:     cmd,
		logPath: logPath,
		service: svc,
		started: time.Now(),
		cancel:  cancel,
	}
	s.setProc(svc.ID, proc)
	s.setStatus(svc.ID, ServiceStatus{Ready: false, Message: "starting"})
	_ = s.recordTelemetry("service_start", map[string]interface{}{"service_id": svc.ID})

	// Wait for readiness in background.
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.waitForReadiness(cmdCtx, svc); err != nil {
			s.setStatus(svc.ID, ServiceStatus{Ready: false, Message: err.Error()})
			_ = s.recordTelemetry("service_not_ready", map[string]interface{}{"service_id": svc.ID, "error": err.Error()})
		} else {
			s.setStatus(svc.ID, ServiceStatus{Ready: true, Message: "ready"})
			_ = s.recordTelemetry("service_ready", map[string]interface{}{"service_id": svc.ID})
		}
	}()

	// Watch for unexpected exits.
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		err := cmd.Wait()
		code := exitCode(err)
		msg := "stopped"
		if err != nil {
			msg = err.Error()
		}
		s.setStatus(svc.ID, ServiceStatus{Ready: false, Message: msg, ExitCode: code})
		_ = s.recordTelemetry("service_exit", map[string]interface{}{"service_id": svc.ID, "exit_code": code, "error": msg})
	}()

	return nil
}

func (s *Supervisor) waitForDependencies(ctx context.Context, svc *manifest.Service) error {
	if len(svc.Dependencies) == 0 {
		return nil
	}
	deadline := time.Now().Add(2 * time.Minute)
	for {
		allReady := true
		for _, dep := range svc.Dependencies {
			status, ok := s.getStatus(dep)
			if !ok {
				return fmt.Errorf("dependency %s missing", dep)
			}
			if !status.Ready {
				allReady = false
				break
			}
		}
		if allReady {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("dependencies not ready: %v", svc.Dependencies)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
}

func (s *Supervisor) waitForReadiness(ctx context.Context, svc manifest.Service) error {
	timeout := time.Duration(svc.Readiness.TimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	switch svc.Readiness.Type {
	case "health_success":
		return s.pollHealth(ctx, svc)
	case "port_open":
		port, err := s.resolvePort(svc.ID, svc.Readiness.PortName)
		if err != nil {
			return err
		}
		return waitForPort(ctx, port)
	case "log_match":
		return s.waitForLogPattern(ctx, svc)
	case "dependencies_ready":
		return s.waitForDependencies(ctx, &svc)
	default:
		return fmt.Errorf("unknown readiness type %q", svc.Readiness.Type)
	}
}

func (s *Supervisor) pollHealth(ctx context.Context, svc manifest.Service) error {
	interval := time.Duration(svc.Health.IntervalMs) * time.Millisecond
	if interval == 0 {
		interval = 500 * time.Millisecond
	}
	timeout := time.Duration(svc.Health.TimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 2 * time.Second
	}
	retries := svc.Health.Retries
	if retries == 0 {
		retries = 3
	}

	for attempt := 0; attempt <= retries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if s.checkHealthOnce(ctx, svc, timeout) {
			return nil
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("health check failed after %d attempts", retries+1)
}

func (s *Supervisor) checkHealthOnce(ctx context.Context, svc manifest.Service, timeout time.Duration) bool {
	switch svc.Health.Type {
	case "http":
		port, err := s.resolvePort(svc.ID, svc.Health.PortName)
		if err != nil {
			return false
		}
		url := fmt.Sprintf("http://127.0.0.1:%d%s", port, svc.Health.Path)
		client := &http.Client{Timeout: timeout}
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		resp, err := client.Do(req)
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode >= 200 && resp.StatusCode < 300
	case "tcp":
		port, err := s.resolvePort(svc.ID, svc.Health.PortName)
		if err != nil {
			return false
		}
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), timeout)
		if err != nil {
			return false
		}
		conn.Close()
		return true
	case "command":
		if len(svc.Health.Command) == 0 {
			return false
		}
		cmd := exec.CommandContext(ctx, svc.Health.Command[0], svc.Health.Command[1:]...)
		if err := cmd.Start(); err != nil {
			return false
		}
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case err := <-done:
			return err == nil
		case <-time.After(timeout):
			_ = cmd.Process.Kill()
			return false
		}
	case "log_match":
		return s.logMatches(svc, svc.Health.Path)
	default:
		return false
	}
}

func (s *Supervisor) waitForLogPattern(ctx context.Context, svc manifest.Service) error {
	if svc.Readiness.Pattern == "" {
		return errors.New("log_match readiness missing pattern")
	}
	re, err := regexp.Compile(svc.Readiness.Pattern)
	if err != nil {
		return fmt.Errorf("invalid readiness pattern: %w", err)
	}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		if s.logMatches(svc, re.String()) {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Supervisor) logMatches(svc manifest.Service, pattern string) bool {
	if svc.LogDir == "" {
		return false
	}
	logPath := manifest.ResolvePath(s.appData, svc.LogDir)
	data, err := os.ReadFile(logPath)
	if err != nil {
		return false
	}
	ok, _ := regexp.Match(pattern, data)
	return ok
}

func (s *Supervisor) resolvePort(serviceID, portName string) (int, error) {
	portGroup, ok := s.portMap[serviceID]
	if !ok {
		return 0, fmt.Errorf("no ports allocated for %s", serviceID)
	}
	port, ok := portGroup[portName]
	if !ok {
		return 0, fmt.Errorf("port %s missing for %s", portName, serviceID)
	}
	return port, nil
}

func waitForPort(ctx context.Context, port int) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func (s *Supervisor) prepareServiceDirs(svc manifest.Service) error {
	for _, dir := range svc.DataDirs {
		path := manifest.ResolvePath(s.appData, dir)
		if err := os.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("create data dir for %s: %w", svc.ID, err)
		}
	}
	if svc.LogDir != "" {
		logPath := manifest.ResolvePath(s.appData, svc.LogDir)
		if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
			return fmt.Errorf("prepare log dir: %w", err)
		}
		if _, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err != nil {
			return fmt.Errorf("prepare log file: %w", err)
		}
	}
	return nil
}

func (s *Supervisor) renderEnvMap(svc manifest.Service, bin manifest.Binary) (map[string]string, error) {
	env := map[string]string{}
	for _, kv := range os.Environ() {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}
	// Standard bundle hints.
	env["APP_DATA_DIR"] = s.appData
	env["BUNDLE_ROOT"] = s.opts.BundlePath

	apply := func(src map[string]string) {
		for k, v := range src {
			env[k] = s.renderValue(v)
		}
	}
	apply(svc.Env)
	apply(bin.Env)

	return env, nil
}

func envMapToList(env map[string]string) []string {
	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, fmt.Sprintf("%s=%s", k, v))
	}
	return out
}

func (s *Supervisor) renderArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		out = append(out, s.renderValue(a))
	}
	return out
}

func (s *Supervisor) renderValue(input string) string {
	replacements := map[string]string{
		"data":   s.appData,
		"bundle": s.opts.BundlePath,
	}
	portLookup := func(token string) (string, bool) {
		parts := strings.Split(token, ".")
		if len(parts) != 2 {
			return "", false
		}
		if svcPorts, ok := s.portMap[parts[0]]; ok {
			if port, ok := svcPorts[parts[1]]; ok {
				return strconv.Itoa(port), true
			}
		}
		return "", false
	}

	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	return re.ReplaceAllStringFunc(input, func(match string) string {
		key := strings.TrimSuffix(strings.TrimPrefix(match, "${"), "}")
		if v, ok := replacements[key]; ok {
			return v
		}
		if port, ok := portLookup(key); ok {
			return port
		}
		return match
	})
}

func (s *Supervisor) applySecrets(env map[string]string, svc manifest.Service) error {
	secrets := s.secretsCopy()
	for _, secretID := range svc.Secrets {
		secret := s.findSecret(secretID)
		if secret == nil {
			return fmt.Errorf("service %s references unknown secret %s", svc.ID, secretID)
		}

		value := strings.TrimSpace(secrets[secretID])
		required := true
		if secret.Required != nil {
			required = *secret.Required
		}

		if value == "" {
			if required {
				return fmt.Errorf("secret %s missing for service %s", secretID, svc.ID)
			}
			continue
		}

		switch secret.Target.Type {
		case "env":
			name := secret.Target.Name
			if name == "" {
				name = strings.ToUpper(secret.ID)
			}
			env[name] = value
		case "file":
			if secret.Target.Name == "" {
				return fmt.Errorf("secret %s missing file path target", secretID)
			}
			path := manifest.ResolvePath(s.appData, secret.Target.Name)
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				return fmt.Errorf("secret %s path setup: %w", secretID, err)
			}
			if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
				return fmt.Errorf("secret %s write: %w", secretID, err)
			}
			envName := fmt.Sprintf("SECRET_FILE_%s", strings.ToUpper(secret.ID))
			env[envName] = path
		default:
			return fmt.Errorf("secret %s has unsupported target type %s", secretID, secret.Target.Type)
		}
	}
	return nil
}

func (s *Supervisor) applyPlaywrightConventions(svc manifest.Service, env map[string]string) error {
	if !serviceUsesPlaywright(svc) {
		return nil
	}

	if _, ok := env["PLAYWRIGHT_DRIVER_PORT"]; !ok {
		if port, err := s.resolvePort(svc.ID, "http"); err == nil {
			env["PLAYWRIGHT_DRIVER_PORT"] = strconv.Itoa(port)
		}
	}
	if _, ok := env["PLAYWRIGHT_DRIVER_URL"]; !ok {
		if port, err := s.resolvePort(svc.ID, "http"); err == nil {
			env["PLAYWRIGHT_DRIVER_URL"] = fmt.Sprintf("http://127.0.0.1:%d", port)
		}
	}
	if _, ok := env["ENGINE"]; !ok {
		env["ENGINE"] = "playwright"
	}

	chromePath := strings.TrimSpace(env["PLAYWRIGHT_CHROMIUM_PATH"])
	if chromePath == "" {
		if fallback := strings.TrimSpace(os.Getenv("ELECTRON_CHROMIUM_PATH")); fallback != "" {
			env["PLAYWRIGHT_CHROMIUM_PATH"] = fallback
			_ = s.recordTelemetry("playwright_chromium_fallback", map[string]interface{}{
				"service_id": svc.ID,
				"fallback":   fallback,
			})
		}
		return nil
	}

	resolved := manifest.ResolvePath(s.opts.BundlePath, chromePath)
	env["PLAYWRIGHT_CHROMIUM_PATH"] = resolved
	if _, err := os.Stat(resolved); err == nil {
		return nil
	}

	if fallback := strings.TrimSpace(os.Getenv("ELECTRON_CHROMIUM_PATH")); fallback != "" {
		env["PLAYWRIGHT_CHROMIUM_PATH"] = fallback
		_ = s.recordTelemetry("playwright_chromium_fallback", map[string]interface{}{
			"service_id": svc.ID,
			"missing":    resolved,
			"fallback":   fallback,
		})
		return nil
	}

	_ = s.recordTelemetry("asset_missing", map[string]interface{}{
		"service_id": svc.ID,
		"path":       resolved,
		"reason":     "playwright_chromium",
	})
	return fmt.Errorf("playwright chromium missing for %s: %s", svc.ID, resolved)
}

func (s *Supervisor) runMigrations(ctx context.Context, svc manifest.Service, bin manifest.Binary, baseEnv map[string]string) error {
	if len(svc.Migrations) == 0 {
		return s.ensureAppVersionRecorded()
	}

	phase := s.installPhase()
	state := s.migrations
	appliedSet := make(map[string]bool)
	for _, v := range state.Applied[svc.ID] {
		appliedSet[v] = true
	}
	envBase := copyStringMap(baseEnv)

	for _, m := range svc.Migrations {
		if m.Version == "" {
			return fmt.Errorf("migration for service %s missing version", svc.ID)
		}
		if appliedSet[m.Version] {
			continue
		}
		runOn := strings.TrimSpace(m.RunOn)
		if runOn == "" {
			runOn = "always"
		}
		switch runOn {
		case "always":
		case "first_install":
			if phase != "first_install" {
				continue
			}
		case "upgrade":
			if phase != "upgrade" {
				continue
			}
		default:
			return fmt.Errorf("migration %s has unsupported run_on=%s", m.Version, runOn)
		}

		if len(m.Command) == 0 {
			return fmt.Errorf("migration %s has no command", m.Version)
		}

		env := copyStringMap(envBase)
		for k, v := range m.Env {
			env[k] = s.renderValue(v)
		}

		if err := s.executeMigration(ctx, svc, m, bin, env); err != nil {
			_ = s.recordTelemetry("migration_failed", map[string]interface{}{"service_id": svc.ID, "version": m.Version, "error": err.Error()})
			return err
		}

		appliedSet[m.Version] = true
		if state.Applied[svc.ID] == nil {
			state.Applied[svc.ID] = []string{}
		}
		state.Applied[svc.ID] = append(state.Applied[svc.ID], m.Version)
		_ = s.recordTelemetry("migration_applied", map[string]interface{}{"service_id": svc.ID, "version": m.Version})
		if err := s.persistMigrations(state); err != nil {
			return err
		}
	}

	state.AppVersion = s.opts.Manifest.App.Version
	s.migrations = state
	return s.persistMigrations(state)
}

func (s *Supervisor) executeMigration(ctx context.Context, svc manifest.Service, m manifest.Migration, bin manifest.Binary, env map[string]string) error {
	cmdArgs := s.renderArgs(m.Command)
	cmdPath := manifest.ResolvePath(s.opts.BundlePath, cmdArgs[0])
	args := cmdArgs[1:]

	_ = s.recordTelemetry("migration_start", map[string]interface{}{"service_id": svc.ID, "version": m.Version})

	cmd := exec.CommandContext(ctx, cmdPath, args...)
	cmd.Env = envMapToList(env)
	if bin.CWD != "" {
		cmd.Dir = manifest.ResolvePath(s.opts.BundlePath, bin.CWD)
	} else {
		cmd.Dir = s.opts.BundlePath
	}

	logWriter, _, err := s.logWriter(svc)
	if err != nil {
		return err
	}
	if logWriter != nil {
		defer logWriter.Close()
		cmd.Stdout = logWriter
		cmd.Stderr = logWriter
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start migration %s: %w", m.Version, err)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("migration %s failed: %w", m.Version, err)
	}
	return nil
}

func (s *Supervisor) ensureAssets(svc manifest.Service) error {
	for _, asset := range svc.Assets {
		path := manifest.ResolvePath(s.opts.BundlePath, asset.Path)
		info, err := os.Stat(path)
		if err != nil {
			_ = s.recordTelemetry("asset_missing", map[string]interface{}{"service_id": svc.ID, "path": asset.Path})
			return fmt.Errorf("asset missing for service %s: %s", svc.ID, asset.Path)
		}
		if info.IsDir() {
			_ = s.recordTelemetry("asset_missing", map[string]interface{}{"service_id": svc.ID, "path": asset.Path, "reason": "expected file"})
			return fmt.Errorf("asset path is a directory: %s", asset.Path)
		}
		if asset.SizeBytes > 0 {
			if err := s.checkAssetSizeBudget(svc, asset, info.Size()); err != nil {
				return err
			}
		}
		if asset.SHA256 != "" {
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read asset %s: %w", asset.Path, err)
			}
			sum := fmt.Sprintf("%x", sha256.Sum256(data))
			if !strings.EqualFold(sum, asset.SHA256) {
				return fmt.Errorf("asset %s checksum mismatch", asset.Path)
			}
		}
	}
	return nil
}

func (s *Supervisor) ensureAppVersionRecorded() error {
	state := s.migrations
	if state.AppVersion == s.opts.Manifest.App.Version {
		return nil
	}
	state.AppVersion = s.opts.Manifest.App.Version
	s.migrations = state
	return s.persistMigrations(state)
}

func (s *Supervisor) installPhase() string {
	if s.migrations.AppVersion == "" {
		return "first_install"
	}
	if s.migrations.AppVersion != s.opts.Manifest.App.Version {
		return "upgrade"
	}
	return "current"
}

func copyStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func serviceUsesPlaywright(svc manifest.Service) bool {
	if strings.Contains(strings.ToLower(svc.ID), "playwright") {
		return true
	}
	for k := range svc.Env {
		if strings.HasPrefix(k, "PLAYWRIGHT_") {
			return true
		}
	}
	return false
}

func (s *Supervisor) checkAssetSizeBudget(svc manifest.Service, asset manifest.Asset, actual int64) error {
	expected := asset.SizeBytes
	slack := sizeBudgetSlack(expected)
	if actual > expected+slack {
		_ = s.recordTelemetry("asset_size_exceeded", map[string]interface{}{
			"service_id":     svc.ID,
			"path":           asset.Path,
			"expected_bytes": expected,
			"actual_bytes":   actual,
			"slack_bytes":    slack,
		})
		return fmt.Errorf("asset %s exceeds size budget: got %d bytes, budget %d (+%d slack)", asset.Path, actual, expected, slack)
	}
	if actual < expected/2 {
		_ = s.recordTelemetry("asset_size_suspicious", map[string]interface{}{
			"service_id":     svc.ID,
			"path":           asset.Path,
			"expected_bytes": expected,
			"actual_bytes":   actual,
		})
		return fmt.Errorf("asset %s is smaller than expected (%d bytes vs %d)", asset.Path, actual, expected)
	}
	if actual > expected {
		_ = s.recordTelemetry("asset_size_warning", map[string]interface{}{
			"service_id":     svc.ID,
			"path":           asset.Path,
			"expected_bytes": expected,
			"actual_bytes":   actual,
		})
	}
	return nil
}

func sizeBudgetSlack(expected int64) int64 {
	if expected <= 0 {
		return 0
	}
	percent := expected / 20 // 5%
	const minSlack = int64(1 * 1024 * 1024)
	if percent < minSlack {
		return minSlack
	}
	return percent
}

func (s *Supervisor) logWriter(svc manifest.Service) (*os.File, string, error) {
	if svc.LogDir == "" {
		return nil, "", nil
	}
	logPath := manifest.ResolvePath(s.appData, svc.LogDir)
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, "", fmt.Errorf("open log file: %w", err)
	}
	return f, logPath, nil
}

func (s *Supervisor) stopServices(ctx context.Context) {
	order, err := topoSort(s.opts.Manifest.Services)
	if err != nil {
		return
	}
	// stop in reverse
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
		if proc.cmd.Process != nil {
			_ = proc.cmd.Process.Signal(os.Interrupt)
			waitCh := make(chan error, 1)
			go func() { waitCh <- proc.cmd.Wait() }()
			select {
			case <-ctx.Done():
				_ = proc.cmd.Process.Kill()
			case <-waitCh:
			case <-time.After(3 * time.Second):
				_ = proc.cmd.Process.Kill()
			}
		}
	}
}

func (s *Supervisor) setStatus(id string, status ServiceStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.serviceStatus[id] = status
}

func (s *Supervisor) getStatus(id string) (ServiceStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st, ok := s.serviceStatus[id]
	return st, ok
}

func (s *Supervisor) setProc(id string, proc *serviceProcess) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.procs[id] = proc
}

func (s *Supervisor) getProc(id string) *serviceProcess {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.procs[id]
}

func topoSort(services []manifest.Service) ([]string, error) {
	graph := make(map[string][]string)
	inDegree := make(map[string]int)
	for _, svc := range services {
		graph[svc.ID] = append(graph[svc.ID], svc.Dependencies...)
		if _, ok := inDegree[svc.ID]; !ok {
			inDegree[svc.ID] = 0
		}
		for _, dep := range svc.Dependencies {
			inDegree[svc.ID]++
			if _, ok := inDegree[dep]; !ok {
				inDegree[dep] = 0
			}
		}
	}

	queue := make([]string, 0)
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	order := make([]string, 0, len(inDegree))
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		order = append(order, id)
		for _, svc := range services {
			for _, dep := range svc.Dependencies {
				if dep == id {
					inDegree[svc.ID]--
					if inDegree[svc.ID] == 0 {
						queue = append(queue, svc.ID)
					}
				}
			}
		}
	}

	if len(order) != len(inDegree) {
		return nil, errors.New("cycle detected in dependencies")
	}
	return order, nil
}

func findService(services []manifest.Service, id string) *manifest.Service {
	for i := range services {
		if services[i].ID == id {
			return &services[i]
		}
	}
	return nil
}

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

func intersection(a []string, b []string) []string {
	set := map[string]bool{}
	for _, v := range b {
		set[v] = true
	}
	var out []string
	for _, v := range a {
		if set[v] {
			out = append(out, v)
		}
	}
	return out
}
