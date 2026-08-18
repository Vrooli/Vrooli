// Package owneridentity verifies the owner credentials issued by
// scenario-authenticator. It is intentionally transport-independent so
// control-plane scenarios can resolve the same subject and explicit scope
// ceiling without copying authenticator key-handling code.
package owneridentity

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	DefaultAuthScenario = "scenario-authenticator"
	ExpectedIssuer      = "scenario-authenticator"
	ExpectedAudience    = "scenario-authenticator:default"
)

var (
	ErrUnauthenticated = errors.New("owner identity is unauthenticated")
	ErrUnavailable     = errors.New("owner identity provider unavailable")
)

// Identity is the verified owner account used to seed an agent run.
type Identity struct {
	Subject   string
	Email     string
	Scopes    []string
	ExpiresAt time.Time
}

// URLResolver resolves a scenario API URL by its lifecycle-managed name.
type URLResolver interface {
	ResolveScenarioURLDefault(context.Context, string) (string, error)
}

// Validator is the narrow seam used by agent-manager. Implementations must
// reject invalid tokens and provider outages; neither condition grants a
// default identity.
type Validator interface {
	Validate(context.Context, string) (Identity, error)
}

type Config struct {
	Resolver     URLResolver
	HTTPClient   *http.Client
	AuthScenario string
	Now          func() time.Time
	JWKSGrace    time.Duration
}

type Client struct {
	resolver URLResolver
	doer     *http.Client
	scenario string
	now      func() time.Time
	grace    time.Duration

	mu        sync.Mutex
	keys      []publicKey
	fetchedAt time.Time
}

func NewClient(cfg Config) *Client {
	doer := cfg.HTTPClient
	if doer == nil {
		doer = &http.Client{Timeout: 10 * time.Second}
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	scenario := strings.TrimSpace(cfg.AuthScenario)
	if scenario == "" {
		scenario = DefaultAuthScenario
	}
	grace := cfg.JWKSGrace
	if grace <= 0 {
		grace = 24 * time.Hour
	}
	return &Client{resolver: cfg.Resolver, doer: doer, scenario: scenario, now: now, grace: grace}
}

var _ Validator = (*Client)(nil)

func (c *Client) Validate(ctx context.Context, raw string) (Identity, error) {
	token := strings.TrimSpace(raw)
	if token == "" {
		return Identity{}, ErrUnauthenticated
	}
	header, payload, signingInput, signature, err := parse(token)
	if err != nil || header.Alg != "RS256" || strings.TrimSpace(header.Kid) == "" {
		return Identity{}, ErrUnauthenticated
	}
	keys, err := c.keysFor(ctx, false)
	if err != nil {
		return Identity{}, err
	}
	if !verify(keys, signingInput, signature, header.Kid) {
		keys, err = c.keysFor(ctx, true)
		if err != nil {
			return Identity{}, err
		}
		if !verify(keys, signingInput, signature, header.Kid) {
			return Identity{}, ErrUnauthenticated
		}
	}
	return payload.identity(c.now())
}

// Reachable performs a read-only health check against the authenticator's
// published JWKS. It deliberately bypasses the grace cache: callers use this
// to report current dependency reachability, while Validate may continue to
// use a recently fetched key set during a short provider outage.
func (c *Client) Reachable(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("%w: nil client", ErrUnavailable)
	}
	_, err := c.fetch(ctx)
	return err
}

type tokenHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
}

type tokenClaims struct {
	UserID   string   `json:"user_id"`
	Subject  string   `json:"sub"`
	Email    string   `json:"email"`
	Scopes   []string `json:"scope"`
	Issuer   string   `json:"iss"`
	Audience any      `json:"aud"`
	Exp      int64    `json:"exp"`
	Nbf      int64    `json:"nbf"`
}

