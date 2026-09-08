// Package entitlementclient verifies LPBS signed entitlement leases and
// provides a cache-first client for untrusted scenario processes.
package entitlementclient

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/consumeridentity"
)

const entitlementHTTPTimeout = 15 * time.Second

var (
	ErrLeaseMalformed       = errors.New("entitlement lease is malformed")
	ErrLeaseSignature       = errors.New("entitlement lease signature is invalid")
	ErrLeaseUnknownKey      = errors.New("entitlement lease key is unpublished")
	ErrLeaseExpired         = errors.New("entitlement lease has expired")
	ErrLeaseUnavailable     = errors.New("entitlement service is unavailable")
	ErrLeaseUnauthorized    = errors.New("entitlement request was unauthorized")
	ErrLeaseIdentityMissing = errors.New("entitlement identity is missing")
	ErrLeaseBinding         = errors.New("entitlement lease binding does not match")
)

const algorithm = "RS256"

// Limit is the server-authoritative limit carried in a lease.
type Limit struct {
	Key       string `json:"key"`
	Value     int64  `json:"value"`
	BundleKey string `json:"bundle_key,omitempty"`
}

// Payload is the signed, time-boxed entitlement contract.
type Payload struct {
	UserIdentity      string    `json:"user_identity"`
	BusinessAccountID string    `json:"business_account_id,omitempty"`
	LinkID            string    `json:"link_id,omitempty"`
	InstallationID    string    `json:"installation_id,omitempty"`
	Resource          string    `json:"resource,omitempty"`
	Audience          string    `json:"audience,omitempty"`
	Scopes            []string  `json:"scopes,omitempty"`
	Status            string    `json:"status"`
	PlanTier          string    `json:"plan_tier,omitempty"`
	PlanRank          int32     `json:"plan_rank"`
	PriceID           string    `json:"price_id,omitempty"`
	Features          []string  `json:"features,omitempty"`
	Limits            []Limit   `json:"limits,omitempty"`
	NotAfter          time.Time `json:"not_after"`
	BillingCycleStart int       `json:"billing_cycle_start,omitempty"`
	Credits           any       `json:"credits,omitempty"`
	Subscription      any       `json:"subscription,omitempty"`
}

// LeaseBinding identifies the desktop context in which a lease may be used.
// Empty fields are intentionally not checked so existing callers that only
// need signature and expiry verification remain compatible. Desktop callers
// should provide every field they know and the exact requested scope set.
type LeaseBinding struct {
	BusinessAccountID string
	InstallationID    string
	Resource          string
	Audience          string
	LinkID            string
	Scopes            []string
}

type tokenHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
	KeyID     string `json:"kid"`
}

// Sign creates a compact RS256 lease. The private key is accepted only on the
// trusted LPBS side; relying scenarios only use Verify with the published JWKS.
func Sign(payload Payload, keyID string, key *rsa.PrivateKey) (string, error) {
	if key == nil || key.N == nil || key.N.BitLen() < 2048 || strings.TrimSpace(keyID) == "" {
		return "", fmt.Errorf("lease signing key is invalid")
	}
	if payload.NotAfter.IsZero() || !payload.NotAfter.After(time.Now().UTC()) {
		return "", fmt.Errorf("lease not_after must be in the future")
	}
	header, err := json.Marshal(tokenHeader{Algorithm: algorithm, Type: "ENTITLEMENT_LEASE", KeyID: keyID})
	if err != nil {
		return "", fmt.Errorf("encode lease header: %w", err)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode lease payload: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(body)
	digest := sha256.Sum256([]byte(encoded))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("sign lease: %w", err)
	}
	return encoded + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

// Verify validates signature, key publication, and the hard expiry boundary.
func Verify(token string, keys *consumeridentity.KeySet, now time.Time) (Payload, error) {
	return VerifyFor(token, keys, now, LeaseBinding{})
}

// VerifyFor validates a lease and, when supplied, binds it to the expected
// desktop installation, resource, audience, link, and exact scope set.
func VerifyFor(token string, keys *consumeridentity.KeySet, now time.Time, binding LeaseBinding) (Payload, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || strings.TrimSpace(token) == "" {
		return Payload{}, ErrLeaseMalformed
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Payload{}, ErrLeaseMalformed
	}
	bodyBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Payload{}, ErrLeaseMalformed
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Payload{}, ErrLeaseMalformed
	}
	var header tokenHeader
	if json.Unmarshal(headerBytes, &header) != nil || header.Algorithm != algorithm || header.Type != "ENTITLEMENT_LEASE" || header.KeyID == "" {
		return Payload{}, ErrLeaseMalformed
	}
	if keys == nil || keys.Keys == nil || keys.Keys[header.KeyID] == nil {
		return Payload{}, ErrLeaseUnknownKey
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(keys.Keys[header.KeyID], crypto.SHA256, digest[:], signature); err != nil {
		return Payload{}, ErrLeaseSignature
	}
	var payload Payload
	if json.Unmarshal(bodyBytes, &payload) != nil || strings.TrimSpace(payload.UserIdentity) == "" || payload.NotAfter.IsZero() {
		return Payload{}, ErrLeaseMalformed
	}
	if now.IsZero() {
		now = time.Now()
	}
	if !now.UTC().Before(payload.NotAfter.UTC()) {
		return Payload{}, ErrLeaseExpired
	}
	if err := validateLeaseBinding(payload, binding); err != nil {
		return Payload{}, err
	}
	return payload, nil
}

func validateLeaseBinding(payload Payload, binding LeaseBinding) error {
	checks := []struct {
		name     string
		expected string
		actual   string
	}{
		{"business account", binding.BusinessAccountID, payload.BusinessAccountID},
		{"installation", binding.InstallationID, payload.InstallationID},
		{"resource", binding.Resource, payload.Resource},
		{"audience", binding.Audience, payload.Audience},
		{"link", binding.LinkID, payload.LinkID},
	}
	for _, check := range checks {
		if strings.TrimSpace(check.expected) != "" && check.expected != check.actual {
			return fmt.Errorf("%w: %s", ErrLeaseBinding, check.name)
		}
	}
	if len(binding.Scopes) > 0 && !sameScopeSet(payload.Scopes, binding.Scopes) {
		return fmt.Errorf("%w: scopes", ErrLeaseBinding)
	}
	return nil
}

func sameScopeSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	seen := make(map[string]struct{}, len(left))
	for _, scope := range left {
		if _, exists := seen[scope]; exists {
			return false
		}
		seen[scope] = struct{}{}
	}
	for _, scope := range right {
		if _, exists := seen[scope]; !exists {
			return false
		}
	}
	return true
}

