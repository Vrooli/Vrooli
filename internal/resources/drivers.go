package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	resourceenv "github.com/vrooli/vrooli/internal/resources/env"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	runtimeenv "github.com/vrooli/vrooli/internal/resources/runtime/env"
	runtimelogs "github.com/vrooli/vrooli/internal/resources/runtime/logs"
	runtimestorage "github.com/vrooli/vrooli/internal/resources/runtime/storage"
	"github.com/vrooli/vrooli/internal/shell"
)

const StatusCodeUnsupportedPlatform = "unsupported_platform"

type resourceDriver interface {
	Name() string
	Status(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, fast bool) (Status, error)
	Run(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, action string, args []string, stdout, stderr io.Writer) error
}

var lookPathCommandFn = exec.LookPath

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
		status.Message = "not installed"
		return status, nil
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
	if healthy {
		status.Health = "healthy"
		status.Message = "healthy"
	} else {
		status.Health = "unhealthy"
		status.Message = "unhealthy"
	}
	return status, nil
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
		if err := composeCommand(ctx, controller, manifest, io.Discard, io.Discard, "pull"); err == nil {
			return nil
		}
		return composeCommand(ctx, controller, manifest, io.Discard, io.Discard, "build")
	case "start":
		return composeCommand(ctx, controller, manifest, io.Discard, io.Discard, "up", "-d")
	case "restart":
		return composeCommand(ctx, controller, manifest, io.Discard, io.Discard, "up", "-d", "--force-recreate")
	case "stop":
		return composeCommand(ctx, controller, manifest, io.Discard, io.Discard, "stop")
	case "uninstall":
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
		return dockerCommand(ctx, controller, io.Discard, io.Discard, "start", name)
	}

	args, err := buildDockerRunArgs(controller, manifest, name)
	if err != nil {
		return err
	}
	return dockerCommand(ctx, controller, io.Discard, io.Discard, args...)
}

// buildDockerRunArgs assembles the `docker run` argument list for a docker-service
// resource and performs the host-side filesystem preparation (mkdir/touch) for any
// declared bind-mount volume sources. It is split out from startDockerService so
// driver argument construction (including the --memory cap) can be unit-tested.
func buildDockerRunArgs(controller *Controller, manifest ResourceManifest, name string) ([]string, error) {
	args := []string{"run", "-d", "--name", name}
	for _, port := range manifest.Ports {
		if port.Container <= 0 {
			continue
		}
		hostPort := port.Host
		if hostPort <= 0 {
			hostPort = port.Container
		}
		args = append(args, "-p", strconv.Itoa(hostPort)+":"+strconv.Itoa(port.Container))
	}
	for key, value := range manifest.Runtime.Env {
		args = append(args, "-e", key+"="+expandResourceRuntimeValue(controller, manifest, value))
	}
	if memLimit := strings.TrimSpace(manifest.Runtime.MemoryLimit); memLimit != "" {
		args = append(args, "--memory", memLimit)
	}
	resourceDir := filepath.Join(controller.Root, "resources", manifest.Name)
	for _, volume := range manifest.Runtime.Volumes {
		source := expandResourceRuntimeValue(controller, manifest, volume.Source)
		if !filepath.IsAbs(source) {
			source = filepath.Join(resourceDir, filepath.FromSlash(source))
		}
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
	return cmdArgs, env
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
	command := manifest.Install.Command
	if len(command) == 0 {
		command = manifest.Install.Platforms[manifestpkg.CurrentPlatform()]
	}
	if len(command) == 0 {
		return nil
	}
	cmd := shell.Command(shell.Spec{
		Name:   command[0],
		Args:   command[1:],
		Dir:    controller.Root,
		Env:    resourceEnvForResource(controller.Root, controller.Home, manifest.Name),
		Stdout: io.Discard,
		Stderr: io.Discard,
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
	return waitErr
}
