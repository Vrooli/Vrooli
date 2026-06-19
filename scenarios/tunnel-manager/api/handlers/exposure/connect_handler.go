package exposure

import (
	"context"
	"log"
	"time"

	"tunnel-manager/internal/exposure"

	"connectrpc.com/connect"

	exposurev1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/exposure"
)

// Deps wires the seams the Connect exposure handler needs.
type Deps struct {
	Service exposure.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) Expose(ctx context.Context, req *connect.Request[exposurev1.ExposeRequest]) (*connect.Response[exposurev1.ExposeResponse], error) {
	lease, url, err := h.deps.Service.Expose(ctx, exposure.ExposeInput{
		Scenario:    req.Msg.Scenario,
		TTL:         time.Duration(req.Msg.TtlSeconds) * time.Second,
		RequestedBy: req.Msg.RequestedBy,
	})
	if err != nil {
		connectErr := exposure.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("exposure.Expose(%q): %v", req.Msg.Scenario, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&exposurev1.ExposeResponse{Lease: leaseToProto(lease), PublicUrl: url}), nil
}

func (h *connectHandler) ExtendLease(ctx context.Context, req *connect.Request[exposurev1.ExtendLeaseRequest]) (*connect.Response[exposurev1.ExtendLeaseResponse], error) {
	lease, err := h.deps.Service.ExtendLease(ctx, req.Msg.LeaseId, time.Duration(req.Msg.TtlSeconds)*time.Second)
	if err != nil {
		connectErr := exposure.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("exposure.ExtendLease(%q): %v", req.Msg.LeaseId, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&exposurev1.ExtendLeaseResponse{Lease: leaseToProto(lease)}), nil
}

func (h *connectHandler) RevokeLease(ctx context.Context, req *connect.Request[exposurev1.RevokeLeaseRequest]) (*connect.Response[exposurev1.RevokeLeaseResponse], error) {
	retracted, err := h.deps.Service.RevokeLease(ctx, req.Msg.LeaseId)
	if err != nil {
		connectErr := exposure.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("exposure.RevokeLease(%q): %v", req.Msg.LeaseId, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&exposurev1.RevokeLeaseResponse{Retracted: retracted}), nil
}

func (h *connectHandler) ListLeases(ctx context.Context, req *connect.Request[exposurev1.ListLeasesRequest]) (*connect.Response[exposurev1.ListLeasesResponse], error) {
	leases, err := h.deps.Service.ListLeases(ctx, leaseStatusFromProto(req.Msg.Status))
	if err != nil {
		h.deps.Logger.Printf("exposure.ListLeases: %v", err)
		return nil, exposure.ToConnectError(err)
	}
	resp := &exposurev1.ListLeasesResponse{Leases: make([]*exposurev1.Lease, 0, len(leases))}
	for _, l := range leases {
		resp.Leases = append(resp.Leases, leaseToProto(l))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ListExposures(ctx context.Context, _ *connect.Request[exposurev1.ListExposuresRequest]) (*connect.Response[exposurev1.ListExposuresResponse], error) {
	exposures, err := h.deps.Service.ListExposures(ctx)
	if err != nil {
		h.deps.Logger.Printf("exposure.ListExposures: %v", err)
		return nil, exposure.ToConnectError(err)
	}
	resp := &exposurev1.ListExposuresResponse{Exposures: make([]*exposurev1.Exposure, 0, len(exposures))}
	for _, e := range exposures {
		resp.Exposures = append(resp.Exposures, exposureToProto(e))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) IsExposed(ctx context.Context, req *connect.Request[exposurev1.IsExposedRequest]) (*connect.Response[exposurev1.IsExposedResponse], error) {
	exposed, url, err := h.deps.Service.IsExposed(ctx, req.Msg.Scenario)
	if err != nil {
		connectErr := exposure.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("exposure.IsExposed(%q): %v", req.Msg.Scenario, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&exposurev1.IsExposedResponse{Exposed: exposed, PublicUrl: url}), nil
}

func (h *connectHandler) Reconcile(ctx context.Context, _ *connect.Request[exposurev1.ReconcileRequest]) (*connect.Response[exposurev1.ReconcileResponse], error) {
	coreEnsured, reaped, err := h.deps.Service.Reconcile(ctx)
	if err != nil {
		h.deps.Logger.Printf("exposure.Reconcile: %v", err)
		return nil, exposure.ToConnectError(err)
	}
	return connect.NewResponse(&exposurev1.ReconcileResponse{
		CoreEnsured:  int32(coreEnsured),
		LeasesReaped: int32(reaped),
	}), nil
}
