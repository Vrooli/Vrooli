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
// whose VRAM is unknown is never counted as viable — selection falls back to a
// CPU-capable tier rather than risk an OOM on an unmeasured device.
func Fit(m Model, h capabilities.Host) HardwareFit {
	needBytes := uint64(m.Hardware.MinVRAMGB) * bytesPerGB
	var gpuViable bool
	var bestKnownVRAM uint64
	var anyKnownGPU bool
	for _, g := range h.GPUs {
		if !g.VRAMKnown() {
			continue
		}
		anyKnownGPU = true
		if g.VRAMBytes > bestKnownVRAM {
			bestKnownVRAM = g.VRAMBytes
		}
		if g.VRAMBytes >= needBytes {
			gpuViable = true
		}
	}
	fit := HardwareFit{GPUViable: gpuViable}
	fit.Runnable = gpuViable || m.Hardware.CPUCapable
	if !gpuViable && anyKnownGPU && needBytes > bestKnownVRAM {
		shortfall := int((needBytes - bestKnownVRAM + bytesPerGB - 1) / bytesPerGB) // round up
		fit.VRAMShortfallGB = shortfall
	}
	return fit
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
	if !r.IsOperation(req.Operation) {
		return Selection{}, fmt.Errorf("%w: %q", ErrUnknownOperation, req.Operation)
	}

	if req.OverrideID != "" {
		return r.selectOverride(req, isEnabled)
	}

	candidates := r.ForOperation(req.Operation)
	var enabledCount int
	var runnable []Model
	worstShortfall := 0
	for _, m := range candidates {
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
		return Selection{}, fmt.Errorf("%w: %q", ErrNoEnabledModel, req.Operation)
	}
	if len(runnable) == 0 {
		if worstShortfall > 0 {
			return Selection{}, fmt.Errorf("%w: %q needs ~%d GB more VRAM, and no CPU-capable model is enabled", ErrNotRunnable, req.Operation, worstShortfall)
		}
		return Selection{}, fmt.Errorf("%w: %q", ErrNotRunnable, req.Operation)
	}

	best := r.rankBest(runnable, req)
	return r.buildSelection(best, req), nil
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
	sel.Reason = fmt.Sprintf("model %q selected by explicit override", m.ID)
	return sel, nil
}

// rankBest returns the highest-ranked model per the documented comparator.
// runnable is assumed non-empty.
func (r *Registry) rankBest(runnable []Model, req SelectRequest) Model {
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
	return items[0].m
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
			if _, known := req.Host.MaxVRAMBytes(); !known {
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
