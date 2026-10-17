package reset

import (
	"context"
	"errors"
	"landing-page-react-vite-api/internal/adminreset"
	"log"
	"time"

	"connectrpc.com/connect"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"

	internaladmin "landing-page-react-vite-api/internal/admin"
)

// Deps wires the AdminReset Connect handler.
type Deps struct {
	Service *adminreset.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the AdminResetService Connect handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ResetDemoData(ctx context.Context, _ *connect.Request[landingv1.ResetDemoDataRequest]) (*connect.Response[landingv1.ResetDemoDataResponse], error) {
	if !internaladmin.ResetEnabled() {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("demo reset disabled"))
	}
	if err := h.deps.Service.Reset(ctx); err != nil {
		h.deps.Logger.Printf("reset.ResetDemoData: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&landingv1.ResetDemoDataResponse{
		Reset_:    true,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}), nil
}
