package hub_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	hub "notification-hub/internal/hub"
	"notification-hub/internal/modules"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/storage"
)

const (
	subjectSource    = "data-backup-manager"
	subjectEventType = "data_backup_manager.backup_failure.decision_requested.v1"
	subjectKey       = "backup-failure:elements"
)

// subjectSpec is a blocked-backup decision: no default, so the sweep never
// resolves it behind the test's back.
func subjectSpec(now time.Time, eventID string) hub.AskSpec {
	spec := decisionSpec(now, eventID)
	spec.Source = subjectSource
	spec.SourceEventType = subjectEventType
	spec.Title = "Decision needed: Backups"
	spec.Question = "Backups to the Elements drive are blocked. Retry or skip?"
	spec.Options = []hub.AskOption{{Key: "retry", Label: "Retry"}, {Key: "skip", Label: "Skip"}}
	spec.Recommended = "retry"
	spec.RecommendationReason = "the drive is mounted again"
	spec.DefaultAnswer = ""
	spec.Reversible = false
	spec.Correlation = map[string]string{"destination": "elements", "failures": "1"}
	spec.SubjectKey = subjectKey
	return spec
}

func askSubject(t *testing.T, service *hub.Service, spec hub.AskSpec) hub.AskResult {
	t.Helper()
	result, err := service.AskDecision(context.Background(), spec, "")
	require.NoError(t, err)
	return result
}

func at(service *hub.Service, now time.Time) { service.SetClockForTest(fixedClock{now: now}) }

func countAsks(t *testing.T, db *database.RoutedDB, where string, args ...any) int {
	t.Helper()
	var count int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM asks WHERE `+where, args...).Scan(&count))
	return count
}

func pushCount(sender *recordingSender) int {
	sender.mu.Lock()
	defer sender.mu.Unlock()
	return len(sender.messages)
}

func waitDelivered(t *testing.T, service *hub.Service, ask hub.AskRecord) {
	t.Helper()
	require.Eventually(t, func() bool {
		n, _, err := service.Get(context.Background(), ask.NotificationID)
		return err == nil && n.State == "delivered"
	}, 2*time.Second, 10*time.Millisecond)
}

func TestSubject_NoOpenDecisionCreatesAndDeliversOne(t *testing.T) {
	service, db := testService(t)
	sender := &recordingSender{}
	service.SetPushSender(sender)
	registerSubscription(t, service, "alice")
	now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	at(service, now)

	result := askSubject(t, service, subjectSpec(now, "evt-1"))

	require.Equal(t, hub.AskCreated, result.Outcome)
	require.Equal(t, subjectKey, result.Ask.SubjectKey)
	require.Equal(t, hub.AskPending, result.Ask.State)
	waitDelivered(t, service, result.Ask)
	require.Equal(t, 1, pushCount(sender))
	require.Equal(t, 1, countAsks(t, db, `subject_key = ?`, subjectKey))
}

func TestSubject_IdenticalRepeatDeliversNothingAndRecordsTheReassertion(t *testing.T) {
	service, db := testService(t)
	sender := &recordingSender{}
	service.SetPushSender(sender)
	registerSubscription(t, service, "alice")
	start := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	at(service, start)
	first := askSubject(t, service, subjectSpec(start, "evt-1"))
	waitDelivered(t, service, first.Ask)

	later := start.Add(10 * time.Minute)
	at(service, later)
	repeat := subjectSpec(later, "evt-2") // a re-send is a new event with the same facts
	repeat.AssertedAt = later
	second := askSubject(t, service, repeat)

	require.Equal(t, hub.AskReasserted, second.Outcome)
	require.Equal(t, first.Ask.ID, second.Ask.ID)
	require.Equal(t, 1, countAsks(t, db, `subject_key = ?`, subjectKey))
	require.Equal(t, 1, pushCount(sender))
	got, err := service.GetAsk(context.Background(), first.Ask.ID)
	require.NoError(t, err)
	reasserted, err := time.Parse(time.RFC3339Nano, got.LastAssertedAt)
	require.NoError(t, err)
	require.True(t, reasserted.Equal(later), "last asserted %s, want %s", got.LastAssertedAt, later)
	require.Zero(t, got.RevisionCount, "an identical re-send is not an update")
}

func TestSubject_ChangedTextUpdatesInPlaceWithoutANewPush(t *testing.T) {
	service, db := testService(t)
	sender := &recordingSender{}
	service.SetPushSender(sender)
	registerSubscription(t, service, "alice")
	publisher := &recordingPublisher{}
	service.SetResolutionPublisher(publisher)
	start := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	at(service, start)
	first := askSubject(t, service, subjectSpec(start, "evt-1"))
	waitDelivered(t, service, first.Ask)

	later := start.Add(time.Hour)
	at(service, later)
	changed := subjectSpec(later, "evt-2")
	changed.Question = "Backups to the Elements drive have been blocked for an hour. Retry or skip?"
	changed.Correlation = map[string]string{"destination": "elements", "failures": "7"}
	changed.Title = "Decision needed: Backups (7 failures)"
	changed.Deadline = later.Add(48 * time.Hour)
	changed.DeadlineSpecified = true
	updated := askSubject(t, service, changed)

	require.Equal(t, hub.AskUpdated, updated.Outcome)
	require.Equal(t, first.Ask.ID, updated.Ask.ID)
	require.Equal(t, first.Notification.ID, updated.Notification.ID)
	require.Equal(t, 1, updated.Ask.RevisionCount)
	require.Equal(t, changed.Question, updated.Ask.Question)
	require.Equal(t, "7", updated.Ask.Correlation["failures"])
	require.Equal(t, later.Add(48*time.Hour).Format(time.RFC3339Nano), updated.Ask.Deadline, "the new revision's deadline applies")

	// The inbox shows the notification, so it carries the current text.
	n, _, err := service.Get(context.Background(), first.Ask.NotificationID)
	require.NoError(t, err)
	require.Equal(t, changed.Title, n.Title)
	require.Contains(t, n.Body, "blocked for an hour")

	// No second notification and no second push.
	require.Equal(t, 1, countAsks(t, db, `subject_key = ?`, subjectKey))
	var notifications int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM notifications`).Scan(&notifications))
	require.Equal(t, 1, notifications)
	time.Sleep(100 * time.Millisecond)
	require.Equal(t, 1, pushCount(sender))

	// The previous text is kept as a revision, newest first.
	revisions, err := service.ListAskRevisions(context.Background(), first.Ask.ID)
	require.NoError(t, err)
	require.Len(t, revisions, 1)
	require.Equal(t, 1, revisions[0].Revision)
	require.Equal(t, "Backups to the Elements drive are blocked. Retry or skip?", revisions[0].Question)
	require.Equal(t, "evt-1", revisions[0].EventID)
	require.Equal(t, "1", revisions[0].Correlation["failures"])

	// An answer to the updated decision carries the latest correlation.
	_, err = service.AnswerAsk(context.Background(), hub.AnswerInput{AskID: first.Ask.ID, Answer: "retry", Actor: "alice"})
	require.NoError(t, err)
	require.NoError(t, service.PublishPendingResolutions(context.Background()))
	require.Eventually(t, func() bool { return len(publisher.snapshot()) == 1 }, time.Second, 10*time.Millisecond)
	correlation := publisher.snapshot()[0].Payload["correlation"].(map[string]any)
	require.Equal(t, "7", correlation["failures"])
}

