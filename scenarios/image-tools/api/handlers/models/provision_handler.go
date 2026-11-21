package models

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
	internalai "image-tools/internal/ai"
	"image-tools/internal/jobs"
	internalmodels "image-tools/internal/models"
	"image-tools/internal/provision"
)

func (h *connectHandler) ProvisionModels(ctx context.Context, req *connect.Request[modelsv1.ProvisionModelsRequest]) (*connect.Response[modelsv1.ProvisionModelsResponse], error) {
	if h.deps.Installer == nil || h.deps.Jobs == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("model provisioning unavailable"))
	}
	ids := req.Msg.GetModelIds()
	plan := provision.Plan{Mode: provision.PlanExplicit}
	if len(ids) == 0 {
		overlay, err := h.overlay(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		plan, err = h.hostFitProvisionPlan(ctx, overlay)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if len(plan.CoveredOperations)+len(plan.UncoveredOperations) == 0 {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("no enabled capability operations are available to provision"))
		}
		ids = append(ids, plan.SelectedModelIDs...)
	}
	selected := make([]string, 0, len(ids))
	seen := map[string]bool{}
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if seen[id] {
			continue
		}
		m, err := h.deps.Installer.Resolve(ctx, id)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		if m.CapabilityLabels.CommercialUse != internalmodels.CommercialUseYes || m.Backend == internalmodels.BackendOpenRouter {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("model %s is not a verified commercial-clean local model", id))
		}
		seen[id] = true
		selected = append(selected, id)
	}
	if len(selected) == 0 {
		if plan.Mode != provision.PlanHostFit {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("no enabled local models"))
		}
	}
	plan.SelectedModelIDs = append([]string(nil), selected...)
	payload, err := json.Marshal(provision.Request{ModelIDs: selected, Plan: plan})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	job, err := h.deps.Jobs.Submit(ctx, jobs.Spec{Operation: provision.Operation, Lane: jobs.LaneCPU, Payload: payload})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&modelsv1.ProvisionModelsResponse{JobId: job.ID, ModelIds: selected}), nil
}

// hostFitProvisionPlan is shared by provisioning and DoctorCatalog so both
// surfaces use the same enabled-capability scope, host facts, and backend
// availability decisions.
func (h *connectHandler) hostFitProvisionPlan(ctx context.Context, overlay map[string]bool) (provision.Plan, error) {
	custom, err := h.customModels(ctx)
	if err != nil {
		return provision.Plan{}, err
	}
	host, err := h.deps.Probe.Inventory(ctx)
	if err != nil {
		return provision.Plan{}, fmt.Errorf("inspect host for provisioning plan: %w", err)
	}
	candidates := append(h.deps.Registry.Models(), custom...)
	enabledByID := make(map[string]bool, len(candidates))
	for _, m := range candidates {
		enabledByID[m.ID] = internalmodels.EffectiveEnabled(m, overlay)
	}
	enabled := func(id string) bool { return enabledByID[id] }
	operations := provision.EnabledOperations(candidates, h.deps.Registry.SortedOperations(), enabled)
	toolCache := map[string]*toolTier{}
	return provision.HostFitPlan(candidates, operations, host, enabled, func(m internalmodels.Model, operation string) string {
		if h.deps.Backends == nil {
			return "backend_registry_unavailable"
		}
		for _, provider := range h.deps.Backends.Providers(operation) {
			if provider.Name() != m.Backend || !provider.Standalone() || provider.IsCloud() {
				continue
			}
			if tool, hasTool := internalai.HostToolForProvider(m.Backend); hasTool {
				tier := h.toolTierFor(ctx, tool, toolCache)
				if tier.tier == "auto" {
					return ""
				}
				return fmt.Sprintf("host_tool_%s:%s", tier.tier, tool)
			}
			if !m.RequiresWeights() || provider.Available(ctx) {
				return ""
			}
			return "backend_unavailable_without_host_tool_provision_route"
		}
		return "no_registered_local_backend_for_operation"
	}), nil
}
