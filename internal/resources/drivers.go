package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	resourceenv "github.com/vrooli/vrooli/internal/resources/env"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	runtimeenv "github.com/vrooli/vrooli/internal/resources/runtime/env"
	runtimelogs "github.com/vrooli/vrooli/internal/resources/runtime/logs"
	runtimestorage "github.com/vrooli/vrooli/internal/resources/runtime/storage"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/shell"
)

const StatusCodeUnsupportedPlatform = "unsupported_platform"

type resourceDriver interface {
	Name() string
	Status(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, fast bool) (Status, error)
	Run(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, action string, args []string, stdout, stderr io.Writer) error
}

var (
	lookPathCommandFn       = exec.LookPath
	runSourceBuildCommandFn = func(cmd *exec.Cmd) error { return cmd.Run() }
)

func driverForManifest(manifest ResourceManifest) (resourceDriver, error) {
	switch manifest.Driver {
	case "docker-service":
		return dockerServiceDriver{}, nil
	case "compose-service":
		return composeServiceDriver{}, nil
	case "external-cli":
		return externalCLIDriver{}, nil
	case "native-cli":
		return nativeCLIDriver{}, nil
	case "cloud-api":
		return cloudAPIDriver{}, nil
	default:
		return nil, fmt.Errorf("driver %q is not yet supported by the native resource control plane", manifest.Driver)
	}
}

func ensureSupportedPlatform(manifest ResourceManifest) error {
	support := manifest.Platforms.SupportForCurrentPlatform()
	if support == "" {
		return nil
	}
	if support == "unsupported" {
		return fmt.Errorf("resource %q is unsupported on %s", manifest.Name, manifestpkg.CurrentPlatform())
	}
	return nil
}

type dockerServiceDriver struct{}

func (dockerServiceDriver) Name() string { return "docker-service" }

type composeServiceDriver struct{}

func (composeServiceDriver) Name() string { return "compose-service" }

func (composeServiceDriver) Status(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, fast bool) (Status, error) {
	status := Status{
		Resource:   item,
		StatusCode: StatusCodeOK,
		Message:    "stopped",
	}

	if err := ensureSupportedPlatform(manifest); err != nil {
		status.StatusCode = StatusCodeUnsupportedPlatform
		status.Message = err.Error()
		status.ProbeError = err.Error()
		return status, nil
	}
	if _, err := lookPathCommandFn("docker"); err != nil {
		status.StatusCode = StatusCodeUnavailable
		status.Message = "docker is unavailable"
		status.ProbeError = err.Error()
		return status, nil
	}

	services, err := inspectComposeServices(ctx, controller, manifest)
	if err != nil {
		status.StatusCode = StatusCodeCommandError
		status.Message = "docker compose ps failed"
		status.ProbeError = err.Error()
		return status, nil
	}
	if len(services) == 0 {
		if state, exists, err := inspectComposeFallbackContainer(ctx, controller, manifest); err != nil {
			status.StatusCode = StatusCodeCommandError
			status.Message = "docker container inspect failed"
			status.ProbeError = err.Error()
			return status, nil
		} else if exists {
			service := composeServiceState{Service: manifest.Name, State: "exited"}
			if state.Running {
				service.State = "running"
			}
			services = []composeServiceState{service}
		} else {
			status.Message = "not installed"
			return status, nil
		}
	}

	status.Installed = true
	for _, service := range services {
		if strings.EqualFold(strings.TrimSpace(service.State), "running") {
			status.Running = true
			break
		}
	}
	if !status.Running {
		healthy := false
		status.Healthy = &healthy
		status.Health = "stopped"
		status.Message = "stopped"
		return status, nil
	}

	healthy := true
	status.Healthy = &healthy
	status.Health = "running"
	status.Message = "running"
	if len(manifest.HealthChecks) > 0 {
		health, err := controller.runResourceHealthChecks(ctx, manifest)
		if err != nil {
			status.StatusCode = StatusCodeCommandError
			status.Message = "health checks failed"
			status.ProbeError = err.Error()
			return status, nil
		}
		healthy = health.Healthy
		status.Healthy = &healthy
		if strings.TrimSpace(health.Message) != "" {
			status.Message = health.Message
		}
	}
	if companions, err := companionStatuses(manifest.Name, manifest.Companions); err != nil {
		status.StatusCode = StatusCodeCommandError
		status.Message = "companion status failed"
		status.ProbeError = err.Error()
		return status, nil
	} else if down := downCompanions(companions); len(down) > 0 {
		status.Raw = statusRawWithCompanions(status.Raw, companions)
		healthy = false
		status.Healthy = &healthy
		status.Health = "unhealthy"
		status.Message = companionDownMessage(manifest.Name, down)
		return status, nil
	} else if len(companions) > 0 {
		status.Raw = statusRawWithCompanions(status.Raw, companions)
	}
	if healthy {
		status.Health = "healthy"
		status.Message = "healthy"
	} else {
		status.Health = "unhealthy"
		status.Message = "unhealthy"
	}
	return status, nil
}

func statusRawWithCompanions(raw json.RawMessage, companions []CompanionStatus) json.RawMessage {
	payload := make(map[string]any)
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &payload)
	}
	payload["companions"] = companions
	out, err := json.Marshal(payload)
	if err != nil {
		return raw
	}
	return out
}

