package identity

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrMalformedToken   = errors.New("malformed token")
	ErrInvalidSignature = errors.New("invalid signature")
	ErrTokenExpired     = errors.New("token expired")
)

// DefaultTTL is the default token lifetime (24 hours).
const DefaultTTL = 24 * time.Hour

// GenerateToken creates a signed token string from claims using the given
// HMAC secret. The format is: base64url(json_claims) + '.' + base64url(hmac_sha256).
func GenerateToken(claims *Claims, secret []byte) (string, error) {
	if claims == nil {
		return "", ErrMalformedToken
	}
	copyClaims := *claims
	if copyClaims.Scopes == nil {
		copyClaims.Scopes = []string{}
	}
	claimsJSON, err := json.Marshal(&copyClaims)
	if err != nil {
		return "", fmt.Errorf("marshal claims: %w", err)
	}

	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)
	sig := sign([]byte(payload), secret)
	return payload + "." + sig, nil
}

// VerifyToken parses and validates a token string. Returns the claims if the
// signature is valid and the token has not expired.
func VerifyToken(token string, secret []byte) (*Claims, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return nil, ErrMalformedToken
	}
	payload, sigStr := parts[0], parts[1]

	// Verify signature.
	expected := sign([]byte(payload), secret)
	if !hmac.Equal([]byte(sigStr), []byte(expected)) {
		return nil, ErrInvalidSignature
	}

	// Decode claims.
	claimsJSON, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, ErrMalformedToken
	}

	var claims Claims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, ErrMalformedToken
	}
	if claims.Scopes == nil {
		claims.Scopes = []string{}
	}

	// Check expiry.
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, ErrTokenExpired
	}

	return &claims, nil
}

// HashToken returns the hex-encoded SHA-256 hash of the token string.
// This is stored in the database instead of the raw token.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h)
}

func sign(payload, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
