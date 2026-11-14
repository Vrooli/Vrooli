package conversations

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
)

// minimalAsk is the smallest ask: a question and a yes/no menu.
func minimalAsk(extra ...string) []string {
	return append([]string{"--question", "Retry tonight's backup now?", "--yes-no"}, extra...)
}

func TestAskNeedsOnlyAQuestionAndAMenuAndFillsInTheRest(t *testing.T) {
	stub := &hubStub{}
	svc := newTestService(t, stub)
	_, _, err := runCommand(t, svc, "ask", minimalAsk()...)
	require.NoError(t, err)
	require.Len(t, stub.asks, 1)
	req := stub.asks[0]
	require.Nil(t, req.Urgency, "no urgency is sent, so the Hub applies its default (normal) to a new ask and keeps an open ask's")
	require.Empty(t, req.GetDeadline(), "no deadline is sent, so the Hub applies its 24 h default and a retry never conflicts over a moving deadline")
	require.Empty(t, req.GetSensitivityLabel(), "the Hub's default (private) applies")
	require.True(t, strings.HasPrefix(req.GetIdempotencyKey(), "ask-"), "the key is derived: %q", req.GetIdempotencyKey())
	require.Equal(t, []*v1.AskOption{{Key: "yes", Label: "Yes"}, {Key: "no", Label: "No"}}, req.GetOptions())
}

func TestAskDerivesTheSameKeyForTheSameTermsAndANewOneForChangedTerms(t *testing.T) {
	stub := &hubStub{}
	svc := newTestService(t, stub)
	for _, args := range [][]string{minimalAsk(), minimalAsk(), minimalAsk("--expires", "2h"), minimalAsk("--expires", "2h"), minimalAsk("--expires", "3h"), {"--question", "Retry tonight's backup now, or after the weekend?", "--yes-no"}} {
		_, _, err := runCommand(t, svc, "ask", args...)
		require.NoError(t, err)
	}
	keys := make([]string, len(stub.asks))
	for i, ask := range stub.asks {
		keys[i] = ask.GetIdempotencyKey()
	}
	require.Equal(t, keys[0], keys[1], "the same terms give the same key, so a retry returns the same ask")
	require.Equal(t, keys[2], keys[3], "a relative expiry is keyed by the duration written, not the time it resolves to, so a resend a minute later is the same request")
	require.NotEqual(t, keys[0], keys[2], "a resend that changes the expiry is not the same request")
	require.NotEqual(t, keys[2], keys[4], "a different duration is a different request")
	require.NotEqual(t, keys[0], keys[5], "changed text is a new revision and needs a new key")
}

func TestAKeyedAskGetsAFreshKeyEachSendSoReturningToEarlierWordingIsNotIgnored(t *testing.T) {
	stub := &hubStub{}
	svc := newTestService(t, stub)
	for i := 0; i < 2; i++ {
		_, _, err := runCommand(t, svc, "ask", "--subject-key", "release-check", "--technical-details", "Run 42 finished.")
		require.NoError(t, err)
	}
	first, second := stub.asks[0].GetIdempotencyKey(), stub.asks[1].GetIdempotencyKey()
	require.True(t, strings.HasPrefix(first, "ask-"), first)
	require.NotEqual(t, first, second, "the Hub reads a key an earlier revision used as that event arriving late")
}

func TestAskKeepsAnExplicitIdempotencyKey(t *testing.T) {
	stub := &hubStub{}
	_, _, err := runCommand(t, newTestService(t, stub), "ask", minimalAsk("--idempotency-key", "mine-1")...)
	require.NoError(t, err)
	require.Equal(t, "mine-1", stub.asks[0].GetIdempotencyKey())
}

func TestAskHumanOutputEndsWithOneMachineReadableLine(t *testing.T) {
	stub := &hubStub{askResp: &v1.AskResponse{AskId: "ask-42", State: "pending"}}
	stdout, _, err := runCommand(t, newTestService(t, stub), "ask", minimalAsk()...)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	last := lines[len(lines)-1]
	require.Regexp(t, `^ask_id=ask-42 state=pending outcome=created idempotency_key=ask-[0-9a-f]{20} next: notification-hub conversations wait ask-42$`, last)
	require.Len(t, lines, 2, "one sentence for people, then the line for programs: %q", stdout)
}

