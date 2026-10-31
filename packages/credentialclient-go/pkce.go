package credentialclient

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrInvalidLoopbackRedirect = errors.New("redirect URI must be an HTTP loopback address")

// PKCEChallenge is one authorization-code transaction. Keep the verifier and
// state in the caller's short-lived process memory only.
type PKCEChallenge struct {
	Verifier  string
	Challenge string
	State     string
}

// NewPKCEChallenge creates an RFC 7636 S256 challenge and CSRF state.
func NewPKCEChallenge() (PKCEChallenge, error) {
	verifier, err := randomURLToken(credentialTokenBytes)
	if err != nil {
		return PKCEChallenge{}, fmt.Errorf("generate PKCE verifier: %w", err)
	}
	state, err := randomURLToken(credentialTokenBytes)
	if err != nil {
		return PKCEChallenge{}, fmt.Errorf("generate authorization state: %w", err)
	}
	digest := sha256.Sum256([]byte(verifier))
	return PKCEChallenge{Verifier: verifier, Challenge: base64.RawURLEncoding.EncodeToString(digest[:]), State: state}, nil
}

// AuthorizationURL builds an authorization-code request. Redirects are
// loopback-only so access or refresh credentials never travel through a
// custom URI scheme owned by an arbitrary application.
func (p PKCEChallenge) AuthorizationURL(authority, clientID, redirectURI string) (string, error) {
	if strings.TrimSpace(p.Verifier) == "" || strings.TrimSpace(p.Challenge) == "" || strings.TrimSpace(p.State) == "" {
		return "", errors.New("PKCE challenge is incomplete")
	}
	if err := ValidateLoopbackRedirect(redirectURI); err != nil {
		return "", err
	}
	base, err := url.Parse(strings.TrimRight(strings.TrimSpace(authority), "/") + "/api/v1/auth/authorize")
	if err != nil || base.Scheme == "" || base.Host == "" {
		return "", errors.New("authorization authority is invalid")
	}
	query := base.Query()
	query.Set("response_type", "code")
	query.Set("client_id", clientID)
	query.Set("redirect_uri", redirectURI)
	query.Set("code_challenge", p.Challenge)
	query.Set("code_challenge_method", "S256")
	query.Set("state", p.State)
	base.RawQuery = query.Encode()
	return base.String(), nil
}

// ValidateLoopbackRedirect rejects custom schemes, non-loopback hosts, and
// redirects without an explicit port.
func ValidateLoopbackRedirect(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "http" || parsed.User != nil || parsed.Hostname() == "" || parsed.Port() == "" {
		return ErrInvalidLoopbackRedirect
	}
	ip := net.ParseIP(parsed.Hostname())
	if ip == nil || !ip.IsLoopback() {
		return ErrInvalidLoopbackRedirect
	}
	return nil
}

// ExchangeAuthorizationCode exchanges a one-time code for the same short-
// lived access and durable-refresh shape used by ConsumerSessionResolver.
func ExchangeAuthorizationCode(ctx context.Context, httpClient *http.Client, authority, clientID, redirectURI, code string, challenge PKCEChallenge) (ConsumerAccess, error) {
	access, _, err := exchangeAuthorizationCode(ctx, httpClient, authority, clientID, redirectURI, code, challenge)
	return access, err
}

// BeginAuthorizationCode starts a loopback authorization transaction without
// persisting the verifier or state.
func (r *ConsumerSessionResolver) BeginAuthorizationCode(authority, clientID, redirectURI string) (PKCEChallenge, string, error) {
	challenge, err := NewPKCEChallenge()
	if err != nil {
		return PKCEChallenge{}, "", err
	}
	authorizationURL, err := challenge.AuthorizationURL(authority, clientID, redirectURI)
	if err != nil {
		return PKCEChallenge{}, "", err
	}
	return challenge, authorizationURL, nil
}

// CompleteAuthorizationCode exchanges the callback code, persists only the
// rotated refresh credential in the credential authority, and keeps the access
// token in this process's session cache.
func (r *ConsumerSessionResolver) CompleteAuthorizationCode(ctx context.Context, authority, clientID, redirectURI, code string, challenge PKCEChallenge) (ConsumerAccess, error) {
	if r == nil || r.Credentials == nil {
		return ConsumerAccess{}, ErrCredentialAuthorityUnavailable
	}
	access, refreshToken, err := exchangeAuthorizationCode(ctx, r.HTTPClient, authority, clientID, redirectURI, code, challenge)
	if err != nil {
		return ConsumerAccess{}, err
	}
	if _, err := r.Credentials.Provision(ctx, ProvisionRequest{Identity: SubscriptionIdentity, Field: SubscriptionField, Value: refreshToken}); err != nil {
		return ConsumerAccess{}, fmt.Errorf("%w: persist authorization refresh token: %v", ErrCredentialAuthorityUnavailable, err)
	}
	r.mu.Lock()
	r.access = access
	r.accessBaseURL = strings.TrimRight(strings.TrimSpace(authority), "/")
	r.mu.Unlock()
	return access, nil
}

func exchangeAuthorizationCode(ctx context.Context, httpClient *http.Client, authority, clientID, redirectURI, code string, challenge PKCEChallenge) (ConsumerAccess, string, error) {
	if err := ValidateLoopbackRedirect(redirectURI); err != nil {
		return ConsumerAccess{}, "", err
	}
	if strings.TrimSpace(code) == "" || strings.TrimSpace(challenge.Verifier) == "" {
		return ConsumerAccess{}, "", errors.New("authorization code exchange is incomplete")
	}
	payload, _ := json.Marshal(map[string]string{
		"grant_type":    "authorization_code",
		"client_id":     clientID,
		"code":          code,
		"redirect_uri":  redirectURI,
		"code_verifier": challenge.Verifier,
	})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(authority, "/")+"/api/v1/auth/token", bytes.NewReader(payload))
	if err != nil {
		return ConsumerAccess{}, "", err
	}
	request.Header.Set("Content-Type", "application/json")
	if httpClient == nil {
		httpClient = &http.Client{Timeout: credentialHTTPTimeout} //nolint:mnd // bounded credential transport timeout
	}
	response, err := httpClient.Do(request)
	if err != nil {
		return ConsumerAccess{}, "", fmt.Errorf("authorization code exchange: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return ConsumerAccess{}, "", ErrSubscriptionUnauthorized
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return ConsumerAccess{}, "", fmt.Errorf("authorization code exchange status %d", response.StatusCode)
	}
	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresAt    string `json:"expires_at"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return ConsumerAccess{}, "", fmt.Errorf("decode authorization code response: %w", err)
	}
	expiresAt, err := time.Parse(time.RFC3339, result.ExpiresAt)
	if err != nil || result.AccessToken == "" || result.RefreshToken == "" || !expiresAt.After(time.Now().UTC()) {
		return ConsumerAccess{}, "", ErrSubscriptionUnavailable
	}
	return ConsumerAccess{AccessToken: result.AccessToken, ExpiresAt: expiresAt.UTC()}, result.RefreshToken, nil
}

func randomURLToken(size int) (string, error) {
	if size <= 0 {
		return "", errors.New("token size must be positive")
	}
	raw := make([]byte, size)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
