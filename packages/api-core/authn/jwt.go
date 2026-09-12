package authn

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

	"github.com/vrooli/api-core/identity"
)

const (
	DefaultJWTCacheTTL   = 5 * time.Minute
	DefaultJWTStaleGrace = 24 * time.Hour
	DefaultJWTTokenLimit = 64 * 1024
)

// JWTConfig describes a provider-issued RS256 JWT contract. Provider-specific
// request headers remain in the provider adapter; this package owns the
// signature, issuer, audience, expiry, and JWKS rules.
type JWTConfig struct {
	Source   identity.AuthSource
	Issuer   string
	Audience string
	// AcceptedAudiences is a bounded migration set. Audience remains the
	// canonical value emitted in the resulting principal; accepted values are
	// verification-only and must be removed after the migration window.
	AcceptedAudiences []string
	Kind              identity.ActorKind
	JWKSURL           string
	ResolveJWKS       func(context.Context) (string, error)
	CookieName        string
	Doer              interface {
		Do(*http.Request) (*http.Response, error)
	}
	Client       *http.Client
	Now          func() time.Time
	CacheTTL     time.Duration
	StaleGrace   time.Duration
	ClockSkew    time.Duration
	TokenLimit   int
	RequireEmail bool
}

// JWTVerifier verifies request credentials locally after acquiring a JWKS. It
// never sends a presented token to the issuer. A stale key set can be used for
// a bounded outage window, but an unknown key still fails closed.
type JWTVerifier struct {
	cfg JWTConfig

	mu         sync.RWMutex
	refreshMu  sync.Mutex
	keys       map[string]*rsa.PublicKey
	fetchedAt  time.Time
	generation uint64
}

func NewJWTVerifier(cfg JWTConfig) *JWTVerifier {
	if strings.TrimSpace(string(cfg.Source)) == "" || cfg.Source == identity.SourceUnknown {
		cfg.Source = identity.SourceScenarioAuthenticator
	}
	if strings.TrimSpace(string(cfg.Kind)) == "" || cfg.Kind == identity.ActorUnknown {
		cfg.Kind = identity.ActorHuman
	}
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: 5 * time.Second}
	}
	if cfg.Doer == nil {
		cfg.Doer = cfg.Client
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.CacheTTL <= 0 {
		cfg.CacheTTL = DefaultJWTCacheTTL
	}
	if cfg.StaleGrace <= 0 {
		cfg.StaleGrace = DefaultJWTStaleGrace
	}
	if cfg.TokenLimit <= 0 {
		cfg.TokenLimit = DefaultJWTTokenLimit
	}
	return &JWTVerifier{cfg: cfg}
}

func (v *JWTVerifier) Source() identity.AuthSource {
	if v == nil {
		return identity.SourceUnknown
	}
	return v.cfg.Source
}

func (v *JWTVerifier) VerifyRequest(ctx context.Context, req *http.Request) (identity.Principal, error) {
	if req == nil {
		return identity.Principal{}, identity.NewFailure(identity.FailureMissing, v.Source())
	}
	raw := strings.TrimSpace(req.Header.Get("Authorization"))
	if strings.HasPrefix(raw, "Bearer ") {
		raw = strings.TrimSpace(strings.TrimPrefix(raw, "Bearer "))
	} else {
		raw = ""
	}
	if raw == "" && strings.TrimSpace(v.cfg.CookieName) != "" {
		if cookie, err := req.Cookie(v.cfg.CookieName); err == nil {
			raw = strings.TrimSpace(cookie.Value)
		}
	}
	if raw == "" {
		return identity.Principal{}, identity.NewFailure(identity.FailureMissing, v.Source())
	}
	return v.Verify(ctx, raw)
}

