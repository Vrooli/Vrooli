package sessions

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/nodereach"
	"github.com/vrooli/api-core/scopecatalog"
	"web-console/internal/backend"
	"web-console/internal/continuity"
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
	DeleteSession(ctx context.Context, id string) error
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
	RemoveAgentHomes    func(sessionID string) error
	Now                 func() time.Time
	Remote              RemoteService
	LifecycleLedger     continuity.Ledger
	ContinuityCatalog   interface {
		EnqueueTombstone(context.Context, string) error
		List(context.Context, string, int) ([]continuity.CatalogRecord, bool, error)
		ListAll(context.Context, string) ([]continuity.CatalogRecord, error)
	}
}

func (a *Adapter) logger() *log.Logger {
	if a.Logger != nil {
		return a.Logger
	}
	return log.Default()
}

func (a *Adapter) updateCatalogLifecycle(ctx context.Context, sessionID string, state continuity.State) error {
	if a.ContinuityCatalog == nil {
		return nil
	}
	catalog, ok := a.ContinuityCatalog.(continuity.LifecycleCatalog)
	if !ok {
		return nil
	}
	if err := catalog.UpdateLifecycleState(ctx, sessionID, state); err != nil {
		return fmt.Errorf("update continuity catalog for %s: %w", sanitizeID(sessionID), err)
	}
	return nil
}

func (a *Adapter) beginLifecycleReceipt(ctx context.Context, command, sessionID string, from, to continuity.State) (continuity.Receipt, bool, error) {
	if err := continuity.ValidateState(from); err != nil {
		return continuity.Receipt{}, false, fmt.Errorf("invalid lifecycle receipt prior state: %w", err)
	}
	if err := continuity.ValidateState(to); err != nil {
		return continuity.Receipt{}, false, fmt.Errorf("invalid lifecycle receipt resulting state: %w", err)
	}
	if a.LifecycleLedger == nil {
		return continuity.Receipt{}, false, nil
	}
	op := lifecycleOperationID(ctx, command, sessionID)
	receipt, err := a.LifecycleLedger.Put(ctx, continuity.Receipt{OperationID: op, SessionID: sessionID, ActorKind: "operator", Command: command, FromState: from, ToState: to, ReasonCode: command, Status: "pending", CreatedAt: time.Now().UTC()})
	if err != nil {
		if a.Metrics != nil {
			a.Metrics.ContinuityFailures.Add(1)
		}
		return continuity.Receipt{}, false, err
	}
	if a.Metrics != nil {
		a.Metrics.ContinuityReceipts.Add(1)
	}
	// A recorded failure is not a verdict on every future attempt. The web UI
	// derives this operation id from the session id alone, so replaying a
	// failure here made the operator's Close button permanently dead for that
	// session — one refusal, then the same refusal forever, with no way to ask
	// again. Reopening lets this attempt run and record its own outcome, the
	// way catalog reconciliation already retries its failed receipts.
	if receipt.Status == "failed" {
		retryable, ok := a.LifecycleLedger.(continuity.RetryableLedger)
		if !ok {
			return receipt, true, nil
		}
		reopened, reopenErr := retryable.Reopen(ctx, op)
		if reopenErr != nil {
			return continuity.Receipt{}, false, fmt.Errorf("reopen failed %s receipt %q: %w", command, op, reopenErr)
		}
		receipt = reopened
	}
	return receipt, receipt.Status != "pending", nil
}

func lifecycleOperationID(ctx context.Context, command, sessionID string) string {
	op := OperationID(ctx)
	if op == "" {
		op = "web-console:" + command + ":" + sessionID
	}
	return op
}

func (a *Adapter) replayLifecycleReceipt(ctx context.Context, command, sessionID string) (bool, error) {
	if a.LifecycleLedger == nil {
		return false, nil
	}
	receipt, err := a.LifecycleLedger.Get(ctx, lifecycleOperationID(ctx, command, sessionID))
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return true, fmt.Errorf("read %s receipt: %v: %w", command, err, ErrInternal)
	}
	// Only a decided success short-circuits. A failed receipt falls through so
	// the caller re-runs the command; beginLifecycleReceipt reopens the receipt
	// so the retry records its own outcome rather than inheriting the old one.
	if receipt.Status == "failed" {
		return false, nil
	}
	return true, replayLifecycleResult(receipt, command)
}

