package identity

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/vrooli/api-core/owneridentity"
)

type Verifier interface {
	Validate(context.Context, string) (owneridentity.Identity, error)
}

// Subject verifies the bearer credential issued by scenario-authenticator.
// The legacy subject header is accepted only when no verifier is wired, which
// keeps isolated handler tests useful while production always fails closed.
func Subject(ctx context.Context, headers http.Header, verifier Verifier) (string, error) {
	token := strings.TrimSpace(strings.TrimPrefix(headers.Get("Authorization"), "Bearer "))
	if verifier == nil {
		if subject := strings.TrimSpace(headers.Get("X-Vrooli-Identity-Subject")); subject != "" {
			return subject, nil
		}
	}
	if token == "" || verifier == nil {
		return "", owneridentity.ErrUnauthenticated
	}
	identity, err := verifier.Validate(ctx, token)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(identity.Subject) == "" {
		return "", errors.New("verified identity has no subject")
	}
	return identity.Subject, nil
}
