package audit

import (
	"context"
	"log"

	internalaudit "tunnel-manager/internal/audit"

	"connectrpc.com/connect"

	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/audit"
)

// Deps wires the seams the Connect audit handler needs.
type Deps struct {
	Service internalaudit.Service
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

func (h *connectHandler) RunAudit(ctx context.Context, _ *connect.Request[auditv1.RunAuditRequest]) (*connect.Response[auditv1.RunAuditResponse], error) {
	results, err := h.deps.Service.RunAudit(ctx)
	if err != nil {
		h.deps.Logger.Printf("audit.RunAudit: %v", err)
		return nil, internalaudit.ToConnectError(err)
	}
	resp := &auditv1.RunAuditResponse{
		Results:        make([]*auditv1.PortAuditResult, 0, len(results)),
		ViolationCount: int32(internalaudit.ViolationCount(results)),
	}
	for _, r := range results {
		resp.Results = append(resp.Results, domainToProto(r))
	}
	return connect.NewResponse(resp), nil
}
