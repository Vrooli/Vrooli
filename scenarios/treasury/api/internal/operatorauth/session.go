package operatorauth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/vrooli/api-core/browsersession"
	coreidentity "github.com/vrooli/api-core/identity"
)

// HumanVerifier verifies a scenario-authenticator access token and returns
// only claims established by that authority.
type HumanVerifier interface {
	Verify(context.Context, string) (coreidentity.Principal, error)
}

// SessionOrStatic keeps the established operator API realm for non-browser
// clients while allowing the Treasury UI to authenticate with a same-origin
// HttpOnly browser session. A presented browser cookie always selects the
// human path; invalid cookies never fall back to a static bearer token.
type SessionOrStatic struct {
	Static          Authorizer
	Verifier        HumanVerifier
	HumanSubjects   []string
	OperatorSubject string
}

func (a SessionOrStatic) Authorize(ctx context.Context, headers http.Header) (Identity, error) {
	if headers == nil {
		return Identity{}, ErrRequired
	}
	request := &http.Request{Header: headers}
	cookie, err := request.Cookie(browsersession.AccessCookie)
	if err == nil && strings.TrimSpace(cookie.Value) != "" {
		if strings.TrimSpace(headers.Get(HeaderAgentToken)) != "" || strings.TrimSpace(headers.Get(HeaderOperatorToken)) != "" || strings.TrimSpace(headers.Get("Authorization")) != "" {
			return Identity{}, ErrDenied
		}
		if a.Verifier == nil {
			return Identity{}, ErrUnavailable
		}
		principal, verifyErr := a.Verifier.Verify(ctx, strings.TrimSpace(cookie.Value))
		if verifyErr != nil {
			return Identity{}, ErrDenied
		}
		if !principal.IsVerified() || !principal.Kind.IsHuman() || strings.TrimSpace(principal.Subject) == "" {
			return Identity{}, ErrDenied
		}
		if !containsExactSubject(a.HumanSubjects, strings.TrimSpace(principal.Subject)) || strings.TrimSpace(a.OperatorSubject) == "" {
			return Identity{}, ErrDenied
		}
		return Identity{Subject: strings.TrimSpace(a.OperatorSubject)}, nil
	}
	if errors.Is(err, http.ErrNoCookie) {
		if a.Static == nil {
			return Identity{}, ErrUnavailable
		}
		return a.Static.Authorize(ctx, headers)
	}
	return Identity{}, ErrDenied
}

func containsExactSubject(subjects []string, wanted string) bool {
	for _, subject := range subjects {
		if strings.TrimSpace(subject) == wanted {
			return true
		}
	}
	return false
}

var _ Authorizer = SessionOrStatic{}
