package hub_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	hub "notification-hub/internal/hub"
)

// A revision says only what changes. These tests drive the Hub's revision
// semantics directly: a spec whose Keep set names the fields it left unsaid.

// partialSpec is a revision of subjectSpec's ask that says nothing but the
// identity of the request; the caller says what changes and clears its Keep.
func partialSpec(eventID string) hub.AskSpec {
	full := subjectSpec(time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC), eventID)
	return hub.AskSpec{
		Recipient: full.Recipient, Source: full.Source, SourceEventType: full.SourceEventType, SubjectKey: full.SubjectKey,
		IdempotencyKey: eventID, Deadline: full.Deadline, SensitivityLabel: full.SensitivityLabel, VerifiedProducer: true,
		Keep: hub.AskKeep{
			Question: true, Options: true, Recommended: true, RecommendationReason: true, DefaultAnswer: true, Reversible: true,
			Urgency: true, ContextURL: true, BigPicture: true, TechnicalDetails: true, ReassertWithin: true, AcceptsContextRequests: true,
		},
	}
}

// richAsk opens the ask the revisions below start from: every optional field set.
func richAsk(t *testing.T, service *hub.Service) hub.AskRecord {
	t.Helper()
	spec := subjectSpec(time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC), "evt-0")
	spec.BigPicture = "Backups to the Elements drive stopped at 02:00."
	spec.TechnicalDetails = "Run 41; drive mounted read-only."
	spec.ContextURL = "https://example.test/backups"
	spec.Urgency = "high"
	spec.ReassertWithin = 15 * time.Minute
	spec.AcceptsContextRequests = true
	return askSubject(t, service, spec).Ask
}

func TestRevision_KeepsEveryFieldItDoesNotSay(t *testing.T) {
	service, _ := testService(t)
	at(service, time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC))
	first := richAsk(t, service)

	revision := partialSpec("evt-1")
	revision.TechnicalDetails, revision.Keep.TechnicalDetails = "Run 42; drive remounted.", false
	got := askSubject(t, service, revision)

	require.Equal(t, hub.AskUpdated, got.Outcome)
	require.Equal(t, first.ID, got.Ask.ID)
	require.Equal(t, "Run 42; drive remounted.", got.Ask.TechnicalDetails)
	require.Equal(t, first.Question, got.Ask.Question)
	require.Equal(t, first.Options, got.Ask.Options)
	require.Equal(t, "retry", got.Ask.Recommended)
	require.Equal(t, "the drive is mounted again", got.Ask.RecommendationReason)
	require.Equal(t, "Backups to the Elements drive stopped at 02:00.", got.Ask.BigPicture)
	require.Equal(t, "https://example.test/backups", got.Ask.ContextURL)
	require.Equal(t, "high", got.Ask.Urgency)
	require.Equal(t, 900, got.Ask.ReassertWithinSeconds, "the lapse window is not dropped by a revision that says nothing about it")
	require.True(t, got.Ask.AcceptsContextRequests)
	require.Equal(t, got.Notification.Body, first.Question+"\nRecommended: Retry — the drive is mounted again", "the notification is rendered from the whole ask")
}

func TestRevision_AFieldItSaysEmptyIsCleared(t *testing.T) {
	service, _ := testService(t)
	at(service, time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC))
	richAsk(t, service)

	revision := partialSpec("evt-1")
	revision.BigPicture, revision.Keep.BigPicture = "", false
	revision.ReassertWithin, revision.Keep.ReassertWithin = 0, false
	got := askSubject(t, service, revision)

	require.Equal(t, hub.AskUpdated, got.Outcome)
	require.Empty(t, got.Ask.BigPicture, "saying it empty clears it")
	require.Zero(t, got.Ask.ReassertWithinSeconds)
	require.Equal(t, "Run 41; drive mounted read-only.", got.Ask.TechnicalDetails, "the rest stays")
}

