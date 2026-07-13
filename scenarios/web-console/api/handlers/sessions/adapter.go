package sessions

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"web-console/internal/backend"
	"web-console/internal/events"
	intmetrics "web-console/internal/metrics"
	"web-console/internal/policy"
	intsessions "web-console/internal/sessions"
	"web-console/internal/sessionstore"
	"web-console/session"
)

// SessionManager is the slice of session.Manager the Adapter depends on.
type SessionManager interface {
	Create(shell string, cols, rows uint16, backend backend.ID, policy *policy.Policy) (*session.Session, error)
	Get(id string) (*session.Session, bool)
	List() []*session.Session
	Delete(id string) error
	RecoveryProgress() session.RecoveryProgress
}

// ConversationsStore is the minimal seam for moving/clearing conversation
// state during the session lifecycle. The production *ConversationStore
// satisfies it. CopySession carries a recovered session's prior message
// history onto its fresh replacement id so the messages view is not empty
// after reattach.
type ConversationsStore interface {
	DeleteSession(id string)
	CopySession(oldID, newID string) error
}

// CodexCheckpoints is the minimal seam for clearing per-source ingestion
// checkpoint state on session deletion. Both the codex byte-offset store and
// the generic agent-transcript checkpoint store (Grok/OpenCode) satisfy it.
type CodexCheckpoints interface {
	DeleteSession(id string) error
}

// Adapter is the production Service implementation. It is constructed in
// api/main.go with typed deps — no *Server import — and passed to Module.
type Adapter struct {
	Manager          SessionManager
	Store            sessionstore.Store
	Idempotency      *intsessions.IdempotencyCache
	Events           *events.Logger
	Metrics          *intmetrics.Metrics
	Conversations    ConversationsStore
	CodexCheckpoints CodexCheckpoints
	AgentCheckpoints CodexCheckpoints
	CopyCodexHome    func(oldID, newID string) error
	Logger           *log.Logger
}

func (a *Adapter) logger() *log.Logger {
	if a.Logger != nil {
		return a.Logger
	}
	return log.Default()
}

// -----------------------------------------------------------------------------
// CRUD
// -----------------------------------------------------------------------------

func (a *Adapter) Create(_ context.Context, in CreateInput) (Session, error) {
	if in.IdempotencyKey != "" {
		if cached, ok := a.Idempotency.Get(in.IdempotencyKey); ok {
			a.logger().Printf("create-session: idempotency hit for key %q, returning cached session %s", in.IdempotencyKey, cached.ID)
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
			return Session{}, fmt.Errorf("%w: %s", ErrInvalidArgument, err.Error())
		}
		policyPtr = &p
	}

	sess, err := a.Manager.Create(in.Shell, uint16(in.Cols), uint16(in.Rows), backend.ID(in.Backend), policyPtr)
	if err != nil {
		return Session{}, mapCreateError(err)
	}

	if a.Store != nil && (in.LaunchCommand != "" || in.AgentType != "") {
		agentType := intsessions.NormalizeAgentType(in.AgentType)
		_ = a.Store.UpdateAgentInfo(sess.ID, sessionstore.AgentInfo{
			AgentType:     agentType,
			LaunchCommand: in.LaunchCommand,
		})
	}

	// Provenance: an origin-less create can only be programmatic (every
	// first-party UI client sets origin explicitly), so normalize before we
	// persist and echo it back.
	origin := intsessions.NormalizeOrigin(in.Origin)
	if a.Store != nil {
		_ = a.Store.SetProvenance(sess.ID, origin, in.Owner, in.DisplayLabel)
	}

	// Server-side launch execution: paste the launch command into the fresh
	// PTY so it runs exactly once, mirroring the Recover paste seam (bracketed
	// paste + trailing newline to execute). Best-effort — the session already
	// exists and was returned to the caller, so a paste failure must not fail
	// the create and orphan the pane; unlike Recover (where the resume paste is
	// the whole point), the command here is a convenience the user can retype.
	if in.ExecuteLaunchCommand && in.LaunchCommand != "" {
		if err := sess.SendInput(session.InputText(in.LaunchCommand + "\n").AsPaste().WithSource("launch")); err != nil {
			a.logger().Printf("create-session[%s]: paste launch command: %v", sess.ID, err)
		}
	}

	a.Events.Emit(events.SessionCreated, sess.ID, map[string]string{
		"shell":   sess.Shell,
		"cols":    fmt.Sprintf("%d", sess.Cols),
		"rows":    fmt.Sprintf("%d", sess.Rows),
		"backend": string(sess.Backend),
		"origin":  string(origin),
		"owner":   in.Owner,
		"label":   in.DisplayLabel,
		"agent":   string(intsessions.NormalizeAgentType(in.AgentType)),
	})
	a.Metrics.SessionsCreated.Add(1)
	a.Metrics.ActiveSessions.Add(1)

	resp := intsessions.FromSession(sess)
	resp.Origin = string(origin)
	resp.Owner = in.Owner
	resp.DisplayLabel = in.DisplayLabel
	if in.IdempotencyKey != "" {
		a.Idempotency.Set(in.IdempotencyKey, resp)
	}
	return responseToHandlerSession(resp), nil
}

