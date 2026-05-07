package main

// session_recovery_handler.go: HTTP surface for the persistent-session
// recovery flow described in
// docs/plans/persistent-session-recovery-hardening-plan.md §4.
//
//   GET    /api/v1/sessions/recoverable          -> list awaiting_recovery rows
//   POST   /api/v1/sessions/{id}/recover         -> spawn fresh pane + resume agent
//   DELETE /api/v1/sessions/recoverable/{id}     -> mark dismissed (preserves on-disk state)
//
// The recover endpoint replaces the seven-step manual procedure from
// docs/guides/SESSION_RECOVERY.md "Recover Codex Panes".

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
)

// RecoverableSessionResponse is the JSON shape returned by the recoverable list
// and includes orphaned-state context the UI banner needs.
type RecoverableSessionResponse struct {
	ID              string    `json:"id"`
	Backend         BackendID `json:"backend"`
	Shell           string    `json:"shell"`
	Cols            int       `json:"cols"`
	Rows            int       `json:"rows"`
	CreatedAt       string    `json:"created_at"`
	OrphanedAt      string    `json:"orphaned_at"`
	LastActivityAt  string    `json:"last_activity_at,omitempty"`
	AgentType       AgentType `json:"agent_type"`
	AgentSessionID  string    `json:"agent_session_id,omitempty"`
	LaunchCommand   string    `json:"launch_command,omitempty"`
	CWD             string    `json:"cwd,omitempty"`
	LastRolloutPath string    `json:"last_rollout_path,omitempty"`
	Recoverable     bool      `json:"recoverable"`
	NotRecoverable  string    `json:"not_recoverable_reason,omitempty"`
}

// RecoverSessionResponse is returned by POST /api/v1/sessions/{id}/recover.
type RecoverSessionResponse struct {
	OldSessionID  string `json:"old_session_id"`
	NewSessionID  string `json:"new_session_id"`
	AgentType     string `json:"agent_type"`
	CommandSent   string `json:"command_sent"`
	CodexHomeCopy bool   `json:"codex_home_copied"`
}

// handleListRecoverable returns awaiting_recovery rows ordered by recency.
func (s *Server) handleListRecoverable(w http.ResponseWriter, r *http.Request) {
	if s.sessionStore == nil {
		writeJSON(w, http.StatusOK, []RecoverableSessionResponse{})
		return
	}
	rows, err := s.sessionStore.ListRecoverable()
	if err != nil {
		log.Printf("list-recoverable: %v", err)
		writeCatalogError(w, "internal_error", "Failed to list recoverable sessions")
		return
	}
	out := make([]RecoverableSessionResponse, 0, len(rows))
	for _, m := range rows {
		out = append(out, toRecoverableResponse(m))
	}
	writeJSON(w, http.StatusOK, out)
}

