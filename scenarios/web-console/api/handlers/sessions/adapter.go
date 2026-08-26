package sessions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"web-console/internal/backend"
	"web-console/internal/events"
	intmetrics "web-console/internal/metrics"
	"web-console/internal/policy"
	intsessions "web-console/internal/sessions"
	"web-console/internal/sessionstore"
	intworkspace "web-console/internal/workspace"
	"web-console/session"
)

// SessionManager is the slice of session.Manager the Adapter depends on.
type SessionManager interface {
	Create(ctx context.Context, shell string, cols, rows uint16, backend backend.ID, policy *policy.Policy) (*session.Session, error)
	CreateWithWorkingDir(ctx context.Context, shell string, cols, rows uint16, backend backend.ID, policy *policy.Policy, workingDir string) (*session.Session, error)
	CreateWithOptions(ctx context.Context, shell string, cols, rows uint16, backend backend.ID, policy *policy.Policy, workingDir string, tmuxMouseMode bool) (*session.Session, error)
	Get(id string) (*session.Session, bool)
	List() []*session.Session
	Delete(ctx context.Context, id string) error
	Archive(ctx context.Context, id string) error
	RecoveryProgress() session.RecoveryProgress
}

// ConversationsStore is the minimal seam for moving/clearing conversation
// state during the session lifecycle. The production *ConversationStore
// satisfies it. CopySession carries a recovered session's prior message
// history onto its fresh replacement id so the messages view is not empty
// after reattach.
type ConversationsStore interface {
	DeleteSession(ctx context.Context, id string)
	CopySession(ctx context.Context, oldID, newID string) error
	HasConversationAfter(ctx context.Context, sessionID string, after time.Time) bool
	CountSessionEvents(ctx context.Context, sessionID string) int64
	SessionStorageBytes(ctx context.Context, sessionID string) int64
}

// CodexCheckpoints is the minimal seam for clearing per-source ingestion
// checkpoint state on session deletion. Both the codex byte-offset store and
// the generic agent-transcript checkpoint store (Grok/OpenCode) satisfy it.
type CodexCheckpoints interface {
	DeleteSession(ctx context.Context, id string) error
}

