package hub

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/operatordecision"
)

// Decision subjects: one open decision per situation.
//
// An ask that carries a SubjectKey is about an ongoing situation (a blocked
// backup, a mode left on). The hub keeps at most one open ask per
// (source, source_event_type, subject_key), enforced by a partial unique index,
// so a producer that re-sends its decision updates the one the operator already
// sees instead of stacking new ones. The rules are documented in
// docs/concepts/INTEGRATIONS.md ("Decision subjects").

// Outcomes of AskDecision for an ask with a subject key.
const (
	// AskCreated: no open decision existed; a new one was created and delivered.
	AskCreated = "created"
	// AskReasserted: the open decision already said this; nothing was delivered
	// and the hub recorded that the producer still needs it.
	AskReasserted = "reasserted"
	// AskUpdated: the open decision kept its answer menu and took the request's
	// text, correlation and deadline as a new revision. No new push.
	AskUpdated = "updated"
	// AskReplaced: the answer menu changed, so the open decision closed as
	// withdrawn and a new one was created and delivered.
	AskReplaced = "replaced"
	// AskSuppressed: the operator dismissed a decision with these exact terms.
	AskSuppressed = "suppressed"
	// AskReturned: the operator answered (or the default applied) moments ago,
	// so the producer gets that decision back instead of a duplicate.
	AskReturned = "returned"
	// AskStale: the request was asserted before the subject's last withdrawal.
	AskStale = "stale"
)

const (
	// AskReplacedReason is shown on a decision a newer version replaced.
	AskReplacedReason = "Replaced by a newer version of this decision"
	// answerGrace is how long an identical request returns the answered or
	// defaulted decision, so a producer that has not yet processed the answer
	// does not open a second one.
	answerGrace = 15 * time.Minute
	// neverDeliveredExpiry is how long after its deadline an undelivered
	// default-bearing ask stays open before it expires.
	neverDeliveredExpiry = 7 * 24 * time.Hour
	// subjectTimeLayout is fixed width, so these timestamps order as strings.
	subjectTimeLayout = "2006-01-02T15:04:05.000000000Z"
	// maxRetainedRevisions bounds the revision history kept per ask.
	maxRetainedRevisions = 50
	// maxRetainedEventIDs bounds the earlier event ids kept per ask.
	maxRetainedEventIDs = 200
	maxSubjectAttempts  = 4
)

// ErrAskStale reports a request asserted before its subject's last withdrawal.
var ErrAskStale = errors.New("decision request was asserted before the subject's last withdrawal")

// AskResult is what AskDecision did with a request.
type AskResult struct {
	Ask          AskRecord
	Notification Notification
	// Outcome is one of the Ask* outcomes above for an ask with a subject key,
	// and empty for an ask without one.
	Outcome string
}

// AskRevision is one earlier version of an updated ask.
type AskRevision struct {
	Revision             int
	EventID              string
	Title                string
	Question             string
	RecommendationReason string
	ContextURL           string
	BigPicture           string
	TechnicalDetails     string
	Options              []AskOption
	Correlation          map[string]string
	Deadline             string
	RevisedAt            string
}

func normalizeSubject(spec *AskSpec) error {
	spec.SubjectKey = strings.TrimSpace(spec.SubjectKey)
	if spec.SubjectKey != "" && !operatordecision.SubjectKeyRule.MatchString(spec.SubjectKey) {
		return fmt.Errorf("%w: subject_key must be 1-128 letters, digits, '.', '_', ':', '/', '@' or '-', starting with a letter or digit", ErrInvalidArgument)
	}
	if spec.ReassertWithin != 0 {
		if spec.SubjectKey == "" {
			return fmt.Errorf("%w: reassert_within_seconds needs a subject_key", ErrInvalidArgument)
		}
		if spec.ReassertWithin < operatordecision.MinReassertWithin || spec.ReassertWithin > operatordecision.MaxReassertWithin {
			return fmt.Errorf("%w: reassert_within_seconds must be between %d and %d", ErrInvalidArgument, int(operatordecision.MinReassertWithin/time.Second), int(operatordecision.MaxReassertWithin/time.Second))
		}
	}
	return nil
}

func subjectTime(t time.Time) string { return t.UTC().Format(subjectTimeLayout) }

// assertedTime is the producer's assertion time, never later than now and now
// when the producer gave none.
func assertedTime(asserted, now time.Time) time.Time {
	if asserted.IsZero() || asserted.After(now) {
		return now
	}
	return asserted.UTC()
}

type askRowEncoding struct{ allowed, options, correlation string }

// encodeAskRow renders the JSON columns of an ask row.
func encodeAskRow(spec AskSpec) (askRowEncoding, error) {
	allowed := make([]string, 0, len(spec.Options))
	for _, option := range spec.Options {
		allowed = append(allowed, option.Key)
	}
	allowedJSON, _ := json.Marshal(allowed)
	optionsJSON, _ := json.Marshal(spec.Options)
	correlationJSON, err := encodeCorrelation(spec)
	if err != nil {
		return askRowEncoding{}, err
	}
	return askRowEncoding{allowed: string(allowedJSON), options: string(optionsJSON), correlation: correlationJSON}, nil
}

