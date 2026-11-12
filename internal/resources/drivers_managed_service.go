package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/lightingruntime"

	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/credentialauthority"
	"github.com/vrooli/vrooli/internal/securestore"
	valuehelpers "github.com/vrooli/vrooli/internal/values"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

type managedServiceDriver struct{}

const (
	managedServiceHealthy   = "healthy"
	managedServiceUnhealthy = "unhealthy"
)

func (managedServiceDriver) Name() string { return accelBridgeManagedService }

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
	if l.Manifest.Lifecycle.ExclusiveGroup != "" {
		return nil, fmt.Errorf("exclusive group %q requires managed-private lifecycle; shared broker activation refused", l.Manifest.Lifecycle.ExclusiveGroup)
	}
	if l.Controller == nil || instance.Resource != l.Manifest.Name || instance.Provider != resourcedeployment.ProviderManagedShared {
		return nil, fmt.Errorf("owner lifecycle does not match a managed shared %s instance", l.Manifest.Name)
	}
	driver := managedServiceDriver{}
	item := Resource{Name: l.Manifest.Name}
	switch action {
	case brokerTransportStart, brokerTransportRestart:
		var result any
		err := withManagedServiceLifecycleLock(ctx, l.Manifest.Name, func() error {
			if err := l.Controller.withResourceCapacity(ctx, l.Manifest.Name, action, func(ctx context.Context) error {
				return driver.runUserHosted(ctx, l.Controller, item, l.Manifest, action, io.Discard)
			}); err != nil {
				return err
			}
			var err error
			result, err = driver.Status(ctx, l.Controller, item, l.Manifest, false)
			return err
		})
		return result, err
	case brokerTransportStop:
		var result any
		err := withManagedServiceLifecycleLock(ctx, l.Manifest.Name, func() error {
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
	if manifest.Name == openRGBResourceName {
		if err := controller.ValidateOpenRGBNativeResourceStorage(ctx); err != nil {
			status.StatusCode = StatusCodeCommandError
			status.Message = "native OpenRGB storage configuration is unverified"
			status.ProbeError = err.Error()
			return status, nil
		}
	}
	if manifest.ManagedService != nil && manifest.ManagedService.TaskScopedContainer {
		status.Installed = true
		status.Message = "task-scoped runtime; no singleton resource process (use `vrooli resource n8n runtime status --sandbox-id <uuid> --json`)"
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

	// Running identity is an independent fact from the current release declaration.
	// Inspect it first so staging an update cannot make a working service appear down.
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
	status.DeclaredArtifactVersion = manifest.ManagedService.Artifact.Version
	if raw, err := json.Marshal(state); err == nil {
		status.Raw = raw
	}
	if controller.healthObservationOnly {
		return d.observeManagedServiceHealth(ctx, controller, status, manifest, state, running, fast)
	}
	if running {
		status.Installed, status.Running = true, true
		status.RunningArtifactVersion = state.ArtifactVersion
		status.ArtifactUpdateAvailable = state.ArtifactVersion != status.DeclaredArtifactVersion
		if err := verifyManagedServiceRunningArtifact(ctx, state); err != nil {
			status.ArtifactVerification = "failed"
			healthy := false
			status.Healthy = &healthy
			// Kept strict: a process whose image cannot be proven to be the
			// bytes it launched from is not admitted as a dependency. The
			// remedy is one command, so the message names it.
			status.StatusCode, status.ProbeError = StatusCodeUnavailable, err.Error()
			status.Message = managedServiceUnverifiableImageMessage(manifest.Name)
			return status, nil
		}
		status.ArtifactVerification = "verified"
		if info, err := controller.inspectManagedServiceArtifactWithRunning(ctx, manifest, &state); err != nil {
			status.ArtifactStagingError = err.Error()
		} else {
			status.StagedArtifactVersion = info.Version
			status.ArtifactUpdateAvailable = status.ArtifactUpdateAvailable || state.ArtifactSHA256 != info.SHA256
		}
		return d.withRunningHealth(ctx, controller, status, manifest, fast)
	}
	artifactPath, err := managedServiceArtifactPath(controller, manifest)
	if err != nil {
		return status, err
	}
	artifact, err := managedServiceArtifactForLaunch(ctx, controller, manifest, artifactPath)
	if err != nil {
		var drift *FactDriftError
		if errors.As(err, &drift) {
			// The bytes are intact and the host moved. Saying "unavailable"
			// here is what left reranker with no cure: it named neither the
			// cause nor the command that fixes it.
			status.Installed = true
			status.StatusCode = StatusCodeNeedsReacquire
			status.Message = "host facts changed since install; re-acquire with `" + drift.Remediation() + "`"
			status.ProbeError = drift.Error()
			return status, nil
		}
		status.StatusCode = StatusCodeUnavailable
		status.Message = "verified managed-service artifact is unavailable for this host"
		status.ProbeError = err.Error()
		return status, nil
	}
	if err := artifact.VerifyFileContext(ctx, artifactPath); err != nil {
		status.ArtifactVerification = "failed"
		// A digest mismatch is not the same event as a missing or unreadable
		// artifact, and reporting both as "unavailable" is what let a staged,
		// running, GPU-resident resource read as not installed with no cure
		// named. Classify it: the bytes are staged, so the operator needs to
		// know whether the claim moved or the bytes did, and the command that
		// repairs either.
		var mismatch *resourcedeployment.ArtifactChecksumMismatchError
		if errors.As(err, &mismatch) {
			cause := classifyArtifactMismatch(artifactPath, manifest.Name, mismatch.Want, mismatch.Layout)
			if cause != nil {
				status.Installed = true
				status.StatusCode = StatusCodeNeedsReacquire
				status.Message = "staged artifact no longer matches its manifest (" + string(cause.Cause) + "); re-acquire with `" + cause.Remediation() + "`"
				status.ProbeError = cause.Error()
				return status, nil
			}
		}
		status.StatusCode = StatusCodeUnavailable
		status.Message = "verified managed-service artifact is unavailable"
		status.ProbeError = err.Error()
		return status, nil
	}
	status.Installed = true
	status.ArtifactVerification = "verified"
	if declaration := manifest.EffectiveAcceleration(); declaration != nil && len(declaration.Backends) > 0 {
		// A stopped resource cannot have placement evidence, but its declared
		// mode remains known. The explicit sentinel distinguishes that state
		// from a resource that has no acceleration declaration.
		status.DeclaredMode = declaration.Backends[0]
		status.ObservedMode = "not_evaluated"
	}
	status.StagedArtifactVersion = manifest.ManagedService.Artifact.Version
	healthy := false
	status.Healthy = &healthy
	status.Health = "stopped"
	return status, nil
}

func (d managedServiceDriver) withRunningHealth(ctx context.Context, controller *Controller, status Status, manifest ResourceManifest, fast bool) (Status, error) {
	if declaration := manifest.EffectiveAcceleration(); declaration != nil && len(declaration.Backends) > 0 {
		status.DeclaredMode, status.ObservedMode = declaration.Backends[0], "not_evaluated"
	}
	status.Running = true
	healthy := true
	status.Healthy = &healthy
	status.Health = "running"
	status.Message = "running"
	if fast || len(manifest.HealthChecks) == 0 {
		// Placement is a cheap, independent observation. Keep the expensive
		// readiness/liveness checks out of the fast path, but never leave
		// accelerator fields blank merely because those checks were skipped.
		health := controller.foldPlacementIntoHealth(ctx, manifest, HealthResult{
			Healthy: true,
			Serving: true,
		})
		status = applyHealthToStatus(status, health)
		return d.withCompanionStatus(status, manifest)
	}
	health, err := controller.runResourceHealthChecks(ctx, manifest)
	if err != nil {
		status.StatusCode = StatusCodeCommandError
		status.Message = "health checks failed"
		status.ProbeError = err.Error()
		return status, nil
	}
	status = applyHealthToStatus(status, health)
	if status.Message == "" {
		if health.Healthy {
			status.Message = managedServiceHealthy
		} else {
			status.Message = managedServiceUnhealthy
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
		// Optional companions remain visible in the message but do not change
		// the primary resource health verdict.
		status.Message = companionDownMessage(manifest.Name, down)
	}
	if down := requiredDownCompanions(companions); len(down) > 0 {
		healthy := false
		status.Healthy = &healthy
		status.Health = managedServiceUnhealthy
		status.Message = companionDownMessage(manifest.Name, down)
	}
	return status, nil
}

//nolint:gocyclo // managed-service execution handles provider, process, health, and diagnostic action branches.
func (d managedServiceDriver) Run(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, action string, args []string, stdout, stderr io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if manifest.Name == openRGBResourceName && (action == brokerTransportStart || action == brokerTransportRestart) {
		return runOpenRGBNativeResourceAction(ctx, controller, action, args)
	}
	if manifest.Name == openRGBResourceName && action == brokerTransportStop && (runtime.GOOS == "darwin" || runtime.GOOS == "windows") {
		if len(args) != 0 {
			return errors.New("OpenRGB native stop accepts no process or launch overrides")
		}
		if err := requireOpenRGBNativeOperatorAuthority(); err != nil {
			return err
		}
		if err := controller.ValidateOpenRGBNativeResourceStorage(ctx); err != nil {
			return err
		}
		owner, err := lightingruntime.NewController(controller.Home, lightingruntime.Owners{Resource: NewOpenRGBHeadless(controller, controller.OpenRGBDevicePolicyVerifier()), Scenario: openRGBNativeStopScenario{quiesce: controller.openRGBPublicOwners().quiesce}})
		if err != nil {
			return err
		}
		return owner.Disable(ctx)
	}

	if manifest.ManagedService != nil && manifest.ManagedService.TaskScopedContainer {
		if err := ensureSupportedPlatform(manifest); err != nil {
			return managedServiceDriverError(item.Name, action, "Platform", err)
		}
		if action == "status" {
			status, err := d.Status(ctx, controller, item, manifest, !slices.Contains(args, "--no-fast"))
			if err != nil {
				return err
			}
			if slices.Contains(args, "--format") && nextArgValue(args, "--format") == "json" {
				return json.NewEncoder(stdout).Encode(map[string]any{"installed": status.Installed, "running": status.Running, "healthy": status.Healthy, "health": status.Health, "message": status.Message, "provider": "managed-private"})
			}
			_, err = fmt.Fprintf(stdout, "%s: %s\n", item.Name, status.Message)
			return err
		}
		return managedServiceDriverError(item.Name, action, "Lifecycle", fmt.Errorf("task-scoped container lifecycle is available only through `vrooli resource n8n runtime ...`; generic singleton lifecycle is disabled"))
	}
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
		status, err := d.Status(ctx, controller, item, manifest, !slices.Contains(args, "--no-fast"))
		if err != nil {
			return err
		}
		if slices.Contains(args, "--format") && nextArgValue(args, "--format") == "json" {
			return json.NewEncoder(stdout).Encode(map[string]any{
				"installed": status.Installed, "running": status.Running, "healthy": status.Healthy,
				"health": status.Health, "message": status.Message, "provider": mode,
			})
		}
		_, err = fmt.Fprintf(stdout, "%s: %s\n", item.Name, status.Message)
		return err
	case "install":
		return ensureManagedServiceArtifact(ctx, controller, manifest)
	case brokerTransportStart, brokerTransportRestart, brokerTransportStop, driversDockerUninstall, driversCliLogs:
		mode, err := managedServiceProvider(manifest, args)
		if err != nil {
			return managedServiceDriverError(item.Name, action, "Provider", err)
		}
		if managedServiceLifecycleAction(action) {
			release, err := acquireManagedLifecycleLocks(ctx, manifest)
			if err != nil {
				return managedServiceDriverError(item.Name, action, "Lifecycle", err)
			}
			defer release()
			if err := ensureManagedUpdateLifecycleAvailable(manifest.Name, action); err != nil {
				return managedServiceDriverError(item.Name, action, "Lifecycle", err)
			}
		}
		if manifest.Lifecycle.ExclusiveGroup != "" {
			if mode != resourcedeployment.ProviderManagedPrivate {
				return fmt.Errorf("exclusive resource groups require the managed-private provider")
			}
			if action == brokerTransportStart || action == brokerTransportRestart {
				if err := checkExclusiveResourceGroup(controller, manifest); err != nil {
					return managedServiceDriverError(item.Name, action, "ExclusiveGroup", err)
				}
				if err := pruneIdleExclusiveModelCaches(ctx, controller, manifest); err != nil {
					_, _ = fmt.Fprintf(stderr, "warning: inactive model-cache retention skipped: %s\n", err)
				}
			}
		}
		if manifest.Name == "whisper" && mode == resourcedeployment.ProviderManagedPrivate && (action == brokerTransportStart || action == brokerTransportRestart) {
			supervisor, _, supervisorErr := managedServiceSupervisorFor(manifest.Name)
			if supervisorErr != nil {
				return supervisorErr
			}
			if err := recoverWhisperOpenResidency(ctx, controller, manifest, supervisor); err != nil {
				return fmt.Errorf("reconcile interrupted Whisper residency before new admission: %w", err)
			}
		}
		// Admission is serialized with stop/restart. A lease created while
		// queued outside this lock can be released by the preceding stop.
		runErr := controller.withResourceCapacity(ctx, manifest.Name, action, func(ctx context.Context) error {
			if mode == resourcedeployment.ProviderManagedShared {
				if action == brokerTransportStart || action == brokerTransportRestart {
					if err := migrateLegacyDockerStorage(ctx, controller, manifest); err != nil {
						return err
					}
				}
				return d.runUserHosted(ctx, controller, item, manifest, action, stdout, args...)
			}
			if mode == resourcedeployment.ProviderManagedDiscovered && (action == brokerTransportStart || action == brokerTransportRestart) {
				return d.runDiscovered(ctx, controller, item, manifest, action, args)
			}
			if mode != resourcedeployment.ProviderManagedPrivate {
				return managedServiceDriverError(item.Name, action, "Provider", fmt.Errorf("%s provider does not grant local lifecycle authority; use the broker or provider-specific client", mode))
			}
			if action == brokerTransportStart || action == brokerTransportRestart {
				if err := migrateLegacyDockerStorage(ctx, controller, manifest); err != nil {
					return err
				}
			}
			return d.runPrivate(ctx, controller, item, manifest, action, stdout, args...)
		})
		if runErr == nil && mode == resourcedeployment.ProviderManagedPrivate && (action == brokerTransportStart || action == brokerTransportRestart || action == brokerTransportStop) {
			event := modelCacheEventStarted
			if action == brokerTransportStop {
				event = modelCacheEventStopped
			}
			if err := noteModelCaches(ctx, manifest, event); err != nil {
				_, _ = fmt.Fprintf(stderr, "warning: model-cache usage not recorded; cleanup remains protected: %s\n", err)
			}
		}
		if runErr == nil && mode == resourcedeployment.ProviderManagedPrivate && action == driversDockerUninstall {
			runErr = uninstallManagedServiceArtifact(ctx, controller, manifest)
		}
		return runErr
	default:
		return managedServiceDriverError(item.Name, action, "Driver", fmt.Errorf("action %q is not supported by driver %q", action, d.Name()))
	}
}

// runUserHosted owns shared lifecycle mechanics while a resource registration
// owns its bootstrap protocol and management material.
func (d managedServiceDriver) runUserHosted(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, action string, stdout io.Writer, args ...string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if manifest.Name == openRGBResourceName && (action == brokerTransportStart || action == brokerTransportRestart) {
		return ErrOpenRGBPreExecWitnessUnavailable
	}
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
	case brokerTransportStart:
		if err := gateAcceleratorReadiness(ctx, manifest, os.Stderr); err != nil {
			return err
		}
		if err := d.startPrivate(ctx, controller, manifest, supervisor); err != nil {
			return err
		}
	case brokerTransportRestart:
		// Acquire and verify replacement bytes before interrupting the
		// serving instance. A network or checksum failure must preserve it.
		if err := ensureManagedServiceArtifact(ctx, controller, manifest); err != nil {
			return fmt.Errorf("prepare managed-service restart: %w", err)
		}
		stopCtx, cancel := managedServiceStopContext(ctx, manifest)
		err := stopManagedService(stopCtx, manifest, supervisor)
		cancel()
		if err != nil {
			return err
		}
		if err := d.startPrivate(ctx, controller, manifest, supervisor); err != nil {
			return err
		}
	case brokerTransportStop, driversDockerUninstall, driversCliLogs:
		return d.runPrivate(ctx, controller, item, manifest, action, stdout, args...)
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
	client := &http.Client{Timeout: tuning.ControlPlaneClientTimeout()}
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

func (d managedServiceDriver) runPrivate(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, action string, stdout io.Writer, args ...string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if manifest.Name == openRGBResourceName && (action == brokerTransportStart || action == brokerTransportRestart) {
		return ErrOpenRGBPreExecWitnessUnavailable
	}
	supervisor, _, err := managedServiceSupervisorFor(manifest.Name)
	if err != nil {
		return err
	}
	switch action {
	case brokerTransportStart:
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
		if err := ctx.Err(); err != nil {
			return err
		}
		startCompanions(manifest.Name, manifest.Companions, manifest.Orchestration.RecoveryAttempts, os.Stderr, managedServiceParentPID(supervisor))
		if err := waitForManagedServiceHealth(ctx, controller, manifest); err != nil {
			return err
		}
		if err := verifyManagedServiceRunning(supervisor); err != nil {
			return err
		}
		if err := verifyStartedPlacement(ctx, controller, manifest, os.Stderr); err != nil {
			return err
		}
		if hasStableActivityEdge(manifest.Name) {
			if err := controlResourceActivityEdge(ctx, manifest.Name, "resume"); err != nil {
				return fmt.Errorf("resume %s activity edge after verified start: %w", manifest.Name, err)
			}
		}
		if err := recordQualificationObservation(controller.Root, manifest.Name); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "warning: record portability qualification for %q: %s\n", manifest.Name, err)
		}
		return nil
	case brokerTransportRestart:
		if err := gateAcceleratorReadiness(ctx, manifest, os.Stderr); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := ensureManagedServiceArtifact(ctx, controller, manifest); err != nil {
			return fmt.Errorf("prepare managed-service restart: %w", err)
		}
		stableActivityEdge := hasStableActivityEdge(manifest.Name)
		var whisperTransition *whisperResidencyTransition
		if stableActivityEdge {
			if err := controlResourceActivityEdge(ctx, manifest.Name, "drain"); err != nil {
				return fmt.Errorf("drain %s activity edge before placement restart: %w", manifest.Name, err)
			}
			if manifest.Name == "whisper" && strings.EqualFold(strings.TrimSpace(os.Getenv("VROOLI_GPU")), "off") {
				whisperTransition, err = beginWhisperCPUTransition(ctx, controller, manifest)
				if err != nil {
					resumeErr := controlWhisperActivityEdge(ctx, "resume")
					return errors.Join(fmt.Errorf("begin durable Whisper CPU placement transition: %w", err), resumeErr)
				}
			}
		} else {
			stopCompanions(manifest.Name, manifest.Companions, os.Stderr)
		}
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
		if err := ctx.Err(); err != nil {
			return err
		}
		startCompanions(manifest.Name, manifest.Companions, manifest.Orchestration.RecoveryAttempts, os.Stderr, managedServiceParentPID(supervisor))
		if err := waitForManagedServiceHealth(ctx, controller, manifest); err != nil {
			return err
		}
		if err := verifyManagedServiceRunning(supervisor); err != nil {
			return err
		}
		if err := verifyStartedPlacement(ctx, controller, manifest, os.Stderr); err != nil {
			return err
		}
		if err := recordQualificationObservation(controller.Root, manifest.Name); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "warning: record portability qualification for %q: %s\n", manifest.Name, err)
		}
		if whisperTransition != nil {
			if err := whisperTransition.completeCPU(ctx, controller, manifest, supervisor); err != nil {
				return fmt.Errorf("complete durable Whisper CPU placement transition: %w", err)
			}
		}
		if stableActivityEdge {
			if err := controlResourceActivityEdge(ctx, manifest.Name, "resume"); err != nil {
				return fmt.Errorf("resume %s activity edge after verified restart: %w", manifest.Name, err)
			}
		}
		return nil
	case brokerTransportStop, "uninstall":
		if err := ctx.Err(); err != nil {
			return err
		}
		if manifest.Name == "whisper" && strings.TrimSpace(os.Getenv("VROOLI_CAPACITY_UNLOAD")) == "1" {
			if len(args) != 0 {
				return fmt.Errorf("capacity-driven Whisper unload accepts no launch or process overrides")
			}
			return stopWhisperForCapacityUnload(ctx, controller, manifest, supervisor)
		}
		if hasStableActivityEdge(manifest.Name) {
			if err := controlResourceActivityEdge(ctx, manifest.Name, "drain"); err != nil {
				return fmt.Errorf("drain %s activity edge before stop: %w", manifest.Name, err)
			}
		}
		stopCompanions(manifest.Name, manifest.Companions, os.Stderr)
		stopCtx, cancel := managedServiceStopContext(ctx, manifest)
		defer cancel()
		return capacity.ReleaseResourceAdmissionAfterStop(stopCtx, capacity.AdmitOptions{Root: controller.Root, ResourceName: manifest.Name, Log: os.Stderr}, func() error { return stopManagedService(stopCtx, manifest, supervisor) })
	case "logs":
		return supervisor.LogsTail(stdout, managedServiceLogTailLines(args))
	default:
		return fmt.Errorf("unsupported private managed-service action %q", action)
	}
}