func (v *JWTVerifier) Verify(ctx context.Context, raw string) (identity.Principal, error) {
	if v == nil {
		return identity.Principal{}, identity.NewFailure(identity.FailureUnavailable, identity.SourceUnknown)
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return identity.Principal{}, identity.NewFailure(identity.FailureMissing, v.Source())
	}
	if len(raw) > v.cfg.TokenLimit {
		return identity.Principal{}, identity.NewFailure(identity.FailureInvalid, v.Source())
	}
	header, claims, signingInput, signature, err := parseJWT(raw)
	if err != nil || header.Alg != "RS256" || strings.TrimSpace(header.Kid) == "" {
		return identity.Principal{}, v.failure(identity.FailureInvalid, header, claims, false)
	}
	if strings.TrimSpace(v.cfg.Issuer) == "" || strings.TrimSpace(v.cfg.Audience) == "" {
		return identity.Principal{}, identity.NewFailure(identity.FailureUnavailable, v.Source())
	}
	keys, err := v.keysFor(ctx, false)
	if err != nil {
		return identity.Principal{}, v.failure(identity.FailureUnavailable, header, claims, false)
	}
	key := keys[header.Kid]
	if key == nil {
		keys, err = v.keysFor(ctx, true)
		if err != nil {
			return identity.Principal{}, v.failure(identity.FailureUnavailable, header, claims, false)
		}
		key = keys[header.Kid]
	}
	if key == nil {
		return identity.Principal{}, v.failure(identity.FailureInvalid, header, claims, false)
	}
	digest := sha256.Sum256(signingInput)
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature); err != nil {
		// A provider can rotate the material behind an existing kid. Refresh once
		// on a signature miss, just as we do for an unknown kid, then retry with
		// the refreshed key set.
		keys, refreshErr := v.keysFor(ctx, true)
		if refreshErr != nil {
			return identity.Principal{}, v.failure(identity.FailureUnavailable, header, claims, false)
		}
		key = keys[header.Kid]
		if key == nil || rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature) != nil {
			return identity.Principal{}, v.failure(identity.FailureInvalid, header, claims, false)
		}
	}
	if claims.Issuer != v.cfg.Issuer {
		return identity.Principal{}, v.failure(identity.FailureInvalid, header, claims, false)
	}
	audienceMatched := containsAudience(claims.Audience, v.cfg.Audience)
	if !audienceMatched {
		for _, accepted := range v.cfg.AcceptedAudiences {
			if containsAudience(claims.Audience, strings.TrimSpace(accepted)) {
				audienceMatched = true
				break
			}
		}
	}
	if !audienceMatched {
		return identity.Principal{}, v.failure(identity.FailureInvalid, header, claims, false)
	}
	now := v.cfg.Now().UTC()
	if claims.ExpiresAt <= 0 || now.After(time.Unix(claims.ExpiresAt, 0).Add(v.cfg.ClockSkew)) {
		return identity.Principal{}, v.failure(identity.FailureExpired, header, claims, audienceMatched)
	}
	if claims.NotBefore > 0 && now.Add(v.cfg.ClockSkew).Before(time.Unix(claims.NotBefore, 0)) {
		return identity.Principal{}, v.failure(identity.FailureInvalid, header, claims, audienceMatched)
	}
	if claims.IssuedAt > 0 && now.Add(v.cfg.ClockSkew).Before(time.Unix(claims.IssuedAt, 0)) {
		return identity.Principal{}, v.failure(identity.FailureInvalid, header, claims, audienceMatched)
	}
	subject := strings.TrimSpace(claims.UserID)
	if subject == "" {
		subject = strings.TrimSpace(claims.Subject)
	}
	if subject == "" {
		return identity.Principal{}, v.failure(identity.FailureInvalid, header, claims, audienceMatched)
	}
	kind := v.cfg.Kind
	if declared := strings.TrimSpace(claims.ActorKind); declared != "" {
		kind = identity.ActorKind(declared)
	}
	if kind != identity.ActorHuman && kind != identity.ActorAgent && kind != identity.ActorService {
		return identity.Principal{}, v.failure(identity.FailureInvalid, header, claims, audienceMatched)
	}
	if v.cfg.RequireEmail && strings.TrimSpace(claims.Email) == "" {
		return identity.Principal{}, v.failure(identity.FailureInvalid, header, claims, audienceMatched)
	}
	return identity.Principal{
		Kind: kind, Subject: subject, Email: strings.TrimSpace(claims.Email), Realm: strings.TrimSpace(claims.Realm),
		Roles: append([]string(nil), claims.Roles...), Scopes: append([]string(nil), claims.Scopes...), Verified: true,
		Source: v.Source(), Sources: []identity.AuthSource{v.Source()}, Issuer: claims.Issuer, Audience: v.cfg.Audience,
		ExpiresAt: time.Unix(claims.ExpiresAt, 0).UTC(),
	}, nil
}