func TestAskJSONIsTheStableShape(t *testing.T) {
	stub := &hubStub{askResp: &v1.AskResponse{
		AskId: "ask-42", State: "pending",
		Warnings: []*v1.PacketWarning{{Code: "missing_big_picture", Field: "big_picture", Message: "add one", Severity: "advice"}},
	}}
	stdout, stderr, err := runCommand(t, newTestService(t, stub), "ask", minimalAsk("--json")...)
	require.NoError(t, err)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &parsed), "stdout is pure JSON: %q", stdout)
	require.Len(t, parsed, 4, "exactly ask_id, state, outcome and warnings: %v", parsed)
	require.Equal(t, "ask-42", parsed["ask_id"])
	require.Equal(t, "pending", parsed["state"])
	require.Equal(t, "", parsed["outcome"], "present even when empty")
	warnings := parsed["warnings"].([]any)
	require.Len(t, warnings, 1, "warnings are in the JSON, not only on stderr")
	require.Equal(t, "missing_big_picture", warnings[0].(map[string]any)["code"])
	require.Contains(t, stderr, "missing_big_picture", "and still on stderr for a person")

	stub = &hubStub{askResp: &v1.AskResponse{AskId: "ask-43", State: "pending"}}
	stdout, _, err = runCommand(t, newTestService(t, stub), "ask", minimalAsk("--json")...)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal([]byte(stdout), &parsed))
	require.Equal(t, []any{}, parsed["warnings"], "an empty list, never absent")
}

func TestAskWithNoMenuOrAQuestionSaysWhatIsNeeded(t *testing.T) {
	svc := newTestService(t, &hubStub{})
	_, _, err := runCommand(t, svc, "ask", "--question", "Retry?")
	require.ErrorContains(t, err, "--yes-no")
	_, _, err = runCommand(t, svc, "ask", "--yes-no")
	require.ErrorContains(t, err, "needs a question")
	_, _, err = runCommand(t, svc, "ask", "--question", "Retry?", "--yes-no", "--option", "a", "--option", "b")
	require.ErrorContains(t, err, "--yes-no")
}

func TestAskBodyFileGivesTheQuestionBackgroundAndDetails(t *testing.T) {
	body := writeFile(t, t.TempDir(), "ask.md", `# Backup decision

Retry tonight's backup to the
Elements drive now?

Some context that has no heading.

## Background
The drive was unplugged on Tuesday and is mounted again.

## Details
volume=elements, last_error=EIO

## Notes to self
Not for the operator.
`)
	stub := &hubStub{}
	_, stderr, err := runCommand(t, newTestService(t, stub), "ask", "--yes-no", "--body-file", body)
	require.NoError(t, err)
	req := stub.asks[0]
	require.Equal(t, "Retry tonight's backup to the Elements drive now?", req.GetQuestion())
	require.Equal(t, "Some context that has no heading.\n\nThe drive was unplugged on Tuesday and is mounted again.", req.GetBigPicture())
	require.Equal(t, "volume=elements, last_error=EIO", req.GetTechnicalDetails())
	require.Contains(t, stderr, `"Notes to self"`, "a section with no place in an ask is reported, not silently lost")
}

func TestAskBodyFileReadsStandardInputAndRefusesTheSamePartTwice(t *testing.T) {
	stub := &hubStub{}
	svc := newTestService(t, stub)
	svc.stdin = strings.NewReader("Retry now?\n\n## Details\nvolume=elements\n")
	_, _, err := runCommand(t, svc, "ask", "--yes-no", "--body-file", "-")
	require.NoError(t, err)
	require.Equal(t, "Retry now?", stub.asks[0].GetQuestion())

	body := writeFile(t, t.TempDir(), "ask.md", "Retry now?\n")
	_, _, err = runCommand(t, svc, "ask", "--yes-no", "--body-file", body, "--question", "Retry?")
	require.ErrorContains(t, err, "both give the same text")

	empty := writeFile(t, t.TempDir(), "empty.md", "## Background\nonly background\n")
	_, _, err = runCommand(t, svc, "ask", "--yes-no", "--body-file", empty)
	require.ErrorContains(t, err, "no question")
}

