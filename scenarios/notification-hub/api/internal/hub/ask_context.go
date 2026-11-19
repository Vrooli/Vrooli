package hub

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
)

// Context requests: "I need more context" keeps the ask open and sends the
// recipient's question to the requester. The requester answers by revising the
// ask in place with a written context_response (and any change the answer
// needs), which marks every open request addressed and stores the answer with it
// (docs/concepts/DECISION_PACKETS.md, "Context requests").

const (
	// AskContextRequestedEventType tells the requester the recipient asked for
	// more context on an open ask. It is published once per request.
	AskContextRequestedEventType = operatordecision.AskContextRequestedEventType

	maxContextQuestionRunes = 2_000
	// maxContextResponseCodePoints bounds the requester's written answer to the
	// operator's questions.
	maxContextResponseCodePoints = 4_000
	// maxOpenContextRequests bounds the unaddressed requests of one ask, so a
	// stuck client cannot flood the requester's event stream.
	maxOpenContextRequests = 5
	// MaxWaitSlice is the longest one Wait call lasts, whatever deadline the
	// caller asked for. A connection held for hours is cut by write deadlines,
	// proxies and tunnels, and the caller sees a bare EOF; ending the call
	// cleanly, ask still open, lets the caller wait again. Clients that want a
	// longer wait loop (the CLI does).
	MaxWaitSlice = 10 * time.Minute
	// WaitSliceReason says why Wait returned although neither the answer nor the
	// caller's deadline was reached.
	WaitSliceReason = "wait_slice: the ask is still open and this wait reached its longest length; wait again"
	// WaitYieldReason retains the resumable wait_slice prefix for existing clients.
	WaitYieldReason = "wait_slice: the ask is still open and the service is preparing for maintenance; wait again"
	// waitFallbackPoll is how often Wait re-reads an ask without a change
	// notice. Notices cover every change this process makes; the poll covers a
	// change made behind its back.
	waitFallbackPoll = time.Second
	// contextOutboxBatch bounds one drain of the context-requested outbox.
	contextOutboxBatch = 50
)

// ErrContextRequestLimit reports an ask that already has the most unaddressed
// context requests it accepts.
var ErrContextRequestLimit = errors.New("too many unanswered context requests on this ask")

// ErrContextNotAccepted reports a context request for an ask whose requester did
// not declare that it answers them. The operator is never offered the button for
// such an ask, so this is a stale or hand-made call; nothing was stored.
var ErrContextNotAccepted = errors.New("context_not_accepted: this ask's requester cannot answer questions; decide with what the ask shows")

// AskContextRequest is one question the recipient asked about an open ask.
type AskContextRequest struct {
	ID          string
	Question    string
	RequestedBy string
	RequestedAt string
	// AddressedAt and AddressedRevision are set when the requester revised the
	// ask after the request; AddressedRevision is the revision number it became.
	AddressedAt       string
	AddressedRevision int
	// Response is the requester's written answer (AskSpec.ContextResponse); empty
	// when the requester revised the ask without one.
	Response string
}

// Addressed reports whether the requester revised the ask since the request.
func (r AskContextRequest) Addressed() bool { return r.AddressedAt != "" }

const contextRequestColumns = `id, question, requested_by, requested_at, addressed_at, addressed_revision, response`

func scanContextRequest(row rowScanner) (AskContextRequest, error) {
	var request AskContextRequest
	err := row.Scan(&request.ID, &request.Question, &request.RequestedBy, &request.RequestedAt, &request.AddressedAt, &request.AddressedRevision, &request.Response)
	return request, err
}