func TestRevision_ANewMenuTakesTheOldRecommendationAndDefaultWithIt(t *testing.T) {
	service, _ := testService(t)
	at(service, time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC))
	first := richAsk(t, service)

	revision := partialSpec("evt-1")
	revision.Options, revision.Keep.Options = []hub.AskOption{{Key: "skip", Label: "Skip"}, {Key: "later", Label: "Later"}}, false
	got := askSubject(t, service, revision)

	require.Equal(t, hub.AskReplaced, got.Outcome, "a changed menu replaces the ask")
	require.NotEqual(t, first.ID, got.Ask.ID)
	require.Empty(t, got.Ask.Recommended, "the old recommendation named an answer of the old menu")
	require.Empty(t, got.Ask.RecommendationReason, "a reason belongs to the answer it explains")
	require.Empty(t, got.Ask.DefaultAnswer)
	require.Equal(t, first.Question, got.Ask.Question, "the question is kept")
	require.Equal(t, first.BigPicture, got.Ask.BigPicture)
	require.Equal(t, 900, got.Ask.ReassertWithinSeconds)
}

func TestRevision_ARecommendationItChangesDoesNotKeepTheOldReason(t *testing.T) {
	service, _ := testService(t)
	at(service, time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC))
	richAsk(t, service)

	revision := partialSpec("evt-1")
	revision.Recommended, revision.Keep.Recommended = "skip", false
	got := askSubject(t, service, revision)

	require.Equal(t, "skip", got.Ask.Recommended)
	require.Empty(t, got.Ask.RecommendationReason, "the old reason explained the old recommendation")
}

func TestRevision_WithNoOpenAskNeedsAQuestionAndAnswers(t *testing.T) {
	service, _ := testService(t)
	at(service, time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC))

	revision := partialSpec("evt-1")
	revision.TechnicalDetails, revision.Keep.TechnicalDetails = "Run 42.", false
	_, err := service.AskDecision(context.Background(), revision, "")

	require.ErrorIs(t, err, hub.ErrInvalidArgument)
	require.ErrorContains(t, err, "no open ask has the subject key")

	first := partialSpec("evt-2")
	first.Question, first.Keep.Question = "Retry the backup?", false
	first.Options, first.Keep.Options = []hub.AskOption{{Key: "yes", Label: "Yes"}, {Key: "no", Label: "No"}}, false
	created := askSubject(t, service, first)
	require.Equal(t, hub.AskCreated, created.Outcome)
	require.Equal(t, "normal", created.Ask.Urgency, "an unsaid field of a new ask takes its default")
}

func TestRevision_CheckAskPreviewsTheSameMerge(t *testing.T) {
	service, _ := testService(t)
	at(service, time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC))
	first := richAsk(t, service)

	revision := partialSpec("evt-1")
	revision.Question, revision.Keep.Question = "Retry tonight's backup now?", false
	check, err := service.CheckAsk(context.Background(), revision, hub.CheckInput{})

	require.NoError(t, err)
	require.Equal(t, first.ID, check.RevisesAskID)
	require.Contains(t, check.PushBody, "Retry tonight's backup now?")
	require.Contains(t, check.PushBody, "Recommended: Retry — the drive is mounted again", "the preview carries the kept recommendation")
}

