package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

func registerRoutes(mux *http.ServeMux, manager *sessionManager, metrics *metricsRegistry) {
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"note":   "web console must run behind authenticated proxy",
		})
	})

	mux.HandleFunc("/metrics", metrics.serveHTTP)

	sessionsHandler := manager.makeProxyGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleCreateSession(w, r, manager)
		case http.MethodGet:
			writeJSON(w, http.StatusOK, manager.listSummaries())
		default:
			w.Header().Set("Allow", "GET, POST")
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

func handleCreateSession(w http.ResponseWriter, r *http.Request, manager *sessionManager) {
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