// Reachable refreshes the JWKS without using the cache. It is for status and
// recovery surfaces, not request authorization.
func (v *JWTVerifier) Reachable(ctx context.Context) error {
	if v == nil {
		return identity.NewFailure(identity.FailureUnavailable, identity.SourceUnknown)
	}
	if _, err := v.fetch(ctx); err != nil {
		return identity.NewFailure(identity.FailureUnavailable, v.Source())
	}
	return nil
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
}

type jwtClaims struct {
	UserID    string          `json:"user_id"`
	Subject   string          `json:"sub"`
	Email     string          `json:"email"`
	Realm     string          `json:"realm"`
	ActorKind string          `json:"actor_kind"`
	Roles     []string        `json:"roles"`
	Scopes    []string        `json:"scope"`
	Issuer    string          `json:"iss"`
	Audience  json.RawMessage `json:"aud"`
	ExpiresAt int64           `json:"exp"`
	NotBefore int64           `json:"nbf"`
	IssuedAt  int64           `json:"iat"`
}

func parseJWT(raw string) (jwtHeader, jwtClaims, []byte, []byte, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return jwtHeader{}, jwtClaims{}, nil, nil, errors.New("malformed jwt")
	}
	decode := func(part string, target any) error {
		data, err := base64.RawURLEncoding.DecodeString(part)
		if err != nil {
			return err
		}
		return json.Unmarshal(data, target)
	}
	var header jwtHeader
	if err := decode(parts[0], &header); err != nil {
		return jwtHeader{}, jwtClaims{}, nil, nil, errors.New("invalid jwt header")
	}
	var rawClaims struct {
		UserID    string          `json:"user_id"`
		Subject   string          `json:"sub"`
		Email     string          `json:"email"`
		Realm     string          `json:"realm"`
		ActorKind string          `json:"actor_kind"`
		Roles     json.RawMessage `json:"roles"`
		Scope     json.RawMessage `json:"scope"`
		Issuer    string          `json:"iss"`
		Audience  json.RawMessage `json:"aud"`
		Exp       int64           `json:"exp"`
		Nbf       int64           `json:"nbf"`
		Iat       int64           `json:"iat"`
	}
	if err := decode(parts[1], &rawClaims); err != nil {
		return jwtHeader{}, jwtClaims{}, nil, nil, errors.New("invalid jwt claims")
	}
	roles, err := stringList(rawClaims.Roles, false)
	if err != nil {
		return jwtHeader{}, jwtClaims{}, nil, nil, errors.New("invalid jwt roles")
	}
	scopes, err := stringList(rawClaims.Scope, true)
	if err != nil {
		return jwtHeader{}, jwtClaims{}, nil, nil, errors.New("invalid jwt scope")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(signature) == 0 {
		return jwtHeader{}, jwtClaims{}, nil, nil, errors.New("invalid jwt signature")
	}
	claims := jwtClaims{UserID: rawClaims.UserID, Subject: rawClaims.Subject, Email: rawClaims.Email, Realm: rawClaims.Realm, ActorKind: rawClaims.ActorKind, Roles: roles, Scopes: scopes, Issuer: rawClaims.Issuer, Audience: rawClaims.Audience, ExpiresAt: rawClaims.Exp, NotBefore: rawClaims.Nbf, IssuedAt: rawClaims.Iat}
	return header, claims, []byte(parts[0] + "." + parts[1]), signature, nil
}

