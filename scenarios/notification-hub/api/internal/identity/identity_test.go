package identity

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

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
	if _, err := ProducerSubject(context.Background(), headers, verifier, "recipient"); err == nil {
		t.Fatal("unverified run header must not fall back to a valid human credential")
	}
	ctx := provenance.NewContext(context.Background(), provenance.Provenance{
		Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified,
		RunID: "verified-run", Scopes: []string{"notification-hub:write"},
	})
	headers.Del("Authorization")
	if _, err := Subject(ctx, headers, verifier); err == nil {
		t.Fatal("a producer run must not authorize owner-only decisions")
	}
	if _, err := ProducerSubject(ctx, headers, verifier, ""); err == nil {
		t.Fatal("a producer cannot choose a recipient when none is configured")
	}
	refused := context.WithValue(ctx, ownerContextKey{}, ownerContext{refused: true})
	if _, err := ProducerSubject(refused, headers, verifier, "recipient"); err == nil {
		t.Fatal("a refused owner credential cannot be rescued by producer identity")
	}
}
