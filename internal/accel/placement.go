package accel

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

const (
	placementParameterA = 3
)

// PlacementTarget names the thing whose backend is being read. It is a closed
// sum: a host process, a container, or a compose service. The evidence differs
// per target and per backend; the decision, the retry loop, the typed error and
// the operator message do not.
type PlacementTarget interface{ isPlacementTarget() }

// HostProcess is a resource that runs directly on the host: managed-service and
// native-cli. This is the target kind that had no verification at all before,
// and it is exactly where a silent CPU fallback happens.
type HostProcess struct {
	// PID is the process the control plane supervises.
	PID int
	// Serving indicates that the supervised resource has a live workload
	// surface. An accelerator host with a serving process and no resident GPU
	// row is therefore an observed CPU placement, not an absent workload.
	Serving bool
	// Name is the process name, used only in messages.
	Name string
	// ExecutablePrefix is the directory the resource's own executables live
	// under. It exists because the supervised pid is often not the pid that
	// holds the device: ollama supervises `ollama serve`, which spawns
	// `llama-server`, and only the child appears in the compute rows. A
	// compute process whose executable sits inside the resource's own artifact
	// tree belongs to that resource.
	ExecutablePrefix string
	// NoWorkloadReason is supplied by the resource layer when an empty
	// accelerator process table is expected before a workload is loaded.
	NoWorkloadReason string
}

// Container is a resource that runs as a single named container.
type Container struct{ Name string }

// ComposeService is a resource that runs as one service of a compose project.
type ComposeService struct {
	Project string
	Service string
	// Container is the resolved container name for the running service. The
	// caller resolves it because only the caller can talk to compose.
	Container string
}

func (HostProcess) isPlacementTarget()    {}
func (Container) isPlacementTarget()      {}
func (ComposeService) isPlacementTarget() {}

// Describe renders a target for an operator-facing message.
func Describe(target PlacementTarget) string {
	switch value := target.(type) {
	case HostProcess:
		if strings.TrimSpace(value.Name) != "" {
			return fmt.Sprintf("process %s (pid %d)", value.Name, value.PID)
		}
		return fmt.Sprintf("pid %d", value.PID)
	case Container:
		return fmt.Sprintf("container %q", value.Name)
	case ComposeService:
		if strings.TrimSpace(value.Container) != "" {
			return fmt.Sprintf("container %q", value.Container)
		}
		return fmt.Sprintf("compose service %s/%s", value.Project, value.Service)
	}
	return "unknown target"
}

// State is the verdict on a placement observation.
type State string

const (
	// StateOK means the observed backend equals the declared backend.
	StateOK State = "ok"
	// StateDrift means the resource is serving on a backend below the one it
	// declared. It is serving, so it is not down; it is degraded.
	StateDrift State = "drift"
	// StateUndetermined means the placement signal is not present yet. It is
	// distinct from drift: an absent workload cannot prove CPU placement and
	// must not block a start.
	StateUndetermined State = "undetermined"
	// BackendUndetermined is the public placement verdict name. Its State type
	// prevents callers from treating an unreadable verdict as a boolean.
	BackendUndetermined = StateUndetermined
)

// AccessState is what a device-level probe observed inside a container. It
// mirrors internal/gpuaccess's vocabulary without importing it, so this package
// keeps no path to os/exec.
type AccessState string

const (
	AccessOK      AccessState = "ok"
	AccessRevoked AccessState = "revoked"
	AccessUnknown AccessState = "unknown"
)

// ContainerProbe opens a backend's device node from inside a running container.
// It is the seam that keeps command execution out of this package: the resource
// layer supplies an implementation backed by internal/gpuaccess.
type ContainerProbe func(ctx context.Context, container string, backend Backend) (AccessState, string)

