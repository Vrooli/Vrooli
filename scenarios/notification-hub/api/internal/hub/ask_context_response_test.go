package hub_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	hub "notification-hub/internal/hub"
)

// The requester answers the operator's "I need more context" question with a
// written response. It is stored with the question it answers, not with the
// ask's terms, and the questions stay as the history of what was asked and
// answered.

func respond(t *testing.T, service *hub.Service, key string, change func(*hub.AskSpec)) hub.AskResult {
	t.Helper()
	spec := partialSpec(key)
	spec.Title, spec.Correlation = "Decision needed: Backups", map[string]string{"destination": "elements", "failures": "1"}
	change(&spec)
	result, err := service.AskDecision(context.Background(), spec, "")
	require.NoError(t, err)
	return result
}

func withResponse(text string) func(*hub.AskSpec) {
	return func(spec *hub.AskSpec) { spec.ContextResponse = text }
}

func TestContextResponse_AnswersTheQuestionAndChangesNothingElse(t *testing.T) {
	service, _ := testService(t)
	at(service, formNow)
	ask := richAsk(t, service)
	ctx := context.Background()
	request, err := service.RequestContext(ctx, ask.ID, "alice", "Which drive?")
	require.NoError(t, err)

	at(service, formNow.Add(time.Minute))
	got := respond(t, service, "evt-1", withResponse("  The Elements drive, mounted at /mnt/elements.  "))

	require.Equal(t, hub.AskUpdated, got.Outcome, "the ask's packet changed: the question has its answer")
	require.Equal(t, ask.ID, got.Ask.ID)
	require.Zero(t, got.Ask.OpenContextRequestCount)
	require.Equal(t, ask.Question, got.Ask.Question)
	require.Equal(t, ask.BigPicture, got.Ask.BigPicture, "the response is not written into the ask's text")
	require.Equal(t, ask.RevisionCount, got.Ask.RevisionCount, "no term changed, so there is no new revision to show")
	require.Empty(t, withoutPageAdvice(got.Warnings))

	listed, err := service.ListContextRequests(ctx, ask.ID)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, request.ID, listed[0].ID)
	require.Equal(t, "The Elements drive, mounted at /mnt/elements.", listed[0].Response)
	require.Equal(t, hub.StoredTime(formNow.Add(time.Minute)), listed[0].AddressedAt)
	require.Equal(t, ask.RevisionCount, listed[0].AddressedRevision, "it answers the ask as the operator sees it")
}

func TestContextResponse_WakesAWaiterThatReturnedForTheQuestion(t *testing.T) {
	service, _ := testService(t)
	at(service, formNow)
	ask := richAsk(t, service)
	ctx := context.Background()
	_, err := service.RequestContext(ctx, ask.ID, "alice", "Which drive?")
	require.NoError(t, err)
	waited, err := service.Wait(ctx, ask.ID, hub.WaitOptions{Deadline: formNow, ReturnOnContextRequest: true})
	require.NoError(t, err)
	require.NotNil(t, waited.ContextRequest)

	respond(t, service, "evt-1", withResponse("The Elements drive."))
	waited, err = service.Wait(ctx, ask.ID, hub.WaitOptions{Deadline: formNow, ReturnOnContextRequest: true})
	require.NoError(t, err)
	require.Nil(t, waited.ContextRequest, "the answered question no longer ends the wait")
}