func TestSubject_ChangedAnswerMenuReplacesAndDeliversTheNewDecision(t *testing.T) {
	menus := map[string]func(*hub.AskSpec){
		"option keys": func(s *hub.AskSpec) {
			s.Options = []hub.AskOption{{Key: "retry", Label: "Retry"}, {Key: "skip", Label: "Skip"}, {Key: "pause", Label: "Pause"}}
		},
		"recommended": func(s *hub.AskSpec) { s.Recommended = "skip" },
		"default":     func(s *hub.AskSpec) { s.DefaultAnswer, s.Reversible = "skip", true },
		"urgency":     func(s *hub.AskSpec) { s.Urgency = "critical" },
		"reversible":  func(s *hub.AskSpec) { s.Reversible = true },
	}
	for name, change := range menus {
		t.Run(name, func(t *testing.T) {
			service, db := testService(t)
			sender := &recordingSender{}
			service.SetPushSender(sender)
			registerSubscription(t, service, "alice")
			publisher := &recordingPublisher{}
			service.SetResolutionPublisher(publisher)
			start := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
			at(service, start)
			old := askSubject(t, service, subjectSpec(start, "evt-1"))
			waitDelivered(t, service, old.Ask)

			next := subjectSpec(start, "evt-2")
			change(&next)
			replaced := askSubject(t, service, next)

			require.Equal(t, hub.AskReplaced, replaced.Outcome)
			require.NotEqual(t, old.Ask.ID, replaced.Ask.ID)
			closed, err := service.GetAsk(context.Background(), old.Ask.ID)
			require.NoError(t, err)
			require.Equal(t, hub.AskWithdrawn, closed.State)
			require.Equal(t, "Replaced by a newer version of this decision", closed.Reason)
			require.Equal(t, replaced.Ask.ID, closed.SupersededByAskID)
			require.Empty(t, closed.Answer)
			require.Equal(t, 1, countAsks(t, db, `subject_key = ? AND state IN ('pending','escalated')`, subjectKey), "one open decision per subject")
			waitDelivered(t, service, replaced.Ask)
			require.Equal(t, 2, pushCount(sender), "the replacement is delivered")

			// The producer learns which decision replaced the old one.
			require.NoError(t, service.PublishPendingResolutions(context.Background()))
			require.Eventually(t, func() bool { return len(publisher.snapshot()) == 1 }, time.Second, 10*time.Millisecond)
			payload := publisher.snapshot()[0].Payload
			require.Equal(t, hub.AskWithdrawn, payload["state"])
			require.Equal(t, subjectKey, payload["subject_key"])
			require.Equal(t, replaced.Ask.ID, payload["superseded_by_ask_id"])
		})
	}
}

func TestSubject_LabelOnlyChangeIsText_NotANewMenu(t *testing.T) {
	service, _ := testService(t)
	now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	at(service, now)
	first := askSubject(t, service, subjectSpec(now, "evt-1"))
	relabeled := subjectSpec(now, "evt-2")
	relabeled.Options = []hub.AskOption{{Key: "retry", Label: "Retry the backup now"}, {Key: "skip", Label: "Skip this run"}}

	got := askSubject(t, service, relabeled)

	require.Equal(t, hub.AskUpdated, got.Outcome)
	require.Equal(t, first.Ask.ID, got.Ask.ID)
	require.Equal(t, "Retry the backup now", got.Ask.Options[0].Label)
}

func TestSubject_SubjectsAreScopedToSourceTypeAndKey(t *testing.T) {
	service, db := testService(t)
	now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	at(service, now)
	askSubject(t, service, subjectSpec(now, "evt-1"))
	otherKey := subjectSpec(now, "evt-2")
	otherKey.SubjectKey = "backup-failure:other-drive"
	otherSource := subjectSpec(now, "evt-3")
	otherSource.Source = "storage-manager"
	otherType := subjectSpec(now, "evt-4")
	otherType.SourceEventType = "data_backup_manager.restore.decision_requested.v1"
	for _, spec := range []hub.AskSpec{otherKey, otherSource, otherType} {
		require.Equal(t, hub.AskCreated, askSubject(t, service, spec).Outcome)
	}
	require.Equal(t, 4, countAsks(t, db, `state = 'pending'`))
}

func TestSubject_DismissedDecisionIsNotReopenedByIdenticalTermsOnly(t *testing.T) {
	service, db := testService(t)
	start := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	at(service, start)
	first := askSubject(t, service, subjectSpec(start, "evt-1"))
	_, err := service.DismissAsk(context.Background(), first.Ask.ID, "alice", "not now")
	require.NoError(t, err)

	later := start.Add(6 * time.Hour)
	at(service, later)
	suppressed := askSubject(t, service, subjectSpec(later, "evt-2"))
	require.Equal(t, hub.AskSuppressed, suppressed.Outcome)
	require.Equal(t, first.Ask.ID, suppressed.Ask.ID)
	require.Equal(t, hub.AskDismissed, suppressed.Ask.State)
	require.Equal(t, 1, countAsks(t, db, `subject_key = ?`, subjectKey), "no new decision until the terms change")

	changed := subjectSpec(later, "evt-3")
	changed.Question = "Backups to the Elements drive are still blocked and a second drive is failing too."
	reopened := askSubject(t, service, changed)
	require.Equal(t, hub.AskCreated, reopened.Outcome)
	require.NotEqual(t, first.Ask.ID, reopened.Ask.ID)
	require.Equal(t, hub.AskPending, reopened.Ask.State)
}

