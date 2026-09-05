package models

import (
	"errors"
	"fmt"
	"sort"

	"image-tools/internal/capabilities"
)

const bytesPerGB = 1 << 30

// Selection errors. Callers distinguish them to produce actionable messages
// (e.g. surface install/enable hints vs. a hardware shortfall).
var (
	// ErrUnknownOperation is returned when the requested op is not in the vocabulary.
	ErrUnknownOperation = errors.New("unknown operation")
	// ErrNoEnabledModel is returned when no enabled model serves the operation.
	ErrNoEnabledModel = errors.New("no enabled model for operation")
	// ErrNotRunnable is returned when enabled models exist but none can run on the host.
	ErrNotRunnable = errors.New("no enabled model can run on this host")
	// ErrOverrideInvalid is returned when an explicit override cannot be honored.
	ErrOverrideInvalid = errors.New("model override invalid")
)

// SelectRequest is the input to the hardware-fit selector.
type SelectRequest struct {
	// Operation is the image operation to run (must be in the vocabulary).
	Operation string
	// Host is the probed hardware snapshot (from internal/capabilities).
	Host capabilities.Host
	// OverrideID, when non-empty, forces a specific model (still validated for
	// op-support, enabled-state, and host-runnability).
	OverrideID string
	// QualityPolicy tunes candidate ordering: "fast" preserves cheap defaults,
	// "balanced" keeps historical best-fit ordering, and "quality" prefers
	// quality-tier local models, then BYOK cloud, before low-quality defaults.
	QualityPolicy string
	// AllowBYOK permits BYOK/cloud catalog candidates such as openrouter-image.
	AllowBYOK bool
}

// Selection is the selector's verdict for one operation.
type Selection struct {
	// Model is the chosen registry entry.
	Model Model
	// GPUViable is true when the model will run on a detected GPU with known,
	// sufficient VRAM. Unknown VRAM is treated conservatively as NOT viable.
	GPUViable bool
	// Reason is a short, user-facing explanation of why this model was chosen.
	Reason string
	// Warnings carries non-fatal cautions (CPU time warning, conservative
	// unknown-VRAM fallback, default not host-optimal, …).
	Warnings []string
}

// HardwareFit assesses how a model fits a host.
type HardwareFit struct {
	// GPUViable: a GPU with known VRAM >= the model's minimum is present.
	GPUViable bool
	// Runnable: the model can run at all (GPU-viable or CPU-capable).
	Runnable bool
	// VRAMShortfallGB > 0 when GPUs exist with known VRAM but none meets the
	// minimum; 0 when GPU-viable, no GPU/unknown VRAM, or no GPU requirement.
	VRAMShortfallGB int
}

// Fit computes how model m fits host h. The conservative rule (per plan): a GPU
// whose free VRAM is unknown or below the model's minimum is never counted as
// viable — selection falls back to a CPU-capable tier rather than risk an OOM on
// an unmeasured or currently busy device.
func Fit(m Model, h capabilities.Host) HardwareFit {
	needBytes := uint64(m.Hardware.MinVRAMGB) * bytesPerGB
	var gpuViable bool
	var bestKnownFreeVRAM uint64
	var anyKnownGPU bool
	for _, g := range h.GPUs {
		free, known := g.VRAMFreeBytes()
		if !known {
			continue
		}
		anyKnownGPU = true
		if free > bestKnownFreeVRAM {
			bestKnownFreeVRAM = free
		}
		if needBytes > 0 && free >= needBytes {
			gpuViable = true
		}
	}
	fit := HardwareFit{GPUViable: gpuViable}
	fit.Runnable = gpuViable || m.Hardware.CPUCapable
	if !gpuViable && anyKnownGPU && needBytes > bestKnownFreeVRAM {
		shortfall := int((needBytes - bestKnownFreeVRAM + bytesPerGB - 1) / bytesPerGB) // round up
		fit.VRAMShortfallGB = shortfall
	}
	return fit
}

// SupportsHost reports whether the model declares a build for the host's
// os/arch. An empty OSArch list means "no constraint declared" and is treated as
// supported (the deterministic/built-in tiers omit it).
func (m Model) SupportsHost(h capabilities.Host) bool {
	if len(m.Hardware.OSArch) == 0 {
		return true
	}
	want := h.OS + "/" + h.Arch
	for _, oa := range m.Hardware.OSArch {
		if oa == want {
			return true
		}
	}
	return false
}

