// DOC: docs/concepts/ARCHITECTURE.md#data-flow
// DOC: docs/internal/ERROR_SEMANTICS.md

package main

import (
	"log"

	workspaceH "web-console/handlers/workspace"
	"web-console/internal/events"
)

// workspaceAdapter implements workspaceH.Service over the existing
// WorkspaceStore + EventBus. Validation rules (unknown scope, missing
// group, etc.) live here so the Connect handler stays a pure transport.
type workspaceAdapter struct {
	srv *Server
}

func newWorkspaceAdapter(s *Server) *workspaceAdapter {
	return &workspaceAdapter{srv: s}
}

func (a *workspaceAdapter) GetLayout() (workspaceH.Layout, error) {
	layout, err := a.srv.workspace.GetLayout()
	if err != nil {
		return workspaceH.Layout{}, err
	}
	return workspaceH.Layout{
		ActivePane: layout.ActivePane,
		Panes:      panesToTransport(layout.Panes),
		Groups:     groupsToTransport(layout.Groups),
	}, nil
}

func (a *workspaceAdapter) SaveLayout(activePane string, paneOrder []string) error {
	if err := a.srv.workspace.SavePaneOrder(activePane, paneOrder); err != nil {
		return err
	}
	a.srv.events.Emit(events.WorkspaceLayoutUpdated, "", map[string]string{
		"active_pane": activePane,
	})
	return nil
}

func (a *workspaceAdapter) UpdatePane(req workspaceH.UpdatePaneRequest) (workspaceH.Pane, error) {
	// Start from defaults; existing values preserve unset fields.
	pane := &WorkspacePane{
		SessionID:   req.SessionID,
		Name:        defaultPaneName,
		HeaderColor: defaultPaneHeaderColor,
		ThemeID:     defaultPaneThemeID,
		FontSize:    defaultPaneFontSize,
	}

	if layout, err := a.srv.workspace.GetLayout(); err == nil {
		for _, p := range layout.Panes {
			if p.SessionID == req.SessionID {
				pane.Name = p.Name
				pane.HeaderColor = p.HeaderColor
				pane.ThemeID = p.ThemeID
				pane.FontSize = p.FontSize
				pane.SortOrder = p.SortOrder
				pane.GroupID = p.GroupID
				pane.SupportsMessagesView = p.SupportsMessagesView
				break
			}
		}
	}

	if req.HasName {
		pane.Name = req.Name
	}
	if req.HasHeaderColor {
		pane.HeaderColor = req.HeaderColor
	}
	if req.HasThemeID {
		pane.ThemeID = req.ThemeID
	}
	if req.HasFontSize {
		pane.FontSize = req.FontSize
	}
	if req.HasSortOrder {
		pane.SortOrder = req.SortOrder
	}
	if req.HasGroupID {
		pane.GroupID = req.GroupID
	}
	if req.HasSupportsMessagesView {
		pane.SupportsMessagesView = req.SupportsMessagesView
	}

	if err := a.srv.workspace.UpsertPane(pane); err != nil {
		return workspaceH.Pane{}, err
	}

	a.srv.events.Emit(events.PaneUpdated, req.SessionID, map[string]string{
		"name": pane.Name,
	})

	return paneToTransport(pane), nil
}

func (a *workspaceAdapter) DeletePane(sessionID string) {
	if err := a.srv.workspace.DeletePane(sessionID); err != nil {
		log.Printf("workspace.DeletePane: %v", err)
	}
}

func (a *workspaceAdapter) CreateGroup(name, color string) (workspaceH.Group, error) {
	g, err := a.srv.workspace.CreateGroup(name, color)
	if err != nil {
		return workspaceH.Group{}, err
	}
	a.srv.events.Emit(events.TabGroupCreated, "", map[string]string{
		"group_id": g.ID,
		"name":     g.Name,
	})
	return groupToTransport(g), nil
}

func (a *workspaceAdapter) UpdateGroup(req workspaceH.UpdateGroupRequest) (workspaceH.Group, error) {
	var name, color *string
	var collapsed *bool
	if req.HasName {
		name = &req.Name
	}
	if req.HasColor {
		color = &req.Color
	}
	if req.HasIsCollapsed {
		collapsed = &req.IsCollapsed
	}

	g, err := a.srv.workspace.UpdateGroup(req.ID, name, color, collapsed)
	if err != nil {
		if err.Error() == "group not found" {
			return workspaceH.Group{}, workspaceH.ErrGroupNotFound
		}
		return workspaceH.Group{}, err
	}
	a.srv.events.Emit(events.TabGroupUpdated, "", map[string]string{
		"group_id": g.ID,
		"name":     g.Name,
	})
	return groupToTransport(g), nil
}

func (a *workspaceAdapter) DeleteGroup(id string) {
	removed, err := a.srv.workspace.DeleteGroup(id)
	if err != nil {
		log.Printf("workspace.DeleteGroup: %v", err)
	}
	if removed {
		a.srv.events.Emit(events.TabGroupDeleted, "", map[string]string{
			"group_id": id,
		})
	}
}

func paneToTransport(p *WorkspacePane) workspaceH.Pane {
	return workspaceH.Pane{
		SessionID:            p.SessionID,
		Name:                 p.Name,
		HeaderColor:          p.HeaderColor,
		ThemeID:              p.ThemeID,
		FontSize:             p.FontSize,
		SortOrder:            p.SortOrder,
		GroupID:              p.GroupID,
		SupportsMessagesView: p.SupportsMessagesView,
		CreatedAt:            p.CreatedAt,
		UpdatedAt:            p.UpdatedAt,
	}
}

func panesToTransport(in []*WorkspacePane) []workspaceH.Pane {
	out := make([]workspaceH.Pane, 0, len(in))
	for _, p := range in {
		out = append(out, paneToTransport(p))
	}
	return out
}

func groupToTransport(g *TabGroup) workspaceH.Group {
	return workspaceH.Group{
		ID:          g.ID,
		Name:        g.Name,
		Color:       g.Color,
		SortOrder:   g.SortOrder,
		IsCollapsed: g.IsCollapsed,
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
	}
}

func groupsToTransport(in []*TabGroup) []workspaceH.Group {
	out := make([]workspaceH.Group, 0, len(in))
	for _, g := range in {
		out = append(out, groupToTransport(g))
	}
	return out
}
