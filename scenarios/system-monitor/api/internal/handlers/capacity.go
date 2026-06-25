package handlers

// DOC: docs/reference/api-endpoints.md#capacity

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"connectrpc.com/connect"
	engine "github.com/vrooli/vrooli/internal/capacity"
	capacitypb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/capacity"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/apierrors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/convert"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/httputil"
)

// CapacityHandler exposes the platform capacity ledger + policy levers as the
// system-monitor governance surface. Reads are advisory observability; the only
// mutation is policy (claim mutation flows through the broker, not here).
type CapacityHandler struct {
	log      *slog.Logger
	capacity CapacityProvider
}

// NewCapacityHandler creates a capacity handler.
func NewCapacityHandler(capacity CapacityProvider, log *slog.Logger) *CapacityHandler {
	return &CapacityHandler{log: log, capacity: capacity}
}

// GetCapacityOverview handles the typed Connect-RPC capacity overview contract.
func (h *CapacityHandler) GetCapacityOverview(ctx context.Context, _ *connect.Request[capacitypb.GetCapacityOverviewRequest]) (*connect.Response[capacitypb.GetCapacityOverviewResponse], error) {
	overview, err := h.capacity.Overview(ctx)
	if err != nil {
		return nil, connectError(apierrors.Internal("failed to read capacity overview", err))
	}

	return connect.NewResponse(&capacitypb.GetCapacityOverviewResponse{
		Success:          true,
		Gpus:             convert.GpuContentionsToProto(overview.GPUs),
		Claims:           convert.CapacityClaimsToProto(overview.Claims),
		SensingAvailable: overview.SensingAvailable,
		Warnings:         overview.Warnings,
	}), nil
}

// ListCapacityClaims handles the typed Connect-RPC capacity claim listing contract.
func (h *CapacityHandler) ListCapacityClaims(ctx context.Context, req *connect.Request[capacitypb.ListCapacityClaimsRequest]) (*connect.Response[capacitypb.ListCapacityClaimsResponse], error) {
	claims, err := h.capacity.ListClaims(ctx, req.Msg.GetOwnerId(), req.Msg.GetActiveOnly())
	if err != nil {
		return nil, connectError(apierrors.Internal("failed to list capacity claims", err))
	}

	return connect.NewResponse(&capacitypb.ListCapacityClaimsResponse{
		Success: true,
		Claims:  convert.CapacityClaimsToProto(claims),
	}), nil
}

// ReconcileCapacity handles the typed Connect-RPC reconciliation contract.
func (h *CapacityHandler) ReconcileCapacity(ctx context.Context, _ *connect.Request[capacitypb.ReconcileCapacityRequest]) (*connect.Response[capacitypb.ReconcileCapacityResponse], error) {
	findings, err := h.capacity.Reconcile(ctx)
	if err != nil {
		return nil, connectError(apierrors.Unavailable("capacity reconciliation (host sensing)"))
	}

	return connect.NewResponse(&capacitypb.ReconcileCapacityResponse{
		Success:  true,
		Findings: convert.CapacityFindingsToProto(findings),
	}), nil
}

// GetCapacityPolicy handles the typed Connect-RPC policy listing contract.
func (h *CapacityHandler) GetCapacityPolicy(ctx context.Context, _ *connect.Request[capacitypb.GetCapacityPolicyRequest]) (*connect.Response[capacitypb.GetCapacityPolicyResponse], error) {
	entries, err := h.capacity.Policy(ctx)
	if err != nil {
		return nil, connectError(apierrors.Internal("failed to read capacity policy", err))
	}

	return connect.NewResponse(&capacitypb.GetCapacityPolicyResponse{
		Success: true,
		Levers:  convert.PolicyLeversToProto(entries),
	}), nil
}

// SetCapacityPolicy handles the typed Connect-RPC policy mutation contract.
func (h *CapacityHandler) SetCapacityPolicy(ctx context.Context, req *connect.Request[capacitypb.SetCapacityPolicyRequest]) (*connect.Response[capacitypb.SetCapacityPolicyResponse], error) {
	if req.Msg.GetKey() == "" {
		return nil, connectError(apierrors.Validation("key", "is required"))
	}

	entries, err := h.capacity.SetPolicy(ctx, req.Msg.GetKey(), req.Msg.GetValue())
	if err != nil {
		if errors.Is(err, engine.ErrInvalidClaim) {
			return nil, connectError(apierrors.Validation("value", err.Error()))
		}
		return nil, connectError(apierrors.Internal("failed to update capacity policy", err))
	}

	return connect.NewResponse(&capacitypb.SetCapacityPolicyResponse{
		Success: true,
		Levers:  convert.PolicyLeversToProto(entries),
	}), nil
}