func managedServiceLogTailLines(args []string) int {
	for index, arg := range args {
		if arg == "--tail" && index+1 < len(args) {
			if lines, err := strconv.Atoi(args[index+1]); err == nil {
				return lines
			}
		}
		if value, ok := strings.CutPrefix(arg, "--tail="); ok {
			if lines, err := strconv.Atoi(value); err == nil {
				return lines
			}
		}
	}
	return 200
}

func (d managedServiceDriver) bootstrapPrivate(ctx context.Context, manifest ResourceManifest, supervisor *ManagedServiceSupervisor) error {
	if err := ctx.Err(); err != nil {
		return err
	}
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

func managedServiceParentPID(supervisor *ManagedServiceSupervisor) int {
	if supervisor == nil {
		return 0
	}
	state, running, err := supervisor.Status()
	if err != nil || !running {
		return 0
	}
	return state.PID
}

func (d managedServiceDriver) startPrivate(ctx context.Context, controller *Controller, manifest ResourceManifest, supervisor *ManagedServiceSupervisor) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if manifest.Name == openRGBResourceName {
		return ErrOpenRGBPreExecWitnessUnavailable
	}
	// Reconcile ownership before any port preflight. A service already running
	// under this supervisor is an idempotent successful start; probing its port
	// first incorrectly rejects that healthy owned process as a conflict.
	if _, running, err := supervisor.Status(); err != nil {
		return fmt.Errorf("reconcile managed-service ownership before start: %w", err)
	} else if running {
		return nil
	}
	if err := verifyManagedServiceDataGroupAvailability(ctx, manifest); err != nil {
		return err
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
	if err := d.startPrivateAt(ctx, controller, manifest, supervisor, path); err != nil {
		return err
	}
	retainManagedServiceArtifactsAfterActivation(ctx, controller, manifest, supervisor)
	return nil
}

// managedServiceRuntimeEnvironment is shared by service launch and import
// qualification. It resolves the declared closure without writing service
// configuration, credentials, bootstrap state or model data.
func managedServiceRuntimeEnvironment(ctx context.Context, controller *Controller, manifest ResourceManifest, path string, artifact resourcedeployment.ServiceArtifact) ([]string, error) {
	env := resourceEnvForResource(controller.Root, controller.Home, manifest.Name)
	if manifest.ManagedService != nil && strings.TrimSpace(manifest.ManagedService.RunAsUser) != "" {
		env = valuehelpers.SetEnv(env, "VROOLI_MANAGED_SERVICE_RUN_AS_USER", strings.TrimSpace(manifest.ManagedService.RunAsUser))
	}
	// Expose only the verified artifact directory to the supervised process.
	// Resources such as Ollama ship a native executable alongside runtime
	// libraries; the manifest may point the service at that sibling directory
	// without discovering arbitrary host binaries.
	env = valuehelpers.SetEnv(env, "VROOLI_MANAGED_SERVICE_ARTIFACT", path)
	artifactRoot := filepath.Dir(path)
	if strings.EqualFold(strings.TrimSpace(artifact.Layout), "dir") {
		artifactRoot = path
	}
	env = valuehelpers.SetEnv(env, "RESOURCE_ARTIFACT_DIR", artifactRoot)
	acquisitionEnv, err := acquisitionTargetRuntimeEnv(ctx, manifest, artifactRoot)
	if err != nil {
		return nil, err
	}
	for key, value := range acquisitionEnv {
		env = valuehelpers.SetEnv(env, key, value)
	}
	for _, port := range manifest.Ports {
		if port.Host > 0 {
			env = valuehelpers.SetEnv(env, resourcedeployment.ServicePortEnvironmentName(port.Name), fmt.Sprintf("%d", port.Host))
		}
	}
	for key, value := range manifest.ManagedService.Environment {
		env = valuehelpers.SetEnv(env, key, value)
	}
	env = renderManagedServiceEnvironment(env, manifest.ManagedService.Environment)
	if strings.TrimSpace(manifest.ManagedService.EnvironmentFile) != "" {
		fileEnv, err := readManagedServiceEnvironmentFile(manifest.ManagedService.EnvironmentFile, env)
		if err != nil {
			return nil, err
		}
		for key, value := range fileEnv {
			env = valuehelpers.SetEnv(env, key, value)
		}
	}
	env, err = managedServiceBackendEnvironment(ctx, manifest, env)
	if err != nil {
		return nil, err
	}
	return env, nil
}

func (d managedServiceDriver) startPrivateAt(ctx context.Context, controller *Controller, manifest ResourceManifest, supervisor *ManagedServiceSupervisor, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if manifest.Name == openRGBResourceName {
		return ErrOpenRGBPreExecWitnessUnavailable
	}
	if err := verifyManagedServiceDataGroupAvailability(ctx, manifest); err != nil {
		return err
	}
	if err := d.verifyArtifactAt(ctx, controller, manifest, path); err != nil {
		return err
	}
	artifact, err := managedServiceArtifactForLaunch(ctx, controller, manifest, path)
	if err != nil {
		return err
	}
	env, err := managedServiceRuntimeEnvironment(ctx, controller, manifest, path, artifact)
	if err != nil {
		return err
	}
	if err := ensureManagedServiceRunAsUser(ctx, manifest, env, path); err != nil {
		return err
	}
	if err := writeManagedServiceConfig(controller, manifest, env); err != nil {
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
	if err := runManagedServiceBootstrap(ctx, controller, manifest, path, env); err != nil {
		cleanupCredentials()
		return err
	}
	arguments := renderManagedServiceValues(manifest.ManagedService.ArgumentsFor(runtime.GOOS, runtime.GOARCH), env)
	backendArguments, err := managedServiceBackendArguments(ctx, manifest)
	if err != nil {
		cleanupCredentials()
		return err
	}
	arguments = append(arguments, renderManagedServiceValues(backendArguments, env)...)
	if manifest.ManagedService.UpdatePolicy != nil && runtime.GOOS == "linux" {
		supervisor.verifyPlacement = func(pid int) error {
			scope, err := platform.ProcessScope(pid)
			if err != nil {
				return err
			}
			if filepath.Base(scope) != resourceServiceScopeName(manifest.Name, "service")+".scope" || filepath.Base(filepath.Dir(scope)) != platform.ServicesSlice {
				return fmt.Errorf("process is outside the enrolled resource service scope")
			}
			return nil
		}
	}
	_, err = supervisor.StartContext(ctx, path, artifact, arguments, env, filepath.Dir(path), manifest.ManagedService.ProcessLimits)
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
	if err := os.MkdirAll(root, tuning.PermPrivateDir); err != nil {
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
			field = managedUpdateCredentialDefaultField
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
		if err := os.MkdirAll(filepath.Dir(destination), tuning.PermPrivateDir); err != nil {
			cleanup()
			return nil, fmt.Errorf("create managed-service credential directory: %w", err)
		}
		if err := config.WriteOwnedFileAtomic(destination, []byte(value+"\n"), tuning.PermSecret); err != nil {
			cleanup()
			return nil, fmt.Errorf("install managed-service credential file: %w", err)
		}
		if err := securestore.RestrictCredentialFile(destination); err != nil {
			cleanup()
			return nil, fmt.Errorf("restrict managed-service credential file: %w", err)
		}
		paths = append(paths, destination)
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

func (d managedServiceDriver) verifyArtifactAt(ctx context.Context, controller *Controller, manifest ResourceManifest, path string) error {
	artifact, err := managedServiceArtifactForLaunch(ctx, controller, manifest, path)
	if err != nil {
		return err
	}
	if strings.EqualFold(artifact.Verification, "host-tool") {
		choice, selected, err := controller.ManagedServiceSource(ctx, manifest.Name)
		if err != nil {
			return err
		}
		if !selected || choice.Mode != "host" || choice.Executable != path {
			return fmt.Errorf("resource %s adopted host executable identity is not selected", manifest.Name)
		}
		actual, err := managedFileSHA256(path)
		if err != nil {
			return err
		}
		if actual != choice.SHA256 {
			return fmt.Errorf("resource %s adopted executable changed after source selection", manifest.Name)
		}
		return nil
	}
	return artifact.VerifyFileContext(ctx, path)
}

// runDiscovered accepts only an explicit executable candidate, verifies it
// against the manifest digest and version, then launches it under Vrooli's
// supervisor with Vrooli-owned configuration and state. It never discovers or
// adopts a running endpoint.
func (d managedServiceDriver) runDiscovered(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, action string, args []string) error {
	if manifest.Name == openRGBResourceName && (action == brokerTransportStart || action == brokerTransportRestart) {
		return ErrOpenRGBPreExecWitnessUnavailable
	}
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
	if action == brokerTransportRestart {
		stopCtx, cancel := managedServiceStopContext(ctx, manifest)
		err = stopManagedService(stopCtx, manifest, supervisor)
		cancel()
		if err != nil {
			return err
		}
		running = false
	}
	if !running {
		if err := d.verifyArtifactAt(ctx, controller, manifest, path); err != nil {
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
		if info.IsDir() || info.Mode()&tuning.PermExecuteMask == 0 {
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
	output, err := shell.NewCommandContext(ctx, path, "--version").CombinedOutput()
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

func writeManagedServiceConfig(controller *Controller, manifest ResourceManifest, env []string) error {
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
	if manifest.Name == "openrgb" {
		home := values["HOME"]
		if strings.TrimSpace(home) == "" {
			resolvedHome, homeErr := os.UserHomeDir()
			if homeErr != nil {
				return fmt.Errorf("resolve home directory for OpenRGB config isolation: %w", homeErr)
			}
			home = resolvedHome
		}
		if err := rejectOpenRGBAmbientConfigRoot(root, home); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), tuning.PermPrivateDir); err != nil {
		return err
	}
	if err := ensureManagedServiceConfigPath(root, path); err != nil {
		return err
	}
	resourceRoot := ""
	if controller != nil {
		resourceRoot = filepath.Join(controller.Root, "resources", manifest.Name)
	}
	content, err := managedServiceConfigContent(resourceRoot, config)
	if err != nil {
		return err
	}
	content = renderManagedServiceValue(content, values)
	if manifest.Name == "openrgb" {
		selected := []string(nil)
		if controller != nil {
			selected, _, err = controller.selectedOpenRGBDetectors(manifest)
			if err != nil {
				return fmt.Errorf("load machine-local OpenRGB detector selection: %w", err)
			}
		}
		content, err = mergeOpenRGBConfigWithSelection(path, content, selected)
		if err != nil {
			return err
		}
	}
	return atomicWriteManagedServiceConfig(path, []byte(content), tuning.PermSecret)
}

func ensureManagedServiceConfigPath(root, path string) error {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve managed-service config root: %w", err)
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve managed-service config path: %w", err)
	}
	if rel, err := filepath.Rel(rootAbs, pathAbs); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("managed-service config path escapes resource config root")
	}
	resolvedRoot, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return fmt.Errorf("resolve managed-service config root: %w", err)
	}
	resolvedParent, err := filepath.EvalSymlinks(filepath.Dir(pathAbs))
	if err != nil {
		return fmt.Errorf("resolve managed-service config directory: %w", err)
	}
	if rel, err := filepath.Rel(resolvedRoot, resolvedParent); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("managed-service config directory resolves outside resource config root")
	}
	if info, err := os.Lstat(pathAbs); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("managed-service config destination must be a regular file")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect managed-service config destination: %w", err)
	}
	return nil
}

func rejectOpenRGBAmbientConfigRoot(root, home string) error {
	resourceRoot, err := resolvedPathWithExistingPrefix(root)
	if err != nil {
		return fmt.Errorf("resolve OpenRGB resource config root: %w", err)
	}
	ambientRoot, err := resolvedPathWithExistingPrefix(filepath.Join(home, ".config"))
	if err != nil {
		return fmt.Errorf("resolve default OpenRGB config root: %w", err)
	}
	if filepath.Clean(resourceRoot) == filepath.Clean(ambientRoot) {
		return fmt.Errorf("OpenRGB resource config root aliases the ambient ~/.config directory")
	}
	return nil
}

func resolvedPathWithExistingPrefix(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	missing := []string{}
	for candidate := filepath.Clean(abs); ; candidate = filepath.Dir(candidate) {
		resolved, err := filepath.EvalSymlinks(candidate)
		if err == nil {
			for i := len(missing) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, missing[i])
			}
			return filepath.Clean(resolved), nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(candidate)
		if parent == candidate {
			return "", err
		}
		missing = append(missing, filepath.Base(candidate))
	}
}

// mergeOpenRGBConfig enforces the pinned static detector map and suppresses
// user-configured dynamic HID registrations while retaining other settings.
// A malformed existing config is left untouched for recovery.
func mergeOpenRGBConfig(path, pinned string) (string, error) {
	return mergeOpenRGBConfigWithSelection(path, pinned, nil)
}

func mergeOpenRGBConfigWithSelection(path, pinned string, selected []string) (string, error) {
	current, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("read existing OpenRGB config: %w", err)
	}
	if len(current) > 8<<20 {
		return "", fmt.Errorf("existing OpenRGB config exceeds 8 MiB safety limit")
	}
	return mergeOpenRGBConfigContentWithSelection(current, pinned, selected)
}

