package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	sessionsH "web-console/handlers/sessions"
	"web-console/internal/backend"
	"web-console/internal/events"
	"web-console/internal/policy"
	"web-console/internal/pty"
	"web-console/internal/sessionstore"
)

// sessionsAdapter implements sessionsH.Service against the server's
// SessionManager, SessionStore, policy resolver, and recovery flow.
type sessionsAdapter struct {
	srv *Server
}

func newSessionsAdapter(s *Server) *sessionsAdapter { return &sessionsAdapter{srv: s} }

// -----------------------------------------------------------------------------
// CRUD
// -----------------------------------------------------------------------------

func (a *sessionsAdapter) Create(_ context.Context, in sessionsH.CreateInput) (sessionsH.Session, error) {
	srv := a.srv

	// Idempotent replay.
	if in.IdempotencyKey != "" {
		if cached, ok := srv.idempotency.Get(in.IdempotencyKey); ok {
			log.Printf("create-session: idempotency hit for key %q, returning cached session %s", in.IdempotencyKey, cached.ID)
			return responseToHandlerSession(cached), nil
		}
	}

	var policyPtr *policy.Policy
	if in.HasPolicy {
		p := policy.Policy{
			Mode:     policy.Mode(in.Policy.Mode),
			Duration: in.Policy.Duration,
		}
		if err := policy.Validate(p); err != nil {
			return sessionsH.Session{}, fmt.Errorf("%w: %s", sessionsH.ErrInvalidArgument, err.Error())
		}
		policyPtr = &p
	}

	sess, err := srv.sessions.Create(in.Shell, uint16(in.Cols), uint16(in.Rows), backend.ID(in.Backend), policyPtr)
	if err != nil {
		return sessionsH.Session{}, mapCreateError(err)
	}

	if srv.sessionStore != nil && (in.LaunchCommand != "" || in.AgentType != "") {
		agentType := normalizeAgentType(in.AgentType)
		_ = srv.sessionStore.UpdateAgentInfo(sess.ID, sessionstore.AgentInfo{
			AgentType:     agentType,
			LaunchCommand: in.LaunchCommand,
		})
	}

	srv.events.Emit(events.SessionCreated, sess.ID, map[string]string{
		"shell":   sess.Shell,
		"cols":    fmt.Sprintf("%d", sess.Cols),
		"rows":    fmt.Sprintf("%d", sess.Rows),
		"backend": string(sess.Backend),
	})
	srv.metrics.SessionsCreated.Add(1)
	srv.metrics.ActiveSessions.Add(1)

	resp := sessionToResponse(sess)
	if in.IdempotencyKey != "" {
		srv.idempotency.Set(in.IdempotencyKey, resp)
	}
	return responseToHandlerSession(resp), nil
}

func (a *sessionsAdapter) List(_ context.Context) ([]sessionsH.Session, error) {
	live := a.srv.sessions.List()
	out := make([]sessionsH.Session, 0, len(live))
	for _, sess := range live {
		out = append(out, responseToHandlerSession(sessionToResponse(sess)))
	}
	return out, nil
}

func (a *sessionsAdapter) Get(_ context.Context, id string) (sessionsH.Session, error) {
	sess, ok := a.srv.sessions.Get(id)
	if !ok {
		return sessionsH.Session{}, fmt.Errorf("session %q: %w", sanitizeID(id), sessionsH.ErrNotFound)
	}
	return responseToHandlerSession(sessionToResponse(sess)), nil
}

func (a *sessionsAdapter) Delete(_ context.Context, id string) error {
	srv := a.srv
	if err := srv.sessions.Delete(id); err == nil {
		if srv.conversations != nil {
			srv.conversations.DeleteSession(id)
		}
		if srv.codexCheckpointStore != nil {
			_ = srv.codexCheckpointStore.DeleteSession(id)
		}
		srv.events.Emit(events.SessionDeleted, id, nil)
		srv.metrics.SessionsDeleted.Add(1)
		srv.metrics.ActiveSessions.Add(-1)
	}
	// Idempotent: post-condition "session does not exist" is satisfied either way.
	return nil
}

// -----------------------------------------------------------------------------
// Recovery
// -----------------------------------------------------------------------------

func (a *sessionsAdapter) ListRecoverable(_ context.Context) ([]sessionsH.RecoverableSession, error) {
	if a.srv.sessionStore == nil {
		return nil, nil
	}
	rows, err := a.srv.sessionStore.ListRecoverable()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", sessionsH.ErrInternal, err.Error())
	}
	out := make([]sessionsH.RecoverableSession, 0, len(rows))
	for _, m := range rows {
		out = append(out, toHandlerRecoverable(m))
	}
	return out, nil
}

