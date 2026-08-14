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

var (
	ErrLeaseMalformed       = errors.New("entitlement lease is malformed")
	ErrLeaseSignature       = errors.New("entitlement lease signature is invalid")
	ErrLeaseUnknownKey      = errors.New("entitlement lease key is unpublished")
	ErrLeaseExpired         = errors.New("entitlement lease has expired")
	ErrLeaseUnavailable     = errors.New("entitlement service is unavailable")
	ErrLeaseUnauthorized    = errors.New("entitlement request was unauthorized")
	ErrLeaseIdentityMissing = errors.New("entitlement identity is missing")
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
	return payload, nil
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
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"), ResolveAccess: resolve, HTTPClient: httpClient, Now: time.Now, cache: make(map[string]Payload)}
}

// Get returns a locally verified lease where possible, and refreshes it when
// absent or expired. A valid cached lease is never discarded merely because a
// refresh request failed.
func (c *Client) Get(ctx context.Context, identity string) (Payload, error) {
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
	if ok && now.UTC().Before(cached.NotAfter.UTC()) {
		return cached, nil
	}
	if c.ResolveAccess == nil || c.BaseURL == "" {
		return Payload{}, ErrLeaseUnavailable
	}
	token, err := c.ResolveAccess(ctx, c.BaseURL)
	if err != nil {
		return Payload{}, fmt.Errorf("%w: resolve access: %v", ErrLeaseUnavailable, err)
	}
	return c.getWithAccess(ctx, identity, token, now)
}

// GetWithAccess verifies a lease using an already-resolved short-lived access
// token. The token is used only for this request and is never cached.
func (c *Client) GetWithAccess(ctx context.Context, identity, accessToken string) (Payload, error) {
	identity = strings.ToLower(strings.TrimSpace(identity))
	if identity == "" {
		return Payload{}, ErrLeaseIdentityMissing
	}
	now := time.Now()
	if c.Now != nil {
		now = c.Now()
	}
	return c.getWithAccess(ctx, identity, accessToken, now)
}

func (c *Client) getWithAccess(ctx context.Context, identity, accessToken string, now time.Time) (Payload, error) {
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
	payload, err := Verify(responseBody.Lease, c.keys, now)
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