// RequestContext records the recipient's question about an open ask. The ask
// stays open. Only the ask's recipient may ask, and only while the ask waits.
// Repeating a question that is still unanswered returns the existing request,
// so a double tap cannot flood the requester.
func (s *Service) RequestContext(ctx context.Context, askID, actor, question string) (AskContextRequest, error) {
	question = strings.TrimSpace(question)
	if err := checkText("question", question, 1, maxContextQuestionRunes); err != nil {
		return AskContextRequest{}, fmt.Errorf("%w: %s", ErrInvalidArgument, err.Error())
	}
	// A transaction that holds the write lock from its first statement makes the
	// state check, the limit, and the insert one decision.
	tx, err := s.beginWriteTx(ctx)
	if err != nil {
		return AskContextRequest{}, err
	}
	defer func() { _ = tx.Rollback() }()
	ask, err := scanAsk(tx.QueryRowContext(ctx, askSelect+` WHERE a.id = ?`, strings.TrimSpace(askID)))
	if err != nil {
		return AskContextRequest{}, err
	}
	if strings.TrimSpace(actor) == "" || actor != ask.Recipient {
		return AskContextRequest{}, ErrNotFound
	}
	if !ask.Open() {
		return AskContextRequest{}, fmt.Errorf("%w: ask is already %s", ErrAskResolved, ask.State)
	}
	if !ask.AcceptsContextRequests {
		return AskContextRequest{}, ErrContextNotAccepted
	}
	existing, err := listContextRequestsTx(ctx, tx, ask.ID, true)
	if err != nil {
		return AskContextRequest{}, err
	}
	for _, request := range existing {
		if request.Question == question {
			return request, nil
		}
	}
	if len(existing) >= maxOpenContextRequests {
		return AskContextRequest{}, fmt.Errorf("%w: %d requests are waiting for the requester to revise the ask", ErrContextRequestLimit, len(existing))
	}
	request := AskContextRequest{ID: uuid.NewString(), Question: question, RequestedBy: actor, RequestedAt: storedTime(s.clock.Now())}
	if _, err := tx.ExecContext(ctx, `INSERT INTO ask_context_requests (id, ask_id, question, requested_by, requested_at) VALUES (?, ?, ?, ?, ?)`,
		request.ID, ask.ID, request.Question, request.RequestedBy, request.RequestedAt); err != nil {
		return AskContextRequest{}, err
	}
	if err := tx.Commit(); err != nil {
		return AskContextRequest{}, err
	}
	s.askChanges.notify()
	s.kickResolutions(ctx)
	return request, nil
}

// ListContextRequests returns an ask's context requests, oldest first.
func (s *Service) ListContextRequests(ctx context.Context, askID string) ([]AskContextRequest, error) {
	return listContextRequestsTx(ctx, s.db, strings.TrimSpace(askID), false)
}

