package hub_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
	"github.com/vrooli/api-core/schedule"
	"google.golang.org/protobuf/types/known/structpb"

	hub "notification-hub/internal/hub"
)

// answeringSpec is a backup decision whose requester answers context requests by
// revising it. Every revision restates the declaration, as it restates the rest.
func answeringSpec(now time.Time, eventID string) hub.AskSpec {
	spec := subjectSpec(now, eventID)
	spec.AcceptsContextRequests = true
	return spec
}

// openSubjectAsk opens a backup decision (recipient alice) with the answer form.
func openSubjectAsk(t *testing.T, service *hub.Service) hub.AskRecord {
	t.Helper()
	at(service, formNow)
	return askSubject(t, service, answeringSpec(formNow, "evt-1")).Ask
}

// revise sends the subject ask again with changed text, which updates it in place.
func revise(t *testing.T, service *hub.Service, eventID string, at_ time.Time, question string) hub.AskResult {
	t.Helper()
	at(service, at_)
	spec := answeringSpec(at_, eventID)
	spec.Question = question
	result := askSubject(t, service, spec)
	require.Equal(t, hub.AskUpdated, result.Outcome)
	return result
}

func TestRequestContext_RecordsTheQuestionAndKeepsTheAskOpen(t *testing.T) {
	service, _ := testService(t)
	ask := openSubjectAsk(t, service)
	ctx := context.Background()

	request, err := service.RequestContext(ctx, ask.ID, "alice", "  What did the last run say?  ")
	require.NoError(t, err)
	require.NotEmpty(t, request.ID)
	require.Equal(t, "What did the last run say?", request.Question)
	require.Equal(t, "alice", request.RequestedBy)
	require.Equal(t, hub.StoredTime(formNow), request.RequestedAt)
	require.False(t, request.Addressed())

	got, err := service.GetAsk(ctx, ask.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskPending, got.State, "asking for context never answers or closes the ask")
	require.Equal(t, 1, got.OpenContextRequestCount)
	listed, err := service.ListContextRequests(ctx, ask.ID)
	require.NoError(t, err)
	require.Equal(t, []hub.AskContextRequest{request}, listed)

	page, err := service.ListAsksPage(ctx, "alice", hub.AskListOptions{OpenOnly: true})
	require.NoError(t, err)
	require.Equal(t, 1, page.Asks[0].OpenContextRequestCount, "list rows carry the count")
}

func TestRequestContext_IsRecipientOnlyAndNeedsAnOpenAsk(t *testing.T) {
	service, _ := testService(t)
	ask := openSubjectAsk(t, service)
	ctx := context.Background()

	_, err := service.RequestContext(ctx, ask.ID, "bob", "Why?")
	require.ErrorIs(t, err, hub.ErrNotFound, "another recipient cannot even tell the ask exists")
	_, err = service.RequestContext(ctx, ask.ID, "", "Why?")
	require.ErrorIs(t, err, hub.ErrNotFound)
	_, err = service.RequestContext(ctx, "no-such-ask", "alice", "Why?")
	require.ErrorIs(t, err, hub.ErrNotFound)

	for name, question := range map[string]string{"empty": "", "blank": "  \n ", "too long": strings.Repeat("é", 2001), "invalid utf8": "bad \xff"} {
		_, err = service.RequestContext(ctx, ask.ID, "alice", question)
		require.ErrorIs(t, err, hub.ErrInvalidArgument, name)
	}
	_, err = service.RequestContext(ctx, ask.ID, "alice", strings.Repeat("é", 2000))
	require.NoError(t, err, "2000 characters is the limit, counted in characters")

	_, err = service.AnswerAsk(ctx, hub.AnswerInput{AskID: ask.ID, Answer: "retry", Actor: "alice"})
	require.NoError(t, err)
	_, err = service.RequestContext(ctx, ask.ID, "alice", "Too late?")
	require.ErrorIs(t, err, hub.ErrAskResolved)
}

