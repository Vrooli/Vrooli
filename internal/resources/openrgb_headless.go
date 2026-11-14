package resources

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/vrooli/binaryfetch"
	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/lightingruntime"
	"github.com/vrooli/vrooli/internal/network"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	resourcehealth "github.com/vrooli/vrooli/internal/resources/runtime/health"
	"github.com/vrooli/vrooli/internal/tuning"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

// ErrOpenRGBDevicePolicyUnavailable means no host owner can attest scoped
// device access for the unprivileged native launcher.
var ErrOpenRGBDevicePolicyUnavailable = errors.New("no host verifier for narrowly scoped OpenRGB device access")

// ErrOpenRGBPreExecWitnessUnavailable means static device metadata could be
// checked, but no native launch owner can prove access after final confinement
// and before the process executes. Headless enable must remain inactive.
var ErrOpenRGBPreExecWitnessUnavailable = errors.New("OpenRGB requires a qualified native launch witness of selected-device access inside final service confinement before exec")

var errOpenRGBNotReady = errors.New("OpenRGB listener or protocol is still starting")

// OpenRGBDevicePolicyCheck identifies the exact inputs the host permission
// owner must verify. PID zero asks for a prospective check of the current
// launch identity. A positive PID also requires live access verification for
// that exact native process incarnation. The detector digest binds the policy
// to the pinned config input, independently of broader OS device access.
type OpenRGBDevicePolicyCheck struct {
	// Account is the exact enrolled native service identity; Linux retains EffectiveUID.
	Account string

	// DeviceGenerationSHA256 is overwritten by the trusted permission verifier.
	DeviceGenerationSHA256 string
	PolicyID               string
	Artifact               ManagedServiceArtifact
	DetectorSHA256         string
	EffectiveUID           int
	PID                    int
	Incarnation            string
}

// OpenRGBDevicePolicyVerifier is supplied by the control-plane host permission
// owner. It observes qualified, machine-local device policy without probing
// controllers or changing permissions. Success requires a qualified source-bound detector scope, persistent read/write
// access to selected physical nodes, and a retained witness inside final native
// confinement. Effective access includes inherited groups and ACLs.
type OpenRGBDevicePolicyVerifier func(context.Context, OpenRGBDevicePolicyCheck) error

// OpenRGBHeadless is the staged-only resource owner for the lighting runtime.
// It leaves acquisition and device permission provisioning to explicit setup.
type OpenRGBHeadless struct {
	controller *Controller
	verify     OpenRGBDevicePolicyVerifier
}

func NewOpenRGBHeadless(controller *Controller, verify OpenRGBDevicePolicyVerifier) *OpenRGBHeadless {
	return &OpenRGBHeadless{controller: controller, verify: verify}
}

var _ lightingruntime.ResourceOwner = (*OpenRGBHeadless)(nil)

type openRGBHeadlessLaunch struct {
	manifest ResourceManifest
	endpoint netip.AddrPort
	ready    ResourceHealthCheck
	policy   OpenRGBDevicePolicyCheck
	retained *openRGBFrozenUpdateRole
	settings []byte
}

type openRGBReadiness struct {
	status   func() (ManagedServiceState, bool, error)
	listener func(netip.AddrPort, int) error
	protocol func(context.Context, ResourceHealthCheck) error
}

func (a *OpenRGBHeadless) VerifyLightingSetup(ctx context.Context, selection lightingruntime.Selection) error {
	if a == nil || a.controller == nil {
		return errors.New("OpenRGB headless resource controller is unavailable")
	}
	if err := a.controller.ValidateOpenRGBNativeResourceStorage(ctx); err != nil {
		return err
	}
	return withManagedServiceLifecycleLock(ctx, openRGBResourceName, func() error {
		release, err := acquireResourceArtifactOwnerLock(ctx, openRGBResourceName)
		if err != nil {
			return err
		}
		defer release()
		launch, err := a.inspectSetupSelection(ctx, selection, false)
		if err != nil {
			return err
		}
		if a.verify == nil {
			return ErrOpenRGBDevicePolicyUnavailable
		}
		return a.verify(ctx, launch.policy)
	})
}

