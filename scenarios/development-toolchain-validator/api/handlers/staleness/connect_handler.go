package staleness

import (
	"context"
	"log"

	stalenessdom "development-toolchain-validator/internal/staleness"

	"connectrpc.com/connect"

	stalenessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/staleness"
)

// Deps wires the seams the Connect staleness handler needs.
type Deps struct {
	Service stalenessdom.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect handler for StalenessService.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ListStale(ctx context.Context, _ *connect.Request[stalenessv1.ListStaleRequest]) (*connect.Response[stalenessv1.ListStaleResponse], error) {
	entries, err := h.deps.Service.ListStale(ctx)
	if err != nil {
		h.deps.Logger.Printf("staleness.ListStale: %v", err)
		return nil, stalenessdom.ToConnectError(err)
	}
	resp := &stalenessv1.ListStaleResponse{Entries: make([]*stalenessv1.StaleEntry, 0, len(entries))}
	for _, e := range entries {
		resp.Entries = append(resp.Entries, domainToProto(e))
	}
	return connect.NewResponse(resp), nil
}
