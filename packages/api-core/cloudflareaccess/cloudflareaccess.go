// Package cloudflareaccess verifies Cloudflare Access application assertions
// at the origin. It owns no Cloudflare management credentials or policy APIs.
package cloudflareaccess

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
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/identity"
)

const (
	DefaultClockSkew    = 30 * time.Second
	DefaultJWKSCacheTTL = 5 * time.Minute
	DefaultTokenLimit   = 16 * 1024
	DefaultHTTPTimeout  = 5 * time.Second
	assertionHeader     = "Cf-Access-Jwt-Assertion"
)

type Config struct {
	TeamDomain   string
	Audience     string
	JWKSURL      string
	RequireUser  bool
	ClockSkew    time.Duration
	JWKSCacheTTL time.Duration
	TokenLimit   int
	Client       *http.Client
	Now          func() time.Time
}

func (c Config) Validate() error {
	if _, err := normalizeTeamDomain(c.TeamDomain); err != nil {
		return err
	}
	if strings.TrimSpace(c.Audience) == "" {
		return errors.New("cloudflare access audience is required")
	}
	if c.ClockSkew < 0 || c.ClockSkew > 10*time.Minute {
		return errors.New("cloudflare access clock skew must be between 0 and 10m")
	}
	if c.JWKSCacheTTL <= 0 || c.JWKSCacheTTL > 24*time.Hour {
		return errors.New("cloudflare access JWKS cache TTL must be between 1s and 24h")
	}
	if c.TokenLimit <= 0 || c.TokenLimit > 1<<20 {
		return errors.New("cloudflare access token limit must be between 1 and 1048576 bytes")
	}
	return nil
}

func ConfigFromEnv(getenv func(string) string) (Config, error) {
	if getenv == nil {
		getenv = func(key string) string { return "" }
	}
	parseDuration := func(key string, fallback time.Duration) (time.Duration, error) {
		raw := strings.TrimSpace(getenv(key))
		if raw == "" {
			return fallback, nil
		}
		value, err := time.ParseDuration(raw)
		if err != nil {
			return 0, fmt.Errorf("%s: invalid duration", key)
		}
		return value, nil
	}
	skew, err := parseDuration("VROOLI_CLOUDFLARE_ACCESS_CLOCK_SKEW", DefaultClockSkew)
	if err != nil {
		return Config{}, err
	}
	ttl, err := parseDuration("VROOLI_CLOUDFLARE_ACCESS_JWKS_TTL", DefaultJWKSCacheTTL)
	if err != nil {
		return Config{}, err
	}
	requireUser := true
	if raw := strings.TrimSpace(getenv("VROOLI_CLOUDFLARE_ACCESS_REQUIRE_USER")); raw != "" {
		requireUser, err = strconv.ParseBool(raw)
		if err != nil {
			return Config{}, errors.New("VROOLI_CLOUDFLARE_ACCESS_REQUIRE_USER: invalid boolean")
		}
	}
	cfg := Config{TeamDomain: getenv("VROOLI_CLOUDFLARE_ACCESS_TEAM_DOMAIN"), Audience: getenv("VROOLI_CLOUDFLARE_ACCESS_AUDIENCE"), JWKSURL: getenv("VROOLI_CLOUDFLARE_ACCESS_JWKS_URL"), RequireUser: requireUser, ClockSkew: skew, JWKSCacheTTL: ttl, TokenLimit: DefaultTokenLimit, Client: &http.Client{Timeout: DefaultHTTPTimeout}, Now: time.Now}
	return cfg, cfg.Validate()
}

type Verifier struct {
	cfg        Config
	teamDomain string
	mu         sync.RWMutex
	refreshMu  sync.Mutex
	keys       map[string]*rsa.PublicKey
	fetchedAt  time.Time
	generation uint64
}

