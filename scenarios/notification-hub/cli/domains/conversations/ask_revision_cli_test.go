package conversations

import (
	"testing"

	"github.com/stretchr/testify/require"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
)

func TestAskRemovesOneAttachmentByNameWithoutUploadingAnything(t *testing.T) {
	stub := &hubStub{askResp: &v1.AskResponse{AskId: "ask-1", Outcome: "updated"}}
	_, stderr, err := runCommand(t, newTestService(t, stub), "ask", "--subject-key", "release", "--remove-attachment", "notes.md", "--remove-attachment", "old.png")
	require.NoError(t, err)
	require.Equal(t, []string{"notes.md", "old.png"}, stub.asks[0].GetRemoveAttachments())
	require.Empty(t, stub.asks[0].GetAttachments())
	require.False(t, stub.asks[0].GetClearAttachments())
	require.Contains(t, stderr, "Kept as the open ask has them:")
	require.NotContains(t, stderr, "the attachments", "a revision that removes an attachment did say something about them")
}

func TestAskRemovesFormQuestionsByIdAndSaysItChangedTheForm(t *testing.T) {
	stub := &hubStub{askResp: &v1.AskResponse{AskId: "ask-1", Outcome: "updated"}}
	_, stderr, err := runCommand(t, newTestService(t, stub), "ask", "--subject-key", "release", "--remove-question", "owner", "--remove-question", "when")
	require.NoError(t, err)
	require.Equal(t, []string{"owner", "when"}, stub.asks[0].GetRemoveQuestions())
	require.Empty(t, stub.asks[0].GetQuestions())
	require.False(t, stub.asks[0].GetClearQuestions())
	require.NotContains(t, stderr, "the form", "a revision that removes a question did say something about the form")
}

func TestAskSendsTheContextResponseAsItsOwnField(t *testing.T) {
	stub := &hubStub{askResp: &v1.AskResponse{AskId: "ask-1", Outcome: "updated"}}
	_, stderr, err := runCommand(t, newTestService(t, stub), "ask", "--subject-key", "release", "--context-response", "  The Elements drive.  ")
	require.NoError(t, err)
	sent := stub.asks[0]
	require.Equal(t, "The Elements drive.", sent.GetContextResponse())
	require.Nil(t, sent.BigPicture, "the answer is not written into the big picture")
	require.Nil(t, sent.TechnicalDetails)
	require.Empty(t, sent.GetQuestion())
	require.Contains(t, stderr, "Kept as the open ask has them:")
}

func TestAskSaysWhatARevisionKept(t *testing.T) {
	stub := &hubStub{askResp: &v1.AskResponse{AskId: "ask-1", Outcome: "updated"}}
	_, stderr, err := runCommand(t, newTestService(t, stub), "ask", "--subject-key", "release", "--technical-details", "Run 42.")
	require.NoError(t, err)
	require.Contains(t, stderr, "Kept as the open ask has them: the question, the options, the recommendation, the big picture, the context link, the urgency, the sender, the attachments, the form.")
	require.NotContains(t, stderr, "technical details")

	stub = &hubStub{askResp: &v1.AskResponse{AskId: "ask-1", Outcome: "updated"}}
	_, stderr, err = runCommand(t, newTestService(t, stub), "ask", "--subject-key", "release", "--from", "Release bot")
	require.NoError(t, err)
	require.NotContains(t, stderr, "the sender", "a revision that names its sender does not keep the old one")

	stub = &hubStub{askResp: &v1.AskResponse{AskId: "ask-1", Outcome: "created"}}
	_, stderr, err = runCommand(t, newTestService(t, stub), "ask", minimalAsk("--subject-key", "release")...)
	require.NoError(t, err)
	require.NotContains(t, stderr, "Kept", "a new ask kept nothing")

	stub = &hubStub{askResp: &v1.AskResponse{AskId: "ask-2", Outcome: "replaced"}}
	_, stderr, err = runCommand(t, newTestService(t, stub), "ask", "--subject-key", "release", "--option", "a", "--option", "b")
	require.NoError(t, err)
	require.NotContains(t, stderr, "the recommendation", "new options take the old recommendation with them")
	require.NotContains(t, stderr, "the options")
}

