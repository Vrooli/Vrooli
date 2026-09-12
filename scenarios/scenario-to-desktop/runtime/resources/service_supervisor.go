package resources

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/envkit-go"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/infra"
)

// ServiceStatus is the credential-free runtime state shown by the desktop
// supervisor. The server is an internal component, never a launch-surface app.
type ServiceStatus struct {
	Resource    string              `json:"resource"`
	PID         int                 `json:"pid,omitempty"`
	Running     bool                `json:"running"`
	LogPath     string              `json:"log_path,omitempty"`
	Ports       map[string]int      `json:"ports,omitempty"`
	Message     string              `json:"message,omitempty"`
	Provider    string              `json:"provider,omitempty"`
	Observation ProviderObservation `json:"provider_observation"`
}

// ProviderObservation is the credential-free runtime evidence contract. It
// describes selection and readiness without exposing endpoints, tokens, or
// operator configuration. The broker remains the authority for shared leases;
// the supervisor remains the authority for private lifecycle.
type ProviderObservation struct {
	DeploymentMode   string     `json:"deployment_mode"`
	ProviderTier     string     `json:"provider_tier"`
	ServiceIdentity  string     `json:"service_identity"`
	ArtifactDigest   string     `json:"artifact_digest,omitempty"`
	Readiness        string     `json:"readiness"`
	FallbackDecision string     `json:"fallback_decision,omitempty"`
	SafeRouteClass   string     `json:"safe_route_class"`
	LeaseExpiresAt   *time.Time `json:"lease_expires_at,omitempty"`
}

type runningService struct {
	cmd     *exec.Cmd
	logFile *os.File
	status  ServiceStatus
}

// ServiceSupervisor starts only service artifacts explicitly selected in a
// verified bundle plan. It does not discover host binaries or inspect ports.
// A caller may provide a consented broker resolver for scoped shared reuse;
// without one, standalone bundles are always app-private.
type ServiceSupervisor struct {
	bundleRoot     string
	appDataDir     string
	sharedResolver SharedServiceResolver

	mu                 sync.RWMutex
	running            map[string]*runningService
	statuses           map[string]ServiceStatus
	bindings           map[string]SharedServiceBinding
	privateEnvironment map[string]map[string]string
}

func NewServiceSupervisor(bundleRoot, appDataDir string, resolver ...SharedServiceResolver) *ServiceSupervisor {
	var sharedResolver SharedServiceResolver
	if len(resolver) > 0 {
		sharedResolver = resolver[0]
	}
	return &ServiceSupervisor{
		bundleRoot:         bundleRoot,
		appDataDir:         appDataDir,
		sharedResolver:     sharedResolver,
		running:            make(map[string]*runningService),
		statuses:           make(map[string]ServiceStatus),
		bindings:           make(map[string]SharedServiceBinding),
		privateEnvironment: make(map[string]map[string]string),
	}
}

func (s *ServiceSupervisor) Start(ctx context.Context, plan *Plan) error {
	if plan == nil {
		return nil
	}
	for _, item := range plan.Resources {
		if item.OS != runtimeOS() || item.Architecture != runtime.GOARCH || item.Mode != "bundled-service" {
			continue
		}
		if s.tryShared(ctx, item) {
			continue
		}
		if err := s.startOne(ctx, item); err != nil {
			return err
		}
	}
	return nil
}

