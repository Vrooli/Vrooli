package conversations

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliutil"

	"github.com/stretchr/testify/require"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	"google.golang.org/protobuf/types/known/structpb"
)

func exitCodeOf(t *testing.T, err error) int {
	t.Helper()
	var coded interface{ ExitCode() int }
	require.True(t, errors.As(err, &coded), "error carries no exit status: %v", err)
	return coded.ExitCode()
}

func TestWaitReturnsPageSteeringAndImageReferencesToProducer(t *testing.T) {
	changes, err := structpb.NewStruct(map[string]any{"pace": "thursday", "slots": float64(4)})
	require.NoError(t, err)
	stub := &hubStub{
		waitResp: &v1.WaitResponse{AskId: "ask-7", State: "answered", Answer: "continue", Changes: changes, Images: []*v1.AnswerImage{{Name: "plan.png", Sha256: "abc123", MediaType: "image/png", SizeBytes: 128}}},
	}
	stdout, stderr, callErr := runCommand(t, newTestService(t, stub), "wait", "--ask-id", "ask-7", "--timeout", "10m", "--json")
	require.NoError(t, callErr)
	require.Empty(t, stderr)
	var output map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &output))
	require.Equal(t, map[string]any{"pace": "thursday", "slots": float64(4)}, output["changes"])
	images := output["images"].([]any)
	require.Equal(t, "plan.png", images[0].(map[string]any)["name"])
	require.Equal(t, "abc123", images[0].(map[string]any)["sha256"])
	require.Empty(t, stub.gets, "Wait carries the page answer without a follow-up read")

	stdout, _, callErr = runCommand(t, newTestService(t, stub), "wait", "--ask-id", "ask-7", "--timeout", "10m")
	require.NoError(t, callErr)
	require.Contains(t, stdout, "Page selections:")
	require.Contains(t, stdout, `"pace": "thursday"`)
	require.Contains(t, stdout, "plan.png")
}

var openWithContextRequest = &v1.WaitResponse{
	AskId: "ask-7", State: "pending",
	ContextRequest: &v1.AskContextRequest{Id: "c1", Question: "Which region is this for?"},
}

func TestWaitAsksToBeWokenByContextRequestsByDefault(t *testing.T) {
	stub := &hubStub{waitResp: &v1.WaitResponse{AskId: "ask-7", State: "answered", Answer: "yes"}}
	svc := newTestService(t, stub)
	_, _, err := runCommand(t, svc, "wait", "--ask-id", "ask-7", "--timeout", "10m")
	require.NoError(t, err)
	require.True(t, stub.waits[0].GetReturnOnContextRequest())
	deadline, err := time.Parse(time.RFC3339, stub.waits[0].GetDeadline())
	require.NoError(t, err)
	require.WithinDuration(t, time.Now().Add(10*time.Minute), deadline, time.Minute)
}

func TestWaitUntilAnswerKeepsWaitingThroughContextRequests(t *testing.T) {
	stub := &hubStub{waitResp: &v1.WaitResponse{AskId: "ask-7", State: "pending"}}
	svc := newTestService(t, stub)
	svc.now = deadlinePassesAfterTheFirstLook()
	_, _, err := runCommand(t, svc, "wait", "--ask-id", "ask-7", "--timeout", "10m", "--until-answer")
	require.Equal(t, ExitWaitTimedOut, exitCodeOf(t, err))
	require.False(t, stub.waits[0].GetReturnOnContextRequest())
}

func TestWaitPrintsTheAnswerSummaryWhenAnswered(t *testing.T) {
	stub := &hubStub{waitResp: &v1.WaitResponse{
		AskId: "ask-7", State: "answered", Answer: "approve",
		AnswerSummary: "Answer: Approve (approve)\nLayout: Cards (cards)",
	}}
	svc := newTestService(t, stub)
	stdout, stderr, err := runCommand(t, svc, "wait", "--ask-id", "ask-7", "--timeout", "10m")
	require.NoError(t, err)
	require.Contains(t, stdout, "Ask ask-7: answered")
	require.Contains(t, stdout, "Answer: Approve (approve)")
	require.Contains(t, stdout, "Layout: Cards (cards)")
	require.Empty(t, stderr)
}