func (a *Adapter) replayRecoverReceipt(ctx context.Context, oldID string) (RecoverResult, bool, error) {
	if a.LifecycleLedger == nil {
		return RecoverResult{}, false, nil
	}
	receipt, err := a.LifecycleLedger.Get(ctx, lifecycleOperationID(ctx, "recover", oldID))
	if errors.Is(err, sql.ErrNoRows) {
		return RecoverResult{}, false, nil
	}
	if err != nil {
		return RecoverResult{}, true, fmt.Errorf("read recover receipt: %v: %w", err, ErrInternal)
	}
	if receipt.Status != "succeeded" {
		return RecoverResult{}, true, replayLifecycleResult(receipt, "recover")
	}
	if receipt.ActorID == "" {
		return RecoverResult{}, true, fmt.Errorf("recover operation %q has no replacement session: %w", receipt.OperationID, ErrFailedPrecondition)
	}
	return RecoverResult{OldSessionID: oldID, NewSessionID: receipt.ActorID}, true, nil
}

func (a *Adapter) finishLifecycleReceipt(ctx context.Context, receipt continuity.Receipt, operationErr error) error {
	if a.LifecycleLedger == nil || receipt.OperationID == "" {
		return nil
	}
	status, code := "succeeded", ""
	if operationErr != nil {
		status, code = "failed", "lifecycle_operation_failed"
	}
	_, err := a.LifecycleLedger.Complete(ctx, receipt.OperationID, status, code, time.Now().UTC())
	if a.Metrics != nil {
		if err != nil || operationErr != nil {
			a.Metrics.ContinuityFailures.Add(1)
		}
	}
	return err
}

func (a *Adapter) finishLifecycleResult(ctx context.Context, receipt continuity.Receipt, operationErr error, command string) error {
	if finishErr := a.finishLifecycleReceipt(ctx, receipt, operationErr); finishErr != nil {
		completionErr := fmt.Errorf("complete %s receipt: %v: %w", command, finishErr, ErrInternal)
		if operationErr != nil {
			return errors.Join(operationErr, completionErr)
		}
		return completionErr
	}
	return operationErr
}

func replayLifecycleResult(receipt continuity.Receipt, command string) error {
	switch receipt.Status {
	case "succeeded":
		return nil
	case "failed":
		message := fmt.Sprintf("%s operation %q previously failed", command, receipt.OperationID)
		if receipt.ErrorCode != "" {
			message += ": " + receipt.ErrorCode
		}
		return fmt.Errorf("%s: %w", message, ErrFailedPrecondition)
	default:
		return fmt.Errorf("%s operation %q has unresolved status %q: %w", command, receipt.OperationID, receipt.Status, ErrFailedPrecondition)
	}
}

// recordLifecycleFailure makes a rejected lifecycle command observable too.
// A failed precondition does not change the source state, but it still needs a
// durable, idempotent result so an operator can distinguish a safe refusal
// from an unrecorded or interrupted request.
func (a *Adapter) recordLifecycleFailure(ctx context.Context, command, sessionID string, from, to continuity.State, operationErr error) error {
	if a.LifecycleLedger == nil {
		return operationErr
	}
	receipt, replay, err := a.beginLifecycleReceipt(ctx, command, sessionID, from, to)
	if err != nil {
		return errors.Join(operationErr, fmt.Errorf("record %s refusal: %w", command, err))
	}
	if replay {
		return operationErr
	}
	if err := a.finishLifecycleReceipt(ctx, receipt, operationErr); err != nil {
		return errors.Join(operationErr, fmt.Errorf("complete %s refusal receipt: %w", command, err))
	}
	return operationErr
}

