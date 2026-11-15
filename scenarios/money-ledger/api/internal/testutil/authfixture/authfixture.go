// Package authfixture stands in for the two identity providers Money Ledger
// trusts — scenario-authenticator and Cloudflare Access — with one signing key
// and the key endpoints each verifier fetches. New points Money Ledger's real
// provider chain (access.Authentication) at them through the same environment
// the lifecycle sets, so tests exercise the production verifiers, not fakes.
package authfixture

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/browsersession"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// Audience is Money Ledger's scenario-authenticator audience.
	Audience = "scenario-authenticator:money-ledger"
	// AccessTeam and AccessAudience identify the fixture Access application.
	AccessTeam     = "https://team.example.test"
	AccessAudience = "money-ledger-access-fixture"
	// OwnerSubject is the owner's scenario-authenticator account, and
	// OwnerAccessSubject the owner's Access identity. The fixture owner-binding
	// policy names both as the one owner.
	OwnerSubject       = "owner-account"
	OwnerAccessSubject = "owner-access-identity"
	// WriteScope is the capability every writer holds.
	WriteScope = "money-ledger:write"
	// TreasuryWriter and LandingPageWriter are the declared service writers.
	TreasuryWriter    = "treasury-money-ledger"
	LandingPageWriter = "landing-page-business-suite-money-ledger-writer"
	// OwnerEmail and OwnerPassword sign the owner's account in through the
	// fixture provider; OtherEmail and OtherPassword sign in a person the
	// owner-binding policy does not name.
	OwnerEmail    = "owner@example.test"
	OwnerPassword = "owner-password" // gitleaks:allow fixture credential for the in-memory provider
	OtherEmail    = "someone@example.test"
	OtherPassword = "someone-password" // gitleaks:allow fixture credential for the in-memory provider
	OtherSubject  = "someone-else"
)

// Identity signs tokens and assertions the configured verifiers accept.
type Identity struct {
	t   testing.TB
	key *rsa.PrivateKey
}

// New starts the key endpoints and sets the provider environment for the test.
func New(t testing.TB) *Identity {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate fixture key: %v", err)
	}
	jwk := map[string]string{"kty": "RSA", "kid": "fixture", "alg": "RS256", "use": "sig", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())}
	keys := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{jwk}})
	}))
	t.Cleanup(keys.Close)
	bindings, _ := json.Marshal([]map[string]any{
		{"source": "scenario_authenticator", "subject": OwnerSubject, "canonical_subject": OwnerSubject, "scopes": []string{"*"}},
		{"source": "cloudflare_access", "subject": OwnerAccessSubject, "canonical_subject": OwnerSubject, "scopes": []string{"*"}},
	})
	for key, value := range map[string]string{
		"VROOLI_AUTH_PROVIDERS":                 "cloudflare_access,scenario_authenticator",
		"VROOLI_AUTH_SCENARIO_JWKS_URL":         keys.URL,
		"VROOLI_AUTH_SCENARIO_AUDIENCE":         Audience,
		"VROOLI_AUTH_SCENARIO_ISSUER":           "scenario-authenticator",
		"VROOLI_CLOUDFLARE_ACCESS_TEAM_DOMAIN":  AccessTeam,
		"VROOLI_CLOUDFLARE_ACCESS_AUDIENCE":     AccessAudience,
		"VROOLI_CLOUDFLARE_ACCESS_JWKS_URL":     keys.URL,
		"VROOLI_CLOUDFLARE_ACCESS_REQUIRE_USER": "true",
		"VROOLI_AUTH_OWNER_BINDINGS":            string(bindings),
	} {
		t.Setenv(key, value)
	}
	return &Identity{t: t, key: key}
}

// Token is a scenario-authenticator token for subject of the given actor kind
// ("human", "service" or "agent").
func (i *Identity) Token(subject, kind, audience string, expiry time.Time, scopes ...string) string {
	if scopes == nil {
		scopes = []string{}
	}
	return i.sign(map[string]any{"sub": subject, "user_id": subject, "actor_kind": kind, "iss": "scenario-authenticator", "aud": audience, "scope": scopes, "exp": expiry.Unix(), "iat": time.Now().Add(-time.Minute).Unix()})
}