func (s *ServiceSupervisor) tryShared(ctx context.Context, item Item) bool {
	if s.sharedResolver == nil || item.Service == nil {
		return false
	}
	binding, err := s.sharedResolver.ResolveSharedService(ctx, item)
	if err != nil {
		s.mu.Lock()
		s.statuses[item.Resource] = ServiceStatus{Resource: item.Resource, Message: "shared unavailable; using private bundled service", Observation: ProviderObservation{DeploymentMode: "bundled", ProviderTier: "none", ServiceIdentity: item.Resource, Readiness: "fallback", FallbackDecision: "private-bundle", SafeRouteClass: "private-bundle"}}
		s.mu.Unlock()
		return false
	}
	if strings.TrimSpace(binding.Endpoint) == "" {
		s.mu.Lock()
		s.statuses[item.Resource] = ServiceStatus{Resource: item.Resource, Message: "shared binding was empty; using private bundled service", Observation: ProviderObservation{DeploymentMode: "bundled", ProviderTier: "none", ServiceIdentity: item.Resource, Readiness: "fallback", FallbackDecision: "private-bundle", SafeRouteClass: "private-bundle"}}
		s.mu.Unlock()
		return false
	}
	if binding.ExpiresAt.IsZero() || !binding.ExpiresAt.After(time.Now()) {
		s.mu.Lock()
		s.statuses[item.Resource] = ServiceStatus{Resource: item.Resource, Message: "shared binding was expired; using private bundled service", Observation: ProviderObservation{DeploymentMode: "bundled", ProviderTier: "none", ServiceIdentity: item.Resource, Readiness: "fallback", FallbackDecision: "private-bundle", SafeRouteClass: "private-bundle"}}
		s.mu.Unlock()
		return false
	}
	provider := strings.TrimSpace(binding.Provider)
	if provider == "" {
		provider = "managed-shared"
	}
	s.mu.Lock()
	s.bindings[item.Resource] = SharedServiceBinding{Endpoint: binding.Endpoint, Environment: cloneEnvironment(binding.Environment), ExpiresAt: binding.ExpiresAt, Provider: provider}
	expiresAt := binding.ExpiresAt
	s.statuses[item.Resource] = ServiceStatus{Resource: item.Resource, Running: true, Message: "using consented shared service", Provider: provider, Observation: ProviderObservation{DeploymentMode: "bundled", ProviderTier: provider, ServiceIdentity: item.Resource, Readiness: "ready", SafeRouteClass: "shared-resource", LeaseExpiresAt: &expiresAt}}
	s.mu.Unlock()
	return true
}

func (s *ServiceSupervisor) startOne(ctx context.Context, item Item) error {
	if item.Service == nil {
		return fmt.Errorf("bundled service %s has no server declaration", item.Resource)
	}
	if err := verifyService(s.bundleRoot, item); err != nil {
		return err
	}
	s.mu.RLock()
	_, exists := s.running[item.Resource]
	s.mu.RUnlock()
	if exists {
		return nil
	}
	launch, err := s.prepareLaunch(item)
	if err != nil {
		return err
	}
	logPath := filepath.Join(launch.logDir, "service.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open bundled service log: %w", err)
	}
	service := item.Service
	servicePorts, err := allocateServicePorts(service.Ports)
	if err != nil {
		logFile.Close()
		return fmt.Errorf("allocate bundled service ports: %w", err)
	}
	artifactPath := filepath.Join(s.bundleRoot, "resources", item.Resource, service.Artifact)
	launchPath := artifactPath
	workingDir := filepath.Dir(artifactPath)
	if service.Layout == "dir" {
		launchPath = filepath.Join(artifactPath, filepath.FromSlash(service.EntryPath))
		workingDir = artifactPath
	}
	environmentMap, arguments := launch.environment(service, servicePorts)
	if err := writeServiceConfig(service.Config, launch.configDir, environmentMap); err != nil {
		logFile.Close()
		return fmt.Errorf("write bundled service config: %w", err)
	}
	if err := runServiceBootstrap(ctx, service, artifactPath, launch, environmentMap); err != nil {
		logFile.Close()
		return err
	}
	cmd := exec.CommandContext(ctx, launchPath, arguments...)
	infra.ConfigureProcessCommand(cmd)
	cmd.Env = environmentList(environmentMap)
	cmd.Dir = workingDir
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("start bundled service %s: %w", item.Resource, err)
	}
	s.mu.RLock()
	fallbackDecision := s.statuses[item.Resource].Observation.FallbackDecision
	s.mu.RUnlock()
	running := &runningService{cmd: cmd, logFile: logFile, status: ServiceStatus{Resource: item.Resource, PID: cmd.Process.Pid, Running: true, LogPath: logPath, Ports: servicePorts, Message: "running", Provider: "managed-private", Observation: ProviderObservation{DeploymentMode: "bundled", ProviderTier: "managed-private", ServiceIdentity: item.Resource, ArtifactDigest: "sha256:" + service.SHA256, Readiness: "starting", FallbackDecision: fallbackDecision, SafeRouteClass: "private-bundle"}}}
	s.mu.Lock()
	s.running[item.Resource] = running
	s.statuses[item.Resource] = running.status
	s.mu.Unlock()
	go s.wait(item.Resource, running)
	privateEnvironment, err := bootstrapPrivateService(ctx, item, servicePorts, s.appDataDir)
	if err != nil {
		_ = cmd.Process.Kill()
		return fmt.Errorf("bootstrap bundled service %s: %w", item.Resource, err)
	}
	if err := waitForServiceHealth(ctx, service.HealthChecks, environmentMap, service.Ports); err != nil {
		_ = cmd.Process.Kill()
		return fmt.Errorf("bundled service %s did not become healthy: %w", item.Resource, err)
	}
	s.mu.Lock()
	if len(privateEnvironment) > 0 {
		s.privateEnvironment[item.Resource] = cloneEnvironment(privateEnvironment)
	}
	if status, ok := s.statuses[item.Resource]; ok && status.Running {
		status.Message = "healthy"
		status.Observation.Readiness = "ready"
		s.statuses[item.Resource] = status
	}
	s.mu.Unlock()
	return nil
}