func encodeCorrelation(spec AskSpec) (string, error) {
	correlation := make(map[string]string, len(spec.Correlation)+1)
	for key, value := range spec.Correlation {
		correlation[key] = value
	}
	if _, exists := correlation[providerDecisionCorrelationKey]; exists {
		return "", fmt.Errorf("%w: correlation uses a reserved provider decision key", ErrInvalidArgument)
	}
	if spec.ProviderDecisionContext != nil {
		encoded, err := encodeProviderDecisionContext(spec.ProviderDecisionContext)
		if err != nil {
			return "", err
		}
		correlation[providerDecisionCorrelationKey] = encoded
	}
	if len(correlation) == 0 {
		return "", nil
	}
	encoded, _ := json.Marshal(correlation)
	if len(encoded) > maxAskCorrelationLen {
		return "", fmt.Errorf("%w: correlation exceeds %d bytes", ErrInvalidArgument, maxAskCorrelationLen)
	}
	return string(encoded), nil
}

// insertAskTx writes one ask row. asserted is "" for an ask without a subject.
func insertAskTx(ctx context.Context, tx dbTx, id, notificationID string, spec AskSpec, enc askRowEncoding, now, asserted string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO asks (id, notification_id, question, allowed_answers, deadline, state, reason, created_at, updated_at, options_json, recommended, recommendation_reason, default_answer, reversible, urgency, context_url, big_picture, technical_details, source, source_event_type, requester_ref, correlation_json, subject_key, last_asserted_at, reassert_within_seconds, current_event_id) VALUES (?, ?, ?, ?, ?, 'pending', '', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, notificationID, spec.Question, enc.allowed, spec.Deadline.UTC().Format(time.RFC3339Nano), now, now, enc.options, spec.Recommended, strings.TrimSpace(spec.RecommendationReason), spec.DefaultAnswer, boolInt(spec.Reversible), spec.Urgency, strings.TrimSpace(spec.ContextURL), spec.BigPicture, spec.TechnicalDetails, spec.Source, strings.TrimSpace(spec.SourceEventType), spec.IdempotencyKey, enc.correlation,
		spec.SubjectKey, asserted, int(spec.ReassertWithin/time.Second), spec.IdempotencyKey)
	return err
}

// requestLifecycleGateTx serializes an event-sourced request with a withdrawal
// that named its event id. Agent Manager's withdrawable requests reserve their
// ref; every other event-sourced request only checks it. Direct asks (no
// source event type) have no lifecycle ledger.
func (s *Service) requestLifecycleGateTx(ctx context.Context, tx dbTx, spec AskSpec) error {
	if spec.SourceEventType == "" || spec.IdempotencyKey == "" {
		return nil
	}
	var withdrawn bool
	var err error
	if spec.Source == "agent-manager" && withdrawableDecision(spec.SourceEventType) {
		withdrawn, err = reserveDecisionRequestTx(ctx, tx, spec.Source, spec.IdempotencyKey, spec.SourceEventType, s.clock.Now().UTC().Format(time.RFC3339Nano))
	} else {
		withdrawn, err = checkDecisionRequestLifecycleTx(ctx, tx, spec.Source, spec.IdempotencyKey, spec.SourceEventType)
	}
	if err != nil {
		return err
	}
	if withdrawn {
		return ErrAskWithdrawn
	}
	return nil
}