// Overview handles GET /api/v1/capacity/overview — per-GPU contention + claims.
func (h *CapacityHandler) Overview(w http.ResponseWriter, r *http.Request) {
	overview, err := h.capacity.Overview(r.Context())
	if err != nil {
		httputil.HandleError(w, h.log, r, apierrors.Internal("failed to read capacity overview", err))
		return
	}
	httputil.SafeProtoJSON(w, h.log, r, &capacitypb.GetCapacityOverviewResponse{
		Success:          true,
		Gpus:             convert.GpuContentionsToProto(overview.GPUs),
		Claims:           convert.CapacityClaimsToProto(overview.Claims),
		SensingAvailable: overview.SensingAvailable,
		Warnings:         overview.Warnings,
	})
}

// ListClaims handles GET /api/v1/capacity/claims — the claim ledger.
func (h *CapacityHandler) ListClaims(w http.ResponseWriter, r *http.Request) {
	ownerID := r.URL.Query().Get("owner_id")
	activeOnly := false
	if v := r.URL.Query().Get("active_only"); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			httputil.HandleError(w, h.log, r, apierrors.Validation("active_only", "must be a boolean"))
			return
		}
		activeOnly = parsed
	}

	claims, err := h.capacity.ListClaims(r.Context(), ownerID, activeOnly)
	if err != nil {
		httputil.HandleError(w, h.log, r, apierrors.Internal("failed to list capacity claims", err))
		return
	}
	httputil.SafeProtoJSON(w, h.log, r, &capacitypb.ListCapacityClaimsResponse{
		Success: true,
		Claims:  convert.CapacityClaimsToProto(claims),
	})
}

// Reconcile handles GET /api/v1/capacity/reconcile — unclaimed-consumer warnings.
func (h *CapacityHandler) Reconcile(w http.ResponseWriter, r *http.Request) {
	findings, err := h.capacity.Reconcile(r.Context())
	if err != nil {
		httputil.HandleError(w, h.log, r, apierrors.Unavailable("capacity reconciliation (host sensing)"))
		return
	}
	httputil.SafeProtoJSON(w, h.log, r, &capacitypb.ReconcileCapacityResponse{
		Success:  true,
		Findings: convert.CapacityFindingsToProto(findings),
	})
}

// GetPolicy handles GET /api/v1/capacity/policy — the tunable levers.
func (h *CapacityHandler) GetPolicy(w http.ResponseWriter, r *http.Request) {
	entries, err := h.capacity.Policy(r.Context())
	if err != nil {
		httputil.HandleError(w, h.log, r, apierrors.Internal("failed to read capacity policy", err))
		return
	}
	httputil.SafeProtoJSON(w, h.log, r, &capacitypb.GetCapacityPolicyResponse{
		Success: true,
		Levers:  convert.PolicyLeversToProto(entries),
	})
}

// SetPolicy handles POST /api/v1/capacity/policy — update one lever.
func (h *CapacityHandler) SetPolicy(w http.ResponseWriter, r *http.Request) {
	var req capacitypb.SetCapacityPolicyRequest
	if err := httputil.DecodeProtoJSON(r, &req); err != nil {
		httputil.HandleError(w, h.log, r, apierrors.Validation("body", "Invalid JSON payload"))
		return
	}
	if req.GetKey() == "" {
		httputil.HandleError(w, h.log, r, apierrors.Validation("key", "is required"))
		return
	}

	entries, err := h.capacity.SetPolicy(r.Context(), req.GetKey(), req.GetValue())
	if err != nil {
		if errors.Is(err, engine.ErrInvalidClaim) {
			httputil.HandleError(w, h.log, r, apierrors.Validation("value", err.Error()))
			return
		}
		httputil.HandleError(w, h.log, r, apierrors.Internal("failed to update capacity policy", err))
		return
	}
	httputil.SafeProtoJSON(w, h.log, r, &capacitypb.SetCapacityPolicyResponse{
		Success: true,
		Levers:  convert.PolicyLeversToProto(entries),
	})
}
