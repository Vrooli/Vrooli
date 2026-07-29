package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

type managedServiceDriver struct{}

func (managedServiceDriver) Name() string { return "managed-service" }

// ManagedServiceOwnerLifecycle binds the broker's owner-only management
// surface to the same verified driver used by the control plane. It accepts no
// caller-selected endpoint, PID, command, or artifact: all lifecycle input is
// fixed by the manifest and Controller that created the host.
type ManagedServiceOwnerLifecycle struct {
	Controller *Controller
	Manifest   ResourceManifest
}

func NewManagedServiceOwnerLifecycle(controller *Controller, manifest ResourceManifest) (OwnerLifecycle, error) {
	if controller == nil || strings.TrimSpace(manifest.Name) == "" {
		return nil, fmt.Errorf("managed-service owner lifecycle requires controller and manifest")
	}
	return ManagedServiceOwnerLifecycle{Controller: controller, Manifest: manifest}, nil
}

func (l ManagedServiceOwnerLifecycle) Manage(ctx context.Context, instance ManagedInstance, action string) (any, error) {
	if l.Controller == nil || instance.Resource != l.Manifest.Name || instance.Provider != resourcedeployment.ProviderManagedShared {
		return nil, fmt.Errorf("owner lifecycle does not match a managed shared %s instance", l.Manifest.Name)
	}
	driver := managedServiceDriver{}
	item := Resource{Name: l.Manifest.Name}
	switch action {
	case "start", "restart":
		if err := driver.runUserHosted(ctx, l.Controller, item, l.Manifest, action, io.Discard); err != nil {
			return nil, err
		}
		return driver.Status(ctx, l.Controller, item, l.Manifest, false)
	case "stop":
		if err := driver.runPrivate(ctx, l.Controller, item, l.Manifest, action, io.Discard); err != nil {
			return nil, err
		}
		return driver.Status(ctx, l.Controller, item, l.Manifest, true)
	case "inspect":
		return driver.Status(ctx, l.Controller, item, l.Manifest, false)
	case "logs":
		var output bytes.Buffer
		if err := driver.runPrivate(ctx, l.Controller, item, l.Manifest, action, &output); err != nil {
			return nil, err
		}
		return map[string]string{"logs": output.String()}, nil
	default:
		return nil, fmt.Errorf("unsupported managed-service owner action %q", action)
	}
}

func (d managedServiceDriver) Status(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, fast bool) (Status, error) {
	status := Status{Resource: item, StatusCode: StatusCodeOK, Message: "stopped"}
	if err := ensureSupportedPlatform(manifest); err != nil {
		status.StatusCode = StatusCodeUnsupportedPlatform
		status.Message = err.Error()
		status.ProbeError = err.Error()
		return status, nil
	}
	mode, err := managedServiceProvider(manifest, nil)
	if err != nil {
		status.StatusCode = StatusCodeUnavailable
		status.Message = "managed-service provider policy is invalid"
		status.ProbeError = err.Error()
		return status, nil
	}
	if mode != resourcedeployment.ProviderManagedPrivate && mode != resourcedeployment.ProviderManagedShared {
		status.Installed = true
		status.Message = fmt.Sprintf("%s provider is not locally managed", mode)
		return status, nil
	}

	artifactPath, err := managedServiceArtifactPath(controller, manifest)
	if err != nil {
		return status, err
	}
	artifact, err := manifest.ManagedService.Artifact.ForCurrentPlatform()
	if err != nil {
		status.StatusCode = StatusCodeUnavailable
		status.Message = "verified managed-service artifact is unsupported on this platform"
		status.ProbeError = err.Error()
		return status, nil
	}
	if err := artifact.VerifyFile(artifactPath); err != nil {
		status.StatusCode = StatusCodeUnavailable
		status.Message = "verified managed-service artifact is unavailable"
		status.ProbeError = err.Error()
		return status, nil
	}
	status.Installed = true
	supervisor, _, err := managedServiceSupervisorFor(manifest.Name)
	if err != nil {
		return status, err
	}
	state, running, err := supervisor.Status()
	if err != nil {
		status.StatusCode = StatusCodeCommandError
		status.Message = "managed-service state is invalid"
		status.ProbeError = err.Error()
		return status, nil
	}
	if raw, err := json.Marshal(state); err == nil {
		status.Raw = raw
	}
	if !running {
		healthy := false
		status.Healthy = &healthy
		status.Health = "stopped"
		return status, nil
	}
	status.Running = true
	healthy := true
	status.Healthy = &healthy
	status.Health = "running"
	status.Message = "running"
	if fast || len(manifest.HealthChecks) == 0 {
		return status, nil
	}
	health, err := controller.runResourceHealthChecks(ctx, manifest)
	if err != nil {
		status.StatusCode = StatusCodeCommandError
		status.Message = "health checks failed"
		status.ProbeError = err.Error()
		return status, nil
	}
	healthy = health.Healthy
	status.Healthy = &healthy
	if healthy {
		status.Health = "healthy"
		status.Message = "healthy"
	} else {
		status.Health = "unhealthy"
		status.Message = health.Message
		if status.Message == "" {
			status.Message = "unhealthy"
		}
	}
	return status, nil
}

