package main

import (
	"context"

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