// AccessTokenResolver supplies the short-lived consumer token from the shared
// credential session. Implementations must never persist the access token.
type AccessTokenResolver interface {
	Resolve(ctx context.Context, baseURL string) (string, error)
}

// Client fetches and verifies leases, serving a valid cached lease during a
// network outage until its signed not_after boundary.
type Client struct {
	BaseURL       string
	HTTPClient    *http.Client
	ResolveAccess func(context.Context, string) (string, error)
	Now           func() time.Time

	mu    sync.RWMutex
	cache map[string]Payload
	keys  *consumeridentity.KeySet
}

func NewClient(baseURL string, resolve func(context.Context, string) (string, error), httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: entitlementHTTPTimeout} //nolint:mnd // bounded entitlement transport timeout
	}
	return &Client{BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"), ResolveAccess: resolve, HTTPClient: httpClient, Now: time.Now, cache: make(map[string]Payload)}
}

// Get returns a locally verified lease where possible, and refreshes it when
// absent or expired. A valid cached lease is never discarded merely because a
// refresh request failed.
func (c *Client) Get(ctx context.Context, identity string) (Payload, error) {
	return c.GetFor(ctx, identity, LeaseBinding{})
}

// GetFor returns a locally verified lease bound to the requested desktop
// context, refreshing it when absent, expired, or bound to another context.
func (c *Client) GetFor(ctx context.Context, identity string, binding LeaseBinding) (Payload, error) {
	identity = strings.ToLower(strings.TrimSpace(identity))
	if identity == "" {
		return Payload{}, ErrLeaseIdentityMissing
	}
	now := time.Now()
	if c.Now != nil {
		now = c.Now()
	}
	c.mu.RLock()
	cached, ok := c.cache[identity]
	c.mu.RUnlock()
	var cachedBindingErr error
	if ok && now.UTC().Before(cached.NotAfter.UTC()) {
		if err := validateLeaseBinding(cached, binding); err == nil {
			return cached, nil
		} else {
			cachedBindingErr = err
		}
	}
	if c.ResolveAccess == nil || c.BaseURL == "" {
		if cachedBindingErr != nil {
			return Payload{}, cachedBindingErr
		}
		return Payload{}, ErrLeaseUnavailable
	}
	token, err := c.ResolveAccess(ctx, c.BaseURL)
	if err != nil {
		return Payload{}, fmt.Errorf("%w: resolve access: %v", ErrLeaseUnavailable, err)
	}
	return c.getWithAccess(ctx, identity, token, now, binding)
}

// GetWithAccess verifies a lease using an already-resolved short-lived access
// token. The token is used only for this request and is never cached.
func (c *Client) GetWithAccess(ctx context.Context, identity, accessToken string) (Payload, error) {
	return c.GetWithAccessFor(ctx, identity, accessToken, LeaseBinding{})
}