func TestSubject_AnsweredDecisionIsReturnedForFifteenMinutesThenReopens(t *testing.T) {
	service, _ := testService(t)
	start := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	at(service, start)
	first := askSubject(t, service, subjectSpec(start, "evt-1"))
	resolvedAt := start.Add(2 * time.Minute)
	at(service, resolvedAt)
	_, err := service.AnswerAsk(context.Background(), hub.AnswerInput{AskID: first.Ask.ID, Answer: "retry", Actor: "alice"})
	require.NoError(t, err)

	within := resolvedAt.Add(14 * time.Minute)
	at(service, within)
	returned := askSubject(t, service, subjectSpec(within, "evt-2"))
	require.Equal(t, hub.AskReturned, returned.Outcome, "the producer has not processed the answer yet")
	require.Equal(t, first.Ask.ID, returned.Ask.ID)
	require.Equal(t, hub.AskAnswered, returned.Ask.State)

	after := resolvedAt.Add(16 * time.Minute)
	at(service, after)
	reopened := askSubject(t, service, subjectSpec(after, "evt-3"))
	require.Equal(t, hub.AskCreated, reopened.Outcome)
	require.NotEqual(t, first.Ask.ID, reopened.Ask.ID)
}

func TestSubject_DefaultedDecisionIsReturnedForFifteenMinutes(t *testing.T) {
	service, _ := testService(t)
	service.SetPushSender(&recordingSender{})
	registerSubscription(t, service, "alice")
	start := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	at(service, start)
	spec := subjectSpec(start, "evt-1")
	spec.DefaultAnswer, spec.Reversible = "retry", true
	spec.Deadline, spec.DeadlineSpecified = start.Add(time.Minute), true
	first := askSubject(t, service, spec)
	require.Eventually(t, func() bool {
		got, err := service.GetAsk(context.Background(), first.Ask.ID)
		return err == nil && got.FirstDeliveredAt != ""
	}, 2*time.Second, 10*time.Millisecond)
	defaultedAt := start.Add(13 * time.Hour)
	at(service, defaultedAt)
	require.NoError(t, service.ProcessEscalations(context.Background()))
	got, err := service.GetAsk(context.Background(), first.Ask.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskDefaulted, got.State)

	again := subjectSpec(defaultedAt, "evt-2")
	again.DefaultAnswer, again.Reversible = "retry", true
	again.Deadline, again.DeadlineSpecified = start.Add(time.Minute), true
	returned := askSubject(t, service, again)
	require.Equal(t, hub.AskReturned, returned.Outcome)
	require.Equal(t, first.Ask.ID, returned.Ask.ID)
}

func TestSubject_ChangedTermsAfterAnswerOpenANewDecisionImmediately(t *testing.T) {
	service, _ := testService(t)
	start := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	at(service, start)
	first := askSubject(t, service, subjectSpec(start, "evt-1"))
	_, err := service.AnswerAsk(context.Background(), hub.AnswerInput{AskID: first.Ask.ID, Answer: "retry", Actor: "alice"})
	require.NoError(t, err)

	changed := subjectSpec(start, "evt-2")
	changed.Question = "A different drive is blocked now."
	got := askSubject(t, service, changed)
	require.Equal(t, hub.AskCreated, got.Outcome)
}