func TestRequestContext_RepeatingAnOpenQuestionIsOneRequestAndTheLimitIsFive(t *testing.T) {
	service, db := testService(t)
	ask := openSubjectAsk(t, service)
	ctx := context.Background()

	first, err := service.RequestContext(ctx, ask.ID, "alice", "Why now?")
	require.NoError(t, err)
	again, err := service.RequestContext(ctx, ask.ID, "alice", "Why now?")
	require.NoError(t, err)
	require.Equal(t, first.ID, again.ID, "a double tap is not a second request")
	require.Equal(t, 1, countRows(t, db, `ask_context_requests`))

	for i := 2; i <= 5; i++ {
		_, err = service.RequestContext(ctx, ask.ID, "alice", fmt.Sprintf("Question %d?", i))
		require.NoError(t, err)
	}
	_, err = service.RequestContext(ctx, ask.ID, "alice", "Question 6?")
	require.ErrorIs(t, err, hub.ErrContextRequestLimit)
	require.NotErrorIs(t, err, hub.ErrInvalidArgument, "the limit is its own failure, not a bad request")
	repeated, err := service.RequestContext(ctx, ask.ID, "alice", "Question 3?")
	require.NoError(t, err, "repeating an open question still returns it at the limit")
	require.Equal(t, "Question 3?", repeated.Question)

	// A revision addresses them all, so the operator can ask again.
	revise(t, service, "evt-2", formNow.Add(time.Minute), "Backups to the Elements drive are blocked since Tuesday. Retry or skip?")
	_, err = service.RequestContext(ctx, ask.ID, "alice", "Question 6?")
	require.NoError(t, err)
}

func TestRequestContext_ARevisionAddressesEveryOpenRequest(t *testing.T) {
	service, _ := testService(t)
	ask := openSubjectAsk(t, service)
	ctx := context.Background()
	first, err := service.RequestContext(ctx, ask.ID, "alice", "What failed?")
	require.NoError(t, err)
	second, err := service.RequestContext(ctx, ask.ID, "alice", "Since when?")
	require.NoError(t, err)

	// Re-sending the same text is a re-assertion, not an answer to the question.
	at(service, formNow.Add(time.Minute))
	same := answeringSpec(formNow.Add(time.Minute), "evt-1b")
	require.Equal(t, hub.AskReasserted, askSubject(t, service, same).Outcome)
	got, err := service.GetAsk(ctx, ask.ID)
	require.NoError(t, err)
	require.Equal(t, 2, got.OpenContextRequestCount)

	revisedAt := formNow.Add(time.Hour)
	revised := revise(t, service, "evt-2", revisedAt, "Backups to the Elements drive failed twice since Tuesday. Retry or skip?")
	require.Equal(t, 0, revised.Ask.OpenContextRequestCount)
	listed, err := service.ListContextRequests(ctx, ask.ID)
	require.NoError(t, err)
	require.Len(t, listed, 2)
	for i, request := range listed {
		require.True(t, request.Addressed())
		require.Equal(t, hub.StoredTime(revisedAt), request.AddressedAt)
		require.Equal(t, 1, request.AddressedRevision, "the revision number the ask became")
		require.Equal(t, []string{first.ID, second.ID}[i], request.ID, "oldest first")
	}

	later, err := service.RequestContext(ctx, ask.ID, "alice", "And the logs?")
	require.NoError(t, err)
	revise(t, service, "evt-3", revisedAt.Add(time.Hour), "Backups failed twice since Tuesday; logs attached. Retry or skip?")
	listed, err = service.ListContextRequests(ctx, ask.ID)
	require.NoError(t, err)
	require.Equal(t, later.ID, listed[2].ID)
	require.Equal(t, 2, listed[2].AddressedRevision)
	require.Equal(t, 1, listed[0].AddressedRevision, "an earlier request keeps the revision that addressed it")
}

