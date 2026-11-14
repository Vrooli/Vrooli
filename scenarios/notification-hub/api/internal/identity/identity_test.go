package identity

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"notification-hub/internal/hub"

	"github.com/vrooli/api-core/hubsender"
	apiidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/owneridentity"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
)

type verifierFunc func(context.Context, string) (owneridentity.Identity, error)

func (f verifierFunc) Validate(ctx context.Context, token string) (owneridentity.Identity, error) {
	return f(ctx, token)
}

// [REQ:NOTIFICA-P0-007] Authentication is verified by the shared owner
// identity seam; an arbitrary subject header is not accepted in production.
func TestSubject_RequiresVerifiedBearerIdentity(t *testing.T) {
	verifier := verifierFunc(func(_ context.Context, token string) (owneridentity.Identity, error) {
		if token != "signed-token" {
			return owneridentity.Identity{}, errors.New("invalid token")
		}
		return owneridentity.Identity{Subject: "owner-1", ExpiresAt: time.Now().Add(time.Hour)}, nil
	})
	request, _ := http.NewRequest(http.MethodPost, "http://notification-hub", nil)
	request.Header.Set("X-Vrooli-Identity-Subject", "forged")
	if _, err := Subject(context.Background(), request.Header, verifier); err == nil {
		t.Fatal("forged subject header must not bypass verifier")
	}
	request.Header.Set("Authorization", "Bearer signed-token")
	subject, err := Subject(context.Background(), request.Header, verifier)
	if err != nil || subject != "owner-1" {
		t.Fatalf("verified subject = %q, err=%v", subject, err)
	}
}

func TestSubjectAcceptsTheOwnerCookieWhenNoBearerIsSent(t *testing.T) {
	headers := http.Header{}
	headers.Add("Cookie", (&http.Cookie{Name: OwnerCookieName, Value: "cookie-token"}).String())
	if got := Token(headers); got != "cookie-token" {
		t.Fatalf("cookie token = %q", got)
	}
	headers.Set("Authorization", "Bearer bearer-token")
	if got := Token(headers); got != "bearer-token" {
		t.Fatalf("bearer token = %q, want the bearer to win", got)
	}
}

func TestProducerIdentityDoesNotGrantOwnerAuthority(t *testing.T) {
	verifier := verifierFunc(func(context.Context, string) (owneridentity.Identity, error) {
		return owneridentity.Identity{Subject: "verified-human"}, nil
	})
	headers := http.Header{}
	headers.Set(cliutil.HeaderAgentIdentityToken, "presented-run-token")
	headers.Set("Authorization", "Bearer verified-human-token")
	if _, err := ProducerFor(context.Background(), headers, verifier, "recipient"); err == nil {
		t.Fatal("unverified run header must not fall back to a valid human credential")
	}
	ctx := provenance.NewContext(context.Background(), provenance.Provenance{
		Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "verified-run",
	})
	headers.Del("Authorization")
	if _, err := Subject(ctx, headers, verifier); err == nil {
		t.Fatal("a producer run must not authorize owner-only decisions")
	}
	if _, err := ProducerFor(ctx, headers, verifier, ""); err == nil {
		t.Fatal("a producer cannot choose a recipient when none is configured")
	}
	refused := context.WithValue(ctx, ownerContextKey{}, ownerContext{refused: true})
	if _, err := ProducerFor(refused, headers, verifier, "recipient"); err == nil {
		t.Fatal("a refused owner credential cannot be rescued by producer identity")
	}
}

// Verified callers retain stable producer keys, while identityless direct
// producers share the configured recipient and the anonymous source bucket.
func TestProducerSourceKeysAndIdentitylessFallback(t *testing.T) {
	verifier := verifierFunc(func(_ context.Context, token string) (owneridentity.Identity, error) {
		if token != "owner-token" {
			return owneridentity.Identity{}, errors.New("invalid token")
		}
		return owneridentity.Identity{Subject: "owner-1"}, nil
	})
	headers := http.Header{}
	headers.Set(cliutil.HeaderAgentIdentityToken, "run-token")
	ctx := provenance.NewContext(context.Background(), provenance.Provenance{
		Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run-7",
	})
	producer, err := ProducerFor(ctx, headers, verifier, "configured-owner")
	if err != nil || producer != (Producer{Recipient: "configured-owner", Key: "run:run-7"}) {
		t.Fatalf("zero-scope verified run = %+v, err=%v", producer, err)
	}
	owner := http.Header{}
	owner.Set("Authorization", "Bearer owner-token")
	producer, err = ProducerFor(context.Background(), owner, verifier, "configured-owner")
	if err != nil || producer != (Producer{Recipient: "owner-1", Key: "owner:owner-1", Subject: "owner-1"}) {
		t.Fatalf("verified owner = %+v, err=%v", producer, err)
	}
	producer, err = ProducerForDirect(context.Background(), http.Header{}, verifier, ServiceVerifiers{}, "configured-owner")
	if err != nil || producer != (Producer{Recipient: "configured-owner", Key: "source:anonymous"}) {
		t.Fatalf("identityless producer = %+v, err=%v", producer, err)
	}
}

