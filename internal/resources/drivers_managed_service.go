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

	"github.com/vrooli/vrooli/internal/resources/securestore"
	credentialauthority "github.com/vrooli/vrooli/internal/secrets"
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
		var result any
		err := withManagedServiceLifecycleLock(l.Manifest.Name, func() error {
			if err := driver.runUserHosted(ctx, l.Controller, item, l.Manifest, action, io.Discard); err != nil {
				return err
			}
			var err error
			result, err = driver.Status(ctx, l.Controller, item, l.Manifest, false)
			return err
		})
		return result, err
	case "stop":
		var result any
		err := withManagedServiceLifecycleLock(l.Manifest.Name, func() error {
			if err := driver.runPrivate(ctx, l.Controller, item, l.Manifest, action, io.Discard); err != nil {
				return err
			}
			var err error
			result, err = driver.Status(ctx, l.Controller, item, l.Manifest, true)
			return err
		})
		return result, err
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
	artifact, err := managedServiceArtifactForLaunch(ctx, manifest)
	if err != nil {
		status.StatusCode = StatusCodeUnavailable
		status.Message = "verified managed-service artifact is unavailable for this host"
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
		return d.withCompanionStatus(status, manifest)
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
	return d.withCompanionStatus(status, manifest)
}

func (d managedServiceDriver) withCompanionStatus(status Status, manifest ResourceManifest) (Status, error) {
	companions, err := companionStatuses(manifest.Name, manifest.Companions)
	if err != nil {
		status.StatusCode = StatusCodeCommandError
		status.Message = "companion status failed"
		status.ProbeError = err.Error()
		return status, nil
	}
	if len(companions) == 0 {
		return status, nil
	}
	status.Raw = statusRawWithCompanions(status.Raw, companions)
	if down := downCompanions(companions); len(down) > 0 {
		healthy := false
		status.Healthy = &healthy
		status.Health = "unhealthy"
		status.Message = companionDownMessage(manifest.Name, down)
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
		return ensureManagedServiceArtifact(ctx, controller, manifest)
	case "start", "restart", "stop", "uninstall", "logs":
		mode, err := managedServiceProvider(manifest, args)
		if err != nil {
			return managedServiceDriverError(item.Name, action, "Provider", err)
		}
		if managedServiceLifecycleAction(action) {
			release, err := acquireManagedServiceLifecycleLock(manifest.Name)
			if err != nil {
				return managedServiceDriverError(item.Name, action, "Lifecycle", err)
			}
			defer release()
		}
		if mode == resourcedeployment.ProviderManagedShared {
			if action == "start" || action == "restart" {
				if err := migrateLegacyDockerStorage(ctx, controller, manifest); err != nil {
					return err
				}
			}
			return d.runUserHosted(ctx, controller, item, manifest, action, stdout)
		}
		if mode == resourcedeployment.ProviderManagedDiscovered && (action == "start" || action == "restart") {
			return d.runDiscovered(ctx, controller, item, manifest, action, args)
		}
		if mode != resourcedeployment.ProviderManagedPrivate {
			return managedServiceDriverError(item.Name, action, "Provider", fmt.Errorf("%s provider does not grant local lifecycle authority; use the broker or provider-specific client", mode))
		}
		if action == "start" || action == "restart" {
			if err := migrateLegacyDockerStorage(ctx, controller, manifest); err != nil {
				return err
			}
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
		if err := d.startPrivate(ctx, controller, manifest, supervisor); err != nil {
			return err
		}
	case "restart":
		stopCtx, cancel := managedServiceStopContext(ctx, manifest)
		err := stopManagedService(stopCtx, manifest, supervisor)
		cancel()
		if err != nil {
			return err
		}
		if err := d.startPrivate(ctx, controller, manifest, supervisor); err != nil {
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
	for _, preferred := range []string{"http", "api"} {
		for _, port := range manifest.Ports {
			if port.Name == preferred && port.Host > 0 {
				return fmt.Sprintf("http://127.0.0.1:%d", port.Host), nil
			}
		}
	}
	for _, port := range manifest.Ports {
		if port.Host > 0 {
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

func (d managedServiceDriver) runPrivate(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, action string, stdout io.Writer) error {
	supervisor, _, err := managedServiceSupervisorFor(manifest.Name)
	if err != nil {
		return err
	}
	switch action {
	case "start":
		if err := d.startPrivate(ctx, controller, manifest, supervisor); err != nil {
			return err
		}
		if err := d.bootstrapPrivate(ctx, manifest, supervisor); err != nil {
			return err
		}
		// Companions are part of the resource's readiness contract. Start them
		// before the health wait so a companion that supplies capacity liveness
		// can create/heartbeat the resource claim while the managed service is
		// coming up. Waiting first deadlocks resources whose status is unhealthy
		// while their required companion is down.
		startCompanions(manifest.Name, manifest.Companions, manifest.Orchestration.RecoveryAttempts, os.Stderr)
		if err := waitForManagedServiceHealth(ctx, controller, manifest); err != nil {
			return err
		}
		if err := verifyManagedServiceRunning(supervisor); err != nil {
			return err
		}
		return nil
	case "restart":
		stopCompanions(manifest.Name, manifest.Companions, os.Stderr)
		stopCtx, cancel := managedServiceStopContext(ctx, manifest)
		err := stopManagedService(stopCtx, manifest, supervisor)
		cancel()
		if err != nil {
			return err
		}
		if err := d.startPrivate(ctx, controller, manifest, supervisor); err != nil {
			return err
		}
		if err := d.bootstrapPrivate(ctx, manifest, supervisor); err != nil {
			return err
		}
		startCompanions(manifest.Name, manifest.Companions, manifest.Orchestration.RecoveryAttempts, os.Stderr)
		if err := waitForManagedServiceHealth(ctx, controller, manifest); err != nil {
			return err
		}
		if err := verifyManagedServiceRunning(supervisor); err != nil {
			return err
		}
		return nil
	case "stop", "uninstall":
		stopCompanions(manifest.Name, manifest.Companions, os.Stderr)
		stopCtx, cancel := managedServiceStopContext(ctx, manifest)
		defer cancel()
		return stopManagedService(stopCtx, manifest, supervisor)
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

func (d managedServiceDriver) startPrivate(ctx context.Context, controller *Controller, manifest ResourceManifest, supervisor *ManagedServiceSupervisor) error {
	// Reconcile ownership before any port preflight. A service already running
	// under this supervisor is an idempotent successful start; probing its port
	// first incorrectly rejects that healthy owned process as a conflict.
	if _, running, err := supervisor.Status(); err != nil {
		return fmt.Errorf("reconcile managed-service ownership before start: %w", err)
	} else if running {
		return nil
	}
	if err := ensureManagedServiceArtifact(ctx, controller, manifest); err != nil {
		return err
	}
	if err := ensureManagedServicePortsAvailable(manifest); err != nil {
		return err
	}
	path, err := managedServiceArtifactPath(controller, manifest)
	if err != nil {
		return err
	}
	return d.startPrivateAt(ctx, controller, manifest, supervisor, path)
}

func (d managedServiceDriver) startPrivateAt(ctx context.Context, controller *Controller, manifest ResourceManifest, supervisor *ManagedServiceSupervisor, path string) error {
	if err := d.verifyArtifactAt(ctx, manifest, path); err != nil {
		return err
	}
	artifact, err := managedServiceArtifactForLaunch(ctx, manifest)
	if err != nil {
		return err
	}
	env := resourceEnvForResource(controller.Root, controller.Home, manifest.Name)
	// Expose only the verified artifact directory to the supervised process.
	// Resources such as Ollama ship a native executable alongside runtime
	// libraries; the manifest may point the service at that sibling directory
	// without discovering arbitrary host binaries.
	env = setEnvValue(env, "VROOLI_MANAGED_SERVICE_ARTIFACT", path)
	artifactRoot := filepath.Dir(path)
	if strings.EqualFold(strings.TrimSpace(artifact.Layout), "dir") {
		artifactRoot = path
	}
	env = setEnvValue(env, "RESOURCE_ARTIFACT_DIR", artifactRoot)
	acquisitionEnv, err := acquisitionTargetRuntimeEnv(ctx, manifest, artifactRoot)
	if err != nil {
		return err
	}
	for key, value := range acquisitionEnv {
		env = setEnvValue(env, key, value)
	}
	for _, port := range manifest.Ports {
		if port.Host > 0 {
			env = setEnvValue(env, managedServicePortEnvName(port.Name), fmt.Sprintf("%d", port.Host))
		}
	}
	for key, value := range manifest.ManagedService.Environment {
		env = setEnvValue(env, key, value)
	}
	env = renderManagedServiceEnvironment(env, manifest.ManagedService.Environment)
	if strings.TrimSpace(manifest.ManagedService.EnvironmentFile) != "" {
		fileEnv, err := readManagedServiceEnvironmentFile(manifest.ManagedService.EnvironmentFile, env)
		if err != nil {
			return err
		}
		for key, value := range fileEnv {
			env = setEnvValue(env, key, value)
		}
	}
	if err := writeManagedServiceConfig(manifest, env); err != nil {
		return err
	}
	credentialFiles, err := materializeManagedServiceCredentialFiles(manifest, env)
	if err != nil {
		return err
	}
	cleanupCredentials := func() {
		for _, path := range credentialFiles {
			_ = os.Remove(path)
		}
	}
	if err := runManagedServiceBootstrap(ctx, manifest, path, env); err != nil {
		cleanupCredentials()
		return err
	}
	arguments := renderManagedServiceValues(manifest.ManagedService.Arguments, env)
	_, err = supervisor.Start(path, artifact, arguments, env, filepath.Dir(path), manifest.ManagedService.ProcessLimits)
	if err != nil {
		cleanupCredentials()
	}
	return err
}

// materializeManagedServiceCredentialFiles is the only managed-service path
// that reads a credential value. It writes an authority field to a short-lived
// runtime file, never to env or argv, and replaces files atomically so a stale
// token cannot be observed halfway through a restart.
func materializeManagedServiceCredentialFiles(manifest ResourceManifest, env []string) ([]string, error) {
	if manifest.ManagedService == nil || len(manifest.ManagedService.CredentialFiles) == 0 {
		return nil, nil
	}
	values := managedServiceEnvValues(env)
	root := strings.TrimSpace(values["RESOURCE_STATE_DIR"])
	if root == "" {
		return nil, fmt.Errorf("managed-service credential files require RESOURCE_STATE_DIR")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, fmt.Errorf("create managed-service credential runtime directory: %w", err)
	}
	authority, err := credentialauthority.DefaultAuthority()
	if err != nil {
		return nil, fmt.Errorf("managed-service credential authority unavailable: %w", err)
	}
	paths := make([]string, 0, len(manifest.ManagedService.CredentialFiles))
	cleanup := func() {
		for _, path := range paths {
			_ = os.Remove(path)
		}
	}
	for _, declaration := range manifest.ManagedService.CredentialFiles {
		identity, err := credentialauthority.ParseIdentity(declaration.LogicalID)
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("managed-service credential file identity: %w", err)
		}
		field := declaration.Field
		if strings.TrimSpace(field) == "" {
			field = "value"
		}
		value, err := authority.Resolve(identity, field)
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("managed-service credential %s:%s unavailable: %w", declaration.LogicalID, field, err)
		}
		relative := filepath.Clean(filepath.FromSlash(strings.TrimSpace(declaration.Path)))
		if relative == "." || relative == ".." || filepath.IsAbs(declaration.Path) || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			cleanup()
			return nil, fmt.Errorf("managed-service credential file path must remain under RESOURCE_STATE_DIR")
		}
		destination := filepath.Join(root, relative)
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			cleanup()
			return nil, fmt.Errorf("create managed-service credential directory: %w", err)
		}
		temporary, err := os.CreateTemp(filepath.Dir(destination), ".credential-*")
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("create managed-service credential file: %w", err)
		}
		temporaryName := temporary.Name()
		ok := false
		defer func() {
			if !ok {
				_ = os.Remove(temporaryName)
			}
		}()
		if err := temporary.Chmod(0o600); err != nil {
			cleanup()
			_ = temporary.Close()
			return nil, fmt.Errorf("restrict managed-service credential file: %w", err)
		}
		_, err = temporary.WriteString(value + "\n")
		if err == nil {
			err = temporary.Sync()
		}
		if closeErr := temporary.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("write managed-service credential file: %w", err)
		}
		if err := os.Rename(temporaryName, destination); err != nil {
			cleanup()
			return nil, fmt.Errorf("install managed-service credential file: %w", err)
		}
		if err := securestore.RestrictCredentialFile(destination); err != nil {
			cleanup()
			return nil, fmt.Errorf("restrict managed-service credential file: %w", err)
		}
		paths = append(paths, destination)
		ok = true
	}
	return paths, nil
}

// readManagedServiceEnvironmentFile reads a resource-owned state file without
// invoking a shell. It is intentionally a small KEY=VALUE format: comments and
// blank lines are ignored, keys must be ordinary environment names, and the
// file is resolved beneath RESOURCE_DATA_DIR.
func readManagedServiceEnvironmentFile(relativePath string, env []string) (map[string]string, error) {
	clean := filepath.Clean(strings.TrimSpace(relativePath))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || filepath.IsAbs(relativePath) {
		return nil, fmt.Errorf("managed-service environment file must remain under RESOURCE_DATA_DIR")
	}
	values := managedServiceEnvValues(env)
	root := strings.TrimSpace(values["RESOURCE_DATA_DIR"])
	if root == "" {
		return nil, fmt.Errorf("managed-service environment file requires RESOURCE_DATA_DIR")
	}
	path := filepath.Join(root, clean)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// The file is an optional resource-owned override. A fresh install may
		// legitimately have no override yet; the manifest environment remains
		// authoritative until an operator writes this file.
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read managed-service environment file: %w", err)
	}
	result := make(map[string]string)
	for lineNumber, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || !validManagedServiceEnvironmentKey(key) || strings.ContainsRune(value, '\x00') {
			return nil, fmt.Errorf("invalid managed-service environment file entry on line %d", lineNumber+1)
		}
		result[key] = value
	}
	return result, nil
}

