package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

func registerRoutes(mux *http.ServeMux, manager *sessionManager, metrics *metricsRegistry, ws *workspace) {
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"note":   "web console must run behind authenticated proxy",
		})
	})

	mux.HandleFunc("/metrics", metrics.serveHTTP)

	// Workspace endpoints
	workspaceHandler := manager.makeProxyGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetWorkspace(w, r, ws)
		case http.MethodPut:
			handleUpdateWorkspace(w, r, ws)
		case http.MethodPatch:
			handlePatchWorkspace(w, r, ws)
		default:
			w.Header().Set("Allow", "GET, PUT, PATCH")
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}))
	mux.Handle("/api/v1/workspace", workspaceHandler)

	workspaceStreamHandler := manager.makeProxyGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		handleWorkspaceWebSocket(w, r, ws, metrics)
	}))
	mux.Handle("/api/v1/workspace/stream", workspaceStreamHandler)

	workspaceTabsHandler := manager.makeProxyGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleCreateTab(w, r, ws)
		default:
			w.Header().Set("Allow", "POST")
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}))
	mux.Handle("/api/v1/workspace/tabs", workspaceTabsHandler)

	workspaceTabDetailHandler := manager.makeProxyGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		trimmed := strings.TrimPrefix(r.URL.Path, "/api/v1/workspace/tabs/")
		if trimmed == "" {
			writeJSONError(w, http.StatusBadRequest, "tab id required")
			return
		}
		parts := strings.Split(trimmed, "/")
		tabID := parts[0]

		switch r.Method {
		case http.MethodPatch:
			handleUpdateTab(w, r, ws, tabID)
		case http.MethodDelete:
			handleDeleteTab(w, r, ws, tabID)
		default:
			w.Header().Set("Allow", "PATCH, DELETE")
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}))
	mux.Handle("/api/v1/workspace/tabs/", workspaceTabDetailHandler)

	// AI command generation endpoint
	generateCommandHandler := manager.makeProxyGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		handleGenerateCommand(w, r)
	}))
	mux.Handle("/api/generate-command", generateCommandHandler)

	// Session endpoints
	sessionsHandler := manager.makeProxyGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleCreateSession(w, r, manager, ws)
		case http.MethodGet:
			writeJSON(w, http.StatusOK, manager.listSummaries())
		case http.MethodDelete:
			terminated := manager.deleteAllSessions(reasonClientRequested)
			writeJSON(w, http.StatusOK, map[string]any{
				"status":     "terminating_all",
				"terminated": terminated,
			})
		default:
			w.Header().Set("Allow", "GET, POST, DELETE")
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}))
	mux.Handle("/api/v1/sessions", sessionsHandler)

	sessionDetailHandler := manager.makeProxyGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		trimmed := strings.TrimPrefix(r.URL.Path, "/api/v1/sessions/")
		parts := strings.Split(trimmed, "/")
		if len(parts) == 0 || parts[0] == "" {
			writeJSONError(w, http.StatusBadRequest, "session id required")
			return
		}

		id := parts[0]
		if len(parts) == 2 && parts[1] == "stream" {
			if r.Method != http.MethodGet {
				writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			session, ok := manager.getSession(id)
			if !ok {
				writeJSONError(w, http.StatusNotFound, "session not found")
				return
			}
			handleWebSocketStream(w, r, session, metrics)
			return
		}

		if len(parts) > 1 && parts[1] != "" && parts[1] != "panic" {
			writeJSONError(w, http.StatusNotFound, "endpoint not found")
			return
		}

		session, ok := manager.getSession(id)
		if !ok {
			writeJSONError(w, http.StatusNotFound, "session not found")
			return
		}

		if len(parts) == 2 && parts[1] == "panic" {
			if r.Method != http.MethodPost {
				writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			session.Close(reasonPanicStop)
			writeJSON(w, http.StatusAccepted, map[string]string{"status": "panic_stop_triggered"})
			return
		}

		switch r.Method {
		case http.MethodDelete:
			manager.deleteSession(id, reasonClientRequested)
			writeJSON(w, http.StatusOK, map[string]string{"status": "terminating"})
		case http.MethodGet:
			writeJSON(w, http.StatusOK, sessionSummary{
				ID:           session.id,
				CreatedAt:    session.createdAt,
				ExpiresAt:    session.expiresAt,
				LastActivity: session.lastActivityTime(),
				State:        "active",
				Command:      session.commandName,
				Args:         append([]string{}, session.commandArgs...),
			})
		default:
			writeJSONError(w, http.StatusMethodNotAllowed, "unsupported action")
		}
	}))
	mux.Handle("/api/v1/sessions/", sessionDetailHandler)
}

func handleGetWorkspace(w http.ResponseWriter, _ *http.Request, ws *workspace) {
	state, err := ws.getState()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to get workspace state")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(state)
}

func handleUpdateWorkspace(w http.ResponseWriter, r *http.Request, ws *workspace) {
	var req struct {
		ActiveTabID string    `json:"activeTabId"`
		Tabs        []tabMeta `json:"tabs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := ws.updateState(req.ActiveTabID, req.Tabs); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func handlePatchWorkspace(w http.ResponseWriter, r *http.Request, ws *workspace) {
	var req struct {
		KeyboardToolbarMode *string `json:"keyboardToolbarMode,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	if req.KeyboardToolbarMode != nil {
		if err := ws.setKeyboardToolbarMode(*req.KeyboardToolbarMode); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func handleCreateTab(w http.ResponseWriter, r *http.Request, ws *workspace) {
	var tab tabMeta
	if err := json.NewDecoder(r.Body).Decode(&tab); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := ws.addTab(tab); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			writeJSONError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, tab)
}

func handleUpdateTab(w http.ResponseWriter, r *http.Request, ws *workspace, tabID string) {
	var req struct {
		Label   string `json:"label"`
		ColorID string `json:"colorId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := ws.updateTab(tabID, req.Label, req.ColorID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func handleDeleteTab(w http.ResponseWriter, _ *http.Request, ws *workspace, tabID string) {
	if err := ws.removeTab(tabID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func handleCreateSession(w http.ResponseWriter, r *http.Request, manager *sessionManager, ws *workspace) {
	var req createSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != http.ErrBodyNotAllowed && err != io.EOF {
		writeJSONError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	s, err := manager.createSession(req)
	if err != nil {
		if strings.Contains(err.Error(), "capacity") {
			writeJSONError(w, http.StatusTooManyRequests, err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Attach session to tab if tabId provided
	if req.TabID != "" {
		_ = ws.attachSession(req.TabID, s.id)
	}

	resp := createSessionResponse{
		ID:        s.id,
		CreatedAt: s.createdAt,
		ExpiresAt: s.expiresAt,
		Command:   s.commandName,
		Args:      append([]string{}, s.commandArgs...),
	}
	writeJSON(w, http.StatusCreated, resp)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, apiError{Message: message})
}

func handleGenerateCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Prompt  string   `json:"prompt"`
		Context []string `json:"context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	if req.Prompt == "" {
		writeJSONError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	command, err := generateCommandWithOllama(req.Prompt, req.Context)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"command": command,
	})
}
