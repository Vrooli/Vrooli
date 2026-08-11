package credentialclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

const (
	SubscriptionIdentity = "vrooli/lpbs-account"
	SubscriptionField    = "refresh-token"
)

var (
	ErrCredentialAuthorityUnavailable = errors.New("credential authority unavailable")
	ErrCredentialAbsent               = errors.New("subscription credential is not configured")
	ErrSubscriptionUnavailable        = errors.New("subscription authority unavailable")
	ErrSubscriptionUnauthorized       = errors.New("subscription session is unauthorized")
)

type ConsumerAccess struct {
	AccessToken string
	ExpiresAt   time.Time
}

type ConsumerSessionResolver struct {
	Credentials Client
	LPBSBaseURL string
	HTTPClient  *http.Client
	Now         func() time.Time

	mu            sync.Mutex
	refreshMu     sync.Mutex
	access        ConsumerAccess
	accessBaseURL string
}

// Resolve refreshes a shared device session when needed. The refresh token is
// durable only in the credential authority; the access token remains in this
// process memory and is discarded by Clear.
func (r *ConsumerSessionResolver) Resolve(ctx context.Context) (ConsumerAccess, error) {
	if r == nil {
		return ConsumerAccess{}, ErrCredentialAuthorityUnavailable
	}
	return r.resolve(ctx, r.LPBSBaseURL)
}

// ResolveAt resolves the shared session against an explicitly selected LPBS
// base URL. Deployments that can switch between a hosted authority and a local
// LPBS instance use this to keep refresh and entitlement requests on the same
// issuer; the credential authority remains unchanged.
func (r *ConsumerSessionResolver) ResolveAt(ctx context.Context, lpbsBaseURL string) (ConsumerAccess, error) {
	if r == nil {
		return ConsumerAccess{}, ErrCredentialAuthorityUnavailable
	}
	return r.resolve(ctx, lpbsBaseURL)
}

func (r *ConsumerSessionResolver) resolve(ctx context.Context, lpbsBaseURL string) (ConsumerAccess, error) {
	if r == nil || r.Credentials == nil {
		return ConsumerAccess{}, ErrCredentialAuthorityUnavailable
	}
	base := strings.TrimRight(strings.TrimSpace(lpbsBaseURL), "/")
	if base == "" {
		return ConsumerAccess{}, fmt.Errorf("%w: LPBS base URL is empty", ErrSubscriptionUnavailable)
	}

	// Refresh-token rotation is single-use. Serialize the complete refresh
	// transaction so concurrent requests cannot race the same credential and
	// accidentally invalidate one another's newly-issued token.
	r.refreshMu.Lock()
	defer r.refreshMu.Unlock()

	now := time.Now
	if r.Now != nil {
		now = r.Now
	}
	r.mu.Lock()
	if r.access.AccessToken != "" && r.accessBaseURL == base && now().Before(r.access.ExpiresAt.Add(-30*time.Second)) {
		access := r.access
		r.mu.Unlock()
		return access, nil
	}
	// A token is issuer-bound. Never reuse a warm token after the caller
	// switches between a hosted and local LPBS authority.
	if r.accessBaseURL != base {
		r.access = ConsumerAccess{}
		r.accessBaseURL = ""
	}
	r.mu.Unlock()

	status, err := r.Credentials.Status(ctx, SubscriptionIdentity, SubscriptionField)
	if err != nil {
		return ConsumerAccess{}, fmt.Errorf("%w: status: %v", ErrCredentialAuthorityUnavailable, err)
	}
	if status.ProviderState == "absent" || status.ProviderState == "unavailable" {
		return ConsumerAccess{}, fmt.Errorf("%w: provider state %s", ErrCredentialAuthorityUnavailable, status.ProviderState)
	}
	if !status.Configured {
		return ConsumerAccess{}, ErrCredentialAbsent
	}
	refreshToken, err := r.Credentials.Resolve(ctx, SubscriptionIdentity, SubscriptionField)
	if err != nil {
		if errors.Is(err, credentialauthority.ErrUnconfigured) {
			return ConsumerAccess{}, ErrCredentialAbsent
		}
		return ConsumerAccess{}, fmt.Errorf("%w: resolve: %v", ErrCredentialAuthorityUnavailable, err)
	}
	if strings.TrimSpace(refreshToken) == "" {
		return ConsumerAccess{}, ErrCredentialAbsent
	}

	payload, _ := json.Marshal(map[string]string{"refresh_token": refreshToken})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/auth/refresh", bytes.NewReader(payload))
	if err != nil {
		return ConsumerAccess{}, fmt.Errorf("%w: create refresh request: %v", ErrSubscriptionUnavailable, err)
	}
	request.Header.Set("Content-Type", "application/json")
	client := r.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	response, err := client.Do(request)
	if err != nil {
		return ConsumerAccess{}, fmt.Errorf("%w: refresh request: %v", ErrSubscriptionUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return ConsumerAccess{}, ErrSubscriptionUnauthorized
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return ConsumerAccess{}, fmt.Errorf("%w: status %d", ErrSubscriptionUnavailable, response.StatusCode)
	}
	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresAt    string `json:"expires_at"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return ConsumerAccess{}, fmt.Errorf("%w: decode refresh response: %v", ErrSubscriptionUnavailable, err)
	}
	expiresAt, err := time.Parse(time.RFC3339, result.ExpiresAt)
	if err != nil || !expiresAt.After(now().UTC()) || strings.TrimSpace(result.AccessToken) == "" || strings.TrimSpace(result.RefreshToken) == "" {
		return ConsumerAccess{}, fmt.Errorf("%w: refresh response is incomplete", ErrSubscriptionUnavailable)
	}
	if _, err := r.Credentials.Provision(ctx, ProvisionRequest{Identity: SubscriptionIdentity, Field: SubscriptionField, Value: result.RefreshToken}); err != nil {
		return ConsumerAccess{}, fmt.Errorf("%w: persist rotated refresh token: %v", ErrCredentialAuthorityUnavailable, err)
	}
	access := ConsumerAccess{AccessToken: result.AccessToken, ExpiresAt: expiresAt.UTC()}
	r.mu.Lock()
	r.access = access
	r.accessBaseURL = base
	r.mu.Unlock()
	return access, nil
}

func (r *ConsumerSessionResolver) Clear() {
	if r == nil {
		return
	}
	r.refreshMu.Lock()
	defer r.refreshMu.Unlock()
	r.mu.Lock()
	r.access = ConsumerAccess{}
	r.accessBaseURL = ""
	r.mu.Unlock()
}