func TestSubject_RequestAssertedBeforeTheLastWithdrawalIsStale(t *testing.T) {
	service, db := testService(t)
	start := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	at(service, start)
	first := askSubject(t, service, subjectSpec(start, "evt-1"))

	withdrawnAt := start.Add(time.Hour)
	at(service, withdrawnAt.Add(time.Minute))
	closed, err := service.WithdrawDecisionSubject(context.Background(), hub.SubjectWithdrawal{
		Source: subjectSource, SourceEventType: subjectEventType, SubjectKey: subjectKey,
		Reason: "Backups to the Elements drive succeeded again at 10:00.", AssertedAt: withdrawnAt,
	})
	require.NoError(t, err)
	require.True(t, closed)
	got, err := service.GetAsk(context.Background(), first.Ask.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskWithdrawn, got.State)
	require.Equal(t, "Backups to the Elements drive succeeded again at 10:00.", got.Reason)

	// A delayed request from before the withdrawal is not delivered.
	stale := subjectSpec(start, "evt-late")
	stale.AssertedAt = start.Add(30 * time.Minute)
	stale.Question = "an older, delayed question"
	result, err := service.AskDecision(context.Background(), stale, "")
	require.NoError(t, err)
	require.Equal(t, hub.AskStale, result.Outcome)
	require.Empty(t, result.Ask.ID)
	require.Equal(t, 1, countAsks(t, db, `subject_key = ?`, subjectKey), "a stale request creates nothing")
	var notifications int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM notifications`).Scan(&notifications))
	require.Equal(t, 1, notifications)
	_, _, legacyErr := service.AskWithSpec(context.Background(), stale)
	require.ErrorIs(t, legacyErr, hub.ErrAskStale)

	// The situation recurring after the withdrawal opens a new decision.
	recurred := subjectSpec(start, "evt-recurred")
	recurred.AssertedAt = withdrawnAt.Add(30 * time.Second)
	require.Equal(t, hub.AskCreated, askSubject(t, service, recurred).Outcome)
}

func TestSubject_WithdrawalClosesOnlyAnOpenDecisionAndKeepsOtherOutcomes(t *testing.T) {
	service, _ := testService(t)
	publisher := &recordingPublisher{}
	service.SetResolutionPublisher(publisher)
	now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	at(service, now)
	withdraw := func(key string) bool {
		changed, err := service.WithdrawDecisionSubject(context.Background(), hub.SubjectWithdrawal{
			Source: subjectSource, SourceEventType: subjectEventType, SubjectKey: key, Reason: "recovered",
		})
		require.NoError(t, err)
		return changed
	}

	answered := askSubject(t, service, subjectSpec(now, "evt-a"))
	_, err := service.AnswerAsk(context.Background(), hub.AnswerInput{AskID: answered.Ask.ID, Answer: "skip", Actor: "alice"})
	require.NoError(t, err)
	require.False(t, withdraw(subjectKey), "an answered decision keeps its outcome")
	got, _ := service.GetAsk(context.Background(), answered.Ask.ID)
	require.Equal(t, hub.AskAnswered, got.State)
	require.Equal(t, "skip", got.Answer)

	dismissedSpec := subjectSpec(now, "evt-d")
	dismissedSpec.SubjectKey = "backup-failure:dismissed"
	dismissed := askSubject(t, service, dismissedSpec)
	_, err = service.DismissAsk(context.Background(), dismissed.Ask.ID, "alice", "")
	require.NoError(t, err)
	require.False(t, withdraw("backup-failure:dismissed"), "a dismissed decision keeps its outcome")
	got, _ = service.GetAsk(context.Background(), dismissed.Ask.ID)
	require.Equal(t, hub.AskDismissed, got.State)

	openSpec := subjectSpec(now, "evt-o")
	openSpec.SubjectKey = "backup-failure:open"
	open := askSubject(t, service, openSpec)
	require.True(t, withdraw("backup-failure:open"))
	got, _ = service.GetAsk(context.Background(), open.Ask.ID)
	require.Equal(t, hub.AskWithdrawn, got.State)
	require.Equal(t, "recovered", got.Reason)
	require.Empty(t, got.AnsweredBy)
	require.False(t, withdraw("backup-failure:open"), "repeating a withdrawal is a no-op")

	// Another source cannot close it.
	other := askSubject(t, service, func() hub.AskSpec {
		spec := subjectSpec(now, "evt-x")
		spec.SubjectKey = "backup-failure:x"
		return spec
	}())
	changed, err := service.WithdrawDecisionSubject(context.Background(), hub.SubjectWithdrawal{
		Source: "someone-else", SourceEventType: subjectEventType, SubjectKey: "backup-failure:x", Reason: "nope",
	})
	require.NoError(t, err)
	require.False(t, changed)
	got, _ = service.GetAsk(context.Background(), other.Ask.ID)
	require.Equal(t, hub.AskPending, got.State)

	require.NoError(t, service.PublishPendingResolutions(context.Background()))
	var withdrawn []string
	for _, event := range publisher.snapshot() {
		if event.Payload["state"] == hub.AskWithdrawn {
			withdrawn = append(withdrawn, fmt.Sprint(event.Payload["subject_key"], "|", event.Payload["reason"]))
		}
	}
	require.Equal(t, []string{"backup-failure:open|recovered"}, withdrawn)
}

func TestSubject_WithdrawalByRequesterRefMatchesEarlierRevisionsAndSuccessors(t *testing.T) {
	service, _ := testService(t)
	now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	at(service, now)
	first := askSubject(t, service, subjectSpec(now, "evt-1"))
	update := subjectSpec(now, "evt-2")
	update.Question = "Changed question."
	askSubject(t, service, update)
	replace := subjectSpec(now, "evt-3")
	replace.Recommended = "skip"
	replacement := askSubject(t, service, replace)
	require.Equal(t, hub.AskReplaced, replacement.Outcome)

	// "evt-1" opened the first decision; the producer still holds that id.
	changed, err := service.WithdrawDecisionSubject(context.Background(), hub.SubjectWithdrawal{
		Source: subjectSource, SourceEventType: subjectEventType, RequesterRef: "evt-1", Reason: "recovered",
	})
	require.NoError(t, err)
	require.True(t, changed, "closing by an old id closes what carries that situation now")
	closed, err := service.GetAsk(context.Background(), replacement.Ask.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskWithdrawn, closed.State)
	require.Equal(t, "recovered", closed.Reason)
	original, err := service.GetAsk(context.Background(), first.Ask.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskReplacedReason, original.Reason, "the replaced decision keeps its own closure")
}

func TestSubject_RevisionEventIdMatchesWithdrawal(t *testing.T) {
	service, _ := testService(t)
	now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	at(service, now)
	first := askSubject(t, service, subjectSpec(now, "evt-1"))
	update := subjectSpec(now, "evt-2")
	update.Question = "Changed question."
	askSubject(t, service, update)
	update2 := subjectSpec(now, "evt-3")
	update2.Question = "Changed again."
	askSubject(t, service, update2)

	// evt-2 was the current revision until evt-3 replaced it.
	changed, err := service.WithdrawDecisionSubject(context.Background(), hub.SubjectWithdrawal{
		Source: subjectSource, SourceEventType: subjectEventType, RequesterRef: "evt-2", Reason: "recovered",
	})
	require.NoError(t, err)
	require.True(t, changed)
	got, err := service.GetAsk(context.Background(), first.Ask.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskWithdrawn, got.State)
}

func TestSubject_WithdrawalBeforeItsRequestLeavesTheDelayedRequestUndelivered(t *testing.T) {
	service, db := testService(t)
	now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	at(service, now)

	changed, err := service.WithdrawDecisionSubject(context.Background(), hub.SubjectWithdrawal{
		Source: subjectSource, SourceEventType: subjectEventType, RequesterRef: "evt-early", Reason: "recovered",
	})
	require.NoError(t, err)
	require.False(t, changed)

	// A subject-less request carrying that id is not delivered either.
	plain := subjectSpec(now, "evt-early")
	plain.SubjectKey = ""
	_, _, err = service.AskWithSpec(context.Background(), plain)
	require.ErrorIs(t, err, hub.ErrAskWithdrawn)
	// Nor is the same id with a subject key.
	_, err = service.AskDecision(context.Background(), subjectSpec(now, "evt-early"), "")
	require.ErrorIs(t, err, hub.ErrAskWithdrawn)
	require.Zero(t, countAsks(t, db, `1=1`))
	var notifications int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM notifications`).Scan(&notifications))
	require.Zero(t, notifications)
}

func TestSubject_OutOfOrderEventsDoNotRollTheDecisionBack(t *testing.T) {
	service, _ := testService(t)
	start := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	at(service, start.Add(2*time.Hour))
	first := subjectSpec(start, "evt-1")
	first.AssertedAt = start
	askSubject(t, service, first)
	newer := subjectSpec(start, "evt-3")
	newer.AssertedAt = start.Add(time.Hour)
	newer.Question = "Newest text."
	require.Equal(t, hub.AskUpdated, askSubject(t, service, newer).Outcome)

	// A delayed re-send asserted before the newest one changes nothing.
	delayed := subjectSpec(start, "evt-2")
	delayed.AssertedAt = start.Add(30 * time.Minute)
	delayed.Question = "Delayed older text."
	got := askSubject(t, service, delayed)
	require.Equal(t, hub.AskReasserted, got.Outcome)
	require.Equal(t, "Newest text.", got.Ask.Question)

	// A withdrawal asserted before the newest confirmation does not close it.
	changed, err := service.WithdrawDecisionSubject(context.Background(), hub.SubjectWithdrawal{
		Source: subjectSource, SourceEventType: subjectEventType, SubjectKey: subjectKey,
		Reason: "recovered", AssertedAt: start.Add(45 * time.Minute),
	})
	require.NoError(t, err)
	require.False(t, changed)
	current, err := service.GetAsk(context.Background(), got.Ask.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskPending, current.State)
	// One asserted after it does.
	changed, err = service.WithdrawDecisionSubject(context.Background(), hub.SubjectWithdrawal{
		Source: subjectSource, SourceEventType: subjectEventType, SubjectKey: subjectKey,
		Reason: "recovered", AssertedAt: start.Add(90 * time.Minute),
	})
	require.NoError(t, err)
	require.True(t, changed)
}