func (a *OpenRGBHeadless) EnsureOpenRGB(ctx context.Context, selection lightingruntime.Selection) (lightingruntime.Backend, error) {
	if a == nil || a.controller == nil {
		return lightingruntime.Backend{}, errors.New("OpenRGB headless resource controller is unavailable")
	}
	if a.verify == nil {
		return lightingruntime.Backend{}, ErrOpenRGBDevicePolicyUnavailable
	}

	if runtime.GOOS == "linux" {
		if (runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64") || os.Geteuid() <= 0 {
			return lightingruntime.Backend{}, errors.New("headless OpenRGB requires the qualified unprivileged Linux native resource owner")
		}
	} else if err := validateOpenRGBNativeRuntimeAuthority(runtime.GOOS, runtime.GOARCH, func() (platform.ProcessSecurityContext, error) { return platform.ProcessSecurity(os.Getpid()) }); err != nil {
		return lightingruntime.Backend{}, err
	}
	return a.nativeOpenRGBLifecycle(ctx, selection, false)
}

func (a *OpenRGBHeadless) nativeOpenRGBLifecycle(ctx context.Context, selection lightingruntime.Selection, restart bool) (lightingruntime.Backend, error) {
	if err := a.controller.ValidateOpenRGBNativeResourceStorage(ctx); err != nil {
		return lightingruntime.Backend{}, err
	}
	var backend lightingruntime.Backend
	err := withManagedServiceLifecycleLock(ctx, openRGBResourceName, func() error {
		var err error
		backend, err = a.nativeOpenRGBLocked(ctx, selection, restart)
		return err
	})
	return backend, err
}
func (a *OpenRGBHeadless) nativeOpenRGBLocked(ctx context.Context, selection lightingruntime.Selection, restart bool) (lightingruntime.Backend, error) {
	if err := ensureManagedUpdateLifecycleAvailable(openRGBResourceName, brokerTransportStart); err != nil {
		return lightingruntime.Backend{}, err
	}
	release, err := acquireResourceArtifactOwnerLock(ctx, openRGBResourceName)
	if err != nil {
		return lightingruntime.Backend{}, err
	}
	defer release()
	launch, err := a.inspectSelection(ctx, selection)
	if err != nil {
		return lightingruntime.Backend{}, err
	}
	if err := a.verify(ctx, launch.policy); err != nil {
		return lightingruntime.Backend{}, fmt.Errorf("verify OpenRGB device policy %q before launch: %w", selection.DevicePolicyID, err)
	}
	stage, stageErr := a.controller.verifyOpenRGBRoleArtifact(ctx, launch.policy.Artifact)
	if stageErr != nil {
		return lightingruntime.Backend{}, errors.Join(ErrOpenRGBPreExecWitnessUnavailable, fmt.Errorf("qualified sealed OpenRGB 1.1 envelope stage is required: %w", stageErr))
	}
	if e := verifyOpenRGBReleaseRuntimePrerequisites(ctx, stage.appDir); e != nil {
		return lightingruntime.Backend{}, e
	}

	if restart {
		if err := a.controller.openRGBPublicOwners().quiesce(ctx); err != nil {
			return lightingruntime.Backend{}, err
		}
		if err := a.retireOpenRGBBackendLocked(ctx, launch); err != nil {
			return lightingruntime.Backend{}, err
		}
	}

	return a.ensureOwned(ctx, launch)
}

func (a *OpenRGBHeadless) retireOpenRGBBackendLocked(ctx context.Context, launch openRGBHeadlessLaunch) error {
	return completeOpenRGBDisableCleanup(
		func() error {
			store, err := newOpenRGBInitialStore()
			if err != nil {
				return err
			}
			id, err := store.current()
			if os.IsNotExist(err) {
				return nil
			}
			if err != nil {
				return err
			}
			attempt, err := store.read(id)
			if err != nil {
				return err
			}
			return retireOpenRGBInitial(ctx, store, attempt)
		},
		func() error {
			supervisor, _, err := managedServiceSupervisorFor(openRGBResourceName)
			if err != nil {
				return err
			}
			stopCtx, cancel := managedServiceStopContext(ctx, launch.manifest)
			defer cancel()
			return stopManagedService(stopCtx, launch.manifest, supervisor)
		})
}