func validManagedServiceEnvironmentKey(key string) bool {
	for index, r := range key {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9' && index > 0) || (r == '_' && index > 0) {
			continue
		}
		return false
	}
	return key != ""
}

func (d managedServiceDriver) verifyArtifactAt(ctx context.Context, manifest ResourceManifest, path string) error {
	artifact, err := managedServiceArtifactForLaunch(ctx, manifest)
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
	supervisor, _, err := managedServiceSupervisorFor(manifest.Name)
	if err != nil {
		return err
	}
	running := false
	if action == "start" {
		_, running, err = supervisor.Status()
		if err != nil {
			return fmt.Errorf("reconcile managed-discovered ownership before start: %w", err)
		}
	}
	if action == "restart" {
		stopCtx, cancel := managedServiceStopContext(ctx, manifest)
		err = stopManagedService(stopCtx, manifest, supervisor)
		cancel()
		if err != nil {
			return err
		}
		running = false
	}
	if !running {
		if err := d.verifyArtifactAt(ctx, manifest, path); err != nil {
			return managedServiceDriverError(item.Name, action, "Provider", err)
		}
		if err := verifyManagedDiscoveredVersion(ctx, path, manifest.ManagedService.Artifact.Version); err != nil {
			return managedServiceDriverError(item.Name, action, "Provider", err)
		}
		if err := ensureManagedServicePortsAvailable(manifest); err != nil {
			return err
		}
		if err := d.startPrivateAt(ctx, controller, manifest, supervisor, path); err != nil {
			return err
		}
	}
	if err := d.bootstrapPrivate(ctx, manifest, supervisor); err != nil {
		return err
	}
	if err := waitForManagedServiceHealth(ctx, controller, manifest); err != nil {
		return err
	}
	return verifyManagedServiceRunning(supervisor)
}