func (d managedServiceDriver) Run(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, action string, args []string, stdout, stderr io.Writer) error {
	if err := ensureSupportedPlatform(manifest); err != nil {
		return managedServiceDriverError(item.Name, action, "Platform", err)
	}
	switch action {
	case "status":
		mode, err := managedServiceProvider(manifest, args)
		if err != nil {
			return managedServiceDriverError(item.Name, action, "Provider", err)
		}
		if mode == resourcedeployment.ProviderAttachOnly {
			return d.statusAttachOnly(ctx, item, manifest, args, stdout)
		}
		status, err := d.Status(ctx, controller, item, manifest, !containsString(args, "--no-fast"))
		if err != nil {
			return err
		}
		if containsString(args, "--format") && nextArgValue(args, "--format") == "json" {
			return json.NewEncoder(stdout).Encode(map[string]any{
				"installed": status.Installed, "running": status.Running, "healthy": status.Healthy,
				"health": status.Health, "message": status.Message, "provider": mode,
			})
		}
		_, err = fmt.Fprintf(stdout, "%s: %s\n", item.Name, status.Message)
		return err
	case "install":
		return d.verifyArtifact(controller, manifest)
	case "start", "restart", "stop", "uninstall", "logs":
		mode, err := managedServiceProvider(manifest, args)
		if err != nil {
			return managedServiceDriverError(item.Name, action, "Provider", err)
		}
		if mode == resourcedeployment.ProviderManagedShared && (action == "start" || action == "restart") {
			return d.runUserHosted(ctx, controller, item, manifest, action, stdout)
		}
		if mode == resourcedeployment.ProviderManagedDiscovered && (action == "start" || action == "restart") {
			return d.runDiscovered(ctx, controller, item, manifest, action, args)
		}
		if mode != resourcedeployment.ProviderManagedPrivate {
			return managedServiceDriverError(item.Name, action, "Provider", fmt.Errorf("%s provider does not grant local lifecycle authority; use the broker or provider-specific client", mode))
		}
		return d.runPrivate(ctx, controller, item, manifest, action, stdout)
	default:
		return managedServiceDriverError(item.Name, action, "Driver", fmt.Errorf("action %q is not supported by driver %q", action, d.Name()))
	}
}