// FitClass refines a HardwareFit into a single host-aware badge class for the
// model picker:
//
//	"gpu"               will run on a detected GPU (fast)
//	"cpu"               will run on the CPU (no GPU acceleration / conservative fallback)
//	"insufficient_vram" a GPU is present but free VRAM is short and there is no CPU path
//	"no_gpu"            the model needs a GPU and none was detected
//	"unsupported_os"    no build for this host's os/arch
//
// Unlike the bare runnable/gpu_viable booleans, this lets the UI render an
// affirmative, host-aware label ("Runs on your GPU") instead of a static
// requirement chip that reads as a warning even when the host can run the model.
func FitClass(m Model, h capabilities.Host, fit HardwareFit) string {
	if !m.SupportsHost(h) {
		return "unsupported_os"
	}
	if fit.GPUViable {
		return "gpu"
	}
	if m.Hardware.CPUCapable {
		return "cpu"
	}
	if h.HasGPU() {
		return "insufficient_vram"
	}
	return "no_gpu"
}

// EnabledFunc reports a model's runtime-enabled state (the SQLite overlay over
// the seed's .Enabled). A nil EnabledFunc falls back to Model.Enabled.
type EnabledFunc func(id string) bool

func (r *Registry) enabled(m Model, isEnabled EnabledFunc) bool {
	if isEnabled == nil {
		return m.Enabled
	}
	return isEnabled(m.ID)
}

// Select picks the best-fit enabled model for req.Operation on req.Host,
// honoring the per-op default and any explicit override. isEnabled overlays
// runtime enable/disable state; pass nil to use the seed defaults.
//
// Ranking among runnable, enabled candidates (highest wins):
//  1. GPU-viable on this host (faster, higher fidelity)
//  2. tier rank (quality > default-variant > default > nice-to-have)
//  3. is the seeded default for this op (stability tie-break)
//  4. lower min-VRAM (more headroom), then id ascending (determinism)
//
// In the shipped seed only CPU-capable defaults are enabled, so this reduces to
// "the default" until an operator opts a quality tier in; on a fitting GPU host
// an enabled quality tier then wins, which is the intended best-fit behavior.
func (r *Registry) Select(req SelectRequest, isEnabled EnabledFunc) (Selection, error) {
	candidates, err := r.SelectCandidates(req, isEnabled)
	if err != nil {
		return Selection{}, err
	}
	return candidates[0], nil
}

// SelectCandidates returns runnable candidates in deterministic policy order.
// Select returns the first candidate; resolver uses the full list to skip a
// higher-quality candidate whose provider is not currently available.
func (r *Registry) SelectCandidates(req SelectRequest, isEnabled EnabledFunc) ([]Selection, error) {
	if !r.IsOperation(req.Operation) {
		return nil, fmt.Errorf("%w: %q", ErrUnknownOperation, req.Operation)
	}

	if req.OverrideID != "" {
		sel, err := r.selectOverride(req, isEnabled)
		if err != nil {
			return nil, err
		}
		return []Selection{sel}, nil
	}

	candidates := r.ForOperation(req.Operation)
	var enabledCount int
	var runnable []Model
	worstShortfall := 0
	for _, m := range candidates {
		if m.Backend == BackendOpenRouter && !req.AllowBYOK {
			continue
		}
		if !r.enabled(m, isEnabled) {
			continue
		}
		enabledCount++
		fit := Fit(m, req.Host)
		if fit.Runnable {
			runnable = append(runnable, m)
		} else if fit.VRAMShortfallGB > worstShortfall {
			worstShortfall = fit.VRAMShortfallGB
		}
	}
	if enabledCount == 0 {
		return nil, fmt.Errorf("%w: %q", ErrNoEnabledModel, req.Operation)
	}
	if len(runnable) == 0 {
		if worstShortfall > 0 {
			return nil, fmt.Errorf("%w: %q needs ~%d GB more VRAM, and no CPU-capable model is enabled", ErrNotRunnable, req.Operation, worstShortfall)
		}
		return nil, fmt.Errorf("%w: %q", ErrNotRunnable, req.Operation)
	}

	ranked := r.rankCandidates(runnable, req)
	out := make([]Selection, 0, len(ranked))
	for _, m := range ranked {
		sel := r.buildSelection(m, req)
		if !sel.GPUViable {
			shortfall := Fit(m, req.Host).VRAMShortfallGB
			if worstShortfall > shortfall {
				shortfall = worstShortfall
			}
			appendFreeVRAMShortfallWarning(&sel, shortfall)
		}
		out = append(out, sel)
	}
	return out, nil
}

