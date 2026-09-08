package main

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/vrooli/api-core/discovery"
)

var errAuthenticatorTokenInvalid = errors.New("authenticator token is invalid")

type authenticatorIdentity struct {
	Subject  string
	IssuedAt time.Time
}

type authenticatorVerifier struct {
	jwksURL string
	issuer  string
	aud     string
	client  *http.Client
	resolve func(context.Context) (string, error)

	mu      sync.Mutex
	keys    map[string]*rsa.PublicKey
	refresh time.Time
}

func newAuthenticatorVerifierFromEnv() *authenticatorVerifier {
	jwksURL := strings.TrimSpace(getEnv("SECRETS_MANAGER_AUTHENTICATOR_JWKS_URL"))
	issuer := strings.TrimSpace(getEnv("SECRETS_MANAGER_AUTHENTICATOR_ISSUER"))
	if issuer == "" {
		issuer = "scenario-authenticator"
	}
	audience := strings.TrimSpace(getEnv("SECRETS_MANAGER_AUTHENTICATOR_AUDIENCE"))
	if audience == "" {
		audience = "scenario-authenticator:default"
	}
	return &authenticatorVerifier{
		jwksURL: jwksURL, issuer: issuer, aud: audience,
		client: &http.Client{Timeout: 5 * time.Second}, keys: make(map[string]*rsa.PublicKey),
		resolve: func(ctx context.Context) (string, error) {
			base, err := discovery.ResolveScenarioURLDefault(ctx, "scenario-authenticator")
			if err != nil {
				return "", err
			}
			return strings.TrimRight(base, "/") + "/.well-known/jwks.json", nil
		},
	}
}

func (v *authenticatorVerifier) Verify(ctx context.Context, raw string) (authenticatorIdentity, error) {
	if v == nil || strings.TrimSpace(raw) == "" {
		return authenticatorIdentity{}, errAuthenticatorTokenInvalid
	}
	if err := v.refreshKeys(ctx, false); err != nil {
		return authenticatorIdentity{}, fmt.Errorf("%w: JWKS unavailable", errAuthenticatorTokenInvalid)
	}
	identity, err := v.parse(raw)
	if err == nil {
		return identity, nil
	}
	if refreshErr := v.refreshKeys(ctx, true); refreshErr != nil {
		return authenticatorIdentity{}, errAuthenticatorTokenInvalid
	}
	identity, err = v.parse(raw)
	if err != nil {
		return authenticatorIdentity{}, errAuthenticatorTokenInvalid
	}
	return identity, nil
}

func (v *authenticatorVerifier) parse(raw string) (authenticatorIdentity, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodRS256 {
			return nil, errAuthenticatorTokenInvalid
		}
		kid, ok := token.Header["kid"].(string)
		if !ok || strings.TrimSpace(kid) == "" {
			return nil, errAuthenticatorTokenInvalid
		}
		v.mu.Lock()
		key := v.keys[kid]
		v.mu.Unlock()
		if key == nil {
			return nil, errAuthenticatorTokenInvalid
		}
		return key, nil
	}, jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer(v.issuer), jwt.WithAudience(v.aud))
	if err != nil || token == nil || !token.Valid {
		return authenticatorIdentity{}, errAuthenticatorTokenInvalid
	}
	subject, ok := claims["sub"].(string)
	if !ok || strings.TrimSpace(subject) == "" {
		return authenticatorIdentity{}, errAuthenticatorTokenInvalid
	}
	var issuedAt time.Time
	if rawIssuedAt, ok := claims["iat"].(float64); ok && rawIssuedAt > 0 {
		issuedAt = time.Unix(int64(rawIssuedAt), 0).UTC()
	}
	return authenticatorIdentity{Subject: subject, IssuedAt: issuedAt}, nil
}

func (v *authenticatorVerifier) refreshKeys(ctx context.Context, force bool) error {
	if strings.TrimSpace(v.jwksURL) == "" && v.resolve != nil {
		jwksURL, err := v.resolve(ctx)
		if err != nil {
			return err
		}
		v.jwksURL = strings.TrimSpace(jwksURL)
	}
	if v.jwksURL == "" {
		return errors.New("JWKS URL is not configured")
	}
	v.mu.Lock()
	if !force && len(v.keys) > 0 && time.Now().Before(v.refresh) {
		v.mu.Unlock()
		return nil
	}
	v.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return err
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS returned HTTP %d", resp.StatusCode)
	}
	var document struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			Alg string `json:"alg"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&document); err != nil {
		return err
	}
	keys := make(map[string]*rsa.PublicKey, len(document.Keys))
	for _, jwk := range document.Keys {
		if jwk.Kid == "" || jwk.Kty != "RSA" || (jwk.Alg != "" && jwk.Alg != "RS256") {
			continue
		}
		n, err := base64.RawURLEncoding.DecodeString(jwk.N)
		if err != nil || len(n) == 0 {
			continue
		}
		e, err := base64.RawURLEncoding.DecodeString(jwk.E)
		if err != nil || len(e) == 0 || len(e) > 4 {
			continue
		}
		var exponent uint32
		for _, b := range e {
			exponent = exponent<<8 | uint32(b)
		}
		if exponent == 0 {
			continue
		}
		keys[jwk.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(exponent)}
	}
	if len(keys) == 0 {
		return errors.New("JWKS contains no usable RS256 keys")
	}
	v.mu.Lock()
	v.keys = keys
	v.refresh = time.Now().Add(5 * time.Minute)
	v.mu.Unlock()
	return nil
}