// runUserHosted owns shared lifecycle mechanics while a resource registration
// owns its bootstrap protocol and management material.
func (d managedServiceDriver) runUserHosted(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, action string, stdout io.Writer) error {
	bootstrap, ok := managedSharedBootstrapperFor(manifest.Name)
	if !ok {
		return fmt.Errorf("user-hosted bootstrap adapter is not registered for %s", manifest.Name)
	}
	host, err := OpenUserResourceHost(defaultManagedSharedSecureStore(), "local-user")
	if err != nil {
		return err
	}
	// The instance ID is stable across restart and belongs to the supervisor.
	supervisor, _, err := managedServiceSupervisorFor(manifest.Name)
	if err != nil {
		return err
	}
	state, _, err := supervisor.Status()
	if err != nil {
		return err
	}
	instanceID := state.InstanceID
	if instanceID == "" {
		instanceID = manifest.Name + "-user-host"
	}
	if err := host.SecureStorageReady(instanceID); err != nil {
		return fmt.Errorf("user-hosted %s requires operating-system secure storage: %w", manifest.Name, err)
	}
	// Vault and other resource-native shared bootstrappers must run after the
	// verified process is reachable but before generic health claims success.
	// Calling runPrivate here would reject an intentionally uninitialized Vault
	// before its secure bootstrapper can initialize it.
	switch action {
	case "start":
		if err := d.startPrivate(controller, manifest, supervisor); err != nil {
			return err
		}
	case "restart":
		stopCtx, cancel := managedServiceStopContext(ctx, manifest)
		err := supervisor.Stop(stopCtx)
		cancel()
		if err != nil {
			return err
		}
		if err := d.startPrivate(controller, manifest, supervisor); err != nil {
			return err
		}
	case "stop", "uninstall", "logs":
		return d.runPrivate(ctx, controller, item, manifest, action, stdout)
	default:
		return fmt.Errorf("unsupported shared managed-service action %q", action)
	}
	state, running, err := supervisor.Status()
	if err != nil || !running {
		return fmt.Errorf("verify user-hosted %s ownership: %w", manifest.Name, err)
	}
	endpoint, err := managedServiceLoopbackEndpoint(manifest)
	if err != nil {
		return err
	}
	attestation, err := supervisor.Attest(endpoint, "broker-owner:"+host.OwnerScope)
	if err != nil {
		return err
	}
	if err := bootstrap(ctx, host, ManagedInstance{ID: state.InstanceID, Resource: manifest.Name, Provider: resourcedeployment.ProviderManagedShared, OwnerScope: host.OwnerScope, CapabilityVersion: manifest.ManagedService.Artifact.Version, Endpoint: endpoint, Attestation: attestation}, "control-plane"); err != nil {
		return err
	}
	if err := waitForManagedServiceHealth(ctx, controller, manifest); err != nil {
		return err
	}
	return verifyManagedServiceRunning(supervisor)
}

func managedServiceLoopbackEndpoint(manifest ResourceManifest) (string, error) {
	for _, port := range manifest.Ports {
		if port.Name == "http" && port.Host > 0 {
			return fmt.Sprintf("http://127.0.0.1:%d", port.Host), nil
		}
	}
	return "", fmt.Errorf("managed-service %s has no loopback HTTP port", manifest.Name)
}

func (d managedServiceDriver) statusAttachOnly(ctx context.Context, item Resource, manifest ResourceManifest, args []string, stdout io.Writer) error {
	endpoint, err := managedServiceEndpointArg(args)
	if err != nil {
		return managedServiceDriverError(item.Name, "status", "Provider", err)
	}
	base, err := url.Parse(endpoint)
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return managedServiceDriverError(item.Name, "status", "Provider", fmt.Errorf("attach-only endpoint must be an HTTP(S) URL without credentials, query, or fragment"))
	}
	path := manifest.ManagedService.AttachHealthPath
	requestURL := base.ResolveReference(&url.URL{Path: path}).String()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return managedServiceDriverError(item.Name, "status", "Provider", fmt.Errorf("attach-only endpoint health request failed: %w", err))
	}
	defer response.Body.Close()
	expected := []int(nil)
	for _, check := range manifest.HealthChecks {
		if check.Type == "http" {
			expected = check.ExpectedStatus
			break
		}
	}
	if len(expected) == 0 && response.StatusCode >= 200 && response.StatusCode < 300 {
		_, err := fmt.Fprintf(stdout, "%s: attach-only endpoint healthy\n", item.Name)
		return err
	}
	for _, status := range expected {
		if response.StatusCode == status {
			_, err := fmt.Fprintf(stdout, "%s: attach-only endpoint healthy\n", item.Name)
			return err
		}
	}
	return managedServiceDriverError(item.Name, "status", "Provider", fmt.Errorf("attach-only endpoint returned unexpected HTTP status %d", response.StatusCode))
}

func managedServiceEndpointArg(args []string) (string, error) {
	for index, arg := range args {
		if arg == "--endpoint" {
			if index+1 < len(args) && strings.TrimSpace(args[index+1]) != "" {
				return strings.TrimSpace(args[index+1]), nil
			}
			break
		}
		if value, ok := strings.CutPrefix(arg, "--endpoint="); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value), nil
		}
	}
	return "", fmt.Errorf("attach-only provider requires --endpoint <url>")
}