// GetWithAccessFor verifies a lease fetched with an already-resolved access
// token and binds it to the requested desktop context.
func (c *Client) GetWithAccessFor(ctx context.Context, identity, accessToken string, binding LeaseBinding) (Payload, error) {
	identity = strings.ToLower(strings.TrimSpace(identity))
	if identity == "" {
		return Payload{}, ErrLeaseIdentityMissing
	}
	now := time.Now()
	if c.Now != nil {
		now = c.Now()
	}
	return c.getWithAccess(ctx, identity, accessToken, now, binding)
}

// Cached returns the verified lease already held by this client without
// attempting a network request. A lease is usable offline only until its
// signed not_after boundary; transport-independent callers must treat an
// expired result as a free-tier decision.
func (c *Client) Cached(identity string) (Payload, error) {
	return c.CachedAt(identity, time.Now().UTC())
}

// CachedAt is the deterministic form of Cached used by boundary checks and
// tests. It never refreshes or changes the cache.
func (c *Client) CachedAt(identity string, now time.Time) (Payload, error) {
	identity = strings.ToLower(strings.TrimSpace(identity))
	if identity == "" {
		return Payload{}, ErrLeaseIdentityMissing
	}
	c.mu.RLock()
	payload, ok := c.cache[identity]
	c.mu.RUnlock()
	if !ok {
		return Payload{}, ErrLeaseUnavailable
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if !now.UTC().Before(payload.NotAfter.UTC()) {
		return Payload{}, ErrLeaseExpired
	}
	return payload, nil
}

// CachedFor returns a valid cached lease only when it matches the supplied
// desktop binding. It never refreshes or changes the cache.
func (c *Client) CachedFor(identity string, binding LeaseBinding) (Payload, error) {
	return c.CachedForAt(identity, binding, time.Now().UTC())
}

// CachedForAt is the deterministic form of CachedFor used by boundary checks
// and tests.
func (c *Client) CachedForAt(identity string, binding LeaseBinding, now time.Time) (Payload, error) {
	payload, err := c.CachedAt(identity, now)
	if err != nil {
		return Payload{}, err
	}
	if err := validateLeaseBinding(payload, binding); err != nil {
		return Payload{}, err
	}
	return payload, nil
}

func (c *Client) getWithAccess(ctx context.Context, identity, accessToken string, now time.Time, binding LeaseBinding) (Payload, error) {
	if strings.TrimSpace(accessToken) == "" {
		return Payload{}, ErrLeaseUnauthorized
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/v1/entitlements?user="+url.QueryEscape(identity), nil)
	if err != nil {
		return Payload{}, fmt.Errorf("%w: create request: %v", ErrLeaseUnavailable, err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response, err := c.HTTPClient.Do(request)
	if err != nil {
		return Payload{}, fmt.Errorf("%w: request: %v", ErrLeaseUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return Payload{}, ErrLeaseUnauthorized
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Payload{}, fmt.Errorf("%w: status %d", ErrLeaseUnavailable, response.StatusCode)
	}
	var responseBody struct {
		Lease string `json:"lease"`
	}
	if err := json.NewDecoder(response.Body).Decode(&responseBody); err != nil || responseBody.Lease == "" {
		return Payload{}, fmt.Errorf("%w: malformed lease response", ErrLeaseUnavailable)
	}
	if err := c.refreshKeys(ctx); err != nil {
		return Payload{}, err
	}
	payload, err := VerifyFor(responseBody.Lease, c.keys, now, binding)
	if err != nil {
		return Payload{}, err
	}
	if !strings.EqualFold(payload.UserIdentity, identity) {
		return Payload{}, ErrLeaseUnauthorized
	}
	c.mu.Lock()
	c.cache[identity] = payload
	c.mu.Unlock()
	return payload, nil
}

func (c *Client) refreshKeys(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/.well-known/jwks.json", nil)
	if err != nil {
		return fmt.Errorf("%w: create key request: %v", ErrLeaseUnavailable, err)
	}
	response, err := c.HTTPClient.Do(request)
	if err != nil {
		return fmt.Errorf("%w: key request: %v", ErrLeaseUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("%w: key status %d", ErrLeaseUnavailable, response.StatusCode)
	}
	var raw json.RawMessage
	if err := json.NewDecoder(response.Body).Decode(&raw); err != nil || len(raw) == 0 {
		return fmt.Errorf("%w: malformed key response", ErrLeaseUnavailable)
	}
	keys, err := consumeridentity.ParseJWKS(raw)
	if err != nil {
		return fmt.Errorf("%w: parse key response: %v", ErrLeaseUnavailable, err)
	}
	c.mu.Lock()
	c.keys = keys
	c.mu.Unlock()
	return nil
}

func (c *Client) HasFeature(payload Payload, feature string) bool {
	for _, candidate := range payload.Features {
		if candidate == feature {
			return true
		}
	}
	return false
}

func (c *Client) Limit(payload Payload, key, bundle string) (int64, bool) {
	for _, limit := range payload.Limits {
		if limit.Key == key && (limit.BundleKey == "" || limit.BundleKey == bundle) {
			return limit.Value, true
		}
	}
	return 0, false
}
