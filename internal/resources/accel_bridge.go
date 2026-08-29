package resources

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/accel"
	"github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/gpuaccess"
	resourcecontrol "github.com/vrooli/vrooli/internal/resources/control"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

const (
	accelBridgeManagedService = "managed-service"
)

const (
	accelBridgeExternalCli = "external-cli"
	accelBridgeNativeCli   = "native-cli"
	accelBridgeNoWorkload  = "no workload is resident, so placement cannot be read yet"
)

// This file is the only place internal/resources and internal/accel meet.
// internal/accel owns the decision, the retry loop and the operator message;
// this bridge supplies the three things only the resource layer can know: what
// the manifest declared, where the resource is running, and how to reach a
// container.

// accelSpecFor projects a manifest's accelerator declaration into the shape
// internal/accel decides on. ok is false when the resource declares no
// accelerator at all.
func accelSpecFor(manifest ResourceManifest) (accel.Spec, bool) {
	declaration := manifest.EffectiveAcceleration()
	if declaration == nil {
		return accel.Spec{}, false
	}
	backends, err := accel.ParseBackends(declaration.Backends)
	if err != nil || len(backends) == 0 {
		return accel.Spec{}, false
	}
	spec := accel.Spec{Resource: manifest.Name, Backends: backends, Require: declaration.EffectiveRequire()}
	if !spec.Accelerated() {
		return spec, false
	}
	return spec, true
}

// accelContainerProbe adapts internal/gpuaccess, which owns the docker exec,
// into the seam internal/accel accepts. This is what keeps os/exec out of
// internal/accel.
func accelContainerProbe(ctx context.Context, container string, backend accel.Backend) (accel.AccessState, string) {
	state, reason := gpuaccess.VerifyWithExec(ctx, container, string(backend), verifyContainerGPUExec)
	switch state {
	case gpuaccess.OK:
		return accel.AccessOK, reason
	case gpuaccess.Revoked:
		return accel.AccessRevoked, reason
	}
	return accel.AccessUnknown, reason
}

// accelFactSource is the injectable seam over live host accelerator sensing.
// Every accelerator read in this package goes through it, so a unit run never
// depends on whether the machine running the tests happens to have a device.
var accelFactSource accel.FactSource = accel.HostFactSource{}

// accelVerifier is the production verifier: real host facts, a real container
// probe, and the retry cadence the drivers used to implement twice.
func accelVerifier() accel.Verifier {
	return accel.Verifier{
		Facts:     accel.StaticFactSource{},
		Container: accelContainerProbe,
		Sleep:     gpuVerificationSleep,
	}
}

// accelVerifierWithFacts is accelVerifier reading live host facts. Collecting
// the snapshot is the expensive part, so callers that already hold one pass it
// instead.
func accelVerifierWithFacts() accel.Verifier {
	verifier := accelVerifier()
	verifier.Facts = accelFactSource
	return verifier
}

// resourceArtifactPrefix is the directory a resource's own executables live
// under. internal/accel uses it to attribute a compute process to a resource
// when the device-holding pid is a child of the supervised one.
func resourceArtifactPrefix(manifest ResourceManifest) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	root, err := managedServiceArtifactStoreRoot(home)
	if err != nil {
		return ""
	}
	return filepath.Join(root, manifest.Name)
}

// placementTargetFor resolves what to inspect for a running resource. It
// returns ok=false when the resource is not running, because an absent process
// has no placement to read.
func placementTargetFor(ctx context.Context, controller *Controller, manifest ResourceManifest) (accel.PlacementTarget, bool, error) {
	switch manifest.Driver {
	case accelBridgeManagedService, accelBridgeNativeCli:
		process := accel.HostProcess{
			Name:             manifest.Name,
			ExecutablePrefix: resourceArtifactPrefix(manifest),
			NoWorkloadReason: noWorkloadReasonFor(manifest),
		}
		supervisor, _, err := managedServiceSupervisorFor(manifest.Name)
		if err == nil {
			if state, running, statusErr := supervisor.Status(); statusErr == nil && running {
				process.PID = state.PID
				process.Serving = true
			}
		}
		return process, true, nil
	}
	return nil, false, nil
}

func noWorkloadReasonFor(manifest ResourceManifest) string {
	return accelBridgeNoWorkload
}

// verifyStartedPlacement is the single post-start placement check for every
// driver: container, compose service, and host process alike. Finding the thing
// to inspect differs per driver; the decision, the retry, the typed error and
// the operator message do not.
//
// A revoked device is a typed error with a repair command. Anything else
// unverifiable is a warning: the resource is serving, and refusing to start it
// because the control plane could not read a device would be worse than saying
// so.
func verifyStartedPlacement(ctx context.Context, controller *Controller, manifest ResourceManifest, warning io.Writer) error {
	placement, err := observePlacement(ctx, controller, manifest)
	if err != nil {
		var revoked *accel.AccessRevokedError
		if errors.As(err, &revoked) {
			return &Error{
				Code:      "gpu_access_revoked",
				Resource:  manifest.Name,
				Operation: "start",
				Category:  "GPU",
				Err:       revoked,
			}
		}
		return err
	}
	if placement == nil || placement.State == accel.StateOK {
		return nil
	}
	if warning == nil {
		return nil
	}
	if placement.State == accel.StateDrift {
		// The resource is serving, on a backend below the one it declared. That
		// is a state to report, not a reason to refuse the start.
		_, _ = fmt.Fprintf(warning, "warning: resource %q declared %s but is running on %s: %s\n", manifest.Name, placement.Declared, placement.Observed, placement.Reason)
		return nil
	}
	_, _ = fmt.Fprintf(warning, "warning: GPU access for resource %q is unknown: %s; resource started, but it is not verified\n", manifest.Name, placement.Reason)
	return nil
}

