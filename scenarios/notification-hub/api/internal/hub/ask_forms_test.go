package hub_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"google.golang.org/protobuf/types/known/structpb"

	hub "notification-hub/internal/hub"
	"notification-hub/internal/modules"
	"notification-hub/internal/testutil/hubtest"
)

var formNow = time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)

func layoutQuestions() []hub.AskQuestion {
	return []hub.AskQuestion{
		{
			ID: "layout", Type: "single_choice", Prompt: "Layout?", Required: true,
			Choices:     []hub.AskQuestionChoice{{Key: "cards", Label: "Card stack"}, {Key: "table", Label: "Table"}},
			Recommended: &hub.AskQuestionResponse{Selected: []string{"cards"}},
		},
		{
			ID: "plan", Type: "section_review", Prompt: "Plan sections",
			Sections: []hub.AskQuestionSection{{Key: "storage", Title: "Storage"}, {Key: "retention", Title: "Retention"}},
		},
		{ID: "why", Type: "text", Prompt: "Anything else?"},
	}
}

// formDecision is decisionSpec (options close and re-aim) with an answer form
// and no default, so the sweep never resolves it behind a test's back.
func formDecision(key string) hub.AskSpec {
	spec := decisionSpec(formNow, key)
	spec.DefaultAnswer, spec.Reversible = "", false
	spec.Questions = layoutQuestions()
	return spec
}

func openFormAsk(t *testing.T, service *hub.Service, key string) hub.AskRecord {
	t.Helper()
	result, err := service.AskDecision(context.Background(), formDecision(key), "")
	require.NoError(t, err)
	return result.Ask
}