func stopManagedService(ctx context.Context, manifest ResourceManifest, supervisor *ManagedServiceSupervisor) error {
	defer cleanupManagedServiceCredentialFiles(manifest)
	if manifest.ManagedService == nil || manifest.ManagedService.Shutdown == nil {
		return supervisor.Stop(ctx)
	}
	switch strings.ToLower(strings.TrimSpace(manifest.ManagedService.Shutdown.Signal)) {
	case resourcedeployment.ServiceShutdownTerminate:
		return supervisor.Stop(ctx)
	case resourcedeployment.ServiceShutdownInterrupt:
		return supervisor.StopWithSignal(ctx, os.Interrupt)
	default:
		return fmt.Errorf("managed-service %s has invalid shutdown signal %q", manifest.Name, manifest.ManagedService.Shutdown.Signal)
	}
}

func cleanupManagedServiceCredentialFiles(manifest ResourceManifest) {
	if manifest.ManagedService == nil || len(manifest.ManagedService.CredentialFiles) == 0 {
		return
	}
	paths, err := resourceStoragePaths(manifest.Name)
	if err != nil {
		return
	}
	for _, declaration := range manifest.ManagedService.CredentialFiles {
		relative := filepath.Clean(filepath.FromSlash(strings.TrimSpace(declaration.Path)))
		if relative == "." || relative == ".." || filepath.IsAbs(declaration.Path) || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			continue
		}
		_ = os.Remove(filepath.Join(paths.StateDir, relative))
	}
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