func mergeOpenRGBConfigContent(current []byte, pinned string) (string, error) {
	return mergeOpenRGBConfigContentWithSelection(current, pinned, nil)
}

func mergeOpenRGBConfigContentWithSelection(current []byte, pinned string, selected []string) (string, error) {
	var pinnedRoot map[string]json.RawMessage
	if err := json.Unmarshal([]byte(pinned), &pinnedRoot); err != nil {
		return "", fmt.Errorf("decode pinned OpenRGB config: %w", err)
	}
	pinnedDetectors, ok := pinnedRoot["Detectors"]
	if !ok || len(pinnedDetectors) == 0 || string(pinnedDetectors) == "null" {
		return "", fmt.Errorf("pinned OpenRGB config has no detector policy")
	}
	var pinnedDetectorSettings map[string]json.RawMessage
	if err := json.Unmarshal(pinnedDetectors, &pinnedDetectorSettings); err != nil || pinnedDetectorSettings == nil {
		return "", fmt.Errorf("pinned OpenRGB config has invalid detector settings")
	}
	pinnedDetectorMap, ok := pinnedDetectorSettings["detectors"]
	if !ok || len(pinnedDetectorMap) == 0 || string(pinnedDetectorMap) == "null" {
		return "", fmt.Errorf("pinned OpenRGB config has no detector map")
	}
	var detectorPolicy map[string]bool
	if err := json.Unmarshal(pinnedDetectorMap, &detectorPolicy); err != nil || len(detectorPolicy) == 0 {
		return "", fmt.Errorf("pinned OpenRGB config has invalid detector map")
	}
	detectorPolicy = maps.Clone(detectorPolicy)
	for _, name := range selected {
		if _, exists := detectorPolicy[name]; !exists {
			return "", fmt.Errorf("selected OpenRGB detector %q is absent from the pinned catalog", name)
		}
		detectorPolicy[name] = true
	}
	selectedDetectorMap, err := json.Marshal(detectorPolicy)
	if err != nil {
		return "", fmt.Errorf("encode selected OpenRGB detector map: %w", err)
	}
	pinnedDetectorMap = selectedDetectorMap

	merged := make(map[string]json.RawMessage)
	if len(current) > 0 {
		if len(current) > 8<<20 {
			return "", fmt.Errorf("existing OpenRGB config exceeds 8 MiB safety limit")
		}
		if err := json.Unmarshal(current, &merged); err != nil || merged == nil {
			return "", fmt.Errorf("existing OpenRGB config is not a JSON object; refusing to replace it")
		}
	}
	detectorSettings := pinnedDetectorSettings
	if existing, ok := merged["Detectors"]; ok {
		detectorSettings = make(map[string]json.RawMessage)
		if err := json.Unmarshal(existing, &detectorSettings); err != nil || detectorSettings == nil {
			return "", fmt.Errorf("existing OpenRGB detector settings are not a JSON object; refusing to replace them")
		}
	}
	detectorSettings["detectors"] = pinnedDetectorMap
	mergedDetectors, err := json.Marshal(detectorSettings)
	if err != nil {
		return "", fmt.Errorf("encode merged OpenRGB detector settings: %w", err)
	}
	merged["Detectors"] = mergedDetectors
	for _, key := range []string{"QMKOpenRGBDevices", "QMKVialRGBDevices"} {
		settingsJSON, exists := merged[key]
		var settings map[string]json.RawMessage
		if exists {
			if err := json.Unmarshal(settingsJSON, &settings); err != nil || settings == nil {
				return "", fmt.Errorf("existing OpenRGB dynamic detector settings %q are not a JSON object; refusing to replace them", key)
			}
		} else {
			settings = make(map[string]json.RawMessage)
		}
		// OpenRGB 1.1 processes these manually configured dynamic HID
		// registrations outside Detectors.detectors. Preserve their tuning
		// settings, but prevent them from expanding the startup allowlist.
		settings["devices"] = json.RawMessage("[]")
		settingsJSON, err := json.Marshal(settings)
		if err != nil {
			return "", fmt.Errorf("encode OpenRGB dynamic detector settings %q: %w", key, err)
		}
		merged[key] = settingsJSON
	}
	if err := freezeOpenRGBLocalSettings(merged); err != nil {
		return "", err
	}
	data, err := marshalOpenRGBSettings(merged)
	if err != nil {
		return "", fmt.Errorf("encode merged OpenRGB config: %w", err)
	}
	return string(data) + "\n", nil
}