func NewVerifier(cfg Config) (*Verifier, error) {
	// Human authority is the only supported mode for this provider. The field
	// remains in Config so runtime bindings can state the policy explicitly,
	// but a zero-value config must never silently admit service identities.
	cfg.RequireUser = true
	if cfg.ClockSkew == 0 {
		cfg.ClockSkew = DefaultClockSkew
	}
	if cfg.JWKSCacheTTL == 0 {
		cfg.JWKSCacheTTL = DefaultJWKSCacheTTL
	}
	if cfg.TokenLimit == 0 {
		cfg.TokenLimit = DefaultTokenLimit
	}
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: DefaultHTTPTimeout}
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	teamDomain, err := normalizeTeamDomain(cfg.TeamDomain)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.Audience) == "" {
		return nil, errors.New("cloudflare access audience is required")
	}
	if cfg.JWKSURL == "" {
		cfg.JWKSURL = strings.TrimRight(teamDomain, "/") + "/cdn-cgi/access/certs"
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &Verifier{cfg: cfg, teamDomain: teamDomain}, nil
}

func (v *Verifier) Source() identity.AuthSource { return identity.SourceCloudflareAccess }

func (v *Verifier) VerifyRequest(ctx context.Context, req *http.Request) (identity.Principal, error) {
	if req == nil {
		return identity.Principal{}, identity.NewFailure(identity.FailureMissing, v.Source())
	}
	if strings.TrimSpace(req.Header.Get("CF-Access-Client-Id")) != "" || strings.TrimSpace(req.Header.Get("CF-Access-Client-Secret")) != "" {
		return identity.Principal{}, identity.NewFailure(identity.FailureService, v.Source())
	}
	raw := strings.TrimSpace(req.Header.Get(assertionHeader))
	if raw == "" {
		return identity.Principal{}, identity.NewFailure(identity.FailureMissing, v.Source())
	}
	return v.Verify(ctx, raw)
}

func (v *Verifier) Verify(ctx context.Context, raw string) (identity.Principal, error) {
	if len(raw) > v.cfg.TokenLimit {
		return identity.Principal{}, identity.NewFailure(identity.FailureInvalid, v.Source())
	}
	header, claims, signature, signingInput, err := parse(raw)
	if err != nil || header.Alg != "RS256" || strings.TrimSpace(header.Kid) == "" {
		return identity.Principal{}, v.failure(identity.FailureInvalid, header, claims, false)
	}
	keys, err := v.publicKeys(ctx, false)
	if err != nil {
		return identity.Principal{}, v.failure(identity.FailureUnavailable, header, claims, false)
	}
	key := keys[header.Kid]
	if key == nil {
		keys, err = v.publicKeys(ctx, true)
		if err != nil {
			return identity.Principal{}, v.failure(identity.FailureUnavailable, header, claims, false)
		}
		key = keys[header.Kid]
	}
	if key == nil {
		return identity.Principal{}, v.failure(identity.FailureInvalid, header, claims, false)
	}
	digest := sha256.Sum256([]byte(signingInput))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature); err != nil {
		return identity.Principal{}, v.failure(identity.FailureInvalid, header, claims, false)
	}
	if claims.Issuer != v.teamDomain {
		failure := v.failure(identity.FailureInvalid, header, claims, false)
		failure.Issuer = claims.Issuer
		return identity.Principal{}, failure
	}
	audienceMatched := contains(claims.Audience, v.cfg.Audience)
	if !audienceMatched {
		failure := v.failure(identity.FailureInvalid, header, claims, false)
		failure.Issuer = claims.Issuer
		return identity.Principal{}, failure
	}
	now := v.cfg.Now().UTC()
	if claims.ExpiresAt == 0 || now.After(time.Unix(claims.ExpiresAt, 0).Add(v.cfg.ClockSkew)) {
		return identity.Principal{}, v.failure(identity.FailureExpired, header, claims, audienceMatched)
	}
	if claims.NotBefore != 0 && now.Add(v.cfg.ClockSkew).Before(time.Unix(claims.NotBefore, 0)) {
		return identity.Principal{}, v.failure(identity.FailureInvalid, header, claims, audienceMatched)
	}
	if claims.IssuedAt != 0 && now.Add(v.cfg.ClockSkew).Before(time.Unix(claims.IssuedAt, 0)) {
		return identity.Principal{}, v.failure(identity.FailureInvalid, header, claims, audienceMatched)
	}
	if claims.Type != "app" {
		return identity.Principal{}, v.failure(identity.FailureInvalid, header, claims, audienceMatched)
	}
	if claims.ServiceTokenStatus || strings.TrimSpace(claims.ServiceTokenID) != "" || strings.TrimSpace(claims.CommonName) != "" || strings.TrimSpace(claims.Subject) == "" {
		return identity.Principal{}, v.failure(identity.FailureService, header, claims, audienceMatched)
	}
	if v.cfg.RequireUser && strings.TrimSpace(claims.Email) == "" {
		return identity.Principal{}, v.failure(identity.FailureInvalid, header, claims, audienceMatched)
	}
	return identity.Principal{Kind: identity.ActorHuman, Subject: strings.TrimSpace(claims.Subject), Email: strings.TrimSpace(claims.Email), Verified: true, Source: v.Source(), Sources: []identity.AuthSource{v.Source()}, Issuer: claims.Issuer, Audience: v.cfg.Audience, ExpiresAt: time.Unix(claims.ExpiresAt, 0)}, nil
}