func (a *OpenRGBHeadless) inspectSelection(ctx context.Context, selection lightingruntime.Selection) (openRGBHeadlessLaunch, error) {
	if launch, found, err := a.inspectRetainedOpenRGBSelection(ctx, selection); found || err != nil {
		return launch, err
	}
	return a.inspectSetupSelection(ctx, selection, true)
}

func (a *OpenRGBHeadless) inspectSetupSelection(ctx context.Context, selection lightingruntime.Selection, requireEnabled bool) (openRGBHeadlessLaunch, error) {
	manifest, err := a.controller.managedServiceArtifactManifest(openRGBResourceName)
	if err != nil {
		return openRGBHeadlessLaunch{}, err
	}
	endpoint, ready, err := openRGBHeadlessEndpoint(manifest)
	if err != nil {
		return openRGBHeadlessLaunch{}, err
	}
	state, err := a.controller.operatorStateService().Load(ctx)
	if err != nil {
		return openRGBHeadlessLaunch{}, err
	}
	choice := state.Resources[openRGBResourceName]
	if choice.Source == nil || requireEnabled && (choice.Enabled == nil || !*choice.Enabled) {
		return openRGBHeadlessLaunch{}, errors.New("OpenRGB requires explicit resource enablement and source selection")
	}
	var path string
	var resolved resourcedeployment.ServiceArtifact
	if choice.Source.Mode == "host" {
		artifact, inspectErr := a.controller.InspectManagedServiceArtifact(ctx, openRGBResourceName)
		if inspectErr != nil {
			return openRGBHeadlessLaunch{}, fmt.Errorf("verify adopted OpenRGB executable: %w", inspectErr)
		}
		path = artifact.Path
		resolved = manifest.ManagedService.Artifact
		resolved.Version, resolved.SHA256 = artifact.Version, artifact.SHA256
	} else if choice.Source.Mode == "managed" {
		path, resolved, err = a.stagedArtifact(ctx, manifest)
	} else {
		err = errors.New("OpenRGB source must be explicitly selected as managed or host")
	}
	if err != nil {
		return openRGBHeadlessLaunch{}, fmt.Errorf("verify staged OpenRGB artifact: %w", err)
	}
	artifact := ManagedServiceArtifact{Resource: manifest.Name, Path: path, Version: resolved.Version, SHA256: resolved.SHA256}
	if artifact.Version != selection.Version || artifact.SHA256 != selection.SHA256 || strings.TrimSpace(selection.DevicePolicyID) == "" {
		return openRGBHeadlessLaunch{}, errors.New("verified OpenRGB artifact or device policy differs from the selected headless setup")
	}
	policyRecord, err := a.controller.loadOpenRGBDevicePolicy(manifest)
	if err != nil {
		return openRGBHeadlessLaunch{}, fmt.Errorf("load selected machine-local OpenRGB device policy: %w", err)
	}
	if policyRecord == nil {
		return openRGBHeadlessLaunch{}, errors.New("explicit machine-local OpenRGB device policy is required")
	}
	policyID, err := policyRecord.ID()
	policyArtifact := policyRecord.Policy.ArtifactSHA256
	account := ""
	if policyRecord.NativePolicy != nil {
		policyArtifact = policyRecord.NativePolicy.ArtifactSHA256
		account = policyRecord.NativePolicy.Account
	}
	if err != nil || policyID != selection.DevicePolicyID || policyArtifact != artifact.SHA256 {
		return openRGBHeadlessLaunch{}, errors.Join(errors.New("machine-local OpenRGB device policy differs from the selected setup"), err)
	}
	config := *manifest.ManagedService.Config
	if config.ContentFile != "" {
		content, err := readManagedUpdateConfigFile(filepath.Join(a.controller.Root, "resources", openRGBResourceName), config.ContentFile)
		if err != nil {
			return openRGBHeadlessLaunch{}, fmt.Errorf("freeze OpenRGB detector config: %w", err)
		}
		config.Content, config.ContentFile = string(content), ""
	}
	if len(config.Content) > managedUpdateConfigMaxBytes || strings.ContainsRune(config.Content, '$') {
		return openRGBHeadlessLaunch{}, errors.New("OpenRGB detector policy must be a bounded fixed config input")
	}
	// Pass frozen launch inputs to the driver. A staged-only launch has no
	// acquisition-time overrides or hardware-dependent target resolution.
	service := *manifest.ManagedService
	service.Config, service.Artifact, service.Acquisition = &config, resolved, nil
	manifest.ManagedService = &service
	effectiveUID := os.Geteuid()
	if account != "" {
		effectiveUID = 0
	}
	return openRGBHeadlessLaunch{manifest: manifest, endpoint: endpoint, ready: ready, policy: OpenRGBDevicePolicyCheck{
		Account: account, PolicyID: selection.DevicePolicyID, Artifact: artifact, DetectorSHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(config.Content))), EffectiveUID: effectiveUID,
	}}, nil
}