func atomicWriteManagedServiceConfig(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".managed-service-config-*.tmp")
	if err != nil {
		return fmt.Errorf("create managed-service config temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set managed-service config permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write managed-service config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync managed-service config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close managed-service config: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("atomically install managed-service config: %w", err)
	}
	return nil
}

func managedServiceConfigContent(resourceRoot string, config *resourcedeployment.ServiceConfig) (string, error) {
	if config == nil {
		return "", nil
	}
	content := config.Content
	if config.ContentFile != "" {
		if resourceRoot == "" {
			return "", fmt.Errorf("managed-service config content_file requires a resource root")
		}
		base, err := filepath.EvalSymlinks(resourceRoot)
		if err != nil {
			return "", fmt.Errorf("resolve managed-service config asset root: %w", err)
		}
		asset := filepath.Join(base, filepath.FromSlash(config.ContentFile))
		resolved, err := filepath.EvalSymlinks(asset)
		if err != nil {
			return "", fmt.Errorf("resolve managed-service config asset: %w", err)
		}
		rel, err := filepath.Rel(base, resolved)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("managed-service config asset escapes resource root")
		}
		data, err := os.ReadFile(resolved)
		if err != nil {
			return "", fmt.Errorf("read managed-service config asset: %w", err)
		}
		content = string(data)
	}
	if strings.TrimSpace(content) == "" || strings.ContainsRune(content, '\x00') {
		return "", fmt.Errorf("managed-service config source is empty or contains NUL")
	}
	return content, nil
}