func (a *sessionsAdapter) DismissRecoverable(_ context.Context, id string) error {
	srv := a.srv
	if srv.sessionStore == nil {
		return fmt.Errorf("session store not configured: %w", sessionsH.ErrNotFound)
	}
	meta, err := srv.sessionStore.Get(id)
	if err != nil {
		return fmt.Errorf("no session row with id %q: %w", sanitizeID(id), sessionsH.ErrNotFound)
	}
	if meta.Status != sessionstore.StatusAwaitingRecovery {
		return fmt.Errorf("session %q is in status %q, not awaiting_recovery: %w", sanitizeID(id), meta.Status, sessionsH.ErrFailedPrecondition)
	}
	if err := srv.sessionStore.MarkDismissed(id, ""); err != nil {
		return fmt.Errorf("mark dismissed: %v: %w", err, sessionsH.ErrInternal)
	}
	return nil
}

func (a *sessionsAdapter) Recover(_ context.Context, in sessionsH.RecoverInput) (sessionsH.RecoverResult, error) {
	srv := a.srv
	oldID := in.ID
	if srv.sessionStore == nil {
		return sessionsH.RecoverResult{}, fmt.Errorf("session store not configured: %w", sessionsH.ErrNotFound)
	}

	if in.IdempotencyKey != "" {
		if cached, ok := srv.idempotency.Get("recover:" + oldID + ":" + in.IdempotencyKey); ok {
			return sessionsH.RecoverResult{
				OldSessionID:    oldID,
				NewSessionID:    cached.ID,
				CodexHomeCopied: false, // not preserved across replay
			}, nil
		}
	}

	old, err := srv.sessionStore.Get(oldID)
	if err != nil {
		return sessionsH.RecoverResult{}, fmt.Errorf("no session row with id %q: %w", sanitizeID(oldID), sessionsH.ErrNotFound)
	}
	if old.Status != sessionstore.StatusAwaitingRecovery {
		return sessionsH.RecoverResult{}, fmt.Errorf("session %q is in status %q, not awaiting_recovery: %w", sanitizeID(oldID), old.Status, sessionsH.ErrFailedPrecondition)
	}
	if old.AgentType == sessionstore.AgentNone {
		return sessionsH.RecoverResult{}, fmt.Errorf("no agent identity recorded: %w", sessionsH.ErrFailedPrecondition)
	}
	if old.AgentType == sessionstore.AgentClaude && old.AgentSessionID == "" {
		return sessionsH.RecoverResult{}, fmt.Errorf("claude session id is required: %w", sessionsH.ErrFailedPrecondition)
	}

	cols := old.Cols
	rows := old.Rows
	if cols == 0 {
		cols = 120
	}
	if rows == 0 {
		rows = 36
	}
	policy := old.Policy
	newSess, err := srv.sessions.Create(old.Shell, cols, rows, backend.Persistent, &policy)
	if err != nil {
		log.Printf("recover[%s]: create new session: %v", oldID, err)
		return sessionsH.RecoverResult{}, mapCreateError(err)
	}
	srv.events.Emit(events.SessionCreated, newSess.ID, map[string]string{
		"shell":     newSess.Shell,
		"cols":      fmt.Sprintf("%d", newSess.Cols),
		"rows":      fmt.Sprintf("%d", newSess.Rows),
		"backend":   string(newSess.Backend),
		"recovered": "true",
		"from":      oldID,
	})

	codexHomeCopied := false
	if old.AgentType == sessionstore.AgentCodex {
		if err := copyCodexHome(oldID, newSess.ID); err != nil {
			log.Printf("recover[%s -> %s]: copy codex home: %v", oldID, newSess.ID, err)
			return sessionsH.RecoverResult{}, fmt.Errorf("copy codex home: %v: %w", err, sessionsH.ErrInternal)
		}
		codexHomeCopied = true
	}

	_ = srv.sessionStore.UpdateAgentInfo(newSess.ID, sessionstore.AgentInfo{
		AgentType:      old.AgentType,
		AgentSessionID: old.AgentSessionID,
		LaunchCommand:  old.LaunchCommand,
		CWD:            old.CWD,
	})

	cmd := buildResumeCommand(old)
	if err := newSess.WriteInput([]byte(cmd), pty.KindPaste); err != nil {
		log.Printf("recover[%s -> %s]: WriteInput: %v", oldID, newSess.ID, err)
		return sessionsH.RecoverResult{}, fmt.Errorf("paste resume command: %v: %w", err, sessionsH.ErrInternal)
	}

	if err := srv.sessionStore.MarkDismissed(oldID, newSess.ID); err != nil {
		log.Printf("recover[%s -> %s]: MarkDismissed: %v", oldID, newSess.ID, err)
	}

	res := sessionsH.RecoverResult{
		OldSessionID:    oldID,
		NewSessionID:    newSess.ID,
		AgentType:       string(old.AgentType),
		CommandSent:     cmd,
		CodexHomeCopied: codexHomeCopied,
	}

	if in.IdempotencyKey != "" {
		srv.idempotency.Set("recover:"+oldID+":"+in.IdempotencyKey, SessionResponse{
			ID:              res.NewSessionID,
			Backend:         backend.Persistent,
			SurvivesRestart: true,
			Recovered:       true,
		})
	}
	return res, nil
}