func (d managedServiceDriver) verifyArtifact(controller *Controller, manifest ResourceManifest) error {
	path, err := managedServiceArtifactPath(controller, manifest)
	if err != nil {
		return err
	}
	artifact, err := manifest.ManagedService.Artifact.ForCurrentPlatform()
	if err != nil {
		return err
	}
	return artifact.VerifyFile(path)
}

func (d managedServiceDriver) runPrivate(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, action string, stdout io.Writer) error {
	supervisor, _, err := managedServiceSupervisorFor(manifest.Name)
	if err != nil {
		return err
	}
	switch action {
	case "start":
		if err := d.startPrivate(controller, manifest, supervisor); err != nil {
			return err
		}
		if err := d.bootstrapPrivate(ctx, manifest, supervisor); err != nil {
			return err
		}
		if err := waitForManagedServiceHealth(ctx, controller, manifest); err != nil {
			return err
		}
		return verifyManagedServiceRunning(supervisor)
	case "restart":
		stopCtx, cancel := managedServiceStopContext(ctx, manifest)
		err := supervisor.Stop(stopCtx)
		cancel()
		if err != nil {
			return err
		}
		if err := d.startPrivate(controller, manifest, supervisor); err != nil {
			return err
		}
		if err := d.bootstrapPrivate(ctx, manifest, supervisor); err != nil {
			return err
		}
		if err := waitForManagedServiceHealth(ctx, controller, manifest); err != nil {
			return err
		}
		return verifyManagedServiceRunning(supervisor)
	case "stop", "uninstall":
		stopCtx, cancel := managedServiceStopContext(ctx, manifest)
		defer cancel()
		return supervisor.Stop(stopCtx)
	case "logs":
		return supervisor.Logs(stdout)
	default:
		return fmt.Errorf("unsupported private managed-service action %q", action)
	}
}

func (d managedServiceDriver) bootstrapPrivate(ctx context.Context, manifest ResourceManifest, supervisor *ManagedServiceSupervisor) error {
	bootstrap, ok := managedPrivateBootstrapperFor(manifest.Name)
	if !ok {
		return nil
	}
	state, running, err := supervisor.Status()
	if err != nil {
		return fmt.Errorf("verify private %s ownership before bootstrap: %w", manifest.Name, err)
	}
	if !running {
		return fmt.Errorf("verify private %s ownership before bootstrap: service is not running", manifest.Name)
	}
	endpoint, err := managedServiceLoopbackEndpoint(manifest)
	if err != nil {
		return err
	}
	return bootstrap(ctx, state, endpoint)
}

func (d managedServiceDriver) startPrivate(controller *Controller, manifest ResourceManifest, supervisor *ManagedServiceSupervisor) error {
	if err := d.verifyArtifact(controller, manifest); err != nil {
		return err
	}
	if err := ensureManagedServicePortsAvailable(manifest); err != nil {
		return err
	}
	path, err := managedServiceArtifactPath(controller, manifest)
	if err != nil {
		return err
	}
	return d.startPrivateAt(controller, manifest, supervisor, path)
}

func (d managedServiceDriver) startPrivateAt(controller *Controller, manifest ResourceManifest, supervisor *ManagedServiceSupervisor, path string) error {
	if err := d.verifyArtifactAt(manifest, path); err != nil {
		return err
	}
	env := resourceEnvForResource(controller.Root, controller.Home, manifest.Name)
	for _, port := range manifest.Ports {
		if port.Host > 0 {
			env = setEnvValue(env, managedServicePortEnvName(port.Name), fmt.Sprintf("%d", port.Host))
		}
	}
	for key, value := range manifest.ManagedService.Environment {
		env = setEnvValue(env, key, value)
	}
	env = renderManagedServiceEnvironment(env, manifest.ManagedService.Environment)
	if err := writeManagedServiceConfig(manifest, env); err != nil {
		return err
	}
	arguments := renderManagedServiceValues(manifest.ManagedService.Arguments, env)
	artifact, err := manifest.ManagedService.Artifact.ForCurrentPlatform()
	if err != nil {
		return err
	}
	_, err = supervisor.Start(path, artifact, arguments, env, filepath.Dir(path))
	return err
}