func stringList(raw json.RawMessage, allowSpaceSeparated bool) ([]string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var many []string
	if json.Unmarshal(raw, &many) == nil {
		return compactStrings(many), nil
	}
	if allowSpaceSeparated {
		var one string
		if json.Unmarshal(raw, &one) == nil {
			return compactStrings(strings.Fields(one)), nil
		}
	}
	return nil, errors.New("expected string list")
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func containsAudience(raw json.RawMessage, expected string) bool {
	var one string
	if json.Unmarshal(raw, &one) == nil {
		return one == expected
	}
	var many []string
	if json.Unmarshal(raw, &many) != nil {
		return false
	}
	for _, value := range many {
		if value == expected {
			return true
		}
	}
	return false
}

func (v *JWTVerifier) failure(class identity.FailureClass, header jwtHeader, claims jwtClaims, audienceMatched bool) *identity.Failure {
	failure := identity.NewFailure(class, v.Source())
	failure.KeyID = strings.TrimSpace(header.Kid)
	failure.Issuer = strings.TrimSpace(claims.Issuer)
	failure.AudienceMatched = audienceMatched
	return failure
}

func (v *JWTVerifier) keysFor(ctx context.Context, force bool) (map[string]*rsa.PublicKey, error) {
	now := v.cfg.Now().UTC()
	v.mu.RLock()
	if len(v.keys) > 0 && !force && now.Before(v.fetchedAt.Add(v.cfg.CacheTTL)) {
		keys := v.keys
		v.mu.RUnlock()
		return keys, nil
	}
	observedGeneration := v.generation
	v.mu.RUnlock()

	v.refreshMu.Lock()
	defer v.refreshMu.Unlock()
	v.mu.RLock()
	if len(v.keys) > 0 && !force && now.Before(v.fetchedAt.Add(v.cfg.CacheTTL)) {
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
	keys, err := v.fetch(ctx)
	if err != nil {
		v.mu.RLock()
		cached := v.keys
		freshEnough := len(cached) > 0 && now.Before(v.fetchedAt.Add(v.cfg.StaleGrace))
		v.mu.RUnlock()
		if freshEnough {
			return cached, nil
		}
		return nil, err
	}
	v.mu.Lock()
	v.keys, v.fetchedAt, v.generation = keys, now, v.generation+1
	v.mu.Unlock()
	return keys, nil
}

func (v *JWTVerifier) fetch(ctx context.Context) (map[string]*rsa.PublicKey, error) {
	url := strings.TrimSpace(v.cfg.JWKSURL)
	if v.cfg.ResolveJWKS != nil {
		resolved, err := v.cfg.ResolveJWKS(ctx)
		if err != nil {
			return nil, err
		}
		url = strings.TrimSpace(resolved)
	}
	if url == "" {
		return nil, errors.New("jwks url is not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := v.cfg.Doer.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("jwks endpoint returned %d", resp.StatusCode)
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
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&set); err != nil {
		return nil, err
	}
	keys := make(map[string]*rsa.PublicKey, len(set.Keys))
	for _, item := range set.Keys {
		kid := strings.TrimSpace(item.Kid)
		if item.Kty != "RSA" || kid == "" || (item.Alg != "" && item.Alg != "RS256") || item.N == "" || item.E == "" {
			continue
		}
		n, err := base64.RawURLEncoding.DecodeString(item.N)
		if err != nil || len(n) == 0 {
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
		keys[kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: e}
	}
	if len(keys) == 0 {
		return nil, errors.New("jwks contained no usable rsa keys")
	}
	return keys, nil
}