func TestContextResponse_TwoRoundsOfQuestionAndAnswerAreBothKept(t *testing.T) {
	service, _ := testService(t)
	at(service, formNow)
	ask := richAsk(t, service)
	ctx := context.Background()

	first, err := service.RequestContext(ctx, ask.ID, "alice", "Which drive?")
	require.NoError(t, err)
	at(service, formNow.Add(time.Hour))
	respond(t, service, "evt-1", withResponse("The Elements drive."))

	second, err := service.RequestContext(ctx, ask.ID, "alice", "Since when?")
	require.NoError(t, err)
	at(service, formNow.Add(2*time.Hour))
	// The second answer comes with a change to the ask itself.
	got := respond(t, service, "evt-2", func(spec *hub.AskSpec) {
		spec.ContextResponse = "Since Tuesday 02:00."
		spec.TechnicalDetails, spec.Keep.TechnicalDetails = "Run 44; failing since Tuesday.", false
	})
	require.Equal(t, hub.AskUpdated, got.Outcome)
	require.Equal(t, ask.RevisionCount+1, got.Ask.RevisionCount)

	listed, err := service.ListContextRequests(ctx, ask.ID)
	require.NoError(t, err)
	require.Len(t, listed, 2)
	require.Equal(t, []string{first.ID, second.ID}, []string{listed[0].ID, listed[1].ID}, "oldest first")
	require.Equal(t, "The Elements drive.", listed[0].Response)
	require.Equal(t, "Since Tuesday 02:00.", listed[1].Response)
	require.Equal(t, hub.StoredTime(formNow.Add(time.Hour)), listed[0].AddressedAt)
	require.Equal(t, hub.StoredTime(formNow.Add(2*time.Hour)), listed[1].AddressedAt)
	require.Equal(t, got.Ask.RevisionCount, listed[1].AddressedRevision)
	require.Equal(t, "Run 44; failing since Tuesday.", got.Ask.TechnicalDetails)
}

func TestContextResponse_AnswersEveryOpenQuestionAtOnce(t *testing.T) {
	service, _ := testService(t)
	at(service, formNow)
	ask := richAsk(t, service)
	ctx := context.Background()
	for _, question := range []string{"Which drive?", "Since when?"} {
		_, err := service.RequestContext(ctx, ask.ID, "alice", question)
		require.NoError(t, err)
	}

	respond(t, service, "evt-1", withResponse("The Elements drive, since Tuesday."))
	listed, err := service.ListContextRequests(ctx, ask.ID)
	require.NoError(t, err)
	require.Len(t, listed, 2)
	for _, request := range listed {
		require.Equal(t, "The Elements drive, since Tuesday.", request.Response, "one answer covers the questions open when it was sent")
	}
}

func TestContextResponse_ARevisionWithoutOneStillAddressesTheQuestionWithNoAnswer(t *testing.T) {
	service, _ := testService(t)
	at(service, formNow)
	ask := richAsk(t, service)
	ctx := context.Background()
	_, err := service.RequestContext(ctx, ask.ID, "alice", "Which drive?")
	require.NoError(t, err)

	respond(t, service, "evt-1", func(spec *hub.AskSpec) {
		spec.BigPicture, spec.Keep.BigPicture = "Backups to the Elements drive stopped at 02:00.", false
		spec.TechnicalDetails, spec.Keep.TechnicalDetails = "Run 44.", false
	})
	listed, err := service.ListContextRequests(ctx, ask.ID)
	require.NoError(t, err)
	require.True(t, listed[0].Addressed())
	require.Empty(t, listed[0].Response)
}

func TestContextResponse_WithNoOpenQuestionIsDroppedWithAWarningAndTheRestApplies(t *testing.T) {
	service, db := testService(t)
	at(service, formNow)
	richAsk(t, service)

	got := respond(t, service, "evt-1", func(spec *hub.AskSpec) {
		spec.ContextResponse = "Nobody asked, but here is more."
		spec.TechnicalDetails, spec.Keep.TechnicalDetails = "Run 44.", false
	})
	require.Equal(t, hub.AskUpdated, got.Outcome, "the rest of the request is applied")
	require.Equal(t, "Run 44.", got.Ask.TechnicalDetails)
	var codes []string
	for _, warning := range got.Warnings {
		codes = append(codes, warning.Code)
		if warning.Code == hub.WarnContextResponseUnused {
			require.Equal(t, hub.WarningDropped, warning.Severity)
			require.Equal(t, "context_response", warning.Field)
		}
	}
	require.Contains(t, codes, hub.WarnContextResponseUnused)
	require.Zero(t, countRows(t, db, `ask_context_requests`))
}

