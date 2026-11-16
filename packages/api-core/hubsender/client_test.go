package hubsender

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/api-core/servicetoken"
	"github.com/vrooli/cli-core/cliutil"
	notificationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications"
	notificationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications/notifications_v1connect"
)

type notificationsStub struct {
	notificationsconnect.UnimplementedNotificationsServiceHandler
	auth  string
	agent string
	got   *notificationsv1.SendRequest
	err   error
}

func (s *notificationsStub) Send(_ context.Context, req *connect.Request[notificationsv1.SendRequest]) (*connect.Response[notificationsv1.SendResponse], error) {
	s.auth, s.got = req.Header().Get("Authorization"), req.Msg
	s.agent = req.Header().Get(cliutil.HeaderAgentIdentityToken)
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&notificationsv1.SendResponse{}), nil
}

func newClientFixture(t *testing.T, stub *notificationsStub, credential func() (string, error)) *Client {
	t.Helper()
	_, handler := notificationsconnect.NewNotificationsServiceHandler(stub)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return NewClient("storage-manager", func(context.Context) (string, error) { return credential() },
		WithBaseURL(func(context.Context) (string, error) { return server.URL, nil }),
		WithTokenOptions(servicetoken.WithExchanger(func(context.Context, string) (servicetoken.Exchanged, error) {
			return servicetoken.Exchanged{
				AccessToken: "sender-token", ExpiresAt: time.Now().Add(10 * time.Minute),
				Subject: Principal("storage-manager"), Audience: Audience, Scopes: []string{Scope},
			}, nil
		})))
}

func TestSendPresentsTheSenderTokenAndTheNotice(t *testing.T) {
	stub := &notificationsStub{}
	client := newClientFixture(t, stub, func() (string, error) { return "cred", nil })
	notice := Notice{Title: "t", Body: "b", Urgency: "high", SensitivityLabel: "private", IdempotencyKey: "k", ContextURL: "/asks/ask-1"}
	if err := client.Send(context.Background(), notice); err != nil {
		t.Fatal(err)
	}
	got := stub.got
	if stub.auth != "Bearer sender-token" || got.GetTitle() != "t" || got.GetBody() != "b" || got.GetUrgency() != "high" || got.GetSensitivityLabel() != "private" || got.GetIdempotencyKey() != "k" || got.GetContextUrl() != "/asks/ask-1" {
		t.Fatalf("auth=%q request=%+v", stub.auth, got)
	}
}

// The operator issues the credential while the scenario runs: the client built
// before that sends the next notice, and nothing reaches the Hub before it.
func TestSendUsesACredentialIssuedAfterTheClientWasBuilt(t *testing.T) {
	stub := &notificationsStub{}
	issued := false
	client := newClientFixture(t, stub, func() (string, error) {
		if !issued {
			return "", errors.New("no such entry")
		}
		return "cred", nil
	})
	notice := Notice{Body: "b", SensitivityLabel: "private", IdempotencyKey: "k"}
	for range 2 {
		if err := client.Send(context.Background(), notice); !errors.Is(err, servicetoken.ErrNotIssued) {
			t.Fatalf("before issuance: %v, want ErrNotIssued", err)
		}
	}
	if stub.got != nil {
		t.Fatal("nothing may reach the Hub before a credential exists")
	}
	issued = true
	if err := client.Send(context.Background(), notice); err != nil || stub.auth != "Bearer sender-token" {
		t.Fatalf("after issuance: err=%v auth=%q", err, stub.auth)
	}
}

func TestSendFailuresAreErrSendFailed(t *testing.T) {
	stub := &notificationsStub{err: connect.NewError(connect.CodeUnauthenticated, errors.New("no"))}
	client := newClientFixture(t, stub, func() (string, error) { return "cred", nil })
	if err := client.Send(context.Background(), Notice{Body: "b"}); !errors.Is(err, ErrSendFailed) {
		t.Fatalf("err = %v", err)
	}
	if err := (*Client)(nil).Send(context.Background(), Notice{}); !errors.Is(err, ErrSendFailed) {
		t.Fatalf("nil client: %v", err)
	}
}

// A notice a scenario sends as itself, from inside a request an agent run made,
// carries the sender token alone, never the run's identity.
func TestSendNeverForwardsTheIdentityOfTheRunWhoseRequestLedToIt(t *testing.T) {
	stub := &notificationsStub{}
	client := newClientFixture(t, stub, func() (string, error) { return "credential", nil })

	inbound := provenance.Middleware(provenance.VerifierFunc(func(string) (*cliutil.VerifyResult, error) {
		return &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{RunID: "run-a"}}, nil
	}))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := client.Send(r.Context(), Notice{Title: "t", Body: "b"}); err != nil {
			t.Errorf("Send: %v", err)
		}
	}))
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "run-a-credential")
	inbound.ServeHTTP(httptest.NewRecorder(), req)

	if stub.agent != "" {
		t.Fatalf("the Hub received the agent identity %q with the sender's notice", stub.agent)
	}
	if stub.auth == "" {
		t.Fatal("the notice must still present the sender token")
	}
}
