package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/accel"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
)

const StatusCodeUnsupportedPlatform = "unsupported_platform"

type resourceDriver interface {
	Name() string
	Status(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, fast bool) (Status, error)
	Run(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, action string, args []string, stdout, stderr io.Writer) error
}

var (
	lookPathCommandFn         = exec.LookPath
	runSourceBuildCommandFn   = func(cmd *exec.Cmd) error { return cmd.Run() }
	runDockerLifecycleCommand = runDockerLifecycleCommandReal
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
	case "managed-service":
		return managedServiceDriver{}, nil
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
	mode, gpuReason, gpuState := observedGPUStatus(ctx, controller, manifest, services)

	healthy := true
	status.Healthy = &healthy
	status.Health = "running"
	status.Message = "running"
	if mode != "" {
		status.Raw = statusRawWithMode(status.Raw, mode, gpuState, gpuReason)
	}
	if len(manifest.HealthChecks) > 0 {
		health, err := controller.runResourceHealthChecks(ctx, manifest)
		if err != nil {
			status.StatusCode = StatusCodeCommandError
			status.Message = "health checks failed"
			status.ProbeError = err.Error()
			return status, nil
		}
		status = applyHealthToStatus(status, health)
		healthy = health.Healthy
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
	if gpuState != string(GPUAccessOK) && mode != "" {
		healthy = false
		status.Healthy = &healthy
	}
	if healthy {
		status.Health = "healthy"
		status.Message = "healthy"
	} else {
		status.Health = "unhealthy"
		status.Message = "unhealthy"
	}
	if mode != "" {
		status.Message = fmt.Sprintf("%s (mode: %s)", status.Message, mode)
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

func statusRawWithMode(raw json.RawMessage, mode string, stateAndReason ...string) json.RawMessage {
	payload := make(map[string]any)
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &payload)
	}
	payload["mode"] = mode
	if len(stateAndReason) > 0 && strings.TrimSpace(stateAndReason[0]) != "" {
		payload["gpu_state"] = stateAndReason[0]
	}
	if len(stateAndReason) > 1 && strings.TrimSpace(stateAndReason[1]) != "" {
		payload["gpu_reason"] = stateAndReason[1]
	}
	out, err := json.Marshal(payload)
	if err != nil {
		return raw
	}
	return out
}

// observedGPUStatus reports the backend a running resource is on, for the
// status paths that do not run health checks (an externally managed service, or
// a resource with no declared health check). It is a thin adapter over the one
// placement verifier: there is no second opinion about placement anywhere.
func observedGPUStatus(ctx context.Context, controller *Controller, manifest ResourceManifest, _ []composeServiceState) (string, string, string) {
	placement, err := observePlacement(ctx, controller, manifest)
	if err != nil || placement == nil {
		return "", "", ""
	}
	observed := string(placement.Observed)
	if observed == "" {
		observed = "unknown"
	}
	return observed, placement.Reason, string(placement.State)
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
				"raw":        status.Raw,
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
		if err := gateAcceleratorReadiness(ctx, manifest, stderr); err != nil {
			return err
		}
		before, err := inspectComposeServices(ctx, controller, manifest)
		if err != nil {
			return err
		}
		networkBefore, err := inspectComposeNetwork(ctx, controller, manifest)
		if err != nil {
			return err
		}
		if running, err := composeFallbackContainerHealthy(ctx, controller, manifest); err != nil {
			return err
		} else if running {
			return verifyStartedPlacement(ctx, controller, manifest, stderr)
		}
		if err := composeCommand(ctx, controller, manifest, io.Discard, stderr, "up", "-d"); err != nil {
			return err
		}
		startCompanions(manifest.Name, manifest.Companions, manifest.Orchestration.RecoveryAttempts, stderr)
		if len(before) == 0 {
			if err := recordComposeArtifacts(ctx, controller, manifest, true, networkBefore); err != nil {
				return err
			}
		}
		return verifyStartedPlacement(ctx, controller, manifest, stderr)
	case "restart":
		if err := gateAcceleratorReadiness(ctx, manifest, stderr); err != nil {
			return err
		}
		before, err := inspectComposeServices(ctx, controller, manifest)
		if err != nil {
			return err
		}
		networkBefore, err := inspectComposeNetwork(ctx, controller, manifest)
		if err != nil {
			return err
		}
		if err := composeCommand(ctx, controller, manifest, io.Discard, stderr, "up", "-d", "--force-recreate"); err != nil {
			return err
		}
		startCompanions(manifest.Name, manifest.Companions, manifest.Orchestration.RecoveryAttempts, stderr)
		if len(before) == 0 {
			if err := recordComposeArtifacts(ctx, controller, manifest, true, networkBefore); err != nil {
				return err
			}
		}
		return verifyStartedPlacement(ctx, controller, manifest, stderr)
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
			if _, accelerated := accelSpecFor(manifest); accelerated {
				mode, reason, state := observedGPUStatus(ctx, controller, manifest, nil)
				status.Raw = statusRawWithMode(status.Raw, mode, state, reason)
				status.Message += fmt.Sprintf(" (mode: %s)", mode)
			}
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
			status = applyHealthToStatus(status, health)
			healthy = health.Healthy
		} else {
			status.Message = "running"
			status.Healthy = &healthy
			serving := healthy
			status.Serving = &serving
		}
		if _, accelerated := accelSpecFor(manifest); accelerated {
			mode, reason, state := observedGPUStatus(ctx, controller, manifest, nil)
			status.Raw = statusRawWithMode(status.Raw, mode, state, reason)
			status.Message += fmt.Sprintf(" (mode: %s)", mode)
			if state != string(accel.StateOK) {
				healthy = false
				status.Healthy = &healthy
				status.Health = "unhealthy"
				status.Message = "unhealthy" + fmt.Sprintf(" (mode: %s)", mode)
			}
		}
		if healthy {
			status.Health = "healthy"
			status.Message = "healthy"
		} else {
			status.Health = "unhealthy"
			status.Message = "unhealthy"
		}
		status = appendOllamaProcessor(ctx, controller, manifest, status)
		return status, nil
	}

	if external, err := probeExternalDockerService(ctx, controller, manifest); err == nil && external {
		status.Running = true
		healthy := true
		status.Healthy = &healthy
		status.Health = "healthy"
		status.Message = "healthy (external)"
		if _, accelerated := accelSpecFor(manifest); accelerated {
			mode, reason, state := observedGPUStatus(ctx, controller, manifest, nil)
			status.Raw = statusRawWithMode(status.Raw, mode, state, reason)
			status.Message += fmt.Sprintf(" (mode: %s)", mode)
			if state != string(accel.StateOK) {
				healthy = false
				status.Healthy = &healthy
				status.Health = "unhealthy"
				status.Message = "unhealthy" + fmt.Sprintf(" (mode: %s)", mode)
			}
		}
		return status, nil
	}

	healthy := false
	status.Healthy = &healthy
	status.Health = "stopped"
	status.Message = "stopped"
	return status, nil
}