// handleDismissRecoverable marks an awaiting_recovery row as dismissed without
// recovery. On-disk state (CODEX_HOME) is preserved; only the DB row is hidden
// from the recoverable list.
func (s *Server) handleDismissRecoverable(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if s.sessionStore == nil {
		writeCatalogError(w, "session_not_found", "session store not configured")
		return
	}
	meta, err := s.sessionStore.Get(id)
	if err != nil {
		writeCatalogError(w, "session_not_found", "No session row with id "+id)
		return
	}
	if meta.Status != SessionStatusAwaitingRecovery {
		writeCatalogError(w, "recovery_not_eligible", fmt.Sprintf("session %s is in status %q, not awaiting_recovery", id, meta.Status))
		return
	}
	if err := s.sessionStore.MarkDismissed(id, ""); err != nil {
		writeCatalogError(w, "internal_error", "Failed to mark dismissed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "id": id})
}

// handleRecoverSession spawns a fresh persistent pane, copies the orphan's
// CODEX_HOME into the new pane's path, pastes the agent resume command, and
// marks the orphan dismissed. Idempotent on X-Idempotency-Key.
func (s *Server) handleRecoverSession(w http.ResponseWriter, r *http.Request) {
	oldID := mux.Vars(r)["id"]
	if s.sessionStore == nil {
		writeCatalogError(w, "session_not_found", "session store not configured")
		return
	}

	idemKey := r.Header.Get("X-Idempotency-Key")
	if idemKey != "" {
		if cached, ok := s.idempotency.Get("recover:" + oldID + ":" + idemKey); ok {
			writeJSON(w, http.StatusOK, cached)
			return
		}
	}

	old, err := s.sessionStore.Get(oldID)
	if err != nil {
		writeCatalogError(w, "session_not_found", "No session row with id "+oldID)
		return
	}
	if old.Status != SessionStatusAwaitingRecovery {
		writeCatalogError(w, "recovery_not_eligible", fmt.Sprintf("session %s is in status %q, not awaiting_recovery", oldID, old.Status))
		return
	}
	if old.AgentType == AgentTypeNone {
		writeCatalogError(w, "recovery_not_eligible", "session has no agent identity recorded; nothing to resume")
		return
	}
	if old.AgentType == AgentTypeClaude && old.AgentSessionID == "" {
		writeCatalogError(w, "recovery_claude_session_id_required", "")
		return
	}

	// 1. Create the fresh persistent pane, inheriting size + policy from the
	// orphan (so the user's preferences carry forward without an extra round
	// trip).
	cols := old.Cols
	rows := old.Rows
	if cols == 0 {
		cols = 120
	}
	if rows == 0 {
		rows = 36
	}
	policy := old.Policy
	newSess, err := s.sessions.Create(old.Shell, cols, rows, BackendPersistent, &policy)
	if err != nil {
		log.Printf("recover[%s]: create new session: %v", oldID, err)
		writeAppError(w, classifyCreateError(err))
		return
	}
	s.events.Emit(EventSessionCreated, newSess.ID, map[string]string{
		"shell":     newSess.Shell,
		"cols":      fmt.Sprintf("%d", newSess.Cols),
		"rows":      fmt.Sprintf("%d", newSess.Rows),
		"backend":   string(newSess.Backend),
		"recovered": "true",
		"from":      oldID,
	})

	// 2. Copy the orphan's CODEX_HOME into the new pane's path so codex resume
	// finds the prior history. Skip for non-codex agents (Claude history is
	// global under ~/.claude/projects/, no per-pane copy needed).
	codexHomeCopied := false
	if old.AgentType == AgentTypeCodex {
		if err := copyCodexHome(oldID, newSess.ID); err != nil {
			log.Printf("recover[%s -> %s]: copy codex home: %v", oldID, newSess.ID, err)
			writeCatalogError(w, "recovery_failed", "Failed to copy codex home: "+err.Error())
			return
		}
		codexHomeCopied = true
	}

	// 3. Persist agent identity on the new row so the next orphan-recovery
	// cycle (if it happens again) knows what to do.
	_ = s.sessionStore.UpdateAgentInfo(newSess.ID, AgentInfo{
		AgentType:      old.AgentType,
		AgentSessionID: old.AgentSessionID,
		LaunchCommand:  old.LaunchCommand,
		CWD:            old.CWD,
	})

	// 4. Build and paste the resume command. Same code path the WS uses
	// (session.WriteInput → tmux send-keys -l), so kind-aware dispatch and
	// stdin_ack semantics stay consistent.
	cmd := buildResumeCommand(old)
	if err := newSess.WriteInput([]byte(cmd), InputKindPaste); err != nil {
		log.Printf("recover[%s -> %s]: WriteInput: %v", oldID, newSess.ID, err)
		writeCatalogError(w, "recovery_failed", "Failed to paste resume command: "+err.Error())
		return
	}

	// 5. Mark orphan dismissed and record the new id for audit.
	if err := s.sessionStore.MarkDismissed(oldID, newSess.ID); err != nil {
		// Non-fatal — the resume already happened, but log and keep going.
		log.Printf("recover[%s -> %s]: MarkDismissed: %v", oldID, newSess.ID, err)
	}

	resp := RecoverSessionResponse{
		OldSessionID:  oldID,
		NewSessionID:  newSess.ID,
		AgentType:     string(old.AgentType),
		CommandSent:   cmd,
		CodexHomeCopy: codexHomeCopied,
	}

	if idemKey != "" {
		// Reuse SessionResponse cache as a free typed cache by encoding via
		// SessionResponse fields we don't use; cleaner to add a separate
		// recover-cache, but the existing idempotency cache is keyed on string
		// already. Wrap in a SessionResponse-shaped value-of-record.
		s.idempotency.Set("recover:"+oldID+":"+idemKey, SessionResponse{
			ID:              resp.NewSessionID,
			Backend:         BackendPersistent,
			SurvivesRestart: true,
			Recovered:       true,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

func toRecoverableResponse(m SessionMetadata) RecoverableSessionResponse {
	out := RecoverableSessionResponse{
		ID:              m.ID,
		Backend:         m.Backend,
		Shell:           m.Shell,
		Cols:            int(m.Cols),
		Rows:            int(m.Rows),
		CreatedAt:       m.Created.UTC().Format(time.RFC3339),
		OrphanedAt:      formatTimeOrEmpty(m.OrphanedAt),
		LastActivityAt:  formatTimeOrEmpty(m.LastActivityAt),
		AgentType:       m.AgentType,
		AgentSessionID:  m.AgentSessionID,
		LaunchCommand:   m.LaunchCommand,
		CWD:             m.CWD,
		LastRolloutPath: m.LastRolloutPath,
	}
	out.Recoverable, out.NotRecoverable = recoverabilityOf(m)
	return out
}

func recoverabilityOf(m SessionMetadata) (bool, string) {
	switch m.AgentType {
	case AgentTypeNone:
		return false, "no agent identity recorded"
	case AgentTypeClaude:
		if m.AgentSessionID == "" {
			return false, "claude session id is required (resuming the wrong project is unsafe)"
		}
		return true, ""
	case AgentTypeCodex:
		// Codex can resume by id OR fall back to --last given a copied home.
		return true, ""
	default:
		return false, "unknown agent type: " + string(m.AgentType)
	}
}

// buildResumeCommand returns the literal string to paste into the new pane's
// stdin. Includes a trailing newline so it executes immediately. Never returns
// the empty string.
func buildResumeCommand(m SessionMetadata) string {
	switch m.AgentType {
	case AgentTypeCodex:
		if m.AgentSessionID != "" {
			return "codex --yolo resume " + m.AgentSessionID + "\n"
		}
		return "codex --yolo resume --last\n"
	case AgentTypeClaude:
		// Caller has already checked AgentSessionID != "".
		return "claude --resume " + m.AgentSessionID + " --dangerously-skip-permissions\n"
	}
	return "echo 'no agent identity recorded; nothing to resume'\n"
}

// copyCodexHome rsyncs (or copies via tar fallback) the per-session CODEX_HOME
// from the orphan to the fresh pane. Bounded: codex homes contain symlinks to
// global config + per-session rollouts, typically < 50MB.
func copyCodexHome(oldID, newID string) error {
	src := sessionCodexHome(oldID)
	dst := sessionCodexHome(newID)
	if _, err := os.Stat(src); err != nil {
		// Nothing to copy — the codex tailer never saw a rollout. Recovery
		// can still proceed (`codex --yolo resume <id>` will go fetch from the
		// upstream rollout path if codex knows the global location).
		return nil
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return fmt.Errorf("mkdirall %s: %w", dst, err)
	}
	// Prefer rsync; fall back to a `cp -a` if rsync is missing.
	if path, err := exec.LookPath("rsync"); err == nil {
		out, err := exec.Command(path, "-a", "--", src+"/", dst+"/").CombinedOutput()
		if err != nil {
			return fmt.Errorf("rsync %s -> %s: %v: %s", src, dst, err, string(out))
		}
		return nil
	}
	if path, err := exec.LookPath("cp"); err == nil {
		out, err := exec.Command(path, "-a", filepath.Join(src, "."), dst).CombinedOutput()
		if err != nil {
			return fmt.Errorf("cp -a %s -> %s: %v: %s", src, dst, err, string(out))
		}
		return nil
	}
	return fmt.Errorf("neither rsync nor cp available")
}