func runManagedServiceBootstrap(ctx context.Context, manifest ResourceManifest, artifactPath string, env []string) error {
	if manifest.ManagedService == nil || manifest.ManagedService.Bootstrap == nil {
		return nil
	}
	artifact, err := managedServiceArtifactForLaunch(ctx, manifest)
	if err != nil {
		return err
	}
	bootstrap := manifest.ManagedService.Bootstrap
	if err := bootstrap.Validate(); err != nil {
		return err
	}
	values := managedServiceEnvValues(env)
	dataRoot := strings.TrimSpace(values["RESOURCE_DATA_DIR"])
	configRoot := strings.TrimSpace(values["RESOURCE_CONFIG_DIR"])
	artifactRoot := filepath.Dir(artifactPath)
	if strings.EqualFold(strings.TrimSpace(artifact.Layout), "dir") {
		artifactRoot = artifactPath
	}
	marker := filepath.Join(dataRoot, filepath.FromSlash(bootstrap.Marker))
	if _, err := os.Stat(marker); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect managed-service bootstrap marker: %w", err)
	}
	bootstrapEnv := append([]string(nil), env...)
	var passwordPath string
	if strings.TrimSpace(bootstrap.PasswordEnv) != "" {
		password := values[strings.TrimSpace(bootstrap.PasswordEnv)]
		if password == "" {
			return fmt.Errorf("managed-service bootstrap requires %s", bootstrap.PasswordEnv)
		}
		if strings.TrimSpace(bootstrap.PasswordFile) == "" || configRoot == "" {
			return fmt.Errorf("managed-service bootstrap password_file and RESOURCE_CONFIG_DIR are required")
		}
		passwordPath = filepath.Join(configRoot, filepath.FromSlash(bootstrap.PasswordFile))
		if err := os.MkdirAll(filepath.Dir(passwordPath), 0o700); err != nil {
			return fmt.Errorf("create managed-service bootstrap secret directory: %w", err)
		}
		if err := os.WriteFile(passwordPath, []byte(password+"\n"), 0o600); err != nil {
			return fmt.Errorf("write managed-service bootstrap secret: %w", err)
		}
		defer os.Remove(passwordPath)
	}
	executable := filepath.Join(artifactRoot, filepath.FromSlash(bootstrap.Executable))
	arguments := renderManagedServiceValues(bootstrap.Arguments, bootstrapEnv)
	command := exec.CommandContext(ctx, executable, arguments...)
	command.Dir = artifactRoot
	command.Env = bootstrapEnv
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("run managed-service bootstrap for %s: %w: %s", manifest.Name, err, strings.TrimSpace(string(output)))
	}
	if _, err := os.Stat(marker); err != nil {
		return fmt.Errorf("managed-service bootstrap for %s completed without marker %s: %w", manifest.Name, bootstrap.Marker, err)
	}
	return nil
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
	seconds := managedServiceStartTimeoutSeconds(controller, manifest)
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