func TestAskDryRunUploadsNothingAndGivesTheCheckWhatItReads(t *testing.T) {
	dir := t.TempDir()
	page := writeFile(t, dir, "why.html", "<html><meta name=viewport content=x></html>")
	plan := writeFile(t, dir, "plan.md", "## Storage\n")
	other := writeFile(t, dir, "other.md", "not read by the check")
	form := writeFile(t, dir, "form.yaml", "- {id: plan, type: section_review, prompt: Review it., sections_from: plan.md}\n")
	stub := &hubStub{checkResp: &v1.CheckAskResponse{PushTitle: "Decision required", PushBody: "Ship it?"}}

	_, _, err := runCommand(t, newTestService(t, stub), "ask", askArgs("--explainer", page, "--attach", plan, "--attach", other, "--form", form, "--dry-run")...)
	require.NoError(t, err)

	require.Empty(t, stub.statCalls, "a dry run asks the Hub nothing about blobs")
	require.Empty(t, stub.puts, "a dry run uploads nothing")
	require.Empty(t, stub.asks)
	require.Len(t, stub.checks, 1)
	attached := stub.checks[0].GetAsk().GetAttachments()
	require.Len(t, attached, 3)
	for _, attachment := range attached {
		require.Len(t, attachment.GetSha256(), 64, "each attachment is stated by its hash and size")
		require.NotZero(t, attachment.GetSizeBytes())
	}
	contents := stub.checks[0].GetFileContents()
	require.Len(t, contents, 2, "only what the check reads is sent: the explainer page and the plan the review follows")
	require.Equal(t, "## Storage\n", string(contents[attached[1].GetSha256()]))
	require.Contains(t, string(contents[attached[0].GetSha256()]), "viewport")
	require.NotContains(t, contents, attached[2].GetSha256())
}

func TestAskDryRunWithAndWaitIsRefusedBeforeAnythingHappens(t *testing.T) {
	stub := &hubStub{}
	_, _, err := runCommand(t, newTestService(t, stub), "ask", minimalAsk("--dry-run", "--and-wait")...)
	require.ErrorContains(t, err, "--dry-run creates no ask, so there is nothing to wait for")
	require.Empty(t, stub.checks)
	require.Empty(t, stub.asks)
}

func TestGetTakesTheAskIDAsAPositionalOrAsAskID(t *testing.T) {
	stub := &hubStub{getAsk: &v1.Ask{Id: "ask-7", State: "pending", Question: "Q?"}}
	svc := newTestService(t, stub)
	_, _, err := runCommand(t, svc, "get", "ask-7")
	require.NoError(t, err)
	_, _, err = runCommand(t, svc, "get", "--ask-id", "ask-7")
	require.NoError(t, err)
	require.Equal(t, "ask-7", stub.gets[0].GetAskId())
	require.Equal(t, "ask-7", stub.gets[1].GetAskId(), "get names the ask the way wait and withdraw do")
}

func TestAskWaitTimeoutIsNamedForTheWaitNotTheAsk(t *testing.T) {
	stub := &hubStub{}
	_, _, err := runCommand(t, newTestService(t, stub), "ask", minimalAsk("--and-wait", "--timeout", "tomorrow")...)
	require.ErrorContains(t, err, `--timeout "tomorrow"`)
	_, _, err = runCommand(t, newTestService(t, stub), "ask", minimalAsk("--expires", "soon")...)
	require.ErrorContains(t, err, `--expires "soon"`)
}

func TestListLimitIsABoundedWholeNumber(t *testing.T) {
	for value, want := range map[string]int32{"": 0, " ": 0, "1": 1, "50": 50, "200": 200} {
		got, err := parsePageLimit(value)
		require.NoError(t, err, value)
		require.Equal(t, want, got, value)
	}
	for _, value := range []string{"0", "-5", "201", "abc", "1.5", "99999999999"} {
		_, err := parsePageLimit(value)
		require.ErrorContains(t, err, "--limit must be a whole number from 1 to 200", value)
	}
}
