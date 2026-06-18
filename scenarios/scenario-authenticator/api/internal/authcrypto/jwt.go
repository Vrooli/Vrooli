package authcrypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents the JWT claims. Ported verbatim from the old
// models/claims.go: `user_id` is the FROZEN primary identity claim
// device-sync-hub pins (NOT `sub`). The embedded RegisteredClaims carry `iss`,
// `aud`, `exp`, `iat` and — additively per the contract — `sub` is mirrored to
// `user_id` for OIDC friendliness without disturbing the frozen claim.
type Claims struct {
	UserID string   `json:"user_id"`
	Email  string   `json:"email"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

// TokenInput is the subject material for minting an access token.
type TokenInput struct {
	UserID   string
	Email    string
	Roles    []string
	Audience string // realm-qualified aud; empty is rejected by Sign
}

// Signer mints and verifies RS256 access tokens over a single keypair.
type Signer struct {
	keys   *Keys
	issuer string
	expiry time.Duration
	now    func() time.Time
}

// SignerConfig configures a Signer.
type SignerConfig struct {
	// Issuer is the `iss` claim (FROZEN to realm.Issuer for the contract).
	Issuer string
	// Expiry is the access-token TTL. Zero falls back to the env/default
	// resolution (JWT_EXPIRY_MINUTES, default 60, clamped 1..1440).
	Expiry time.Duration
	// Now overrides the clock (tests). Defaults to time.Now.
	Now func() time.Time
}

// NewSigner constructs a Signer over the given keypair.
func NewSigner(keys *Keys, cfg SignerConfig) *Signer {
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	expiry := cfg.Expiry
	if expiry <= 0 {
		expiry = resolveExpiry()
	}
	return &Signer{keys: keys, issuer: cfg.Issuer, expiry: expiry, now: now}
}

// Keys exposes the underlying keypair (for JWKS publication).
func (s *Signer) Keys() *Keys { return s.keys }

// Expiry reports the configured access-token TTL.
func (s *Signer) Expiry() time.Duration { return s.expiry }

// Sign mints a signed RS256 access token for the input. CORRECTION (§8): the
// token header carries `kid` so a rotation-aware verifier can select the key.
func (s *Signer) Sign(in TokenInput) (string, error) {
	if in.UserID == "" {
		return "", fmt.Errorf("authcrypto: empty user id")
	}
	if in.Audience == "" {
		return "", fmt.Errorf("authcrypto: empty audience")
	}
	now := s.now()
	claims := &Claims{
		UserID: in.UserID,
		Email:  in.Email,
		Roles:  in.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   in.UserID, // additive mirror of user_id for OIDC friendliness
			Issuer:    s.issuer,
			Audience:  jwt.ClaimStrings{in.Audience},
			ExpiresAt: jwt.NewNumericDate(now.Add(s.expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = s.keys.kid
	return token.SignedString(s.keys.private)
}

// Validate parses and verifies a token, enforcing the RS256 method-lock
// (rejecting `none` and any HS*/EC* algorithm — algorithm-confusion defense,
// ported verbatim) and, when expectedAudience is non-empty, that the token's
// `aud` matches it. A cross-realm/aud token is rejected even with one realm
// (OT-P0-008): a misconfiguration here is a cross-tenant leak.
func (s *Signer) Validate(tokenString, expectedAudience string) (*Claims, error) {
	parser := jwt.NewParser(jwt.WithValidMethods([]string{"RS256"}), jwt.WithTimeFunc(s.now))
	token, err := parser.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.keys.public, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	if expectedAudience != "" && !hasAudience(claims.Audience, expectedAudience) {
		return nil, fmt.Errorf("audience mismatch")
	}
	return claims, nil
}

func hasAudience(auds jwt.ClaimStrings, want string) bool {
	for _, a := range auds {
		if a == want {
			return true
		}
	}
	return false
}

// resolveExpiry ports the old GenerateToken expiry resolution: 60 minutes
// default, overridable by JWT_EXPIRY_MINUTES, clamped to (0, 1440].
func resolveExpiry() time.Duration {
	minutes := 60
	if env := os.Getenv("JWT_EXPIRY_MINUTES"); env != "" {
		if m, err := strconv.Atoi(env); err == nil && m > 0 && m <= 1440 {
			minutes = m
		}
	}
	return time.Duration(minutes) * time.Minute
}

// GenerateRefreshToken returns a 32-byte hex-encoded random refresh token.
// Ported verbatim.
func GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// HashToken returns the hex SHA-256 of a token. Ported verbatim — refresh
// tokens, blacklist entries, and session-revocation lookups all key on this.
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// GenerateSecureToken returns a hex-encoded random token of length bytes.
func GenerateSecureToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