// Service is an hour-long service principal token in Money Ledger's audience.
func (i *Identity) Service(subject string, scopes ...string) string {
	return i.Token(subject, "service", Audience, time.Now().Add(time.Hour), scopes...)
}

// Person is an hour-long human token in Money Ledger's audience.
func (i *Identity) Person(subject string, scopes ...string) string {
	return i.Token(subject, "human", Audience, time.Now().Add(time.Hour), scopes...)
}

// Owner is the owner's own scenario-authenticator session, as the CLI's
// --local-owner exchange obtains it. Like the live owner account it carries no
// Money Ledger scope of its own; the owner binding grants it.
func (i *Identity) Owner() string { return i.Person(OwnerSubject) }

// Accounts is a scenario-authenticator stand-in for browser sign-in: it signs
// in the owner and one other person by password, for Money Ledger's resource
// only, and renews from the refresh tokens it issued.
func (i *Identity) Accounts() browsersession.Accounts { return &accounts{identity: i} }

type accounts struct{ identity *Identity }

var people = map[string]struct{ password, subject string }{
	OwnerEmail: {OwnerPassword, OwnerSubject},
	OtherEmail: {OtherPassword, OtherSubject},
}

func (a *accounts) Login(_ context.Context, r *connect.Request[accountsv1.LoginRequest]) (*connect.Response[accountsv1.LoginResponse], error) {
	person, known := people[r.Msg.GetEmail()]
	if !known || person.password != r.Msg.GetPassword() || r.Msg.GetResource() != "money-ledger" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid credentials"))
	}
	return connect.NewResponse(&accountsv1.LoginResponse{Tokens: a.issue(r.Msg.GetEmail())}), nil
}

func (a *accounts) Refresh(_ context.Context, r *connect.Request[accountsv1.RefreshRequest]) (*connect.Response[accountsv1.RefreshResponse], error) {
	email, ok := strings.CutPrefix(r.Msg.GetRefreshToken(), "refresh:")
	if _, known := people[email]; !ok || !known {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid refresh token"))
	}
	return connect.NewResponse(&accountsv1.RefreshResponse{Tokens: a.issue(email)}), nil
}

func (a *accounts) Logout(context.Context, *connect.Request[accountsv1.LogoutRequest]) (*connect.Response[accountsv1.LogoutResponse], error) {
	return connect.NewResponse(&accountsv1.LogoutResponse{}), nil
}

func (a *accounts) issue(email string) *accountsv1.TokenPair {
	expiry := time.Now().Add(time.Hour)
	subject := people[email].subject
	token := a.identity.sign(map[string]any{"sub": subject, "user_id": subject, "email": email, "actor_kind": "human", "iss": "scenario-authenticator", "aud": Audience, "scope": []string{}, "exp": expiry.Unix(), "iat": time.Now().Add(-time.Minute).Unix()})
	return &accountsv1.TokenPair{AccessToken: token, RefreshToken: "refresh:" + email, AccessTokenExpiresAt: timestamppb.New(expiry), Audience: Audience}
}

// Access is a Cloudflare Access application assertion for a signed-in user.
func (i *Identity) Access(subject, email string, expiry time.Time) string {
	return i.sign(map[string]any{"iss": AccessTeam, "aud": []string{AccessAudience}, "sub": subject, "email": email, "type": "app", "exp": expiry.Unix(), "iat": time.Now().Add(-time.Minute).Unix()})
}

// OwnerAccess is the owner's Access assertion, as the tunnel adds it.
func (i *Identity) OwnerAccess() string {
	return i.Access(OwnerAccessSubject, OwnerEmail, time.Now().Add(time.Hour))
}

func (i *Identity) sign(claims map[string]any) string {
	i.t.Helper()
	header, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": "fixture", "typ": "JWT"})
	body, _ := json.Marshal(claims)
	unsigned := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(body)
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, i.key, crypto.SHA256, digest[:])
	if err != nil {
		i.t.Fatalf("sign fixture token: %v", err)
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)
}