func TestRequestContext_AReplacedAskKeepsItsRequestsAndTheNewAskCarriesThem(t *testing.T) {
	service, _ := testService(t)
	ask := openSubjectAsk(t, service)
	ctx := context.Background()
	_, err := service.RequestContext(ctx, ask.ID, "alice", "What failed?")
	require.NoError(t, err)

	later := formNow.Add(time.Hour)
	at(service, later)
	menu := answeringSpec(later, "evt-2")
	menu.Options = append(menu.Options, hub.AskOption{Key: "wait", Label: "Wait a day"})
	replaced := askSubject(t, service, menu)
	require.Equal(t, hub.AskReplaced, replaced.Outcome)
	require.Zero(t, replaced.Ask.OpenContextRequestCount, "the replacement answers the request, so none is open on the new ask")
	old, err := service.GetAsk(ctx, ask.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskWithdrawn, old.State)
	_, err = service.RequestContext(ctx, ask.ID, "alice", "Still there?")
	require.ErrorIs(t, err, hub.ErrAskResolved)
	carried, err := service.ListContextRequests(ctx, replaced.Ask.ID)
	require.NoError(t, err)
	require.Len(t, carried, 1, "the operator finds their question on the ask they see now")
	require.Equal(t, "What failed?", carried[0].Question)
	require.True(t, carried[0].Addressed(), "the replacement is the revision that answered it")
}

func TestWait_ReturnsOnAContextRequestOnlyWhenAsked(t *testing.T) {
	service, _ := testService(t)
	ask := openSubjectAsk(t, service)
	ctx := context.Background()
	request, err := service.RequestContext(ctx, ask.ID, "alice", "What failed?")
	require.NoError(t, err)

	got, err := service.Wait(ctx, ask.ID, hub.WaitOptions{Deadline: formNow, ReturnOnContextRequest: true})
	require.NoError(t, err)
	require.NotNil(t, got.ContextRequest)
	require.Equal(t, request, *got.ContextRequest)
	require.Equal(t, hub.AskPending, got.Ask.State)
	require.Contains(t, got.Reason, "more context")

	// Without the flag the same call keeps waiting for the answer, and stops at
	// the caller's deadline with the ask still open.
	plain, err := service.Wait(ctx, ask.ID, hub.WaitOptions{Deadline: formNow})
	require.NoError(t, err)
	require.Nil(t, plain.ContextRequest)
	require.Contains(t, plain.Reason, "the wait timed out")

	// The oldest unaddressed request is the one returned; addressing clears it.
	_, err = service.RequestContext(ctx, ask.ID, "alice", "Since when?")
	require.NoError(t, err)
	got, err = service.Wait(ctx, ask.ID, hub.WaitOptions{Deadline: formNow, ReturnOnContextRequest: true})
	require.NoError(t, err)
	require.Equal(t, "What failed?", got.ContextRequest.Question)
	revise(t, service, "evt-2", formNow.Add(time.Minute), "Backups failed twice since Tuesday. Retry or skip?")
	after, err := service.Wait(ctx, ask.ID, hub.WaitOptions{Deadline: formNow.Add(time.Minute), ReturnOnContextRequest: true})
	require.NoError(t, err)
	require.Nil(t, after.ContextRequest, "a revised ask has nothing left to answer, so Wait goes back to waiting")
	require.Contains(t, after.Reason, "the wait timed out")
}

func TestWait_WakesWhenTheAskChangesWithoutPolling(t *testing.T) {
	service, _ := testService(t)
	ask := openSubjectAsk(t, service)
	// With the fallback poll pushed out of reach, only a change notice can wake
	// a waiter.
	service.SetWaitFallbackPollForTest(time.Hour)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	type outcome struct {
		result hub.WaitResult
		err    error
	}
	start := func(options hub.WaitOptions) <-chan outcome {
		done := make(chan outcome, 1)
		go func() {
			result, err := service.Wait(ctx, ask.ID, options)
			done <- outcome{result, err}
		}()
		return done
	}
	forContext := start(hub.WaitOptions{ReturnOnContextRequest: true})
	forAnswer := start(hub.WaitOptions{})
	time.Sleep(50 * time.Millisecond)
	select {
	case <-forContext:
		t.Fatal("Wait returned before anything changed")
	default:
	}

	_, err := service.RequestContext(ctx, ask.ID, "alice", "What failed?")
	require.NoError(t, err)
	got := <-forContext
	require.NoError(t, got.err)
	require.Equal(t, "What failed?", got.result.ContextRequest.Question)
	select {
	case <-forAnswer:
		t.Fatal("a caller that did not ask for context requests must keep waiting")
	case <-time.After(100 * time.Millisecond):
	}

	_, err = service.AnswerAsk(ctx, hub.AnswerInput{AskID: ask.ID, Answer: "retry", Actor: "alice", Note: "go"})
	require.NoError(t, err)
	answered := <-forAnswer
	require.NoError(t, answered.err)
	require.Equal(t, hub.AskAnswered, answered.result.Ask.State)
	require.Equal(t, "retry", answered.result.Ask.Answer)
	require.Equal(t, "go", answered.result.Ask.Note)
}