func (a *OpenRGBHeadless) stagedArtifact(ctx context.Context, manifest ResourceManifest) (string, resourcedeployment.ServiceArtifact, error) {
	var empty resourcedeployment.ServiceArtifact
	acquisition := manifest.ManagedService.Acquisition
	for _, candidate := range acquisition.Targets {
		for fact := range candidate.When {
			if fact != "os" && fact != "arch" {
				return "", empty, fmt.Errorf("headless OpenRGB does not resolve hardware-dependent artifact predicate %q", fact)
			}
		}
	}
	facts := binaryfetch.Facts{"os": runtime.GOOS, "arch": runtime.GOARCH}
	target, err := acquisition.Resolve(facts)
	if err != nil {
		return "", empty, err
	}
	if target.Executable != "" || len(target.RuntimeEnv) != 0 {
		return "", empty, errors.New("headless OpenRGB requires a staged artifact without acquisition-time launch overrides")
	}
	artifact, err := managedServiceArtifactForTarget(manifest, target)
	if err != nil {
		return "", empty, err
	}
	path, err := managedServiceArtifactPath(a.controller, manifest)
	if err != nil {
		return "", empty, err
	}
	if err := checkFactDrift(path, manifest.Name, facts, target); err != nil {
		return "", empty, err
	}
	if err := artifact.VerifyFileContext(ctx, path); err != nil {
		return "", empty, err
	}
	launchPath, err := artifact.LaunchPath(path)
	if err != nil {
		return "", empty, err
	}
	info, err := os.Stat(launchPath)
	if err != nil {
		return "", empty, err
	}
	if !info.Mode().IsRegular() || runtime.GOOS != "windows" && info.Mode().Perm()&tuning.PermExecuteMask == 0 {
		return "", empty, errors.New("staged OpenRGB entry is not a regular executable")
	}
	return path, artifact, nil
}