func countRows(t *testing.T, db *database.RoutedDB, table string) int {
	t.Helper()
	var count int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM `+table).Scan(&count))
	return count
}

func TestAskForm_PersistsValidQuestionsAndReportsDroppedOnes(t *testing.T) {
	service, _ := testService(t)
	at(service, formNow)
	spec := formDecision("evt-form")
	spec.Questions = append(spec.Questions,
		hub.AskQuestion{ID: "bad id", Type: "text", Prompt: "Dropped?"},
		hub.AskQuestion{ID: "when", Type: "date_picker", Prompt: "When?"},
	)
	result, err := service.AskDecision(context.Background(), spec, "")
	require.NoError(t, err, "a broken question never fails the ask")

	got, err := service.GetAsk(context.Background(), result.Ask.ID)
	require.NoError(t, err)
	var ids []string
	for _, question := range got.Questions {
		ids = append(ids, question.ID)
	}
	require.Equal(t, []string{"layout", "plan", "why", "when"}, ids)
	require.Equal(t, "date_picker", got.Questions[3].Type, "an unknown type is kept and shown as free text")
	require.Equal(t, &hub.AskQuestionResponse{QuestionID: "layout", Selected: []string{"cards"}}, got.Questions[0].Recommended)

	byCode := map[string]hub.PacketWarning{}
	for _, warning := range result.Warnings {
		byCode[warning.Code] = warning
	}
	require.Equal(t, hub.WarningDropped, byCode["question_invalid"].Severity)
	require.Equal(t, "questions[3]", byCode["question_invalid"].Field)
	require.Contains(t, byCode["question_invalid"].Message, "question id")
	require.Equal(t, hub.WarningAdvice, byCode["question_type_unknown"].Severity)
	require.Equal(t, "questions[4]", byCode["question_type_unknown"].Field)
}

func TestAskForm_ARetryMustCarryTheSameQuestions(t *testing.T) {
	service, db := testService(t)
	at(service, formNow)
	first := openFormAsk(t, service, "evt-retry")
	again, err := service.AskDecision(context.Background(), formDecision("evt-retry"), "")
	require.NoError(t, err)
	require.Equal(t, first.ID, again.Ask.ID)
	require.Equal(t, 1, countAsks(t, db, `requester_ref = 'evt-retry'`))

	changed := formDecision("evt-retry")
	changed.Questions = changed.Questions[:1]
	_, err = service.AskDecision(context.Background(), changed, "")
	require.ErrorIs(t, err, hub.ErrAskIdentityConflict, "the same event id for a different form is a client bug")
}

func TestAskForm_ChangedQuestionsUpdateASubjectAskInPlaceWithoutANewPush(t *testing.T) {
	service, _ := testService(t)
	sender := &recordingSender{}
	service.SetPushSender(sender)
	registerSubscription(t, service, "alice")
	at(service, formNow)
	spec := subjectSpec(formNow, "evt-1")
	spec.Questions = layoutQuestions()
	first := askSubject(t, service, spec)
	waitDelivered(t, service, first.Ask)

	same := subjectSpec(formNow, "evt-1b")
	same.Questions = layoutQuestions()
	require.Equal(t, hub.AskReasserted, askSubject(t, service, same).Outcome, "an identical form changes nothing")

	later := formNow.Add(time.Hour)
	at(service, later)
	changed := subjectSpec(later, "evt-2")
	changed.Questions = layoutQuestions()
	changed.Questions[0].Prompt = "Which layout should the page use?"
	changed.Questions[0].Choices = append(changed.Questions[0].Choices, hub.AskQuestionChoice{Key: "grid", Label: "Grid"})
	updated := askSubject(t, service, changed)

	require.Equal(t, hub.AskUpdated, updated.Outcome, "a form change is a text change, not a new menu")
	require.Equal(t, first.Ask.ID, updated.Ask.ID)
	require.Equal(t, 1, updated.Ask.RevisionCount)
	require.Equal(t, "Which layout should the page use?", updated.Ask.Questions[0].Prompt)
	require.Len(t, updated.Ask.Questions[0].Choices, 3)
	require.Equal(t, 1, pushCount(sender), "no new push for a text-level change")

	// A revision that does not mention the form keeps it; clearing it is asked for
	// and is also an in-place update.
	at(service, later.Add(time.Minute))
	kept := askSubject(t, service, subjectSpec(later.Add(time.Minute), "evt-3"))
	require.Equal(t, hub.AskReasserted, kept.Outcome, "no form sent means the form stays, so nothing changed")
	require.Len(t, kept.Ask.Questions, 3)
	at(service, later.Add(2*time.Minute))
	clearing := subjectSpec(later.Add(2*time.Minute), "evt-4")
	clearing.ClearQuestions = true
	cleared := askSubject(t, service, clearing)
	require.Equal(t, hub.AskUpdated, cleared.Outcome)
	require.Empty(t, cleared.Ask.Questions)
}

func TestAnswerForm_ResponsesAreValidatedStoredAndReturnedToTheRequester(t *testing.T) {
	service, db := testService(t)
	publisher := &recordingPublisher{}
	service.SetResolutionPublisher(publisher)
	at(service, formNow)
	ask := openFormAsk(t, service, "evt-answer")
	ctx := context.Background()

	for name, responses := range map[string][]hub.AskQuestionResponse{
		"unknown question":    {{QuestionID: "layout", Selected: []string{"cards"}}, {QuestionID: "nope", Text: "x"}},
		"wrong shape":         {{QuestionID: "layout", Selected: []string{"cards"}}, {QuestionID: "why", Selected: []string{"cards"}}},
		"unknown choice":      {{QuestionID: "layout", Selected: []string{"grid"}}},
		"required unanswered": {{QuestionID: "why", Text: "because"}},
		"unknown verdict":     {{QuestionID: "layout", Selected: []string{"cards"}}, {QuestionID: "plan", Sections: []hub.AskSectionVerdict{{SectionKey: "storage", Verdict: "maybe"}}}},
	} {
		_, err := service.AnswerAsk(ctx, hub.AnswerInput{AskID: ask.ID, Answer: "close", Actor: "alice", Responses: responses})
		require.ErrorIs(t, err, hub.ErrInvalidArgument, name)
	}
	still, err := service.GetAsk(ctx, ask.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskPending, still.State, "a refused answer leaves the ask open")
	require.Zero(t, countRows(t, db, `answers`))

	answered, err := service.AnswerAsk(ctx, hub.AnswerInput{
		AskID: ask.ID, Answer: "close", Actor: "alice", Note: "Keep the retention period short.",
		Responses: []hub.AskQuestionResponse{
			{QuestionID: "plan", Sections: []hub.AskSectionVerdict{{SectionKey: "storage", Verdict: "approve"}, {SectionKey: "retention", Verdict: "change", Note: "keep 30 days, not 90"}}},
			{QuestionID: "layout", Selected: []string{"cards"}},
		},
	})
	require.NoError(t, err)
	const wantSummary = `Answer: Close the goal (close)
Note: Keep the retention period short.
Layout: Card stack (cards)
Plan sections:
  - Retention: change — keep 30 days, not 90
  - Other sections: approve`
	require.Equal(t, wantSummary, answered.AnswerSummary)
	require.Equal(t, []string{"layout", "plan"}, []string{answered.Responses[0].QuestionID, answered.Responses[1].QuestionID}, "responses come back in question order")

	read, err := service.GetAsk(ctx, ask.ID)
	require.NoError(t, err)
	require.Equal(t, answered.Responses, read.Responses)
	require.Equal(t, wantSummary, read.AnswerSummary)

	waited, err := service.Wait(ctx, ask.ID, hub.WaitOptions{})
	require.NoError(t, err)
	require.Equal(t, "close", waited.Ask.Answer)
	require.Equal(t, "Keep the retention period short.", waited.Ask.Note)
	require.Equal(t, answered.Responses, waited.Ask.Responses)
	require.Equal(t, wantSummary, waited.Ask.AnswerSummary)

	require.NoError(t, service.PublishPendingResolutions(ctx))
	require.Eventually(t, func() bool { return len(publisher.snapshot()) == 1 }, asyncWait, 10*time.Millisecond)
	payload := publisher.snapshot()[0].Payload
	_, err = structpb.NewStruct(payload)
	require.NoError(t, err, "the event payload must encode through the Vrooli Events serializer")
	require.Equal(t, wantSummary, payload["answer_summary"])
	require.Equal(t, hub.ResponsePayload(answered.Responses), payload["responses"])
	require.Equal(t, "close", payload["answer"], "existing payload fields are unchanged")
}

func TestAnswerForm_RequiredQuestionsBindOnlyForTheRequestersOptions(t *testing.T) {
	service, _ := testService(t)
	at(service, formNow)
	ctx := context.Background()

	other := openFormAsk(t, service, "evt-other")
	_, err := service.AnswerAsk(ctx, hub.AnswerInput{AskID: other.ID, Answer: "other", Actor: "alice"})
	require.ErrorIs(t, err, hub.ErrInvalidArgument, "Other still needs its note")
	answered, err := service.AnswerAsk(ctx, hub.AnswerInput{AskID: other.ID, Answer: "other", Actor: "alice", Note: "Neither layout; build a prototype."})
	require.NoError(t, err, "Other never needs the required form")
	require.Empty(t, answered.Responses)
	require.Equal(t, "Answer: Other (other)\nNote: Neither layout; build a prototype.", answered.AnswerSummary)

	chosen := openFormAsk(t, service, "evt-chosen")
	_, err = service.AnswerAsk(ctx, hub.AnswerInput{AskID: chosen.ID, Answer: "re-aim", Actor: "alice"})
	require.ErrorIs(t, err, hub.ErrInvalidArgument)
	require.Contains(t, err.Error(), `question "layout" is required`)
}

func TestAnswerForm_AnAskWithoutAFormRefusesResponses(t *testing.T) {
	service, _ := testService(t)
	at(service, formNow)
	ask, _, err := service.AskWithSpec(context.Background(), decisionSpec(formNow, "evt-bare"))
	require.NoError(t, err)
	_, err = service.AnswerAsk(context.Background(), hub.AnswerInput{AskID: ask.ID, Answer: "close", Actor: "alice", Responses: []hub.AskQuestionResponse{{QuestionID: "layout", Selected: []string{"cards"}}}})
	require.ErrorIs(t, err, hub.ErrInvalidArgument)
	answered, err := service.AnswerAsk(context.Background(), hub.AnswerInput{AskID: ask.ID, Answer: "close", Actor: "alice"})
	require.NoError(t, err)
	require.Equal(t, "Answer: Close the goal (close)", answered.AnswerSummary)
}

func TestAnswerForm_ALateAnswerAfterTheDefaultCarriesResponses(t *testing.T) {
	service, db := testService(t)
	at(service, formNow)
	spec := decisionSpec(formNow, "evt-late")
	spec.Questions = layoutQuestions()
	ask := askSubjectless(t, service, spec)
	_, err := db.ExecContext(context.Background(), `UPDATE asks SET state='defaulted', default_applied_at=?, resolved_at=? WHERE id=?`, formNow.Format(time.RFC3339Nano), formNow.Format(time.RFC3339Nano), ask.ID)
	require.NoError(t, err)
	defaulted, err := service.GetAsk(context.Background(), ask.ID)
	require.NoError(t, err)
	require.Empty(t, defaulted.AnswerSummary, "a default is not an operator answer")

	late, err := service.AnswerAsk(context.Background(), hub.AnswerInput{AskID: ask.ID, Answer: "close", Actor: "alice", Responses: []hub.AskQuestionResponse{{QuestionID: "layout", Selected: []string{"table"}}}})
	require.NoError(t, err)
	require.True(t, late.Late)
	require.Equal(t, []hub.AskQuestionResponse{{QuestionID: "layout", Selected: []string{"table"}}}, late.Responses)
}

func askSubjectless(t *testing.T, service *hub.Service, spec hub.AskSpec) hub.AskRecord {
	t.Helper()
	result, err := service.AskDecision(context.Background(), spec, "")
	require.NoError(t, err)
	return result.Ask
}

func TestPacketCheck_AdviceRidesOnTheAskResultAndNeverBlocksIt(t *testing.T) {
	service, _ := testService(t)
	at(service, formNow)
	spec := subjectSpec(formNow, "evt-advice")
	spec.Question = "[BAS] Retry S-14 using /home/matthew/notes/backup.md?"
	spec.RecommendationReason = ""
	spec.ContextURL = "http://localhost:3000/run/1"
	result := askSubject(t, service, spec)
	require.Equal(t, hub.AskCreated, result.Outcome, "advice never blocks")
	var codes []string
	for _, warning := range result.Warnings {
		codes = append(codes, warning.Code)
		require.Equal(t, hub.WarningAdvice, warning.Severity)
	}
	require.ElementsMatch(t, []string{"internal_id_in_question", "path_in_question", "missing_big_picture", "missing_recommendation_reason", "local_context_url"}, codes)

	good := subjectSpec(formNow, "evt-good")
	good.SubjectKey = "backup-failure:other"
	good.Question = "Should tonight's backup skip the Elements drive until it is plugged in?"
	good.BigPicture = "The drive has been unplugged since Tuesday, so tonight's backup would fail."
	require.Empty(t, askSubject(t, service, good).Warnings)
}

func TestPacketCheck_ReadsTheExplainerFromTheHubsOwnCopy(t *testing.T) {
	env := newAttachmentEnv(t)
	page := env.file(t, "explainer.html", `<!doctype html><html><body><img src="https://cdn.example.com/a.png"><img src="chart.png"><form></form></body></html>`)
	page.Role = hub.AttachmentRoleExplainer
	spec := subjectSpec(env.now, "evt-explainer")
	spec.BigPicture = "The drive has been unplugged since Tuesday."
	spec.Attachments = []hub.AskAttachment{page}
	result := askSubject(t, env.service, spec)
	var codes []string
	for _, warning := range result.Warnings {
		if warning.Field == "attachments[0]" {
			codes = append(codes, warning.Code)
		}
	}
	require.ElementsMatch(t, []string{"explainer_network", "explainer_no_viewport", "explainer_form", "explainer_unknown_attachment"}, codes)
}

func TestAnswerForm_QuestionsAndAttachmentNamesAreCrossChecked(t *testing.T) {
	env := newAttachmentEnv(t)
	spec := formDecision("evt-variants")
	spec.Questions = []hub.AskQuestion{{
		ID: "layout", Type: "variant", Prompt: "Which layout?",
		Choices: []hub.AskQuestionChoice{{Key: "cards", Attachment: "cards.png"}, {Key: "table", Attachment: "table.png"}},
	}}
	spec.Attachments = []hub.AskAttachment{env.file(t, "cards.png", "png bytes")}
	result, err := env.service.AskDecision(context.Background(), spec, "")
	require.NoError(t, err)
	require.Len(t, result.Ask.Questions, 1, "an unknown attachment name keeps the question")
	var unknown []hub.PacketWarning
	for _, warning := range result.Warnings {
		if warning.Code == "question_attachment_unknown" {
			unknown = append(unknown, warning)
		}
	}
	require.Len(t, unknown, 1)
	require.Equal(t, "questions[0].choices[1]", unknown[0].Field)
	require.Contains(t, unknown[0].Message, "table.png")
}

func TestAskForm_MigrationAddsTheFormColumnsToAnExistingDatabase(t *testing.T) {
	_, db := testService(t)
	ctx := context.Background()
	for _, stmt := range []string{
		`DROP TABLE ask_context_requests`,
		`ALTER TABLE asks DROP COLUMN questions_json`,
		`ALTER TABLE answers DROP COLUMN responses_json`,
	} {
		_, err := db.ExecContext(ctx, stmt)
		require.NoError(t, err, stmt)
	}

	// A restart on the deployed shape: schema.sql adds the table, the guarded
	// ALTERs add the columns, and a second restart changes nothing.
	for range 2 {
		require.NoError(t, database.EnsureSchemas(ctx, db.Primary(), modules.AllSchemas()...))
		hubtest.New(t, db, nil, nil)
	}
	for table, column := range map[string]string{"asks": "questions_json", "answers": "responses_json", "ask_context_requests": "published_at"} {
		var found int
		require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, table, column).Scan(&found))
		require.Equal(t, 1, found, table+"."+column)
	}

	restarted := hubtest.New(t, db, nil, nil)
	at(restarted, formNow)
	ask := openFormAsk(t, restarted, "evt-migrated")
	require.Len(t, ask.Questions, 3)
}

func TestAnswerForm_AttachmentNotesAreValidatedStoredAndReturnedToTheRequester(t *testing.T) {
	env := newAttachmentEnv(t)
	publisher := &recordingPublisher{}
	env.service.SetResolutionPublisher(publisher)
	ctx := context.Background()
	asked := env.ask(t, "evt-notes",
		env.file(t, "character.glb", "mesh"),
		env.folder(t, "shots", map[string]string{"day-1/intro.png": "png one", "day-2/outro.png": "png two"}))
	ask := asked.Ask

	for name, notes := range map[string][]hub.AskAttachmentNote{
		"unknown attachment": {{Attachment: "nope", Note: "x"}},
		"unknown entry":      {{Attachment: "shots", Path: "day-3/x.png", Note: "x"}},
		"path on a file":     {{Attachment: "character.glb", Path: "x", Note: "x"}},
	} {
		_, err := env.service.AnswerAsk(ctx, hub.AnswerInput{AskID: ask.ID, Answer: "retry", Actor: "alice", AttachmentNotes: notes})
		require.ErrorIs(t, err, hub.ErrInvalidArgument, name)
	}
	still, err := env.service.GetAsk(ctx, ask.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskPending, still.State, "a refused note leaves the ask open")

	notes := []hub.AskAttachmentNote{
		{Attachment: "character.glb", Note: "Hair is too glossy."},
		{Attachment: "shots", Path: "day-1/intro.png", Note: "Crop the top."},
	}
	answered, err := env.service.AnswerAsk(ctx, hub.AnswerInput{AskID: ask.ID, Answer: "retry", Actor: "alice", AttachmentNotes: notes})
	require.NoError(t, err)
	require.Equal(t, notes, answered.AttachmentNotes)
	const wantSummary = "Answer: Retry (retry)\nNotes on files:\n  - character.glb: Hair is too glossy.\n  - shots/day-1/intro.png: Crop the top."
	require.Equal(t, wantSummary, answered.AnswerSummary)

	waited, err := env.service.Wait(ctx, ask.ID, hub.WaitOptions{})
	require.NoError(t, err)
	require.Equal(t, notes, waited.Ask.AttachmentNotes)
	require.NoError(t, env.service.PublishPendingResolutions(ctx))
	require.Eventually(t, func() bool { return len(publisher.snapshot()) == 1 }, asyncWait, 10*time.Millisecond)
	payload := publisher.snapshot()[0].Payload
	_, err = structpb.NewStruct(payload)
	require.NoError(t, err)
	require.Equal(t, hub.AttachmentNotePayload(notes), payload["attachment_notes"])
	require.Equal(t, wantSummary, payload["answer_summary"])
}

func TestAnswerForm_AnAskWithoutAttachmentsRefusesNotes(t *testing.T) {
	service, _ := testService(t)
	at(service, formNow)
	ask := openFormAsk(t, service, "evt-no-attachments")
	_, err := service.AnswerAsk(context.Background(), hub.AnswerInput{AskID: ask.ID, Answer: "other", Note: "n", Actor: "alice", AttachmentNotes: []hub.AskAttachmentNote{{Attachment: "x.png", Note: "n"}}})
	require.ErrorIs(t, err, hub.ErrInvalidArgument)
}

func TestAnswerForm_AnAnswerCheckedAgainstARevisedFormIsRefusedNotStored(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	at(service, formNow)
	form := func(eventID, prompt string) hub.AskSpec {
		spec := subjectSpec(formNow, eventID)
		spec.Questions = []hub.AskQuestion{{ID: "when", Type: "text", Prompt: prompt}}
		return spec
	}
	first := askSubject(t, service, form("evt-1", "When?")).Ask
	answer := hub.AnswerInput{AskID: first.ID, Answer: "retry", Actor: "alice", Responses: []hub.AskQuestionResponse{{QuestionID: "when", Text: "tonight"}}}

	// The requester revises the form after the answer was validated and before it is stored.
	revisions := 0
	service.SetBeforeAnswerCommitForTest(func() {
		revisions++
		require.Equal(t, hub.AskUpdated, askSubject(t, service, form("evt-2", "At what time?")).Outcome)
	})
	_, err := service.AnswerAsk(ctx, answer)
	require.ErrorIs(t, err, hub.ErrAskChanged)
	require.Contains(t, err.Error(), "reload")
	require.NotErrorIs(t, err, hub.ErrAskResolved, "the ask is still open")
	current, err := service.GetAsk(ctx, first.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskPending, current.State)
	require.Equal(t, "At what time?", current.Questions[0].Prompt)
	require.Zero(t, countRows(t, db, `answers`), "responses are never stored against a form that is gone")

	// After a reload the same answer is accepted against the new revision.
	service.SetBeforeAnswerCommitForTest(nil)
	answered, err := service.AnswerAsk(ctx, answer)
	require.NoError(t, err)
	require.Equal(t, "Answer: Retry (retry)\nAt what time: tonight", answered.AnswerSummary)
	require.Equal(t, 1, revisions)
}

// droppedWarnings are the packet check's findings that a part was left out.
func droppedWarnings(result hub.AskResult) []hub.PacketWarning {
	var dropped []hub.PacketWarning
	for _, warning := range result.Warnings {
		if warning.Severity == hub.WarningDropped || strings.HasSuffix(warning.Code, "_removed") {
			dropped = append(dropped, warning)
		}
	}
	return dropped
}

func attachmentNames(list []hub.AskAttachment) []string {
	names := make([]string, 0, len(list))
	for _, attachment := range list {
		names = append(names, attachment.Name)
	}
	return names
}

func questionIDs(list []hub.AskQuestion) []string {
	ids := make([]string, 0, len(list))
	for _, question := range list {
		ids = append(ids, question.ID)
	}
	return ids
}

// A revision keeps what it does not mention: an agent that fixes the wording of
// an ask must not lose the operator's files, explainer or form by forgetting to
// send them again.
func TestRevision_KeepsTheAttachmentsExplainerAndFormItDoesNotMention(t *testing.T) {
	env := newAttachmentEnv(t)
	ctx := context.Background()
	page := env.file(t, "explainer.html", `<!doctype html><meta name="viewport" content="width=device-width"><p>hi</p>`)
	page.Role = hub.AttachmentRoleExplainer
	chart := env.file(t, "chart.png", "png")
	first := subjectSpec(env.now, "evt-1")
	first.Attachments = []hub.AskAttachment{page, chart}
	first.Questions = layoutQuestions()
	opened, err := env.service.AskDecision(ctx, first, "")
	require.NoError(t, err)

	bare := subjectSpec(env.now, "evt-2")
	bare.Question = "Backups failed twice since Tuesday. Retry or skip?"
	result, err := env.service.AskDecision(ctx, bare, "")
	require.NoError(t, err)
	require.Equal(t, hub.AskUpdated, result.Outcome)
	require.Equal(t, opened.Ask.ID, result.Ask.ID)
	require.Empty(t, droppedWarnings(result), "nothing was left out, so there is nothing to say")
	require.Equal(t, []string{"explainer.html", "chart.png"}, attachmentNames(env.attachments(t, result.Ask.ID)))
	require.Equal(t, []string{"layout", "plan", "why"}, questionIDs(result.Ask.Questions))

	// Sending the same text again is a re-assertion: the kept packet is the same.
	again := subjectSpec(env.now, "evt-3")
	again.Question = bare.Question
	result, err = env.service.AskDecision(ctx, again, "")
	require.NoError(t, err)
	require.Equal(t, hub.AskReasserted, result.Outcome)
}

func TestRevision_LaysWhatItSendsOverWhatTheAskHas(t *testing.T) {
	env := newAttachmentEnv(t)
	ctx := context.Background()
	page := env.file(t, "explainer.html", `<!doctype html><meta name="viewport" content="width=device-width"><p>one</p>`)
	page.Role = hub.AttachmentRoleExplainer
	chart := env.file(t, "chart.png", "png")
	first := subjectSpec(env.now, "evt-1")
	first.Attachments = []hub.AskAttachment{page, chart}
	first.Questions = layoutQuestions()
	_, err := env.service.AskDecision(ctx, first, "")
	require.NoError(t, err)

	newPage := env.file(t, "explainer-v2.html", `<!doctype html><meta name="viewport" content="width=device-width"><p>two</p>`)
	newPage.Role = hub.AttachmentRoleExplainer
	newChart := env.file(t, "chart.png", "png-new")
	extra := env.file(t, "extra.md", "notes")
	revised := subjectSpec(env.now, "evt-2")
	revised.Attachments = []hub.AskAttachment{newPage, newChart, extra}
	revised.Questions = []hub.AskQuestion{
		{ID: "plan", Type: "text", Prompt: "Plan changes?"},
		{ID: "when", Type: "text", Prompt: "When?"},
	}
	result, err := env.service.AskDecision(ctx, revised, "")
	require.NoError(t, err)
	require.Equal(t, hub.AskUpdated, result.Outcome)

	stored := env.attachments(t, result.Ask.ID)
	require.Equal(t, []string{"explainer-v2.html", "chart.png", "extra.md"}, attachmentNames(stored),
		"a sent explainer replaces the explainer, a sent file replaces the file of its name, and a new file is added")
	require.Equal(t, sha("png-new"), stored[1].SHA256)
	require.Equal(t, []string{"layout", "plan", "why", "when"}, questionIDs(result.Ask.Questions),
		"a sent question takes the place of the one with its id and a new one goes last")
	require.Equal(t, "Plan changes?", result.Ask.Questions[1].Prompt)
}

func TestRevision_ClearsWhatItIsToldToClear(t *testing.T) {
	env := newAttachmentEnv(t)
	ctx := context.Background()
	chart := env.file(t, "chart.png", "png")
	first := subjectSpec(env.now, "evt-1")
	first.Attachments = []hub.AskAttachment{chart}
	first.Questions = layoutQuestions()
	_, err := env.service.AskDecision(ctx, first, "")
	require.NoError(t, err)

	replacement := env.file(t, "table.png", "png2")
	cleared := subjectSpec(env.now, "evt-2")
	cleared.Attachments = []hub.AskAttachment{replacement}
	cleared.ClearAttachments, cleared.ClearQuestions = true, true
	result, err := env.service.AskDecision(ctx, cleared, "")
	require.NoError(t, err)
	require.Equal(t, hub.AskUpdated, result.Outcome)
	require.Equal(t, []string{"table.png"}, attachmentNames(env.attachments(t, result.Ask.ID)))
	require.Empty(t, result.Ask.Questions)
	require.Empty(t, droppedWarnings(result), "clearing is asked for, not advised against")
}

func TestRevision_ANewMenuCarriesThePacketToTheReplacingAsk(t *testing.T) {
	env := newAttachmentEnv(t)
	ctx := context.Background()
	chart := env.file(t, "chart.png", "png")
	first := subjectSpec(env.now, "evt-1")
	first.Attachments = []hub.AskAttachment{chart}
	first.Questions = layoutQuestions()
	opened, err := env.service.AskDecision(ctx, first, "")
	require.NoError(t, err)

	menu := subjectSpec(env.now, "evt-2")
	menu.Options = append(menu.Options, hub.AskOption{Key: "wait", Label: "Wait a day"})
	result, err := env.service.AskDecision(ctx, menu, "")
	require.NoError(t, err)
	require.Equal(t, hub.AskReplaced, result.Outcome)
	require.NotEqual(t, opened.Ask.ID, result.Ask.ID)
	require.Equal(t, []string{"chart.png"}, attachmentNames(env.attachments(t, result.Ask.ID)))
	require.Equal(t, []string{"layout", "plan", "why"}, questionIDs(result.Ask.Questions))
}

func TestRevision_AFirstAskAndAnAskAfterADismissalCarryOnlyWhatTheyBring(t *testing.T) {
	env := newAttachmentEnv(t)
	ctx := context.Background()
	chart := env.file(t, "chart.png", "png")
	first := subjectSpec(env.now, "evt-1")
	first.Attachments = []hub.AskAttachment{chart}
	opened, err := env.service.AskDecision(ctx, first, "")
	require.NoError(t, err)
	_, err = env.service.DismissAsk(ctx, opened.Ask.ID, "alice", "no")
	require.NoError(t, err)

	later := subjectSpec(env.now, "evt-2")
	later.Question = "Backups are still blocked. Retry or skip?"
	result, err := env.service.AskDecision(ctx, later, "")
	require.NoError(t, err)
	require.Equal(t, hub.AskCreated, result.Outcome)
	require.Empty(t, env.attachments(t, result.Ask.ID), "a closed ask's packet is not carried into a new ask")
}