// Revisions that arrive together each change one of two fields. Both changes must
// survive: each is merged against the ask the others stored, not against the ask
// it saw when it arrived. A revision that restored the other field's old text
// would show as that round's tag missing.
func TestRevision_ConcurrentPartialRevisionsEachKeepWhatTheOthersChanged(t *testing.T) {
	service, _ := testService(t)
	at(service, time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC))
	first := richAsk(t, service)

	const rounds, revisers = 10, 12
	for round := 0; round < rounds; round++ {
		start := make(chan struct{})
		errs := make(chan error, revisers)
		var wg sync.WaitGroup
		for i := 0; i < revisers; i++ {
			revision := partialSpec(fmt.Sprintf("evt-%02d-%02d", round, i))
			if i%2 == 0 {
				revision.TechnicalDetails, revision.Keep.TechnicalDetails = fmt.Sprintf("technical round %d reviser %02d", round, i), false
			} else {
				revision.BigPicture, revision.Keep.BigPicture = fmt.Sprintf("background round %d reviser %02d", round, i), false
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				_, err := service.AskDecision(context.Background(), revision, "")
				errs <- err
			}()
		}
		close(start)
		wg.Wait()
		close(errs)
		for err := range errs {
			require.NoError(t, err)
		}

		got, err := service.GetAsk(context.Background(), first.ID)
		require.NoError(t, err)
		require.Contains(t, got.TechnicalDetails, fmt.Sprintf("technical round %d ", round), "round %d: a technical revision survived the background ones", round)
		require.Contains(t, got.BigPicture, fmt.Sprintf("background round %d ", round), "round %d: a background revision survived the technical ones", round)
		require.Equal(t, first.Question, got.Question)
		require.Equal(t, "retry", got.Recommended)
		require.Equal(t, (round+1)*revisers, got.RevisionCount)
	}
}

// directPartial is partialSpec for a direct request (an agent's CLI or API call):
// it carries no event type, and its idempotency key only lets the sender retry.
func directPartial(key string) hub.AskSpec {
	spec := partialSpec(key)
	spec.SourceEventType = ""
	return spec
}

// openDirectAsk opens the ask a direct sender revises, with the key "direct-0".
func openDirectAsk(t *testing.T, service *hub.Service) hub.AskRecord {
	t.Helper()
	spec := subjectSpec(time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC), "direct-0")
	spec.SourceEventType = ""
	spec.TechnicalDetails = "Run 41."
	return askSubject(t, service, spec).Ask
}

func detailsRevision(key, details string) hub.AskSpec {
	revision := directPartial(key)
	revision.TechnicalDetails, revision.Keep.TechnicalDetails = details, false
	return revision
}

// A sender that derives its key from what it says sends A's key again when it
// returns to A's wording. That is a new revision, not a retry of the first one.
func TestRevision_ReturningToEarlierTermsOfADirectAskIsAppliedNotReplayed(t *testing.T) {
	service, _ := testService(t)
	at(service, time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC))
	first := openDirectAsk(t, service)

	b := askSubject(t, service, detailsRevision("terms-b", "Run 42."))
	require.Equal(t, hub.AskUpdated, b.Outcome)
	require.Equal(t, "Run 42.", b.Ask.TechnicalDetails)

	a := askSubject(t, service, detailsRevision("terms-a", "Run 41."))
	require.Equal(t, hub.AskUpdated, a.Outcome)
	require.Equal(t, "Run 41.", a.Ask.TechnicalDetails)

	// A's key again: it is not the key of the last applied request (terms-a was).
	back := askSubject(t, service, detailsRevision("terms-b", "Run 42."))
	require.Equal(t, hub.AskUpdated, back.Outcome, "B's key was used before A; it is not a retry now")
	require.Equal(t, first.ID, back.Ask.ID)
	require.Equal(t, "Run 42.", back.Ask.TechnicalDetails)
	require.Equal(t, 3, back.Ask.RevisionCount)
}

func TestRevision_RetryingTheLatestRequestOfADirectAskChangesNothing(t *testing.T) {
	service, _ := testService(t)
	at(service, time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC))
	openDirectAsk(t, service)

	b := askSubject(t, service, detailsRevision("terms-b", "Run 42."))
	retry := askSubject(t, service, detailsRevision("terms-b", "Run 42."))
	require.Equal(t, hub.AskReasserted, retry.Outcome, "the retry of the latest request is recognised")
	require.Equal(t, b.Ask.RevisionCount, retry.Ask.RevisionCount)

	// Someone else moved the ask on; the retry of the request before it must not undo that.
	other := askSubject(t, service, detailsRevision("terms-c", "Run 43."))
	require.Equal(t, hub.AskUpdated, other.Outcome)
	late := askSubject(t, service, detailsRevision("terms-b", "Run 42."))
	require.Equal(t, hub.AskUpdated, late.Outcome, "a direct key that is no longer the latest applied request is a new revision")
	require.Equal(t, "Run 42.", late.Ask.TechnicalDetails)
}