func openRGBHeadlessEndpoint(manifest ResourceManifest) (netip.AddrPort, ResourceHealthCheck, error) {
	var endpoint netip.AddrPort
	var ready ResourceHealthCheck
	service := manifest.ManagedService
	if err := ensureSupportedPlatform(manifest); err != nil {
		return endpoint, ready, err
	}
	mode, err := managedServiceProvider(manifest, nil)
	if err != nil || mode != resourcedeployment.ProviderManagedPrivate || service == nil || service.Config == nil || service.Config.Path != "OpenRGB/OpenRGB.json" ||
		service.RunAsUser != "" || service.EnvironmentFile != "" || service.Bootstrap != nil || len(service.CredentialFiles) != 0 || len(manifest.Companions) != 0 || manifest.Lifecycle.ExclusiveGroup != "" {
		return endpoint, ready, errors.New("OpenRGB manifest has no supported private headless launch contract")
	}
	portKey := "${RESOURCE_PORT_SDK}"
	const loopback = "127.0.0.1"
	args := []string{"--server", "--server-host", loopback, "--server-port", portKey, "--config", "${RESOURCE_CONFIG_DIR}/OpenRGB"}
	if len(manifest.Ports) != 1 || manifest.Ports[0].Name != "sdk" || manifest.Ports[0].Host < 1 || manifest.Ports[0].Host > 65535 ||
		!slices.Equal(service.ArgumentsFor(runtime.GOOS, runtime.GOARCH), args) || !openRGBConstructorEnvironmentIsolated(runtime.GOOS, "${RESOURCE_CONFIG_DIR}", service.Environment) {
		return endpoint, ready, errors.New("OpenRGB manifest has no qualified isolated loopback SDK binding")
	}
	endpoint = netip.AddrPortFrom(netip.MustParseAddr(loopback), uint16(manifest.Ports[0].Host))
	for _, check := range manifest.HealthChecks {
		if check.Kind == manifestpkg.HealthCheckReadiness && check.Type == manifestpkg.HealthCheckOpenRGBV6 && check.Target == loopback+":"+portKey {
			check.Target = endpoint.String()
			return endpoint, check, nil
		}
	}
	return endpoint, ready, errors.New("OpenRGB manifest has no SDK v6 readiness check on the selected endpoint")
}

func (a *OpenRGBHeadless) ensureOwned(ctx context.Context, launch openRGBHeadlessLaunch) (lightingruntime.Backend, error) {
	supervisor, _, err := managedServiceSupervisorFor(openRGBResourceName)
	if err != nil {
		return lightingruntime.Backend{}, err
	}
	readiness := openRGBReadiness{status: supervisor.Status, listener: observeOpenRGBHeadlessListener, protocol: probeOpenRGBHeadlessSDK}
	state, running, err := readiness.status()
	if err != nil {
		return lightingruntime.Backend{}, err
	}
	if running {
		backend, err := a.observeReady(ctx, launch, readiness, state)
		if err == nil {
			return backend, nil
		}
		if !errors.Is(err, ErrOpenRGBDeviceGenerationChanged) {
			return lightingruntime.Backend{}, err
		}
		return refreshOpenRGBDeviceGeneration(ctx, err, a.controller.openRGBPublicOwners().quiesce, func(ctx context.Context) error { return a.retireOpenRGBBackendLocked(ctx, launch) }, func(ctx context.Context) (lightingruntime.Backend, error) {
			return a.startOpenRGBInitialLocked(ctx, launch, supervisor)
		})
	}
	return a.startOpenRGBInitialLocked(ctx, launch, supervisor)
}
func (a *OpenRGBHeadless) startOpenRGBInitialLocked(ctx context.Context, launch openRGBHeadlessLaunch, supervisor *ManagedServiceSupervisor) (lightingruntime.Backend, error) {
	var backend lightingruntime.Backend
	err := a.controller.withResourceCapacity(ctx, openRGBResourceName, brokerTransportStart, func(ctx context.Context) error {
		var ensureErr error
		backend, ensureErr = a.ensureOpenRGBInitial(ctx, launch, supervisor)
		return ensureErr
	})
	return backend, err
}

func (a *OpenRGBHeadless) waitReady(ctx context.Context, launch openRGBHeadlessLaunch, readiness openRGBReadiness) (lightingruntime.Backend, error) {
	seconds := managedServiceStartTimeoutSeconds(a.controller, launch.manifest)
	if seconds <= 0 {
		seconds = 60
	}
	ctx, cancel := context.WithTimeout(ctx, tuning.ManagedServiceConfiguredTimeout(time.Duration(seconds)*time.Second))
	defer cancel()
	for {
		state, running, err := readiness.status()
		if err != nil {
			return lightingruntime.Backend{}, err
		}
		if !running {
			return lightingruntime.Backend{}, errors.New("owned OpenRGB process exited before readiness")
		}
		backend, err := a.observeReady(ctx, launch, readiness, state)
		if err == nil {
			return backend, nil
		}
		if !errors.Is(err, errOpenRGBNotReady) {
			return lightingruntime.Backend{}, err
		}
		select {
		case <-ctx.Done():
			return lightingruntime.Backend{}, fmt.Errorf("owned OpenRGB readiness did not converge: %w", errors.Join(ctx.Err(), err))
		case <-time.After(tuning.LifecyclePollInterval()):
		}
	}
}

