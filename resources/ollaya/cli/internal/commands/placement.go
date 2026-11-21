package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/policy"
	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/upstream"
)

// Placement policy is enforced before a local model is asked to load.
//
// Ollaya loads a model on first use and falls back to the CPU when the GPU
// allocation fails. For Laya that is fine (it serves well on the CPU); for
// Winnow it turns a fast failure into a 4-55 second CPU load that looks like a
// hang and corrupts latency budgets. The model-policy `placement` block says
// where a model may run, and this file refuses a request that cannot be served
// there with a typed, retryable reason (insufficient_vram) so ai-gateway falls
// back to its next candidate instead of waiting on a CPU load.
//
// The capacity question is asked of the shared broker (`vrooli capacity admit`,
// which is engine.Decide without a recorded claim). This file never does its
// own free-VRAM arithmetic.

// isAcceleratorDevice reports whether a daemon-reported device is an
// accelerator. Ollaya reports "cuda:0" or "cpu".
func isAcceleratorDevice(device string) bool {
	device = strings.ToLower(strings.TrimSpace(device))
	if device == "" || device == "cpu" {
		return false
	}
	return strings.Contains(device, "cuda") || strings.HasPrefix(device, "gpu") || strings.Contains(device, "vulkan") || strings.Contains(device, "rocm")
}

// requireModelPlacement refuses a local model whose declared placement cannot
// be honoured. It never fails a request because a probe is unavailable; an
// unknown placement is left to the daemon's own hard device pin.
func (h *Handlers) requireModelPlacement(ctx context.Context, p policy.Policy, resolution policy.Resolution) error {
	if resolution.Locality != policy.LocalityLocal {
		return nil
	}
	model, ok := p.Models[resolution.Model]
	if !ok || !model.RequiresGPU() {
		return nil
	}

	// A resident model already reports the device it was loaded on. This is the
	// cheapest answer and the one that matters most: a CPU-resident Winnow is
	// exactly the state this policy forbids.
	if device, found, err := h.residentDevice(ctx, resolution.Model); err == nil && found {
		if !isAcceleratorDevice(device) {
			return &upstream.ProviderError{
				Code:    upstream.CodeInsufficientVRAM,
				Message: fmt.Sprintf("%s is loaded on %s; its policy requires the GPU", resolution.Model, device),
			}
		}
		return nil
	}

	// Not resident. The control plane owns placement truth for the running
	// resource, so ask it rather than guessing from process state.
	if mode, err := h.hostPlacement(ctx); err == nil && strings.TrimSpace(mode) != "" && !isAcceleratorDevice(mode) {
		return &upstream.ProviderError{
			Code:    upstream.CodeInsufficientVRAM,
			Message: fmt.Sprintf("%s requires the GPU but ollaya is running on %s; restart it on cuda instead of paying a CPU load", resolution.Model, mode),
		}
	}

	// Finally ask the shared broker whether the declared footprint fits right
	// now. A refusal is typed; an unavailable broker is not a refusal.
	if model.Placement.VRAMBytes > 0 {
		if admitted, reason, err := h.admitVRAM(ctx, model.Placement.VRAMBytes); err == nil && !admitted {
			return &upstream.ProviderError{
				Code:    upstream.CodeInsufficientVRAM,
				Message: fmt.Sprintf("%s needs %s of VRAM; the capacity broker refused admission: %s", resolution.Model, humanGiB(model.Placement.VRAMBytes), reason),
			}
		}
	}
	return nil
}

func (h *Handlers) residentDevice(ctx context.Context, model string) (string, bool, error) {
	if h.ResidentDevice != nil {
		return h.ResidentDevice(ctx, model)
	}
	return h.client().ResidentDevice(ctx, model)
}

// hostPlacement reads the observed backend from the control plane's own status,
// which already reconciles the manifest, the selected backend and nvidia-smi.
func (h *Handlers) hostPlacement(ctx context.Context) (string, error) {
	if h.HostPlacement != nil {
		return h.HostPlacement(ctx)
	}
	raw, err := h.controlPlane(ctx, "vrooli", "resource", "status", "ollaya", "--json")
	if err != nil {
		return "", err
	}
	var status struct {
		ObservedMode string `json:"observed_mode"`
		ModeDrift    bool   `json:"mode_drift"`
	}
	if err := json.Unmarshal(raw, &status); err != nil {
		return "", err
	}
	return status.ObservedMode, nil
}

func (h *Handlers) admitVRAM(ctx context.Context, bytes int64) (bool, string, error) {
	if h.CapacityAdmit != nil {
		return h.CapacityAdmit(ctx, bytes)
	}
	raw, err := h.controlPlane(ctx, "vrooli", "capacity", "admit",
		"--owner-kind", "resource", "--owner-id", "ollaya",
		"--resource-kind", "vram", "--gpu-index", "0",
		"--preferred", fmt.Sprintf("%d", bytes), "--floor", fmt.Sprintf("%d", bytes),
		"--priority", "service", "--json")
	if err != nil {
		return false, "", err
	}
	var out struct {
		Admitted bool `json:"admitted"`
		Verdict  struct {
			Kind   string `json:"kind"`
			Reason string `json:"reason"`
		} `json:"verdict"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return false, "", err
	}
	reason := strings.TrimSpace(out.Verdict.Reason)
	if reason == "" {
		reason = out.Verdict.Kind
	}
	return out.Admitted, reason, nil
}

func (h *Handlers) controlPlane(ctx context.Context, name string, args ...string) ([]byte, error) {
	if h.RunControlPlane != nil {
		return h.RunControlPlane(ctx, name, args...)
	}
	return exec.CommandContext(ctx, name, args...).Output()
}

func humanGiB(bytes int64) string {
	const gib = 1024 * 1024 * 1024
	if bytes <= 0 {
		return "0 GiB"
	}
	return fmt.Sprintf("%.1f GiB", float64(bytes)/float64(gib))
}