// observePlacement reads the backend a running resource is on. A nil placement
// with a nil error means the resource declares no accelerator, so there is
// nothing to verify.
func observePlacement(ctx context.Context, controller *Controller, manifest ResourceManifest) (*accel.Placement, error) {
	spec, ok := accelSpecFor(manifest)
	if !ok {
		return nil, nil
	}

	verifier := accelVerifierWithFacts()
	snapshot, err := verifier.Facts.Snapshot(ctx)
	if err != nil {
		return nil, fmt.Errorf("verify accelerator placement for %s: %w", manifest.Name, err)
	}
	verifier.Facts = accel.StaticFactSource{Inventory: snapshot}

	readiness, err := accel.ReadinessFromSnapshot(snapshot, spec)
	if err != nil {
		return nil, err
	}
	readiness, overrideReason := applyGPUOverride(readiness, spec)
	target, resolved, err := placementTargetFor(ctx, controller, manifest)
	if err != nil {
		return nil, err
	}
	if !resolved {
		return nil, nil
	}

	placement, err := verifier.VerifyPlacement(ctx, manifest.Name, target, readiness.Selected)
	if err != nil {
		return nil, err
	}
	// The resource may be on the backend it was *given* while still below the
	// backend it *declared*. Report the declared backend so the operator sees
	// the whole story rather than a locally consistent one.
	placement.Declared = readiness.Declared
	if placement.State == accel.StateOK && readiness.Drift {
		placement.State = accel.StateDrift
		reason := overrideReason
		if reason == "" {
			reason = fmt.Sprintf("declared %s is not reachable on this host", readiness.Declared)
		}
		placement.Reason = fmt.Sprintf("%s; %s", placement.Reason, reason)
	}
	return &placement, nil
}

// applyGPUOverride folds the VROOLI_GPU escape hatch into the selected backend.
//
// The override changes which backend a resource is *given*, never whether the
// result is *reported*. Running below the declared backend stays visible even
// when an operator asked for it, because a status surface that hides a
// deliberate downgrade is indistinguishable from one that hides an accidental
// one. The reason names the override so the operator can see it is their own
// doing.
func applyGPUOverride(readiness accel.ReadinessResult, spec accel.Spec) (accel.ReadinessResult, string) {
	switch gpuOverride() {
	case "off":
		if readiness.Selected == accel.BackendCPU {
			return readiness, ""
		}
		readiness.Selected = accel.BackendCPU
		readiness.Drift = readiness.Declared != accel.BackendCPU
		return readiness, gpuOverrideEnvVar + "=off selected the cpu backend"
	case "on":
		if len(spec.Backends) == 0 || readiness.Selected == spec.Backends[0] {
			return readiness, ""
		}
		readiness.Selected = spec.Backends[0]
		readiness.Drift = false
		return readiness, gpuOverrideEnvVar + "=on forced the declared backend regardless of the host probe"
	}
	return readiness, ""
}

// applyHealthToStatus is the single place a health verdict becomes a status
// object, so every driver reports mode drift the same way. Without it each
// driver would decide independently what a healthy resource means, so this bridge
// serving from the wrong device.
func applyHealthToStatus(status Status, health HealthResult) Status {
	healthy := health.Healthy
	serving := health.Serving
	status.Healthy = &healthy
	status.Serving = &serving
	status.DeclaredMode = health.DeclaredMode
	status.ObservedMode = health.ObservedMode
	status.ModeReason = health.ModeReason
	status.ModeDrift = health.ModeDrift
	if strings.TrimSpace(health.Message) != "" {
		status.Message = health.Message
	}
	switch {
	case health.PlacementUndetermined && healthy:
		status.StatusCode = resourcecontrol.StatusCodePlacementUndetermined
		status.Health = scenarioruntime.HealthStatusHealthy
		if status.Message == "" {
			status.Message = "healthy; placement undetermined: " + health.ModeReason
		}
	case health.ModeDrift:
		// Serving on a backend below the declared one. running stays true,
		// serving stays true, healthy is false: a consumer that restarts on
		// healthy:false alone would loop against a working resource.
		status.StatusCode = resourcecontrol.StatusCodeModeDrift
		status.Health = "degraded"
	case healthy:
		status.Health = scenarioruntime.HealthStatusHealthy
	default:
		status.Health = scenarioruntime.HealthStatusUnhealthy
	}
	return status
}