func TestContextResponse_OnItsOwnWithNoQuestionChangesNothing(t *testing.T) {
	service, _ := testService(t)
	at(service, formNow)
	ask := richAsk(t, service)

	got := respond(t, service, "evt-1", withResponse("Nobody asked."))
	require.Equal(t, hub.AskReasserted, got.Outcome)
	require.Equal(t, ask.RevisionCount, got.Ask.RevisionCount)
	require.Contains(t, warningCodes(got.Warnings), hub.WarnContextResponseUnused)
}

func TestContextResponse_RetryingTheSameRequestIsNotAppliedTwice(t *testing.T) {
	service, _ := testService(t)
	at(service, formNow)
	ask := richAsk(t, service)
	ctx := context.Background()
	_, err := service.RequestContext(ctx, ask.ID, "alice", "Which drive?")
	require.NoError(t, err)

	respond(t, service, "evt-1", withResponse("The Elements drive."))
	again := respond(t, service, "evt-1", withResponse("The Elements drive."))
	require.Equal(t, hub.AskReasserted, again.Outcome, "the retry finds the request already applied")
	require.NotContains(t, warningCodes(again.Warnings), hub.WarnContextResponseUnused, "a replay is not judged against the ask as it is now")
	listed, err := service.ListContextRequests(ctx, ask.ID)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, "The Elements drive.", listed[0].Response)
}

func TestContextResponse_IsBoundedInCharacters(t *testing.T) {
	service, _ := testService(t)
	at(service, formNow)
	ask := richAsk(t, service)
	ctx := context.Background()
	_, err := service.RequestContext(ctx, ask.ID, "alice", "Which drive?")
	require.NoError(t, err)

	spec := partialSpec("evt-1")
	spec.ContextResponse = strings.Repeat("é", 4001)
	_, err = service.AskDecision(ctx, spec, "")
	require.ErrorIs(t, err, hub.ErrInvalidArgument)
	require.Contains(t, err.Error(), "context_response")
	spec.ContextResponse = strings.Repeat("é", 4000)
	_, err = service.AskDecision(ctx, spec, "")
	require.NoError(t, err, "4000 characters is the limit, counted in characters")
}

func TestContextResponse_AReplacementAnswersTheQuestionAndCarriesTheHistory(t *testing.T) {
	service, _ := testService(t)
	at(service, formNow)
	ask := richAsk(t, service)
	ctx := context.Background()
	_, err := service.RequestContext(ctx, ask.ID, "alice", "What would waiting do?")
	require.NoError(t, err)

	at(service, formNow.Add(time.Hour))
	replaced := respond(t, service, "evt-1", func(spec *hub.AskSpec) {
		spec.ContextResponse = "Nothing is lost for a day, so I added a wait option."
		spec.Options, spec.Keep.Options = []hub.AskOption{{Key: "retry", Label: "Retry"}, {Key: "skip", Label: "Skip"}, {Key: "wait", Label: "Wait a day"}}, false
	})
	require.Equal(t, hub.AskReplaced, replaced.Outcome)
	require.NotEqual(t, ask.ID, replaced.Ask.ID)
	require.Zero(t, replaced.Ask.OpenContextRequestCount)

	for _, id := range []string{ask.ID, replaced.Ask.ID} {
		listed, err := service.ListContextRequests(ctx, id)
		require.NoError(t, err)
		require.Len(t, listed, 1, "the question is on the closed ask and on the ask the operator now sees")
		require.Equal(t, "What would waiting do?", listed[0].Question)
		require.Equal(t, "Nothing is lost for a day, so I added a wait option.", listed[0].Response)
		require.True(t, listed[0].Addressed())
	}
}

