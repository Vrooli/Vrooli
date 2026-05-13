// Package workspace is the HTTP-handler home for the workspace domain.
// It exposes the generated Connect-RPC WorkspaceService (proto schema:
// packages/proto/schemas/web-console/v1/workspace).
package workspace

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	workspaceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/workspace/workspace_v1connect"

	"web-console/internal/module"
)

// Service is the seam the Connect handler depends on. The concrete
// implementation lives in package main (adapts WorkspaceStore +
// EventBus to satisfy this interface).
type Service interface {
	GetLayout() (Layout, error)
	SaveLayout(activePane string, paneOrder []string) error
	UpdatePane(req UpdatePaneRequest) (Pane, error)
	DeletePane(sessionID string)
	CreateGroup(name, color string) (Group, error)
	UpdateGroup(req UpdateGroupRequest) (Group, error)
	DeleteGroup(id string)
}

// Pane is the transport-neutral pane shape.
type Pane struct {
	SessionID            string
	Name                 string
	HeaderColor          string
	ThemeID              string
	FontSize             int
	SortOrder            int
	GroupID              string
	SupportsMessagesView bool
	CreatedAt            string
	UpdatedAt            string
}

// Group is the transport-neutral group shape.
type Group struct {
	ID          string
	Name        string
	Color       string
	SortOrder   int
	IsCollapsed bool
	CreatedAt   string
	UpdatedAt   string
}

// Layout bundles the workspace's full state.
type Layout struct {
	ActivePane string
	Panes      []Pane
	Groups     []Group
}

// UpdatePaneRequest carries the session id plus optional field overrides.
// Each Has* flag indicates whether the paired field should be applied.
type UpdatePaneRequest struct {
	SessionID string

	Name                    string
	HasName                 bool
	HeaderColor             string
	HasHeaderColor          bool
	ThemeID                 string
	HasThemeID              bool
	FontSize                int
	HasFontSize             bool
	SortOrder               int
	HasSortOrder            bool
	GroupID                 string
	HasGroupID              bool
	SupportsMessagesView    bool
	HasSupportsMessagesView bool
}

// UpdateGroupRequest carries the group id plus optional field overrides.
type UpdateGroupRequest struct {
	ID             string
	Name           string
	HasName        bool
	Color          string
	HasColor       bool
	IsCollapsed    bool
	HasIsCollapsed bool
}

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