type serviceLaunchDirectories struct{ dataDir, configDir, logDir string }

func (s *ServiceSupervisor) prepareLaunch(item Item) (serviceLaunchDirectories, error) {
	root := filepath.Join(s.appDataDir, "resources", item.Resource)
	dirs := serviceLaunchDirectories{filepath.Join(root, "data"), filepath.Join(root, "config"), filepath.Join(root, "logs")}
	for _, directory := range []string{dirs.dataDir, dirs.configDir, dirs.logDir} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return serviceLaunchDirectories{}, fmt.Errorf("create bundled service directory: %w", err)
		}
	}
	return dirs, nil
}

func (dirs serviceLaunchDirectories) environment(service *Service, ports map[string]int) (map[string]string, []string) {
	overrides := map[string]string{"VROOLI_RESOURCE_DATA_DIR": dirs.dataDir, "VROOLI_RESOURCE_CONFIG_DIR": dirs.configDir, "VROOLI_RESOURCE_LOGS_DIR": dirs.logDir, "RESOURCE_DATA_DIR": dirs.dataDir, "RESOURCE_CONFIG_DIR": dirs.configDir, "RESOURCE_LOGS_DIR": dirs.logDir, "VROOLI_MANAGED_PROVIDER": "managed-private"}
	for key, value := range service.Environment {
		overrides[key] = value
	}
	for name, port := range ports {
		envName := servicePortEnvName(name)
		overrides[envName] = fmt.Sprintf("%d", port)
		overrides["VROOLI_"+envName] = fmt.Sprintf("%d", port)
	}
	env := envkitEntriesToMap(envkit.WithOverlay(envkit.Env(os.Environ()), envkit.Resource, envkitMap(overrides)))
	for key, value := range env {
		env[key] = expandServiceTemplate(value, env)
	}
	arguments := make([]string, len(service.Arguments))
	for index, value := range service.Arguments {
		arguments[index] = expandServiceTemplate(value, env)
	}
	return env, arguments
}

func (s *ServiceSupervisor) wait(resource string, running *runningService) {
	err := running.cmd.Wait()
	running.logFile.Close()
	message := "stopped"
	if err != nil {
		message = err.Error()
	}
	s.mu.Lock()
	delete(s.running, resource)
	s.statuses[resource] = ServiceStatus{Resource: resource, LogPath: running.status.LogPath, Message: message}
	s.mu.Unlock()
}