// Placement is the answer to "which backend is this running resource on".
type Placement struct {
	// Declared is the backend the resource asked the platform for.
	Declared Backend `json:"declared"`
	// Observed is the backend the host says the process is using. Empty when
	// State is StateUndetermined.
	Observed Backend `json:"observed,omitempty"`
	// State is StateOK, StateDrift or StateUndetermined.
	State State `json:"state"`
	// Reason is the evidence, in the words of whatever produced it.
	Reason string `json:"reason"`
	// Target describes what was inspected.
	Target string `json:"target"`
}

// Drifted reports whether the resource is serving below its declared backend.
func (p Placement) Drifted() bool { return p.State == StateDrift }

// ErrAccessRevoked is returned when a container that previously had the device
// can no longer open it. This is the daemon-reload failure mode: the host GPU
// is healthy and the container's access is gone.
var ErrAccessRevoked = errors.New("gpu_access_revoked")

// AccessRevokedError carries the operator message the drivers used to build
// themselves. The message text is preserved exactly, because consumers and
// runbooks match on it.
type AccessRevokedError struct {
	Resource string
	Target   string
	Reason   string
}

func (e *AccessRevokedError) Error() string {
	return fmt.Sprintf("%s cannot open /dev/nvidiactl (%s); repair with `vrooli resource restart %s`", e.Target, e.Reason, e.Resource)
}

// Unwrap lets errors.Is(err, ErrAccessRevoked) succeed.
func (e *AccessRevokedError) Unwrap() error { return ErrAccessRevoked }

// Verifier reads observed placement. Every seam is injectable so a test needs
// neither an accelerator nor a container runtime.
type Verifier struct {
	// Facts reads the host. Required.
	Facts FactSource
	// Container probes a device node inside a container. nil means container
	// targets report StateUndetermined with a named reason rather than failing.
	Container ContainerProbe
	// Attempts is how many times to look before giving up. Zero means 3.
	Attempts int
	// Backoff is the pause between attempts. Zero means 250ms.
	Backoff time.Duration
	// Sleep is the pause implementation. nil uses a context-aware timer.
	Sleep func(ctx context.Context, d time.Duration) error
}

func (v Verifier) attempts() int {
	if v.Attempts > 0 {
		return v.Attempts
	}
	return placementParameterA
}

func (v Verifier) backoff() time.Duration {
	if v.Backoff > 0 {
		return v.Backoff
	}
	return tuning.FastHealthPollInterval()
}

