package workspace

import (
	"context"
	"errors"
	"log"

	"web-console/internal/events"
	wsdomain "web-console/internal/workspace"
)

// EventEmitter is the events-bus seam (subset of internal/events.Logger).
type EventEmitter interface {
	Emit(eventType, sessionID string, details map[string]string)
}

// Adapter is the production Service implementation. Constructed in
// api/main.go with typed deps and passed to Module.
type Adapter struct {
	Store  wsdomain.Store
	Events EventEmitter
	Logger *log.Logger
}

func (a *Adapter) logger() *log.Logger {
	if a.Logger != nil {
		return a.Logger
	}
	return log.Default()
}

func (a *Adapter) GetLayout(ctx context.Context) (Layout, error) {
	return a.Store.GetLayout(ctx)
}

func (a *Adapter) SaveLayout(ctx context.Context, activePane string, paneOrder []string) error {
	if err := a.Store.SavePaneOrder(ctx, activePane, paneOrder); err != nil {
		return err
	}
	a.Events.Emit(events.WorkspaceLayoutUpdated, "", map[string]string{
		"active_pane": activePane,
	})
	return nil
}

func (a *Adapter) UpdatePane(ctx context.Context, req UpdatePaneRequest) (Pane, error) {
	pane := Pane{
		SessionID:   req.SessionID,
		Name:        wsdomain.DefaultPaneName,
		HeaderColor: wsdomain.DefaultPaneHeaderColor,
		ThemeID:     wsdomain.DefaultPaneThemeID,
		FontSize:    wsdomain.DefaultPaneFontSize,
	}

	if layout, err := a.Store.GetLayout(ctx); err == nil {
		for _, p := range layout.Panes {
			if p.SessionID == req.SessionID {
				pane.Name = p.Name
				pane.HeaderColor = p.HeaderColor
				pane.ThemeID = p.ThemeID
				pane.FontSize = p.FontSize
				pane.SortOrder = p.SortOrder
				pane.GroupID = p.GroupID
				pane.SupportsMessagesView = p.SupportsMessagesView
				pane.ManuallyUnread = p.ManuallyUnread
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
	if req.HasManuallyUnread {
		pane.ManuallyUnread = req.ManuallyUnread
	}

	if err := a.Store.UpsertPane(ctx, pane); err != nil {
		return Pane{}, err
	}

	a.Events.Emit(events.PaneUpdated, req.SessionID, map[string]string{
		"name": pane.Name,
	})

	return pane, nil
}

func (a *Adapter) DeletePane(ctx context.Context, sessionID string) {
	if err := a.Store.DeletePane(ctx, sessionID); err != nil {
		a.logger().Printf("workspace.DeletePane: %v", err)
	}
}

func (a *Adapter) CreateGroup(ctx context.Context, name, color string) (Group, error) {
	g, err := a.Store.CreateGroup(ctx, name, color)
	if err != nil {
		return Group{}, err
	}
	a.Events.Emit(events.TabGroupCreated, "", map[string]string{
		"group_id": g.ID,
		"name":     g.Name,
	})
	return g, nil
}

func (a *Adapter) UpdateGroup(ctx context.Context, req UpdateGroupRequest) (Group, error) {
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

	g, err := a.Store.UpdateGroup(ctx, req.ID, name, color, collapsed)
	if err != nil {
		if errors.Is(err, wsdomain.ErrGroupNotFound) {
			return Group{}, ErrGroupNotFound
		}
		return Group{}, err
	}
	a.Events.Emit(events.TabGroupUpdated, "", map[string]string{
		"group_id": g.ID,
		"name":     g.Name,
	})
	return g, nil
}

func (a *Adapter) DeleteGroup(ctx context.Context, id string) {
	removed, err := a.Store.DeleteGroup(ctx, id)
	if err != nil {
		a.logger().Printf("workspace.DeleteGroup: %v", err)
	}
	if removed {
		a.Events.Emit(events.TabGroupDeleted, "", map[string]string{
			"group_id": id,
		})
	}
}

// Roles delegate straight to the store. Unlike panes and groups they emit no
// workspace events: a role is a slot definition, not a live surface, and the
// session it points at already emits its own lifecycle events. Adding a
// second event stream for the same state change would double-count.

func (a *Adapter) ListRoles(ctx context.Context, groupID string) ([]Role, error) {
	return a.Store.ListRoles(ctx, groupID)
}

func (a *Adapter) CreateRole(ctx context.Context, req CreateRoleRequest) (Role, error) {
	return a.Store.CreateRole(ctx, req)
}

func (a *Adapter) UpdateRole(ctx context.Context, req UpdateRoleRequest) (Role, error) {
	return a.Store.UpdateRole(ctx, req)
}

// DeleteRole is idempotent, matching the store: deleting an id that is
// already gone is the state the caller asked for, not an error.
func (a *Adapter) DeleteRole(ctx context.Context, id string) error {
	_, err := a.Store.DeleteRole(ctx, id)
	return err
}
