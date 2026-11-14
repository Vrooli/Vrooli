package hub_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	hub "notification-hub/internal/hub"
)

// What a revision does to the packet: removing one attachment, a review that
// follows a file, a dry run that uploads nothing, and the packet check that does
// not repeat itself.

// revise sends a partial revision of the subject's ask and returns its result.
func (e *attachmentEnv) revise(t *testing.T, key string, change func(*hub.AskSpec)) hub.AskResult {
	t.Helper()
	spec := partialSpec(key)
	// An event producer says its own title and correlation on every send, as subjectSpec does.
	spec.Title, spec.Correlation = "Decision needed: Backups", map[string]string{"destination": "elements", "failures": "1"}
	change(&spec)
	result, err := e.service.AskDecision(context.Background(), spec, "")
	require.NoError(t, err)
	return result
}

func withAttachments(attachments ...hub.AskAttachment) func(*hub.AskSpec) {
	return func(spec *hub.AskSpec) { spec.Attachments = attachments }
}

const planV1 = "# Release plan\n\n## Storage\nMove it.\n\n## Retention\nKeep 90 days.\n"

func followingPlan() []hub.AskQuestion {
	return []hub.AskQuestion{{ID: "plan", Type: "section_review", Prompt: "Review each section.", SectionsFrom: "plan.md"}}
}

func sectionKeys(question hub.AskQuestion) []string {
	var keys []string
	for _, section := range question.Sections {
		keys = append(keys, section.Key)
	}
	return keys
}

func TestRevision_RemovingOneAttachmentByNameKeepsTheRest(t *testing.T) {
	env := newAttachmentEnv(t)
	first := env.ask(t, "evt-0", env.file(t, "plan.md", planV1), env.file(t, "notes.md", "notes"),
		hub.AskAttachment{Name: "explainer.html", Kind: hub.AttachmentFile, Role: hub.AttachmentRoleExplainer, SHA256: env.put(t, "<html></html>")})
	env.advance(t, time.Minute)

	removed := env.revise(t, "evt-1", func(spec *hub.AskSpec) { spec.RemoveAttachments = []string{"notes.md"} })
	require.Equal(t, hub.AskUpdated, removed.Outcome)
	require.ElementsMatch(t, []string{"explainer.html", "plan.md"}, names(env.attachments(t, first.Ask.ID)), "only the named attachment went")

	env.advance(t, time.Minute)
	dropped := env.revise(t, "evt-2", func(spec *hub.AskSpec) { spec.RemoveAttachments = []string{"explainer.html"} })
	require.Equal(t, hub.AskUpdated, dropped.Outcome)
	require.Equal(t, []string{"plan.md"}, names(env.attachments(t, first.Ask.ID)), "the explainer is removed by its name too")
}

func TestRevision_RemovingAndSendingTheSameNameReplacesIt(t *testing.T) {
	env := newAttachmentEnv(t)
	first := env.ask(t, "evt-0", env.file(t, "plan.md", planV1), env.file(t, "notes.md", "notes"))
	env.advance(t, time.Minute)

	env.revise(t, "evt-1", func(spec *hub.AskSpec) {
		spec.RemoveAttachments = []string{"plan.md"}
		spec.Attachments = []hub.AskAttachment{env.file(t, "plan.md", "plan v2")}
	})
	list := env.attachments(t, first.Ask.ID)
	require.ElementsMatch(t, []string{"plan.md", "notes.md"}, names(list))
	for _, attachment := range list {
		if attachment.Name == "plan.md" {
			require.Equal(t, sha("plan v2"), attachment.SHA256)
		}
	}
}

func TestRevision_RemovingANameTheAskDoesNotHaveSaysSoAndChangesNothing(t *testing.T) {
	env := newAttachmentEnv(t)
	first := env.ask(t, "evt-0", env.file(t, "plan.md", planV1))
	env.advance(t, time.Minute)

	got := env.revise(t, "evt-1", func(spec *hub.AskSpec) { spec.RemoveAttachments = []string{"plan.mdd"} })
	require.Equal(t, hub.AskReasserted, got.Outcome, "nothing changed")
	warning := findWarning(t, got.Warnings, hub.WarnAttachmentRemoveUnknown)
	require.Contains(t, warning.Message, `"plan.mdd"`)
	require.Equal(t, []string{"plan.md"}, names(env.attachments(t, first.Ask.ID)))
}

func TestRevision_AReviewThatFollowsAFileTakesItsHeadingsAsSections(t *testing.T) {
	env := newAttachmentEnv(t)
	spec := subjectSpec(env.now, "evt-0")
	spec.Attachments = []hub.AskAttachment{env.file(t, "plan.md", planV1)}
	spec.Questions = followingPlan()
	first, err := env.service.AskDecision(context.Background(), spec, "")
	require.NoError(t, err)

	require.Equal(t, []string{"storage", "retention"}, sectionKeys(first.Ask.Questions[0]))
	require.Equal(t, "plan.md", first.Ask.Questions[0].Sections[0].Attachment)
	require.Equal(t, "plan.md", first.Ask.Questions[0].SectionsFrom)
}