func lifecycleStateForMetadata(meta sessionstore.Metadata) continuity.State {
	if !meta.ArchivedAt.IsZero() {
		return continuity.StateArchived
	}
	if meta.Status == sessionstore.StatusAwaitingRecovery {
		return continuity.StateRecoverable
	}
	return continuity.StateLive
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
			// Remote creation crosses the browser-facing error boundary just like
			// local creation. Keep Bridge/nodereach details out of the Connect
			// response and preserve the actionable scope/recovery classification.
			return Session{}, mapCreateError(err)
		}
		if a.Store != nil && (in.LaunchCommand != "" || in.AgentType != "") {
			agentType := intsessions.NormalizeAgentType(in.AgentType)
			if err := a.Store.UpdateAgentInfo(ctx, created.ID, sessionstore.AgentInfo{AgentType: agentType, LaunchCommand: in.LaunchCommand}); err != nil {
				return Session{}, fmt.Errorf("persist remote session metadata for %q: %v: %w", sanitizeID(created.ID), err, ErrInternal)
			}
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

	// The manager may already share this store and have persisted the base row
	// during creation. Ensure the adapter-owned store has the row as well before
	// applying enrichment: a provider can exit immediately after launch, and
	// enrichment must remain attributable even when the runtime handle is gone.
	if a.Store != nil {
		if err := a.Store.Save(ctx, sessionstore.Metadata{
			ID:       sess.ID,
			Backend:  sess.Backend,
			Shell:    sess.Shell,
			Cols:     sess.Cols,
			Rows:     sess.Rows,
			Policy:   sess.GetPolicy(),
			Created:  sess.CreatedAt,
			Detached: sess.Backend == backend.Persistent,
			CWD:      in.WorkingDir,
		}); err != nil {
			return a.failCreateAfterPersistenceError(ctx, sess.ID, err)
		}
	}

	if a.Store != nil && (in.LaunchCommand != "" || in.AgentType != "") {
		agentType := intsessions.NormalizeAgentType(in.AgentType)
		if err := a.Store.UpdateAgentInfo(ctx, sess.ID, sessionstore.AgentInfo{
			AgentType:     agentType,
			LaunchCommand: in.LaunchCommand,
		}); err != nil {
			return a.failCreateAfterPersistenceError(ctx, sess.ID, err)
		}
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
		if err := a.Store.SetProvenance(ctx, sess.ID, origin, owner, displayLabel); err != nil {
			return a.failCreateAfterPersistenceError(ctx, sess.ID, err)
		}
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

// failCreateAfterPersistenceError makes a partially persisted create
// fail-closed. The runtime and its base metadata row were created before the
// adapter could persist the richer identity fields; returning success here
// would expose an untracked pane to recovery and reconciliation.
func (a *Adapter) failCreateAfterPersistenceError(ctx context.Context, sessionID string, persistErr error) (Session, error) {
	cleanupErrs := []error{fmt.Errorf("persist session metadata for %q: %v", sanitizeID(sessionID), persistErr)}
	if a.Manager != nil {
		if err := a.Manager.Delete(ctx, sessionID); err != nil {
			cleanupErrs = append(cleanupErrs, fmt.Errorf("terminate partially created session %q: %w", sanitizeID(sessionID), err))
		}
	}
	if a.Store != nil {
		if err := a.Store.Delete(ctx, sessionID); err != nil {
			cleanupErrs = append(cleanupErrs, fmt.Errorf("remove partially persisted session %q: %w", sanitizeID(sessionID), err))
		}
	}
	return Session{}, fmt.Errorf("create session failed closed: %w", errors.Join(cleanupErrs...))
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
		if activity, ok := sess.Activity(); ok {
			s.Activity = activityToProto(s.ID, activity)
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

	// Catalog records are the recovery projection of durable conversation
	// evidence. Include records that have no session-store row so a missing live
	// projection cannot make an intact transcript disappear from the archive
	// drawer. They are intentionally read-only here: without session metadata
	// there is no safe process/recovery operation to expose.
	if a.ContinuityCatalog != nil {
		catalogRows, catalogErr := a.ContinuityCatalog.ListAll(ctx, "")
		if catalogErr != nil {
			return nil, fmt.Errorf("%w: list continuity catalog: %s", ErrInternal, catalogErr)
		}
		for _, record := range catalogRows {
			if _, exists := collapsed[record.SessionID]; exists || record.LifecycleState == continuity.StateDeleted {
				continue
			}
			collapsed[record.SessionID] = sessionstore.Metadata{
				ID:             record.SessionID,
				AgentType:      sessionstore.Agent(record.AgentType),
				AgentSessionID: record.AgentSessionID,
				CWD:            record.CWD,
				Created:        record.CreatedAt,
				LastActivityAt: record.LastActivityAt,
				ArchivedAt:     record.LastActivityAt,
				Status:         sessionstore.StatusDismissed,
			}
		}
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
		if _, hasSessionRow := byID[row.ID]; !hasSessionRow {
			entry.RestoreState = RestoreStateReadOnly
			entry.RestoreStateReason = "session metadata is absent; transcript remains available for inspection, but process recovery requires a native session record"
		}
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

func (a *Adapter) pruneAgentHistoryWithReceipt(ctx context.Context, candidate retentionCandidate) (int64, error) {
	if a.PruneAgentHistory == nil {
		return 0, fmt.Errorf("%w: agent-history prune is unavailable", ErrInternal)
	}
	if a.LifecycleLedger == nil {
		return a.PruneAgentHistory(candidate.meta)
	}
	receiptCtx := ctx
	if operationID := OperationID(ctx); operationID != "" {
		receiptCtx = WithOperationID(ctx, operationID+":agent-home:"+candidate.meta.ID)
	}
	receipt, replay, err := a.beginLifecycleReceipt(receiptCtx, "retention_agent_home", candidate.meta.ID, continuity.StateArchived, continuity.StateArchived)
	if err != nil {
		return 0, fmt.Errorf("begin agent-history retention receipt: %w", err)
	}
	if replay {
		if err := replayLifecycleResult(receipt, "retention_agent_home"); err != nil {
			return 0, err
		}
		return candidate.homeBytes, nil
	}
	reclaimed, pruneErr := a.PruneAgentHistory(candidate.meta)
	if finishErr := a.finishLifecycleReceipt(receiptCtx, receipt, pruneErr); finishErr != nil {
		if pruneErr != nil {
			return 0, errors.Join(pruneErr, finishErr)
		}
		return 0, fmt.Errorf("complete agent-history retention receipt: %w", finishErr)
	}
	if pruneErr != nil {
		return 0, pruneErr
	}
	return reclaimed, nil
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
				reclaimed, pruneErr := a.pruneAgentHistoryWithReceipt(ctx, candidate)
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
				deleteCtx := ctx
				if operationID := OperationID(ctx); operationID != "" {
					// A prune request can contain multiple destructive child
					// operations. Give each child its own durable receipt so one
					// candidate cannot consume the parent idempotency key and
					// accidentally suppress later deletions.
					deleteCtx = WithOperationID(ctx, operationID+":delete:"+candidate.meta.ID)
				}
				if err := a.Delete(deleteCtx, candidate.meta.ID); err != nil {
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
	fromState := continuity.StateLive
	managed := false
	if a.Manager != nil {
		_, managed = a.Manager.Get(id)
	}
	if !managed && a.Store != nil {
		if meta, getErr := a.Store.Get(ctx, id); getErr == nil {
			fromState = lifecycleStateForMetadata(meta)
		}
	}
	receipt, replay, err := a.beginLifecycleReceipt(ctx, "delete", id, fromState, continuity.StateDeleted)
	if err != nil {
		return fmt.Errorf("begin delete receipt: %v: %w", err, ErrInternal)
	}
	if replay {
		return replayLifecycleResult(receipt, "delete")
	}
	operationErr := func() error {
		managed := false
		if a.Manager != nil {
			_, managed = a.Manager.Get(id)
		}
		persisted := false
		if a.Store != nil {
			_, err := a.Store.Get(ctx, id)
			persisted = err == nil
		}
		if !managed && !persisted {
			return nil
		}
		if a.ContinuityCatalog != nil {
			if err := a.ContinuityCatalog.EnqueueTombstone(ctx, id); err != nil {
				return fmt.Errorf("queue external search tombstone: %v: %w", err, ErrInternal)
			}
		}
		if managed {
			if err := a.Manager.Delete(ctx, id); err != nil {
				return fmt.Errorf("delete live session %q: %v: %w", sanitizeID(id), err, ErrInternal)
			}
			a.Metrics.ActiveSessions.Add(-1)
		}
		if a.Conversations != nil {
			if err := a.Conversations.DeleteSession(ctx, id); err != nil {
				return fmt.Errorf("delete conversation evidence for %q: %v: %w", sanitizeID(id), err, ErrInternal)
			}
		}
		if a.CodexCheckpoints != nil {
			if err := a.CodexCheckpoints.DeleteSession(ctx, id); err != nil {
				return fmt.Errorf("delete Codex checkpoint for %q: %v: %w", sanitizeID(id), err, ErrInternal)
			}
		}
		if a.AgentCheckpoints != nil {
			if err := a.AgentCheckpoints.DeleteSession(ctx, id); err != nil {
				return fmt.Errorf("delete agent checkpoint for %q: %v: %w", sanitizeID(id), err, ErrInternal)
			}
		}
		if a.RemoveAgentHomes != nil {
			if err := a.RemoveAgentHomes(id); err != nil {
				return fmt.Errorf("delete agent homes for %q: %v: %w", sanitizeID(id), err, ErrInternal)
			}
		}
		// Keep the metadata row until every durable evidence source has
		// acknowledged deletion. If any earlier step fails, the row remains
		// visible so the explicit delete can be retried instead of leaving a
		// silent metadata orphan.
		if persisted && a.Store != nil {
			if err := a.Store.Delete(ctx, id); err != nil {
				return fmt.Errorf("delete archived session %q: %v: %w", sanitizeID(id), err, ErrInternal)
			}
		}
		a.Events.Emit(events.SessionDeleted, id, nil)
		a.Metrics.SessionsDeleted.Add(1)
		return nil
	}()
	return a.finishLifecycleResult(ctx, receipt, operationErr, "delete")
}

// Archive stops the live process while preserving every durable artifact. It
// is intentionally separate from Delete, whose cascade remains the explicit
// permanent-destruction path.
func (a *Adapter) Archive(ctx context.Context, id string) error {
	receipt, replay, err := a.beginLifecycleReceipt(ctx, "archive", id, continuity.StateLive, continuity.StateArchived)
	if err != nil {
		return fmt.Errorf("begin archive receipt: %v: %w", err, ErrInternal)
	}
	if replay {
		return replayLifecycleResult(receipt, "archive")
	}
	if a.Store == nil {
		operationErr := fmt.Errorf("session store not configured: %w", ErrInternal)
		return a.finishLifecycleResult(ctx, receipt, operationErr, "archive")
	}
	// Archive has to be able to close anything List showed. A live session can
	// outlive its metadata row — a routed test lease writes its rows to a
	// database that is later discarded while the PTY keeps running — and
	// refusing here left the operator looking at a row they could see, could
	// not close, and whose failure the ledger then replayed forever. Delete
	// already treats "managed OR persisted" as enough; Archive now agrees, and
	// re-persists the row from the live session so the archive is real (it
	// lands in the archive view and can be unarchived) rather than a silent
	// stop with no durable trace.
	if _, err := a.Store.Get(ctx, id); err != nil {
		if !a.persistLiveSessionRow(ctx, id) {
			operationErr := fmt.Errorf("no session row with id %q: %w", sanitizeID(id), ErrNotFound)
			return a.finishLifecycleResult(ctx, receipt, operationErr, "archive")
		}
	}
	var operationErr error
	if err := a.Store.MarkArchived(ctx, id, time.Now().UTC()); err != nil {
		operationErr = fmt.Errorf("mark archived: %v: %w", err, ErrInternal)
	} else if err := a.updateCatalogLifecycle(ctx, id, continuity.StateArchived); err != nil {
		operationErr = fmt.Errorf("mark continuity catalog archived: %v: %w", err, ErrInternal)
	} else if a.Manager != nil {
		if _, managed := a.Manager.Get(id); managed {
			if err := a.Manager.Archive(ctx, id); err != nil {
				operationErr = fmt.Errorf("stop archived session %q: %v: %w", sanitizeID(id), err, ErrInternal)
			} else {
				a.Metrics.ActiveSessions.Add(-1)
			}
		}
	}
	return a.finishLifecycleResult(ctx, receipt, operationErr, "archive")
}

// persistLiveSessionRow re-creates the metadata row for a session the manager
// is still running but the store has no row for. It reports whether the row
// now exists. The live session is the authority for everything the row needs
// except provenance, which is genuinely unknown here and stays empty rather
// than being invented.
func (a *Adapter) persistLiveSessionRow(ctx context.Context, id string) bool {
	if a.Manager == nil || a.Store == nil {
		return false
	}
	sess, ok := a.Manager.Get(id)
	if !ok {
		return false
	}
	if err := a.Store.Save(ctx, sessionstore.Metadata{
		ID:       sess.ID,
		Backend:  sess.Backend,
		Shell:    sess.Shell,
		Cols:     sess.Cols,
		Rows:     sess.Rows,
		Policy:   sess.GetPolicy(),
		Created:  sess.CreatedAt,
		Detached: sess.Backend == backend.Persistent,
	}); err != nil {
		a.logger().Printf("archive[%s]: re-persist metadata for orphaned live session: %v", sanitizeID(id), err)
		return false
	}
	return true
}

// Unarchive clears the archive marker for the short undo path. It does not
// create a process or run an agent resume command.
func (a *Adapter) Unarchive(ctx context.Context, id string) error {
	if replay, err := a.replayLifecycleReceipt(ctx, "unarchive", id); replay {
		return err
	}
	if a.Store == nil {
		return a.recordLifecycleFailure(ctx, "unarchive", id, continuity.StateArchived, continuity.StateLive,
			fmt.Errorf("session store not configured: %w", ErrInternal))
	}
	meta, err := a.Store.Get(ctx, id)
	if err != nil {
		return a.recordLifecycleFailure(ctx, "unarchive", id, continuity.StateArchived, continuity.StateLive,
			fmt.Errorf("session %q: %w", sanitizeID(id), ErrNotFound))
	}
	if meta.ArchivedAt.IsZero() {
		return a.recordLifecycleFailure(ctx, "unarchive", id, lifecycleStateForMetadata(meta), continuity.StateLive,
			fmt.Errorf("session %q is not archived: %w", sanitizeID(id), ErrFailedPrecondition))
	}
	receipt, replay, err := a.beginLifecycleReceipt(ctx, "unarchive", id, continuity.StateArchived, continuity.StateLive)
	if err != nil {
		return fmt.Errorf("begin unarchive receipt: %v: %w", err, ErrInternal)
	}
	if replay {
		return replayLifecycleResult(receipt, "unarchive")
	}
	if err := a.Store.MarkUnarchived(ctx, id); err != nil {
		opErr := fmt.Errorf("unarchive session %q: %v: %w", sanitizeID(id), err, ErrNotFound)
		return a.finishLifecycleResult(ctx, receipt, opErr, "unarchive")
	}
	if err := a.updateCatalogLifecycle(ctx, id, continuity.StateLive); err != nil {
		opErr := fmt.Errorf("unarchive continuity catalog %q: %v: %w", sanitizeID(id), err, ErrInternal)
		return a.finishLifecycleResult(ctx, receipt, opErr, "unarchive")
	}
	return a.finishLifecycleResult(ctx, receipt, nil, "unarchive")
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
	if replay, err := a.replayLifecycleReceipt(ctx, "dismiss", id); replay {
		return err
	}
	if a.Store == nil {
		return a.recordLifecycleFailure(ctx, "dismiss", id, continuity.StateRecoverable, continuity.StateArchived,
			fmt.Errorf("session store not configured: %w", ErrNotFound))
	}
	meta, err := a.Store.Get(ctx, id)
	if err != nil {
		return a.recordLifecycleFailure(ctx, "dismiss", id, continuity.StateRecoverable, continuity.StateArchived,
			fmt.Errorf("no session row with id %q: %w", sanitizeID(id), ErrNotFound))
	}
	if meta.Status != sessionstore.StatusAwaitingRecovery {
		return a.recordLifecycleFailure(ctx, "dismiss", id, lifecycleStateForMetadata(meta), continuity.StateArchived,
			fmt.Errorf("session %q is in status %q, not awaiting_recovery: %w", sanitizeID(id), meta.Status, ErrFailedPrecondition))
	}
	receipt, replay, err := a.beginLifecycleReceipt(ctx, "dismiss", id, continuity.StateRecoverable, continuity.StateArchived)
	if err != nil {
		return fmt.Errorf("begin dismiss receipt: %v: %w", err, ErrInternal)
	}
	if replay {
		return replayLifecycleResult(receipt, "dismiss")
	}
	if err := a.Store.MarkDismissed(ctx, id, ""); err != nil {
		opErr := fmt.Errorf("mark dismissed: %v: %w", err, ErrInternal)
		return a.finishLifecycleResult(ctx, receipt, opErr, "dismiss")
	}
	if err := a.updateCatalogLifecycle(ctx, id, continuity.StateArchived); err != nil {
		opErr := fmt.Errorf("dismiss continuity catalog %q: %v: %w", sanitizeID(id), err, ErrInternal)
		return a.finishLifecycleResult(ctx, receipt, opErr, "dismiss")
	}
	return a.finishLifecycleResult(ctx, receipt, nil, "dismiss")
}

func (a *Adapter) Recover(ctx context.Context, in RecoverInput) (res RecoverResult, err error) {
	oldID := in.ID
	if replay, ok, replayErr := a.replayRecoverReceipt(ctx, oldID); ok {
		return replay, replayErr
	}
	if a.Store == nil {
		failure := fmt.Errorf("session store not configured: %w", ErrNotFound)
		return RecoverResult{}, a.recordLifecycleFailure(ctx, "recover", oldID, continuity.StateRecoverable, continuity.StateLive, failure)
	}

	if in.IdempotencyKey != "" {
		if a.Idempotency != nil {
			if cached, ok := a.Idempotency.Get("recover:" + oldID + ":" + in.IdempotencyKey); ok {
				return RecoverResult{
					OldSessionID:    oldID,
					NewSessionID:    cached.ID,
					CodexHomeCopied: false,
				}, nil
			}
		}
	}

	old, err := a.Store.Get(ctx, oldID)
	if err != nil {
		failure := fmt.Errorf("no session row with id %q: %w", sanitizeID(oldID), ErrNotFound)
		return RecoverResult{}, a.recordLifecycleFailure(ctx, "recover", oldID, continuity.StateRecoverable, continuity.StateLive, failure)
	}
	isCrashRecovery := old.Status == sessionstore.StatusAwaitingRecovery
	isArchived := !old.ArchivedAt.IsZero()
	if !isCrashRecovery && !isArchived {
		failure := fmt.Errorf("session %q is neither awaiting_recovery nor archived: %w", sanitizeID(oldID), ErrFailedPrecondition)
		return RecoverResult{}, a.recordLifecycleFailure(ctx, "recover", oldID, continuity.StateLive, continuity.StateLive, failure)
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
			failure := fmt.Errorf("%s: %w", reason, ErrFailedPrecondition)
			return RecoverResult{}, a.recordLifecycleFailure(ctx, "recover", oldID, continuity.StateArchived, continuity.StateLive, failure)
		}
	} else if ok, reason := intsessions.Recoverability(old); !ok {
		failure := fmt.Errorf("%s: %w", reason, ErrFailedPrecondition)
		return RecoverResult{}, a.recordLifecycleFailure(ctx, "recover", oldID, continuity.StateRecoverable, continuity.StateLive, failure)
	}
	fromState := continuity.StateRecoverable
	if isArchived {
		fromState = continuity.StateArchived
	}
	receipt, replay, err := a.beginLifecycleReceipt(ctx, "recover", oldID, fromState, continuity.StateLive)
	if err != nil {
		return RecoverResult{}, fmt.Errorf("begin recover receipt: %v: %w", err, ErrInternal)
	}
	if replay {
		return RecoverResult{}, fmt.Errorf("recovery operation %q is already recorded; retry with its original idempotency key: %w", receipt.OperationID, ErrFailedPrecondition)
	}
	defer func() {
		if finishErr := a.finishLifecycleReceipt(ctx, receipt, err); finishErr != nil {
			completionErr := fmt.Errorf("complete recover receipt: %v: %w", finishErr, ErrInternal)
			res = RecoverResult{}
			if err != nil {
				err = errors.Join(err, completionErr)
			} else {
				err = completionErr
			}
		}
	}()

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

	if err := a.Store.UpdateAgentInfo(ctx, newSess.ID, sessionstore.AgentInfo{
		AgentType:      old.AgentType,
		AgentSessionID: old.AgentSessionID,
		LaunchCommand:  old.LaunchCommand,
		CWD:            old.CWD,
	}); err != nil {
		return RecoverResult{}, fmt.Errorf("persist recovered agent identity: %v: %w", err, ErrInternal)
	}
	// Carry provenance onto the recovered session so it keeps its original
	// origin/owner/label in the sidebar.
	if err := a.Store.SetProvenance(ctx, newSess.ID, old.Origin, old.Owner, old.DisplayLabel); err != nil {
		return RecoverResult{}, fmt.Errorf("persist recovered provenance: %v: %w", err, ErrInternal)
	}
	if a.Workspace != nil {
		if err := a.Workspace.ReassignPane(ctx, oldID, newSess.ID); err != nil {
			a.logger().Printf("recover[%s -> %s]: migrate workspace pane: %v", oldID, newSess.ID, err)
			return RecoverResult{}, fmt.Errorf("migrate workspace pane: %v: %w", err, ErrInternal)
		}
		// A role points at a session id. Recovery mints a new one, so the
		// role has to follow the pane or it would keep naming a session that
		// no longer exists — and a handoff aimed at that role would be
		// delivered to nothing. Fatal for the same reason the pane move is:
		// a half-migrated workspace identity is worse than a failed recovery
		// the operator can retry. No-op when the session backs no role.
		if err := a.Workspace.ReassignRoleSession(ctx, oldID, newSess.ID); err != nil {
			a.logger().Printf("recover[%s -> %s]: migrate workspace role: %v", oldID, newSess.ID, err)
			return RecoverResult{}, fmt.Errorf("migrate workspace role: %v: %w", err, ErrInternal)
		}
	}

	// Carry the prior conversation history onto the new session id so the
	// messages view is populated after reattach. A recovery that claims success
	// without preserving the transcript would make the durable evidence appear
	// lost, so copy failure is fatal and the receipt records the failure.
	messagesCopied := false
	if a.Conversations != nil {
		if err := a.Conversations.CopySession(ctx, oldID, newSess.ID); err != nil {
			return RecoverResult{}, fmt.Errorf("copy conversation history: %v: %w", err, ErrInternal)
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
		return RecoverResult{}, fmt.Errorf("mark recovered source dismissed: %v: %w", err, ErrInternal)
	}
	if err := a.updateCatalogLifecycle(ctx, oldID, continuity.StateArchived); err != nil {
		return RecoverResult{}, fmt.Errorf("mark recovered source in continuity catalog: %v: %w", err, ErrInternal)
	}

	a.logger().Printf("recover[%s -> %s]: agent=%s codexHome=%t messages=%t", oldID, newSess.ID, old.AgentType, codexHomeCopied, messagesCopied)

	res = RecoverResult{
		OldSessionID:    oldID,
		NewSessionID:    newSess.ID,
		AgentType:       string(old.AgentType),
		CommandSent:     cmd,
		CodexHomeCopied: codexHomeCopied,
		MessagesCopied:  messagesCopied,
	}
	if ledger, ok := a.LifecycleLedger.(continuity.ResultLedger); ok {
		if _, err := ledger.SetActorID(ctx, receipt.OperationID, res.NewSessionID); err != nil {
			return RecoverResult{}, fmt.Errorf("persist recovered session in receipt: %v: %w", err, ErrInternal)
		}
	}

	if in.IdempotencyKey != "" {
		if a.Idempotency != nil {
			a.Idempotency.Set("recover:"+oldID+":"+in.IdempotencyKey, intsessions.Response{
				ID:              res.NewSessionID,
				Backend:         backend.Persistent,
				SurvivesRestart: true,
				Recovered:       true,
			})
		}
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
	if a.Store != nil {
		if err := a.Store.UpdatePolicy(ctx, sess.ID, pol); err != nil {
			return PolicyView{}, fmt.Errorf("persist policy for session %q: %v: %w", sanitizeID(id), err, ErrInternal)
		}
	}
	sess.SetPolicy(pol)
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
	// Remote errors must be redacted before they cross the browser boundary.
	// nodereach.Error deliberately retains node/transport diagnostics for logs,
	// but its Error string is not a safe operator-facing response.
	var nodeErr *nodereach.Error
	if errors.As(err, &nodeErr) {
		switch nodeErr.Kind {
		case nodereach.ErrMissingScope:
			requiredScope := strings.TrimSpace(nodeErr.Scope)
			if requiredScope == "" {
				requiredScope, _ = scopecatalog.TransportScope("interactive-session:write")
			}
			return fmt.Errorf("%w: remote node lacks required scope %q; manage the machine permissions", ErrTargetUnavailable, requiredScope)
		case nodereach.ErrNodeNotFound:
			return fmt.Errorf("%w: remote node was not found; refresh the machine catalog", ErrTargetNotFound)
		case nodereach.ErrNodeUnavailable, nodereach.ErrBridgeUnavailable:
			return fmt.Errorf("%w: remote node is offline or Bridge is unavailable; reconnect the machine and refresh", ErrTargetUnavailable)
		case nodereach.ErrMissingReauth:
			return fmt.Errorf("%w: remote machine authorization has expired; manage the machine permissions", ErrTargetUnavailable)
		case nodereach.ErrHandshakeRejected:
			return fmt.Errorf("%w: the remote node rejected the session handshake; refresh the machine and try again", ErrTargetUnavailable)
		case nodereach.ErrTransport, nodereach.ErrStreaming:
			return fmt.Errorf("%w: the remote session transport is unavailable; reconnect the machine and try again", ErrTargetUnavailable)
		case nodereach.ErrInvalidRequest:
			return fmt.Errorf("%w: the remote session request was rejected; check the machine configuration", ErrInvalidArgument)
		}
	}
	switch {
	case errors.Is(err, ErrTargetNotFound):
		return fmt.Errorf("%w: remote node was not found; refresh the machine catalog", ErrTargetNotFound)
	case errors.Is(err, ErrTargetUnavailable):
		// Capability preflight errors are already operator-safe and carry the
		// capability id plus its recovery action. Preserve that detail so a
		// refused launch explains what the operator can do next. Transport
		// failures keep the deliberately generic message below.
		if strings.Contains(err.Error(), `capability "`) {
			return fmt.Errorf("%w: %s", ErrTargetUnavailable, strings.TrimPrefix(err.Error(), ErrTargetUnavailable.Error()+": "))
		}
		return fmt.Errorf("%w: the remote target is unavailable; refresh the machine catalog and try again", ErrTargetUnavailable)
	case errors.Is(err, ErrRemoteUnavailable):
		return fmt.Errorf("%w: remote session service is unavailable; reconnect the machine and try again", ErrRemoteUnavailable)
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
