// DOC: docs/concepts/ARCHITECTURE.md#data-flow
// DOC: docs/internal/ERROR-SEMANTICS.md

package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// --- Workspace Layout Handlers ---

// LayoutResponse is the JSON shape returned by GET /api/v1/workspace/layout.
type LayoutResponse struct {
	ActivePane string           `json:"active_pane"`
	Panes      []*WorkspacePane `json:"panes"`
	Groups     []*TabGroup      `json:"groups"`
}

// handleGetLayout returns the full workspace layout: ordered panes and tab groups.
// GET /api/v1/workspace/layout
func (s *Server) handleGetLayout(w http.ResponseWriter, r *http.Request) {
	layout, err := s.workspace.GetLayout()
	if err != nil {
		log.Printf("get-layout [%s]: %v", getRequestID(r), err)
		writeAppError(w, errorCatalog["internal_error"])
		return
	}

	writeJSON(w, http.StatusOK, LayoutResponse{
		ActivePane: layout.ActivePane,
		Panes:      layout.Panes,
		Groups:     layout.Groups,
	})
}

// SaveLayoutRequest is the JSON body for PUT /api/v1/workspace/layout.
type SaveLayoutRequest struct {
	ActivePane string   `json:"active_pane"`
	PaneOrder  []string `json:"pane_order"`
}

// handleSaveLayout persists pane ordering and the active pane selection.
// PUT /api/v1/workspace/layout
func (s *Server) handleSaveLayout(w http.ResponseWriter, r *http.Request) {
	var req SaveLayoutRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if err := s.workspace.SavePaneOrder(req.ActivePane, req.PaneOrder); err != nil {
		log.Printf("save-layout [%s]: %v", getRequestID(r), err)
		writeAppError(w, errorCatalog["internal_error"])
		return
	}

	s.events.Emit(EventWorkspaceLayoutUpdated, "", map[string]string{
		"active_pane": req.ActivePane,
	})

	w.WriteHeader(http.StatusNoContent)
}

// --- Workspace Pane Handlers ---

// UpdatePaneRequest is the JSON body for PUT /api/v1/workspace/panes/{session_id}.
type UpdatePaneRequest struct {
	Name        *string `json:"name,omitempty"`
	HeaderColor *string `json:"header_color,omitempty"`
	ThemeID     *string `json:"theme_id,omitempty"`
	FontSize    *int    `json:"font_size,omitempty"`
	SortOrder   *int    `json:"sort_order,omitempty"`
	GroupID     *string `json:"group_id,omitempty"`
}

// handleUpdatePane creates or updates a single pane's metadata.
// PUT /api/v1/workspace/panes/{session_id}
func (s *Server) handleUpdatePane(w http.ResponseWriter, r *http.Request) {
	sessionID := mux.Vars(r)["session_id"]

	var req UpdatePaneRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	// Build pane from request, using defaults for unset fields
	pane := &WorkspacePane{
		SessionID:   sessionID,
		Name:        defaultPaneName,
		HeaderColor: defaultPaneHeaderColor,
		ThemeID:     defaultPaneThemeID,
		FontSize:    defaultPaneFontSize,
	}

	// Get existing pane to preserve unchanged fields
	layout, err := s.workspace.GetLayout()
	if err == nil {
		for _, p := range layout.Panes {
			if p.SessionID == sessionID {
				pane.Name = p.Name
				pane.HeaderColor = p.HeaderColor
				pane.ThemeID = p.ThemeID
				pane.FontSize = p.FontSize
				pane.SortOrder = p.SortOrder
				pane.GroupID = p.GroupID
				break
			}
		}
	}

	// Apply request overrides
	if req.Name != nil {
		pane.Name = *req.Name
	}
	if req.HeaderColor != nil {
		pane.HeaderColor = *req.HeaderColor
	}
	if req.ThemeID != nil {
		pane.ThemeID = *req.ThemeID
	}
	if req.FontSize != nil {
		pane.FontSize = *req.FontSize
	}
	if req.SortOrder != nil {
		pane.SortOrder = *req.SortOrder
	}
	if req.GroupID != nil {
		pane.GroupID = *req.GroupID
	}

	if err := s.workspace.UpsertPane(pane); err != nil {
		log.Printf("update-pane [%s]: %v", getRequestID(r), err)
		writeAppError(w, errorCatalog["internal_error"])
		return
	}

	s.events.Emit(EventPaneUpdated, sessionID, map[string]string{
		"name": pane.Name,
	})

	writeJSON(w, http.StatusOK, pane)
}

// handleDeletePane removes pane metadata. Idempotent: always returns 204.
// DELETE /api/v1/workspace/panes/{session_id}
func (s *Server) handleDeletePane(w http.ResponseWriter, r *http.Request) {
	sessionID := mux.Vars(r)["session_id"]

	if err := s.workspace.DeletePane(sessionID); err != nil {
		log.Printf("delete-pane [%s]: %v", getRequestID(r), err)
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Tab Group Handlers ---

// CreateGroupRequest is the JSON body for POST /api/v1/workspace/groups.
type CreateGroupRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// handleCreateGroup creates a new tab group.
// POST /api/v1/workspace/groups
func (s *Server) handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	var req CreateGroupRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	group, err := s.workspace.CreateGroup(req.Name, req.Color)
	if err != nil {
		log.Printf("create-group [%s]: %v", getRequestID(r), err)
		writeAppError(w, errorCatalog["internal_error"])
		return
	}

	s.events.Emit(EventTabGroupCreated, "", map[string]string{
		"group_id": group.ID,
		"name":     group.Name,
	})

	writeJSON(w, http.StatusCreated, group)
}

// UpdateGroupRequest is the JSON body for PUT /api/v1/workspace/groups/{id}.
type UpdateGroupRequest struct {
	Name        *string `json:"name,omitempty"`
	Color       *string `json:"color,omitempty"`
	IsCollapsed *bool   `json:"is_collapsed,omitempty"`
}

// handleUpdateGroup modifies a tab group's properties.
// PUT /api/v1/workspace/groups/{id}
func (s *Server) handleUpdateGroup(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var req UpdateGroupRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	group, err := s.workspace.UpdateGroup(id, req.Name, req.Color, req.IsCollapsed)
	if err != nil {
		if err.Error() == "group not found" {
			writeCatalogError(w, "group_not_found", "Tab group "+sanitizeID(id)+" not found")
			return
		}
		log.Printf("update-group [%s]: %v", getRequestID(r), err)
		writeAppError(w, errorCatalog["internal_error"])
		return
	}

	s.events.Emit(EventTabGroupUpdated, "", map[string]string{
		"group_id": group.ID,
		"name":     group.Name,
	})

	writeJSON(w, http.StatusOK, group)
}

// handleDeleteGroup removes a tab group. Idempotent: always returns 204.
// Panes in the group get group_id = NULL (via ON DELETE SET NULL in schema).
// DELETE /api/v1/workspace/groups/{id}
func (s *Server) handleDeleteGroup(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	removed, err := s.workspace.DeleteGroup(id)
	if err != nil {
		log.Printf("delete-group [%s]: %v", getRequestID(r), err)
	}
	if removed {
		s.events.Emit(EventTabGroupDeleted, "", map[string]string{
			"group_id": id,
		})
	}

	w.WriteHeader(http.StatusNoContent)
}