// A run's conversation and owner come only from its verified identity, and they
// are what its audience is made of: its conversation lineage, and, naming a
// subject key, the asks of its own owner.
func TestAVerifiedRunsProducerAndAudienceCarryItsConversationAndOwner(t *testing.T) {
	ctx := provenance.NewContext(context.Background(), provenance.Provenance{
		Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified,
		RunID: "run-7", ConversationID: " thread-1 ", Subject: " owner-9 ",
	})
	headers := http.Header{}
	headers.Set(cliutil.HeaderAgentIdentityToken, "run-token")
	producer, err := ProducerFor(ctx, headers, nil, "configured-owner")
	want := Producer{Recipient: "configured-owner", Key: "run:run-7", Conversation: "thread-1", Subject: "owner-9"}
	if err != nil || producer != want {
		t.Fatalf("producer = %+v, err=%v, want %+v", producer, err, want)
	}
	if got := producer.Audience(); got != (hub.AskAudience{CreatedBy: "run:run-7", Conversation: "thread-1", Subject: "owner-9"}) {
		t.Fatalf("run audience = %+v", got)
	}
	reader, err := ReaderFor(ctx, headers, nil, ServiceVerifiers{})
	if err != nil || reader != producer.Audience() {
		t.Fatalf("a run reads as the same audience it writes as: %+v, err=%v", reader, err)
	}
	if got := (Producer{Key: "owner:o", Subject: "o"}).Audience(); !got.Owner {
		t.Fatalf("the owner reads everything: %+v", got)
	}
	if got := (Producer{Key: anonymousSourceKey}).Audience(); got != (hub.AskAudience{}) {
		t.Fatalf("an identityless producer reads nothing: %+v", got)
	}
}

type senderVerifier map[string]apiidentity.Principal

func (v senderVerifier) Verify(_ context.Context, token string) (apiidentity.Principal, error) {
	if principal, ok := v[token]; ok {
		return principal, nil
	}
	return apiidentity.Principal{}, errors.New("invalid token")
}

func TestProducerForDirectAdmitsAVerifiedServiceSenderAsItsOwnSource(t *testing.T) {
	senders := senderVerifier{"treasury": {
		Kind: apiidentity.ActorService, Subject: "treasury-notification-hub", Verified: true,
		Audience: hubsender.Audience, Scopes: []string{hubsender.Scope},
	}}
	services := ServiceVerifiers{Senders: senders}
	bearer := func(token string) http.Header {
		h := http.Header{}
		h.Set("Authorization", "Bearer "+token)
		return h
	}
	producer, err := ProducerForDirect(context.Background(), bearer("treasury"), nil, services, "configured-owner")
	want := Producer{Recipient: "configured-owner", Key: "service:treasury", Source: "treasury"}
	if err != nil || producer != want {
		t.Fatalf("sender = %+v, err=%v, want %+v", producer, err, want)
	}
	if producer.Anonymous() || producer.Creator() != "service:treasury" {
		t.Fatalf("a sender is a verified producer: %+v", producer)
	}
	if got := producer.Audience(); got != (hub.AskAudience{CreatedBy: "service:treasury"}) {
		t.Fatalf("a sender's audience is what it sent: %+v", got)
	}

	for name, tc := range map[string]struct {
		headers   http.Header
		services  ServiceVerifiers
		recipient string
	}{
		"unknown token":      {bearer("nothing"), services, "configured-owner"},
		"no sender verifier": {bearer("treasury"), ServiceVerifiers{}, "configured-owner"},
		"no recipient":       {bearer("treasury"), services, " "},
		"agent token alongside": {func() http.Header {
			h := bearer("treasury")
			h.Set(cliutil.HeaderAgentIdentityToken, "forged")
			return h
		}(), services, "configured-owner"},
	} {
		if got, err := ProducerForDirect(context.Background(), tc.headers, nil, tc.services, tc.recipient); err == nil || got != (Producer{}) {
			t.Fatalf("%s: producer = %+v, err=%v, want a refusal and no anonymous fallback", name, got, err)
		}
	}
}

// A service token is never read as an ask reader or sender of another role: a
// sender reads only what it sent, by the key the Hub recorded for it.
func TestReaderForAdmitsASenderOnlyToItsOwnAsks(t *testing.T) {
	services := ServiceVerifiers{Senders: senderVerifier{"treasury": {
		Kind: apiidentity.ActorService, Subject: "treasury-notification-hub", Verified: true,
		Audience: hubsender.Audience, Scopes: []string{hubsender.Scope},
	}}}
	headers := http.Header{}
	headers.Set("Authorization", "Bearer treasury")
	got, err := ReaderFor(context.Background(), headers, nil, services)
	if err != nil || got != (hub.AskAudience{CreatedBy: "service:treasury"}) {
		t.Fatalf("sender reader = %+v, err=%v", got, err)
	}
	if _, err := ReaderFor(context.Background(), headers, nil, ServiceVerifiers{}); err == nil {
		t.Fatal("without a sender verifier a service bearer reads nothing")
	}
}
