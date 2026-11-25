package main

import (
	"context"
	"encoding/json"
	"net/http"

	"web-console/internal/dbx"
)

const (
	activationFirstCommandRun = "activation.first_command_run"
	activationFirstAICommand  = "activation.first_ai_command_accepted"
	activationProfileKey      = "default"
)

func (s *Server) emitActivationOnce(ctx context.Context, eventType string) {
	if s == nil || s.db == nil || s.events == nil {
		return
	}
	result, err := s.db.Primary().ExecContext(ctx, `INSERT OR IGNORE INTO web_console_activation_events(profile_key, event_type) VALUES (?, ?)`, activationProfileKey, eventType)
	if err != nil {
		return
	}
	if inserted, _ := result.RowsAffected(); inserted == 1 {
		s.events.Emit(eventType, "", map[string]string{"profile": activationProfileKey})
	}
}

func ensureActivationEvents(ctx context.Context, db dbx.Handle) error {
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS web_console_activation_events (
		profile_key TEXT NOT NULL,
		event_type TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY(profile_key, event_type)
	)`)
	return err
}

type activationStatus struct {
	FirstCommandRun        bool `json:"first_command_run"`
	FirstAICommandAccepted bool `json:"first_ai_command_accepted"`
}

func (s *Server) activationStatusHandler(w http.ResponseWriter, r *http.Request) {
	if !webConsoleSameOrigin(w, r) {
		return
	}
	if s == nil || s.db == nil {
		http.Error(w, "activation status unavailable", http.StatusServiceUnavailable)
		return
	}
	rows, err := s.db.Primary().QueryContext(r.Context(), `SELECT event_type FROM web_console_activation_events WHERE profile_key = ?`, activationProfileKey)
	if err != nil {
		http.Error(w, "activation status unavailable", http.StatusServiceUnavailable)
		return
	}
	defer rows.Close()
	status := activationStatus{}
	for rows.Next() {
		var eventType string
		if err := rows.Scan(&eventType); err != nil {
			http.Error(w, "activation status unavailable", http.StatusServiceUnavailable)
			return
		}
		switch eventType {
		case activationFirstCommandRun:
			status.FirstCommandRun = true
		case activationFirstAICommand:
			status.FirstAICommandAccepted = true
		}
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "activation status unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}
