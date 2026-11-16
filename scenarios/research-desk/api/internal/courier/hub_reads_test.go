package courier

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/serviceprincipal"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/askreader"

	convv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	convconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"
	notifv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications"
	notifconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications/notifications_v1connect"

	"research-desk/internal/peer"
)

// hubStub is Notification Hub as the desk reads it: it shows asks only to a
// caller presenting the desk's ask-reader token, as the real Hub does.
type hubStub struct {
	convconnect.UnimplementedConversationsServiceHandler
	notifconnect.UnimplementedNotificationsServiceHandler

	mu          sync.Mutex
	askAuth     []string
	noticeAuth  []string
	askReaderOK string
	asks        []*convv1.Ask
}

func (s *hubStub) ListAsks(_ context.Context, req *connect.Request[convv1.ListAsksRequest]) (*connect.Response[convv1.ListAsksResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	got := req.Header().Get("Authorization")
	s.askAuth = append(s.askAuth, got)
	if got != "Bearer "+s.askReaderOK {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("a verified owner, agent run or service identity is required to read asks"))
	}
	return connect.NewResponse(&convv1.ListAsksResponse{Asks: s.asks}), nil
}

func (s *hubStub) List(_ context.Context, req *connect.Request[notifv1.ListRequest]) (*connect.Response[notifv1.ListResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.noticeAuth = append(s.noticeAuth, req.Header().Get("Authorization"))
	return connect.NewResponse(&notifv1.ListResponse{}), nil
}

func deskHub(t *testing.T, stub *hubStub, tokens *askreader.Tokens) *EventHub {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(convconnect.NewConversationsServiceHandler(stub))
	mux.Handle(notifconnect.NewNotificationsServiceHandler(stub))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return &EventHub{
		Hub:       &peer.Peer{Name: "notification-hub", BaseURL: func(context.Context) (string, error) { return server.URL, nil }},
		AskReader: tokens,
	}
}

// deskTokens is the desk's ask-reader identity over a scripted credential
// authority and scenario-authenticator.
func deskTokens(credential askreader.CredentialSource) *askreader.Tokens {
	return askreader.NewTokens(Source, credential, askreader.WithExchanger(func(context.Context, string) (askreader.Exchanged, error) {
		return askreader.Exchanged{
			AccessToken: "desk-token", ExpiresAt: time.Now().Add(10 * time.Minute),
			Subject: "research-desk-ask-reader", Audience: askreader.Audience, Scopes: []string{askreader.Scope},
		}, nil
	}))
}

func deskAsk(id string) *convv1.Ask {
	return &convv1.Ask{Id: id, Source: Source, State: "pending", Question: "Keep the video tool?", TechnicalDetails: "Desk key: " + testSpec.Key}
}

func TestTheDeskReadsItsAsksBackAsItsAskReaderService(t *testing.T) {
	stub := &hubStub{askReaderOK: "desk-token", asks: []*convv1.Ask{deskAsk("a1")}}
	hub := deskHub(t, stub, deskTokens(func(context.Context) (string, error) { return "issued-credential", nil }))
	asks, err := hub.Asks(context.Background(), AskPrefix)
	if err != nil {
		t.Fatal(err)
	}
	if len(asks) != 1 || asks[0].ID != "a1" || asks[0].Key != testSpec.Key {
		t.Fatalf("asks read back = %+v", asks)
	}
	if len(stub.askAuth) != 1 || stub.askAuth[0] != "Bearer desk-token" {
		t.Fatalf("the ask read must carry the desk's ask-reader bearer, sent %v", stub.askAuth)
	}
}

func TestTheDeskNeverPresentsItsAskReaderTokenOnANoticeRead(t *testing.T) {
	stub := &hubStub{askReaderOK: "desk-token"}
	hub := deskHub(t, stub, deskTokens(func(context.Context) (string, error) { return "issued-credential", nil }))
	if _, err := hub.Notices(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(stub.noticeAuth) != 1 || stub.noticeAuth[0] != "" {
		t.Fatalf("a notice read must carry no ask-reader credential, sent %v", stub.noticeAuth)
	}
}

func TestWithoutAnIssuedCredentialTheDeskNamesTheOperatorCommandAndNeverReachesTheHub(t *testing.T) {
	stub := &hubStub{askReaderOK: "desk-token"}
	hub := deskHub(t, stub, deskTokens(func(context.Context) (string, error) { return "", errors.New("no such credential entry") }))
	_, err := hub.Asks(context.Background(), AskPrefix)
	if !errors.Is(err, askreader.ErrNotIssued) || !strings.Contains(err.Error(), serviceprincipal.IssueCommand) {
		t.Fatalf("err = %v, want the missing credential named with %s", err, serviceprincipal.IssueCommand)
	}
	if len(stub.askAuth) != 0 {
		t.Fatalf("an unidentified read must never be sent to the Hub: %v", stub.askAuth)
	}
	if refused := hub.Hub.Refused(); refused == nil || !strings.Contains(refused.Error(), serviceprincipal.IssueCommand) {
		t.Fatalf("the desk must remember why the Hub read was refused, got %v", refused)
	}
}

func TestAHubThatRefusesTheDesksTokenIsReportedAsRefusal(t *testing.T) {
	stub := &hubStub{askReaderOK: "another-token"}
	hub := deskHub(t, stub, deskTokens(func(context.Context) (string, error) { return "issued-credential", nil }))
	_, err := hub.Asks(context.Background(), AskPrefix)
	if connect.CodeOf(errors.Unwrap(err)) != connect.CodeUnauthenticated && !strings.Contains(err.Error(), "unauthenticated") {
		t.Fatalf("err = %v, want the Hub's refusal", err)
	}
	if hub.Hub.Refused() == nil {
		t.Fatal("the Hub's refusal must be remembered")
	}
}

func TestADeskWithoutAnAskReaderIdentityIsRefusedByTheHubNotGivenAWiderRoute(t *testing.T) {
	stub := &hubStub{askReaderOK: "desk-token"}
	hub := deskHub(t, stub, nil)
	if _, err := hub.Asks(context.Background(), AskPrefix); err == nil {
		t.Fatal("the Hub must refuse a read that presents no identity")
	}
	if len(stub.askAuth) != 1 || stub.askAuth[0] != "" {
		t.Fatalf("no credential is invented for the desk, sent %v", stub.askAuth)
	}
}
