package accel

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

// ErrNoBackendReady is returned when a resource declares require:required and
// the host can reach none of the non-CPU backends it named. Callers match on it
// with errors.Is; the wrapping NoBackendReadyError carries the detail.
var ErrNoBackendReady = errors.New("no declared accelerator backend is ready")

// NoBackendReadyError explains which backends were considered, why each was
// rejected, and the exact command that repairs the host.
type NoBackendReadyError struct {
	Resource    string
	Considered  []BackendVerdict
	Remediation string
}

func (e *NoBackendReadyError) Error() string {
	reasons := make([]string, 0, len(e.Considered))
	for _, verdict := range e.Considered {
		reasons = append(reasons, fmt.Sprintf("%s: %s", verdict.Backend, verdict.Reason))
	}
	message := fmt.Sprintf("resource %q requires an accelerator but no declared backend is ready (%s)", e.Resource, strings.Join(reasons, "; "))
	if e.Remediation != "" {
		message += "; repair with `" + e.Remediation + "`"
	}
	return message
}

// Unwrap lets errors.Is(err, ErrNoBackendReady) succeed.
func (e *NoBackendReadyError) Unwrap() error { return ErrNoBackendReady }

// BackendVerdict records why one declared backend was chosen or skipped.
type BackendVerdict struct {
	Backend Backend `json:"backend"`
	Ready   bool    `json:"ready"`
	Reason  string  `json:"reason"`
}

// ReadinessResult is the answer to "which backend should this resource start
// on, and what did the host say about the ones it skipped".
type ReadinessResult struct {
	// Selected is the first declared backend the host can reach. It is
	// BackendCPU when nothing else is reachable and require is not required.
	Selected Backend `json:"selected"`
	// Declared is the resource's first-choice backend, whether or not it is
	// reachable. Comparing it with Selected is what makes drift computable
	// before the process has even started.
	Declared Backend `json:"declared"`
	// Drift is true when the resource will start below its first choice and
	// its require state makes that worth reporting.
	Drift bool `json:"drift"`
	// Considered lists every declared backend with the host's verdict on it.
	Considered []BackendVerdict `json:"considered"`
}

// acceleratorRemediation names the host state that most often makes a declared
// backend unreachable, and the command that repairs it. Privilege is requested
// only by vrooli setup, never by resource runtime code, so the remediation is
// always a command for the operator to run rather than something to escalate.
var acceleratorRemediation = map[Backend]string{
	BackendCUDA:   "vrooli host safeguard nvidia_driver --sudo-mode ask",
	BackendROCm:   "vrooli setup --include-optional",
	BackendVulkan: "vrooli setup --include-optional",
}

// Readiness answers which declared backend this host can actually reach.
//
// It never returns an error for a preferred or opportunistic resource: falling
// back to CPU is a reportable state, not a failure. It returns
// ErrNoBackendReady only when the resource declared require:required.
func Readiness(ctx context.Context, source FactSource, spec Spec) (ReadinessResult, error) {
	snapshot, err := source.Snapshot(ctx)
	if err != nil {
		return ReadinessResult{}, fmt.Errorf("read host accelerator facts: %w", err)
	}
	return ReadinessFromSnapshot(snapshot, spec)
}

// ReadinessFromSnapshot is Readiness against a snapshot the caller already
// holds, so a start path that has collected facts once does not collect twice.
func ReadinessFromSnapshot(snapshot hostinventory.Snapshot, spec Spec) (ReadinessResult, error) {
	declared := spec.Backends
	if len(declared) == 0 {
		declared = []Backend{BackendCPU}
	}
	reachable := ReachableBackends(snapshot)

	result := ReadinessResult{
		Declared:   declared[0],
		Considered: make([]BackendVerdict, 0, len(declared)),
	}
	selected := Backend("")
	for _, backend := range declared {
		ready := slices.Contains(reachable, backend)
		verdict := BackendVerdict{Backend: backend, Ready: ready}
		switch {
		case ready && selected == "":
			verdict.Reason = fmt.Sprintf("host reports %s reachable", backend)
			selected = backend
		case ready:
			verdict.Reason = fmt.Sprintf("host reports %s reachable, but %s is preferred", backend, selected)
		default:
			verdict.Reason = unreachableReason(backend, snapshot)
		}
		result.Considered = append(result.Considered, verdict)
	}

	if selected == "" {
		// CPU is reachable on every host by construction, so this only happens
		// when the resource declared no CPU floor at all.
		if spec.EffectiveRequire() == RequireRequired {
			return result, &NoBackendReadyError{
				Resource:    spec.Resource,
				Considered:  result.Considered,
				Remediation: RemediationFor(declared),
			}
		}
		selected = BackendCPU
		result.Considered = append(result.Considered, BackendVerdict{
			Backend: BackendCPU,
			Ready:   true,
			Reason:  "no declared backend is reachable; falling back to the cpu floor",
		})
	}

	result.Selected = selected
	if selected != result.Declared && spec.EffectiveRequire() != RequireNone {
		result.Drift = true
	}
	if spec.EffectiveRequire() == RequireRequired && selected == BackendCPU && spec.Accelerated() {
		return result, &NoBackendReadyError{
			Resource:    spec.Resource,
			Considered:  result.Considered,
			Remediation: RemediationFor(declared),
		}
	}
	return result, nil
}

// unreachableReason explains what the host said about a backend it cannot
// reach, using only what hostinventory observed.
func unreachableReason(backend Backend, snapshot hostinventory.Snapshot) string {
	status := snapshot.ProbeStatuses
	switch backend {
	case BackendCUDA:
		switch status["nvidia_gpu"] {
		case "not_present":
			return hostinventory.ToolNvidiaSMI + " is not installed on this host"
		case "no_devices":
			return hostinventory.ToolNvidiaSMI + " reports no devices"
		case "failed":
			return "the " + hostinventory.ToolNvidiaSMI + " probe failed; the driver may be loaded without its device nodes"
		case "unsupported":
			return "this platform has no NVIDIA probe"
		}
		return "the host reports no CUDA device"
	case BackendMetal:
		if snapshot.OS != "darwin" {
			return fmt.Sprintf("metal is only reachable on darwin; this host is %s", snapshot.OS)
		}
		return "the host enumerated no Metal-capable device"
	case BackendROCm:
		switch status["rocm"] {
		case "unsupported":
			return fmt.Sprintf("the ROCm kernel compute interface is Linux-only; this host is %s", snapshot.OS)
		case "no_devices":
			return "no ROCm kernel compute interface is present"
		}
		return "the host reports no ROCm device"
	case BackendVulkan:
		if status["vulkan"] == "unsupported" {
			return fmt.Sprintf("no Vulkan loader location is known for %s", snapshot.OS)
		}
		return "the host has no Vulkan installable client driver manifest"
	case BackendCPU:
		return "the cpu backend is always reachable"
	}
	return fmt.Sprintf("the host reports no %s device", backend)
}

// RemediationFor picks the repair command for the highest-preference
// accelerated backend the resource declared. Empty means nothing known repairs
// it, which is itself worth showing rather than inventing a command.
func RemediationFor(declared []Backend) string {
	for _, backend := range declared {
		if command, ok := acceleratorRemediation[backend]; ok {
			return command
		}
	}
	return ""
}