func TestAskDeclaresContextRequestsClearsAndSendsTheDeadlineOnlyWhenGiven(t *testing.T) {
	stub := &hubStub{}
	svc := newTestService(t, stub)
	_, _, err := runCommand(t, svc, "ask", minimalAsk("--subject-key", "backup", "--expires", "2h", "--clear-attachments", "--clear-questions")...)
	require.NoError(t, err)
	req := stub.asks[0]
	require.True(t, req.GetClearAttachments())
	require.True(t, req.GetClearQuestions())
	require.True(t, req.GetAcceptsContextRequests())
	deadline, err := time.Parse(time.RFC3339, req.GetDeadline())
	require.NoError(t, err)
	require.WithinDuration(t, time.Now().Add(2*time.Hour), deadline, time.Minute)
}

func TestAskDryRunChecksAndPreviewsWithoutSending(t *testing.T) {
	stub := &hubStub{checkResp: &v1.CheckAskResponse{
		PushTitle: "Decision required", PushBody: "Retry tonight's backup now?\nRecommended: Yes",
		RevisesAskId: "ask-9",
		Warnings:     []*v1.PacketWarning{{Code: "missing_big_picture", Field: "big_picture", Message: "add one", Severity: "advice"}},
	}}
	stdout, stderr, err := runCommand(t, newTestService(t, stub), "ask", minimalAsk("--subject-key", "backup", "--dry-run")...)
	require.NoError(t, err)
	require.Empty(t, stub.asks, "a dry run never sends the ask")
	require.Len(t, stub.checks, 1)
	require.Equal(t, "Retry tonight's backup now?", stub.checks[0].GetAsk().GetQuestion())
	require.Contains(t, stdout, "Dry run: nothing was stored and no ask was created.")
	require.Contains(t, stdout, "Would revise open ask ask-9 in place")
	require.Contains(t, stdout, "Title: Decision required")
	require.Contains(t, stdout, "Recommended: Yes")
	require.Contains(t, stderr, "missing_big_picture")

	stub.checkResp = &v1.CheckAskResponse{PushTitle: "Decision required", PushBody: "q"}
	stdout, _, err = runCommand(t, newTestService(t, stub), "ask", minimalAsk("--dry-run")...)
	require.NoError(t, err)
	require.Contains(t, stdout, "Would create a new ask")

	stdout, _, err = runCommand(t, newTestService(t, stub), "ask", minimalAsk("--dry-run", "--json")...)
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(stdout)), "JSON stdout stays pure: %q", stdout)
}

func TestAskAndWaitWaitsForTheAnswerAndExitsWithWaitsStatus(t *testing.T) {
	stub := &hubStub{
		askResp:  &v1.AskResponse{AskId: "ask-42", State: "pending"},
		waitResp: &v1.WaitResponse{AskId: "ask-42", State: "answered", Answer: "yes"},
	}
	stdout, _, err := runCommand(t, newTestService(t, stub), "ask", minimalAsk("--and-wait")...)
	require.NoError(t, err)
	require.Len(t, stub.waits, 1)
	require.Equal(t, "ask-42", stub.waits[0].GetAskId())
	deadline, err := time.Parse(time.RFC3339, stub.waits[0].GetDeadline())
	require.NoError(t, err)
	require.WithinDuration(t, time.Now().Add(30*time.Minute), deadline, time.Minute, "--timeout defaults to 30m")
	require.Contains(t, stdout, "ask_id=ask-42 state=answered answer=yes")

	stub = &hubStub{
		askResp:  &v1.AskResponse{AskId: "ask-42", State: "pending", Outcome: "created"},
		waitResp: &v1.WaitResponse{AskId: "ask-42", State: "dismissed"},
	}
	_, _, err = runCommand(t, newTestService(t, stub), "ask", minimalAsk("--and-wait", "--timeout", "5m")...)
	require.Equal(t, ExitClosedUnanswered, exitCodeOf(t, err))
}

func TestAskAndWaitFollowsAKeyedAskByItsKey(t *testing.T) {
	stub := &hubStub{
		askResp:  &v1.AskResponse{AskId: "ask-42", State: "pending"},
		waitResp: &v1.WaitResponse{AskId: "ask-42", State: "answered"},
	}
	_, _, err := runCommand(t, newTestService(t, stub), "ask", minimalAsk("--subject-key", "backup", "--and-wait")...)
	require.NoError(t, err)
	require.Equal(t, "backup", stub.waits[0].GetSubjectKey(), "a key keeps working if the ask is replaced")
	require.Empty(t, stub.waits[0].GetAskId())
}

