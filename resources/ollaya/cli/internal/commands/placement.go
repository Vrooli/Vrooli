package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil/hostsem"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/policy"
	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/upstream"
)

// Placement policy is enforced before a local model is asked to load.
//
// The daemon loads a model on first use. When its memory does not fit it
// either fails the load or, on a CPU-pinned daemon, runs a model that was
// meant for the accelerator at a fraction of the speed. Both look like a hang
// and corrupt latency budgets. The model-policy `placement` block says where a
// model may run and what it costs there, and this file refuses a request that
// cannot be served, with a typed retryable reason (insufficient_vram or
// insufficient_memory), so ai-gateway continues to its next candidate instead
// of waiting on a load that cannot succeed.
//
// The capacity question is asked of the shared broker (`vrooli capacity admit`,
// which is engine.Decide without a recorded claim). This file never does its
// own free-memory arithmetic. The broker excludes Ollaya's own claim, so the
// models already resident in the daemon are added to the ask: a model fits only
// if it fits beside the others, and idle residents are unloaded largest first
// when that makes room, because a decision that was asked for outranks a model
// that is merely warm.

const (
	admitKindVRAM = "vram"
	admitKindRAM  = "ram"
	// placementTimeout bounds the fit check a role resolution performs: a
	// resident read, a /proc read, and a broker call that takes about 0.1s.
	placementTimeout = 10 * time.Second
)

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
// be honoured right now, unloading idle residents when that makes room. It
// never fails a request because a probe is unavailable: an unknown placement is
// left to the daemon's own hard device pin.
//
// A model that is not resident is about to be loaded by the decision call that
// follows, so admission and that load are one critical section per daemon: the
// returned release must be called after the decision returns. The broker
// answers from the memory that is free now, and a load in flight has not taken
// it yet, so two concurrent callers would each be told the same memory is
// free and both would load into it. The second caller waits for the slot and
// then either finds the model resident (no load, no broker call) or asks the
// broker with the first load already visible. A resident model takes no slot,
// so warm decisions never queue.
func (h *Handlers) requireModelPlacement(ctx context.Context, p policy.Policy, resolution policy.Resolution) (func(), error) {
	noop := func() {}
	if resolution.Locality != policy.LocalityLocal {
		return noop, nil
	}
	targets := residentTargets(p, resolution.Model)
	if h.anyResident(ctx, targets) {
		return noop, h.placementGate(ctx, p, resolution, true)
	}
	release, err := h.acquireLoadSlot(ctx)
	if err != nil {
		return noop, &upstream.ProviderError{
			Code:    upstream.CodeProviderOverloaded,
			Message: fmt.Sprintf("%s waited for another model load on this daemon and gave up: %v", resolution.Model, err),
		}
	}
	if err := h.placementGate(ctx, p, resolution, true); err != nil {
		release()
		return noop, err
	}
	if h.anyResident(ctx, targets) {
		// Another caller loaded it while this one waited; nothing is left to load.
		release()
		return noop, nil
	}
	return release, nil
}

func (h *Handlers) anyResident(ctx context.Context, targets []residentTarget) bool {
	residents, err := h.residents(ctx)
	if err != nil {
		return false
	}
	for _, resident := range residents {
		if residentIsTarget(resident.Name, targets) {
			return true
		}
	}
	return false
}

// acquireLoadSlot takes the per-daemon load slot, a one-slot host semaphore
// shared by every resource-ollaya process on the machine. It waits as long as
// the caller's context allows.
func (h *Handlers) acquireLoadSlot(ctx context.Context) (func(), error) {
	if h.LoadSlot != nil {
		return h.LoadSlot(ctx)
	}
	sem, err := hostsem.New(h.loadLockDir(), 1)
	if err != nil {
		return nil, err
	}
	return sem.Acquire(ctx)
}

func (h *Handlers) loadLockDir() string {
	getenv := h.GetEnv
	if getenv == nil {
		getenv = os.Getenv
	}
	if dir := strings.TrimSpace(getenv("OLLAYA_LOAD_LOCK_DIR")); dir != "" {
		return dir
	}
	if runtime := strings.TrimSpace(getenv("VROOLI_RUNTIME_DIR")); runtime != "" {
		return filepath.Join(runtime, "resources", "ollaya", "load")
	}
	if cache, err := os.UserCacheDir(); err == nil && cache != "" {
		return filepath.Join(cache, "vrooli", "resources", "ollaya", "load")
	}
	return filepath.Join(os.TempDir(), "vrooli", "resources", "ollaya", "load")
}