// -----------------------------------------------------------------------------
// Policy
// -----------------------------------------------------------------------------

func (a *sessionsAdapter) GetPolicy(_ context.Context, id string) (sessionsH.PolicyView, error) {
	sess, ok := a.srv.sessions.Get(id)
	if !ok {
		return sessionsH.PolicyView{}, fmt.Errorf("session %q: %w", sanitizeID(id), sessionsH.ErrNotFound)
	}
	return policyViewFor(sess, sess.GetPolicy()), nil
}

func (a *sessionsAdapter) UpdatePolicy(_ context.Context, id string, in sessionsH.Policy) (sessionsH.PolicyView, error) {
	srv := a.srv
	sess, ok := srv.sessions.Get(id)
	if !ok {
		return sessionsH.PolicyView{}, fmt.Errorf("session %q: %w", sanitizeID(id), sessionsH.ErrNotFound)
	}
	pol := policy.Policy{Mode: policy.Mode(in.Mode), Duration: in.Duration}
	if err := policy.Validate(pol); err != nil {
		return sessionsH.PolicyView{}, fmt.Errorf("%w: %s", sessionsH.ErrInvalidArgument, err.Error())
	}
	oldPolicy := sess.GetPolicy()
	sess.SetPolicy(pol)
	if srv.sessionStore != nil {
		_ = srv.sessionStore.UpdatePolicy(sess.ID, pol)
	}
	if oldPolicy.Mode != pol.Mode || oldPolicy.Duration != pol.Duration {
		srv.events.Emit(events.SessionPolicyUpdate, sess.ID, map[string]string{
			"mode":     in.Mode,
			"duration": in.Duration,
		})
	}
	return policyViewFor(sess, pol), nil
}

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

func mapCreateError(err error) error {
	switch {
	case errors.Is(err, ErrSessionLimitReached):
		return fmt.Errorf("%w: %v", sessionsH.ErrResourceExhausted, err)
	case errors.Is(err, ErrBackendUnavailable):
		return fmt.Errorf("%w: %v", sessionsH.ErrUnavailable, err)
	case errors.Is(err, ErrBackendUnknown):
		return fmt.Errorf("%w: %v", sessionsH.ErrInvalidArgument, err)
	case errors.Is(err, ErrPTYSpawnFailed):
		// Match the legacy "do not leak internal PTY details" behavior.
		return fmt.Errorf("%w: failed to start terminal process", sessionsH.ErrInternal)
	default:
		return fmt.Errorf("%w: %v", sessionsH.ErrInternal, err)
	}
}

func responseToHandlerSession(r SessionResponse) sessionsH.Session {
	return sessionsH.Session{
		ID:              r.ID,
		Shell:           r.Shell,
		CreatedAt:       r.CreatedAt,
		Cols:            r.Cols,
		Rows:            r.Rows,
		Backend:         string(r.Backend),
		SurvivesRestart: r.SurvivesRestart,
		Policy:          sessionsH.Policy{Mode: string(r.Policy.Mode), Duration: r.Policy.Duration},
		Busy:            r.Busy,
		Recovered:       r.Recovered,
	}
}

func toHandlerRecoverable(m sessionstore.Metadata) sessionsH.RecoverableSession {
	out := sessionsH.RecoverableSession{
		ID:              m.ID,
		Backend:         string(m.Backend),
		Shell:           m.Shell,
		Cols:            int(m.Cols),
		Rows:            int(m.Rows),
		CreatedAt:       m.Created.UTC().Format(time.RFC3339),
		OrphanedAt:      formatTimeOrEmpty(m.OrphanedAt),
		LastActivityAt:  formatTimeOrEmpty(m.LastActivityAt),
		AgentType:       string(m.AgentType),
		AgentSessionID:  m.AgentSessionID,
		LaunchCommand:   m.LaunchCommand,
		CWD:             m.CWD,
		LastRolloutPath: m.LastRolloutPath,
	}
	out.Recoverable, out.NotRecoverable = recoverabilityOf(m)
	return out
}

func policyViewFor(sess *Session, pol policy.Policy) sessionsH.PolicyView {
	view := sessionsH.PolicyView{
		SessionID: sess.ID,
		Policy:    sessionsH.Policy{Mode: string(pol.Mode), Duration: pol.Duration},
	}
	ttl := policy.ResolveTTL(pol)
	if ttl > 0 {
		expiresAt := sess.CreatedAt.Add(ttl)
		view.ExpiresAt = expiresAt.UTC().Format(time.RFC3339)
		remaining := time.Until(expiresAt).Seconds()
		if remaining < 0 {
			remaining = 0
		}
		view.TTLSeconds = remaining
		view.HasExpiry = true
	}
	return view
}