func parse(raw string) (tokenHeader, tokenClaims, []byte, []byte, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return tokenHeader{}, tokenClaims{}, nil, nil, ErrUnauthenticated
	}
	decode := func(value string, target any) error {
		data, err := base64.RawURLEncoding.DecodeString(value)
		if err != nil {
			return err
		}
		return json.Unmarshal(data, target)
	}
	var header tokenHeader
	var claims tokenClaims
	if err := decode(parts[0], &header); err != nil {
		return tokenHeader{}, tokenClaims{}, nil, nil, ErrUnauthenticated
	}
	if err := decode(parts[1], &claims); err != nil {
		return tokenHeader{}, tokenClaims{}, nil, nil, ErrUnauthenticated
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return tokenHeader{}, tokenClaims{}, nil, nil, ErrUnauthenticated
	}
	return header, claims, []byte(parts[0] + "." + parts[1]), sig, nil
}

type publicKey struct {
	kid string
	key *rsa.PublicKey
}

func (c *Client) keysFor(ctx context.Context, force bool) ([]publicKey, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	if len(c.keys) > 0 && !force && now.Sub(c.fetchedAt) <= c.grace {
		return c.keys, nil
	}
	keys, err := c.fetch(ctx)
	if err != nil {
		if len(c.keys) > 0 && now.Sub(c.fetchedAt) <= c.grace {
			return c.keys, nil
		}
		return nil, err
	}
	c.keys, c.fetchedAt = keys, now
	return keys, nil
}

func (c *Client) fetch(ctx context.Context) ([]publicKey, error) {
	if c.resolver == nil {
		return nil, fmt.Errorf("%w: no scenario resolver", ErrUnavailable)
	}
	base, err := c.resolver.ResolveScenarioURLDefault(ctx, c.scenario)
	if err != nil {
		return nil, fmt.Errorf("%w: resolve authenticator: %v", ErrUnavailable, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/.well-known/jwks.json", nil)
	if err != nil {
		return nil, fmt.Errorf("%w: build JWKS request: %v", ErrUnavailable, err)
	}
	resp, err := c.doer.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: fetch JWKS: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("%w: JWKS returned %d", ErrUnavailable, resp.StatusCode)
	}
	var set struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			Alg string `json:"alg"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&set); err != nil {
		return nil, fmt.Errorf("%w: decode JWKS: %v", ErrUnavailable, err)
	}
	keys := make([]publicKey, 0, len(set.Keys))
	for _, item := range set.Keys {
		if item.Kty != "RSA" || (item.Alg != "" && item.Alg != "RS256") || item.N == "" || item.E == "" {
			continue
		}
		n, err := base64.RawURLEncoding.DecodeString(item.N)
		if err != nil {
			continue
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(item.E)
		if err != nil || len(eBytes) == 0 || len(eBytes) > 4 {
			continue
		}
		e := 0
		for _, b := range eBytes {
			e = e<<8 | int(b)
		}
		if e < 2 || e%2 == 0 {
			continue
		}
		keys = append(keys, publicKey{kid: strings.TrimSpace(item.Kid), key: &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: e}})
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("%w: JWKS contained no RSA keys", ErrUnavailable)
	}
	return keys, nil
}

func verify(keys []publicKey, input, sig []byte, kid string) bool {
	digest := sha256.Sum256(input)
	for _, key := range keys {
		if key.key == nil || key.kid != kid {
			continue
		}
		if err := rsa.VerifyPKCS1v15(key.key, crypto.SHA256, digest[:], sig); err == nil {
			return true
		}
	}
	return false
}

func (c tokenClaims) identity(now time.Time) (Identity, error) {
	subject := strings.TrimSpace(c.UserID)
	if subject == "" {
		subject = strings.TrimSpace(c.Subject)
	}
	if subject == "" || c.Issuer != ExpectedIssuer || !audienceContains(c.Audience, ExpectedAudience) {
		return Identity{}, ErrUnauthenticated
	}
	if c.Exp <= 0 || !now.Before(time.Unix(c.Exp, 0)) || (c.Nbf > 0 && now.Before(time.Unix(c.Nbf, 0))) {
		return Identity{}, ErrUnauthenticated
	}
	scopes := append([]string(nil), c.Scopes...)
	if scopes == nil {
		scopes = []string{}
	}
	return Identity{Subject: subject, Email: c.Email, Scopes: scopes, ExpiresAt: time.Unix(c.Exp, 0).UTC()}, nil
}

func audienceContains(value any, expected string) bool {
	switch v := value.(type) {
	case string:
		return v == expected
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s == expected {
				return true
			}
		}
	}
	return false
}
