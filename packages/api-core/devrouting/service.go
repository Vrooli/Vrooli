// Package devrouting wires the dev-only RoutingService proto contract to a
// RoutedDB, so an external orchestrator (test-genie) can install and clear a
// runtime test DB pool on a live scenario.
//
// The service is gated by projectmeta.IsDevelopment(): Register is a no-op in
// production mode, so the RPC path returns 404 to any caller. The
// VROOLI_TEST_MODE_FORCE_ENABLE escape hatch from package apihttp also opens
// this service.
package devrouting

import (
	"context"
	"errors"
	"net/http"
	"os"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/projectmeta"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing"
	"github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing/routing_v1connect"
)

// Service implements the generated routing_v1connect.RoutingServiceHandler
// by forwarding calls into a *database.RoutedDB.
type Service struct {
	db *database.RoutedDB
}

// NewService returns a Service bound to db.
func NewService(db *database.RoutedDB) *Service {
	return &Service{db: db}
}

// InstallTestPool implements routing_v1connect.RoutingServiceHandler.
//
// Lease conflicts map to connect.CodeAlreadyExists; the response carries
// the active lease so callers can surface a useful error. Other failures
// map to CodeInternal.
func (s *Service) InstallTestPool(ctx context.Context, req *connect.Request[routingv1.InstallTestPoolRequest]) (*connect.Response[routingv1.InstallTestPoolResponse], error) {
	leaseID := req.Msg.GetLeaseId()
	if err := s.db.InstallTestPool(ctx, req.Msg.GetDsn(), leaseID); err != nil {
		var conflict *database.ErrLeaseConflict
		if errors.As(err, &conflict) {
			return nil, connect.NewError(connect.CodeAlreadyExists, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&routingv1.InstallTestPoolResponse{ActiveLeaseId: leaseID}), nil
}

// ClearTestPool implements routing_v1connect.RoutingServiceHandler.
//
// Lease mismatches map to connect.CodeFailedPrecondition.
func (s *Service) ClearTestPool(ctx context.Context, req *connect.Request[routingv1.ClearTestPoolRequest]) (*connect.Response[routingv1.ClearTestPoolResponse], error) {
	if err := s.db.ClearTestPool(req.Msg.GetLeaseId()); err != nil {
		var mismatch *database.ErrLeaseMismatch
		if errors.As(err, &mismatch) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&routingv1.ClearTestPoolResponse{}), nil
}

// Mux is the minimal interface satisfied by *http.ServeMux and the
// gorilla/mux router so callers can pass whichever router they already use.
type Mux interface {
	Handle(pattern string, handler http.Handler)
}

// Register mounts the RoutingService Connect handler onto mux, but only if the
// project is running in development mode (or the
// VROOLI_TEST_MODE_FORCE_ENABLE escape hatch is set). In production mode
// Register is a no-op and the RPC path returns whatever 404 the underlying
// mux serves for unknown paths.
//
// Register returns true when it actually mounted the handler. Production
// callers can use the return value to log/observe that the dev surface is
// disabled.
func Register(mux Mux, db *database.RoutedDB, opts ...connect.HandlerOption) bool {
	if !enabled() {
		return false
	}
	path, handler := routing_v1connect.NewRoutingServiceHandler(NewService(db), opts...)
	mux.Handle(path, handler)
	return true
}

func enabled() bool {
	return projectmeta.IsDevelopment() || os.Getenv(apihttp.TestModeForceEnableEnv) == "1"
}