func runManagedServiceBootstrap(ctx context.Context, controller *Controller, manifest ResourceManifest, artifactPath string, env []string) error {
	if manifest.ManagedService == nil || manifest.ManagedService.Bootstrap == nil {
		return nil
	}
	artifact, err := managedServiceArtifactForLaunch(ctx, controller, manifest, artifactPath)
	if err != nil {
		return err
	}
	bootstrap := manifest.ManagedService.Bootstrap
	if err := bootstrap.Validate(); err != nil {
		return err
	}
	// The staged tree differs per target, so the bootstrap tool and its
	// arguments are resolved from the manifest rather than branched on here.
	resolved := bootstrap.ForPlatform(runtime.GOOS, runtime.GOARCH)
	bootstrap = &resolved
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
		if err := os.MkdirAll(filepath.Dir(passwordPath), tuning.PermPrivateDir); err != nil {
			return fmt.Errorf("create managed-service bootstrap secret directory: %w", err)
		}
		if err := os.WriteFile(passwordPath, []byte(password+"\n"), tuning.PermSecret); err != nil {
			return fmt.Errorf("write managed-service bootstrap secret: %w", err)
		}
		if name := strings.TrimSpace(manifest.ManagedService.RunAsUser); name != "" && os.Geteuid() == 0 {
			account, lookupErr := user.Lookup(name)
			if lookupErr != nil {
				return fmt.Errorf("look up managed-service user %q: %w", name, lookupErr)
			}
			uid, uidErr := strconv.Atoi(account.Uid)
			gid, gidErr := strconv.Atoi(account.Gid)
			if uidErr != nil || gidErr != nil {
				return fmt.Errorf("parse managed-service user %q identity", name)
			}
			if err := os.Chown(passwordPath, uid, gid); err != nil {
				return fmt.Errorf("assign managed-service bootstrap secret to %s: %w", name, err)
			}
		}
		defer os.Remove(passwordPath)
	}
	executable := filepath.Join(artifactRoot, filepath.FromSlash(bootstrap.Executable))
	arguments := renderManagedServiceValues(bootstrap.Arguments, bootstrapEnv)
	command := shell.NewCommandContext(ctx, executable, arguments...)
	if userName := strings.TrimSpace(manifest.ManagedService.RunAsUser); userName != "" && os.Geteuid() == 0 {
		runAsArgs := managedServiceRunAsArgs("runuser", userName, executable, arguments, bootstrapEnv)
		command = shell.NewCommandContext(ctx, "runuser", runAsArgs...)
	}
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