// beginWriteTx opens a transaction that holds the database write lock from its
// first statement. These transactions read the subject's state and then write
// it; under SQLite's deferred locking two of them can both read and then fail
// to upgrade (SQLITE_BUSY_SNAPSHOT) instead of waiting their turn. A statement
// that writes no rows still takes the lock, so the later reads see the state
// the write will be based on.
func (s *Service) beginWriteTx(ctx context.Context) (*sql.Tx, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE decision_subject_withdrawals SET updated_at = updated_at WHERE 0`); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	return tx, nil
}

func (s *Service) askSubject(ctx context.Context, spec AskSpec, pushSource string) (AskResult, error) {
	var lastErr error
	for attempt := 0; attempt < maxSubjectAttempts; attempt++ {
		result, retry, err := s.askSubjectOnce(ctx, spec, pushSource)
		if !retry {
			return result, err
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return AskResult{}, ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 10 * time.Millisecond):
		}
	}
	return AskResult{}, lastErr
}

// retryableSubjectError reports a lost race: a concurrent request for the same
// subject (or event) won the unique index, or the database was busy. The next
// attempt re-reads the winner and applies the rules to it.
func retryableSubjectError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "unique constraint") || strings.Contains(text, "database is locked") || strings.Contains(text, "sqlite_busy")
}

const askSubjectWhere = ` WHERE a.source = ? AND a.source_event_type = ? AND a.subject_key = ?`

func openSubjectAskTx(ctx context.Context, tx dbTx, source, eventType, subject string) (AskRecord, error) {
	return scanAsk(tx.QueryRowContext(ctx, askSelect+askSubjectWhere+` AND a.state IN ('pending','escalated') LIMIT 1`, source, eventType, subject))
}

func latestSubjectAskTx(ctx context.Context, tx dbTx, source, eventType, subject string) (AskRecord, error) {
	return scanAsk(tx.QueryRowContext(ctx, askSelect+askSubjectWhere+` ORDER BY a.created_at DESC, a.id DESC LIMIT 1`, source, eventType, subject))
}

// askByEventIDTx finds the ask an event id belongs to: its current requester_ref
// (the latest revision's event) or an id it carried before.
func askByEventIDTx(ctx context.Context, tx dbTx, source, eventType, eventID string) (AskRecord, error) {
	if eventID == "" {
		return AskRecord{}, ErrNotFound
	}
	return scanAsk(tx.QueryRowContext(ctx, askSelect+` WHERE a.source = ? AND a.source_event_type = ? AND (a.requester_ref = ? OR EXISTS (SELECT 1 FROM ask_event_ids r WHERE r.ask_id = a.id AND r.event_id = ?)) ORDER BY a.created_at DESC, a.id DESC LIMIT 1`, source, eventType, eventID, eventID))
}

func subjectWithdrawalAssertedTx(ctx context.Context, tx dbTx, source, eventType, subject string) (time.Time, bool, error) {
	var asserted string
	err := tx.QueryRowContext(ctx, `SELECT asserted_at FROM decision_subject_withdrawals WHERE source = ? AND source_event_type = ? AND subject_key = ?`, source, eventType, subject).Scan(&asserted)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}
	parsed, parseErr := time.Parse(time.RFC3339Nano, asserted)
	if parseErr != nil {
		return time.Time{}, false, nil
	}
	return parsed, true, nil
}

// sameSubjectMenu reports whether a request offers the operator the same
// choice: recipient, answer keys in order, recommendation, default,
// reversibility, urgency, sensitivity and provider binding. Option labels are
// text, not menu.
func sameSubjectMenu(existing AskRecord, n Notification, spec AskSpec) bool {
	if existing.Recipient != spec.Recipient || existing.Recommended != spec.Recommended ||
		existing.DefaultAnswer != spec.DefaultAnswer || existing.Reversible != spec.Reversible ||
		existing.Urgency != spec.Urgency || n.Urgency != spec.Urgency || n.SensitivityLabel != spec.SensitivityLabel ||
		len(existing.Options) != len(spec.Options) ||
		!sameProviderDecisionContext(existing.ProviderDecisionContext, spec.ProviderDecisionContext) {
		return false
	}
	for index := range existing.Options {
		if existing.Options[index].Key != spec.Options[index].Key {
			return false
		}
	}
	return true
}

// sameSubjectContent reports whether a request says the same thing: title,
// question, reasons, links, explanations, option labels and correlation.
func sameSubjectContent(existing AskRecord, n Notification, spec AskSpec) bool {
	if len(existing.Options) != len(spec.Options) {
		return false
	}
	for index := range existing.Options {
		if existing.Options[index].Label != spec.Options[index].Label {
			return false
		}
	}
	return n.Title == spec.Title && existing.Question == spec.Question &&
		existing.RecommendationReason == strings.TrimSpace(spec.RecommendationReason) &&
		existing.ContextURL == strings.TrimSpace(spec.ContextURL) &&
		existing.BigPicture == spec.BigPicture && existing.TechnicalDetails == spec.TechnicalDetails &&
		sameCorrelation(existing.Correlation, spec.Correlation)
}

func (s *Service) askSubjectOnce(ctx context.Context, spec AskSpec, pushSource string) (AskResult, bool, error) {
	now := s.clock.Now().UTC()
	asserted := assertedTime(spec.AssertedAt, now)
	tx, err := s.beginWriteTx(ctx)
	if err != nil {
		return AskResult{}, retryableSubjectError(err), err
	}
	defer tx.Rollback()
	fail := func(err error) (AskResult, bool, error) { return AskResult{}, retryableSubjectError(err), err }

	if err := s.requestLifecycleGateTx(ctx, tx, spec); err != nil {
		return AskResult{}, false, err
	}
	// A redelivered event (or one of an earlier revision) changes nothing: the
	// ask already moved past it.
	replay, err := replayedSubjectRequestTx(ctx, tx, spec)
	if err != nil || replay != nil {
		if err != nil {
			return fail(err)
		}
		return *replay, false, nil
	}
	withdrawnAt, hasWithdrawal, err := subjectWithdrawalAssertedTx(ctx, tx, spec.Source, spec.SourceEventType, spec.SubjectKey)
	if err != nil {
		return fail(err)
	}
	if hasWithdrawal && withdrawnAt.After(asserted) {
		return AskResult{Outcome: AskStale}, false, nil
	}
	open, err := openSubjectAskTx(ctx, tx, spec.Source, spec.SourceEventType, spec.SubjectKey)
	if err == nil {
		return s.applyToOpenSubjectAsk(ctx, tx, open, spec, pushSource, now, asserted)
	}
	if !errors.Is(err, ErrNotFound) {
		return fail(err)
	}
	// No open decision. The subject's latest decision decides whether an
	// identical request is suppressed, answered a moment ago, or new.
	settled, err := settledSubjectResultTx(ctx, tx, spec, now)
	if err != nil || settled != nil {
		if err != nil {
			return fail(err)
		}
		return *settled, false, nil
	}
	return s.createSubjectAsk(ctx, tx, nil, spec, pushSource, now, asserted, AskCreated)
}

// replayedSubjectRequestTx recognizes a request whose event id already opened
// or revised an ask. It returns that ask unchanged, ErrAskWithdrawn when the
// ask was closed, or nil when the event id is new. Reusing an id for different
// terms, or for an ask of another subject or type, is a conflict.
func replayedSubjectRequestTx(ctx context.Context, tx dbTx, spec AskSpec) (*AskResult, error) {
	replay, err := askByEventIDTx(ctx, tx, spec.Source, spec.SourceEventType, spec.IdempotencyKey)
	if errors.Is(err, ErrNotFound) {
		if _, err := askByRequesterTx(ctx, tx, spec.Source, spec.IdempotencyKey); err == nil {
			return nil, ErrAskIdentityConflict
		} else if !errors.Is(err, ErrNotFound) {
			return nil, err
		}
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if replay.SubjectKey != spec.SubjectKey {
		return nil, ErrAskIdentityConflict
	}
	if replay.State == AskWithdrawn {
		return nil, ErrAskWithdrawn
	}
	n, err := notificationByIDTx(ctx, tx, replay.NotificationID)
	if err != nil {
		return nil, err
	}
	// The same id for different terms is a client bug, not a redelivery. An
	// earlier revision's id legitimately differs from the current terms.
	if spec.IdempotencyKey == replay.RequesterRef && !(sameSubjectMenu(replay, n, spec) && sameSubjectContent(replay, n, spec)) {
		return nil, ErrAskIdentityConflict
	}
	return &AskResult{Ask: replay, Notification: n, Outcome: AskReasserted}, nil
}

// settledSubjectResultTx answers a request that finds no open decision but
// repeats the subject's latest closed one: dismissed terms stay dismissed, and
// a decision answered or defaulted under answerGrace ago is handed back. It
// returns nil when the request deserves a new decision.
func settledSubjectResultTx(ctx context.Context, tx dbTx, spec AskSpec, now time.Time) (*AskResult, error) {
	last, err := latestSubjectAskTx(ctx, tx, spec.Source, spec.SourceEventType, spec.SubjectKey)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	n, err := notificationByIDTx(ctx, tx, last.NotificationID)
	if err != nil {
		return nil, err
	}
	if !sameSubjectMenu(last, n, spec) || !sameSubjectContent(last, n, spec) {
		return nil, nil
	}
	switch last.State {
	case AskDismissed:
		return &AskResult{Ask: last, Notification: n, Outcome: AskSuppressed}, nil
	case AskAnswered, AskDefaulted:
		if resolved, parseErr := time.Parse(time.RFC3339Nano, last.ResolvedAt); parseErr == nil && now.Sub(resolved) < answerGrace {
			return &AskResult{Ask: last, Notification: n, Outcome: AskReturned}, nil
		}
	}
	return nil, nil
}

// createSubjectAsk inserts the notification and ask in tx, first closing the
// ask it replaces (so the unique open-subject index admits the new one).
func (s *Service) createSubjectAsk(ctx context.Context, tx *sql.Tx, replaced *AskRecord, spec AskSpec, pushSource string, now, asserted time.Time, outcome string) (AskResult, bool, error) {
	n, created, err := s.persistNotificationUsing(ctx, tx, SendInput{RequestedBy: spec.Recipient, Title: spec.Title, Body: spec.Body, Urgency: spec.Urgency, SensitivityLabel: spec.SensitivityLabel, IdempotencyKey: spec.IdempotencyKey})
	if err != nil {
		return AskResult{}, retryableSubjectError(err), err
	}
	if !created {
		// The notification identity is taken by something that is not this
		// subject's ask.
		return AskResult{}, false, ErrAskIdentityConflict
	}
	enc, err := encodeAskRow(spec)
	if err != nil {
		return AskResult{}, false, err
	}
	id := uuid.NewString()
	nowText := now.Format(time.RFC3339Nano)
	if replaced != nil {
		if _, err := tx.ExecContext(ctx, `UPDATE asks SET state='withdrawn', reason=?, resolved_at=?, resolution_published_at='', superseded_by_ask_id=?, updated_at=? WHERE id=? AND state IN ('pending','escalated')`, AskReplacedReason, nowText, id, nowText, replaced.ID); err != nil {
			return AskResult{}, retryableSubjectError(err), err
		}
	}
	if err := insertAskTx(ctx, tx, id, n.ID, spec, enc, nowText, subjectTime(asserted)); err != nil {
		return AskResult{}, retryableSubjectError(err), fmt.Errorf("persist ask: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return AskResult{}, retryableSubjectError(err), err
	}
	s.observeCatalog(ctx, n, nowText)
	if replaced != nil {
		s.kickResolutions(ctx)
	}
	if pushSource == "" {
		s.dispatch(n.ID)
	} else {
		s.dispatchDirect(n.ID, pushSource)
	}
	record, err := s.GetAsk(ctx, id)
	return AskResult{Ask: record, Notification: n, Outcome: outcome}, false, err
}

func (s *Service) applyToOpenSubjectAsk(ctx context.Context, tx *sql.Tx, open AskRecord, spec AskSpec, pushSource string, now, asserted time.Time) (AskResult, bool, error) {
	n, err := notificationByIDTx(ctx, tx, open.NotificationID)
	if err != nil {
		return AskResult{}, retryableSubjectError(err), err
	}
	// A re-send the producer made before its latest confirmation arrived late;
	// it must not roll the decision back.
	if lastAsserted, parseErr := time.Parse(time.RFC3339Nano, open.LastAssertedAt); parseErr == nil && asserted.Before(lastAsserted) {
		return AskResult{Ask: open, Notification: n, Outcome: AskReasserted}, false, nil
	}
	if !sameSubjectMenu(open, n, spec) {
		return s.createSubjectAsk(ctx, tx, &open, spec, pushSource, now, asserted, AskReplaced)
	}
	// An explicit deadline from the producer applies; without one the ask keeps
	// its own, and an escalated ask owns its deadline outright.
	deadline := open.Deadline
	if spec.DeadlineSpecified && open.State == AskPending {
		deadline = spec.Deadline.UTC().Format(time.RFC3339Nano)
	}
	outcome := AskReasserted
	if sameSubjectContent(open, n, spec) {
		err = reassertSubjectAskTx(ctx, tx, open, spec, deadline, now, asserted)
	} else {
		outcome = AskUpdated
		err = updateSubjectAskTx(ctx, tx, open, n, spec, deadline, now, asserted)
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		return AskResult{}, retryableSubjectError(err), err
	}
	record, err := s.GetAsk(ctx, open.ID)
	if err != nil {
		return AskResult{}, false, err
	}
	// The updated text is on the notification, which the result reports.
	current, _, err := s.Get(ctx, n.ID)
	return AskResult{Ask: record, Notification: current, Outcome: outcome}, false, err
}

// reassertSubjectAskTx records that the producer still needs an unchanged
// decision (and any explicit new deadline). Nothing is delivered.
func reassertSubjectAskTx(ctx context.Context, tx dbTx, open AskRecord, spec AskSpec, deadline string, now, asserted time.Time) error {
	if err := retireEventIDTx(ctx, tx, open, spec.IdempotencyKey, now.Format(time.RFC3339Nano)); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `UPDATE asks SET last_asserted_at=?, reassert_within_seconds=?, requester_ref=?, current_event_id=?, deadline=? WHERE id=? AND state IN ('pending','escalated')`,
		subjectTime(asserted), int(spec.ReassertWithin/time.Second), spec.IdempotencyKey, spec.IdempotencyKey, deadline, open.ID)
	return err
}

// retireEventIDTx moves the ask's requester_ref to the latest revision's event
// id: the previous id stays matchable (redeliveries, withdrawals) in
// ask_event_ids, bounded to the newest maxRetainedEventIDs.
func retireEventIDTx(ctx context.Context, tx dbTx, open AskRecord, latest, nowText string) error {
	if latest == open.RequesterRef {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO ask_event_ids (ask_id, event_id, retired_at) VALUES (?, ?, ?)`, open.ID, open.RequesterRef, nowText); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `DELETE FROM ask_event_ids WHERE ask_id = ? AND rowid NOT IN (SELECT rowid FROM ask_event_ids WHERE ask_id = ? ORDER BY rowid DESC LIMIT ?)`, open.ID, open.ID, maxRetainedEventIDs)
	return err
}

