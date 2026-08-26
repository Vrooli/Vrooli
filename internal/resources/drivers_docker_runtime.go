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

	"github.com/vrooli/vrooli/internal/accel"
	"github.com/vrooli/vrooli/internal/cliinstall"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	runtimeenv "github.com/vrooli/vrooli/internal/resources/runtime/env"
	runtimelogs "github.com/vrooli/vrooli/internal/resources/runtime/logs"
	runtimestorage "github.com/vrooli/vrooli/internal/resources/runtime/storage"
	"github.com/vrooli/vrooli/internal/shell"
)

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

func inspectDockerArtifact(ctx context.Context, controller *Controller, kind, name string) cliinstall.ObservedBefore {
	if strings.TrimSpace(name) == "" {
		return cliinstall.ObservedUnknown
	}
	if _, err := dockerOutput(ctx, controller, kind, "inspect", name); err == nil {
		return cliinstall.ObservedPresent
	} else if isMissingDockerArtifact(err) {
		return cliinstall.ObservedAbsent
	}
	return cliinstall.ObservedUnknown
}

func isMissingDockerArtifact(err error) bool {
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	for _, marker := range []string{
		"no such object",
		"no such image",
		"no such volume",
		"no such network",
		"no such container",
		"not found",
		"does not exist",
	} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}

func inspectDockerVolumeArtifacts(ctx context.Context, controller *Controller, manifest ResourceManifest) map[string]cliinstall.ObservedBefore {
	observed := make(map[string]cliinstall.ObservedBefore)
	for _, volume := range manifest.Runtime.Volumes {
		name := strings.TrimSpace(volume.Source)
		if name == "" || filepath.IsAbs(name) || strings.HasPrefix(name, ".") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
			continue
		}
		observed[name] = inspectDockerArtifact(ctx, controller, "volume", name)
	}
	return observed
}

func recordDockerArtifact(controller *Controller, manifest ResourceManifest, kind cliinstall.InstallEntryKind, name string, before cliinstall.ObservedBefore, node string) error {
	action := cliinstall.ActionInstalled
	if before == cliinstall.ObservedPresent {
		action = cliinstall.ActionAdopted
	}
	return cliinstall.RecordContainerArtifactWithProvenance(controller.Home, cliinstall.ScopeRuntime, kind, name, manifest.Name, node, false, before, action)
}

func recordDockerImageArtifact(controller *Controller, manifest ResourceManifest, before cliinstall.ObservedBefore) error {
	if strings.TrimSpace(manifest.Runtime.Image) == "" {
		return nil
	}
	node, _ := os.Hostname()
	if err := recordDockerArtifact(controller, manifest, cliinstall.EntryImage, manifest.Runtime.Image, before, node); err != nil {
		return fmt.Errorf("record resource %q image provenance: %w", manifest.Name, err)
	}
	return nil
}

func startDockerService(ctx context.Context, controller *Controller, manifest ResourceManifest, restart bool, warning io.Writer) error {
	if err := gateAcceleratorReadiness(ctx, manifest, warning); err != nil {
		return err
	}
	state, exists, err := inspectDockerContainer(ctx, controller, manifest)
	if err != nil {
		return err
	}

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
	imageBefore := inspectDockerArtifact(ctx, controller, "image", manifest.Runtime.Image)
	volumeBefore := inspectDockerVolumeArtifacts(ctx, controller, manifest)
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
			if err := dockerCommand(ctx, controller, io.Discard, io.Discard, "restart", name); err != nil {
				return err
			}
			return verifyStartedPlacement(ctx, controller, manifest, warning)
		}
		if state.Running {
			return verifyStartedPlacement(ctx, controller, manifest, warning)
		}

		if err := preflightPortConflict(manifest); err != nil {
			return err
		}
		if err := dockerCommand(ctx, controller, io.Discard, io.Discard, "start", name); err != nil {
			return err
		}
		return verifyStartedPlacement(ctx, controller, manifest, warning)
	}

	if err := preflightPortConflict(manifest); err != nil {
		return err
	}

	args, err := buildDockerRunArgs(ctx, controller, manifest, name)
	if err != nil {
		return err
	}
	if err := dockerCommand(ctx, controller, io.Discard, io.Discard, args...); err != nil {
		return err
	}
	if err := recordDockerArtifacts(controller, manifest, imageBefore, volumeBefore); err != nil {
		return err
	}
	return verifyStartedPlacement(ctx, controller, manifest, warning)
}

