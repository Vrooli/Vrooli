package models

import (
	"context"
	"errors"
	"log"

	internalcaps "image-tools/internal/capabilities"
	internalmodels "image-tools/internal/models"

	"connectrpc.com/connect"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
)

// Deps wires the seams the Connect models handler needs.
type Deps struct {
	// Registry is the loaded, validated seed catalog.
	Registry *internalmodels.Registry
	// Store persists the runtime enabled-state overlay.
	Store *internalmodels.Store
	// Probe reports host hardware facts for SelectModel's hardware-fit preview.
	Probe  internalcaps.Probe
	Logger *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the ModelsService handler. Registry, Store, and Probe
// are required (a nil dep is a wiring bug surfaced as an internal error).
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) overlay(ctx context.Context) (map[string]bool, error) {
	if h.deps.Store == nil {
		return nil, nil
	}
	return h.deps.Store.LoadOverlay(ctx)
}

func (h *connectHandler) ListModels(ctx context.Context, req *connect.Request[modelsv1.ListModelsRequest]) (*connect.Response[modelsv1.ListModelsResponse], error) {
	overlay, err := h.overlay(ctx)
	if err != nil {
		h.deps.Logger.Printf("models.ListModels overlay: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("load model state"))
	}

	op := req.Msg.GetOperation()
	var entries []internalmodels.Model
	if op != "" {
		if !h.deps.Registry.IsOperation(op) {
			return nil, connect.NewError(connect.CodeInvalidArgument, internalmodels.ErrUnknownOperation)
		}
		entries = h.deps.Registry.ForOperation(op)
	} else {
		entries = h.deps.Registry.Models()
	}

	resp := &modelsv1.ListModelsResponse{Models: make([]*modelsv1.Model, 0, len(entries))}
	for _, m := range entries {
		resp.Models = append(resp.Models, domainToProto(m, internalmodels.EffectiveEnabled(m, overlay)))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetModel(ctx context.Context, req *connect.Request[modelsv1.GetModelRequest]) (*connect.Response[modelsv1.GetModelResponse], error) {
	m, ok := h.deps.Registry.ByID(req.Msg.GetId())
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("model not found"))
	}
	overlay, err := h.overlay(ctx)
	if err != nil {
		h.deps.Logger.Printf("models.GetModel overlay: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("load model state"))
	}
	return connect.NewResponse(&modelsv1.GetModelResponse{
		Model: domainToProto(m, internalmodels.EffectiveEnabled(m, overlay)),
	}), nil
}

func (h *connectHandler) ListOperations(_ context.Context, _ *connect.Request[modelsv1.ListOperationsRequest]) (*connect.Response[modelsv1.ListOperationsResponse], error) {
	return connect.NewResponse(&modelsv1.ListOperationsResponse{
		Operations: h.deps.Registry.Operations(),
	}), nil
}

func (h *connectHandler) SelectModel(ctx context.Context, req *connect.Request[modelsv1.SelectModelRequest]) (*connect.Response[modelsv1.SelectModelResponse], error) {
	host, err := h.deps.Probe.Inventory(ctx)
	if err != nil {
		h.deps.Logger.Printf("models.SelectModel probe: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("probe host hardware"))
	}
	overlay, err := h.overlay(ctx)
	if err != nil {
		h.deps.Logger.Printf("models.SelectModel overlay: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("load model state"))
	}

	sel, err := h.deps.Registry.Select(internalmodels.SelectRequest{
		Operation:  req.Msg.GetOperation(),
		Host:       host,
		OverrideID: req.Msg.GetOverrideId(),
	}, h.deps.Registry.EnabledWithOverlay(overlay))
	if err != nil {
		return nil, selectError(err)
	}

	return connect.NewResponse(&modelsv1.SelectModelResponse{
		Model:     domainToProto(sel.Model, internalmodels.EffectiveEnabled(sel.Model, overlay)),
		GpuViable: sel.GPUViable,
		Reason:    sel.Reason,
		Warnings:  sel.Warnings,
	}), nil
}

func (h *connectHandler) SetModelEnabled(ctx context.Context, req *connect.Request[modelsv1.SetModelEnabledRequest]) (*connect.Response[modelsv1.SetModelEnabledResponse], error) {
	id := req.Msg.GetId()
	m, ok := h.deps.Registry.ByID(id)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("model not found"))
	}
	if h.deps.Store == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("model state store unavailable"))
	}
	if err := h.deps.Store.SetEnabled(ctx, id, req.Msg.GetEnabled()); err != nil {
		h.deps.Logger.Printf("models.SetModelEnabled: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("persist model state"))
	}
	return connect.NewResponse(&modelsv1.SetModelEnabledResponse{
		Model: domainToProto(m, req.Msg.GetEnabled()),
	}), nil
}

func (h *connectHandler) ListBlocklist(_ context.Context, _ *connect.Request[modelsv1.ListBlocklistRequest]) (*connect.Response[modelsv1.ListBlocklistResponse], error) {
	entries := h.deps.Registry.Blocklist()
	resp := &modelsv1.ListBlocklistResponse{Entries: make([]*modelsv1.BlocklistEntry, 0, len(entries))}
	for _, b := range entries {
		resp.Entries = append(resp.Entries, blocklistToProto(b))
	}
	return connect.NewResponse(resp), nil
}

// selectError maps selector errors to actionable Connect codes.
func selectError(err error) error {
	switch {
	case errors.Is(err, internalmodels.ErrUnknownOperation),
		errors.Is(err, internalmodels.ErrOverrideInvalid):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, internalmodels.ErrNoEnabledModel),
		errors.Is(err, internalmodels.ErrNotRunnable):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