func TestSubject_RedeliveredEventsChangeNothing(t *testing.T) {
	service, db := testService(t)
	now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	at(service, now)
	original := subjectSpec(now, "evt-1")
	first := askSubject(t, service, original)
	update := subjectSpec(now, "evt-2")
	update.Question = "Updated question."
	askSubject(t, service, update)

	// The first event arrives again after the update: the ask has moved past it.
	again := askSubject(t, service, original)
	require.Equal(t, first.Ask.ID, again.Ask.ID)
	require.Equal(t, "Updated question.", again.Ask.Question)
	require.Equal(t, 1, again.Ask.RevisionCount)
	// The current event again is the same ask, unchanged.
	require.Equal(t, "Updated question.", askSubject(t, service, update).Ask.Question)
	require.Equal(t, 1, countAsks(t, db, `subject_key = ?`, subjectKey))

	// The same id for different terms is a client bug, not a redelivery.
	conflicting := update
	conflicting.Question = "Same id, other words."
	_, err := service.AskDecision(context.Background(), conflicting, "")
	require.ErrorIs(t, err, hub.ErrAskIdentityConflict)
}

func TestSubject_ReassertLapseClosesTheDecisionAndAReassertKeepsItOpen(t *testing.T) {
	service, _ := testService(t)
	publisher := &recordingPublisher{}
	service.SetResolutionPublisher(publisher)
	start := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	at(service, start)
	spec := subjectSpec(start, "evt-1")
	spec.ReassertWithin = 10 * time.Minute
	kept := askSubject(t, service, spec)
	lapsedSpec := subjectSpec(start, "evt-2")
	lapsedSpec.SubjectKey = "backup-failure:lapses"
	lapsedSpec.ReassertWithin = 10 * time.Minute
	lapsed := askSubject(t, service, lapsedSpec)
	forever := subjectSpec(start, "evt-3")
	forever.SubjectKey = "backup-failure:no-window"
	foreverAsk := askSubject(t, service, forever)

	// Nine minutes in, one producer re-sends; the other stays quiet.
	at(service, start.Add(9*time.Minute))
	again := subjectSpec(start, "evt-4")
	again.AssertedAt = start.Add(9 * time.Minute)
	again.ReassertWithin = 10 * time.Minute
	require.Equal(t, hub.AskReasserted, askSubject(t, service, again).Outcome)
	require.NoError(t, service.ProcessEscalations(context.Background()))
	state := func(id string) hub.AskRecord {
		got, err := service.GetAsk(context.Background(), id)
		require.NoError(t, err)
		return got
	}
	require.Equal(t, hub.AskPending, state(lapsed.Ask.ID).State, "inside its window")

	at(service, start.Add(11*time.Minute))
	require.NoError(t, service.ProcessEscalations(context.Background()))
	closed := state(lapsed.Ask.ID)
	require.Equal(t, hub.AskWithdrawn, closed.State)
	require.Equal(t, "No longer confirmed by data-backup-manager since Nov 13 09:00 UTC", closed.Reason)
	require.Equal(t, hub.AskPending, state(kept.Ask.ID).State, "the re-send moved its window forward")
	require.Equal(t, hub.AskPending, state(foreverAsk.Ask.ID).State, "no window, no lapse")

	at(service, start.Add(20*time.Minute))
	require.NoError(t, service.ProcessEscalations(context.Background()))
	require.Equal(t, hub.AskWithdrawn, state(kept.Ask.ID).State)
	require.Equal(t, hub.AskPending, state(foreverAsk.Ask.ID).State)
	require.NoError(t, service.PublishPendingResolutions(context.Background()))
	require.Eventually(t, func() bool { return len(publisher.snapshot()) == 2 }, time.Second, 10*time.Millisecond)
}

func TestSubject_NeverDeliveredDefaultAsksExpireSevenDaysAfterTheirDeadline(t *testing.T) {
	service, _ := testService(t)
	publisher := &recordingPublisher{}
	service.SetResolutionPublisher(publisher)
	start := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	at(service, start)
	spec := decisionSpec(start, "evt-undelivered") // reversible, default "re-aim", deadline +1h
	ask, _, err := service.AskWithSpec(context.Background(), spec)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		n, _, getErr := service.Get(context.Background(), ask.NotificationID)
		return getErr == nil && n.State == "unroutable"
	}, time.Second, 10*time.Millisecond)

	deadline := start.Add(time.Hour)
	at(service, deadline.Add(7*24*time.Hour-time.Minute))
	require.NoError(t, service.ProcessEscalations(context.Background()))
	got, err := service.GetAsk(context.Background(), ask.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskPending, got.State, "still inside the week")
	require.Contains(t, got.Reason, "operator not reached")

	at(service, deadline.Add(7*24*time.Hour))
	require.NoError(t, service.ProcessEscalations(context.Background()))
	got, err = service.GetAsk(context.Background(), ask.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskExpired, got.State)
	require.Contains(t, got.Reason, "never delivered")
	require.Empty(t, got.Answer, "the withheld default is not applied")
	require.NoError(t, service.PublishPendingResolutions(context.Background()))
	require.Eventually(t, func() bool { return len(publisher.snapshot()) == 1 }, time.Second, 10*time.Millisecond)
	require.Equal(t, hub.AskExpired, publisher.snapshot()[0].Payload["state"])
}