// managedServiceStartTimeoutSeconds returns the authored lifecycle timeout,
// raised only when the manifest declares a driver-supported observed budget.
// The Qdrant collection directory is available before process start, so the
// managed-service driver can account for restore/readiness work without
// turning a fixed timeout into an unbounded wait. If the data root or its
// collection count cannot be read, the declared floor is retained.
func managedServiceStartTimeoutSeconds(controller *Controller, manifest ResourceManifest) int {
	seconds := manifest.Lifecycle.StartTimeoutSeconds
	budget := manifest.Orchestration.StartupBudget
	if controller == nil || budget == nil || budget.Kind != "qdrant_collection_count" {
		return seconds
	}
	env := resourceEnvForResource(controller.Root, controller.Home, manifest.Name)
	dataRoot := ""
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if ok && key == "RESOURCE_DATA_DIR" {
			dataRoot = value
			break
		}
	}
	if strings.TrimSpace(dataRoot) == "" {
		return seconds
	}
	entries, err := os.ReadDir(filepath.Join(dataRoot, "collections"))
	if err != nil {
		return seconds
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			count++
		}
	}
	derived := budget.BaseSeconds + count*budget.PerUnitSeconds
	if budget.MaxSeconds > 0 && derived > budget.MaxSeconds {
		derived = budget.MaxSeconds
	}
	if derived > seconds {
		return derived
	}
	return seconds
}

func managedServiceDriverError(resource, operation, category string, err error) *Error {
	return &Error{Code: ErrorCodeCommandUnavailable, Resource: resource, Operation: operation, Category: category, Err: err}
}