func TestAnAnsweredWaitNamesThePacketAndTheExecutorMessage(t *testing.T) {
	stub := &hubStub{waitResp: &v1.WaitResponse{
		AskId: "ask-7", State: "answered", Answer: "approve", AnswerSummary: "Answer: Approve (approve)",
		Packet: "/home/me/.vrooli/research/topics/t/work/NAV-1/BRIEF.md",
	}}
	svc := newTestService(t, stub)
	stdout, _, err := runCommand(t, svc, "wait", "--ask-id", "ask-7", "--timeout", "10m")
	require.NoError(t, err)
	require.Contains(t, stdout, "Packet: /home/me/.vrooli/research/topics/t/work/NAV-1/BRIEF.md")
	require.Contains(t, stdout, "Execute the approved decision NAV-1. Read /home/me/.vrooli/research/topics/t/work/NAV-1/BRIEF.md; the answer is approve.")
}

func TestWaitFallsBackToTheBareAnswerWithoutASummary(t *testing.T) {
	stub := &hubStub{waitResp: &v1.WaitResponse{AskId: "ask-7", State: "defaulted", Answer: "no", Reason: "default applied"}}
	svc := newTestService(t, stub)
	stdout, _, err := runCommand(t, svc, "wait", "--ask-id", "ask-7", "--timeout", "10m")
	require.NoError(t, err)
	require.Contains(t, stdout, "no")
	require.Contains(t, stdout, "default applied")
}

func TestWaitReturnedForAContextRequestExitsWithTheDocumentedStatus(t *testing.T) {
	stub := &hubStub{waitResp: openWithContextRequest, getAsk: &v1.Ask{Id: "ask-7", SubjectKey: "release-check"}}
	svc := newTestService(t, stub)
	stdout, stderr, err := runCommand(t, svc, "wait", "--ask-id", "ask-7", "--timeout", "10m")
	require.Error(t, err)
	require.Equal(t, ExitContextRequested, exitCodeOf(t, err))
	require.Equal(t, 3, ExitContextRequested, "agents branch on this documented status")

	require.Contains(t, stdout, "Ask ask-7: pending; the operator asked for more context")
	require.Contains(t, stdout, "Which region is this for?")
	for _, want := range []string{
		"Which region is this for?", "--context-response", "the operator sees your answer under their question",
		"ask --subject-key release-check --context-response",
		"the ask keeps everything else", "conversations wait --subject-key release-check",
	} {
		require.Contains(t, stderr, want)
	}
	for _, gone := range []string{
		"withdraw", "--idempotency-key <new key>", "replaces the whole packet", "--attach, --explainer and --form again",
		"Text flags replace the whole text", "ask.big_picture", "the operator needs it to decide",
	} {
		require.NotContains(t, stderr, gone, "the answer is its own field: the agent no longer rewrites the big picture to give it")
	}
}

func TestWaitContextRequestOnAnAskWithoutASubjectKeyExplainsTheWithdrawRoute(t *testing.T) {
	stub := &hubStub{waitResp: openWithContextRequest, getAsk: &v1.Ask{Id: "ask-7"}}
	svc := newTestService(t, stub)
	_, stderr, err := runCommand(t, svc, "wait", "--ask-id", "ask-7", "--timeout", "10m")
	require.Equal(t, ExitContextRequested, exitCodeOf(t, err))
	require.Contains(t, stderr, "cannot be revised in place")
	require.Contains(t, stderr, "withdraw ask-7")
	require.Contains(t, stderr, "same --attach, --explainer and --form")
}

func TestWaitContextRequestStillGuidesWhenTheSubjectKeyLookupFails(t *testing.T) {
	stub := &hubStub{waitResp: openWithContextRequest} // GetAsk answers not-found
	svc := newTestService(t, stub)
	_, stderr, err := runCommand(t, svc, "wait", "--ask-id", "ask-7", "--timeout", "10m")
	require.Equal(t, ExitContextRequested, exitCodeOf(t, err))
	require.Contains(t, stderr, "Which region is this for?")
}

