// Package workspace is the HTTP-handler home for the workspace domain.
// It exposes the generated Connect-RPC WorkspaceService (proto schema:
// packages/proto/schemas/web-console/v1/workspace). Domain types and the
// Store interface live in web-console/internal/workspace; this package
// owns the transport surface only.
package workspace

import (
	"context"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	workspaceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/workspace/workspace_v1connect"

	"web-console/internal/module"
	wsdomain "web-console/internal/workspace"
)

// Service is the seam the Connect handler depends on. The production
// implementation is Adapter (wires Store + EventEmitter).
type Service interface {
	GetLayout(ctx context.Context) (Layout, error)
	SaveLayout(ctx context.Context, activePane string, paneOrder []string) error
	UpdatePane(ctx context.Context, req UpdatePaneRequest) (Pane, error)
	DeletePane(ctx context.Context, sessionID string)
	CreateGroup(ctx context.Context, name, color string) (Group, error)
	UpdateGroup(ctx context.Context, req UpdateGroupRequest) (Group, error)
	DeleteGroup(ctx context.Context, id string)

	// Roles are named positions inside a group. They are additive: a group
	// with no roles behaves exactly as it did before roles existed, so a
	// caller that never touches these keeps the pre-roles behaviour.
	ListRoles(ctx context.Context, groupID string) ([]Role, error)
	CreateRole(ctx context.Context, req CreateRoleRequest) (Role, error)
	UpdateRole(ctx context.Context, req UpdateRoleRequest) (Role, error)
	DeleteRole(ctx context.Context, id string) error
}

// Transport types are aliases to the domain types so handlers and internal
// share one shape. No conversion code lives in this package.
type (
	Pane               = wsdomain.Pane
	Group              = wsdomain.Group
	Layout             = wsdomain.Layout
	UpdatePaneRequest  = wsdomain.UpdatePaneRequest
	UpdateGroupRequest = wsdomain.UpdateGroupRequest
	Role               = wsdomain.Role
	CreateRoleRequest  = wsdomain.CreateRoleRequest
	UpdateRoleRequest  = wsdomain.UpdateRoleRequest
)

// Module wires the workspace domain into the API server.
func Module(svc Service, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, connectHandler := workspaceconnect.NewWorkspaceServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "workspace",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}