func TestAskConflictTellsTheAgentWhatToDoNext(t *testing.T) {
	stub := &hubStub{askErr: connect.NewError(connect.CodeAlreadyExists, errors.New("ask identity is already bound to different immutable terms"))}
	_, _, err := runCommand(t, newTestService(t, stub), "ask", minimalAsk("--idempotency-key", "k1")...)
	require.Error(t, err)
	for _, want := range []string{`"k1"`, "different terms", "drop --idempotency-key", "--subject-key", "updates that ask in place"} {
		require.ErrorContains(t, err, want)
	}
}

func TestAskRefusedWhileWithdrawnTellsTheAgentToSendAgain(t *testing.T) {
	stub := &hubStub{askErr: connect.NewError(connect.CodeFailedPrecondition, errors.New("decision request was withdrawn before ask creation"))}
	_, _, err := runCommand(t, newTestService(t, stub), "ask", minimalAsk("--subject-key", "backup")...)
	require.ErrorContains(t, err, "send the ask again")
}

func TestParseBodyHandlesAFirstParagraphRunningToAHeading(t *testing.T) {
	text, err := parseBody("Retry now?\n## Details\nd\n")
	require.NoError(t, err)
	require.Equal(t, "Retry now?", text.Question)
	require.Equal(t, "d", text.TechnicalDetails)
}

func TestAKeyedAskSendsOnlyWhatTheCommandGaveSoTheHubKeepsTheRest(t *testing.T) {
	stub := &hubStub{}
	_, _, err := runCommand(t, newTestService(t, stub), "ask", "--subject-key", "release-check", "--technical-details", "Run 42 finished.")
	require.NoError(t, err)
	require.Empty(t, stub.gets, "no read of the open ask: the Hub merges the revision where it is written")
	sent := stub.asks[0]
	require.Equal(t, "Run 42 finished.", sent.GetTechnicalDetails())
	require.Empty(t, sent.GetQuestion(), "no question sent: the open ask keeps its own")
	require.Empty(t, sent.GetOptions(), "no answers sent: the open ask keeps its own")
	require.Nil(t, sent.BigPicture)
	require.Nil(t, sent.Recommended)
	require.Nil(t, sent.RecommendationReason)
	require.Nil(t, sent.DefaultAnswer)
	require.Nil(t, sent.Reversible)
	require.Nil(t, sent.Urgency, "the flag's default is not something the command gave")
	require.Nil(t, sent.ContextUrl)
	require.Nil(t, sent.ReassertWithinSeconds)
	require.True(t, sent.GetAcceptsContextRequests(), "a keyed ask declares it answers context requests")
}

func TestACommandThatGivesAFieldSendsItEvenWhenEmptyToClearIt(t *testing.T) {
	stub := &hubStub{}
	_, _, err := runCommand(t, newTestService(t, stub), "ask", "--subject-key", "release-check",
		"--big-picture", "", "--urgency", "low", "--recommend", "hold", "--reversible", "--reassert-within", "15m")
	require.NoError(t, err)
	sent := stub.asks[0]
	require.NotNil(t, sent.BigPicture, "an empty value that was given clears the text")
	require.Empty(t, sent.GetBigPicture())
	require.Equal(t, "low", sent.GetUrgency())
	require.Equal(t, "hold", sent.GetRecommended())
	require.True(t, sent.GetReversible())
	require.EqualValues(t, 900, sent.GetReassertWithinSeconds())
}

func TestAKeyedAskThatNamesAnAnswerMenuSendsIt(t *testing.T) {
	stub := &hubStub{}
	_, _, err := runCommand(t, newTestService(t, stub), "ask", "--subject-key", "release-check", "--option", "ship", "--option", "hold")
	require.NoError(t, err)
	require.Equal(t, []string{"ship", "hold"}, optionKeysOf(stub.asks[0]))
}

func TestAnAskWithoutAKeyStillNeedsItsQuestionAndAnswersBeforeAnythingIsSent(t *testing.T) {
	stub := &hubStub{}
	_, _, err := runCommand(t, newTestService(t, stub), "ask", "--technical-details", "x", "--yes-no")
	require.ErrorContains(t, err, "an ask needs a question")
	_, _, err = runCommand(t, newTestService(t, stub), "ask", "--question", "Ship it?")
	require.Error(t, err, "a menu is needed too")
	require.Empty(t, stub.asks)
}

func optionKeysOf(req *v1.AskRequest) []string {
	var keys []string
	for _, option := range req.GetOptions() {
		keys = append(keys, option.GetKey())
	}
	return keys
}