func TestWait_StopsWhenTheCallerGoesAwayOrTheAskIsUnknown(t *testing.T) {
	service, _ := testService(t)
	ask := openSubjectAsk(t, service)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := service.Wait(ctx, ask.ID, hub.WaitOptions{})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	_, err = service.Wait(context.Background(), "no-such-ask", hub.WaitOptions{})
	require.ErrorIs(t, err, hub.ErrNotFound)
}

func contextEvents(publisher *recordingPublisher) []eventbus.DomainEvent {
	var out []eventbus.DomainEvent
	for _, event := range publisher.snapshot() {
		if event.EventType == hub.AskContextRequestedEventType {
			out = append(out, event)
		}
	}
	return out
}

func TestContextRequestedEvent_PublishesOnceThroughTheOutbox(t *testing.T) {
	service, _ := testService(t)
	publisher := &recordingPublisher{}
	service.SetResolutionPublisher(publisher)
	ask := openSubjectAsk(t, service)
	ctx := context.Background()
	request, err := service.RequestContext(ctx, ask.ID, "alice", "What failed?")
	require.NoError(t, err)

	require.Eventually(t, func() bool { return len(contextEvents(publisher)) == 1 }, asyncWait, 10*time.Millisecond, "the request publishes without waiting for the sweep")
	require.NoError(t, service.PublishPendingResolutions(ctx))
	require.NoError(t, service.PublishPendingContextRequests(ctx))
	require.Len(t, contextEvents(publisher), 1, "a published request is never published again")

	event := contextEvents(publisher)[0]
	require.Equal(t, "notification-hub", event.Source)
	require.Equal(t, "notification_hub.ask.context_requested.v1", event.EventType)
	require.Equal(t, formNow, event.Occurred)
	_, err = structpb.NewStruct(event.Payload)
	require.NoError(t, err, "the payload must encode through the Vrooli Events serializer")
	require.Equal(t, map[string]any{
		"ask_id": ask.ID, "request_id": request.ID, "question": "What failed?", "requested_at": request.RequestedAt,
		"source": subjectSource, "source_event_type": subjectEventType, "requester_ref": "evt-1", "subject_key": subjectKey,
		"correlation": map[string]any{"destination": "elements", "failures": "1"},
	}, event.Payload)

	raw, err := json.Marshal(event.Payload)
	require.NoError(t, err)
	decoded, err := operatordecision.DecodeContextRequest(raw)
	require.NoError(t, err, "a producer decodes the event with the shared struct")
	require.Equal(t, ask.ID, decoded.AskID)
	require.Equal(t, subjectKey, decoded.SubjectKey)
	require.Equal(t, "elements", decoded.Correlation["destination"])
}

type flakyPublisher struct {
	mu     sync.Mutex
	failed bool
	events []eventbus.DomainEvent
}

func (p *flakyPublisher) PublishDomainEvent(_ context.Context, event eventbus.DomainEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if event.EventType == hub.AskContextRequestedEventType && !p.failed {
		p.failed = true
		return errors.New("events unavailable")
	}
	p.events = append(p.events, event)
	return nil
}

func (p *flakyPublisher) published() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.events)
}

