package authn

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
)

// PersonalLocalProvider authenticates the bundled personal_local desktop mode.
// A loopback request must also present the runtime-owned local session token.
// Transport location and the server's OS user are deliberately not identity
// proof: any local process can otherwise make those observations true.
type PersonalLocalProvider struct {
	// TokenFile is the runtime-owned file containing the local session token.
	// The desktop runtime creates it with owner-only permissions.
	TokenFile string
	// SessionToken is an in-memory equivalent used by deterministic harnesses.
	// Production callers should use TokenFile.
	SessionToken string
	// ReadToken is injectable for deterministic tests. When nil, os.ReadFile is
	// used.
	ReadToken func(string) ([]byte, error)
	Scopes    []string
	Realm     string
}

// NewPersonalLocalProvider returns the shared personal-local provider used by
// offline desktop scenarios. Scopes are the scenario's manifest-derived
// coarse capabilities, not a universal onboarding token.
func NewPersonalLocalProvider(scopes ...string) Provider {
	return PersonalLocalProvider{Scopes: append([]string(nil), scopes...)}
}

// NewPersonalLocalProviderWithTokenFile binds personal-local authentication to
// the token created by the owning runtime. An empty path intentionally fails
// closed instead of falling back to the server process user.
func NewPersonalLocalProviderWithTokenFile(tokenFile string, scopes ...string) Provider {
	return PersonalLocalProvider{
		TokenFile: strings.TrimSpace(tokenFile),
		Scopes:    append([]string(nil), scopes...),
	}
}

// NewPersonalLocalProviderWithSessionToken is intended for in-process
// harnesses whose token source is already controlled by the test owner.
func NewPersonalLocalProviderWithSessionToken(token string, scopes ...string) Provider {
	return PersonalLocalProvider{
		SessionToken: strings.TrimSpace(token),
		Scopes:       append([]string(nil), scopes...),
	}
}

func (p PersonalLocalProvider) Source() identity.AuthSource { return identity.SourcePersonalLocal }

func (p PersonalLocalProvider) VerifyRequest(ctx context.Context, req *http.Request) (identity.Principal, error) {
	if req == nil || !isLoopbackRequest(req) {
		return identity.Principal{}, identity.NewFailure(identity.FailureInvalid, p.Source())
	}
	if provenance.FromContext(ctx).IsVerifiedAgent() {
		return identity.Principal{}, identity.NewFailure(identity.FailureInvalid, p.Source())
	}
	presented := extractLocalSessionToken(req)
	if presented == "" {
		return identity.Principal{}, identity.NewFailure(identity.FailureMissing, p.Source())
	}
	expected, err := p.expectedToken()
	if err != nil {
		return identity.Principal{}, identity.NewFailure(identity.FailureUnavailable, p.Source())
	}
	if subtle.ConstantTimeCompare([]byte(presented), []byte(expected)) != 1 {
		return identity.Principal{}, identity.NewFailure(identity.FailureInvalid, p.Source())
	}
	realm := strings.TrimSpace(p.Realm)
	if realm == "" {
		realm = "personal_local"
	}
	subjectDigest := sha256.Sum256([]byte(expected))
	return identity.Principal{
		Kind: identity.ActorHuman, Subject: fmt.Sprintf("local-session:%x", subjectDigest[:8]),
		Realm: realm, Scopes: append([]string(nil), p.Scopes...), Verified: true,
		Source: p.Source(), Sources: []identity.AuthSource{p.Source()},
	}, nil
}

func (p PersonalLocalProvider) expectedToken() (string, error) {
	if token := strings.TrimSpace(p.SessionToken); token != "" {
		return token, nil
	}
	path := strings.TrimSpace(p.TokenFile)
	if path == "" {
		return "", os.ErrNotExist
	}
	reader := p.ReadToken
	if reader == nil {
		reader = os.ReadFile
	}
	token, err := reader(path)
	if err != nil {
		return "", err
	}
	if value := strings.TrimSpace(string(token)); value != "" {
		return value, nil
	}
	return "", os.ErrInvalid
}

func extractLocalSessionToken(req *http.Request) string {
	if req == nil {
		return ""
	}
	value := strings.TrimSpace(req.Header.Get("Authorization"))
	if value == "" {
		return ""
	}
	parts := strings.SplitN(value, " ", 2)
	if len(parts) != 2 {
		return ""
	}
	scheme := strings.ToLower(strings.TrimSpace(parts[0]))
	if scheme != "bearer" && scheme != "localsession" {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func isLoopbackRequest(req *http.Request) bool {
	host := strings.TrimSpace(req.RemoteAddr)
	if host == "" {
		return false
	}
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		host = parsed
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

var _ Provider = PersonalLocalProvider{}