func (a *Adapter) List(_ context.Context) ([]Session, error) {
	live := a.Manager.List()
	// The store is the source of truth for provenance (origin/owner/label);
	// the in-memory session carries only PTY/terminal state. Merge one store
	// read into the live list rather than reading per-session.
	provenance := a.provenanceByID()
	out := make([]Session, 0, len(live))
	for _, sess := range live {
		s := responseToHandlerSession(intsessions.FromSession(sess))
		if p, ok := provenance[s.ID]; ok {
			s.Origin, s.Owner, s.DisplayLabel = string(p.Origin), p.Owner, p.DisplayLabel
		}
		out = append(out, s)
	}
	return out, nil
}

// provenanceByID snapshots stored provenance keyed by session id. Returns an
// empty map when no store is configured (e.g. minimal test servers).
func (a *Adapter) provenanceByID() map[string]sessionstore.Metadata {
	if a.Store == nil {
		return nil
	}
	rows, err := a.Store.List()
	if err != nil {
		a.logger().Printf("list sessions: load provenance: %v", err)
		return nil
	}
	byID := make(map[string]sessionstore.Metadata, len(rows))
	for _, m := range rows {
		byID[m.ID] = m
	}
	return byID
}

// RecoveryStatus exposes startup session-recovery progress for the List
// response so the UI can show an honest "sessions still recovering" indicator.
func (a *Adapter) RecoveryStatus(_ context.Context) RecoveryStatus {
	p := a.Manager.RecoveryProgress()
	rs := RecoveryStatus{
		InProgress:       p.InProgress,
		Total:            p.Total,
		Recovered:        p.Recovered,
		AwaitingRecovery: p.AwaitingRecovery,
		Adopted:          p.Adopted,
	}
	if !p.StartedAt.IsZero() {
		rs.StartedAtUnixMs = p.StartedAt.UnixMilli()
	}
	if !p.CompletedAt.IsZero() {
		rs.CompletedAtUnixMs = p.CompletedAt.UnixMilli()
	}
	return rs
}

func (a *Adapter) Get(_ context.Context, id string) (Session, error) {
	sess, ok := a.Manager.Get(id)
	if !ok {
		return Session{}, fmt.Errorf("session %q: %w", sanitizeID(id), ErrNotFound)
	}
	s := responseToHandlerSession(intsessions.FromSession(sess))
	if a.Store != nil {
		if m, err := a.Store.Get(id); err == nil {
			s.Origin, s.Owner, s.DisplayLabel = string(m.Origin), m.Owner, m.DisplayLabel
		}
	}
	return s, nil
}

func (a *Adapter) Delete(_ context.Context, id string) error {
	if err := a.Manager.Delete(id); err == nil {
		if a.Conversations != nil {
			a.Conversations.DeleteSession(id)
		}
		if a.CodexCheckpoints != nil {
			_ = a.CodexCheckpoints.DeleteSession(id)
		}
		if a.AgentCheckpoints != nil {
			_ = a.AgentCheckpoints.DeleteSession(id)
		}
		a.Events.Emit(events.SessionDeleted, id, nil)
		a.Metrics.SessionsDeleted.Add(1)
		a.Metrics.ActiveSessions.Add(-1)
	}
	return nil
}

// -----------------------------------------------------------------------------
// Recovery
// -----------------------------------------------------------------------------

func (a *Adapter) ListRecoverable(_ context.Context) ([]RecoverableSession, error) {
	if a.Store == nil {
		return nil, nil
	}
	rows, err := a.Store.ListRecoverable()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInternal, err.Error())
	}
	out := make([]RecoverableSession, 0, len(rows))
	for _, m := range rows {
		out = append(out, toHandlerRecoverable(m))
	}
	return out, nil
}