// Adapter is the production Service implementation. It is constructed in
// api/main.go with typed deps — no *Server import — and passed to Module.
type Adapter struct {
	Manager             SessionManager
	Store               sessionstore.Store
	Idempotency         *intsessions.IdempotencyCache
	Events              *events.Logger
	Metrics             *intmetrics.Metrics
	Conversations       ConversationsStore
	CodexCheckpoints    CodexCheckpoints
	AgentCheckpoints    CodexCheckpoints
	Workspace           intworkspace.Store
	CopyCodexHome       func(oldID, newID string) error
	Logger              *log.Logger
	AgentHistoryPresent func(sessionstore.Metadata) bool
	RetentionPolicy     func() ArchiveRetentionPolicy
	AgentHistorySize    func(sessionstore.Metadata) (int64, error)
	PruneAgentHistory   func(sessionstore.Metadata) (int64, error)
	Now                 func() time.Time
	Remote              RemoteService

	// ArchiveGracePeriod is the server-owned undo window. Zero uses the
	// product default; tests may set a negative duration for immediate finalization.
	ArchiveGracePeriod time.Duration
	archiveMu          sync.Mutex
	archiveTimers      map[string]*time.Timer
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

func (a *Adapter) Create(ctx context.Context, in CreateInput) (Session, error) {
	fingerprint := createFingerprint(in)
	if in.IdempotencyKey != "" && a.Idempotency != nil {
		if cached, ok := a.Idempotency.Get(in.IdempotencyKey); ok {
			if cached.Fingerprint != "" && cached.Fingerprint != fingerprint {
				return Session{}, fmt.Errorf("%w: %s", ErrIdempotencyConflict, in.IdempotencyKey)
			}
			a.logger().Printf("create-session: idempotency hit for key %q, returning cached session %s", in.IdempotencyKey, cached.ID)
			return responseToHandlerSession(cached), nil
		}
	}
	if targetID := strings.TrimSpace(in.TargetID); targetID != "" && targetID != "local" {
		if a.Remote == nil {
			return Session{}, fmt.Errorf("%w: remote session service is not configured", ErrRemoteUnavailable)
		}
		created, err := a.Remote.Create(ctx, in)
		if err != nil {
			return Session{}, err
		}
		if in.IdempotencyKey != "" && a.Idempotency != nil {
			cached := handlerSessionToResponse(created)
			cached.Fingerprint = fingerprint
			a.Idempotency.Set(in.IdempotencyKey, cached)
		}
		return created, nil
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

	// Under a routed test lease, force a disposable shape: standard backend
	// (no tmux pane on the operator's shared server, cannot be re-adopted by
	// recovery) and a short expiry so a leaked session reaps itself. See
	// testmode.go.
	bid, policyPtr := applyTestLeaseShape(ctx, backend.ID(in.Backend), policyPtr)

	var sess *session.Session
	var err error
	sess, err = a.Manager.CreateWithOptions(ctx, in.Shell, uint16(in.Cols), uint16(in.Rows), bid, policyPtr, in.WorkingDir, in.TmuxMouseMode)
	if err != nil {
		return Session{}, mapCreateError(err)
	}

	if a.Store != nil && (in.LaunchCommand != "" || in.AgentType != "") {
		agentType := intsessions.NormalizeAgentType(in.AgentType)
		_ = a.Store.UpdateAgentInfo(ctx, sess.ID, sessionstore.AgentInfo{
			AgentType:     agentType,
			LaunchCommand: in.LaunchCommand,
		})
	}

	// Provenance: an origin-less create can only be programmatic (every
	// first-party UI client sets origin explicitly), so normalize before we
	// persist and echo it back.
	origin := intsessions.NormalizeOrigin(in.Origin)
	owner, displayLabel := in.Owner, in.DisplayLabel
	if isTestLease(ctx) {
		// Stamp test provenance so a leaked session is identifiable and
		// bulk-removable rather than indistinguishable from an operator tab.
		owner, displayLabel = testSessionOwner, testSessionLabel
	}
	if a.Store != nil {
		_ = a.Store.SetProvenance(ctx, sess.ID, origin, owner, displayLabel)
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
	if in.IdempotencyKey != "" && a.Idempotency != nil {
		resp.Fingerprint = fingerprint
		a.Idempotency.Set(in.IdempotencyKey, resp)
	}
	return responseToHandlerSession(resp), nil
}

func (a *Adapter) List(ctx context.Context) ([]Session, error) {
	live := a.Manager.List()
	// The store is the source of truth for provenance (origin/owner/label);
	// the in-memory session carries only PTY/terminal state. Merge one store
	// read into the live list rather than reading per-session.
	provenance := a.provenanceByID(ctx)
	recoveredAgents := make(map[string]sessionstore.Agent)
	for _, meta := range provenance {
		if meta.RecoveredInto != "" {
			recoveredAgents[meta.RecoveredInto] = meta.AgentType
		}
	}
	out := make([]Session, 0, len(live))
	for _, sess := range live {
		s := responseToHandlerSession(intsessions.FromSession(sess))
		if p, ok := provenance[s.ID]; ok {
			if !p.ArchivedAt.IsZero() {
				continue
			}
			s.Origin, s.Owner, s.DisplayLabel = string(p.Origin), p.Owner, p.DisplayLabel
		}
		if recoveredAgents[s.ID] == sessionstore.AgentClaude && isClaudeTrackingDegraded(sess, a.Conversations) {
			s.TrackingDegraded = true
		}
		out = append(out, s)
	}
	if a.Remote != nil {
		remote, err := a.Remote.List(ctx)
		if err != nil {
			return nil, fmt.Errorf("%w: list remote sessions: %v", ErrRemoteUnavailable, err)
		}
		out = append(out, remote...)
	}
	return out, nil
}

func (a *Adapter) ListArchived(ctx context.Context) ([]ArchivedSession, error) {
	if a.Store == nil {
		return nil, nil
	}
	archived, err := a.Store.ListArchived(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: list archived sessions: %s", ErrInternal, err)
	}
	all, err := a.Store.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: resolve archive lineages: %s", ErrInternal, err)
	}
	byID := make(map[string]sessionstore.Metadata, len(all))
	for _, row := range all {
		byID[row.ID] = row
	}

	panes := map[string]intworkspace.Pane{}
	groups := map[string]string{}
	if a.Workspace != nil {
		layout, layoutErr := a.Workspace.GetLayout(ctx)
		if layoutErr != nil {
			return nil, fmt.Errorf("%w: load archived workspace identity: %s", ErrInternal, layoutErr)
		}
		for _, pane := range layout.Panes {
			panes[pane.SessionID] = pane
		}
		for _, group := range layout.Groups {
			groups[group.ID] = group.Name
		}
	}

	collapsed := make(map[string]sessionstore.Metadata, len(archived))
	for _, row := range archived {
		newest := intsessions.ResolveLineage(row, byID)
		if newest.ArchivedAt.IsZero() && newest.Status != sessionstore.StatusDismissed && newest.Status != sessionstore.StatusAwaitingRecovery {
			continue
		}
		collapsed[newest.ID] = newest
	}

	result := make([]ArchivedSession, 0, len(collapsed))
	for _, row := range collapsed {
		messageCount := int64(0)
		if a.Conversations != nil {
			messageCount = a.Conversations.CountSessionEvents(ctx, row.ID)
		}
		archivedAt := row.ArchivedAt
		if archivedAt.IsZero() {
			archivedAt = row.OrphanedAt
		}
		if archivedAt.IsZero() {
			archivedAt = row.LastActivityAt
		}
		if archivedAt.IsZero() {
			archivedAt = row.Created
		}
		entry := ArchivedSession{
			ID:               row.ID,
			ArchivedAt:       formatTimeOrEmpty(archivedAt),
			CreatedAt:        formatTimeOrEmpty(row.Created),
			AgentType:        string(row.AgentType),
			AgentSessionID:   row.AgentSessionID,
			CWD:              row.CWD,
			MessageCount:     messageCount,
			AwaitingRecovery: row.Status == sessionstore.StatusAwaitingRecovery,
		}
		entry.RestoreState, entry.RestoreStateReason = a.restoreState(row, messageCount)
		if pane, ok := panes[row.ID]; ok {
			entry.PaneName = pane.Name
			entry.HeaderColor = pane.HeaderColor
			entry.GroupName = groups[pane.GroupID]
		}
		result = append(result, entry)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ArchivedAt > result[j].ArchivedAt })
	return result, nil
}

type retentionCandidate struct {
	meta            sessionstore.Metadata
	messageCount    int64
	transcriptBytes int64
	homeBytes       int64
}

func (a *Adapter) retentionPolicy() ArchiveRetentionPolicy {
	if a.RetentionPolicy == nil {
		return ArchiveRetentionPolicy{}
	}
	return a.RetentionPolicy()
}

func (a *Adapter) now() time.Time {
	if a.Now != nil {
		return a.Now().UTC()
	}
	return time.Now().UTC()
}

func (a *Adapter) measureRetentionRow(ctx context.Context, row sessionstore.Metadata) (retentionCandidate, error) {
	candidate := retentionCandidate{meta: row}
	if a.Conversations != nil {
		candidate.messageCount = a.Conversations.CountSessionEvents(ctx, row.ID)
		candidate.transcriptBytes = a.Conversations.SessionStorageBytes(ctx, row.ID)
	}
	if a.AgentHistorySize != nil {
		homeBytes, err := a.AgentHistorySize(row)
		if err != nil {
			return retentionCandidate{}, err
		}
		candidate.homeBytes = homeBytes
	}
	return candidate, nil
}

func (a *Adapter) retentionCandidates(ctx context.Context) ([]retentionCandidate, error) {
	if a.Store == nil {
		return nil, nil
	}
	rows, err := a.Store.ListRetentionCandidates(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: list archive retention candidates: %s", ErrInternal, err)
	}
	candidates := make([]retentionCandidate, 0, len(rows))
	for _, row := range rows {
		candidate, measureErr := a.measureRetentionRow(ctx, row)
		if measureErr != nil {
			return nil, fmt.Errorf("%w: measure agent history for %s: %s", ErrInternal, row.ID, measureErr)
		}
		candidates = append(candidates, candidate)
	}
	return candidates, nil
}

func (a *Adapter) archiveRetentionStats(ctx context.Context) (ArchiveRetentionStats, error) {
	if a.Store == nil {
		return ArchiveRetentionStats{}, nil
	}
	entries, err := a.ListArchived(ctx)
	if err != nil {
		return ArchiveRetentionStats{}, err
	}
	all, err := a.Store.List(ctx)
	if err != nil {
		return ArchiveRetentionStats{}, fmt.Errorf("%w: list archive metadata for storage totals: %s", ErrInternal, err)
	}
	byID := make(map[string]sessionstore.Metadata, len(all))
	for _, row := range all {
		byID[row.ID] = row
	}
	stats := ArchiveRetentionStats{EntryCount: int64(len(entries))}
	for _, entry := range entries {
		stats.MessageCount += entry.MessageCount
		if a.Conversations != nil {
			stats.TranscriptBytes += a.Conversations.SessionStorageBytes(ctx, entry.ID)
		}
		if row, ok := byID[entry.ID]; ok && a.AgentHistorySize != nil {
			size, sizeErr := a.AgentHistorySize(row)
			if sizeErr != nil {
				return ArchiveRetentionStats{}, fmt.Errorf("%w: measure agent history for %s: %s", ErrInternal, row.ID, sizeErr)
			}
			stats.AgentHomeBytes += size
		}
	}
	stats.TotalBytes = stats.TranscriptBytes + stats.AgentHomeBytes
	return stats, nil
}

func (a *Adapter) GetArchiveRetention(ctx context.Context) (ArchiveRetentionSnapshot, error) {
	stats, err := a.archiveRetentionStats(ctx)
	if err != nil {
		return ArchiveRetentionSnapshot{}, err
	}
	return ArchiveRetentionSnapshot{Policy: a.retentionPolicy(), Stats: stats}, nil
}

// PruneArchive is fail-safe by default: apply=false only reports the ordered
// actions. Candidate membership is already constrained by the store's SQL
// query to rows with a non-empty archived_at value.
func (a *Adapter) PruneArchive(ctx context.Context, apply bool) (ArchivePruneResult, error) {
	candidates, err := a.retentionCandidates(ctx)
	if err != nil {
		return ArchivePruneResult{}, err
	}
	before, err := a.archiveRetentionStats(ctx)
	if err != nil {
		return ArchivePruneResult{}, err
	}
	policy := a.retentionPolicy()
	result := ArchivePruneResult{DryRun: !apply, Before: before, After: before}
	now := a.now()
	projectedBytes := before.TotalBytes

	for _, candidate := range candidates {
		archivedAt := candidate.meta.ArchivedAt
		emptyDue := policy.MessageLessAge > 0 && candidate.messageCount == 0 && !archivedAt.After(now.Add(-policy.MessageLessAge))
		homeDue := policy.AgentHomeAge > 0 && !archivedAt.After(now.Add(-policy.AgentHomeAge))
		overSize := policy.MaxBytes > 0 && projectedBytes > policy.MaxBytes
		if candidate.homeBytes > 0 && (homeDue || overSize || emptyDue) {
			action := ArchivePruneAction{SessionID: candidate.meta.ID, Kind: PruneAgentHome, Bytes: candidate.homeBytes}
			if apply {
				if a.PruneAgentHistory == nil {
					return ArchivePruneResult{}, fmt.Errorf("%w: agent-history prune is unavailable", ErrInternal)
				}
				reclaimed, pruneErr := a.PruneAgentHistory(candidate.meta)
				if pruneErr != nil {
					return ArchivePruneResult{}, fmt.Errorf("%w: prune agent history for %s: %s", ErrInternal, candidate.meta.ID, pruneErr)
				}
				action.Bytes = reclaimed
				action.Applied = true
			}
			result.Actions = append(result.Actions, action)
			result.ReclaimedBytes += action.Bytes
			projectedBytes -= candidate.homeBytes
		}

		if emptyDue {
			action := ArchivePruneAction{SessionID: candidate.meta.ID, Kind: PruneTranscript, Bytes: candidate.transcriptBytes}
			if apply {
				if err := a.Delete(ctx, candidate.meta.ID); err != nil {
					return ArchivePruneResult{}, err
				}
				action.Applied = true
			}
			result.Actions = append(result.Actions, action)
			result.ReclaimedBytes += action.Bytes
			projectedBytes -= candidate.transcriptBytes
		}
	}

	if apply {
		result.After, err = a.archiveRetentionStats(ctx)
		if err != nil {
			return ArchivePruneResult{}, err
		}
	}
	return result, nil
}

func (a *Adapter) restoreState(row sessionstore.Metadata, messageCount int64) (RestoreState, string) {
	if row.AgentType == sessionstore.AgentNone && messageCount == 0 {
		return RestoreStateNothingToRestore, "no agent identity or conversation recorded"
	}
	if ok, reason := intsessions.Recoverability(row); !ok {
		return RestoreStateReadOnly, reason
	}
	present := a.AgentHistoryPresent
	if present == nil {
		present = func(meta sessionstore.Metadata) bool {
			if meta.LastRolloutPath == "" {
				return false
			}
			_, err := os.Stat(meta.LastRolloutPath)
			return err == nil
		}
	}
	if !present(row) {
		return RestoreStateReadOnly, "agent history is no longer available on disk"
	}
	return RestoreStateReopenable, ""
}

const trackingGracePeriod = 2 * time.Minute

func isClaudeTrackingDegraded(sess *session.Session, conversations ConversationsStore) bool {
	if conversations == nil || sess.CreatedAt.IsZero() || time.Since(sess.CreatedAt) < trackingGracePeriod {
		return false
	}
	if outputAt := sess.LastFrameAt(); outputAt.IsZero() || !outputAt.After(sess.CreatedAt) {
		return false
	}
	return !conversations.HasConversationAfter(context.Background(), sess.ID, sess.CreatedAt)
}

// provenanceByID snapshots stored provenance keyed by session id. Returns an
// empty map when no store is configured (e.g. minimal test servers).
func (a *Adapter) provenanceByID(ctx context.Context) map[string]sessionstore.Metadata {
	if a.Store == nil {
		return nil
	}
	rows, err := a.Store.List(ctx)
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
func (a *Adapter) RecoveryStatus(ctx context.Context) RecoveryStatus {
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

func (a *Adapter) Get(ctx context.Context, id string) (Session, error) {
	sess, ok := a.Manager.Get(id)
	if !ok {
		return Session{}, fmt.Errorf("session %q: %w", sanitizeID(id), ErrNotFound)
	}
	s := responseToHandlerSession(intsessions.FromSession(sess))
	if a.Store != nil {
		if m, err := a.Store.Get(ctx, id); err == nil {
			s.Origin, s.Owner, s.DisplayLabel = string(m.Origin), m.Owner, m.DisplayLabel
		}
	}
	return s, nil
}

func (a *Adapter) Delete(ctx context.Context, id string) error {
	a.cancelArchiveTimer(id)
	_, managed := a.Manager.Get(id)
	persisted := false
	if a.Store != nil {
		_, err := a.Store.Get(ctx, id)
		persisted = err == nil
	}
	if !managed && !persisted {
		return nil
	}
	if managed {
		if err := a.Manager.Delete(ctx, id); err != nil {
			return fmt.Errorf("delete live session %q: %v: %w", sanitizeID(id), err, ErrInternal)
		}
		a.Metrics.ActiveSessions.Add(-1)
	} else if err := a.Store.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete archived session %q: %v: %w", sanitizeID(id), err, ErrInternal)
	}
	if a.Conversations != nil {
		a.Conversations.DeleteSession(ctx, id)
	}
	if a.CodexCheckpoints != nil {
		_ = a.CodexCheckpoints.DeleteSession(ctx, id)
	}
	if a.AgentCheckpoints != nil {
		_ = a.AgentCheckpoints.DeleteSession(ctx, id)
	}
	a.Events.Emit(events.SessionDeleted, id, nil)
	a.Metrics.SessionsDeleted.Add(1)
	return nil
}

// Archive stops the live process while preserving every durable artifact. It
// is intentionally separate from Delete, whose cascade remains the explicit
// permanent-destruction path.
func (a *Adapter) Archive(ctx context.Context, id string) error {
	if a.Store == nil {
		return fmt.Errorf("session store not configured: %w", ErrInternal)
	}
	if _, err := a.Store.Get(ctx, id); err != nil {
		return fmt.Errorf("no session row with id %q: %w", sanitizeID(id), ErrNotFound)
	}
	if err := a.Store.MarkArchived(ctx, id, time.Now().UTC()); err != nil {
		return fmt.Errorf("mark archived: %v: %w", err, ErrInternal)
	}

	a.archiveMu.Lock()
	if a.archiveTimers == nil {
		a.archiveTimers = make(map[string]*time.Timer)
	}
	if existing := a.archiveTimers[id]; existing != nil {
		existing.Stop()
	}
	delay := a.ArchiveGracePeriod
	if delay == 0 {
		delay = 8 * time.Second
	}
	finalize := func() {
		if err := a.Manager.Archive(context.Background(), id); err != nil {
			a.logger().Printf("archive session %s: finalize: %v", sanitizeID(id), err)
		} else {
			a.Metrics.ActiveSessions.Add(-1)
		}
		a.archiveMu.Lock()
		delete(a.archiveTimers, id)
		a.archiveMu.Unlock()
	}
	if delay < 0 {
		a.archiveMu.Unlock()
		finalize()
		return nil
	}
	a.archiveTimers[id] = time.AfterFunc(delay, finalize)
	a.archiveMu.Unlock()
	return nil
}

// Unarchive clears the archive marker for the short undo path. It does not
// create a process or run an agent resume command.
func (a *Adapter) Unarchive(ctx context.Context, id string) error {
	if a.Store == nil {
		return fmt.Errorf("session store not configured: %w", ErrInternal)
	}
	a.archiveMu.Lock()
	timer := a.archiveTimers[id]
	if timer == nil || !timer.Stop() {
		a.archiveMu.Unlock()
		return fmt.Errorf("session %q archive undo window expired: %w", sanitizeID(id), ErrFailedPrecondition)
	}
	delete(a.archiveTimers, id)
	a.archiveMu.Unlock()
	if err := a.Store.MarkUnarchived(ctx, id); err != nil {
		return fmt.Errorf("unarchive session %q: %v: %w", sanitizeID(id), err, ErrNotFound)
	}
	return nil
}

func (a *Adapter) cancelArchiveTimer(id string) {
	a.archiveMu.Lock()
	defer a.archiveMu.Unlock()
	if timer := a.archiveTimers[id]; timer != nil {
		timer.Stop()
		delete(a.archiveTimers, id)
	}
}

// -----------------------------------------------------------------------------
// Recovery
// -----------------------------------------------------------------------------

func (a *Adapter) ListRecoverable(ctx context.Context) ([]RecoverableSession, error) {
	if a.Store == nil {
		return nil, nil
	}
	rows, err := a.Store.ListRecoverable(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInternal, err.Error())
	}
	out := make([]RecoverableSession, 0, len(rows))
	panes := map[string]intworkspace.Pane{}
	groups := map[string]string{}
	if a.Workspace != nil {
		layout, err := a.Workspace.GetLayout(ctx)
		if err != nil {
			return nil, fmt.Errorf("%w: load workspace identity: %s", ErrInternal, err.Error())
		}
		for _, pane := range layout.Panes {
			panes[pane.SessionID] = pane
		}
		for _, group := range layout.Groups {
			groups[group.ID] = group.Name
		}
	}
	for _, m := range rows {
		r := toHandlerRecoverable(m)
		if pane, ok := panes[m.ID]; ok {
			r.PaneName, r.HeaderColor, r.GroupName = pane.Name, pane.HeaderColor, groups[pane.GroupID]
		}
		out = append(out, r)
	}
	return out, nil
}

func (a *Adapter) DismissRecoverable(ctx context.Context, id string) error {
	if a.Store == nil {
		return fmt.Errorf("session store not configured: %w", ErrNotFound)
	}
	meta, err := a.Store.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("no session row with id %q: %w", sanitizeID(id), ErrNotFound)
	}
	if meta.Status != sessionstore.StatusAwaitingRecovery {
		return fmt.Errorf("session %q is in status %q, not awaiting_recovery: %w", sanitizeID(id), meta.Status, ErrFailedPrecondition)
	}
	if err := a.Store.MarkDismissed(ctx, id, ""); err != nil {
		return fmt.Errorf("mark dismissed: %v: %w", err, ErrInternal)
	}
	return nil
}