func TestContextResponse_ACarriedQuestionIsAnnouncedToTheRequesterOnce(t *testing.T) {
	service, db := testService(t)
	at(service, formNow)
	ask := richAsk(t, service)
	ctx := context.Background()
	_, err := service.RequestContext(ctx, ask.ID, "alice", "What would waiting do?")
	require.NoError(t, err)
	respond(t, service, "evt-1", func(spec *hub.AskSpec) {
		spec.Options, spec.Keep.Options = []hub.AskOption{{Key: "retry", Label: "Retry"}, {Key: "wait", Label: "Wait"}}, false
	})
	require.Equal(t, 1, countRows(t, db, `ask_context_requests WHERE published_at = ''`), "only the request the requester has not been told of is waiting in the outbox")
}

func TestRevision_RemovingAFormQuestionByIdKeepsTheOthers(t *testing.T) {
	service, _ := testService(t)
	at(service, formNow)
	richAsk(t, service)
	form := []hub.AskQuestion{
		{ID: "layout", Type: "text", Prompt: "Which layout?"},
		{ID: "owner", Type: "text", Prompt: "Who owns it?"},
		{ID: "when", Type: "text", Prompt: "When?"},
	}
	respond(t, service, "evt-1", func(spec *hub.AskSpec) { spec.Questions = form })

	got := respond(t, service, "evt-2", func(spec *hub.AskSpec) { spec.RemoveQuestions = []string{"owner"} })
	require.Equal(t, hub.AskUpdated, got.Outcome)
	require.Equal(t, []string{"layout", "when"}, questionIDs(got.Ask.Questions))
	require.Empty(t, withoutPageAdvice(got.Warnings))
}

func TestRevision_RemovingAndSendingTheSameQuestionIdReplacesIt(t *testing.T) {
	service, _ := testService(t)
	at(service, formNow)
	richAsk(t, service)
	respond(t, service, "evt-1", func(spec *hub.AskSpec) {
		spec.Questions = []hub.AskQuestion{{ID: "layout", Type: "text", Prompt: "Which layout?"}, {ID: "owner", Type: "text", Prompt: "Who owns it?"}}
	})

	got := respond(t, service, "evt-2", func(spec *hub.AskSpec) {
		spec.RemoveQuestions = []string{"owner"}
		spec.Questions = []hub.AskQuestion{{ID: "owner", Type: "text", Prompt: "Which team owns it?"}}
	})
	require.Equal(t, []string{"layout", "owner"}, questionIDs(got.Ask.Questions))
	require.Equal(t, "Which team owns it?", got.Ask.Questions[1].Prompt)
}

func TestRevision_RemovingAnIdTheAskDoesNotHaveSaysSoAndChangesNothing(t *testing.T) {
	service, _ := testService(t)
	at(service, formNow)
	richAsk(t, service)
	respond(t, service, "evt-1", func(spec *hub.AskSpec) {
		spec.Questions = []hub.AskQuestion{{ID: "layout", Type: "text", Prompt: "Which layout?"}}
	})

	got := respond(t, service, "evt-2", func(spec *hub.AskSpec) { spec.RemoveQuestions = []string{"layuot"} })
	require.Equal(t, hub.AskReasserted, got.Outcome)
	require.Equal(t, []string{"layout"}, questionIDs(got.Ask.Questions))
	require.Contains(t, warningCodes(got.Warnings), hub.WarnQuestionRemoveUnknown)
}

func TestRevision_ClearingQuestionsStillDropsEveryOne(t *testing.T) {
	service, _ := testService(t)
	at(service, formNow)
	richAsk(t, service)
	respond(t, service, "evt-1", func(spec *hub.AskSpec) {
		spec.Questions = []hub.AskQuestion{{ID: "layout", Type: "text", Prompt: "Which layout?"}, {ID: "owner", Type: "text", Prompt: "Who owns it?"}}
	})
	got := respond(t, service, "evt-2", func(spec *hub.AskSpec) { spec.ClearQuestions = true })
	require.Empty(t, questionIDs(got.Ask.Questions))
}