// updateSubjectAskTx makes the request the ask's new revision: the previous
// revision is kept, the ask takes the new text, correlation and deadline, and
// the notification the inbox shows takes the new title and body. The
// notification is never re-dispatched, so there is no new push.
func updateSubjectAskTx(ctx context.Context, tx dbTx, open AskRecord, n Notification, spec AskSpec, deadline string, now, asserted time.Time) error {
	nowText := now.Format(time.RFC3339Nano)
	if err := insertRevisionTx(ctx, tx, open, n, nowText); err != nil {
		return err
	}
	if err := retireEventIDTx(ctx, tx, open, spec.IdempotencyKey, nowText); err != nil {
		return err
	}
	enc, err := encodeAskRow(spec)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE asks SET question=?, recommendation_reason=?, context_url=?, big_picture=?, technical_details=?, options_json=?, correlation_json=?, deadline=?, revision_count=revision_count+1, requester_ref=?, current_event_id=?, last_asserted_at=?, reassert_within_seconds=?, updated_at=? WHERE id=? AND state IN ('pending','escalated')`,
		spec.Question, strings.TrimSpace(spec.RecommendationReason), strings.TrimSpace(spec.ContextURL), spec.BigPicture, spec.TechnicalDetails, enc.options, enc.correlation, deadline, spec.IdempotencyKey, spec.IdempotencyKey, subjectTime(asserted), int(spec.ReassertWithin/time.Second), nowText, open.ID); err != nil {
		return err
	}
	body := spec.Body
	if body == renderAskBody(spec) {
		// A generated body names the deadline, which is the ask's, not the
		// request's, when the request gave none.
		effective := spec
		if parsed, parseErr := time.Parse(time.RFC3339Nano, deadline); parseErr == nil {
			effective.Deadline = parsed
		}
		body = renderAskBody(effective)
	}
	_, err = tx.ExecContext(ctx, `UPDATE notifications SET title=?, body=?, updated_at=? WHERE id=?`, spec.Title, body, nowText, n.ID)
	return err
}

// insertRevisionTx keeps the content an in-place update is about to replace.
func insertRevisionTx(ctx context.Context, tx dbTx, open AskRecord, n Notification, now string) error {
	eventID := open.CurrentEventID
	if eventID == "" {
		eventID = open.RequesterRef
	}
	optionsJSON, _ := json.Marshal(open.Options)
	correlationJSON := ""
	if len(open.Correlation) > 0 {
		encoded, _ := json.Marshal(open.Correlation)
		correlationJSON = string(encoded)
	}
	revision := open.RevisionCount + 1
	if _, err := tx.ExecContext(ctx, `INSERT INTO ask_revisions (ask_id, revision, event_id, title, question, recommendation_reason, context_url, big_picture, technical_details, options_json, correlation_json, deadline, revised_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		open.ID, revision, eventID, n.Title, open.Question, open.RecommendationReason, open.ContextURL, open.BigPicture, open.TechnicalDetails, string(optionsJSON), correlationJSON, open.Deadline, now); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `DELETE FROM ask_revisions WHERE ask_id = ? AND revision <= ?`, open.ID, revision-maxRetainedRevisions)
	return err
}