// checkModelPlacement is the side-effect-free form used when a role resolves:
// it refuses only a model that could not fit even if every idle resident were
// unloaded, so the gateway skips a hopeless candidate before spending rights
// or load work on it, and never evicts a warm model for a request that rights
// would refuse anyway.
func (h *Handlers) checkModelPlacement(ctx context.Context, p policy.Policy, resolution policy.Resolution) error {
	return h.placementGate(ctx, p, resolution, false)
}

func (h *Handlers) placementGate(ctx context.Context, p policy.Policy, resolution policy.Resolution, mayUnload bool) error {
	if resolution.Locality != policy.LocalityLocal {
		return nil
	}
	placement := p.EffectivePlacement(resolution.Model)
	if placement == nil {
		return nil
	}
	targets := residentTargets(p, resolution.Model)
	residents, residentsErr := h.residents(ctx)

	// A resident model already holds its memory and reports the device it
	// loaded on. A CPU-resident model that needs the accelerator is exactly the
	// state this policy forbids.
	for _, resident := range residents {
		if !residentIsTarget(resident.Name, targets) {
			continue
		}
		if placement.Device == policy.PlacementDeviceGPU && !isAcceleratorDevice(resident.Device) {
			return &upstream.ProviderError{
				Code:    upstream.CodeInsufficientVRAM,
				Message: fmt.Sprintf("%s is loaded on %s; its policy requires the GPU", resolution.Model, resident.Device),
			}
		}
		return nil
	}

	mode, _ := h.daemonDevice(ctx)
	switch {
	case placement.Device == policy.PlacementDeviceCPU, mode != "" && !isAcceleratorDevice(mode):
		if placement.Device == policy.PlacementDeviceGPU {
			return &upstream.ProviderError{
				Code:    upstream.CodeInsufficientVRAM,
				Message: fmt.Sprintf("%s requires the GPU but ollaya is running on %s; restart it on cuda instead of paying a CPU load", resolution.Model, mode),
			}
		}
		return h.admitHostMemory(ctx, resolution.Model, placement)
	case placement.VRAMBytes > 0:
		return h.admitAccelerator(ctx, resolution.Model, placement, targets, residents, residentsErr, mayUnload)
	}
	return nil
}

func (h *Handlers) admitHostMemory(ctx context.Context, model string, placement *policy.ModelPlacement) error {
	if placement.RAMBytes <= 0 {
		return nil
	}
	if admitted, reason, err := h.admit(ctx, admitKindRAM, placement.RAMBytes); err == nil && !admitted {
		return &upstream.ProviderError{
			Code:    upstream.CodeInsufficientMemory,
			Message: fmt.Sprintf("%s needs %s of host memory; the capacity broker refused admission: %s", model, humanGiB(placement.RAMBytes), reason),
		}
	}
	return nil
}

// admitAccelerator asks whether the model fits on the accelerator beside the
// models already resident, unloading idle residents largest first when that is
// what it takes. The broker excludes Ollaya's own claim, so the residents'
// bytes are added to the ask instead of being assumed free. When unloading is
// not allowed the ask is the footprint alone, which answers "would it fit if
// Ollaya released everything it holds".
func (h *Handlers) admitAccelerator(ctx context.Context, model string, placement *policy.ModelPlacement, targets []residentTarget, residents []upstream.ResidentModel, residentsErr error, mayUnload bool) error {
	var others []upstream.ResidentModel
	if residentsErr == nil && mayUnload {
		for _, resident := range residents {
			if isAcceleratorDevice(resident.Device) && resident.SizeVRAM > 0 && !residentIsTarget(resident.Name, targets) {
				others = append(others, resident)
			}
		}
	}
	sort.Slice(others, func(i, j int) bool { return others[i].SizeVRAM > others[j].SizeVRAM })
	for {
		ask := placement.VRAMBytes
		for _, other := range others {
			ask += other.SizeVRAM
		}
		admitted, reason, err := h.admit(ctx, admitKindVRAM, ask)
		if err != nil || admitted {
			// An unavailable broker is not a refusal.
			return nil
		}
		if len(others) == 0 {
			return &upstream.ProviderError{
				Code:    upstream.CodeInsufficientVRAM,
				Message: fmt.Sprintf("%s needs %s of VRAM; the capacity broker refused admission: %s", model, humanGiB(placement.VRAMBytes), reason),
			}
		}
		// The ask counted this resident's bytes as free. If it cannot be released
		// they are not, so loading would run into the same shortage the broker
		// just described; refuse with the cause instead of paying for a load.
		if err := h.unload(ctx, others[0].Name); err != nil {
			return &upstream.ProviderError{
				Code: upstream.CodeInsufficientVRAM,
				Message: fmt.Sprintf("%s needs %s of VRAM; the capacity broker refused admission (%s) and unloading idle resident %s to make room failed: %v",
					model, humanGiB(placement.VRAMBytes), reason, others[0].Name, err),
			}
		}
		others = others[1:]
	}
}

