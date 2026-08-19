package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	sessionsH "web-console/handlers/sessions"
)

// Create implements sessions.RemoteService. The browser and CLI both reach
// this through the generated SessionsService.Create RPC; this method is the
// only bridge between that typed control call and the credential-bearing
// server-side terminal registry.
func (s *Server) Create(ctx context.Context, in sessionsH.CreateInput) (sessionsH.Session, error) {
	_ = ctx
	target, ok := s.targetByID(strings.TrimSpace(in.TargetID))
	if !ok {
		return sessionsH.Session{}, fmt.Errorf("%w: %s", sessionsH.ErrTargetNotFound, in.TargetID)
	}
	if target.Kind == "local" {
		return sessionsH.Session{}, fmt.Errorf("%w: local target must use the local session backend", sessionsH.ErrTargetUnavailable)
	}
	if !target.Available {
		reason := target.FailureRung
		if reason == "" {
			reason = "target is not dispatchable"
		}
		return sessionsH.Session{}, fmt.Errorf("%w: %s", sessionsH.ErrTargetUnavailable, reason)
	}
	cols, rows := in.Cols, in.Rows
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}
	now := time.Now().UTC()
	id := "remote:" + uuid.NewString()
	s.remoteRegistry().put(remoteTerminalSession{
		ID:                   id,
		Target:               target,
		Shell:                in.Shell,
		WorkingDir:           in.WorkingDir,
		LaunchCommand:        in.LaunchCommand,
		ExecuteLaunchCommand: in.ExecuteLaunchCommand,
		Cols:                 cols,
		Rows:                 rows,
		CreatedAt:            now,
	})
	return sessionsH.Session{
		ID:              id,
		Shell:           in.Shell,
		CreatedAt:       now.Format(time.RFC3339),
		Cols:            cols,
		Rows:            rows,
		Backend:         "standard",
		SurvivesRestart: false,
		Policy:          sessionsH.Policy{Mode: "never"},
		Origin:          "remote",
		Owner:           "target:" + target.ID,
		DisplayLabel:    target.Label,
		Target:          targetToProto(target),
	}, nil
}

func (s *Server) List(ctx context.Context) ([]sessionsH.Session, error) {
	_ = ctx
	s.remoteRegistry().mu.RLock()
	defer s.remoteRegistry().mu.RUnlock()
	items := make([]sessionsH.Session, 0, len(s.remoteRegistry().sessions))
	for _, current := range s.remoteRegistry().sessions {
		items = append(items, remoteSessionToHandler(current))
	}
	return items, nil
}

func (s *Server) Get(ctx context.Context, id string) (sessionsH.Session, error) {
	_ = ctx
	current, ok := s.remoteRegistry().get(id)
	if !ok {
		return sessionsH.Session{}, fmt.Errorf("%w: %s", sessionsH.ErrNotFound, id)
	}
	return remoteSessionToHandler(current), nil
}

func (s *Server) Delete(ctx context.Context, id string) error {
	_ = ctx
	current, ok := s.remoteRegistry().get(id)
	if !ok {
		return fmt.Errorf("%w: %s", sessionsH.ErrNotFound, id)
	}
	if current.cancel != nil {
		current.cancel()
	}
	s.remoteRegistry().delete(id)
	return nil
}

func remoteSessionToHandler(current remoteTerminalSession) sessionsH.Session {
	return sessionsH.Session{
		ID:              current.ID,
		Shell:           current.Shell,
		CreatedAt:       current.CreatedAt.Format(time.RFC3339),
		Cols:            current.Cols,
		Rows:            current.Rows,
		Backend:         "standard",
		SurvivesRestart: false,
		Policy:          sessionsH.Policy{Mode: "never"},
		Origin:          "remote",
		Owner:           "target:" + current.Target.ID,
		DisplayLabel:    current.Target.Label,
		Target:          targetToProto(current.Target),
	}
}