// The operator reviews the plan they can open. A replaced plan must change the
// review with it, though the revision never mentions the form.
func TestRevision_ReplacingTheFileAReviewFollowsChangesTheReview(t *testing.T) {
	env := newAttachmentEnv(t)
	spec := subjectSpec(env.now, "evt-0")
	spec.Attachments = []hub.AskAttachment{env.file(t, "plan.md", planV1)}
	spec.Questions = append(followingPlan(), hub.AskQuestion{ID: "why", Type: "text", Prompt: "Anything else?"})
	first, err := env.service.AskDecision(context.Background(), spec, "")
	require.NoError(t, err)
	env.advance(t, time.Minute)

	revised := env.revise(t, "evt-1", withAttachments(env.file(t, "plan.md", "# Release plan\n\n## Storage\nMove it.\n\n## Rollout\nStaged.\n\n## Retention\nKeep 30 days.\n")))
	require.Equal(t, hub.AskUpdated, revised.Outcome)
	got, err := env.service.GetAsk(context.Background(), first.Ask.ID)
	require.NoError(t, err)
	require.Len(t, got.Questions, 2, "the form stays")
	require.Equal(t, []string{"storage", "rollout", "retention"}, sectionKeys(got.Questions[0]), "the review follows the new plan")
	require.Equal(t, "plan.md", got.Questions[0].SectionsFrom)

	// A revision that touches neither the plan nor the form leaves the sections alone.
	env.advance(t, time.Minute)
	env.revise(t, "evt-2", func(spec *hub.AskSpec) { spec.TechnicalDetails, spec.Keep.TechnicalDetails = "Run 2.", false })
	got, err = env.service.GetAsk(context.Background(), first.Ask.ID)
	require.NoError(t, err)
	require.Equal(t, []string{"storage", "rollout", "retention"}, sectionKeys(got.Questions[0]))
}

func TestRevision_AReviewWhoseFileLostItsHeadingsOrWasRemovedIsLeftOutAndSaysSo(t *testing.T) {
	env := newAttachmentEnv(t)
	spec := subjectSpec(env.now, "evt-0")
	spec.Attachments = []hub.AskAttachment{env.file(t, "plan.md", planV1)}
	spec.Questions = append(followingPlan(), hub.AskQuestion{ID: "why", Type: "text", Prompt: "Anything else?"})
	first, err := env.service.AskDecision(context.Background(), spec, "")
	require.NoError(t, err)
	env.advance(t, time.Minute)

	flat := env.revise(t, "evt-1", withAttachments(env.file(t, "plan.md", "Just prose, no headings.")))
	warning := findWarning(t, flat.Warnings, hub.WarnQuestionInvalid)
	require.Equal(t, hub.WarningDropped, warning.Severity)
	require.Contains(t, warning.Message, "no ## or ### headings")
	got, err := env.service.GetAsk(context.Background(), first.Ask.ID)
	require.NoError(t, err)
	require.Len(t, got.Questions, 1, "the review that cannot follow its file is gone; the rest of the form stays")
	require.Equal(t, "why", got.Questions[0].ID)

	missing := subjectSpec(env.now, "evt-9")
	missing.SubjectKey = "other-subject"
	missing.Questions = followingPlan()
	result, err := env.service.AskDecision(context.Background(), missing, "")
	require.NoError(t, err, "a review that follows an unattached file never fails the ask")
	require.Empty(t, result.Ask.Questions)
	require.Contains(t, findWarning(t, result.Warnings, hub.WarnQuestionInvalid).Message, "is not attached")
}

func TestRevision_AReviewCanFollowAFileInsideAFolder(t *testing.T) {
	env := newAttachmentEnv(t)
	spec := subjectSpec(env.now, "evt-0")
	spec.Attachments = []hub.AskAttachment{env.folder(t, "docs", map[string]string{"plan.md": planV1, "other.md": "# Other\n"})}
	spec.Questions = []hub.AskQuestion{{ID: "plan", Type: "section_review", Prompt: "Review each section.", SectionsFrom: "docs/plan.md"}}
	result, err := env.service.AskDecision(context.Background(), spec, "")
	require.NoError(t, err)

	require.Equal(t, []string{"storage", "retention"}, sectionKeys(result.Ask.Questions[0]))
	require.Equal(t, "docs/plan.md", result.Ask.Questions[0].Sections[0].Attachment)
	for _, warning := range result.Warnings {
		require.NotEqual(t, hub.WarnQuestionAttachmentUnknown, warning.Code, "a file of an attached folder is attached")
	}
}