func (d composeServiceDriver) Run(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, action string, args []string, stdout, stderr io.Writer) error {
	if err := ensureSupportedPlatform(manifest); err != nil {
		return &Error{
			Code:      ErrorCodeCommandUnavailable,
			Resource:  item.Name,
			Operation: action,
			Category:  "Platform",
			Err:       err,
		}
	}

	switch action {
	case "status":
		status, err := d.Status(ctx, controller, item, manifest, !containsString(args, "--no-fast"))
		if err != nil {
			return err
		}
		if containsString(args, "--format") && nextArgValue(args, "--format") == "json" {
			companions, _ := companionStatuses(manifest.Name, manifest.Companions)
			return json.NewEncoder(stdout).Encode(map[string]any{
				"installed":  status.Installed,
				"running":    status.Running,
				"healthy":    status.Healthy,
				"health":     status.Health,
				"message":    status.Message,
				"companions": companions,
			})
		}
		_, err = fmt.Fprintf(stdout, "%s: %s\n", item.Name, status.Message)
		return err
	case "install":
		if err := composeCommand(ctx, controller, manifest, io.Discard, stderr, "pull"); err == nil {
			return nil
		}
		return composeCommand(ctx, controller, manifest, io.Discard, stderr, "build")
	case "start":
		if running, err := composeFallbackContainerHealthy(ctx, controller, manifest); err != nil {
			return err
		} else if running {
			return nil
		}
		if err := composeCommand(ctx, controller, manifest, io.Discard, stderr, "up", "-d"); err != nil {
			return err
		}
		startCompanions(manifest.Name, manifest.Companions, manifest.Orchestration.RecoveryAttempts, stderr)
		return nil
	case "restart":
		if err := composeCommand(ctx, controller, manifest, io.Discard, stderr, "up", "-d", "--force-recreate"); err != nil {
			return err
		}
		startCompanions(manifest.Name, manifest.Companions, manifest.Orchestration.RecoveryAttempts, stderr)
		return nil
	case "stop":
		stopCompanions(manifest.Name, manifest.Companions, stderr)
		return composeCommand(ctx, controller, manifest, io.Discard, io.Discard, "stop")
	case "uninstall":
		stopCompanions(manifest.Name, manifest.Companions, stderr)
		return composeCommand(ctx, controller, manifest, io.Discard, io.Discard, "down", "-v", "--remove-orphans")
	case "logs":
		return composeCommand(ctx, controller, manifest, stdout, stderr, "logs")
	default:
		return &Error{
			Code:      ErrorCodeCommandUnavailable,
			Resource:  item.Name,
			Operation: action,
			Category:  "Driver",
			Err:       fmt.Errorf("action %q is not supported by driver %q", action, d.Name()),
		}
	}
}

func (dockerServiceDriver) Status(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, fast bool) (Status, error) {
	status := Status{
		Resource:   item,
		StatusCode: StatusCodeOK,
		Message:    "stopped",
	}

	if err := ensureSupportedPlatform(manifest); err != nil {
		status.StatusCode = StatusCodeUnsupportedPlatform
		status.Message = err.Error()
		status.ProbeError = err.Error()
		return status, nil
	}
	if _, err := lookPathCommandFn("docker"); err != nil {
		status.StatusCode = StatusCodeUnavailable
		status.Message = "docker is unavailable"
		status.ProbeError = err.Error()
		return status, nil
	}
	status.Installed = true

	state, exists, err := inspectDockerContainer(ctx, controller, manifest)
	if err != nil {
		status.StatusCode = StatusCodeCommandError
		status.Message = "docker inspect failed"
		status.ProbeError = err.Error()
		return status, nil
	}
	if !exists {
		if external, err := probeExternalDockerService(ctx, controller, manifest); err == nil && external {
			status.Running = true
			healthy := true
			status.Healthy = &healthy
			status.Health = "healthy"
			status.Message = "healthy (external)"
			return status, nil
		}
		status.Message = "not installed"
		return status, nil
	}

	status.Running = state.Running
	if state.Running {
		healthy := true
		status.Health = "running"
		if len(manifest.HealthChecks) > 0 {
			health, err := controller.runResourceHealthChecks(ctx, manifest)
			if err != nil {
				status.StatusCode = StatusCodeCommandError
				status.Message = "health checks failed"
				status.ProbeError = err.Error()
				return status, nil
			}
			healthy = health.Healthy
			if health.Message != "" {
				status.Message = health.Message
			}
		} else {
			status.Message = "running"
		}
		status.Healthy = &healthy
		if healthy {
			status.Health = "healthy"
			status.Message = "healthy"
		} else {
			status.Health = "unhealthy"
			status.Message = "unhealthy"
		}
		return status, nil
	}

	if external, err := probeExternalDockerService(ctx, controller, manifest); err == nil && external {
		status.Running = true
		healthy := true
		status.Healthy = &healthy
		status.Health = "healthy"
		status.Message = "healthy (external)"
		return status, nil
	}

	healthy := false
	status.Healthy = &healthy
	status.Health = "stopped"
	status.Message = "stopped"
	return status, nil
}

func (d dockerServiceDriver) Run(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, action string, args []string, stdout, stderr io.Writer) error {
	if err := ensureSupportedPlatform(manifest); err != nil {
		return &Error{
			Code:      ErrorCodeCommandUnavailable,
			Resource:  item.Name,
			Operation: action,
			Category:  "Platform",
			Err:       err,
		}
	}

	switch action {
	case "status":
		status, err := d.Status(ctx, controller, item, manifest, !containsString(args, "--no-fast"))
		if err != nil {
			return err
		}
		if containsString(args, "--format") && nextArgValue(args, "--format") == "json" {
			return json.NewEncoder(stdout).Encode(map[string]any{
				"installed": status.Installed,
				"running":   status.Running,
				"healthy":   status.Healthy,
				"health":    status.Health,
				"message":   status.Message,
			})
		}
		_, err = fmt.Fprintf(stdout, "%s: %s\n", item.Name, status.Message)
		return err
	case "install":
		return ensureDockerImage(ctx, controller, manifest)
	case "start":
		return startDockerService(ctx, controller, manifest, false)
	case "restart":
		return startDockerService(ctx, controller, manifest, true)
	case "stop":
		return stopDockerService(ctx, controller, manifest)
	case "uninstall":
		return uninstallDockerService(ctx, controller, manifest)
	case "logs":
		return dockerCommand(ctx, controller, stdout, stderr, "logs", dockerContainerName(manifest))
	default:
		return &Error{
			Code:      ErrorCodeCommandUnavailable,
			Resource:  item.Name,
			Operation: action,
			Category:  "Driver",
			Err:       fmt.Errorf("action %q is not supported by driver %q", action, d.Name()),
		}
	}
}

type dockerState struct {
	Running bool `json:"Running"`
}

type dockerMount struct {
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
}

type composeServiceState struct {
	Service string `json:"Service"`
	State   string `json:"State"`
	Health  string `json:"Health"`
}