func ensureManagedServiceRunAsUser(ctx context.Context, manifest ResourceManifest, env []string, artifactPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if manifest.ManagedService == nil || os.Geteuid() != 0 {
		return nil
	}
	name := strings.TrimSpace(manifest.ManagedService.RunAsUser)
	if name == "" {
		return nil
	}
	if _, err := user.Lookup(name); err != nil {
		if _, unknown := err.(user.UnknownUserError); !unknown {
			return fmt.Errorf("look up managed-service user %q: %w", name, err)
		}
		cmd := shell.NewCommandContext(ctx, "useradd", "--system", "--home-dir", "/nonexistent", "--shell", "/usr/sbin/nologin", name)
		if output, runErr := cmd.CombinedOutput(); runErr != nil {
			return fmt.Errorf("create managed-service user %q: %w: %s", name, runErr, strings.TrimSpace(string(output)))
		}
	}
	values := managedServiceEnvValues(env)
	for _, path := range []string{artifactPath, values["RESOURCE_DATA_DIR"], values["RESOURCE_CONFIG_DIR"], values["RESOURCE_STATE_DIR"], values["RESOURCE_LOG_DIR"]} {
		if strings.TrimSpace(path) == "" {
			continue
		}
		if err := ensureManagedServiceTraversal(path); err != nil {
			return fmt.Errorf("prepare managed-service path %s for %s: %w", path, name, err)
		}
		uid, gid, err := lookupUserIDs(name)
		if err != nil {
			return err
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return fmt.Errorf("inspect managed-service path %s: %w", path, err)
		}
		if err := filepath.Walk(path, func(current string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			return os.Lchown(current, uid, gid)
		}); err != nil {
			return fmt.Errorf("assign managed-service path %s to %s: %w", path, name, err)
		}
	}
	return nil
}

