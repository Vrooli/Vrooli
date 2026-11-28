package desktoplink

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidRequest = errors.New("invalid desktop link request")
	ErrCodeRejected   = errors.New("desktop link authorization code rejected")
	ErrLinkRevoked    = errors.New("desktop account link is revoked")
)

var identifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// Issue creates a short-lived, one-use code. The raw code is returned only to
// the already authenticated LPBS client and is never persisted or logged.
func (s *Service) Issue(ctx context.Context, req AuthorizationRequest) (string, Authorization, error) {
	if s == nil || s.repo == nil || !validAuthorizationRequest(req) {
		return "", Authorization{}, ErrInvalidRequest
	}
	if req.ExpiresAt.IsZero() {
		req.ExpiresAt = s.now().UTC().Add(2 * time.Minute)
	}
	if !req.ExpiresAt.After(s.now().UTC()) || req.ExpiresAt.After(s.now().UTC().Add(5*time.Minute)) {
		return "", Authorization{}, ErrInvalidRequest
	}
	rawCode, err := randomSecret()
	if err != nil {
		return "", Authorization{}, fmt.Errorf("generate desktop link code: %w", err)
	}
	authID, err := randomSecret()
	if err != nil {
		return "", Authorization{}, fmt.Errorf("generate desktop link id: %w", err)
	}
	authorization := Authorization{
		ID: authID, CodeHash: hashSecret(rawCode), LPBSUserID: strings.TrimSpace(req.LPBSUserID),
		BusinessAccountID: strings.TrimSpace(req.BusinessAccountID), InstallationID: strings.TrimSpace(req.InstallationID),
		Resource: strings.TrimSpace(req.Resource), Audience: strings.TrimSpace(req.Audience), Scopes: clone(req.Scopes),
		CodeChallenge: strings.TrimSpace(req.CodeChallenge), RedirectURI: strings.TrimSpace(req.RedirectURI), ExpiresAt: req.ExpiresAt.UTC(),
	}
	if err := s.repo.CreateAuthorization(ctx, authorization); err != nil {
		return "", Authorization{}, err
	}
	return rawCode, authorization, nil
}

// Redeem consumes a code atomically after checking PKCE, installation, and
// resource binding. localPrincipal must already be verified by the local
// identity authority; this service never treats an email as that identity.
func (s *Service) Redeem(ctx context.Context, code, verifier, localPrincipal, installationID, resource string) (Link, error) {
	if s == nil || s.repo == nil || strings.TrimSpace(code) == "" || strings.TrimSpace(verifier) == "" || !identifierPattern.MatchString(strings.TrimSpace(localPrincipal)) || !identifierPattern.MatchString(strings.TrimSpace(installationID)) || !identifierPattern.MatchString(strings.TrimSpace(resource)) {
		return Link{}, ErrInvalidRequest
	}
	link, err := s.repo.RedeemAuthorization(ctx, hashSecret(strings.TrimSpace(code)), strings.TrimSpace(verifier), strings.TrimSpace(localPrincipal), strings.TrimSpace(installationID), strings.TrimSpace(resource))
	if err != nil {
		return Link{}, ErrCodeRejected
	}
	return link, nil
}

func (s *Service) RevokeForLPBS(ctx context.Context, userID, installationID, resource, actor string) error {
	if s == nil || s.repo == nil || !identifierPattern.MatchString(strings.TrimSpace(userID)) || !identifierPattern.MatchString(strings.TrimSpace(installationID)) || !identifierPattern.MatchString(strings.TrimSpace(resource)) {
		return ErrInvalidRequest
	}
	_, err := s.repo.RevokeByLPBS(ctx, strings.TrimSpace(userID), strings.TrimSpace(installationID), strings.TrimSpace(resource), strings.TrimSpace(actor))
	return err
}

func (s *Service) RevokeForLocal(ctx context.Context, principal, installationID, resource, actor string) error {
	if s == nil || s.repo == nil || !identifierPattern.MatchString(strings.TrimSpace(principal)) || !identifierPattern.MatchString(strings.TrimSpace(installationID)) || !identifierPattern.MatchString(strings.TrimSpace(resource)) {
		return ErrInvalidRequest
	}
	_, err := s.repo.RevokeByLocal(ctx, strings.TrimSpace(principal), strings.TrimSpace(installationID), strings.TrimSpace(resource), strings.TrimSpace(actor))
	return err
}

func (s *Service) StatusForLocal(ctx context.Context, principal, installationID, resource string) (Link, error) {
	if s == nil || s.repo == nil || !identifierPattern.MatchString(strings.TrimSpace(principal)) || !identifierPattern.MatchString(strings.TrimSpace(installationID)) || !identifierPattern.MatchString(strings.TrimSpace(resource)) {
		return Link{}, ErrInvalidRequest
	}
	return s.repo.StatusByLocal(ctx, strings.TrimSpace(principal), strings.TrimSpace(installationID), strings.TrimSpace(resource))
}

func validAuthorizationRequest(req AuthorizationRequest) bool {
	resource := strings.TrimSpace(req.Resource)
	if !identifierPattern.MatchString(strings.TrimSpace(req.LPBSUserID)) || !identifierPattern.MatchString(strings.TrimSpace(req.BusinessAccountID)) || !identifierPattern.MatchString(strings.TrimSpace(req.InstallationID)) || !identifierPattern.MatchString(resource) || !validResourceAudience(resource, req.Audience) || !validPKCEChallenge(req.CodeChallenge) || !validLoopbackRedirect(req.RedirectURI) || len(req.Scopes) == 0 || len(req.Scopes) > 32 {
		return false
	}
	seen := make(map[string]struct{}, len(req.Scopes))
	for _, scope := range req.Scopes {
		scope = strings.TrimSpace(scope)
		if !identifierPattern.MatchString(scope) || !strings.HasPrefix(scope, resource+":") {
			return false
		}
		if _, ok := seen[scope]; ok {
			return false
		}
		seen[scope] = struct{}{}
	}
	return true
}

func validResourceAudience(resource, audience string) bool {
	resource = strings.TrimSpace(resource)
	audience = strings.TrimSpace(audience)
	return identifierPattern.MatchString(resource) && audience == "scenario:"+resource
}

func validPKCEChallenge(value string) bool {
	if len(value) < 43 || len(value) > 128 {
		return false
	}
	for _, char := range value {
		if !(char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || strings.ContainsRune("-._~", char)) {
			return false
		}
	}
	return true
}

func validLoopbackRedirect(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "http" || parsed.User != nil || parsed.Path == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	ip := net.ParseIP(parsed.Hostname())
	if ip == nil || !ip.IsLoopback() {
		return false
	}
	port, err := strconv.Atoi(parsed.Port())
	return err == nil && port > 0 && port < 65536 && parsed.Hostname() != "0.0.0.0"
}

func randomSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashSecret(value string) string {
	digest := sha256.Sum256([]byte(value))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

func clone(values []string) []string { return append([]string(nil), values...) }

func encodeScopes(scopes []string) (string, error) {
	payload, err := json.Marshal(scopes)
	return string(payload), err
}