func dockerContainerName(manifest ResourceManifest) string {
	if strings.TrimSpace(manifest.Runtime.ContainerName) != "" {
		return strings.TrimSpace(manifest.Runtime.ContainerName)
	}
	return "vrooli-" + manifest.Name
}

func composeProjectName(manifest ResourceManifest) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "-", "_", "-")
	return "vrooli-" + replacer.Replace(strings.TrimSpace(manifest.Name))
}

func composeFilePath(controller *Controller, manifest ResourceManifest) string {
	composeFile := strings.TrimSpace(manifest.ComposeFile)
	if composeFile == "" {
		return filepath.Join(controller.Root, "resources", manifest.Name, "compose.yaml")
	}
	if filepath.IsAbs(composeFile) {
		return composeFile
	}
	return filepath.Join(controller.Root, "resources", manifest.Name, filepath.FromSlash(composeFile))
}

func inspectComposeServices(ctx context.Context, controller *Controller, manifest ResourceManifest) ([]composeServiceState, error) {
	output, err := composeOutput(ctx, controller, manifest, "ps", "-a", "--format", "json")
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(normalizeComposePSOutput(output))
	if trimmed == "" {
		return nil, nil
	}

	var services []composeServiceState
	if strings.HasPrefix(trimmed, "[") {
		if err := json.Unmarshal(output, &services); err != nil {
			return nil, fmt.Errorf("parse docker compose ps output: %w", err)
		}
		return services, nil
	}

	lines := strings.Split(trimmed, "\n")
	services = make([]composeServiceState, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var service composeServiceState
		if err := json.Unmarshal([]byte(line), &service); err != nil {
			return nil, fmt.Errorf("parse docker compose ps line: %w", err)
		}
		services = append(services, service)
	}
	return services, nil
}

func inspectComposeFallbackContainer(ctx context.Context, controller *Controller, manifest ResourceManifest) (dockerState, bool, error) {
	for _, name := range composeFallbackContainerNames(manifest) {
		output, err := dockerOutput(ctx, controller, "container", "inspect", name, "--format", "{{json .State}}")
		if err != nil {
			if strings.Contains(err.Error(), "No such object") || strings.Contains(err.Error(), "No such container") {
				continue
			}
			return dockerState{}, false, err
		}
		var state dockerState
		if err := json.Unmarshal(output, &state); err != nil {
			return dockerState{}, false, fmt.Errorf("parse docker inspect state for %s: %w", name, err)
		}
		return state, true, nil
	}
	return dockerState{}, false, nil
}

func composeFallbackContainerNames(manifest ResourceManifest) []string {
	seen := map[string]struct{}{}
	names := make([]string, 0, 3)
	for _, name := range []string{
		strings.TrimSpace(manifest.Runtime.ContainerName),
		strings.TrimSpace(manifest.Name),
		dockerContainerName(manifest),
	} {
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	return names
}

func composeFallbackContainerHealthy(ctx context.Context, controller *Controller, manifest ResourceManifest) (bool, error) {
	state, exists, err := inspectComposeFallbackContainer(ctx, controller, manifest)
	if err != nil || !exists || !state.Running {
		return false, err
	}
	if len(manifest.HealthChecks) == 0 {
		return true, nil
	}
	health, err := controller.runResourceHealthChecks(ctx, manifest)
	if err != nil {
		return false, err
	}
	return health.Healthy, nil
}

func normalizeComposePSOutput(output []byte) string {
	lines := strings.Split(string(output), "\n")
	jsonLines := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
			jsonLines = append(jsonLines, trimmed)
		}
	}
	return strings.Join(jsonLines, "\n")
}

func inspectDockerContainer(ctx context.Context, controller *Controller, manifest ResourceManifest) (dockerState, bool, error) {
	output, err := dockerOutput(ctx, controller, "container", "inspect", dockerContainerName(manifest), "--format", "{{json .State}}")
	if err != nil {
		if strings.Contains(err.Error(), "No such object") || strings.Contains(err.Error(), "No such container") {
			return dockerState{}, false, nil
		}
		return dockerState{}, false, err
	}
	var state dockerState
	if err := json.Unmarshal(output, &state); err != nil {
		return dockerState{}, false, fmt.Errorf("parse docker inspect state: %w", err)
	}
	return state, true, nil
}

func ensureDockerImage(ctx context.Context, controller *Controller, manifest ResourceManifest) error {
	if strings.TrimSpace(manifest.Runtime.Image) == "" {
		return fmt.Errorf("runtime.image is required")
	}
	if _, err := dockerOutput(ctx, controller, "image", "inspect", manifest.Runtime.Image); err == nil {
		return nil
	}
	return dockerCommand(ctx, controller, io.Discard, io.Discard, "pull", manifest.Runtime.Image)
}

