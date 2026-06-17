// Package auth is the integration boundary over the scenario-authenticator
// scenario. Device Sync Hub does not own user identity, password storage, JWT
// signing, OAuth, or 2FA — all of that stays in scenario-authenticator and is
// consumed over HTTP. This package owns exactly two things:
//
//  1. Client — a substitutable seam (the Validator interface) that validates an
//     owner's bearer token against the authenticator and revokes a device's
//     authenticator session on un-pairing.
//  2. Middleware — request-scoped extraction of the owner Identity that the
//     devices/transfer/settings handlers read to know "who is asking".
//
// Failure mode is fail-closed: in production, if the authenticator is
// unreachable or the token is invalid/expired, the request is rejected. There
// is no shipped test-mode bypass (docs/concepts/INTEGRATIONS.md). The owner's
// identity and trust live in different layers on purpose: the authenticator
// answers "is this a valid owner session"; the devices domain answers "is this
// a trusted device". A device token (issued by the devices domain) is the unit
// of device trust; the authenticator JWT is the unit of owner identity.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"device-sync-hub/internal/httpc"
)

// Identity is the owner identity resolved from a validated authenticator token.
// It is the request-scoped answer to "who is the owner making this call". The
// device layer keys every device row to OwnerID.
type Identity struct {
	OwnerID string
	Email   string
	Roles   []string
	// ExpiresAt is the underlying authenticator session expiry, surfaced so a
	// brief validation cache (Phase 3, optional Redis) never re-admits a token
	// past its own expiry.
	ExpiresAt time.Time
}

// HasRole reports whether the identity carries the named role. Owner-level
// destructive operations (clear-all) gate on this.
func (id Identity) HasRole(role string) bool {
	for _, r := range id.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// Validator is the seam the middleware depends on. Production wires *Client;
// handler/middleware tests substitute a fake so they never touch the network.
type Validator interface {
	// Validate resolves the bearer token to an owner Identity. It returns
	// ErrUnauthenticated for an absent/invalid/expired/revoked token and
	// ErrAuthUnavailable when the authenticator could not be reached (the
	// fail-closed-but-distinguishable case — callers may choose to serve a
	// brief cached validation for the latter, never the former).
	Validate(ctx context.Context, bearerToken string) (Identity, error)

	// RevokeSession asks the authenticator to drop a specific session by id
	// (remote sign-out / device revocation). A blank or unknown session id is
	// a no-op that returns nil — revocation of a device that never bound an
	// authenticator session must still succeed at the device layer.
	RevokeSession(ctx context.Context, sessionID string) error
}

// ErrUnauthenticated means the token was absent, malformed, expired, or
// revoked. Handlers translate it to Connect CodeUnauthenticated / HTTP 401.
var ErrUnauthenticated = errors.New("unauthenticated")

// ErrAuthUnavailable means the authenticator could not be reached or returned
// an unexpected status. Distinct from ErrUnauthenticated so a future cache can
// distinguish "definitely not allowed" from "couldn't check right now".
var ErrAuthUnavailable = errors.New("authenticator unavailable")

// Client is the production Validator: a thin HTTP client over
// scenario-authenticator. It carries no state beyond its dependencies, so it is
// safe to share across requests.
type Client struct {
	doer    httpc.Doer
	baseURL string
}

// Config configures the production Client.
type Config struct {
	// BaseURL is the authenticator's root, e.g. "http://localhost:15000". The
	// "/api/v1/..." paths are appended internally.
	BaseURL string
	// Doer is the outbound HTTP seam (an *http.Client with a Timeout in prod).
	Doer httpc.Doer
}

// NewClient constructs the production Validator. A nil Doer defaults to an
// http.Client with a conservative timeout so a hung authenticator can never
// pin a request open indefinitely (fail-closed includes failing *fast*).
func NewClient(cfg Config) *Client {
	doer := cfg.Doer
	if doer == nil {
		doer = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{
		doer:    doer,
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
	}
}

// Compile-time guarantee.
var _ Validator = (*Client)(nil)

// validateResponse mirrors scenario-authenticator's ValidationResponse wire
// shape (api/models/user.go). Only the fields we consume are declared.
type validateResponse struct {
	Valid     bool      `json:"valid"`
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Roles     []string  `json:"roles"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (c *Client) Validate(ctx context.Context, bearerToken string) (Identity, error) {
	token := strings.TrimSpace(bearerToken)
	if token == "" {
		return Identity{}, ErrUnauthenticated
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/auth/validate", nil) //nolint:gosec // baseURL is the operator-configured AUTH_SERVICE_URL (trusted infra config), not user input — not an SSRF sink
	if err != nil {
		return Identity{}, fmt.Errorf("build validate request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.doer.Do(req)
	if err != nil {
		return Identity{}, fmt.Errorf("%w: %v", ErrAuthUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// The authenticator answers 200 with {valid:false} for a bad token, so
		// a non-200 here is a service-level fault, not a token verdict.
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
		return Identity{}, fmt.Errorf("%w: validate returned %d", ErrAuthUnavailable, resp.StatusCode)
	}

	var vr validateResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&vr); err != nil {
		return Identity{}, fmt.Errorf("%w: decode validate response: %v", ErrAuthUnavailable, err)
	}
	if !vr.Valid || vr.UserID == "" {
		return Identity{}, ErrUnauthenticated
	}

	return Identity{
		OwnerID:   vr.UserID,
		Email:     vr.Email,
		Roles:     vr.Roles,
		ExpiresAt: vr.ExpiresAt,
	}, nil
}

func (c *Client) RevokeSession(ctx context.Context, sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/api/v1/sessions/"+sessionID, nil) //nolint:gosec // baseURL is the operator-configured AUTH_SERVICE_URL (trusted infra config), not user input — not an SSRF sink
	if err != nil {
		return fmt.Errorf("build revoke request: %w", err)
	}

	resp, err := c.doer.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAuthUnavailable, err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent, http.StatusNotFound:
		// 404: the session was already gone — revocation is idempotent.
		return nil
	default:
		return fmt.Errorf("%w: revoke returned %d", ErrAuthUnavailable, resp.StatusCode)
	}
}