func (v Verifier) sleep(ctx context.Context, d time.Duration) error {
	if v.Sleep != nil {
		return v.Sleep(ctx, d)
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// VerifyPlacement reads which backend a running resource is on.
//
// It retries, because a process that has just started may not have opened its
// device yet. It returns StateUndetermined rather than guessing when the host cannot
// answer, and it returns an *AccessRevokedError only for the one case that has
// a known repair: a container that lost a device it used to hold.
func (v Verifier) VerifyPlacement(ctx context.Context, resource string, target PlacementTarget, declared Backend) (Placement, error) {
	description := Describe(target)
	placement := Placement{Declared: declared, State: StateUndetermined, Target: description}

	if declared == BackendCPU {
		// Every host can run on the CPU, so a CPU declaration is trivially met
		// and needs no probe.
		placement.Observed = BackendCPU
		placement.State = StateOK
		placement.Reason = "the cpu backend needs no device"
		return placement, nil
	}

	var lastAccess AccessState
	attempts := v.attempts()
	for attempt := range attempts {
		observed, access, reason, err := v.observe(ctx, target, declared)
		if err != nil {
			return placement, err
		}
		lastAccess = access
		placement.Observed = observed
		placement.Reason = reason
		if observed == declared {
			placement.State = StateOK
			return placement, nil
		}
		if observed != "" {
			placement.State = StateDrift
		} else {
			placement.State = StateUndetermined
		}
		if attempt < attempts-1 {
			if err := v.sleep(ctx, v.backoff()); err != nil {
				return placement, err
			}
		}
	}

	if lastAccess == AccessRevoked {
		return placement, &AccessRevokedError{Resource: resource, Target: description, Reason: placement.Reason}
	}
	return placement, nil
}

// observe reads the evidence for one (target, backend) pair. It returns the
// observed backend (empty when the host could not answer), the container access
// state when one was probed, and the evidence text.
func (v Verifier) observe(ctx context.Context, target PlacementTarget, declared Backend) (Backend, AccessState, string, error) {
	switch value := target.(type) {
	case HostProcess:
		return v.observeHostProcess(ctx, value, declared)
	case Container:
		observed, access, reason := v.observeContainer(ctx, value.Name, declared)
		return observed, access, reason, nil
	case ComposeService:
		name := strings.TrimSpace(value.Container)
		if name == "" {
			return "", AccessUnknown, fmt.Sprintf("compose service %s/%s has no running container", value.Project, value.Service), nil
		}
		observed, access, reason := v.observeContainer(ctx, name, declared)
		return observed, access, reason, nil
	}
	return "", AccessUnknown, "unrecognised placement target", nil
}

// observeContainer asks the injected probe whether the container holds the
// backend's device.
func (v Verifier) observeContainer(ctx context.Context, container string, declared Backend) (Backend, AccessState, string) {
	if v.Container == nil {
		return "", AccessUnknown, "no container probe is configured, so placement cannot be read"
	}
	if !containerVerifiable(declared) {
		return "", AccessUnknown, fmt.Sprintf("backend %s has no container device probe", declared)
	}
	state, reason := v.Container(ctx, container, declared)
	switch state {
	case AccessOK:
		return declared, state, reason
	case AccessRevoked:
		// The container is running without the device it declared, so it is on
		// the CPU: revoked access is drift with a known repair, not unknown.
		return BackendCPU, state, reason
	}
	return "", state, reason
}

// containerVerifiable reports whether a backend has a device node a container
// probe can open. Metal has no container story at all on any supported host.
func containerVerifiable(backend Backend) bool {
	switch backend {
	case BackendCUDA, BackendROCm, BackendVulkan:
		return true
	}
	return false
}

// observeHostProcess reads which device a host process holds. This is the path
// that never existed: managed-service and native-cli resources are exactly the
// ones that fall back to the CPU silently.
func (v Verifier) observeHostProcess(ctx context.Context, process HostProcess, declared Backend) (Backend, AccessState, string, error) {
	if process.PID <= 0 {
		return "", AccessUnknown, "no pid was supplied for the host process", nil
	}
	snapshot, err := v.Facts.Snapshot(ctx)
	if err != nil {
		return "", AccessUnknown, "", fmt.Errorf("read host accelerator facts: %w", err)
	}
	switch declared {
	case BackendCUDA:
		return observeCUDAProcess(snapshot, process)
	case BackendROCm:
		return observeROCmProcess(snapshot, process)
	case BackendMetal:
		return observeMetalProcess(snapshot, process)
	case BackendVulkan:
		return observeVulkanProcess(snapshot, process)
	}
	return "", AccessUnknown, fmt.Sprintf("backend %s has no host-process evidence source", declared), nil
}

// observeCUDAProcess matches the pid against the compute-process rows nvidia-smi
// reports. A process holding no device memory is not on the GPU, whatever its
// configuration says.
func observeCUDAProcess(snapshot hostinventory.Snapshot, process HostProcess) (Backend, AccessState, string, error) {
	if snapshot.ProbeStatuses["nvidia_gpu"] == "not_present" {
		return BackendCPU, AccessUnknown, hostinventory.ToolNvidiaSMI + " is not installed, so no process can be on a CUDA device", nil
	}
	if row, ok := matchComputeProcess(snapshot, process); ok {
		return BackendCUDA, AccessOK, fmt.Sprintf("%s reports %s holding %d bytes on GPU %d", hostinventory.ToolNvidiaSMI, describeComputeRow(row), row.UsedBytes, row.GPUIndex), nil
	}
	if len(snapshot.GPUs) == 0 {
		return BackendCPU, AccessUnknown, "the host reports no CUDA device", nil
	}
	if len(snapshot.GPUProcesses) == 0 {
		if process.Serving {
			return BackendCPU, AccessUnknown, fmt.Sprintf("%s lists no compute process for serving pid %d, so it is running on the CPU", hostinventory.ToolNvidiaSMI, process.PID), nil
		}
		return "", AccessUnknown, noWorkloadReason(process), nil
	}
	return BackendCPU, AccessUnknown, fmt.Sprintf("%s lists no compute process for pid %d, so it is running on the CPU", hostinventory.ToolNvidiaSMI, process.PID), nil
}

// observeROCmProcess matches the pid against the AMD compute-process rows.
func observeROCmProcess(snapshot hostinventory.Snapshot, process HostProcess) (Backend, AccessState, string, error) {
	if !snapshot.HasVendorTool(hostinventory.ToolROCmSMI) {
		return "", AccessUnknown, fmt.Sprintf("%s is not installed, so placement of pid %d cannot be read", hostinventory.ToolROCmSMI, process.PID), nil
	}
	if row, ok := matchComputeProcess(snapshot, process); ok {
		return BackendROCm, AccessOK, fmt.Sprintf("%s reports %s holding %d bytes on GPU %d", hostinventory.ToolROCmSMI, describeComputeRow(row), row.UsedBytes, row.GPUIndex), nil
	}
	if len(snapshot.GPUProcesses) == 0 {
		return "", AccessUnknown, noWorkloadReason(process), nil
	}
	return BackendCPU, AccessUnknown, fmt.Sprintf("%s lists no compute process for pid %d, so it is running on the CPU", hostinventory.ToolROCmSMI, process.PID), nil
}

// observeVulkanProcess reports what can be read without a Vulkan loader call.
// Presence of an ICD manifest says a driver could serve Vulkan; it does not say
// this process opened it, so the honest answer is unknown.
func observeVulkanProcess(snapshot hostinventory.Snapshot, process HostProcess) (Backend, AccessState, string, error) {
	if len(snapshot.VulkanICDs) == 0 {
		return BackendCPU, AccessUnknown, "the host has no Vulkan installable client driver manifest", nil
	}
	if len(snapshot.GPUProcesses) == 0 {
		return "", AccessUnknown, noWorkloadReason(process), nil
	}
	return "", AccessUnknown, fmt.Sprintf("the host has %d Vulkan ICD manifests, but per-process Vulkan attribution is not readable from the host", len(snapshot.VulkanICDs)), nil
}

func noWorkloadReason(process HostProcess) string {
	if reason := strings.TrimSpace(process.NoWorkloadReason); reason != "" {
		return reason
	}
	return "no workload is resident, so placement cannot be read yet"
}

// matchComputeProcess finds the compute row that belongs to a supervised
// resource. It matches the supervised pid directly, and also any compute
// process whose executable lives inside the resource's own artifact tree,
// because the pid that holds the device is frequently a child of the pid the
// control plane supervises.
func matchComputeProcess(snapshot hostinventory.Snapshot, process HostProcess) (hostinventory.GPUProcess, bool) {
	for _, row := range snapshot.GPUProcesses {
		if row.PID == process.PID {
			return row, true
		}
	}
	prefix := strings.TrimSpace(process.ExecutablePrefix)
	if prefix == "" {
		return hostinventory.GPUProcess{}, false
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	for _, row := range snapshot.GPUProcesses {
		if strings.HasPrefix(strings.TrimSpace(row.ProcessName), prefix) {
			return row, true
		}
	}
	return hostinventory.GPUProcess{}, false
}

// describeComputeRow names the compute process in an operator-readable way.
func describeComputeRow(row hostinventory.GPUProcess) string {
	name := strings.TrimSpace(row.ProcessName)
	if name == "" {
		return fmt.Sprintf("pid %d", row.PID)
	}
	return fmt.Sprintf("pid %d (%s)", row.PID, name)
}