func startDockerService(ctx context.Context, controller *Controller, manifest ResourceManifest, restart bool) error {
	state, exists, err := inspectDockerContainer(ctx, controller, manifest)
	if err != nil {
		return err
	}
	// An externally managed healthy service wins over a stale stopped container.
	// Check it before inspecting/removing that container: this keeps the resource
	// lifecycle non-invasive when the service was started outside Docker's local
	// state (and avoids requiring container-inspect support in that path).
	if exists && !state.Running {
		if external, err := probeExternalDockerService(ctx, controller, manifest); err == nil && external {
			return nil
		}
		currentRuntime, runtimeErr := inspectDockerRuntime(ctx, controller, manifest)
		if runtimeErr != nil {
			return runtimeErr
		}
		if currentRuntime != dockerRuntimeForManifest(ctx, manifest) {
			if err := dockerCommand(ctx, controller, io.Discard, io.Discard, "rm", dockerContainerName(manifest)); err != nil {
				return fmt.Errorf("remove stopped resource container with stale runtime: %w", err)
			}
			exists = false
		}
	}
	if !exists {
		if external, err := probeExternalDockerService(ctx, controller, manifest); err == nil && external {
			return nil
		}
	}
	if err := ensureDockerImage(ctx, controller, manifest); err != nil {
		return err
	}
	name := dockerContainerName(manifest)
	if exists {
		if err := validateExistingDockerMounts(ctx, controller, manifest); err != nil {
			return err
		}
		if !state.Running {
			if external, err := probeExternalDockerService(ctx, controller, manifest); err == nil && external {
				return nil
			}
		}
		if restart {
			return dockerCommand(ctx, controller, io.Discard, io.Discard, "restart", name)
		}
		if state.Running {
			return nil
		}
		// About to `docker start` a stopped container and re-bind its host ports.
		// The external probe above found no healthy service, so a still-occupied
		// host port is a non-container conflict — fail fast with remediation.
		if err := preflightPortConflict(manifest); err != nil {
			return err
		}
		return dockerCommand(ctx, controller, io.Discard, io.Discard, "start", name)
	}

	// Preflight: we are about to create a brand-new container and bind its host
	// ports. The container does not exist and no healthy external service answered
	// the health probe above, so a still-occupied host port means a non-container
	// process owns it (classically a legacy host-systemd Ollama from before the
	// Docker migration). `docker run -p host:container` would otherwise fail with a
	// cryptic bind error and crash-loop, so we surface an actionable message first.
	if err := preflightPortConflict(manifest); err != nil {
		return err
	}

	args, err := buildDockerRunArgs(ctx, controller, manifest, name)
	if err != nil {
		return err
	}
	return dockerCommand(ctx, controller, io.Discard, io.Discard, args...)
}

// preflightPortConflict reports an actionable error when any of the manifest's
// host ports is already bound by a process other than our own container. It is
// only meaningful on the create-new-container path: the caller has already
// established that our container does not exist and that no healthy external
// service is serving the resource, so a bound port is a genuine host conflict
// rather than the resource already running. This is graceful-degradation design —
// a clear failure with remediation instead of a crash-loop.
func preflightPortConflict(manifest ResourceManifest) error {
	for _, port := range manifest.Ports {
		hostPort := port.Host
		if hostPort <= 0 {
			hostPort = port.Container
		}
		if hostPort <= 0 {
			continue
		}
		hostIP := strings.TrimSpace(port.HostIP)
		protocol := dockerPortProtocol(port)
		if hostPortInUse(hostIP, hostPort, protocol) {
			hostLabel := net.JoinHostPort(hostIP, strconv.Itoa(hostPort))
			if hostIP == "" {
				hostLabel = ":" + strconv.Itoa(hostPort)
			}
			return fmt.Errorf(
				"resource %q cannot start: host port %s/%s is already in use by a non-container process. "+
					"This resource runs %s as a Docker container, but a host service (e.g. a legacy host-systemd %s) is holding the port. "+
					"Stop and remove the host process — e.g. `sudo systemctl disable --now %s` or terminate whatever is listening on :%d — then retry.",
				manifest.Name, hostLabel, protocol, manifest.Name, manifest.Name, manifest.Name, hostPort)
		}
	}
	return nil
}

// hostPortInUse returns true when the given port cannot be bound on the host,
// which indicates another process is already listening on it. We probe by binding
// and immediately releasing rather than shelling out to lsof/ss so the check stays
// dependency-free and cross-platform.
func hostPortInUse(hostIP string, port int, protocol string) bool {
	address := net.JoinHostPort(strings.TrimSpace(hostIP), strconv.Itoa(port))
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "udp":
		packetConn, err := net.ListenPacket("udp", address)
		if err != nil {
			return isAddressInUse(err)
		}
		_ = packetConn.Close()
		return false
	default:
		listener, err := net.Listen("tcp", address)
		if err != nil {
			return isAddressInUse(err)
		}
		_ = listener.Close()
		return false
	}
}

func isAddressInUse(err error) bool {
	return errors.Is(err, syscall.EADDRINUSE)
}

// buildDockerRunArgs assembles the `docker run` argument list for a docker-service
// resource and performs the host-side filesystem preparation (mkdir/touch) for any
// declared bind-mount volume sources. A service must explicitly declare and pass a
// GPU probe before it receives GPU devices; every other service pins runc so a
// host-wide Docker default runtime cannot inject an unavailable NVIDIA hook.
func buildDockerRunArgs(ctx context.Context, controller *Controller, manifest ResourceManifest, name string) ([]string, error) {
	args := []string{"run", "-d", "--name", name}
	useGPU := dockerUsesGPU(ctx, manifest)
	if useGPU {
		args = append(args, "--gpus", "all")
	} else {
		args = append(args, "--runtime", "runc")
	}
	for _, port := range manifest.Ports {
		if port.Container <= 0 {
			continue
		}
		hostPort := port.Host
		if hostPort <= 0 {
			hostPort = port.Container
		}
		args = append(args, "-p", dockerPublishPort(port, hostPort))
	}
	for key, value := range manifest.Runtime.Env {
		args = append(args, "-e", key+"="+expandResourceRuntimeValue(controller, manifest, value))
	}
	if useGPU {
		for key, value := range manifest.GPU.EnvOverrides {
			args = append(args, "-e", key+"="+expandResourceRuntimeValue(controller, manifest, value))
		}
	}
	if memLimit := strings.TrimSpace(manifest.Runtime.MemoryLimit); memLimit != "" {
		args = append(args, "--memory", memLimit)
	}
	resourceDir := filepath.Join(controller.Root, "resources", manifest.Name)
	for _, volume := range manifest.Runtime.Volumes {
		source := resolveDockerVolumeSource(controller, manifest, resourceDir, volume)
		if volumeSourceLooksLikeFile(volume) {
			parent := filepath.Dir(source)
			if err := os.MkdirAll(parent, 0o755); err != nil {
				return nil, fmt.Errorf("create volume source parent %s: %w", parent, err)
			}
			if _, err := os.Stat(source); os.IsNotExist(err) {
				file, createErr := os.Create(source)
				if createErr != nil {
					return nil, fmt.Errorf("create volume source file %s: %w", source, createErr)
				}
				_ = file.Close()
			}
		} else {
			if err := os.MkdirAll(source, 0o755); err != nil {
				return nil, fmt.Errorf("create volume source %s: %w", source, err)
			}
		}
		args = append(args, "-v", source+":"+volume.Target)
	}
	if workDir := strings.TrimSpace(expandResourceRuntimeValue(controller, manifest, manifest.Runtime.WorkingDir)); workDir != "" {
		args = append(args, "-w", workDir)
	}
	args = append(args, manifest.Runtime.Image)
	for _, part := range manifest.Runtime.Command {
		args = append(args, expandResourceRuntimeValue(controller, manifest, part))
	}
	return args, nil
}

