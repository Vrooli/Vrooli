package conversations

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliutil"
	skillsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills"
	skillsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills/skills_v1connect"
)

func reminder(receipt guideReceipt, ask askGuideContext) string {
	var out bytes.Buffer
	writeGuideReminder(&out, receipt, ask)
	return out.String()
}

func TestGuideReminderStaysQuietForAnAgentThatReadTheGuide(t *testing.T) {
	if got := reminder(guideReceipt{Status: guideRead}, askGuideContext{AskID: "a1", SubjectKey: "k"}); got != "" {
		t.Fatalf("reminder printed after a confirmed read: %q", got)
	}
}

func TestGuideReminderTellsAnUnreadAgentHowToReadAndFixInPlace(t *testing.T) {
	got := reminder(guideReceipt{Status: guideNotRead}, askGuideContext{AskID: "a1", SubjectKey: "release-payments", Outcome: "created"})
	for _, want := range []string{"has not read", guideReadCommand, "--subject-key release-payments", "no new push"} {
		if !strings.Contains(got, want) {
			t.Fatalf("reminder lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "withdraw") {
		t.Fatalf("an ask with a subject key can be fixed in place; withdraw advice is wrong:\n%s", got)
	}
}

func TestGuideReminderWithoutConfirmationNamesTheCauseAndTheWithdrawRoute(t *testing.T) {
	got := reminder(guideReceipt{Status: guideUnverified, Reason: "this shell has no agent identity"}, askGuideContext{AskID: "ask-42"})
	for _, want := range []string{"could not confirm", "this shell has no agent identity", guideReadCommand, "withdraw --ask-id ask-42"} {
		if !strings.Contains(got, want) {
			t.Fatalf("reminder lacks %q:\n%s", want, got)
		}
	}
}

func TestGuideReminderSkipsAsksThatCanNoLongerChange(t *testing.T) {
	for _, outcome := range []string{"returned", "suppressed"} {
		if got := reminder(guideReceipt{Status: guideNotRead}, askGuideContext{AskID: "a1", SubjectKey: "k", Outcome: outcome}); got != "" {
			t.Fatalf("%s ask got a reminder: %q", outcome, got)
		}
	}
}

func TestGuideCheckerMakesNoCallWithoutAgentIdentity(t *testing.T) {
	t.Setenv(cliutil.EnvIdentityToken, "")
	called := false
	restore := promptManagerBase
	promptManagerBase = func(context.Context) (string, bool) { called = true; return "", false }
	t.Cleanup(func() { promptManagerBase = restore })

	got := promptManagerGuideChecker(context.Background())
	if got.Status != guideUnverified || called {
		t.Fatalf("got %+v called=%v, want unverified without a lookup", got, called)
	}
}

type receiptStub struct {
	skillsconnect.UnimplementedSkillsServiceHandler
	status   string
	failWith *connect.Error
	sawToken *string
	sawSkill *string
}

func (s receiptStub) GetSkillReadReceipt(_ context.Context, req *connect.Request[skillsv1.GetSkillReadReceiptRequest]) (*connect.Response[skillsv1.GetSkillReadReceiptResponse], error) {
	*s.sawToken = req.Header().Get(cliutil.HeaderAgentIdentityToken)
	*s.sawSkill = req.Msg.GetSkillId()
	if s.failWith != nil {
		return nil, s.failWith
	}
	return connect.NewResponse(&skillsv1.GetSkillReadReceiptResponse{SkillId: req.Msg.GetSkillId(), Status: s.status, RunId: "run-a"}), nil
}

func TestGuideCheckerAsksPromptManagerAboutItsOwnRun(t *testing.T) {
	for name, tc := range map[string]struct {
		stub receiptStub
		want string
	}{
		"not read": {receiptStub{status: guideNotRead}, guideNotRead},
		"read":     {receiptStub{status: guideRead}, guideRead},
		"error":    {receiptStub{failWith: connect.NewError(connect.CodeUnimplemented, nil)}, guideUnavailable},
	} {
		var token, skill string
		tc.stub.sawToken, tc.stub.sawSkill = &token, &skill
		path, handler := skillsconnect.NewSkillsServiceHandler(tc.stub)
		mux := http.NewServeMux()
		mux.Handle(path, handler)
		server := httptest.NewServer(mux)
		restore := promptManagerBase
		promptManagerBase = func(context.Context) (string, bool) { return server.URL, true }
		t.Setenv(cliutil.EnvIdentityToken, "run-a-token")

		got := promptManagerGuideChecker(context.Background())
		promptManagerBase = restore
		server.Close()
		if got.Status != tc.want || token != "run-a-token" || skill != GuideSkillID {
			t.Fatalf("%s: got %+v token=%q skill=%q, want %s with the caller's own token", name, got, token, skill, tc.want)
		}
	}
}
