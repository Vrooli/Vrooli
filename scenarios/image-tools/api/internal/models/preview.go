package models

import (
	"context"
	"fmt"

	"image-tools/internal/capabilities"
)

// Previewer answers "which model would a submit run for this operation on this
// host" for every read surface (`models select`, `ai list`, the model picker's
// selected marker). It is the one place that merges the catalog the way a real
// job does: seed plus operator-registered models, the effective enabled overlay,
// the operator's per-operation pin, the current rights decision and the host's
// hardware fit. A surface that selected from the seed alone would report a
// default no job would ever run.
type Previewer struct {
	Registry *Registry
	Probe    capabilities.Probe
	// Overlay returns the enabled overrides that apply (may be nil: seed only).
	Overlay func(ctx context.Context) (map[string]bool, error)
	// Customs returns operator-registered entries (may be nil: none).
	Customs func(ctx context.Context) ([]Model, error)
	// Pin returns the operator's pinned default model for an operation, "" when
	// none (may be nil: no pins).
	Pin             func(ctx context.Context, operation string) (string, error)
	RightsEvaluator RightsEvaluator
	// CapacityManagedGPU is the GPU-fit rule real jobs use (see the models
	// handler's Deps).
	CapacityManagedGPU bool
}

// PreviewRequest is one preview. Zero values preview the default submit: no
// override, balanced quality policy, no BYOK, GPU allowed.
type PreviewRequest struct {
	Operation     string
	OverrideID    string
	QualityPolicy string
	AllowBYOK     bool
	ForceCPU      bool
}

// PreviewContext is the merged catalog state one or more previews ran against.
// Handlers reuse it to build wire views without re-reading the stores.
type PreviewContext struct {
	// Registry is the seed merged with custom entries.
	Registry *Registry
	// CustomIDs is the set of operator-registered ids.
	CustomIDs map[string]bool
	// Overlay is the active enabled overrides.
	Overlay map[string]bool
	Host    capabilities.Host

	previewer *Previewer
}

// Load probes the host and reads the catalog state once.
func (p *Previewer) Load(ctx context.Context) (*PreviewContext, error) {
	host, err := p.Probe.Inventory(ctx)
	if err != nil {
		return nil, fmt.Errorf("probe host hardware: %w", err)
	}
	pc := &PreviewContext{Host: host, previewer: p, CustomIDs: map[string]bool{}}
	if p.Overlay != nil {
		if pc.Overlay, err = p.Overlay(ctx); err != nil {
			return nil, fmt.Errorf("load model state: %w", err)
		}
	}
	var customs []Model
	if p.Customs != nil {
		if customs, err = p.Customs(ctx); err != nil {
			return nil, fmt.Errorf("load custom model catalog: %w", err)
		}
	}
	if pc.Registry, err = p.Registry.WithCustom(customs); err != nil {
		return nil, fmt.Errorf("load custom model catalog: %w", err)
	}
	for _, m := range customs {
		pc.CustomIDs[m.ID] = true
	}
	return pc, nil
}

// Select previews one operation against the loaded state.
func (pc *PreviewContext) Select(ctx context.Context, req PreviewRequest) (Selection, error) {
	p := pc.previewer
	override := req.OverrideID
	if override == "" && p.Pin != nil {
		if pinned, err := p.Pin(ctx, req.Operation); err == nil {
			override = pinned
		}
	}
	return pc.Registry.Select(SelectRequest{
		Operation:          req.Operation,
		Host:               pc.Host,
		RightsEvaluator:    p.RightsEvaluator,
		Context:            ctx,
		QualityPolicy:      req.QualityPolicy,
		AllowBYOK:          req.AllowBYOK,
		OverrideID:         override,
		ForceCPU:           req.ForceCPU,
		CapacityManagedGPU: p.CapacityManagedGPU,
	}, pc.Registry.EnabledWithOverlay(pc.Overlay))
}

// Select is the one-shot form: load the state and preview a single operation.
func (p *Previewer) Select(ctx context.Context, req PreviewRequest) (Selection, *PreviewContext, error) {
	pc, err := p.Load(ctx)
	if err != nil {
		return Selection{}, nil, err
	}
	sel, err := pc.Select(ctx, req)
	return sel, pc, err
}
