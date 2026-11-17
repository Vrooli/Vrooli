package askreader

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	conversationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"
)

type hubStub struct {
	conversationconnect.UnimplementedConversationsServiceHandler
	auth     string
	agent    string
	agentOp  string
	testMode string
	request  *conversationv1.GetAskRequest
	ask      *conversationv1.Ask
	err      error
}

func (h *hubStub) GetAsk(_ context.Context, req *connect.Request[conversationv1.GetAskRequest]) (*connect.Response[conversationv1.GetAskResponse], error) {
	h.request = req.Msg
	h.auth = req.Header().Get("Authorization")
	h.testMode = req.Header().Get("X-Vrooli-Test-Mode")
	h.agent = req.Header().Get(cliutil.HeaderAgentIdentityToken)
	h.agentOp = req.Header().Get(cliutil.HeaderAgentOperationToken)
	if h.err != nil {
		return nil, h.err
	}
	return connect.NewResponse(&conversationv1.GetAskResponse{Ask: h.ask}), nil
}

func TestReadAskBySubjectKeyUsesTheExactKeyAndReturnsRecurrence(t *testing.T) {
	stub := &hubStub{ask: &conversationv1.Ask{
		Id: "ask-1", SubjectKey: "usage-mode:auto-save-rule:claude",
		Recurrence: &conversationv1.SubjectRecurrence{Count: 2, Dates: []string{"2026-11-15", "2026-11-16"}, PreviousAnswers: []*conversationv1.HistoricalAnswer{{Answer: "auto_save", Note: "Keep it automatic.", AnsweredAt: "2026-11-15T10:00:00Z"}}},
	}}
	got, err := newReaderFixture(t, stub).ReadAskBySubjectKey(context.Background(), " usage-mode:auto-save-rule:claude ")
	if err != nil {
		t.Fatal(err)
	}
	if stub.request.GetSubjectKey() != "usage-mode:auto-save-rule:claude" || stub.request.GetAskId() != "" {
		t.Fatalf("request = %+v, want exact subject key only", stub.request)
	}
	if got.ID != "ask-1" || got.Recurrence.Count != 2 || len(got.Recurrence.Dates) != 2 || len(got.Recurrence.PreviousAnswers) != 1 || got.Recurrence.PreviousAnswers[0].Note != "Keep it automatic." {
		t.Fatalf("ask recurrence = %+v", got.Recurrence)
	}
	wrongSubject := &hubStub{ask: &conversationv1.Ask{Id: "ask-2", SubjectKey: "another-subject"}}
	if _, err := newReaderFixture(t, wrongSubject).ReadAskBySubjectKey(context.Background(), "usage-mode:auto-save-rule:claude"); !errors.Is(err, ErrAskNotFound) {
		t.Fatalf("mismatched subject response = %v, want ErrAskNotFound", err)
	}
}

func TestReadAskUsesTestPoolOnlyWhenExplicitlyEnabled(t *testing.T) {
	stub := &hubStub{ask: &conversationv1.Ask{Id: "ask-1"}}
	reader, _ := newReaderWith(t, stub)
	if _, err := reader.ReadAsk(context.Background(), "ask-1"); err != nil {
		t.Fatal(err)
	}
	if stub.testMode != "" {
		t.Fatalf("ordinary reader sent test-mode header %q", stub.testMode)
	}
	reader, _ = newReaderWith(t, stub, WithTestMode())
	if _, err := reader.ReadAsk(context.Background(), "ask-1"); err != nil {
		t.Fatal(err)
	}
	if stub.testMode != "1" {
		t.Fatalf("explicit test-pool reader header = %q", stub.testMode)
	}
}

func newReaderFixture(t *testing.T, stub *hubStub) *Reader {
	t.Helper()
	reader, _ := newReaderWith(t, stub)
	return reader
}

// newReaderWith builds a Reader over the scripted credential and exchange of the
// returned fixture, and a Hub that serves stub.
func newReaderWith(t *testing.T, stub *hubStub, extra ...ReaderOption) (*Reader, *fixture) {
	t.Helper()
	_, handler := conversationconnect.NewConversationsServiceHandler(stub)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	f := newFixture()
	f.now = time.Now()
	f.result.ExpiresAt = f.now.Add(10 * time.Minute)
	options := []ReaderOption{
		WithBaseURL(func(context.Context) (string, error) { return server.URL, nil }),
		WithTokenOptions(f.options()...),
	}
	reader := NewReader("research-desk", f.credentialSource(), append(options, extra...)...)
	return reader, f
}

func TestReadAskPresentsTheAskReaderTokenAndReturnsTheHubsRecord(t *testing.T) {
	stub := &hubStub{ask: &conversationv1.Ask{
		Id: "ask-1", State: "answered", Answer: "approve", AnsweredBy: "alice", Source: "research-desk",
		SourceEventType: "research_desk.needs_you.decision_requested.v1", RequesterRef: "evt-1", SubjectKey: "k",
		Correlation:     map[string]string{"item": "i1"},
		ContextRequests: []*conversationv1.AskContextRequest{{Id: "r1", Question: "Why?", RequestedAt: "t"}},
		Responses: []*conversationv1.AskQuestionResponse{{QuestionId: "groups", Selected: []string{"keys", "ai-tools"}}, {
			QuestionId: "plan", Sections: []*conversationv1.AskSectionVerdict{{SectionKey: "s1", Verdict: "change", Note: "Shorter."}},
		}},
		AnswerSummary: "Answer: Approve (approve)",
	}}
	got, err := newReaderFixture(t, stub).ReadAsk(context.Background(), " ask-1 ")
	if err != nil {
		t.Fatal(err)
	}
	if stub.auth != "Bearer tok-1" {
		t.Fatalf("Authorization = %q", stub.auth)
	}
	if got.ID != "ask-1" || got.State != "answered" || got.AnsweredBy != "alice" || got.Correlation["item"] != "i1" ||
		len(got.ContextRequests) != 1 || got.ContextRequests[0].ID != "r1" || got.ContextRequests[0].Question != "Why?" {
		t.Fatalf("ask = %+v", got)
	}
	// The form answers come back too, so a scenario applies a group choice from the Hub's record.
	want := []Response{{QuestionID: "groups", Selected: []string{"keys", "ai-tools"}}, {QuestionID: "plan", Sections: []SectionVerdict{{SectionKey: "s1", Verdict: "change", Note: "Shorter."}}}}
	if !reflect.DeepEqual(got.Responses, want) || got.AnswerSummary != "Answer: Approve (approve)" {
		t.Fatalf("responses = %+v, summary = %q", got.Responses, got.AnswerSummary)
	}
}

