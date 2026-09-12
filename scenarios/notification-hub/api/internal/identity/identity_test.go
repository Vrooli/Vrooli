package identity

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/vrooli/api-core/owneridentity"
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