// rowsQuerier is the read half shared by the pool and a transaction.
type rowsQuerier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func listContextRequestsTx(ctx context.Context, q rowsQuerier, askID string, unaddressedOnly bool) ([]AskContextRequest, error) {
	query := `SELECT ` + contextRequestColumns + ` FROM ask_context_requests WHERE ask_id = ?`
	if unaddressedOnly {
		query += ` AND addressed_at = ''`
	}
	rows, err := q.QueryContext(ctx, query+` ORDER BY requested_at ASC, rowid ASC`, askID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AskContextRequest
	for rows.Next() {
		request, err := scanContextRequest(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, request)
	}
	return out, rows.Err()
}

// oldestOpenContextRequest returns the oldest request the requester has not
// addressed yet.
func (s *Service) oldestOpenContextRequest(ctx context.Context, askID string) (*AskContextRequest, error) {
	request, err := scanContextRequest(s.db.QueryRowContext(ctx, `SELECT `+contextRequestColumns+` FROM ask_context_requests WHERE ask_id = ? AND addressed_at = '' ORDER BY requested_at ASC, rowid ASC LIMIT 1`, askID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &request, nil
}

// addressContextRequestsTx marks every open request of an ask addressed by the
// revision the ask just became, and stores the requester's written answer, if
// it gave one, with each of them. The requests stay: they are the history of
// what the operator asked and what they were told.
func addressContextRequestsTx(ctx context.Context, tx dbTx, askID string, revision int, response, now string) error {
	_, err := tx.ExecContext(ctx, `UPDATE ask_context_requests SET addressed_at = ?, addressed_revision = ?, response = ? WHERE ask_id = ? AND addressed_at = ''`, now, revision, response, askID)
	return err
}

// carryContextRequestsTx copies an ask's context requests, answers included, to
// the ask that replaced it. A replacement is a new ask for the same decision, and
// the operator who asked for context must find the question and its answer on the
// ask they now see. The copies are marked published: the requester is told of a
// request once, through the ask that received it, which stays in the outbox.
func carryContextRequestsTx(ctx context.Context, tx dbTx, from, to, now string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO ask_context_requests (id, ask_id, question, requested_by, requested_at, addressed_at, addressed_revision, response, published_at)
SELECT lower(hex(randomblob(16))), ?, question, requested_by, requested_at, addressed_at, 0, response, ? FROM ask_context_requests WHERE ask_id = ?`, to, now, from)
	return err
}

// PublishPendingContextRequests drains the context-requested outbox. The event
// id derives from its content, so a retry after a lost acknowledgement is
// deduplicated by vrooli-events.
func (s *Service) PublishPendingContextRequests(ctx context.Context) error {
	s.resolutions.mu.Lock()
	publisher := s.resolutions.publisher
	s.resolutions.mu.Unlock()
	if publisher == nil {
		return nil
	}
	s.resolutions.drain.Lock()
	defer s.resolutions.drain.Unlock()
	rows, err := s.db.QueryContext(ctx, `SELECT `+contextRequestColumns+`, ask_id FROM ask_context_requests WHERE published_at = '' ORDER BY requested_at ASC, rowid ASC LIMIT ?`, contextOutboxBatch)
	if err != nil {
		return err
	}
	type pending struct {
		request AskContextRequest
		askID   string
	}
	var batch []pending
	for rows.Next() {
		var item pending
		if err := rows.Scan(&item.request.ID, &item.request.Question, &item.request.RequestedBy, &item.request.RequestedAt, &item.request.AddressedAt, &item.request.AddressedRevision, &item.request.Response, &item.askID); err != nil {
			rows.Close()
			return err
		}
		batch = append(batch, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	var failures []error
	for _, item := range batch {
		ask, err := s.GetAsk(ctx, item.askID)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		if err := publisher.PublishDomainEvent(ctx, AskContextRequestedEvent(ask, item.request)); err != nil {
			failures = append(failures, fmt.Errorf("publish %s for ask %s: %w", AskContextRequestedEventType, item.askID, err))
			continue
		}
		if _, err := s.db.ExecContext(ctx, `UPDATE ask_context_requests SET published_at = ? WHERE id = ?`, storedTime(s.clock.Now()), item.request.ID); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

// AskContextRequestedEvent is the requester notification for one context
// request. The payload is facts only; consumers match on source and
// requester_ref (the event id of the revision the ask shows now) or on
// correlation, then revise the ask in place to answer.
func AskContextRequestedEvent(ask AskRecord, request AskContextRequest) eventbus.DomainEvent {
	occurred, _ := time.Parse(time.RFC3339Nano, request.RequestedAt)
	return eventbus.DomainEvent{
		Source:    "notification-hub",
		EventType: AskContextRequestedEventType,
		Occurred:  occurred.UTC(),
		Payload: operatordecision.ContextRequest{
			AskID: ask.ID, RequestID: request.ID, Question: request.Question, RequestedAt: request.RequestedAt,
			Source: ask.Source, SourceEventType: ask.SourceEventType, RequesterRef: ask.RequesterRef,
			SubjectKey: ask.SubjectKey, Correlation: ask.Correlation,
		}.Payload(),
	}
}

func correlationPayload(correlation map[string]string) map[string]any {
	out := make(map[string]any, len(correlation))
	for key, value := range correlation {
		out[key] = value
	}
	return out
}

// publishAskEvents drains every ask outbox: resolutions and context requests.
func (s *Service) publishAskEvents(ctx context.Context) error {
	return errors.Join(s.PublishPendingResolutions(ctx), s.PublishPendingContextRequests(ctx))
}

// askChangeFeed tells waiters that some ask changed. Waiters take the channel
// before they read, so a change between the read and the wait still wakes them.
// The zero value is ready to use.
type askChangeFeed struct {
	mu sync.Mutex
	ch chan struct{}
	// poll overrides waitFallbackPoll when set (tests).
	poll atomic.Int64
	// slice overrides MaxWaitSlice when set (tests).
	slice atomic.Int64
}

// fallbackPoll is how often a waiter re-reads without a notice.
func (f *askChangeFeed) fallbackPoll() time.Duration {
	if poll := time.Duration(f.poll.Load()); poll > 0 {
		return poll
	}
	return waitFallbackPoll
}

// waitSlice is the longest a single Wait lasts.
func (f *askChangeFeed) waitSlice() time.Duration {
	if slice := time.Duration(f.slice.Load()); slice > 0 {
		return slice
	}
	return MaxWaitSlice
}

// SetWaitSliceForTest shortens the longest single Wait, so a test can see a
// wait end by slice without waiting ten minutes.
func (s *Service) SetWaitSliceForTest(slice time.Duration) {
	s.askChanges.slice.Store(int64(slice))
}

// SetWaitFallbackPollForTest changes how often Wait re-reads an ask when no
// change notice arrived, so a test can prove that notices, not polling, wake it.
func (s *Service) SetWaitFallbackPollForTest(interval time.Duration) {
	s.askChanges.poll.Store(int64(interval))
}

// next returns a channel that is closed at the next change.
func (f *askChangeFeed) next() <-chan struct{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.ch == nil {
		f.ch = make(chan struct{})
	}
	return f.ch
}

// notify wakes every waiter that took a channel before this call.
func (f *askChangeFeed) notify() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.ch != nil {
		close(f.ch)
		f.ch = nil
	}
}

// SetBeforeAnswerCommitForTest runs hook after AnswerAsk validated an answer and
// before it is stored, so a test can change the ask in that window. nil removes it.
func (s *Service) SetBeforeAnswerCommitForTest(hook func()) {
	if hook == nil {
		s.beforeAnswerCommit.Store(nil)
		return
	}
	s.beforeAnswerCommit.Store(&hook)
}

// WaitOptions are the caller's terms for Wait.
type WaitOptions struct {
	// Deadline ends the wait; the zero time waits for the ask or the context.
	Deadline time.Time
	// ReturnOnContextRequest also ends the wait while the ask is open and the
	// recipient asked for context the requester has not addressed.
	ReturnOnContextRequest bool
	// Yield asks a resumable wait to return its current state without changing it.
	Yield <-chan struct{}
}

// WaitResult is the state Wait returned for.
type WaitResult struct {
	// Ask is the ask as it is now, including the answer, note, responses and
	// summary once it is resolved.
	Ask AskRecord
	// Reason says why Wait returned when the ask is still open.
	Reason string
	// ContextRequest is the oldest unaddressed request when Wait returned for
	// one.
	ContextRequest *AskContextRequest
}

// Wait blocks until the ask is resolved, the caller's deadline passes, or (when
// asked) the recipient requests context. It never changes the ask: when the
// caller's deadline passes first it returns the open state, and the ask keeps
// waiting for the operator.
func (s *Service) Wait(ctx context.Context, askID string, opts WaitOptions) (WaitResult, error) {
	sliceEnd := s.clock.Now().Add(s.askChanges.waitSlice())
	for {
		// Take the change channel before reading, so a change after the read
		// wakes the next select instead of being lost.
		changed := s.askChanges.next()
		record, err := s.GetAsk(ctx, askID)
		if err != nil {
			return WaitResult{}, err
		}
		if !record.Open() {
			return WaitResult{Ask: record, Reason: record.Reason}, nil
		}
		if opts.ReturnOnContextRequest {
			request, err := s.oldestOpenContextRequest(ctx, record.ID)
			if err != nil {
				return WaitResult{}, err
			}
			if request != nil {
				return WaitResult{Ask: record, Reason: "the operator asked for more context; answer it with context_response", ContextRequest: request}, nil
			}
		}
		pause := s.askChanges.fallbackPoll()
		end, endReason := sliceEnd, WaitSliceReason
		if !opts.Deadline.IsZero() && !opts.Deadline.After(sliceEnd) {
			end, endReason = opts.Deadline, "the wait timed out; the ask is still open"
		}
		remaining := end.Sub(s.clock.Now())
		if remaining <= 0 {
			return WaitResult{Ask: record, Reason: endReason}, nil
		}
		pause = min(pause, remaining)
		timer := time.NewTimer(pause)
		select {
		case <-ctx.Done():
			timer.Stop()
			return WaitResult{}, ctx.Err()
		case <-opts.Yield:
			timer.Stop()
			return WaitResult{Ask: record, Reason: WaitYieldReason}, nil
		case <-changed:
		case <-timer.C:
		}
		timer.Stop()
	}
}