// residentTarget is a model name the policy serves and the manifest digest it is
// pinned to.
type residentTarget struct {
	name, digest string
}

// residentTargets lists the models a role's model can be served from: itself
// and, for a router, its routes.
func residentTargets(p policy.Policy, model string) []residentTarget {
	targets := []residentTarget{{model, p.Models[model].Digest}}
	for _, route := range p.Models[model].Routes {
		targets = append(targets, residentTarget{route, p.Models[route].Digest})
	}
	return targets
}

// residentIsTarget reports whether a daemon-listed resident is one of the
// targets. The daemon lists a digest-pinned model as `name:tag@sha256:<digest>`,
// so the name is compared without the suffix, and a suffix naming a different
// digest than the policy pins is a different version of the model, not the
// target.
func residentIsTarget(resident string, targets []residentTarget) bool {
	name, pinned, _ := strings.Cut(resident, "@")
	for _, target := range targets {
		if name != target.name && name != upstream.InstalledName(target.name) {
			continue
		}
		if pinned == "" || target.digest == "" || strings.TrimPrefix(pinned, "sha256:") == strings.TrimPrefix(target.digest, "sha256:") {
			return true
		}
	}
	return false
}

func (h *Handlers) residents(ctx context.Context) ([]upstream.ResidentModel, error) {
	if h.Residents != nil {
		return h.Residents(ctx)
	}
	return h.client().Residents(ctx)
}

func (h *Handlers) unload(ctx context.Context, model string) error {
	if h.Unload != nil {
		return h.Unload(ctx, model)
	}
	return h.client().Unload(ctx, model)
}

// daemonDevice reports the device the managed daemon is pinned to
// (OLLAYA_DEVICE in its own environment, which the selected backend writes).
// It is a process-environment read, not a status audit: the full status call
// reconciles nvidia-smi and artifact bytes and takes seconds, which a per-
// decision gate cannot afford. Empty means the daemon chooses its own device.
func (h *Handlers) daemonDevice(ctx context.Context) (string, error) {
	if h.DaemonDevice != nil {
		return h.DaemonDevice(ctx)
	}
	records, err := resources.ReadManagedServiceRecords()
	if err != nil {
		return "", err
	}
	for _, record := range records {
		if record.Resource != "ollaya" || record.State.PID <= 0 {
			continue
		}
		env, err := process.ReadEnvironment(record.State.PID)
		if err != nil {
			return "", err
		}
		return strings.ToLower(strings.TrimSpace(env["OLLAYA_DEVICE"])), nil
	}
	return "", fmt.Errorf("managed ollaya owner record not found")
}

// admit asks the broker whether bytes of kind fit in capacity that is free
// right now. `capacity admit` is read-only and never counts another owner's
// reclaimable claim (the broker runs it with NoReclaim), so a yes means the
// load will not need anyone to give memory back. Making room by reclaiming an
// idle owner is a different broker operation that is not offered as a
// side-effect-free question; see the Ollama idle-unload finding in the Phase 2
// notes.
func (h *Handlers) admit(ctx context.Context, kind string, bytes int64) (bool, string, error) {
	if h.CapacityAdmit != nil {
		return h.CapacityAdmit(ctx, kind, bytes)
	}
	args := []string{"capacity", "admit",
		"--owner-kind", "resource", "--owner-id", "ollaya",
		"--resource-kind", kind,
		"--preferred", fmt.Sprintf("%d", bytes), "--floor", fmt.Sprintf("%d", bytes),
		"--priority", "service", "--json"}
	if kind == admitKindVRAM {
		args = append(args, "--gpu-index", "0")
	}
	raw, err := h.controlPlane(ctx, "vrooli", args...)
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