// rankCandidates returns all candidates in descending preference order.
func (r *Registry) rankCandidates(runnable []Model, req SelectRequest) []Model {
	type scored struct {
		m         Model
		gpuViable bool
	}
	items := make([]scored, len(runnable))
	for i, m := range runnable {
		items[i] = scored{m: m, gpuViable: Fit(m, req.Host).GPUViable}
	}
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if req.QualityPolicy == "fast" {
			if da, db := a.m.IsDefaultFor(req.Operation), b.m.IsDefaultFor(req.Operation); da != db {
				return da
			}
		}
		if req.QualityPolicy == "quality" {
			if qa, qb := qualityPolicyRank(a.m), qualityPolicyRank(b.m); qa != qb {
				return qa > qb
			}
		}
		if a.gpuViable != b.gpuViable {
			return a.gpuViable // GPU-viable first
		}
		if ra, rb := a.m.Tier.rank(), b.m.Tier.rank(); ra != rb {
			return ra > rb // higher tier first
		}
		if da, db := a.m.IsDefaultFor(req.Operation), b.m.IsDefaultFor(req.Operation); da != db {
			return da // the seeded default wins the tie
		}
		if a.m.Hardware.MinVRAMGB != b.m.Hardware.MinVRAMGB {
			return a.m.Hardware.MinVRAMGB < b.m.Hardware.MinVRAMGB // more headroom
		}
		return a.m.ID < b.m.ID // deterministic final tie-break
	})
	out := make([]Model, len(items))
	for i, item := range items {
		out[i] = item.m
	}
	return out
}

func qualityPolicyRank(m Model) int {
	if m.Tier == TierQuality {
		return 3
	}
	if m.Backend == BackendOpenRouter {
		return 2
	}
	if m.Tier == TierDefault || m.Tier == TierDefaultVariant {
		return 1
	}
	return 0
}

func (r *Registry) selectOverride(req SelectRequest, isEnabled EnabledFunc) (Selection, error) {
	m, ok := r.ByID(req.OverrideID)
	if !ok {
		return Selection{}, fmt.Errorf("%w: model %q not found", ErrOverrideInvalid, req.OverrideID)
	}
	if !m.ServesOperation(req.Operation) {
		return Selection{}, fmt.Errorf("%w: model %q does not serve operation %q", ErrOverrideInvalid, req.OverrideID, req.Operation)
	}
	if !r.enabled(m, isEnabled) {
		return Selection{}, fmt.Errorf("%w: model %q is disabled", ErrOverrideInvalid, req.OverrideID)
	}
	fit := Fit(m, req.Host)
	if !fit.Runnable {
		if fit.VRAMShortfallGB > 0 {
			return Selection{}, fmt.Errorf("%w: model %q needs ~%d GB more VRAM and is not CPU-capable", ErrOverrideInvalid, req.OverrideID, fit.VRAMShortfallGB)
		}
		return Selection{}, fmt.Errorf("%w: model %q cannot run on this host", ErrOverrideInvalid, req.OverrideID)
	}
	sel := r.buildSelection(m, req)
	if !sel.GPUViable {
		appendFreeVRAMShortfallWarning(&sel, fit.VRAMShortfallGB)
	}
	sel.Reason = fmt.Sprintf("model %q selected by explicit override", m.ID)
	return sel, nil
}

func (r *Registry) buildSelection(m Model, req SelectRequest) Selection {
	fit := Fit(m, req.Host)
	sel := Selection{Model: m, GPUViable: fit.GPUViable}

	switch {
	case fit.GPUViable:
		sel.Reason = fmt.Sprintf("model %q runs on GPU (needs >=%d GB VRAM)", m.ID, m.Hardware.MinVRAMGB)
	case m.Hardware.CPUCapable:
		sel.Reason = fmt.Sprintf("model %q runs on CPU", m.ID)
		warn := fmt.Sprintf("running %q on CPU — expect slower results", m.ID)
		if m.Hardware.SpeedNote != "" {
			warn += ": " + m.Hardware.SpeedNote
		}
		sel.Warnings = append(sel.Warnings, warn)
		if req.Host.HasGPU() {
			if _, known := req.Host.MaxFreeVRAMBytes(); !known {
				sel.Warnings = append(sel.Warnings, "GPU detected but VRAM is unknown; falling back to CPU conservatively")
			}
		}
	}

	if m.IO.Notes != "" {
		// surfaced as informational context, not a caution
		sel.Reason += "; " + m.IO.Notes
	}
	return sel
}

func appendFreeVRAMShortfallWarning(sel *Selection, shortfallGB int) {
	if shortfallGB > 0 {
		sel.Warnings = append(sel.Warnings, fmt.Sprintf("GPU detected but free VRAM is ~%d GB short; falling back to CPU to avoid OOM", shortfallGB))
	}
}