// ListAskRevisions returns the earlier versions of an updated ask, newest first.
func (s *Service) ListAskRevisions(ctx context.Context, askID string) ([]AskRevision, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT revision, event_id, title, question, recommendation_reason, context_url, big_picture, technical_details, options_json, correlation_json, deadline, revised_at FROM ask_revisions WHERE ask_id = ? ORDER BY revision DESC`, strings.TrimSpace(askID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AskRevision
	for rows.Next() {
		var item AskRevision
		var optionsJSON, correlationJSON string
		if err := rows.Scan(&item.Revision, &item.EventID, &item.Title, &item.Question, &item.RecommendationReason, &item.ContextURL, &item.BigPicture, &item.TechnicalDetails, &optionsJSON, &correlationJSON, &item.Deadline, &item.RevisedAt); err != nil {
			return nil, err
		}
		if strings.TrimSpace(optionsJSON) != "" {
			_ = json.Unmarshal([]byte(optionsJSON), &item.Options)
		}
		if strings.TrimSpace(correlationJSON) != "" {
			_ = json.Unmarshal([]byte(correlationJSON), &item.Correlation)
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// SubjectWithdrawal closes the open decision of a producer. It names the
// decision by SubjectKey or RequesterRef (an event id of any revision).
type SubjectWithdrawal struct {
	Source          string
	SourceEventType string
	SubjectKey      string
	RequesterRef    string
	// Reason is shown to the operator on the closed decision.
	Reason string
	// AssertedAt is when the producer saw the decision was no longer needed;
	// zero means now.
	AssertedAt time.Time
}

const maxWithdrawalReasonBytes = operatordecision.MaxWithdrawalReason

// normalized returns the withdrawal with trimmed fields, or an
// ErrInvalidArgument naming what is wrong with it.
func (in SubjectWithdrawal) normalized() (SubjectWithdrawal, error) {
	in.Source = strings.TrimSpace(in.Source)
	in.SourceEventType = strings.TrimSpace(in.SourceEventType)
	in.SubjectKey = strings.TrimSpace(in.SubjectKey)
	in.RequesterRef = strings.TrimSpace(in.RequesterRef)
	in.Reason = strings.TrimSpace(in.Reason)
	switch {
	case in.Source == "" || !strings.HasSuffix(in.SourceEventType, operatordecision.DecisionRequestedSuffix):
		return in, fmt.Errorf("%w: a withdrawal needs its source and the request event type it closes", ErrInvalidArgument)
	case in.SubjectKey == "" && in.RequesterRef == "":
		return in, fmt.Errorf("%w: a withdrawal needs a subject_key or a requester_ref", ErrInvalidArgument)
	case in.SubjectKey != "" && !operatordecision.SubjectKeyRule.MatchString(in.SubjectKey):
		return in, fmt.Errorf("%w: subject_key is invalid", ErrInvalidArgument)
	case len(in.RequesterRef) > 128:
		return in, fmt.Errorf("%w: requester_ref exceeds 128 bytes", ErrInvalidArgument)
	case in.Reason == "" || !utf8.ValidString(in.Reason) || len(in.Reason) > maxWithdrawalReasonBytes:
		return in, fmt.Errorf("%w: a withdrawal reason of 1-%d bytes is required", ErrInvalidArgument, maxWithdrawalReasonBytes)
	}
	return in, nil
}

// WithdrawDecisionSubject closes the producer's own open decision with the
// producer's reason. It is not an answer: no answers row is written, and a
// decision that is not open keeps its outcome. A withdrawal that names a
// subject is remembered, so a request asserted before it is ignored; one that
// names an event id no ask has seen yet is remembered by that id.
//
// It reports whether an open decision was closed.
func (s *Service) WithdrawDecisionSubject(ctx context.Context, in SubjectWithdrawal) (bool, error) {
	if s == nil || s.db == nil {
		return false, errors.New("ask service is unavailable")
	}
	in, err := in.normalized()
	if err != nil {
		return false, err
	}
	now := s.clock.Now().UTC()
	in.AssertedAt = assertedTime(in.AssertedAt, now)
	nowText := now.Format(time.RFC3339Nano)

	var lastErr error
	for attempt := 0; attempt < maxSubjectAttempts; attempt++ {
		changed, err := s.withdrawSubjectOnce(ctx, in, nowText)
		if err == nil || !retryableSubjectError(err) {
			return changed, err
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 10 * time.Millisecond):
		}
	}
	return false, lastErr
}

// withdrawalTargetTx finds the ask a withdrawal names: the open ask of the
// subject, else the ask any of whose revisions an event id opened. A replaced
// decision's successor carries the situation forward, so a producer closing by
// the id it first used closes what replaced it.
func withdrawalTargetTx(ctx context.Context, tx dbTx, in SubjectWithdrawal) (AskRecord, bool, error) {
	if in.SubjectKey != "" {
		ask, err := openSubjectAskTx(ctx, tx, in.Source, in.SourceEventType, in.SubjectKey)
		if err == nil {
			return ask, true, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return AskRecord{}, false, err
		}
	}
	if in.RequesterRef == "" {
		return AskRecord{}, false, nil
	}
	target, err := askByEventIDTx(ctx, tx, in.Source, in.SourceEventType, in.RequesterRef)
	if errors.Is(err, ErrNotFound) {
		return AskRecord{}, false, nil
	}
	if err != nil {
		return AskRecord{}, false, err
	}
	for hops := 0; target.SupersededByAskID != "" && hops < 10; hops++ {
		next, err := scanAsk(tx.QueryRowContext(ctx, askSelect+` WHERE a.id = ?`, target.SupersededByAskID))
		if errors.Is(err, ErrNotFound) {
			break
		}
		if err != nil {
			return AskRecord{}, false, err
		}
		target = next
	}
	return target, true, nil
}

// rememberSubjectWithdrawalTx keeps the latest withdrawal a producer asserted
// for a subject; a request asserted earlier is stale.
func rememberSubjectWithdrawalTx(ctx context.Context, tx dbTx, in SubjectWithdrawal, subject, nowText string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO decision_subject_withdrawals (source, source_event_type, subject_key, asserted_at, reason, updated_at) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(source, source_event_type, subject_key) DO UPDATE SET
		  reason = CASE WHEN excluded.asserted_at > decision_subject_withdrawals.asserted_at THEN excluded.reason ELSE decision_subject_withdrawals.reason END,
		  asserted_at = CASE WHEN excluded.asserted_at > decision_subject_withdrawals.asserted_at THEN excluded.asserted_at ELSE decision_subject_withdrawals.asserted_at END,
		  updated_at = excluded.updated_at`,
		in.Source, in.SourceEventType, subject, subjectTime(in.AssertedAt), in.Reason, nowText)
	return err
}

