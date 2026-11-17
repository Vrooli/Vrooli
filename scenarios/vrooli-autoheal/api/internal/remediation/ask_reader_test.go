package remediation

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/vrooli/api-core/serviceprincipal"

	"github.com/vrooli/api-core/askreader"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	"google.golang.org/protobuf/proto"
)

type staticAskReaderToken struct {
	token string
	err   error
}

func (s staticAskReaderToken) Token(context.Context) (string, error) { return s.token, s.err }

func TestVerifierRefusesToReadWithoutAServiceIdentity(t *testing.T) {
	var hubCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hubCalls.Add(1) }))
	defer server.Close()

	if _, err := NewNotificationHubAskVerifier(server.URL, nil); !errors.Is(err, ErrAskVerifierUnavailable) {
		t.Fatalf("a verifier without a token source must not exist, got %v", err)
	}
	verifier, err := NewNotificationHubAskVerifier(server.URL, askreader.NewTokens(askReaderSource, func(context.Context) (string, error) { return "", errors.New("no such entry") }))
	if err != nil {
		t.Fatalf("NewNotificationHubAskVerifier() error = %v", err)
	}
	_, err = verifier.Verify(context.Background(), "ask-1")
	if !errors.Is(err, ErrAskVerifierUnavailable) || !strings.Contains(err.Error(), serviceprincipal.IssueCommand) {
		t.Fatalf("error = %v, want unavailable naming the operator script", err)
	}
	if hubCalls.Load() != 0 {
		t.Fatalf("the Hub was called %d times without a credential; an unauthenticated read must never be sent", hubCalls.Load())
	}
}

func TestVerifierNeverTreatsAHubRefusalAsApproval(t *testing.T) {
	for name, status := range map[string]int{
		"unauthenticated": http.StatusUnauthorized,
		"not found":       http.StatusNotFound,
		"hub error":       http.StatusInternalServerError,
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
		}))
		verifier, err := NewNotificationHubAskVerifier(server.URL, staticAskReaderToken{token: "t"})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		approval, err := verifier.Verify(context.Background(), "ask-1")
		server.Close()
		if err == nil || ApprovedAsk(approval) {
			t.Fatalf("%s: approval = %+v error = %v, want a refusal", name, approval, err)
		}
	}
}

func TestVerifierReadsAskAsTheAutohealServiceAndOnlyThatAsk(t *testing.T) {
	var sawAsk string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(body)
		var request conversationv1.GetAskRequest
		_ = proto.Unmarshal(body, &request)
		sawAsk = request.GetAskId()
		w.Header().Set("Content-Type", "application/proto")
		out, _ := proto.Marshal(&conversationv1.GetAskResponse{Ask: &conversationv1.Ask{Id: "ask-9", State: "answered", Answer: "approve"}})
		_, _ = w.Write(out)
	}))
	defer server.Close()
	verifier, err := NewNotificationHubAskVerifier(server.URL, staticAskReaderToken{token: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verifier.Verify(context.Background(), "  ask-9  "); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if sawAsk != "ask-9" {
		t.Fatalf("Hub saw ask id %q, want ask-9", sawAsk)
	}
}

func TestAutohealReadsAsksWithItsOwnAskReaderCredential(t *testing.T) {
	identity, err := credentialauthority.ParseIdentity(askreader.CredentialIdentity(askReaderSource))
	if err != nil {
		t.Fatalf("the ask-reader credential address must parse: %v", err)
	}
	if got := string(identity); got != "scenario-authenticator/service-principal/vrooli-autoheal-ask-reader/notification-hub-ask-reader" {
		t.Fatalf("credential address = %q", got)
	}
	var _ AskReaderTokens = NewAskReaderTokens()
}

// The declaration the operator reads before granting anything must say exactly
// what the Hub verifies for autoheal's ask-reader principal.
func TestAutohealDeclaresItsAskReaderPrincipalExactly(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ".vrooli", filepath.Base(serviceprincipal.DeclarationFile)))
	if err != nil {
		t.Fatal(err)
	}
	declared, err := serviceprincipal.ParseDeclarations(raw)
	if err != nil {
		t.Fatal(err)
	}
	want := serviceprincipal.Declaration{Name: askreader.Principal(askReaderSource), Resource: askreader.Resource, Audience: askreader.Audience, Scopes: []string{askreader.Scope}}
	var got []serviceprincipal.Declaration
	for _, d := range declared {
		if d.Name == want.Name {
			got = append(got, d)
		}
	}
	if len(got) != 1 || !reflect.DeepEqual(got[0], want) {
		t.Fatalf("declared %+v, want exactly one %+v", declared, want)
	}
}
