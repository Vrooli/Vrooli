package access

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	authenticatorScenario = "scenario-authenticator"
	authenticatorAudience = "scenario-authenticator:default"
)

type URLResolver interface {
	ResolveScenarioURLDefault(context.Context, string) (string, error)
}

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type JWKSValidator struct {
	resolver URLResolver
	doer     HTTPDoer
	now      func() time.Time

	mu   sync.Mutex
	keys []*rsa.PublicKey
}

func NewJWKSValidator(resolver URLResolver, doer HTTPDoer) *JWKSValidator {
	if doer == nil {
		doer = &http.Client{Timeout: 10 * time.Second}
	}
	return &JWKSValidator{resolver: resolver, doer: doer, now: time.Now}
}

func (v *JWKSValidator) Validate(ctx context.Context, token string) (Identity, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return Identity{}, ErrUnauthenticated
	}
	signingInput, signature, claims, err := parseToken(token)
	if err != nil {
		return Identity{}, ErrUnauthenticated
	}
	keys, err := v.signingKeys(ctx, false)
	if err != nil {
		return Identity{}, err
	}
	if !verifyAny(keys, signingInput, signature) {
		keys, err = v.signingKeys(ctx, true)
		if err != nil {
			return Identity{}, err
		}
		if !verifyAny(keys, signingInput, signature) {
			return Identity{}, ErrUnauthenticated
		}
	}
	return claims.identity(v.now())
}

func (v *JWKSValidator) signingKeys(ctx context.Context, refresh bool) ([]*rsa.PublicKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if len(v.keys) > 0 && !refresh {
		return v.keys, nil
	}
	keys, err := v.fetchKeys(ctx)
	if err != nil {
		return nil, err
	}
	v.keys = keys
	return keys, nil
}

func (v *JWKSValidator) fetchKeys(ctx context.Context) ([]*rsa.PublicKey, error) {
	if v.resolver == nil {
		return nil, ErrUnavailable
	}
	baseURL, err := v.resolver.ResolveScenarioURLDefault(ctx, authenticatorScenario)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/.well-known/jwks.json", nil) //nolint:gosec // lifecycle discovery, never caller input
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	resp, err := v.doer.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
		return nil, fmt.Errorf("%w: jwks returned %d", ErrUnavailable, resp.StatusCode)
	}
	var set struct {
		Keys []struct {
			KTY string `json:"kty"`
			ALG string `json:"alg"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&set); err != nil {
		return nil, fmt.Errorf("%w: decode jwks: %v", ErrUnavailable, err)
	}
	keys := make([]*rsa.PublicKey, 0, len(set.Keys))
	for _, item := range set.Keys {
		if item.KTY != "RSA" || (item.ALG != "" && item.ALG != "RS256") {
			continue
		}
		key, keyErr := rsaKey(item.N, item.E)
		if keyErr == nil {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("%w: jwks contains no RS256 key", ErrUnavailable)
	}
	return keys, nil
}

type tokenClaims struct {
	Subject   string      `json:"user_id"`
	Roles     []string    `json:"roles"`
	Scopes    stringSlice `json:"scope"`
	Issuer    string      `json:"iss"`
	Audience  stringSlice `json:"aud"`
	Expires   int64       `json:"exp"`
	NotBefore int64       `json:"nbf"`
}

type stringSlice []string

func (s *stringSlice) UnmarshalJSON(data []byte) error {
	var one string
	if err := json.Unmarshal(data, &one); err == nil {
		*s = stringSlice{one}
		return nil
	}
	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return err
	}
	*s = many
	return nil
}

func (s stringSlice) contains(value string) bool {
	for _, candidate := range s {
		if candidate == value {
			return true
		}
	}
	return false
}

func (c tokenClaims) identity(now time.Time) (Identity, error) {
	if strings.TrimSpace(c.Subject) == "" || c.Issuer != authenticatorScenario || !c.Audience.contains(authenticatorAudience) {
		return Identity{}, ErrUnauthenticated
	}
	if c.Expires <= 0 || !now.Before(time.Unix(c.Expires, 0)) {
		return Identity{}, ErrUnauthenticated
	}
	if c.NotBefore > 0 && now.Before(time.Unix(c.NotBefore, 0)) {
		return Identity{}, ErrUnauthenticated
	}
	return Identity{Subject: c.Subject, Roles: append([]string(nil), c.Roles...), Scopes: append([]string(nil), c.Scopes...)}, nil
}

func parseToken(token string) (string, []byte, tokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", nil, tokenClaims{}, ErrUnauthenticated
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", nil, tokenClaims{}, err
	}
	var header struct {
		ALG string `json:"alg"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil || header.ALG != "RS256" {
		return "", nil, tokenClaims{}, ErrUnauthenticated
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", nil, tokenClaims{}, err
	}
	var claims tokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", nil, tokenClaims{}, err
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", nil, tokenClaims{}, err
	}
	return parts[0] + "." + parts[1], signature, claims, nil
}

func verifyAny(keys []*rsa.PublicKey, input string, signature []byte) bool {
	digest := sha256.Sum256([]byte(input))
	for _, key := range keys {
		if rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature) == nil {
			return true
		}
	}
	return false
}

func rsaKey(modulus, exponent string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(modulus)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(exponent)
	if err != nil || len(eBytes) == 0 || len(eBytes) > 4 {
		return nil, fmt.Errorf("invalid RSA exponent")
	}
	e := 0
	for _, value := range eBytes {
		e = e<<8 | int(value)
	}
	if e < 3 {
		return nil, fmt.Errorf("invalid RSA exponent")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: e}, nil
}