// rememberWithdrawnRequestTx records that the request an event id names was
// withdrawn before it arrived, so its delivery creates nothing.
func rememberWithdrawnRequestTx(ctx context.Context, tx dbTx, in SubjectWithdrawal, nowText string) error {
	if _, err := tx.ExecContext(ctx, `INSERT INTO decision_request_lifecycle (source, requester_ref, source_event_type, state, reason, created_at, updated_at) VALUES (?, ?, ?, 'withdrawn', ?, ?, ?) ON CONFLICT(source, requester_ref) DO NOTHING`, in.Source, in.RequesterRef, in.SourceEventType, in.Reason, nowText, nowText); err != nil {
		return err
	}
	var lifecycleType string
	if err := tx.QueryRowContext(ctx, `SELECT source_event_type FROM decision_request_lifecycle WHERE source=? AND requester_ref=?`, in.Source, in.RequesterRef).Scan(&lifecycleType); err != nil {
		return err
	}
	if lifecycleType != in.SourceEventType {
		return ErrAskIdentityConflict
	}
	return nil
}

func (s *Service) withdrawSubjectOnce(ctx context.Context, in SubjectWithdrawal, nowText string) (bool, error) {
	tx, err := s.beginWriteTx(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	target, found, err := withdrawalTargetTx(ctx, tx, in)
	if err != nil {
		return false, err
	}
	// A confirmation the producer made after this withdrawal was asserted (the
	// events arrived out of order) keeps the decision open.
	if found && target.Open() {
		if lastAsserted, parseErr := time.Parse(time.RFC3339Nano, target.LastAssertedAt); parseErr == nil && in.AssertedAt.Before(lastAsserted) {
			return false, tx.Commit()
		}
	}
	subject := in.SubjectKey
	if subject == "" && found {
		subject = target.SubjectKey
	}
	if subject != "" {
		if err := rememberSubjectWithdrawalTx(ctx, tx, in, subject, nowText); err != nil {
			return false, err
		}
	}
	if !found && in.RequesterRef != "" {
		if err := rememberWithdrawnRequestTx(ctx, tx, in, nowText); err != nil {
			return false, err
		}
	}
	if !found || !target.Open() {
		// Nothing to close, or an answered, defaulted, expired, dismissed or
		// already withdrawn decision whose outcome stands.
		return false, tx.Commit()
	}
	result, err := tx.ExecContext(ctx, `UPDATE asks SET state='withdrawn', reason=?, resolved_at=?, resolution_published_at='', updated_at=? WHERE id=? AND state IN ('pending','escalated')`, in.Reason, nowText, nowText, target.ID)
	if err != nil {
		return false, err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	if changed == 1 {
		s.kickResolutions(ctx)
		return true, nil
	}
	return false, nil
}

// WithdrawDirectSubject closes the requester's own open direct ask for a
// subject key, the way WithdrawDirectAsk closes one by id. A subject whose
// latest ask is already withdrawn returns it; one with no ask is not found.
func (s *Service) WithdrawDirectSubject(ctx context.Context, requester, subject, reason string) (AskRecord, error) {
	requester = strings.TrimSpace(requester)
	subject = strings.TrimSpace(subject)
	reason = strings.TrimSpace(reason)
	if requester == "" || !operatordecision.SubjectKeyRule.MatchString(subject) {
		return AskRecord{}, fmt.Errorf("%w: subject_key is invalid", ErrInvalidArgument)
	}
	if reason == "" || len(reason) > 2_000 {
		return AskRecord{}, fmt.Errorf("%w: a withdrawal reason of 1-2000 bytes is required", ErrInvalidArgument)
	}
	var lastErr error
	for attempt := 0; attempt < maxSubjectAttempts; attempt++ {
		record, err := s.withdrawDirectSubjectOnce(ctx, requester, subject, reason)
		if err == nil || !retryableSubjectError(err) {
			return record, err
		}
		lastErr = err
	}
	return AskRecord{}, lastErr
}

func (s *Service) withdrawDirectSubjectOnce(ctx context.Context, requester, subject, reason string) (AskRecord, error) {
	now := s.clock.Now().UTC()
	nowText := now.Format(time.RFC3339Nano)
	tx, err := s.beginWriteTx(ctx)
	if err != nil {
		return AskRecord{}, err
	}
	defer tx.Rollback()
	open, err := openSubjectAskTx(ctx, tx, requester, "", subject)
	if errors.Is(err, ErrNotFound) {
		last, lastErr := latestSubjectAskTx(ctx, tx, requester, "", subject)
		if lastErr != nil {
			return AskRecord{}, lastErr
		}
		if last.State == AskWithdrawn {
			return last, nil
		}
		return AskRecord{}, fmt.Errorf("%w: ask is already %s", ErrAskResolved, last.State)
	}
	if err != nil {
		return AskRecord{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO decision_subject_withdrawals (source, source_event_type, subject_key, asserted_at, reason, updated_at) VALUES (?, '', ?, ?, ?, ?)
		ON CONFLICT(source, source_event_type, subject_key) DO UPDATE SET asserted_at = excluded.asserted_at, reason = excluded.reason, updated_at = excluded.updated_at`,
		requester, subject, subjectTime(now), reason, nowText); err != nil {
		return AskRecord{}, err
	}
	result, err := tx.ExecContext(ctx, `UPDATE asks SET state='withdrawn', reason=?, resolved_at=?, resolution_published_at='', updated_at=? WHERE id=? AND state IN ('pending','escalated')`, reason, nowText, nowText, open.ID)
	if err != nil {
		return AskRecord{}, err
	}
	if changed, _ := result.RowsAffected(); changed != 1 {
		return AskRecord{}, fmt.Errorf("%w: ask changed while withdrawing", ErrAskResolved)
	}
	if err := tx.Commit(); err != nil {
		return AskRecord{}, err
	}
	s.kickResolutions(ctx)
	return s.GetAsk(ctx, open.ID)
}

// closeLapsedSubjects closes every open ask whose producer promised to
// re-send it (reassert_within_seconds) and has not for that long.
func (s *Service) closeLapsedSubjects(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id, source, last_asserted_at, created_at, reassert_within_seconds FROM asks WHERE state IN ('pending','escalated') AND reassert_within_seconds > 0`)
	if err != nil {
		return err
	}
	type lapsed struct {
		id, source, assertedAt string
		since                  time.Time
	}
	now := s.clock.Now().UTC()
	var due []lapsed
	for rows.Next() {
		var id, source, assertedAt, createdAt string
		var window int
		if err := rows.Scan(&id, &source, &assertedAt, &createdAt, &window); err != nil {
			rows.Close()
			return err
		}
		since, parseErr := time.Parse(time.RFC3339Nano, assertedAt)
		if parseErr != nil {
			if since, parseErr = time.Parse(time.RFC3339Nano, createdAt); parseErr != nil {
				continue
			}
		}
		if now.Sub(since) >= time.Duration(window)*time.Second {
			due = append(due, lapsed{id: id, source: source, assertedAt: assertedAt, since: since})
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	nowText := now.Format(time.RFC3339Nano)
	for _, item := range due {
		reason := fmt.Sprintf("No longer confirmed by %s since %s", item.source, item.since.UTC().Format("Jan 2 15:04 MST"))
		// last_asserted_at is part of the guard: a re-send that lands while the
		// sweep runs keeps the decision open.
		result, err := s.db.ExecContext(ctx, `UPDATE asks SET state='withdrawn', reason=?, resolved_at=?, resolution_published_at='', updated_at=? WHERE id=? AND state IN ('pending','escalated') AND last_asserted_at=?`, reason, nowText, nowText, item.id, item.assertedAt)
		if err != nil {
			return err
		}
		if changed, _ := result.RowsAffected(); changed == 1 {
			s.kickResolutions(ctx)
		}
	}
	return nil
}