func TestWaitJSONKeepsStdoutPureAndStillSignalsTheContextRequest(t *testing.T) {
	stub := &hubStub{waitResp: openWithContextRequest, getAsk: &v1.Ask{SubjectKey: "k"}}
	svc := newTestService(t, stub)
	stdout, stderr, err := runCommand(t, svc, "wait", "--ask-id", "ask-7", "--timeout", "10m", "--json")
	require.Equal(t, ExitContextRequested, exitCodeOf(t, err))
	var parsed struct {
		State          string `json:"state"`
		ContextRequest struct {
			Question string `json:"question"`
		} `json:"context_request"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &parsed), "stdout must be pure JSON: %q", stdout)
	require.Equal(t, "Which region is this for?", parsed.ContextRequest.Question)
	require.Contains(t, stderr, "--subject-key k")
}

func TestWaitIgnoresAContextRequestOnAnAskThatIsAlreadyClosed(t *testing.T) {
	stub := &hubStub{waitResp: &v1.WaitResponse{
		AskId: "ask-7", State: "answered", Answer: "yes",
		ContextRequest: &v1.AskContextRequest{Question: "stale"},
	}}
	svc := newTestService(t, stub)
	_, stderr, err := runCommand(t, svc, "wait", "--ask-id", "ask-7", "--timeout", "10m")
	require.NoError(t, err, "an answer wins over a stale context request")
	require.Empty(t, stderr)
}

func TestWaitThatRunsOutWithTheAskStillOpenExitsWithStatusFour(t *testing.T) {
	stub := &hubStub{waitResp: &v1.WaitResponse{AskId: "ask-7", State: "pending"}}
	svc := newTestService(t, stub)
	svc.now = deadlinePassesAfterTheFirstLook()
	stdout, stderr, err := runCommand(t, svc, "wait", "--ask-id", "ask-7", "--timeout", "10m")
	require.Equal(t, ExitWaitTimedOut, exitCodeOf(t, err))
	require.Equal(t, 4, ExitWaitTimedOut, "agents branch on this documented status")
	require.Contains(t, stdout, "Ask ask-7: pending")
	require.Contains(t, stdout, "ask_id=ask-7 state=pending")
	require.Empty(t, stderr)
	require.ErrorContains(t, err, "your wait timed out")
	require.ErrorContains(t, err, "the ask has not expired")
	require.ErrorContains(t, err, "notification-hub conversations wait ask-7")
}

func TestWaitShowsAttachmentNotesWhenThereIsNoSummary(t *testing.T) {
	stub := &hubStub{waitResp: &v1.WaitResponse{
		AskId: "ask-7", State: "answered", Answer: "approve",
		AttachmentNotes: []*v1.AskAttachmentNote{{Attachment: "character.glb", Note: "Hair is too glossy."}, {Attachment: "shots", Path: "a.png", Note: "Crop."}},
	}}
	svc := newTestService(t, stub)
	stdout, _, err := runCommand(t, svc, "wait", "--ask-id", "ask-7", "--timeout", "10m")
	require.NoError(t, err)
	require.Contains(t, stdout, "character.glb: Hair is too glossy.")
	require.Contains(t, stdout, "shots/a.png: Crop.")
}

func TestReadsExplainWhatToDoWhenTheHubRefusesAnUnidentifiedShell(t *testing.T) {
	t.Setenv(cliutil.EnvIdentityToken, "")
	stub := &hubStub{readErr: connect.NewError(connect.CodeUnauthenticated, nil)}
	svc := newTestService(t, stub)
	for name, run := range map[string]func() error{
		"get":  func() error { _, _, err := runCommand(t, svc, "get", "ask-1"); return err },
		"list": func() error { _, _, err := runCommand(t, svc, "list"); return err },
		"wait": func() error {
			_, _, err := runCommand(t, svc, "wait", "--ask-id", "ask-1", "--timeout", "1m")
			return err
		},
	} {
		err := run()
		require.Error(t, err, name)
		for _, want := range []string{"no verified identity", "agent run that sent the ask", "NOTIFICATION_HUB_API_TOKEN"} {
			require.ErrorContains(t, err, want, name)
		}
		require.NotContains(t, err.Error(), "goroutine", name+": a clear message, never a stack trace")
	}
}

func TestAnAgentRunToldNotFoundLearnsItReadsItsConversationAndCanNameASubjectKey(t *testing.T) {
	t.Setenv(cliutil.EnvIdentityToken, "run-token")
	stub := &hubStub{readErr: connect.NewError(connect.CodeNotFound, nil)}
	svc := newTestService(t, stub)
	_, _, err := runCommand(t, svc, "get", "someone-elses")
	require.ErrorContains(t, err, "asks of its own conversation")
	require.ErrorContains(t, err, "--subject-key", "the way back to an ask of another conversation is named")

	t.Setenv(cliutil.EnvIdentityToken, "")
	_, _, err = runCommand(t, svc, "get", "someone-elses")
	require.Error(t, err)
	require.NotContains(t, err.Error(), "same run", "a shell without an agent identity gets the plain not-found")
}

func TestWaitBySubjectKeySendsTheKeyAndNeverAnAskID(t *testing.T) {
	stub := &hubStub{waitResp: &v1.WaitResponse{AskId: "ask-9", State: "answered", Answer: "yes"}}
	svc := newTestService(t, stub)
	stdout, _, err := runCommand(t, svc, "wait", "--subject-key", "plan-review", "--timeout", "10m")
	require.NoError(t, err)
	require.Len(t, stub.waits, 1)
	require.Equal(t, "plan-review", stub.waits[0].GetSubjectKey())
	require.Empty(t, stub.waits[0].GetAskId())
	require.Contains(t, stdout, "Ask ask-9: answered")
}

func TestWaitAndGetNameExactlyOneOfAnAskAndASubjectKey(t *testing.T) {
	stub := &hubStub{waitResp: &v1.WaitResponse{AskId: "ask-9", State: "answered"}, getAsk: &v1.Ask{Id: "ask-9"}}
	svc := newTestService(t, stub)
	for name, args := range map[string][]string{
		"wait with both":    {"wait", "--ask-id", "ask-9", "--subject-key", "plan-review", "--timeout", "10m"},
		"wait with neither": {"wait", "--timeout", "10m"},
		"get with both":     {"get", "ask-9", "--subject-key", "plan-review"},
		"get with neither":  {"get"},
	} {
		_, _, err := runCommand(t, svc, args[0], args[1:]...)
		require.ErrorContains(t, err, "exactly one", name)
	}
	require.Empty(t, stub.waits, "a malformed wait never reaches the Hub")
	require.Empty(t, stub.gets, "a malformed read never reaches the Hub")
}

func TestGetBySubjectKeySendsTheKeyAndRendersTheAsk(t *testing.T) {
	stub := &hubStub{getAsk: &v1.Ask{Id: "ask-9", Question: "Keep the plan?", State: "pending", SubjectKey: "plan-review"}}
	svc := newTestService(t, stub)
	stdout, _, err := runCommand(t, svc, "get", "--subject-key", "plan-review")
	require.NoError(t, err)
	require.Len(t, stub.gets, 1)
	require.Equal(t, "plan-review", stub.gets[0].GetSubjectKey())
	require.Empty(t, stub.gets[0].GetAskId())
	require.Contains(t, stdout, "Keep the plan?")
}

// A run that waits by key may be outside the ask's lineage, so reading the ask
// by id for its key would be refused. The guidance must come from the key it
// already named, never from the withdraw-and-ask-again route.
func TestAContextRequestWhileWaitingByKeyGuidesByKeyEvenWhenTheRunCannotReadTheAskByID(t *testing.T) {
	stub := &hubStub{waitResp: openWithContextRequest} // GetAsk by id answers not-found
	svc := newTestService(t, stub)
	_, stderr, err := runCommand(t, svc, "wait", "--subject-key", "release-check", "--timeout", "10m")
	require.Equal(t, ExitContextRequested, exitCodeOf(t, err))
	require.Contains(t, stderr, "--subject-key release-check")
	require.Contains(t, stderr, "conversations wait --subject-key release-check")
	require.NotContains(t, stderr, "cannot be revised in place")
	require.Empty(t, stub.gets, "the key is known, so the Hub is not asked for it")
}

// deadlinePassesAfterTheFirstLook is a clock for a wait whose deadline passes
// before the first answer arrives: the first reading fixes the deadline, every
// later one is an hour on.
func deadlinePassesAfterTheFirstLook() func() time.Time {
	start := time.Now()
	reads := 0
	return func() time.Time {
		reads++
		if reads == 1 {
			return start
		}
		return start.Add(time.Hour)
	}
}

func TestWaitExitStatusSaysWhatHappenedToTheAsk(t *testing.T) {
	for name, tc := range map[string]struct {
		resp     *v1.WaitResponse
		wantExit int
		wantText string
	}{
		"answered":  {&v1.WaitResponse{AskId: "a", State: "answered", Answer: "yes"}, 0, ""},
		"defaulted": {&v1.WaitResponse{AskId: "a", State: "defaulted", Answer: "no", Reason: "default applied"}, 0, ""},
		"withdrawn": {&v1.WaitResponse{AskId: "a", State: "withdrawn", Reason: "replaced by a newer ask"}, ExitClosedUnanswered, "wait by --subject-key"},
		"dismissed": {&v1.WaitResponse{AskId: "a", State: "dismissed"}, ExitClosedUnanswered, "dismissed without an answer"},
		"expired":   {&v1.WaitResponse{AskId: "a", State: "expired", Reason: "expired: never delivered"}, ExitClosedUnanswered, "expired without an answer"},
	} {
		t.Run(name, func(t *testing.T) {
			svc := newTestService(t, &hubStub{waitResp: tc.resp})
			_, _, err := runCommand(t, svc, "wait", "a", "--timeout", "10m")
			if tc.wantExit == 0 {
				require.NoError(t, err)
				return
			}
			require.Equal(t, tc.wantExit, exitCodeOf(t, err))
			require.ErrorContains(t, err, tc.wantText)
		})
	}
	require.Equal(t, 5, ExitClosedUnanswered, "agents branch on this documented status")
}

func TestWaitAcceptsThePositionalIDTheAskFlagAndRefusesAConflict(t *testing.T) {
	stub := &hubStub{waitResp: &v1.WaitResponse{AskId: "ask-9", State: "answered", Answer: "yes"}}
	svc := newTestService(t, stub)
	stdout, _, err := runCommand(t, svc, "wait", "ask-9")
	require.NoError(t, err, "the deadline defaults to 30m")
	require.Equal(t, "ask-9", stub.waits[0].GetAskId())
	require.Contains(t, stdout, "ask_id=ask-9 state=answered answer=yes")
	deadline, err := time.Parse(time.RFC3339, stub.waits[0].GetDeadline())
	require.NoError(t, err)
	require.WithinDuration(t, time.Now().Add(30*time.Minute), deadline, time.Minute)

	_, _, err = runCommand(t, svc, "wait", "ask-9", "--ask-id", "ask-9")
	require.NoError(t, err, "the same id twice is not a conflict")
	_, _, err = runCommand(t, svc, "wait", "ask-9", "--ask-id", "ask-8")
	require.ErrorContains(t, err, "differs")
}

// The Hub ends one wait after ten minutes with the ask still open, and a proxy or
// a restart can cut a long request sooner: the CLI asks again until its own
// deadline instead of failing or reporting the ask as unanswered.
func TestWaitCarriesOnAcrossTheHubsLongestWaitAndDroppedConnections(t *testing.T) {
	open := &v1.WaitResponse{AskId: "ask-7", State: "pending", Reason: "wait_slice: the ask is still open"}
	stub := &hubStub{
		waitSeq:    []*v1.WaitResponse{open, open, open, {AskId: "ask-7", State: "answered", Answer: "yes"}},
		waitErrSeq: []error{nil, connect.NewError(connect.CodeUnavailable, errors.New("unexpected EOF")), connect.NewError(connect.CodeUnavailable, nil)},
	}
	svc := newTestService(t, stub)
	var pauses []time.Duration
	svc.sleep = func(d time.Duration) { pauses = append(pauses, d) }

	stdout, _, err := runCommand(t, svc, "wait", "ask-7", "--timeout", "2h")
	require.NoError(t, err)
	require.Len(t, stub.waits, 4, "a slice end, two dropped connections, then the answer")
	require.Equal(t, []time.Duration{time.Second, 2 * time.Second}, pauses, "a short, growing pause between attempts to reach the Hub")
	require.Contains(t, stdout, "ask_id=ask-7 state=answered answer=yes")
}

func TestWaitGivesUpOnAConnectionFailureOnlyAtTheDeadlineAndNeverOnARefusal(t *testing.T) {
	down := connect.NewError(connect.CodeUnavailable, errors.New("connection refused"))
	stub := &hubStub{waitErrSeq: []error{down, down, down, down, down, down}}
	svc := newTestService(t, stub)
	svc.now = deadlinePassesAfterTheFirstLook()
	svc.sleep = func(time.Duration) {}
	_, _, err := runCommand(t, svc, "wait", "ask-7", "--timeout", "10m")
	require.Error(t, err)
	require.Len(t, stub.waits, 1, "past the deadline a failure is reported, not retried")

	refused := &hubStub{readErr: connect.NewError(connect.CodeUnauthenticated, nil)}
	svc = newTestService(t, refused)
	svc.sleep = func(time.Duration) { t.Fatal("a refusal must not be retried") }
	_, _, err = runCommand(t, svc, "wait", "ask-7", "--timeout", "10m")
	require.ErrorContains(t, err, "no verified identity")
}

func TestWaitRetriesUnauthenticatedRunIdentityOnlyWhileItsBrokerIsUnavailable(t *testing.T) {
	t.Run("broker restart", func(t *testing.T) {
		t.Setenv(cliutil.EnvIdentityToken, "synthetic-run-identity")
		t.Setenv("NOTIFICATION_HUB_API_TOKEN", "")
		t.Setenv("VROOLI_API_TOKEN", "")
		t.Setenv(SendAsEnv, "agent-manager")
		refused := connect.NewError(connect.CodeUnauthenticated, errors.New("stale inherited token"))
		stub := &hubStub{
			waitErrSeq: []error{refused, nil},
			waitSeq:    []*v1.WaitResponse{{AskId: "ask-7", State: "answered", Answer: "yes"}},
		}
		svc := newTestService(t, stub)
		checks := 0
		svc.identityBrokerUnavailable = func() bool {
			checks++
			return checks == 1 // the restarted owner becomes reachable before retry
		}
		svc.sleep = func(time.Duration) {}

		stdout, _, err := runCommand(t, svc, "wait", "ask-7", "--timeout", "10m")
		require.NoError(t, err)
		require.Len(t, stub.waits, 2, "the owner wait is reissued after broker recovery")
		require.Contains(t, stdout, "state=answered answer=yes")
	})

	t.Run("foreign owner while broker is live", func(t *testing.T) {
		t.Setenv(cliutil.EnvIdentityToken, "synthetic-run-identity")
		refused := connect.NewError(connect.CodeUnauthenticated, errors.New("no verified identity"))
		stub := &hubStub{waitErrSeq: []error{refused, nil}, waitSeq: []*v1.WaitResponse{{AskId: "ask-7", State: "answered"}}}
		svc := newTestService(t, stub)
		svc.identityBrokerUnavailable = func() bool { return false }
		svc.sleep = func(time.Duration) { t.Fatal("a live broker makes this refusal authoritative") }

		_, _, err := runCommand(t, svc, "wait", "ask-7", "--timeout", "10m")
		require.ErrorContains(t, err, "no verified identity")
		require.Len(t, stub.waits, 1, "a foreign-owner refusal is not retried")
	})
}

func TestAgentIdentityBrokerUnavailableRequiresInheritedIdentityAndMissingSocket(t *testing.T) {
	t.Setenv(cliutil.EnvIdentityToken, "synthetic-run-identity")
	t.Setenv("NOTIFICATION_HUB_API_TOKEN", "")
	t.Setenv("VROOLI_API_TOKEN", "")
	t.Setenv(SendAsEnv, "")
	shortTemp, err := os.MkdirTemp("/tmp", "n4-")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(shortTemp) })
	t.Setenv("TMPDIR", shortTemp)
	require.True(t, agentIdentityBrokerUnavailable(""), "a missing owner socket can explain a transient run-auth refusal")
	t.Setenv(SendAsEnv, "agent-manager")
	require.True(t, agentIdentityBrokerUnavailable(""), "an inherited run identity takes precedence over the send-as switch")

	socket := cliutil.AgentCredentialSocket()
	require.NoError(t, os.MkdirAll(filepath.Dir(socket), 0o700))
	listener, err := net.Listen("unix", socket)
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })
	require.False(t, agentIdentityBrokerUnavailable(""), "a live owner socket makes Hub refusals authoritative")
	require.False(t, agentIdentityBrokerUnavailable("saved-owner-token"), "an explicit configured token is not refreshed by the run broker")
	t.Setenv(cliutil.EnvIdentityToken, "")
	require.False(t, agentIdentityBrokerUnavailable(""), "anonymous callers do not acquire run-identity retry semantics")
}