// A root-run control plane commonly keeps its runtime home below /root, whose
// default 0700 mode prevents a declared non-root managed service from reaching
// its own artifact and state. Grant search-only access on the ancestor chain;
// the managed-service tree itself remains owned and permissioned separately.
func ensureManagedServiceTraversal(path string) error {
	current := filepath.Clean(path)
	for {
		info, err := os.Stat(current)
		if err != nil {
			if os.IsNotExist(err) {
				current = filepath.Dir(current)
				if current == filepath.Dir(current) {
					return nil
				}
				continue
			}
			return err
		}
		if info.IsDir() && info.Mode().Perm()&0o001 == 0 {
			if err := os.Chmod(current, info.Mode().Perm()|0o001); err != nil {
				return err
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
}

func lookupUserIDs(name string) (int, int, error) {
	entry, err := user.Lookup(name)
	if err != nil {
		return 0, 0, err
	}
	uid, err := strconv.Atoi(entry.Uid)
	if err != nil {
		return 0, 0, err
	}
	gid, err := strconv.Atoi(entry.Gid)
	if err != nil {
		return 0, 0, err
	}
	return uid, gid, nil
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
				env = valuehelpers.SetEnv(env, key, rendered)
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
	request := resourcedeployment.ProviderRequest{Target: resourcedeployment.ProviderTargetControlPlane, SharedConsented: slices.Contains(args, "--shared-consent")}
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
	if err := parent.Err(); err != nil {
		return fmt.Errorf("managed-service readiness interrupted: %w", err)
	}
	if len(manifest.HealthChecks) == 0 {
		return nil
	}
	seconds := managedServiceStartTimeoutSeconds(controller, manifest)
	if seconds <= 0 {
		seconds = 60
	}
	ctx, cancel := context.WithTimeout(parent, tuning.ManagedServiceConfiguredTimeout(time.Duration(seconds)*time.Second))
	defer cancel()
	for {
		health, probeErr := controller.runResourceHealthChecks(ctx, manifest)
		interrupted := func() error {
			diagnostic := health.Message
			if probeErr != nil {
				diagnostic = probeErr.Error()
			}
			return fmt.Errorf("managed-service readiness interrupted: %w (last health check: %s)", ctx.Err(), diagnostic)
		}
		if ctx.Err() != nil {
			return interrupted()
		}
		// Serving below the declared backend is a visible degraded state,
		// not a failed readiness probe. Otherwise an explicit CPU rung can
		// serve correctly while every normal start waits until its timeout.
		if probeErr == nil && (health.Healthy || (health.Serving && health.ModeDrift && health.LivenessFailed == "")) {
			return nil
		}
		select {
		case <-ctx.Done():
			return interrupted()
		case <-time.After(tuning.LifecyclePollInterval()):
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

// managedServiceUnverifiableImageMessage names the remedy when a running
// service's on-disk image no longer matches the digest recorded at launch,
// typically because its artifact path was replaced underneath it.
func managedServiceUnverifiableImageMessage(name string) string {
	return fmt.Sprintf("running %s image differs from its launch fingerprint; restart it with `vrooli resource restart %s`", name, name)
}