func dockerUsesGPU(ctx context.Context, manifest ResourceManifest) bool {
	return manifest.GPU != nil && shouldUseGPU(ctx, manifest.GPU.Probe)
}

func dockerRuntimeForManifest(ctx context.Context, manifest ResourceManifest) string {
	if dockerUsesGPU(ctx, manifest) {
		return ""
	}
	return "runc"
}

func inspectDockerRuntime(ctx context.Context, controller *Controller, manifest ResourceManifest) (string, error) {
	output, err := dockerOutput(ctx, controller, "container", "inspect", dockerContainerName(manifest), "--format", "{{.HostConfig.Runtime}}")
	if err != nil {
		return "", fmt.Errorf("inspect resource container runtime: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func resolveDockerVolumeSource(controller *Controller, manifest ResourceManifest, resourceDir string, volume ResourceVolume) string {
	source := expandResourceRuntimeValue(controller, manifest, volume.Source)
	if !filepath.IsAbs(source) {
		source = filepath.Join(resourceDir, filepath.FromSlash(source))
	}
	return source
}

func validateExistingDockerMounts(ctx context.Context, controller *Controller, manifest ResourceManifest) error {
	if len(manifest.Runtime.Volumes) == 0 {
		return nil
	}
	output, err := dockerOutput(ctx, controller, "container", "inspect", dockerContainerName(manifest), "--format", "{{json .Mounts}}")
	if err != nil {
		return err
	}
	var mounts []dockerMount
	if err := json.Unmarshal(output, &mounts); err != nil {
		return fmt.Errorf("parse docker inspect mounts: %w", err)
	}
	currentByTarget := map[string]string{}
	for _, mount := range mounts {
		target := filepath.Clean(filepath.FromSlash(strings.TrimSpace(mount.Destination)))
		if target == "." {
			continue
		}
		currentByTarget[target] = filepath.Clean(strings.TrimSpace(mount.Source))
	}
	resourceDir := filepath.Join(controller.Root, "resources", manifest.Name)
	for _, volume := range manifest.Runtime.Volumes {
		target := filepath.Clean(filepath.FromSlash(strings.TrimSpace(volume.Target)))
		if target == "." {
			continue
		}
		want := filepath.Clean(resolveDockerVolumeSource(controller, manifest, resourceDir, volume))
		got, ok := currentByTarget[target]
		if !ok || got != want {
			if !ok {
				got = "<missing>"
			}
			return fmt.Errorf(
				"resource %q container %q has stale docker mount for %s: got %s, want %s. "+
					"Back up any live data, remove the stale container with `docker rm -f %s`, then start the resource so it is recreated from the current manifest.",
				manifest.Name, dockerContainerName(manifest), target, got, want, dockerContainerName(manifest))
		}
	}
	return nil
}

func dockerPublishPort(port ResourcePort, hostPort int) string {
	container := strconv.Itoa(port.Container)
	if protocol := dockerPortProtocol(port); protocol != "" && protocol != "tcp" {
		container += "/" + protocol
	}
	published := strconv.Itoa(hostPort) + ":" + container
	if hostIP := strings.TrimSpace(port.HostIP); hostIP != "" {
		published = hostIP + ":" + published
	}
	return published
}

func dockerPortProtocol(port ResourcePort) string {
	protocol := strings.ToLower(strings.TrimSpace(port.Protocol))
	if protocol == "" {
		return "tcp"
	}
	return protocol
}

func probeExternalDockerService(ctx context.Context, controller *Controller, manifest ResourceManifest) (bool, error) {
	if len(manifest.HealthChecks) == 0 {
		return false, nil
	}
	health, err := controller.runResourceHealthChecks(ctx, manifest)
	if err != nil {
		return false, err
	}
	return health.Healthy, nil
}

func stopDockerService(ctx context.Context, controller *Controller, manifest ResourceManifest) error {
	_, exists, err := inspectDockerContainer(ctx, controller, manifest)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	return dockerCommand(ctx, controller, io.Discard, io.Discard, "stop", dockerContainerName(manifest))
}

func uninstallDockerService(ctx context.Context, controller *Controller, manifest ResourceManifest) error {
	_, exists, err := inspectDockerContainer(ctx, controller, manifest)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	return dockerCommand(ctx, controller, io.Discard, io.Discard, "rm", "-f", dockerContainerName(manifest))
}

func dockerOutput(ctx context.Context, controller *Controller, args ...string) ([]byte, error) {
	cmd := shell.Command(shell.Spec{
		Name: "docker",
		Args: args,
		Dir:  controller.Root,
		Env:  resourceEnv(controller.Root, controller.Home),
	})
	result := runCommandResource(ctx, cmd)
	if result.err != nil {
		return nil, fmt.Errorf("%w: %s", result.err, strings.TrimSpace(string(result.output)))
	}
	return result.output, nil
}

func dockerCommand(ctx context.Context, controller *Controller, stdout, stderr io.Writer, args ...string) error {
	cmd := shell.Command(shell.Spec{
		Name:   "docker",
		Args:   args,
		Dir:    controller.Root,
		Env:    resourceEnv(controller.Root, controller.Home),
		Stdout: stdout,
		Stderr: stderr,
	})
	runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if err := cmd.Start(); err != nil {
		return err
	}
	waitErr := cmd.Wait()
	if runCtx.Err() != nil {
		return runCtx.Err()
	}
	return waitErr
}

// composeInvocationArgs builds the docker-compose argument list and process
// environment for a resource, applying any GPU overlay and env overrides when
// the manifest declares a gpu block and the probe (or override) says to use
// it. Keeping this in one place ensures composeOutput and composeCommand stay
// in sync.
func composeInvocationArgs(ctx context.Context, controller *Controller, manifest ResourceManifest) ([]string, []string) {
	cmdArgs := []string{"compose", "-f", composeFilePath(controller, manifest), "--project-name", composeProjectName(manifest)}
	env := resourceEnvForResource(controller.Root, controller.Home, manifest.Name)

	if manifest.GPU != nil && shouldUseGPU(ctx, manifest.GPU.Probe) {
		if overlay := strings.TrimSpace(manifest.GPU.ComposeOverlay); overlay != "" {
			cmdArgs = append(cmdArgs, "-f", composeOverlayPath(controller, manifest, overlay))
		}
		for k, v := range manifest.GPU.EnvOverrides {
			env = append(env, k+"="+v)
		}
	}
	if extra := harvestRuntimeEnvCommand(ctx, manifest); len(extra) > 0 {
		env = append(env, extra...)
	}
	return cmdArgs, env
}

// harvestRuntimeEnvCommand runs the manifest's runtime_env_command (if
// declared) and returns KEY=VALUE pairs harvested from stdout. Failure
// is non-fatal: the driver logs a warning equivalent and falls through
// with the static env. Resources MUST design their commands to be
// idempotent and fast (<5s default).
func harvestRuntimeEnvCommand(ctx context.Context, manifest ResourceManifest) []string {
	spec := manifest.RuntimeEnvCommand
	if spec == nil || strings.TrimSpace(spec.Command) == "" {
		return nil
	}
	timeout := time.Duration(spec.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	out, err := exec.CommandContext(cctx, spec.Command, spec.Args...).Output()
	if err != nil {
		return nil
	}
	var pairs []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.Contains(line, "=") {
			continue
		}
		pairs = append(pairs, line)
	}
	return pairs
}

func composeOverlayPath(controller *Controller, manifest ResourceManifest, overlay string) string {
	if filepath.IsAbs(overlay) {
		return overlay
	}
	return filepath.Join(controller.Root, "resources", manifest.Name, filepath.FromSlash(overlay))
}

func composeOutput(ctx context.Context, controller *Controller, manifest ResourceManifest, args ...string) ([]byte, error) {
	cmdArgs, env := composeInvocationArgs(ctx, controller, manifest)
	cmdArgs = append(cmdArgs, args...)
	cmd := shell.Command(shell.Spec{
		Name: "docker",
		Args: cmdArgs,
		Dir:  controller.Root,
		Env:  env,
	})
	result := runCommandResource(ctx, cmd)
	if result.err != nil {
		return nil, fmt.Errorf("%w: %s", result.err, strings.TrimSpace(string(result.output)))
	}
	return result.output, nil
}

func composeCommand(ctx context.Context, controller *Controller, manifest ResourceManifest, stdout, stderr io.Writer, args ...string) error {
	cmdArgs, env := composeInvocationArgs(ctx, controller, manifest)
	cmdArgs = append(cmdArgs, args...)
	cmd := shell.Command(shell.Spec{
		Name:   "docker",
		Args:   cmdArgs,
		Dir:    controller.Root,
		Env:    env,
		Stdout: stdout,
		Stderr: stderr,
	})
	runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if err := cmd.Start(); err != nil {
		return err
	}
	waitErr := cmd.Wait()
	if runCtx.Err() != nil {
		return runCtx.Err()
	}
	return waitErr
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func nextArgValue(args []string, flag string) string {
	for index, arg := range args {
		if arg == flag && index+1 < len(args) {
			return args[index+1]
		}
		if strings.HasPrefix(arg, flag+"=") {
			return strings.TrimPrefix(arg, flag+"=")
		}
	}
	return ""
}

func expandResourceRuntimeValue(controller *Controller, manifest ResourceManifest, value string) string {
	renderer, err := newResourceRuntimeRenderer(controller, manifest.Name)
	if err != nil {
		value = strings.ReplaceAll(value, "${HOME}", controller.Home)
		value = strings.ReplaceAll(value, "$HOME", controller.Home)
		value = strings.ReplaceAll(value, "${ROOT}", controller.Root)
		value = strings.ReplaceAll(value, "$ROOT", controller.Root)
		return value
	}
	return renderer.RenderValue(value)
}

func newResourceRuntimeRenderer(controller *Controller, resourceName string) (*runtimeenv.Renderer, error) {
	resolver, err := runtimestorage.NewResolver(runtimestorage.ResolverConfig{AppID: "vrooli"})
	if err != nil {
		return nil, err
	}
	paths, err := resolver.Resolve(runtimestorage.Options{ResourceID: resourceName})
	if err != nil {
		return nil, err
	}
	return runtimeenv.NewRenderer(controller.Root, controller.Home, resourceName, paths), nil
}

func managedLogCandidates(controller *Controller, manifest ResourceManifest) []string {
	resolver, err := runtimestorage.NewResolver(runtimestorage.ResolverConfig{AppID: "vrooli"})
	if err != nil {
		return nil
	}
	paths, err := resolver.Resolve(runtimestorage.Options{ResourceID: manifest.Name})
	if err != nil {
		return nil
	}
	return runtimelogs.CandidatePaths(manifestpkg.ResourceManifest(manifest), paths)
}

func volumeSourceLooksLikeFile(volume ResourceVolume) bool {
	sourceBase := filepath.Base(filepath.FromSlash(volume.Source))
	targetBase := filepath.Base(filepath.FromSlash(volume.Target))
	if strings.HasPrefix(sourceBase, ".") || strings.HasPrefix(targetBase, ".") {
		return true
	}
	return strings.Contains(sourceBase, ".") || strings.Contains(targetBase, ".")
}

type externalCLIDriver struct{}

func (externalCLIDriver) Name() string { return "external-cli" }

type nativeCLIDriver struct{}

func (nativeCLIDriver) Name() string { return "native-cli" }

func (d externalCLIDriver) Status(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, fast bool) (Status, error) {
	status := Status{
		Resource:   item,
		StatusCode: StatusCodeOK,
		Message:    "not installed",
	}
	if err := ensureSupportedPlatform(manifest); err != nil {
		status.StatusCode = StatusCodeUnsupportedPlatform
		status.Message = err.Error()
		status.ProbeError = err.Error()
		return status, nil
	}

	binary := externalCLIBinary(manifest)
	if _, err := lookPathCommandFn(binary); err != nil {
		status.StatusCode = StatusCodeUnavailable
		status.Message = fmt.Sprintf("%s is unavailable", binary)
		status.ProbeError = err.Error()
		return status, nil
	}
	if err := probeExternalCLICommand(ctx, controller, manifest); err != nil {
		status.StatusCode = StatusCodeUnavailable
		status.Message = fmt.Sprintf("%s is unavailable", binary)
		status.ProbeError = err.Error()
		healthy := false
		status.Healthy = &healthy
		status.Health = "unhealthy"
		return status, nil
	}

	status.Installed = true
	status.Running = true
	healthy := true
	status.Healthy = &healthy
	status.Health = "healthy"
	status.Message = "available"

	if fast {
		return status, nil
	}
	if len(manifest.HealthChecks) > 0 {
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
			if strings.TrimSpace(health.Message) != "" {
				status.Message = health.Message
			}
			return status, nil
		}
		status.Health = "unhealthy"
		if strings.TrimSpace(health.Message) != "" {
			status.Message = health.Message
		} else {
			status.Message = "unhealthy"
		}
	}

	return status, nil
}

func (d externalCLIDriver) Run(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, action string, args []string, stdout, stderr io.Writer) error {
	if err := ensureSupportedPlatform(manifest); err != nil {
		return &Error{
			Code:      ErrorCodeCommandUnavailable,
			Resource:  item.Name,
			Operation: action,
			Category:  "Platform",
			Err:       err,
		}
	}

	switch action {
	case "status":
		status, err := d.Status(ctx, controller, item, manifest, !containsString(args, "--no-fast"))
		if err != nil {
			return err
		}
		if containsString(args, "--format") && nextArgValue(args, "--format") == "json" {
			return json.NewEncoder(stdout).Encode(map[string]any{
				"installed": status.Installed,
				"running":   status.Running,
				"healthy":   status.Healthy,
				"health":    status.Health,
				"message":   status.Message,
			})
		}
		_, err = fmt.Fprintf(stdout, "%s: %s\n", item.Name, status.Message)
		return err
	case "install":
		return runInstallCommand(ctx, controller, manifest)
	case "start", "restart":
		status, err := d.Status(ctx, controller, item, manifest, false)
		if err != nil {
			return err
		}
		if status.Running && status.Healthy != nil && *status.Healthy {
			_, err := fmt.Fprintf(stdout, "%s does not require a managed start step\n", item.Name)
			return err
		}
		return runInstallCommand(ctx, controller, manifest)
	case "stop":
		_, err := fmt.Fprintf(stdout, "%s does not run as a managed background service\n", item.Name)
		return err
	case "logs":
		if manifest.Capabilities.SupportsLogs {
			candidates := managedLogCandidates(controller, manifest)
			if len(candidates) > 0 {
				_, err := fmt.Fprintf(stdout, "%s managed logs may be available under:\n- %s\n", item.Name, strings.Join(candidates, "\n- "))
				return err
			}
		}
		_, err := fmt.Fprintf(stdout, "%s does not expose managed logs through the resource control plane\n", item.Name)
		return err
	default:
		return &Error{
			Code:      ErrorCodeCommandUnavailable,
			Resource:  item.Name,
			Operation: action,
			Category:  "Driver",
			Err:       fmt.Errorf("action %q is not supported by driver %q", action, d.Name()),
		}
	}
}

func (d nativeCLIDriver) Status(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, fast bool) (Status, error) {
	return externalCLIDriver{}.Status(ctx, controller, item, manifest, fast)
}

func (d nativeCLIDriver) Run(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, action string, args []string, stdout, stderr io.Writer) error {
	return externalCLIDriver{}.Run(ctx, controller, item, manifest, action, args, stdout, stderr)
}

func probeExternalCLICommand(ctx context.Context, controller *Controller, manifest ResourceManifest) error {
	binary := strings.TrimSpace(externalCLIBinary(manifest))
	if binary == "" {
		return nil
	}
	args := append([]string(nil), manifest.VersionArgs...)
	if len(args) == 0 {
		return nil
	}

	cmd := shell.Command(shell.Spec{
		Name:  binary,
		Args:  args,
		Dir:   controller.Root,
		Env:   resourceEnvForResource(controller.Root, controller.Home, manifest.Name),
		Stdin: nil,
	})
	result := runCommandResource(ctx, cmd)
	return result.err
}

type cloudAPIDriver struct{}

func (cloudAPIDriver) Name() string { return "cloud-api" }

func (d cloudAPIDriver) Status(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, fast bool) (Status, error) {
	status := Status{
		Resource:   item,
		Installed:  true,
		Running:    true,
		StatusCode: StatusCodeOK,
		Message:    "configured",
	}
	if err := ensureSupportedPlatform(manifest); err != nil {
		status.StatusCode = StatusCodeUnsupportedPlatform
		status.Message = err.Error()
		status.ProbeError = err.Error()
		return status, nil
	}

	missing, err := resourceenv.MissingCredentialKeys(controller.Root, controller.Home, manifest)
	if err != nil {
		status.StatusCode = StatusCodeCommandError
		status.Message = "credential resolution failed"
		status.ProbeError = err.Error()
		return status, nil
	}
	if len(missing) > 0 {
		healthy := false
		status.Healthy = &healthy
		status.Health = "unhealthy"
		status.Message = "missing credentials: " + strings.Join(missing, ", ")
		return status, nil
	}

	healthy := true
	status.Healthy = &healthy
	status.Health = "healthy"

	if fast {
		return status, nil
	}
	if len(manifest.HealthChecks) > 0 {
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
			if strings.TrimSpace(health.Message) != "" {
				status.Message = health.Message
			}
			return status, nil
		}
		status.Health = "unhealthy"
		if strings.TrimSpace(health.Message) != "" {
			status.Message = health.Message
		} else {
			status.Message = "unhealthy"
		}
	}

	return status, nil
}

func (d cloudAPIDriver) Run(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, action string, args []string, stdout, stderr io.Writer) error {
	if err := ensureSupportedPlatform(manifest); err != nil {
		return &Error{
			Code:      ErrorCodeCommandUnavailable,
			Resource:  item.Name,
			Operation: action,
			Category:  "Platform",
			Err:       err,
		}
	}

	switch action {
	case "status":
		status, err := d.Status(ctx, controller, item, manifest, !containsString(args, "--no-fast"))
		if err != nil {
			return err
		}
		if containsString(args, "--format") && nextArgValue(args, "--format") == "json" {
			return json.NewEncoder(stdout).Encode(map[string]any{
				"installed": status.Installed,
				"running":   status.Running,
				"healthy":   status.Healthy,
				"health":    status.Health,
				"message":   status.Message,
			})
		}
		_, err = fmt.Fprintf(stdout, "%s: %s\n", item.Name, status.Message)
		return err
	case "install":
		return runInstallCommand(ctx, controller, manifest)
	case "start", "restart":
		_, err := fmt.Fprintf(stdout, "%s is a hosted API and does not have a local start step\n", item.Name)
		return err
	case "stop":
		_, err := fmt.Fprintf(stdout, "%s is a hosted API and does not have a local stop step\n", item.Name)
		return err
	case "logs":
		if manifest.Capabilities.SupportsLogs {
			candidates := managedLogCandidates(controller, manifest)
			if len(candidates) > 0 {
				_, err := fmt.Fprintf(stdout, "%s managed logs may be available under:\n- %s\n", item.Name, strings.Join(candidates, "\n- "))
				return err
			}
		}
		_, err := fmt.Fprintf(stdout, "%s does not expose managed logs through the resource control plane\n", item.Name)
		return err
	default:
		return &Error{
			Code:      ErrorCodeCommandUnavailable,
			Resource:  item.Name,
			Operation: action,
			Category:  "Driver",
			Err:       fmt.Errorf("action %q is not supported by driver %q", action, d.Name()),
		}
	}
}

func externalCLIBinary(manifest ResourceManifest) string {
	if strings.TrimSpace(manifest.Binary) != "" {
		return strings.TrimSpace(manifest.Binary)
	}
	return manifest.Name
}

func runInstallCommand(ctx context.Context, controller *Controller, manifest ResourceManifest) error {
	if manifest.CLI != nil && manifest.CLI.Enabled {
		if sourceBuild := manifest.CLI.SourceBuild; sourceBuild != nil {
			return runSourceBuild(ctx, controller, manifest, sourceBuild)
		}
	}
	command := manifest.Install.Command
	if len(command) == 0 {
		command = manifest.Install.Platforms[manifestpkg.CurrentPlatform()]
	}
	if len(command) == 0 {
		return nil
	}
	// Capture stderr (bounded tail) so a failing install surfaces its reason —
	// e.g. a coding-agent resource refusing to clobber a root-owned system
	// copy — instead of a bare "exit status N". Successful installs stay quiet.
	stderrTail := newTailBuffer(16 << 10)
	cmd := shell.Command(shell.Spec{
		Name:   command[0],
		Args:   command[1:],
		Dir:    controller.Root,
		Env:    resourceEnvForResource(controller.Root, controller.Home, manifest.Name),
		Stdout: io.Discard,
		Stderr: stderrTail,
	})
	runCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	if err := cmd.Start(); err != nil {
		return err
	}
	waitErr := cmd.Wait()
	if runCtx.Err() != nil {
		return runCtx.Err()
	}
	if waitErr != nil {
		if detail := strings.TrimSpace(stderrTail.String()); detail != "" {
			return fmt.Errorf("%w\n%s", waitErr, detail)
		}
	}
	return waitErr
}

// runSourceBuild is the Go-native source-checkout installation path. It is
// intentionally separate from deployed resource distribution: developers with
// a checkout need Go to build changed source, while desktop bundles consume
// signed prebuilt artifacts and never reach this code.
func runSourceBuild(ctx context.Context, controller *Controller, manifest ResourceManifest, sourceBuild *scenario.CLISourceBuildConfig) error {
	if sourceBuild.Kind != "go_module" {
		return fmt.Errorf("unsupported resource source build kind %q", sourceBuild.Kind)
	}
	resourceRoot := filepath.Join(controller.Root, "resources", manifest.Name)
	moduleDir := filepath.Join(resourceRoot, filepath.FromSlash(manifest.CLI.Adapter.ModuleDir))
	manifestPath := filepath.Join(resourceRoot, "resource.json")
	installerDir := filepath.Join(controller.Root, "packages", "cli-core")
	installDir := filepath.Join(controller.Home, ".vrooli", "bin")
	spec := cliutil.CanonicalResourceGoModuleFreshnessSpec(resourceRoot, moduleDir, manifest.CLI.Command, manifest.CLI.Freshness.Inputs)
	args := cliutil.GoModuleInstallerArgs(moduleDir, manifestPath, manifest.CLI.Command, installDir, spec)
	stderrTail := newTailBuffer(16 << 10)
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = installerDir
	cmd.Env = resourceEnvForResource(controller.Root, controller.Home, manifest.Name)
	cmd.Stdout = io.Discard
	cmd.Stderr = stderrTail
	if err := runSourceBuildCommandFn(cmd); err != nil {
		if detail := strings.TrimSpace(stderrTail.String()); detail != "" {
			return fmt.Errorf("source-build resource CLI: %w\n%s", err, detail)
		}
		return fmt.Errorf("source-build resource CLI: %w", err)
	}
	return nil
}

// tailBuffer is an io.Writer that retains only the last max bytes written —
// enough to surface the tail of a failing command's stderr (where the error
// reason typically lives) without buffering unbounded output.
type tailBuffer struct {
	max int
	buf []byte
}

func newTailBuffer(max int) *tailBuffer { return &tailBuffer{max: max} }

func (b *tailBuffer) Write(p []byte) (int, error) {
	b.buf = append(b.buf, p...)
	if len(b.buf) > b.max {
		b.buf = b.buf[len(b.buf)-b.max:]
	}
	return len(p), nil
}

func (b *tailBuffer) String() string { return string(b.buf) }