// AccelerationExplanation is the operator-facing answer to "why is this
// resource on this backend": what it declared, what the host can reach, what
// the resolver picked, and where the process actually landed — with a
// remediation command on every failing row.
type AccelerationExplanation struct {
	Resource string `json:"resource"`
	// Declared is the resource's ordered backend preference. Empty means the
	// resource declares no accelerator.
	Declared []string `json:"declared_backends,omitempty"`
	// Require is required, preferred or none.
	Require string `json:"require,omitempty"`
	// HostBackends is what the host reports it can reach.
	HostBackends []string `json:"host_backends"`
	// Facts are the accelerator facts that drove selection.
	Facts map[string]string `json:"facts"`
	// Considered is the host's verdict on each declared backend.
	Considered []accel.BackendVerdict `json:"considered,omitempty"`
	// Selected is the backend the resource will be given.
	Selected string `json:"selected,omitempty"`
	// Placement is where the running process actually landed. Nil when the
	// resource is not running or declares no accelerator.
	Placement *accel.Placement `json:"placement,omitempty"`
	// Remediation is the command that repairs the host when the declared
	// backend is unreachable. Empty when nothing needs repair.
	Remediation string `json:"remediation,omitempty"`
	// Claim is the capacity reservation the resource declares, if any.
	Claim *capacity.ResourceClaimSpec `json:"claim,omitempty"`
}

// ExplainAcceleration assembles the accelerator explanation for one resource.
func (c *Controller) ExplainAcceleration(ctx context.Context, manifest ResourceManifest) (AccelerationExplanation, error) {
	explanation := AccelerationExplanation{Resource: manifest.Name}

	snapshot, err := accelFactSource.Snapshot(ctx)
	if err != nil {
		return explanation, fmt.Errorf("read host accelerator facts: %w", err)
	}
	explanation.Facts = snapshot.AcceleratorFacts()
	for _, backend := range accel.ReachableBackends(snapshot) {
		explanation.HostBackends = append(explanation.HostBackends, string(backend))
	}
	if declaration := manifest.EffectiveAcceleration(); declaration != nil {
		explanation.Claim = declaration.Claim
	}

	spec, accelerated := accelSpecFor(manifest)
	if !accelerated {
		return explanation, nil
	}
	explanation.Require = spec.EffectiveRequire()
	for _, backend := range spec.Backends {
		explanation.Declared = append(explanation.Declared, string(backend))
	}

	readiness, readinessErr := accel.ReadinessFromSnapshot(snapshot, spec)
	explanation.Considered = readiness.Considered
	explanation.Selected = string(readiness.Selected)
	var noBackend *accel.NoBackendReadyError
	if errors.As(readinessErr, &noBackend) {
		explanation.Remediation = noBackend.Remediation
	}

	placement, err := observePlacement(ctx, c, manifest)
	if err != nil {
		return explanation, err
	}
	explanation.Placement = placement
	if placement != nil && placement.State == accel.StateDrift && explanation.Remediation == "" {
		explanation.Remediation = accel.RemediationFor(spec.Backends)
	}
	return explanation, nil
}

// gateAcceleratorReadiness runs before a resource process is launched. It is
// the mechanism that makes the fifteen-hour silent CPU session impossible: a
// resource that declared it needs an accelerator does not start on a host that
// cannot give it one, and a resource that merely prefers one starts with the
// fallback recorded rather than hidden.
//
// It never escalates privilege. When repair needs it, the typed error carries
// the command for the operator to run.
func gateAcceleratorReadiness(ctx context.Context, manifest ResourceManifest, warning io.Writer) error {
	spec, accelerated := accelSpecFor(manifest)
	if !accelerated {
		return nil
	}
	snapshot, err := accelFactSource.Snapshot(ctx)
	if err != nil {
		// A host probe that cannot run is not evidence that the accelerator is
		// absent. Let the start proceed; placement verification after start
		// reports what actually happened.
		if warning != nil {
			_, _ = fmt.Fprintf(warning, "warning: could not read accelerator readiness for %q before start: %v\n", manifest.Name, err)
		}
		return nil
	}

	readiness, err := accel.ReadinessFromSnapshot(snapshot, spec)
	if err != nil {
		var noBackend *accel.NoBackendReadyError
		if errors.As(err, &noBackend) {
			return &Error{
				Code:      "accel_unready",
				Resource:  manifest.Name,
				Operation: "start",
				Category:  "Accelerator",
				Err:       noBackend,
			}
		}
		return err
	}
	readiness, overrideReason := applyGPUOverride(readiness, spec)
	if !readiness.Drift || warning == nil {
		return nil
	}
	reason := overrideReason
	if reason == "" {
		for _, verdict := range readiness.Considered {
			if verdict.Backend == readiness.Declared && !verdict.Ready {
				reason = verdict.Reason
				break
			}
		}
	}
	_, _ = fmt.Fprintf(warning, "warning: resource %q declared %s but this host can only give it %s: %s\n", manifest.Name, readiness.Declared, readiness.Selected, reason)
	return nil
}