func TestContextRequestedEvent_ARequestSurvivesAFailedPublishAndALaterRevision(t *testing.T) {
	service, db := testService(t)
	publisher := &flakyPublisher{}
	ctx := context.Background()
	ask := openSubjectAsk(t, service)
	_, err := service.RequestContext(ctx, ask.ID, "alice", "What failed?")
	require.NoError(t, err)

	// No publisher yet: the request waits in the outbox.
	require.NoError(t, service.PublishPendingContextRequests(ctx))
	service.SetResolutionPublisher(publisher)
	require.Error(t, service.PublishPendingContextRequests(ctx), "a failed publish is reported")
	require.Zero(t, publisher.published())
	var unpublished int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ask_context_requests WHERE published_at = ''`).Scan(&unpublished))
	require.Equal(t, 1, unpublished)

	// The requester revises the ask before the retry. The event still goes out,
	// with the ask's current event id.
	revise(t, service, "evt-2", formNow.Add(time.Minute), "Backups failed twice since Tuesday. Retry or skip?")
	require.NoError(t, service.PublishPendingContextRequests(ctx), "the sweep's retry succeeds")
	require.Equal(t, 1, publisher.published())
	require.Equal(t, "evt-2", publisher.events[0].Payload["requester_ref"])
}

func TestContextRequestsReadsAndRetentionFollowTheAsk(t *testing.T) {
	service, db := testService(t)
	ask := openSubjectAsk(t, service)
	ctx := context.Background()
	_, err := service.RequestContext(ctx, ask.ID, "alice", "What failed?")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `DELETE FROM asks WHERE id = ?`, ask.ID)
	require.NoError(t, err)
	require.Zero(t, countRows(t, db, `ask_context_requests`), "requests are deleted with their ask")
}

// The CLI tells a requester to answer a context request by sending the ask again
// with the same subject key and options and a new idempotency key.
func TestRequestContext_AnsweredByRevisingWithANewIdempotencyKey(t *testing.T) {
	service, db := testService(t)
	sender := &recordingSender{}
	service.SetPushSender(sender)
	registerSubscription(t, service, "alice")
	ask := openSubjectAsk(t, service)
	waitDelivered(t, service, ask)
	ctx := context.Background()
	_, err := service.RequestContext(ctx, ask.ID, "alice", "What failed?")
	require.NoError(t, err)
	pushesBefore := pushCount(sender)

	revisedAt := formNow.Add(time.Minute)
	revised := revise(t, service, "evt-2", revisedAt, "Backups failed twice since Tuesday. Retry or skip?")
	require.Equal(t, ask.ID, revised.Ask.ID, "the same ask, updated in place")
	require.Equal(t, "evt-2", revised.Ask.RequesterRef)
	require.Zero(t, revised.Ask.OpenContextRequestCount, "the open request is addressed")
	require.Equal(t, 1, countAsks(t, db, `subject_key = ?`, subjectKey))
	require.Equal(t, pushesBefore, pushCount(sender), "no new push for the revision")

	// The same key again with different text is a client bug, not an update: it
	// is refused, and nothing changes.
	at(service, revisedAt.Add(time.Minute))
	conflicting := answeringSpec(revisedAt.Add(time.Minute), "evt-2")
	conflicting.Question = "Something else entirely. Retry or skip?"
	_, err = service.AskDecision(ctx, conflicting, "")
	require.ErrorIs(t, err, hub.ErrAskIdentityConflict)
	got, err := service.GetAsk(ctx, ask.ID)
	require.NoError(t, err)
	require.Equal(t, "Backups failed twice since Tuesday. Retry or skip?", got.Question)

	// Redelivering an earlier revision's key changes nothing either.
	stale := answeringSpec(revisedAt.Add(2*time.Minute), "evt-1")
	stale.Question = "An older wording. Retry or skip?"
	replayed, err := service.AskDecision(ctx, stale, "")
	require.NoError(t, err)
	require.Equal(t, hub.AskReasserted, replayed.Outcome)
	require.Equal(t, got.Question, replayed.Ask.Question)
}

func TestRequestContext_RefusesAnAskWhoseRequesterDidNotDeclareItAnswers(t *testing.T) {
	service, _ := testService(t)
	publisher := &recordingPublisher{}
	service.SetResolutionPublisher(publisher)
	at(service, formNow)
	ask := askSubject(t, service, subjectSpec(formNow, "evt-silent")).Ask
	require.False(t, ask.CanRequestContext(), "a requester that never declared it answers gets no button")

	_, err := service.RequestContext(context.Background(), ask.ID, "alice", "Why?")
	require.ErrorIs(t, err, hub.ErrContextNotAccepted)
	got, err := service.GetAsk(context.Background(), ask.ID)
	require.NoError(t, err)
	require.Zero(t, got.OpenContextRequestCount, "a refused question is not stored")
	require.NoError(t, service.PublishPendingContextRequests(context.Background()))
	require.Empty(t, contextEvents(publisher), "and never sent to a sender that cannot reply")
	_, answerErr := service.AnswerAsk(context.Background(), hub.AnswerInput{AskID: ask.ID, Answer: "retry", Actor: "alice"})
	require.NoError(t, answerErr, "the operator can always answer")
}

func TestCanRequestContext_FollowsTheDeclarationOfTheCurrentRevisionAndTheAskState(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	ask := openSubjectAsk(t, service)
	require.True(t, ask.CanRequestContext())

	// A revision restates the declaration. Dropping it updates in place (no push)
	// and closes the button; declaring again reopens it.
	undeclared := subjectSpec(formNow.Add(time.Minute), "evt-1b")
	at(service, formNow.Add(time.Minute))
	result := askSubject(t, service, undeclared)
	require.Equal(t, hub.AskUpdated, result.Outcome)
	require.Equal(t, ask.ID, result.Ask.ID)
	require.False(t, result.Ask.CanRequestContext())
	_, err := service.RequestContext(ctx, ask.ID, "alice", "Why?")
	require.ErrorIs(t, err, hub.ErrContextNotAccepted)

	at(service, formNow.Add(2*time.Minute))
	result = askSubject(t, service, answeringSpec(formNow.Add(2*time.Minute), "evt-1c"))
	require.Equal(t, hub.AskUpdated, result.Outcome)
	require.True(t, result.Ask.CanRequestContext())

	_, err = service.AnswerAsk(ctx, hub.AnswerInput{AskID: ask.ID, Answer: "retry", Actor: "alice"})
	require.NoError(t, err)
	answered, err := service.GetAsk(ctx, ask.ID)
	require.NoError(t, err)
	require.True(t, answered.AcceptsContextRequests)
	require.False(t, answered.CanRequestContext(), "a resolved ask takes no more questions")
}

func TestAcceptsContextRequests_NeedsASubjectKeyBecauseTheAnswerIsARevision(t *testing.T) {
	service, _ := testService(t)
	at(service, formNow)
	spec := decisionSpec(formNow, "evt-plain")
	spec.AcceptsContextRequests = true
	result, err := service.AskDecision(context.Background(), spec, "")
	require.NoError(t, err, "a declaration the Hub cannot honour never fails the ask")
	require.False(t, result.Ask.AcceptsContextRequests)
	require.False(t, result.Ask.CanRequestContext())
	var found bool
	for _, warning := range result.Warnings {
		if warning.Code == hub.WarnContextRequestsNeedSubjectKey {
			found = true
			require.Equal(t, hub.WarningDropped, warning.Severity)
			require.Equal(t, "accepts_context_requests", warning.Field)
		}
	}
	require.True(t, found, "warnings: %+v", result.Warnings)
}

func TestWait_EndsAtItsLongestLengthWithAReasonTheCallerCanActOn(t *testing.T) {
	service, _ := testService(t)
	service.SetClockForTest(schedule.System())
	ask := askSubject(t, service, answeringSpec(time.Now(), "evt-slice")).Ask
	service.SetWaitSliceForTest(50 * time.Millisecond)

	// A deadline far past the slice: the wait still ends, ask open, and says why.
	started := time.Now()
	result, err := service.Wait(context.Background(), ask.ID, hub.WaitOptions{Deadline: time.Now().Add(time.Hour)})
	require.NoError(t, err)
	require.Less(t, time.Since(started), 10*time.Second)
	require.Equal(t, hub.AskPending, result.Ask.State)
	require.Equal(t, hub.WaitSliceReason, result.Reason)

	// A deadline inside the slice keeps its own reason.
	service.SetWaitSliceForTest(time.Hour)
	result, err = service.Wait(context.Background(), ask.ID, hub.WaitOptions{Deadline: time.Now().Add(50 * time.Millisecond)})
	require.NoError(t, err)
	require.Contains(t, result.Reason, "the wait timed out")
}