func (v *Verifier) failure(class identity.FailureClass, header jwtHeader, claims jwtClaims, audienceMatched bool) *identity.Failure {
	failure := identity.NewFailure(class, v.Source())
	failure.KeyID = header.Kid
	failure.Issuer = claims.Issuer
	failure.AudienceMatched = audienceMatched
	return failure
}

func normalizeTeamDomain(raw string) (string, error) {
	value := strings.TrimSpace(strings.TrimRight(raw, "/"))
	if value == "" {
		return "", errors.New("cloudflare access team domain is required")
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.Path != "" && parsed.Path != "/" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("cloudflare access team domain must be an https origin")
	}
	return "https://" + strings.ToLower(parsed.Host), nil
}

type (
	jwtHeader struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	jwtClaims struct {
		Issuer             string   `json:"iss"`
		Audience           []string `json:"aud"`
		ExpiresAt          int64    `json:"exp"`
		IssuedAt           int64    `json:"iat"`
		NotBefore          int64    `json:"nbf"`
		Type               string   `json:"type"`
		Subject            string   `json:"sub"`
		Email              string   `json:"email"`
		ServiceTokenStatus bool     `json:"service_token_status"`
		ServiceTokenID     string   `json:"service_token_id"`
		CommonName         string   `json:"common_name"`
	}
)

func parse(raw string) (jwtHeader, jwtClaims, []byte, string, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return jwtHeader{}, jwtClaims{}, nil, "", errors.New("malformed assertion")
	}
	decode := func(value string, target any) error {
		data, err := base64.RawURLEncoding.DecodeString(value)
		if err != nil {
			return err
		}
		return json.Unmarshal(data, target)
	}
	var header jwtHeader
	var rawClaims struct {
		Issuer             string          `json:"iss"`
		Audience           json.RawMessage `json:"aud"`
		ExpiresAt          int64           `json:"exp"`
		IssuedAt           int64           `json:"iat"`
		NotBefore          int64           `json:"nbf"`
		Type               string          `json:"type"`
		Subject            string          `json:"sub"`
		Email              string          `json:"email"`
		ServiceTokenStatus bool            `json:"service_token_status"`
		ServiceTokenID     string          `json:"service_token_id"`
		CommonName         string          `json:"common_name"`
	}
	if err := decode(parts[0], &header); err != nil {
		return jwtHeader{}, jwtClaims{}, nil, "", errors.New("invalid header")
	}
	if err := decode(parts[1], &rawClaims); err != nil {
		return jwtHeader{}, jwtClaims{}, nil, "", errors.New("invalid claims")
	}
	audience := []string{}
	var many []string
	if json.Unmarshal(rawClaims.Audience, &many) == nil {
		audience = many
	} else {
		var one string
		if json.Unmarshal(rawClaims.Audience, &one) == nil && one != "" {
			audience = []string{one}
		}
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return jwtHeader{}, jwtClaims{}, nil, "", errors.New("invalid signature encoding")
	}
	return header, jwtClaims{Issuer: rawClaims.Issuer, Audience: audience, ExpiresAt: rawClaims.ExpiresAt, IssuedAt: rawClaims.IssuedAt, NotBefore: rawClaims.NotBefore, Type: rawClaims.Type, Subject: rawClaims.Subject, Email: rawClaims.Email, ServiceTokenStatus: rawClaims.ServiceTokenStatus, ServiceTokenID: rawClaims.ServiceTokenID, CommonName: rawClaims.CommonName}, signature, parts[0] + "." + parts[1], nil
}