func (d managedServiceDriver) verifyArtifactAt(manifest ResourceManifest, path string) error {
	artifact, err := manifest.ManagedService.Artifact.ForCurrentPlatform()
	if err != nil {
		return err
	}
	return artifact.VerifyFile(path)
}

// runDiscovered accepts only an explicit executable candidate, verifies it
// against the manifest digest and version, then launches it under Vrooli's
// supervisor with Vrooli-owned configuration and state. It never discovers or
// adopts a running endpoint.
func (d managedServiceDriver) runDiscovered(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, action string, args []string) error {
	path, err := managedDiscoveredExecutable(args)
	if err != nil {
		return managedServiceDriverError(item.Name, action, "Provider", err)
	}
	if err := d.verifyArtifactAt(manifest, path); err != nil {
		return managedServiceDriverError(item.Name, action, "Provider", err)
	}
	if err := verifyManagedDiscoveredVersion(ctx, path, manifest.ManagedService.Artifact.Version); err != nil {
		return managedServiceDriverError(item.Name, action, "Provider", err)
	}
	supervisor, _, err := managedServiceSupervisorFor(manifest.Name)
	if err != nil {
		return err
	}
	if action == "restart" {
		stopCtx, cancel := managedServiceStopContext(ctx, manifest)
		err = supervisor.Stop(stopCtx)
		cancel()
		if err != nil {
			return err
		}
	}
	if err := ensureManagedServicePortsAvailable(manifest); err != nil {
		return err
	}
	if err := d.startPrivateAt(controller, manifest, supervisor, path); err != nil {
		return err
	}
	if err := d.bootstrapPrivate(ctx, manifest, supervisor); err != nil {
		return err
	}
	if err := waitForManagedServiceHealth(ctx, controller, manifest); err != nil {
		return err
	}
	return verifyManagedServiceRunning(supervisor)
}

func managedDiscoveredExecutable(args []string) (string, error) {
	for index, arg := range args {
		value := ""
		if arg == "--executable" && index+1 < len(args) {
			value = args[index+1]
		}
		if candidate, ok := strings.CutPrefix(arg, "--executable="); ok {
			value = candidate
		}
		if strings.TrimSpace(value) == "" {
			continue
		}
		if !filepath.IsAbs(value) {
			return "", fmt.Errorf("managed-discovered executable path must be absolute")
		}
		path, err := filepath.Abs(value)
		if err != nil {
			return "", err
		}
		info, err := os.Stat(path)
		if err != nil {
			return "", fmt.Errorf("read discovered executable: %w", err)
		}
		if info.IsDir() || info.Mode()&0o111 == 0 {
			return "", fmt.Errorf("discovered executable is not executable")
		}
		return path, nil
	}
	return "", fmt.Errorf("managed-discovered provider requires --executable <absolute-path>")
}

func verifyManagedDiscoveredVersion(ctx context.Context, path, version string) error {
	if strings.TrimSpace(version) == "" {
		return fmt.Errorf("manifest artifact version is required")
	}
	output, err := exec.CommandContext(ctx, path, "--version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("run discovered executable version check: %w", err)
	}
	if !strings.Contains(string(output), version) {
		return fmt.Errorf("discovered executable version does not match required %s", version)
	}
	return nil
}

// ensureManagedServicePortsAvailable reserves each declared loopback port long
// enough to prove it is not already owned. This prevents a failed child from
// being mistaken for healthy by a different process that happens to serve the
// manifest health endpoint. The child is still verified by its ownership token
// after health succeeds, so this preflight is never the sole identity proof.
func ensureManagedServicePortsAvailable(manifest ResourceManifest) error {
	for _, port := range manifest.Ports {
		if port.Host <= 0 {
			continue
		}
		listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port.Host)))
		if err != nil {
			return fmt.Errorf("managed-service port %s (%d) is already unavailable: %w", port.Name, port.Host, err)
		}
		if err := listener.Close(); err != nil {
			return fmt.Errorf("release managed-service port %s: %w", port.Name, err)
		}
	}
	return nil
}