func appendOllamaProcessor(ctx context.Context, controller *Controller, manifest ResourceManifest, status Status) Status {
	if manifest.Name != "ollama" {
		return status
	}
	var output bytes.Buffer
	if err := controller.RunResourceCLI("ollama", []string{"health-gpu", "--json"}, &output, io.Discard); err != nil && output.Len() == 0 {
		return status
	}
	var payload map[string]any
	if err := json.Unmarshal(output.Bytes(), &payload); err != nil {
		return status
	}
	processor, _ := payload["processor"].(string)
	if strings.TrimSpace(processor) == "" {
		return status
	}
	var raw map[string]any
	if err := json.Unmarshal(status.Raw, &raw); err != nil {
		raw = map[string]any{}
	}
	raw["processor"] = processor
	if hostGPU, ok := payload["host_nvidia_gpu"].(bool); ok && hostGPU {
		if hasGPU, ok := payload["has_gpu_model"].(bool); ok && !hasGPU {
			unhealthy := false
			status.Healthy = &unhealthy
			status.Health = "unhealthy"
		}
	}
	status.Raw, _ = json.Marshal(raw)
	status.Message += fmt.Sprintf(" (processor: %s)", processor)
	return status
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
				"raw":       status.Raw,
			})
		}
		_, err = fmt.Fprintf(stdout, "%s: %s\n", item.Name, status.Message)
		return err
	case "install":
		imageBefore := inspectDockerArtifact(ctx, controller, "image", manifest.Runtime.Image)
		if err := ensureDockerImage(ctx, controller, manifest); err != nil {
			return err
		}
		return recordDockerImageArtifact(controller, manifest, imageBefore)
	case "start":
		return startDockerService(ctx, controller, manifest, false, stderr)
	case "restart":
		return startDockerService(ctx, controller, manifest, true, stderr)
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
	Name    string `json:"Name"`
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