type (
	jwksResponse struct {
		Keys []jwk `json:"keys"`
	}
	jwk struct {
		Kty string `json:"kty"`
		Alg string `json:"alg"`
		Kid string `json:"kid"`
		N   string `json:"n"`
		E   string `json:"e"`
	}
)

func (v *Verifier) publicKeys(ctx context.Context, force bool) (map[string]*rsa.PublicKey, error) {
	now := v.cfg.Now().UTC()
	v.mu.RLock()
	observedGeneration := v.generation
	if !force && len(v.keys) > 0 && now.Before(v.fetchedAt.Add(v.cfg.JWKSCacheTTL)) {
		keys := v.keys
		v.mu.RUnlock()
		return keys, nil
	}
	v.mu.RUnlock()
	v.refreshMu.Lock()
	defer v.refreshMu.Unlock()
	v.mu.RLock()
	if !force && len(v.keys) > 0 && now.Before(v.fetchedAt.Add(v.cfg.JWKSCacheTTL)) {
		keys := v.keys
		v.mu.RUnlock()
		return keys, nil
	}
	if force && observedGeneration != v.generation && len(v.keys) > 0 {
		keys := v.keys
		v.mu.RUnlock()
		return keys, nil
	}
	v.mu.RUnlock()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.cfg.JWKSURL, nil)
	if err != nil {
		return nil, errors.New("unable to request cloudflare access keys")
	}
	resp, err := v.cfg.Client.Do(req)
	if err != nil {
		return nil, errors.New("unable to retrieve cloudflare access keys")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("cloudflare access key endpoint unavailable")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, errors.New("unable to read cloudflare access keys")
	}
	var set jwksResponse
	if json.Unmarshal(body, &set) != nil {
		return nil, errors.New("invalid cloudflare access key set")
	}
	keys := make(map[string]*rsa.PublicKey, len(set.Keys))
	for _, item := range set.Keys {
		if item.Kty != "RSA" || item.Alg != "RS256" || strings.TrimSpace(item.Kid) == "" {
			continue
		}
		n, nerr := base64.RawURLEncoding.DecodeString(item.N)
		e, eerr := base64.RawURLEncoding.DecodeString(item.E)
		if nerr != nil || eerr != nil {
			continue
		}
		exponent := new(big.Int).SetBytes(e).Int64()
		modulus := new(big.Int).SetBytes(n)
		if exponent <= 0 || modulus.Sign() <= 0 {
			continue
		}
		keys[item.Kid] = &rsa.PublicKey{N: modulus, E: int(exponent)}
	}
	if len(keys) == 0 {
		return nil, errors.New("cloudflare access key set has no usable keys")
	}
	v.mu.Lock()
	v.keys = keys
	v.fetchedAt = now
	v.generation++
	v.mu.Unlock()
	return keys, nil
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