func recordDockerArtifacts(controller *Controller, manifest ResourceManifest, imageBefore cliinstall.ObservedBefore, volumeBefore map[string]cliinstall.ObservedBefore) error {
	node, _ := os.Hostname()
	if strings.TrimSpace(manifest.Runtime.Image) != "" {
		if err := recordDockerArtifact(controller, manifest, cliinstall.EntryImage, manifest.Runtime.Image, imageBefore, node); err != nil {
			return fmt.Errorf("record resource %q image provenance: %w", manifest.Name, err)
		}
	}
	if err := recordDockerArtifact(controller, manifest, cliinstall.EntryContainer, dockerContainerName(manifest), cliinstall.ObservedAbsent, node); err != nil {
		return fmt.Errorf("record resource %q container provenance: %w", manifest.Name, err)
	}
	for _, volume := range manifest.Runtime.Volumes {
		name := strings.TrimSpace(volume.Source)
		if name == "" || filepath.IsAbs(name) || strings.HasPrefix(name, ".") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
			continue
		}
		before := cliinstall.ObservedUnknown
		if observed, ok := volumeBefore[name]; ok {
			before = observed
		}
		if err := recordDockerArtifact(controller, manifest, cliinstall.EntryVolume, name, before, node); err != nil {
			return fmt.Errorf("record resource %q volume provenance: %w", manifest.Name, err)
		}
	}
	return nil
}

func recordComposeArtifacts(ctx context.Context, controller *Controller, manifest ResourceManifest, recordServices, networkBefore bool) error {
	services, err := inspectComposeServices(ctx, controller, manifest)
	if err != nil {
		return fmt.Errorf("inspect resource %q containers for provenance: %w", manifest.Name, err)
	}
	node, _ := os.Hostname()
	if recordServices && len(services) == 0 {
		if err := cliinstall.RecordContainerArtifact(controller.Home, cliinstall.ScopeRuntime, cliinstall.EntryContainer, dockerContainerName(manifest), manifest.Name, node, false); err != nil {
			return fmt.Errorf("record resource %q container provenance: %w", manifest.Name, err)
		}
	}
	for _, service := range services {
		if !recordServices {
			break
		}
		if strings.TrimSpace(service.Name) == "" {
			continue
		}
		if err := cliinstall.RecordContainerArtifact(controller.Home, cliinstall.ScopeRuntime, cliinstall.EntryContainer, service.Name, manifest.Name, node, false); err != nil {
			return fmt.Errorf("record resource %q service container provenance: %w", manifest.Name, err)
		}
	}
	network := composeProjectName(manifest) + "_default"
	if !networkBefore {
		if err := cliinstall.RecordContainerArtifact(controller.Home, cliinstall.ScopeRuntime, cliinstall.EntryNetwork, network, manifest.Name, node, false); err != nil {
			return fmt.Errorf("record resource %q network provenance: %w", manifest.Name, err)
		}
	}
	return nil
}

func inspectComposeNetwork(ctx context.Context, controller *Controller, manifest ResourceManifest) (bool, error) {
	network := composeProjectName(manifest) + "_default"
	_, err := dockerOutput(ctx, controller, "network", "inspect", network)
	if err == nil {
		return true, nil
	}
	if isMissingDockerArtifact(err) {
		return false, nil
	}
	return false, fmt.Errorf("inspect resource %q network: %w", manifest.Name, err)
}

var gpuVerificationSleep = func(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
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
		for key, value := range acceleratedBackendEnv(ctx, manifest) {
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
	backend, _, ok := selectedAcceleratedBackend(ctx, manifest)
	return ok && backend != accel.BackendCPU
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
	return runDockerLifecycleCommand(ctx, controller, stdout, stderr, args...)
}

func runDockerLifecycleCommandReal(ctx context.Context, controller *Controller, stdout, stderr io.Writer, args ...string) error {
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

	if backend, config, ok := selectedAcceleratedBackend(ctx, manifest); ok {
		if overlay := strings.TrimSpace(config.ComposeOverlay); overlay != "" && backend != accel.BackendCPU {
			cmdArgs = append(cmdArgs, "-f", composeOverlayPath(controller, manifest, overlay))
		}
		for key, value := range config.Env {
			env = append(env, key+"="+value)
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