func TestSubject_ConcurrentRequestsLeaveOneOpenDecision(t *testing.T) {
	run := func(t *testing.T, mutate func(i int, spec *hub.AskSpec)) (*hub.Service, *database.RoutedDB) {
		service, db := testService(t)
		now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
		at(service, now)
		const callers = 12
		start := make(chan struct{})
		errs := make(chan error, callers)
		var wg sync.WaitGroup
		for i := 0; i < callers; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				spec := subjectSpec(now, fmt.Sprintf("evt-%02d", i))
				mutate(i, &spec)
				<-start
				_, err := service.AskDecision(context.Background(), spec, "")
				errs <- err
			}(i)
		}
		close(start)
		wg.Wait()
		close(errs)
		for err := range errs {
			require.NoError(t, err)
		}
		require.Equal(t, 1, countAsks(t, db, `subject_key = ? AND state IN ('pending','escalated')`, subjectKey), "exactly one open decision")
		return service, db
	}

	t.Run("same menu, different text", func(t *testing.T) {
		_, db := run(t, func(i int, spec *hub.AskSpec) { spec.Question = fmt.Sprintf("Question revision %d", i) })
		require.Equal(t, 1, countAsks(t, db, `subject_key = ?`, subjectKey), "text changes update the one decision")
	})
	t.Run("different menus", func(t *testing.T) {
		_, db := run(t, func(i int, spec *hub.AskSpec) {
			spec.Options = []hub.AskOption{{Key: "retry", Label: "Retry"}, {Key: fmt.Sprintf("skip-%d", i), Label: "Skip"}}
		})
		total := countAsks(t, db, `subject_key = ?`, subjectKey)
		require.Equal(t, total-1, countAsks(t, db, `subject_key = ? AND state = 'withdrawn' AND superseded_by_ask_id <> ''`, subjectKey), "every other decision was replaced")
	})
	t.Run("identical", func(t *testing.T) {
		_, db := run(t, func(int, *hub.AskSpec) {})
		require.Equal(t, 1, countAsks(t, db, `subject_key = ?`, subjectKey))
	})
}

func TestSubject_RequestsAndWithdrawalsRaceToOneConsistentEnd(t *testing.T) {
	service, db := testService(t)
	now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	at(service, now)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			_, err := service.AskDecision(context.Background(), subjectSpec(now, fmt.Sprintf("evt-%d", i)), "")
			require.NoError(t, err)
		}(i)
		go func() {
			defer wg.Done()
			_, err := service.WithdrawDecisionSubject(context.Background(), hub.SubjectWithdrawal{
				Source: subjectSource, SourceEventType: subjectEventType, SubjectKey: subjectKey, Reason: "recovered",
				AssertedAt: now.Add(-time.Hour),
			})
			require.NoError(t, err)
		}()
	}
	wg.Wait()
	require.LessOrEqual(t, countAsks(t, db, `subject_key = ? AND state IN ('pending','escalated')`, subjectKey), 1)
}

func TestSubject_InvalidSubjectFieldsAreRejected(t *testing.T) {
	service, _ := testService(t)
	now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	for name, mutate := range map[string]func(*hub.AskSpec){
		"leading punctuation": func(s *hub.AskSpec) { s.SubjectKey = "-backup" },
		"space":               func(s *hub.AskSpec) { s.SubjectKey = "backup failure" },
		"too long":            func(s *hub.AskSpec) { s.SubjectKey = strings.Repeat("a", 129) },
		"window without key":  func(s *hub.AskSpec) { s.SubjectKey, s.ReassertWithin = "", time.Hour },
		"window too short":    func(s *hub.AskSpec) { s.ReassertWithin = 30 * time.Second },
		"window too long":     func(s *hub.AskSpec) { s.ReassertWithin = 31 * 24 * time.Hour },
		"negative window":     func(s *hub.AskSpec) { s.ReassertWithin = -time.Minute },
	} {
		t.Run(name, func(t *testing.T) {
			spec := subjectSpec(now, "evt-"+strings.ReplaceAll(name, " ", "-"))
			mutate(&spec)
			_, err := service.AskDecision(context.Background(), spec, "")
			require.ErrorIs(t, err, hub.ErrInvalidArgument)
		})
	}
	for name, withdrawal := range map[string]hub.SubjectWithdrawal{
		"no identity":   {Source: subjectSource, SourceEventType: subjectEventType, Reason: "x"},
		"no reason":     {Source: subjectSource, SourceEventType: subjectEventType, SubjectKey: subjectKey},
		"long reason":   {Source: subjectSource, SourceEventType: subjectEventType, SubjectKey: subjectKey, Reason: strings.Repeat("x", 301)},
		"bad key":       {Source: subjectSource, SourceEventType: subjectEventType, SubjectKey: "bad key", Reason: "x"},
		"not a request": {Source: subjectSource, SourceEventType: "x.y.notification.v1", SubjectKey: subjectKey, Reason: "x"},
		"no source":     {SourceEventType: subjectEventType, SubjectKey: subjectKey, Reason: "x"},
	} {
		t.Run("withdrawal "+name, func(t *testing.T) {
			_, err := service.WithdrawDecisionSubject(context.Background(), withdrawal)
			require.ErrorIs(t, err, hub.ErrInvalidArgument)
		})
	}
}

func TestSubject_DirectAsksUpsertAndWithdrawBySubjectKey(t *testing.T) {
	service, _ := testService(t)
	now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	at(service, now)
	direct := func(key, question string) hub.AskSpec {
		return hub.AskSpec{
			Recipient: "alice", Question: question, SubjectKey: "disk-pressure",
			Options:     []hub.AskOption{{Key: "clean", Label: "Clean"}, {Key: "wait", Label: "Wait"}},
			Recommended: "clean", Deadline: now.Add(24 * time.Hour), DeadlineSpecified: true,
			SensitivityLabel: "private", IdempotencyKey: key,
		}
	}
	first, err := service.AskDecision(context.Background(), direct("k1", "Disk is 91% full."), "agent-1")
	require.NoError(t, err)
	require.Equal(t, hub.AskCreated, first.Outcome)
	require.Equal(t, "alice", first.Ask.Source, "a direct ask's source is its requester")
	updated, err := service.AskDecision(context.Background(), direct("k2", "Disk is 94% full."), "agent-1")
	require.NoError(t, err)
	require.Equal(t, hub.AskUpdated, updated.Outcome)
	require.Equal(t, first.Ask.ID, updated.Ask.ID)

	// Another requester's subject is its own.
	_, err = service.WithdrawDirectSubject(context.Background(), "bob", "disk-pressure", "done")
	require.ErrorIs(t, err, hub.ErrNotFound)

	closed, err := service.WithdrawDirectSubject(context.Background(), "alice", "disk-pressure", "Disk is at 40% again.")
	require.NoError(t, err)
	require.Equal(t, hub.AskWithdrawn, closed.State)
	require.Equal(t, "Disk is at 40% again.", closed.Reason)
	again, err := service.WithdrawDirectSubject(context.Background(), "alice", "disk-pressure", "Disk is at 40% again.")
	require.NoError(t, err)
	require.Equal(t, closed.ID, again.ID, "repeating the withdrawal returns the closed ask")
	_, err = service.WithdrawDirectSubject(context.Background(), "alice", "no-such-subject", "x")
	require.ErrorIs(t, err, hub.ErrNotFound)
}