func (a *Adapter) DismissRecoverable(_ context.Context, id string) error {
	if a.Store == nil {
		return fmt.Errorf("session store not configured: %w", ErrNotFound)
	}
	meta, err := a.Store.Get(id)
	if err != nil {
		return fmt.Errorf("no session row with id %q: %w", sanitizeID(id), ErrNotFound)
	}
	if meta.Status != sessionstore.StatusAwaitingRecovery {
		return fmt.Errorf("session %q is in status %q, not awaiting_recovery: %w", sanitizeID(id), meta.Status, ErrFailedPrecondition)
	}
	if err := a.Store.MarkDismissed(id, ""); err != nil {
		return fmt.Errorf("mark dismissed: %v: %w", err, ErrInternal)
	}
	return nil
}

func (a *Adapter) Recover(_ context.Context, in RecoverInput) (RecoverResult, error) {
	oldID := in.ID
	if a.Store == nil {
		return RecoverResult{}, fmt.Errorf("session store not configured: %w", ErrNotFound)
	}

	if in.IdempotencyKey != "" {
		if cached, ok := a.Idempotency.Get("recover:" + oldID + ":" + in.IdempotencyKey); ok {
			return RecoverResult{
				OldSessionID:    oldID,
				NewSessionID:    cached.ID,
				CodexHomeCopied: false,
			}, nil
		}
	}

	old, err := a.Store.Get(oldID)
	if err != nil {
		return RecoverResult{}, fmt.Errorf("no session row with id %q: %w", sanitizeID(oldID), ErrNotFound)
	}
	if old.Status != sessionstore.StatusAwaitingRecovery {
		return RecoverResult{}, fmt.Errorf("session %q is in status %q, not awaiting_recovery: %w", sanitizeID(oldID), old.Status, ErrFailedPrecondition)
	}
	// Single source of truth for recoverability (and its precise refusal
	// reasons) so every agent type — codex, claude, opencode, grok — is gated
	// identically here and in the recoverable-sessions listing.
	if ok, reason := intsessions.Recoverability(old); !ok {
		return RecoverResult{}, fmt.Errorf("%s: %w", reason, ErrFailedPrecondition)
	}

	cols := old.Cols
	rows := old.Rows
	if cols == 0 {
		cols = 120
	}
	if rows == 0 {
		rows = 36
	}
	pol := old.Policy
	newSess, err := a.Manager.Create(old.Shell, cols, rows, backend.Persistent, &pol)
	if err != nil {
		a.logger().Printf("recover[%s]: create new session: %v", oldID, err)
		return RecoverResult{}, mapCreateError(err)
	}
	a.Events.Emit(events.SessionCreated, newSess.ID, map[string]string{
		"shell":     newSess.Shell,
		"cols":      fmt.Sprintf("%d", newSess.Cols),
		"rows":      fmt.Sprintf("%d", newSess.Rows),
		"backend":   string(newSess.Backend),
		"recovered": "true",
		"from":      oldID,
		"origin":    string(old.Origin),
		"owner":     old.Owner,
		"label":     old.DisplayLabel,
		"agent":     string(old.AgentType),
	})

	codexHomeCopied := false
	if old.AgentType == sessionstore.AgentCodex && a.CopyCodexHome != nil {
		if err := a.CopyCodexHome(oldID, newSess.ID); err != nil {
			a.logger().Printf("recover[%s -> %s]: copy codex home: %v", oldID, newSess.ID, err)
			return RecoverResult{}, fmt.Errorf("copy codex home: %v: %w", err, ErrInternal)
		}
		codexHomeCopied = true
	}

	_ = a.Store.UpdateAgentInfo(newSess.ID, sessionstore.AgentInfo{
		AgentType:      old.AgentType,
		AgentSessionID: old.AgentSessionID,
		LaunchCommand:  old.LaunchCommand,
		CWD:            old.CWD,
	})
	// Carry provenance onto the recovered session so it keeps its original
	// origin/owner/label in the sidebar.
	_ = a.Store.SetProvenance(newSess.ID, old.Origin, old.Owner, old.DisplayLabel)

	// Carry the prior conversation history onto the new session id so the
	// messages view is populated after reattach. Best-effort: a copy failure
	// must not abort recovery — the agent resume is the critical path.
	messagesCopied := false
	if a.Conversations != nil {
		if err := a.Conversations.CopySession(oldID, newSess.ID); err != nil {
			a.logger().Printf("recover[%s -> %s]: copy conversation history: %v", oldID, newSess.ID, err)
		} else {
			messagesCopied = true
		}
	}

	cmd := intsessions.BuildResumeCommand(old)
	if err := newSess.SendInput(session.InputText(cmd).AsPaste().WithSource("recover")); err != nil {
		a.logger().Printf("recover[%s -> %s]: SendInput: %v", oldID, newSess.ID, err)
		return RecoverResult{}, fmt.Errorf("paste resume command: %v: %w", err, ErrInternal)
	}

	if err := a.Store.MarkDismissed(oldID, newSess.ID); err != nil {
		a.logger().Printf("recover[%s -> %s]: MarkDismissed: %v", oldID, newSess.ID, err)
	}

	a.logger().Printf("recover[%s -> %s]: agent=%s codexHome=%t messages=%t", oldID, newSess.ID, old.AgentType, codexHomeCopied, messagesCopied)

	res := RecoverResult{
		OldSessionID:    oldID,
		NewSessionID:    newSess.ID,
		AgentType:       string(old.AgentType),
		CommandSent:     cmd,
		CodexHomeCopied: codexHomeCopied,
		MessagesCopied:  messagesCopied,
	}

	if in.IdempotencyKey != "" {
		a.Idempotency.Set("recover:"+oldID+":"+in.IdempotencyKey, intsessions.Response{
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

func (a *Adapter) GetPolicy(_ context.Context, id string) (PolicyView, error) {
	sess, ok := a.Manager.Get(id)
	if !ok {
		return PolicyView{}, fmt.Errorf("session %q: %w", sanitizeID(id), ErrNotFound)
	}
	return policyViewFor(sess, sess.GetPolicy()), nil
}

func (a *Adapter) UpdatePolicy(_ context.Context, id string, in Policy) (PolicyView, error) {
	sess, ok := a.Manager.Get(id)
	if !ok {
		return PolicyView{}, fmt.Errorf("session %q: %w", sanitizeID(id), ErrNotFound)
	}
	pol := policy.Policy{Mode: policy.Mode(in.Mode), Duration: in.Duration}
	if err := policy.Validate(pol); err != nil {
		return PolicyView{}, fmt.Errorf("%w: %s", ErrInvalidArgument, err.Error())
	}
	oldPolicy := sess.GetPolicy()
	sess.SetPolicy(pol)
	if a.Store != nil {
		_ = a.Store.UpdatePolicy(sess.ID, pol)
	}
	if oldPolicy.Mode != pol.Mode || oldPolicy.Duration != pol.Duration {
		a.Events.Emit(events.SessionPolicyUpdate, sess.ID, map[string]string{
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
	case errors.Is(err, session.ErrSessionLimitReached):
		return fmt.Errorf("%w: %v", ErrResourceExhausted, err)
	case errors.Is(err, session.ErrBackendUnavailable):
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	case errors.Is(err, session.ErrBackendUnknown):
		return fmt.Errorf("%w: %v", ErrInvalidArgument, err)
	case errors.Is(err, session.ErrPTYSpawnFailed):
		return fmt.Errorf("%w: failed to start terminal process", ErrInternal)
	default:
		return fmt.Errorf("%w: %v", ErrInternal, err)
	}
}

func responseToHandlerSession(r intsessions.Response) Session {
	return Session{
		ID:              r.ID,
		Shell:           r.Shell,
		CreatedAt:       r.CreatedAt,
		Cols:            r.Cols,
		Rows:            r.Rows,
		Backend:         string(r.Backend),
		SurvivesRestart: r.SurvivesRestart,
		Policy:          Policy{Mode: string(r.Policy.Mode), Duration: r.Policy.Duration},
		Busy:            r.Busy,
		Recovered:       r.Recovered,
		Origin:          r.Origin,
		Owner:           r.Owner,
		DisplayLabel:    r.DisplayLabel,
	}
}

func toHandlerRecoverable(m sessionstore.Metadata) RecoverableSession {
	out := RecoverableSession{
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
	out.Recoverable, out.NotRecoverable = intsessions.Recoverability(m)
	return out
}

func policyViewFor(sess *session.Session, pol policy.Policy) PolicyView {
	view := PolicyView{
		SessionID: sess.ID,
		Policy:    Policy{Mode: string(pol.Mode), Duration: pol.Duration},
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

func formatTimeOrEmpty(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func sanitizeID(id string) string {
	clean := strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, id)
	if len(clean) > 40 {
		clean = clean[:40] + "..."
	}
	return clean
}