func TestReadAskClassifiesWhyItCouldNotRead(t *testing.T) {
	ctx := context.Background()
	if _, err := newReaderFixture(t, &hubStub{err: connect.NewError(connect.CodeNotFound, errors.New("no"))}).ReadAsk(ctx, "a"); !errors.Is(err, ErrAskNotFound) {
		t.Fatalf("not found: %v", err)
	}
	for name, code := range map[string]connect.Code{"unavailable": connect.CodeUnavailable, "internal": connect.CodeInternal, "refused credential": connect.CodeUnauthenticated} {
		if _, err := newReaderFixture(t, &hubStub{err: connect.NewError(code, errors.New("x"))}).ReadAsk(ctx, "a"); !errors.Is(err, ErrHubUnavailable) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	// An answer for a different ask is not the ask that was asked for.
	if _, err := newReaderFixture(t, &hubStub{ask: &conversationv1.Ask{Id: "other"}}).ReadAsk(ctx, "a"); !errors.Is(err, ErrAskNotFound) {
		t.Fatalf("wrong ask: %v", err)
	}
	if _, err := (*Reader)(nil).ReadAsk(ctx, "a"); err == nil {
		t.Fatal("a nil reader refuses")
	}
	if _, err := (*Reader)(nil).ReadAskBySubjectKey(ctx, "key"); err == nil {
		t.Fatal("a nil reader refuses a subject key")
	}
}

func TestReadAskWithoutACredentialIsNotIssuedAndTheAuthenticatorBeingDownIsUnavailable(t *testing.T) {
	ctx := context.Background()
	reader, f := newReaderWith(t, &hubStub{ask: &conversationv1.Ask{Id: "a"}})
	f.credErr = errors.New("no such entry")
	if _, err := reader.ReadAsk(ctx, "a"); !errors.Is(err, ErrNotIssued) {
		t.Fatalf("no credential: %v", err)
	}
	f.issue("cred-1")
	f.exchangeEr = errors.New("authenticator down")
	if _, err := reader.ReadAsk(ctx, "a"); !errors.Is(err, ErrHubUnavailable) || errors.Is(err, ErrNotIssued) {
		t.Fatalf("exchange failed: %v", err)
	}
}

// The operator issues the credential while the scenario runs: the reader built
// before that reads its very next ask verified, and a credential that is still
// missing is asked for again each time rather than remembered as missing.
func TestACredentialIssuedAfterTheReaderWasBuiltIsUsedByTheNextRead(t *testing.T) {
	ctx := context.Background()
	stub := &hubStub{ask: &conversationv1.Ask{Id: "ask-1", State: "answered", Answer: "approve"}}
	reader, f := newReaderWith(t, stub)
	f.credential = ""
	for range 3 {
		if _, err := reader.ReadAsk(ctx, "ask-1"); !errors.Is(err, ErrNotIssued) {
			t.Fatalf("before issuance: %v, want ErrNotIssued", err)
		}
	}
	if stub.auth != "" {
		t.Fatalf("nothing may reach the Hub before a credential exists, got %q", stub.auth)
	}
	f.issue("cred-1")
	got, err := reader.ReadAsk(ctx, "ask-1")
	if err != nil || got.Answer != "approve" || stub.auth != "Bearer tok-1" {
		t.Fatalf("after issuance: ask=%+v err=%v auth=%q", got, err, stub.auth)
	}
}

// DBM read the operator's answer from inside a request an agent run made (the
// CLI carried the run's identity). The Hub resolves an agent identity before a
// service token, so forwarding it made the Hub treat the read as that run's and
// answer not-found for an ask the scenario raised. The read must present the
// ask-reader token alone, whatever identity the inbound request carried.
func TestReadAskNeverForwardsTheIdentityOfTheRunWhoseRequestLedToIt(t *testing.T) {
	stub := &hubStub{ask: &conversationv1.Ask{Id: "ask-1"}}
	reader, _ := newReaderWith(t, stub)

	inbound := provenance.Middleware(provenance.VerifierFunc(func(string) (*cliutil.VerifyResult, error) {
		return &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{RunID: "run-a"}}, nil
	}))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if provenance.ForwardedIdentityToken(r.Context()) == "" {
			t.Error("fixture: the inbound request should carry a verified agent identity")
		}
		if _, err := reader.ReadAsk(r.Context(), "ask-1"); err != nil {
			t.Errorf("ReadAsk: %v", err)
		}
	}))
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "run-a-credential")
	inbound.ServeHTTP(httptest.NewRecorder(), req)

	if stub.agent != "" || stub.agentOp != "" {
		t.Fatalf("the Hub received the agent identity (%q, %q) with the ask-reader read", stub.agent, stub.agentOp)
	}
	if stub.auth == "" {
		t.Fatal("the read must still present the ask-reader token")
	}
}