// The live database predates subject keys. Opening it must add the columns,
// indexes and tables without touching existing asks.
func TestSubject_MigrationAddsSubjectsToAnExistingDatabase(t *testing.T) {
	ctx := context.Background()
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, primary, modules.AllSchemas()...))
	legacyNotification := insertPending(t, database.NewFromPrimary(primary), "alice", "Legacy question?", "private")
	// Rewind to the deployed shape: the asks table exactly as the live database
	// has it, with none of the subject tables.
	_, err := primary.ExecContext(ctx, `
DROP TABLE ask_revisions;
DROP TABLE decision_subject_withdrawals;
DROP TABLE asks;
CREATE TABLE asks (
  id TEXT PRIMARY KEY,
  notification_id TEXT NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
  question TEXT NOT NULL,
  allowed_answers TEXT NOT NULL,
  deadline TEXT NOT NULL,
  state TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
, options_json TEXT NOT NULL DEFAULT '', recommended TEXT NOT NULL DEFAULT '', recommendation_reason TEXT NOT NULL DEFAULT '', default_answer TEXT NOT NULL DEFAULT '', reversible INTEGER NOT NULL DEFAULT 0, urgency TEXT NOT NULL DEFAULT '', context_url TEXT NOT NULL DEFAULT '', source TEXT NOT NULL DEFAULT '', source_event_type TEXT NOT NULL DEFAULT '', requester_ref TEXT NOT NULL DEFAULT '', correlation_json TEXT NOT NULL DEFAULT '', default_applied_at TEXT NOT NULL DEFAULT '', resolved_at TEXT NOT NULL DEFAULT '', resolution_published_at TEXT NOT NULL DEFAULT '', big_picture TEXT NOT NULL DEFAULT '', technical_details TEXT NOT NULL DEFAULT '');
CREATE UNIQUE INDEX idx_asks_notification ON asks(notification_id);
CREATE UNIQUE INDEX idx_asks_requester_identity ON asks(source, requester_ref) WHERE requester_ref <> '';
CREATE INDEX idx_asks_resolution_outbox ON asks(resolution_published_at, resolved_at, id);`)
	require.NoError(t, err)
	_, err = primary.ExecContext(ctx, `INSERT INTO asks (id, notification_id, question, allowed_answers, deadline, state, created_at, updated_at, options_json, source, source_event_type, requester_ref) VALUES ('legacy-ask', ?, 'Legacy question?', '["a","b"]', '2026-10-07T00:00:00Z', 'pending', '2026-10-06T00:00:00Z', '2026-10-06T00:00:00Z', '', 'delivery-reliability-check', 'delivery_reliability.check.decision_requested.v1', 'legacy-event')`, legacyNotification)
	require.NoError(t, err)

	require.NoError(t, database.EnsureSchemas(ctx, primary, modules.AllSchemas()...), "schema.sql must run on the old shape")
	db := database.NewFromPrimary(primary)
	service := hub.New(db, nil, nil)
	service.SetClockForTest(fixedClock{now: time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)})

	legacy, err := service.GetAsk(ctx, "legacy-ask")
	require.NoError(t, err)
	require.Equal(t, "Legacy question?", legacy.Question)
	require.Equal(t, hub.AskPending, legacy.State)
	require.Empty(t, legacy.SubjectKey)
	require.Zero(t, legacy.RevisionCount)

	for _, column := range []string{"subject_key", "last_asserted_at", "reassert_within_seconds", "revision_count", "superseded_by_ask_id", "current_event_id"} {
		var found int
		require.NoError(t, primary.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info('asks') WHERE name = ?`, column).Scan(&found))
		require.Equal(t, 1, found, column)
	}
	var uniqueOpen int
	require.NoError(t, primary.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_asks_open_subject' AND sql LIKE '%UNIQUE%'`).Scan(&uniqueOpen))
	require.Equal(t, 1, uniqueOpen)
	for _, table := range []string{"ask_revisions", "decision_subject_withdrawals"} {
		var found int
		require.NoError(t, primary.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&found))
		require.Equal(t, 1, found, table)
	}

	// The migrated database enforces one open decision per subject.
	now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	first := askSubject(t, service, subjectSpec(now, "evt-1"))
	require.Equal(t, hub.AskCreated, first.Outcome)
	dupNotification := insertPending(t, db, "alice", "duplicate", "private")
	_, err = primary.ExecContext(ctx, `INSERT INTO asks (id, notification_id, question, allowed_answers, deadline, state, created_at, updated_at, source, source_event_type, requester_ref, subject_key) VALUES ('dup', ?, 'q', '[]', 'd', 'pending', 'c', 'u', ?, ?, 'other-ref', ?)`, dupNotification, subjectSource, subjectEventType, subjectKey)
	require.Error(t, err, "the partial unique index must refuse a second open ask for the subject")
	// A legacy ask resolves as before.
	require.NoError(t, service.ProcessEscalations(ctx))
	got, err := service.GetAsk(ctx, "legacy-ask")
	require.NoError(t, err)
	require.Equal(t, hub.AskExpired, got.State)

	// Re-opening the database (a restart) is idempotent.
	require.NoError(t, database.EnsureSchemas(ctx, primary, modules.AllSchemas()...))
	hub.New(db, nil, nil)
	var open int
	require.NoError(t, primary.QueryRowContext(ctx, `SELECT COUNT(*) FROM asks WHERE subject_key = ?`, subjectKey).Scan(&open))
	require.Equal(t, 1, open)
}

