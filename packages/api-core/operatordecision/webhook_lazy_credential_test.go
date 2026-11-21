package operatordecision

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/askreader"
	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	conversationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"
)

// hubServer is a Notification Hub that holds one genuine ask and answers only
// the ask-reader bearer token it is given.
type hubServer struct {
	conversationconnect.UnimplementedConversationsServiceHandler
	ask   askreader.Ask
	reads atomic.Int32
}

func (h *hubServer) GetAsk(_ context.Context, req *connect.Request[conversationv1.GetAskRequest]) (*connect.Response[conversationv1.GetAskResponse], error) {
	if req.Header().Get("Authorization") != "Bearer ask-reader-token" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("bearer token required"))
	}
	h.reads.Add(1)
	return connect.NewResponse(&conversationv1.GetAskResponse{Ask: &conversationv1.Ask{
		Id: h.ask.ID, State: h.ask.State, Answer: h.ask.Answer, AnsweredBy: h.ask.AnsweredBy, AnsweredByIdentityKind: h.ask.AnsweredByIdentityKind,
		AnsweredByIdentitySource: h.ask.AnsweredByIdentitySource, AnsweredAt: h.ask.AnsweredAt, ResolvedAt: h.ask.ResolvedAt,
		Source: h.ask.Source, SourceEventType: h.ask.SourceEventType, RequesterRef: h.ask.RequesterRef, SubjectKey: h.ask.SubjectKey,
		Correlation: h.ask.Correlation,
	}}), nil
}

// A scenario started before the operator ran `service-principals sync --apply`
// must not need another restart afterwards: the webhook it already runs applies
// the unverified legacy path until the credential exists, then reads every
// resolution back, so a forged one is refused from the next delivery on.
func TestACredentialIssuedWhileTheScenarioRunsTurnsOnVerificationWithoutARestart(t *testing.T) {
	genuine := answeredGrant()
	hub := &hubServer{ask: genuine}
	_, handler := conversationconnect.NewConversationsServiceHandler(hub)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	var issued atomic.Bool
	credential := func(context.Context) (string, error) {
		if !issued.Load() {
			return "", errors.New("no such entry")
		}
		return "cred-1", nil
	}
	reader := askreader.NewReader("storage-manager", credential,
		askreader.WithBaseURL(func(context.Context) (string, error) { return server.URL, nil }),
		askreader.WithTokenOptions(askreader.WithExchanger(func(context.Context, string) (askreader.Exchanged, error) {
			return askreader.Exchanged{
				AccessToken: "ask-reader-token", ExpiresAt: time.Now().Add(10 * time.Minute),
				Subject: askreader.Principal("storage-manager"), Audience: askreader.Audience, Scopes: []string{askreader.Scope},
			}, nil
		})))

	f := newVerifyFixture(t, genuine)
	f.webhook = NewWebhook(WebhookConfig{
		Secret: StaticSecret(verifySecret), Source: "storage-manager", Reader: reader,
		Effects: []DomainEffect{&captureEffect{owns: verifyKind, applied: &f.applied}}, Logf: func(string, ...any) {},
	})

	// Before issuance: the genuine resolution is applied as the unverified legacy path, counted.
	if rec := f.post(AskResolvedEventType, claimOf(genuine)); rec.Code != http.StatusOK {
		t.Fatalf("before issuance: status %d", rec.Code)
	}
	if stats := f.webhook.Stats(); stats.UnverifiedApplied != 1 || stats.Verified != 0 {
		t.Fatalf("before issuance: stats = %+v", stats)
	}

	issued.Store(true)

	// After: the next genuine resolution is verified against the Hub's record.
	if rec := f.post(AskResolvedEventType, claimOf(genuine)); rec.Code != http.StatusOK {
		t.Fatalf("after issuance: status %d", rec.Code)
	}
	if stats := f.webhook.Stats(); stats.Verified != 1 || stats.UnverifiedApplied != 1 {
		t.Fatalf("after issuance: stats = %+v", stats)
	}
	// And a forged answer for the same ask is refused, and applies nothing.
	forged := claimOf(genuine)
	forged["answer"] = "deny"
	applied := len(f.applied)
	if rec := f.post(AskResolvedEventType, forged); rec.Code != http.StatusConflict {
		t.Fatalf("forged after issuance: status %d, want 409", rec.Code)
	}
	if len(f.applied) != applied || hub.reads.Load() != 2 {
		t.Fatalf("a refused resolution applies nothing: applied %d->%d, reads %d", applied, len(f.applied), hub.reads.Load())
	}
}