func (a *OpenRGBHeadless) observeReady(ctx context.Context, launch openRGBHeadlessLaunch, readiness openRGBReadiness, state ManagedServiceState) (lightingruntime.Backend, error) {
	artifact := launch.policy.Artifact
	if state.PID <= 0 || state.Incarnation == "" || state.InstanceID == "" || state.OwnershipTokenHash == "" || state.ArtifactVersion != artifact.Version || state.ArtifactSHA256 != artifact.SHA256 {
		return lightingruntime.Backend{}, errors.New("running OpenRGB process does not match the selected staged identity")
	}
	if state.ArtifactPath != artifact.Path && state.InitialLaunch == nil && state.UpdateLaunch == nil {
		if err := verifyManagedServiceRunningArtifact(ctx, state); err != nil {
			return lightingruntime.Backend{}, fmt.Errorf("verify retained OpenRGB serving artifact: %w", err)
		}
	}
	listenerPID := state.PID
	if runtime.GOOS == "windows" && (state.InitialLaunch != nil || state.UpdateLaunch != nil) {
		var e error
		listenerPID, e = a.controller.openRGBServingListenerPID(ctx, state)
		if e != nil {
			return lightingruntime.Backend{}, e
		}
	}
	if err := readiness.listener(launch.endpoint, listenerPID); err != nil {
		return lightingruntime.Backend{}, err
	}
	policy := launch.policy
	policy.Artifact.Path, policy.PID, policy.Incarnation = state.ArtifactPath, state.PID, state.Incarnation
	if err := a.verify(ctx, policy); err != nil {
		return lightingruntime.Backend{}, fmt.Errorf("verify live OpenRGB device policy %q: %w", policy.PolicyID, err)
	}
	if err := readiness.protocol(ctx, launch.ready); err != nil {
		return lightingruntime.Backend{}, err
	}
	if err := readiness.listener(launch.endpoint, listenerPID); err != nil {
		return lightingruntime.Backend{}, err
	}
	after, running, err := readiness.status()
	if err != nil {
		return lightingruntime.Backend{}, err
	}
	if !running || after.PID != state.PID || after.Incarnation != state.Incarnation || after.InstanceID != state.InstanceID || after.OwnershipTokenHash != state.OwnershipTokenHash ||
		after.ArtifactPath != state.ArtifactPath || after.ArtifactVersion != state.ArtifactVersion || after.ArtifactSHA256 != state.ArtifactSHA256 {
		return lightingruntime.Backend{}, errors.New("owned OpenRGB process changed during readiness observation")
	}
	if err := a.verify(ctx, policy); err != nil {
		return lightingruntime.Backend{}, err
	}
	return lightingruntime.Backend{Endpoint: launch.endpoint, Incarnation: state.Incarnation}, nil
}

func probeOpenRGBHeadlessSDK(ctx context.Context, ready ResourceHealthCheck) error {
	result, err := resourcehealth.RunCheck(ctx, ready, resourcehealth.Config{})
	if err != nil {
		return err
	}
	if !result.Healthy {
		return fmt.Errorf("%w: %s", errOpenRGBNotReady, result.Message)
	}
	return nil
}