func (a *Adapter) Recover(ctx context.Context, in RecoverInput) (RecoverResult, error) {
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

	old, err := a.Store.Get(ctx, oldID)
	if err != nil {
		return RecoverResult{}, fmt.Errorf("no session row with id %q: %w", sanitizeID(oldID), ErrNotFound)
	}
	isCrashRecovery := old.Status == sessionstore.StatusAwaitingRecovery
	isArchived := !old.ArchivedAt.IsZero()
	if !isCrashRecovery && !isArchived {
		return RecoverResult{}, fmt.Errorf("session %q is neither awaiting_recovery nor archived: %w", sanitizeID(oldID), ErrFailedPrecondition)
	}
	// Single source of truth for recoverability (and its precise refusal
	// reasons) so every agent type — codex, claude, opencode, grok — is gated
	// identically here and in the recoverable-sessions listing.
	if isArchived {
		messageCount := int64(0)
		if a.Conversations != nil {
			messageCount = a.Conversations.CountSessionEvents(ctx, old.ID)
		}
		if state, reason := a.restoreState(old, messageCount); state != RestoreStateReopenable {
			return RecoverResult{}, fmt.Errorf("%s: %w", reason, ErrFailedPrecondition)
		}
	} else if ok, reason := intsessions.Recoverability(old); !ok {
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
	newSess, err := a.Manager.CreateWithWorkingDir(ctx, old.Shell, cols, rows, backend.Persistent, &pol, old.CWD)
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

	_ = a.Store.UpdateAgentInfo(ctx, newSess.ID, sessionstore.AgentInfo{
		AgentType:      old.AgentType,
		AgentSessionID: old.AgentSessionID,
		LaunchCommand:  old.LaunchCommand,
		CWD:            old.CWD,
	})
	// Carry provenance onto the recovered session so it keeps its original
	// origin/owner/label in the sidebar.
	_ = a.Store.SetProvenance(ctx, newSess.ID, old.Origin, old.Owner, old.DisplayLabel)
	if a.Workspace != nil {
		if err := a.Workspace.ReassignPane(ctx, oldID, newSess.ID); err != nil {
			a.logger().Printf("recover[%s -> %s]: migrate workspace pane: %v", oldID, newSess.ID, err)
			return RecoverResult{}, fmt.Errorf("migrate workspace pane: %v: %w", err, ErrInternal)
		}
	}

	// Carry the prior conversation history onto the new session id so the
	// messages view is populated after reattach. Best-effort: a copy failure
	// must not abort recovery — the agent resume is the critical path.
	messagesCopied := false
	if a.Conversations != nil {
		if err := a.Conversations.CopySession(ctx, oldID, newSess.ID); err != nil {
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

	if err := a.Store.MarkDismissed(ctx, oldID, newSess.ID); err != nil {
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

func (a *Adapter) GetPolicy(ctx context.Context, id string) (PolicyView, error) {
	sess, ok := a.Manager.Get(id)
	if !ok {
		return PolicyView{}, fmt.Errorf("session %q: %w", sanitizeID(id), ErrNotFound)
	}
	return policyViewFor(sess, sess.GetPolicy()), nil
}

func (a *Adapter) UpdatePolicy(ctx context.Context, id string, in Policy) (PolicyView, error) {
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
		_ = a.Store.UpdatePolicy(ctx, sess.ID, pol)
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
		Recovered:       r.Recovered,
		Origin:          r.Origin,
		Owner:           r.Owner,
		DisplayLabel:    r.DisplayLabel,
		Target:          r.Target,
	}
}

func handlerSessionToResponse(s Session) intsessions.Response {
	return intsessions.Response{
		ID:              s.ID,
		Shell:           s.Shell,
		CreatedAt:       s.CreatedAt,
		Cols:            s.Cols,
		Rows:            s.Rows,
		Backend:         backend.ID(s.Backend),
		SurvivesRestart: s.SurvivesRestart,
		Policy:          policy.Policy{Mode: policy.Mode(s.Policy.Mode), Duration: s.Policy.Duration},
		Recovered:       s.Recovered,
		Origin:          s.Origin,
		Owner:           s.Owner,
		DisplayLabel:    s.DisplayLabel,
		Target:          s.Target,
	}
}

func createFingerprint(in CreateInput) string {
	payload, _ := json.Marshal(struct {
		Shell                string
		Cols                 int
		Rows                 int
		Backend              string
		Policy               Policy
		HasPolicy            bool
		LaunchCommand        string
		ExecuteLaunchCommand bool
		AgentType            string
		Origin               string
		Owner                string
		DisplayLabel         string
		TargetID             string
		WorkingDir           string
		TmuxMouseMode        bool
	}{
		Shell: in.Shell, Cols: in.Cols, Rows: in.Rows, Backend: in.Backend,
		Policy: in.Policy, HasPolicy: in.HasPolicy, LaunchCommand: in.LaunchCommand,
		ExecuteLaunchCommand: in.ExecuteLaunchCommand, AgentType: in.AgentType,
		Origin: in.Origin, Owner: in.Owner, DisplayLabel: in.DisplayLabel,
		TargetID: in.TargetID, WorkingDir: in.WorkingDir, TmuxMouseMode: in.TmuxMouseMode,
	})
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
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
