// Package operatorauth owns Treasury's fail-closed operator-realm boundary.
package operatorauth

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const (
	HeaderOperatorToken = "X-Vrooli-Operator-Token" // #nosec G101 -- header name, not a credential value.
	HeaderAgentToken    = "X-Agent-Identity-Token"  // #nosec G101 -- header name, not a credential value.
)

var (
	ErrUnavailable = errors.New("operator realm unavailable")
	ErrRequired    = errors.New("operator identity required")
	ErrDenied      = errors.New("operator identity denied")
)

type Identity struct {
	Subject string
}

type Authorizer interface {
	Authorize(context.Context, http.Header) (Identity, error)
}

type StaticToken struct {
	token string
}

func NewStaticToken(token string) (*StaticToken, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("%w: TREASURY_OPERATOR_TOKEN or API_TOKEN is required", ErrUnavailable)
	}
	return &StaticToken{token: token}, nil
}

func (a *StaticToken) Authorize(_ context.Context, headers http.Header) (Identity, error) {
	if a == nil || a.token == "" {
		return Identity{}, ErrUnavailable
	}
	if headers == nil {
		return Identity{}, ErrRequired
	}
	// Reject mixed-realm requests before considering an operator credential. A
	// caller must choose a realm; an agent token can never be used as ambient
	// authority beside a more privileged header.
	if strings.TrimSpace(headers.Get(HeaderAgentToken)) != "" {
		return Identity{}, ErrDenied
	}
	candidate := strings.TrimSpace(headers.Get(HeaderOperatorToken))
	if candidate == "" {
		authorization := strings.Fields(strings.TrimSpace(headers.Get("Authorization")))
		if len(authorization) == 2 && strings.EqualFold(authorization[0], "Bearer") {
			candidate = authorization[1]
		}
	}
	if candidate == "" {
		return Identity{}, ErrRequired
	}
	if subtle.ConstantTimeCompare([]byte(candidate), []byte(a.token)) != 1 {
		return Identity{}, ErrDenied
	}
	return Identity{Subject: "local-operator"}, nil
}

type Unavailable struct{ Cause error }

func (u Unavailable) Authorize(context.Context, http.Header) (Identity, error) {
	if u.Cause == nil {
		return Identity{}, ErrUnavailable
	}
	return Identity{}, fmt.Errorf("%w: %v", ErrUnavailable, u.Cause)
}

var (
	_ Authorizer = (*StaticToken)(nil)
	_ Authorizer = Unavailable{}
)