func (s *ServiceSupervisor) Stop(ctx context.Context) error {
	s.mu.RLock()
	services := make([]*runningService, 0, len(s.running))
	for _, service := range s.running {
		services = append(services, service)
	}
	s.mu.RUnlock()
	for _, service := range services {
		if service.cmd.Process == nil {
			continue
		}
		_ = infra.GracefulStopProcess(service.cmd.Process)
	}
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(25 * time.Millisecond)
		defer ticker.Stop()
		for {
			s.mu.RLock()
			empty := len(s.running) == 0
			s.mu.RUnlock()
			if empty {
				close(done)
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		for _, service := range services {
			if service.cmd.Process != nil {
				_ = service.cmd.Process.Kill()
			}
		}
		return ctx.Err()
	}
}

func (s *ServiceSupervisor) Statuses() map[string]ServiceStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]ServiceStatus, len(s.statuses))
	for resource, status := range s.statuses {
		result[resource] = status
	}
	return result
}

// Environment returns app-scoped resource connection settings. It is for the
// bundle's own service launcher only and is never included in status or logs.
func (s *ServiceSupervisor) Environment() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]string)
	for _, binding := range s.bindings {
		for key, value := range binding.Environment {
			result[key] = value
		}
	}
	for _, environment := range s.privateEnvironment {
		for key, value := range environment {
			result[key] = value
		}
	}
	return result
}

func cloneEnvironment(input map[string]string) map[string]string {
	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func envkitMap(values map[string]string) envkit.Env {
	entries := make(envkit.Env, 0, len(values))
	for key, value := range values {
		entries = append(entries, key+"="+value)
	}
	return entries
}

func envkitEntriesToMap(entries envkit.Env) map[string]string {
	values := make(map[string]string, len(entries))
	for _, entry := range entries {
		if key, value, ok := strings.Cut(entry, "="); ok {
			values[key] = value
		}
	}
	return values
}

func environmentList(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, key+"="+values[key])
	}
	return result
}

func expandServiceTemplate(value string, values map[string]string) string {
	return os.Expand(value, func(key string) string {
		if resolved, ok := values[key]; ok {
			return resolved
		}
		return "${" + key + "}"
	})
}

func writeServiceConfig(config *resourcedeployment.ServiceConfig, configDir string, values map[string]string) error {
	if config == nil {
		return nil
	}
	if err := config.Validate(); err != nil {
		return err
	}
	path := filepath.Join(configDir, filepath.FromSlash(config.Path))
	if rel, err := filepath.Rel(configDir, path); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("config path escapes app resource config directory")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(expandServiceTemplate(config.Content, values)), 0o600)
}

func runServiceBootstrap(ctx context.Context, service *Service, artifactPath string, launch serviceLaunchDirectories, values map[string]string) error {
	if service.Bootstrap == nil {
		return nil
	}
	bootstrap := service.Bootstrap
	if err := bootstrap.Validate(); err != nil {
		return err
	}
	marker := filepath.Join(launch.dataDir, filepath.FromSlash(bootstrap.Marker))
	if _, err := os.Stat(marker); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect bundled service bootstrap marker: %w", err)
	}
	artifactRoot := filepath.Dir(artifactPath)
	if strings.EqualFold(strings.TrimSpace(service.Layout), "dir") {
		artifactRoot = artifactPath
	}
	environment := cloneEnvironment(values)
	var passwordPath string
	if strings.TrimSpace(bootstrap.PasswordEnv) != "" {
		password := environment[bootstrap.PasswordEnv]
		if password == "" {
			return fmt.Errorf("bundled service bootstrap requires %s", bootstrap.PasswordEnv)
		}
		if strings.TrimSpace(bootstrap.PasswordFile) == "" {
			return fmt.Errorf("bundled service bootstrap password_file is required")
		}
		passwordPath = filepath.Join(launch.configDir, filepath.FromSlash(bootstrap.PasswordFile))
		if err := os.MkdirAll(filepath.Dir(passwordPath), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(passwordPath, []byte(password+"\n"), 0o600); err != nil {
			return err
		}
		defer os.Remove(passwordPath)
	}
	arguments := make([]string, len(bootstrap.Arguments))
	for index, argument := range bootstrap.Arguments {
		arguments[index] = expandServiceTemplate(argument, environment)
	}
	executable := filepath.Join(artifactRoot, filepath.FromSlash(bootstrap.Executable))
	command := exec.CommandContext(ctx, executable, arguments...)
	command.Dir = artifactRoot
	command.Env = environmentList(environment)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("run bundled service bootstrap: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if _, err := os.Stat(marker); err != nil {
		return fmt.Errorf("bundled service bootstrap completed without marker %s: %w", bootstrap.Marker, err)
	}
	return nil
}

