// Package identity resolves the owner subject behind an HTTP request. It
// mirrors the notification-hub seam: production always verifies the bearer
// credential issued by scenario-authenticator; the legacy subject header is
// accepted only when no verifier is wired, which keeps isolated handler tests
// useful while production fails closed.
package identity

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/vrooli/api-core/owneridentity"
)

const SubjectHeader = "X-Vrooli-Identity-Subject"

// Subject returns the verified owner subject for the request headers.
func Subject(ctx context.Context, headers http.Header, verifier owneridentity.Validator) (string, error) {
	token := strings.TrimSpace(strings.TrimPrefix(headers.Get("Authorization"), "Bearer "))
	if verifier == nil {
		if subject := strings.TrimSpace(headers.Get(SubjectHeader)); subject != "" {
			return subject, nil
		}
	}
	if token == "" || verifier == nil {
		return "", owneridentity.ErrUnauthenticated
	}
	id, err := verifier.Validate(ctx, token)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(id.Subject) == "" {
		return "", errors.New("verified identity has no subject")
	}
	return id.Subject, nil
}