func verifyManagedServiceRunning(supervisor *ManagedServiceSupervisor) error {
	_, running, err := supervisor.Status()
	if err != nil {
		return fmt.Errorf("verify managed-service ownership after health: %w", err)
	}
	if !running {
		return fmt.Errorf("managed-service exited before it could be verified healthy")
	}
	return nil
}

func managedServicePortEnvName(name string) string {
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, name)
	return "RESOURCE_PORT_" + strings.ToUpper(name)
}

func writeManagedServiceConfig(manifest ResourceManifest, env []string) error {
	if manifest.ManagedService == nil || manifest.ManagedService.Config == nil {
		return nil
	}
	config := manifest.ManagedService.Config
	if err := config.Validate(); err != nil {
		return err
	}
	values := managedServiceEnvValues(env)
	root := values["RESOURCE_CONFIG_DIR"]
	if root == "" {
		return fmt.Errorf("managed-service config requires RESOURCE_CONFIG_DIR")
	}
	path := filepath.Join(root, filepath.FromSlash(config.Path))
	if rel, err := filepath.Rel(root, path); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("managed-service config path escapes resource config root")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(renderManagedServiceValue(config.Content, values)), 0o600)
}

func renderManagedServiceValues(arguments, env []string) []string {
	values := managedServiceEnvValues(env)
	rendered := make([]string, len(arguments))
	for i, argument := range arguments {
		rendered[i] = renderManagedServiceValue(argument, values)
	}
	return rendered
}

func managedServiceEnvValues(env []string) map[string]string {
	values := map[string]string{}
	for _, entry := range env {
		if key, value, ok := strings.Cut(entry, "="); ok {
			values[key] = value
		}
	}
	return values
}

func renderManagedServiceEnvironment(env []string, overrides map[string]string) []string {
	for pass := 0; pass < 3; pass++ {
		values := managedServiceEnvValues(env)
		changed := false
		for key, value := range overrides {
			rendered := renderManagedServiceValue(value, values)
			if values[key] != rendered {
				env = setEnvValue(env, key, rendered)
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	return env
}
func renderManagedServiceValue(value string, values map[string]string) string {
	return os.Expand(value, func(key string) string {
		if value, ok := values[key]; ok {
			return value
		}
		return "${" + key + "}"
	})
}

func managedServiceProvider(manifest ResourceManifest, args []string) (resourcedeployment.ProviderMode, error) {
	if manifest.ManagedService == nil {
		return "", fmt.Errorf("managed_service is required")
	}
	request := resourcedeployment.ProviderRequest{Target: resourcedeployment.ProviderTargetControlPlane, SharedConsented: containsString(args, "--shared-consent")}
	for i, arg := range args {
		if arg == "--provider" {
			if i+1 >= len(args) || strings.TrimSpace(args[i+1]) == "" {
				return "", fmt.Errorf("--provider requires a provider mode")
			}
			request.Mode = resourcedeployment.ProviderMode(args[i+1])
		}
		if value, ok := strings.CutPrefix(arg, "--provider="); ok {
			request.Mode = resourcedeployment.ProviderMode(value)
		}
	}
	return manifest.ManagedService.ProviderPolicy.ResolveProvider(request)
}

func waitForManagedServiceHealth(parent context.Context, controller *Controller, manifest ResourceManifest) error {
	if len(manifest.HealthChecks) == 0 {
		return nil
	}
	seconds := manifest.Lifecycle.StartTimeoutSeconds
	if seconds <= 0 {
		seconds = 60
	}
	ctx, cancel := context.WithTimeout(parent, time.Duration(seconds)*time.Second)
	defer cancel()
	for {
		health, err := controller.runResourceHealthChecks(ctx, manifest)
		if err == nil && health.Healthy {
			return nil
		}
		select {
		case <-ctx.Done():
			if err != nil {
				return fmt.Errorf("managed-service health check did not pass: %w", err)
			}
			return fmt.Errorf("managed-service health check did not pass before startup timeout")
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func managedServiceDriverError(resource, operation, category string, err error) *Error {
	return &Error{Code: ErrorCodeCommandUnavailable, Resource: resource, Operation: operation, Category: category, Err: err}
}