func waitForServiceHealth(parent context.Context, checks []HealthCheck, values map[string]string, ports ...[]ServicePort) error {
	if len(checks) == 0 {
		return nil
	}
	deadline := 60 * time.Second
	for _, check := range checks {
		if check.TimeoutSeconds > 0 && time.Duration(check.TimeoutSeconds)*time.Second > deadline {
			deadline = time.Duration(check.TimeoutSeconds) * time.Second
		}
	}
	ctx, cancel := context.WithTimeout(parent, deadline)
	defer cancel()
	client := &http.Client{Timeout: 5 * time.Second}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		ready := true
		for _, check := range checks {
			var servicePorts []ServicePort
			if len(ports) > 0 {
				servicePorts = ports[0]
			}
			target := expandHealthCheckTarget(check.Target, values, servicePorts)
			if check.Type == "tcp" {
				dialer := net.Dialer{Timeout: 5 * time.Second}
				connection, dialErr := dialer.DialContext(ctx, "tcp", target)
				if dialErr != nil {
					ready = false
					break
				}
				_ = connection.Close()
				continue
			}
			request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
			if err != nil {
				return fmt.Errorf("build health request: %w", err)
			}
			response, err := client.Do(request)
			if err != nil {
				ready = false
				break
			}
			_ = response.Body.Close()
			// Vault (and other APIs with the same convention) uses 501 to mean
			// "reachable but initialization is required". A transport response
			// cannot turn that into application readiness, even if a stale plan
			// happened to list 501 among acceptable probe statuses.
			if response.StatusCode == http.StatusNotImplemented {
				ready = false
				break
			}
			if len(check.ExpectedStatus) > 0 && !containsHealthStatus(check.ExpectedStatus, response.StatusCode) {
				ready = false
				break
			}
		}
		if ready {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// expandHealthCheckTarget keeps legacy resource plans compatible with runtime
// port allocation. Resource manifests historically recorded health targets
// with their default host port (for example, MinIO on 9000), while the
// desktop supervisor deliberately allocates a private ephemeral port. When a
// check points at a declared loopback port, resolve that port through the
// corresponding RESOURCE_PORT_* value before probing it.
func expandHealthCheckTarget(target string, values map[string]string, ports []ServicePort) string {
	expanded := expandServiceTemplate(target, values)
	parsed, err := url.Parse(expanded)
	if err != nil || parsed.Hostname() == "" {
		return expanded
	}
	host := parsed.Hostname()
	if host != "127.0.0.1" && host != "localhost" {
		return expanded
	}
	declaredPort, err := strconv.Atoi(parsed.Port())
	if err != nil {
		return expanded
	}
	for _, port := range ports {
		if port.Host != declaredPort {
			continue
		}
		actual, ok := values[servicePortEnvName(port.Name)]
		if !ok || strings.TrimSpace(actual) == "" {
			return expanded
		}
		parsed.Host = net.JoinHostPort(host, actual)
		return parsed.String()
	}
	return expanded
}

func containsHealthStatus(expected []int, actual int) bool {
	for _, status := range expected {
		if status == actual {
			return true
		}
	}
	return false
}

func allocateServicePorts(ports []ServicePort) (map[string]int, error) {
	allocated := make(map[string]int, len(ports))
	for _, port := range ports {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return nil, err
		}
		assigned := listener.Addr().(*net.TCPAddr).Port
		if err := listener.Close(); err != nil {
			return nil, err
		}
		allocated[port.Name] = assigned
	}
	return allocated, nil
}

func servicePortEnvName(name string) string {
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, name)
	return "RESOURCE_PORT_" + strings.ToUpper(name)
}