func (a *OpenRGBHeadless) DisableOpenRGB(ctx context.Context) error {
	if a == nil || a.controller == nil {
		return errors.New("OpenRGB headless resource controller is unavailable")
	}

	if err := requireOpenRGBNativeOperatorAuthority(); err != nil {
		return err
	}

	if err := a.controller.ValidateOpenRGBNativeResourceStorage(ctx); err != nil {
		return err
	}
	return withOpenRGBDisableLock(ctx, func() error {
		if err := a.persistDisabled(ctx); err != nil {
			return err
		}
		return completeOpenRGBDisableCleanup(
			func() error { return a.controller.retireOpenRGBUpdatesForDisable(ctx) },
			func() error {
				store, err := newOpenRGBInitialStore()
				if err != nil {
					return err
				}
				id, err := store.current()
				if os.IsNotExist(err) {
					return nil
				}
				if err != nil {
					return err
				}
				attempt, err := store.read(id)
				if err != nil {
					return err
				}
				return retireOpenRGBInitial(ctx, store, attempt)
			},
			func() error {
				supervisor, _, err := managedServiceSupervisorFor(openRGBResourceName)
				if err != nil {
					return err
				}
				manifest, manifestErr := a.controller.managedServiceArtifactManifest(openRGBResourceName)
				if manifestErr != nil {
					return errors.Join(manifestErr, supervisor.Stop(ctx))
				}
				stopCtx, cancel := managedServiceStopContext(ctx, manifest)
				defer cancel()
				stopErr := stopManagedService(stopCtx, manifest, supervisor)
				_, running, statusErr := supervisor.Status()
				return errors.Join(stopErr, confirmOpenRGBStopped(running, statusErr, manifest, network.CaptureTCPListenerPorts()))
			})
	})
}

// Explicit opt-out owns retirement of active updates. It takes the canonical
// lock without the ordinary-start availability interlock it must reconcile.
func withOpenRGBDisableLock(ctx context.Context, fn func() error) error {
	release, err := acquireManagedServiceLifecycleLock(ctx, openRGBResourceName)
	if err != nil {
		return err
	}
	defer release()
	if err := ctx.Err(); err != nil {
		return err
	}
	return fn()
}

// One failed retained owner must not leave another writer untouched. Absence
// is acknowledged only when every exact cleanup and final observation succeeds.
func completeOpenRGBDisableCleanup(cleanups ...func() error) error {
	var failures []error
	for _, cleanup := range cleanups {
		if err := cleanup(); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func confirmOpenRGBStopped(running bool, statusErr error, manifest ResourceManifest, listeners network.TCPListenerSnapshot) error {
	if running || statusErr != nil {
		return errors.Join(errors.New("OpenRGB managed process absence could not be confirmed"), statusErr)
	}
	if len(manifest.Ports) != 1 || manifest.Ports[0].Name != "sdk" || manifest.Ports[0].Host < 1 || manifest.Ports[0].Host > 65535 {
		return errors.New("OpenRGB SDK port is unavailable for stopped observation")
	}
	port := manifest.Ports[0].Host
	state := listeners.Listening(port)
	if !state.Known || state.Listening {
		return fmt.Errorf("OpenRGB listener absence could not be confirmed on port %d", port)
	}
	return nil
}

func (a *OpenRGBHeadless) persistDisabled(ctx context.Context) error {
	state := a.controller.operatorStateService()
	patch, err := json.Marshal(map[string]any{"resources": map[string]any{openRGBResourceName: map[string]bool{"enabled": false}}})
	if err != nil {
		return err
	}
	if _, err := state.Apply(ctx, patch); err != nil {
		return err
	}
	path, err := state.Path(ctx)
	if err != nil {
		return err
	}
	// Native operator state uses the private durable writer, including Windows
	// write-through publication. Unix still needs the directory entry confirmed
	// before the lighting owner closes its fence; a read-only Windows directory
	// handle cannot be flushed and must not replace its writer's durability.
	if err := confirmOpenRGBPublicationDurability(filepath.Dir(path)); err != nil {
		return fmt.Errorf("confirm durable OpenRGB resource disable: %w", err)
	}
	result, err := state.Load(ctx)
	if err != nil {
		return err
	}
	choice := result.Resources[openRGBResourceName].Enabled
	if choice == nil || *choice {
		return errors.New("OpenRGB resource disable was not confirmed by operator state")
	}
	return nil
}