func TestSubject_RevisionHistoryIsBounded(t *testing.T) {
	service, db := testService(t)
	now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	at(service, now)
	var id string
	for i := 0; i < 60; i++ {
		spec := subjectSpec(now, fmt.Sprintf("evt-%02d", i))
		spec.Question = fmt.Sprintf("Question %d", i)
		id = askSubject(t, service, spec).Ask.ID
	}
	got, err := service.GetAsk(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, 59, got.RevisionCount)
	revisions, err := service.ListAskRevisions(context.Background(), id)
	require.NoError(t, err)
	require.Len(t, revisions, 50)
	require.Equal(t, "Question 58", revisions[0].Question)
	var rows int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM ask_revisions`).Scan(&rows))
	require.Equal(t, 50, rows)
}

// Production opens SQLite with deferred transactions and a connection pool, so
// two same-subject requests can both read "no open decision" and then race to
// insert. The unique index must refuse the loser and the retry must re-read
// the winner and apply the rules to it.
func TestSubject_ConcurrentRequestsOnADeferredPoolLeaveOneOpenDecision(t *testing.T) {
	ctx := context.Background()
	dsn, err := storage.SQLiteDSNAt(filepath.Join(t.TempDir(), "deferred.db"), storage.SQLiteTuning{TxLock: "deferred"})
	require.NoError(t, err)
	primary, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	primary.SetMaxOpenConns(8)
	t.Cleanup(func() { _ = primary.Close() })
	require.NoError(t, database.EnsureSchemas(ctx, primary, modules.AllSchemas()...))
	db := database.NewFromPrimary(primary)
	service := hub.New(db, nil, nil)
	now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	at(service, now)

	for round := 0; round < 3; round++ {
		key := fmt.Sprintf("backup-failure:round-%d", round)
		const callers = 10
		start := make(chan struct{})
		errs := make(chan error, callers)
		var wg sync.WaitGroup
		for i := 0; i < callers; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				spec := subjectSpec(now, fmt.Sprintf("evt-%d-%02d", round, i))
				spec.SubjectKey = key
				if i%2 == 1 {
					spec.Question = fmt.Sprintf("Different words %d", i)
				}
				<-start
				_, askErr := service.AskDecision(ctx, spec, "")
				errs <- askErr
			}(i)
		}
		close(start)
		wg.Wait()
		close(errs)
		for askErr := range errs {
			require.NoError(t, askErr)
		}
		require.Equal(t, 1, countAsks(t, db, `subject_key = ?`, key), "one decision, updated in place")
		require.Equal(t, 1, countAsks(t, db, `subject_key = ? AND state IN ('pending','escalated')`, key))
	}
}

// Producers bind an answer to their current event id, so requester_ref follows
// the latest revision; every earlier id still finds the ask.
func TestSubject_RequesterRefFollowsTheLatestRevision(t *testing.T) {
	service, db := testService(t)
	publisher := &recordingPublisher{}
	service.SetResolutionPublisher(publisher)
	start := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	at(service, start)
	first := askSubject(t, service, subjectSpec(start, "evt-1"))
	require.Equal(t, "evt-1", first.Ask.RequesterRef)

	// A re-send with the same facts and a new event id.
	reassert := subjectSpec(start, "evt-2")
	reassert.AssertedAt = start.Add(time.Minute)
	at(service, start.Add(time.Minute))
	got := askSubject(t, service, reassert)
	require.Equal(t, hub.AskReasserted, got.Outcome)
	require.Equal(t, "evt-2", got.Ask.RequesterRef)

	// New words: updated in place; requester_ref is the new event's id.
	update := subjectSpec(start, "evt-3")
	update.Question = "Changed question."
	update.AssertedAt = start.Add(2 * time.Minute)
	at(service, start.Add(2*time.Minute))
	got = askSubject(t, service, update)
	require.Equal(t, hub.AskUpdated, got.Outcome)
	require.Equal(t, first.Ask.ID, got.Ask.ID)
	require.Equal(t, "evt-3", got.Ask.RequesterRef)
	byLatest, err := service.GetAskByRequester(context.Background(), subjectSource, subjectEventType, "evt-3")
	require.NoError(t, err)
	require.Equal(t, first.Ask.ID, byLatest.ID)

	// Redelivered earlier events (first, reassert, revision) never revert it.
	for _, id := range []string{"evt-1", "evt-2"} {
		redelivered := subjectSpec(start, id)
		redelivered.AssertedAt = start.Add(3 * time.Minute)
		again := askSubject(t, service, redelivered)
		require.Equal(t, first.Ask.ID, again.Ask.ID, id)
		require.Equal(t, "Changed question.", again.Ask.Question, id)
		require.Equal(t, "evt-3", again.Ask.RequesterRef, id)
	}
	// An older assertion with a fresh id is ignored as stale news.
	older := subjectSpec(start, "evt-0")
	older.AssertedAt = start
	older.Question = "Older text."
	ignored := askSubject(t, service, older)
	require.Equal(t, "Changed question.", ignored.Ask.Question)
	require.Equal(t, "evt-3", ignored.Ask.RequesterRef)

	// One ask per (source, requester_ref).
	require.Equal(t, 1, countAsks(t, db, `source = ? AND requester_ref = 'evt-3'`, subjectSource))
	require.Zero(t, countAsks(t, db, `source = ? AND requester_ref IN ('evt-1','evt-2')`, subjectSource))

	// The resolution names the latest id.
	_, err = service.AnswerAsk(context.Background(), hub.AnswerInput{AskID: first.Ask.ID, Answer: "retry", Actor: "alice"})
	require.NoError(t, err)
	require.NoError(t, service.PublishPendingResolutions(context.Background()))
	require.Eventually(t, func() bool { return len(publisher.snapshot()) == 1 }, time.Second, 10*time.Millisecond)
	require.Equal(t, "evt-3", publisher.snapshot()[0].Payload["requester_ref"])
}

func TestSubject_WithdrawalMatchesEveryEarlierEventIDOfAnUpdatedAsk(t *testing.T) {
	for _, id := range []string{"evt-1", "evt-2", "evt-3"} {
		t.Run(id, func(t *testing.T) {
			service, _ := testService(t)
			now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
			at(service, now)
			ask := askSubject(t, service, subjectSpec(now, "evt-1")).Ask
			reassert := subjectSpec(now, "evt-2")
			askSubject(t, service, reassert)
			update := subjectSpec(now, "evt-3")
			update.Question = "Changed."
			askSubject(t, service, update)
			changed, err := service.WithdrawDecisionSubject(context.Background(), hub.SubjectWithdrawal{
				Source: subjectSource, SourceEventType: subjectEventType, RequesterRef: id, Reason: "recovered",
			})
			require.NoError(t, err)
			require.True(t, changed)
			got, err := service.GetAsk(context.Background(), ask.ID)
			require.NoError(t, err)
			require.Equal(t, hub.AskWithdrawn, got.State)
			require.Equal(t, "evt-3", got.RequesterRef)
		})
	}
}