func TestCheckAsk_ADryRunUploadsNothingAndReadsWhatItIsGiven(t *testing.T) {
	env := newAttachmentEnv(t)
	page := `<!doctype html><html><head><meta name="viewport" content="width=device-width"></head><body><img src="https://cdn.example.com/x.png"></body></html>`
	spec := subjectSpec(env.now, "evt-dry")
	spec.Attachments = []hub.AskAttachment{
		{Name: "plan.md", Kind: hub.AttachmentFile, SHA256: sha(planV1), SizeBytes: int64(len(planV1))},
		{Name: "explainer.html", Kind: hub.AttachmentFile, Role: hub.AttachmentRoleExplainer, SHA256: sha(page), SizeBytes: int64(len(page))},
	}
	spec.Questions = followingPlan()

	check, err := env.service.CheckAsk(context.Background(), spec, hub.CheckInput{Contents: map[string][]byte{sha(planV1): []byte(planV1), sha(page): []byte(page)}})
	require.NoError(t, err)
	codes := warningCodes(check.Warnings)
	require.NotContains(t, codes, "attachment_blob_missing", "a file the dry run did not upload is taken as stored")
	require.Contains(t, codes, "explainer_network", "the check read the explainer from what it was given")
	require.NotContains(t, codes, "question_invalid", "the review followed the plan it was given")
	require.Zero(t, env.blobCount(t), "a dry run stores no blob")
	require.Zero(t, countAsks(t, env.db, "1 = 1"), "and no ask")
}

func TestCheckAsk_ARevisionPreviewsTheMergedAskAndMarksWhatIsCarriedOver(t *testing.T) {
	env := newAttachmentEnv(t)
	first := env.ask(t, "evt-0", env.file(t, "plan.md", planV1))
	require.Contains(t, warningCodes(first.Warnings), "missing_big_picture")
	env.advance(t, time.Minute)

	revision := partialSpec("evt-1")
	revision.TechnicalDetails, revision.Keep.TechnicalDetails = "Run 42.", false
	check, err := env.service.CheckAsk(context.Background(), revision, hub.CheckInput{})
	require.NoError(t, err)
	require.Equal(t, first.Ask.ID, check.RevisesAskID)
	require.Contains(t, check.PushBody, first.Ask.Question, "the preview carries the kept question")
	require.True(t, findWarning(t, check.Warnings, "missing_big_picture").CarriedOver, "the open ask already lacked a big picture, and this revision did not touch it")
}

func TestRevision_AFindingIsCarriedOverOnlyWhenTheRevisionLeftItsPartAlone(t *testing.T) {
	env := newAttachmentEnv(t)
	spec := subjectSpec(env.now, "evt-0")
	spec.Question = "[BAS] Retry S-14 tonight?"
	spec.RecommendationReason = ""
	first, err := env.service.AskDecision(context.Background(), spec, "")
	require.NoError(t, err)
	for _, warning := range first.Warnings {
		require.False(t, warning.CarriedOver, "a first send has nothing to carry over: %s", warning.Code)
	}
	env.advance(t, time.Minute)

	// Revising the details repeats every finding; none of them is news.
	details := env.revise(t, "evt-1", func(spec *hub.AskSpec) { spec.TechnicalDetails, spec.Keep.TechnicalDetails = "Run 42.", false })
	require.Equal(t, hub.AskUpdated, details.Outcome)
	require.NotEmpty(t, details.Warnings)
	for _, warning := range details.Warnings {
		require.True(t, warning.CarriedOver, "%s was true before and the revision did not touch it", warning.Code)
	}

	// Rewriting the question with the same fault says it again in full; the
	// findings about the parts left alone stay carried over.
	env.advance(t, time.Minute)
	question := env.revise(t, "evt-2", func(spec *hub.AskSpec) {
		spec.Question, spec.Keep.Question = "[BAS] Retry S-14 tonight, or skip it?", false
	})
	byCode := map[string]hub.PacketWarning{}
	for _, warning := range question.Warnings {
		byCode[warning.Code] = warning
	}
	require.False(t, byCode["internal_id_in_question"].CarriedOver, "the revision rewrote the question, so the finding about it is news")
	require.True(t, byCode["missing_big_picture"].CarriedOver)
	require.True(t, byCode["missing_recommendation_reason"].CarriedOver)
}

func TestRevision_AMessageThatNamesNoPartIsNeverFolded(t *testing.T) {
	// Warnings about the request's own parts (an attachment the Hub left out) are
	// never carried over: they describe what this request sent.
	env := newAttachmentEnv(t)
	env.ask(t, "evt-0")
	env.advance(t, time.Minute)
	gone := env.revise(t, "evt-1", withAttachments(hub.AskAttachment{Name: "ghost.md", Kind: hub.AttachmentFile, SHA256: strings.Repeat("a", 64)}))
	require.False(t, findWarning(t, gone.Warnings, "attachment_blob_missing").CarriedOver)
}