func TestRevision_AKeyReusedForDifferentTermsStillConflicts(t *testing.T) {
	service, _ := testService(t)
	at(service, time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC))
	openDirectAsk(t, service)

	askSubject(t, service, detailsRevision("terms-b", "Run 42."))
	_, err := service.AskDecision(context.Background(), detailsRevision("terms-b", "Run 99."), "")
	require.ErrorIs(t, err, hub.ErrAskIdentityConflict, "the latest request's key with other terms is a client bug")
}

// An event is delivered at least once and in any order: every id the ask has
// carried keeps naming it, so a late delivery of an earlier revision changes nothing.
func TestRevision_AnEarlierEventOfAnEventSourcedAskIsStillAReplay(t *testing.T) {
	service, _ := testService(t)
	at(service, time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC))
	richAsk(t, service)

	one := partialSpec("evt-1")
	one.TechnicalDetails, one.Keep.TechnicalDetails = "Run 42.", false
	askSubject(t, service, one)
	two := partialSpec("evt-2")
	two.TechnicalDetails, two.Keep.TechnicalDetails = "Run 43.", false
	askSubject(t, service, two)

	late := askSubject(t, service, one)
	require.Equal(t, hub.AskReasserted, late.Outcome)
	require.Equal(t, "Run 43.", late.Ask.TechnicalDetails, "a redelivered earlier event does not roll the ask back")
}

// The operator can answer between an agent's last look and its revision. The
// revision is then refused, and the refusal says what became of the ask, so the
// agent reads the answer instead of guessing why nothing was open.
func TestRevision_ARevisionThatFindsTheAskAnsweredSaysSoAndChangesNothing(t *testing.T) {
	service, _ := testService(t)
	at(service, time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC))
	first := openDirectAsk(t, service)
	_, err := service.AnswerAsk(context.Background(), hub.AnswerInput{AskID: first.ID, Answer: "skip", Actor: first.Recipient})
	require.NoError(t, err)

	_, err = service.AskDecision(context.Background(), detailsRevision("late-revision", "Run 42."), "")
	require.ErrorIs(t, err, hub.ErrInvalidArgument)
	require.ErrorContains(t, err, `was answered "skip"`)
	require.ErrorContains(t, err, "conversations get --subject-key "+first.SubjectKey)
	require.ErrorContains(t, err, "your revision was not applied")

	got, err := service.GetAsk(context.Background(), first.ID)
	require.NoError(t, err)
	require.Equal(t, "Run 41.", got.TechnicalDetails, "the answered ask is untouched")
	require.Equal(t, hub.AskAnswered, got.State)
}

// A whole ask for a subject whose ask was answered long ago is a new ask: the
// answer was given to the old one.
func TestRevision_AWholeAskAfterTheAnswerOpensANewAsk(t *testing.T) {
	service, _ := testService(t)
	at(service, time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC))
	first := openDirectAsk(t, service)
	_, err := service.AnswerAsk(context.Background(), hub.AnswerInput{AskID: first.ID, Answer: "skip", Actor: first.Recipient})
	require.NoError(t, err)
	at(service, time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC))

	whole := subjectSpec(time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC), "whole-again")
	whole.SourceEventType = ""
	whole.TechnicalDetails = "Run 42, after the answer."
	got := askSubject(t, service, whole)
	require.Equal(t, hub.AskCreated, got.Outcome)
	require.NotEqual(t, first.ID, got.Ask.ID)
}
