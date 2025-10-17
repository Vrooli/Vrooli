package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"scenario-authenticator/models"
)

func TestGenerateRefreshToken(t *testing.T) {
	token1 := GenerateRefreshToken()
	token2 := GenerateRefreshToken()

	// Should generate a token
	if token1 == "" {
		t.Error("GenerateRefreshToken() returned empty string")
	}

	// Should be hex encoded (64 chars for 32 bytes)
	if len(token1) != 64 {
		t.Errorf("GenerateRefreshToken() returned token of length %d, want 64", len(token1))
	}

	// Should be different each time
	if token1 == token2 {
		t.Error("GenerateRefreshToken() returned same token twice")
	}

	// Should only contain hex characters
	for _, c := range token1 {
		if !strings.ContainsRune("0123456789abcdef", c) {
			t.Errorf("GenerateRefreshToken() returned non-hex character: %c", c)
		}
	}
}

func TestHashToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "simple token",
			token: "test_token_123",
		},
		{
			name:  "empty token",
			token: "",
		},
		{
			name:  "long token",
			token: strings.Repeat("a", 1000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := HashToken(tt.token)
			hash2 := HashToken(tt.token)

			// Should always return same hash for same input
			if hash1 != hash2 {
				t.Error("HashToken() returned different hashes for same input")
			}

			// Should be 64 chars (SHA256 hex)
			if len(hash1) != 64 {
				t.Errorf("HashToken() returned hash of length %d, want 64", len(hash1))
			}

			// Should only contain hex characters
			for _, c := range hash1 {
				if !strings.ContainsRune("0123456789abcdef", c) {
					t.Errorf("HashToken() returned non-hex character: %c", c)
				}
			}
		})
	}

	// Different tokens should produce different hashes
	hash1 := HashToken("token1")
	hash2 := HashToken("token2")
	if hash1 == hash2 {
		t.Error("HashToken() returned same hash for different tokens")
	}
}

func TestGenerateSecureToken(t *testing.T) {
	tests := []struct {
		name   string
		length int
		want   int // expected hex string length (2x byte length)
	}{
		{
			name:   "16 bytes",
			length: 16,
			want:   32,
		},
		{
			name:   "32 bytes",
			length: 32,
			want:   64,
		},
		{
			name:   "1 byte",
			length: 1,
			want:   2,
		},
		{
			name:   "0 bytes",
			length: 0,
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateSecureToken(tt.length)
			if err != nil {
				t.Fatalf("GenerateSecureToken() error = %v", err)
			}

			if len(token) != tt.want {
				t.Errorf("GenerateSecureToken(%d) returned length %d, want %d", tt.length, len(token), tt.want)
			}

			// Should only contain hex characters
			for _, c := range token {
				if !strings.ContainsRune("0123456789abcdef", c) {
					t.Errorf("GenerateSecureToken() returned non-hex character: %c", c)
				}
			}

			// Should be able to decode back to bytes
			bytes, err := hex.DecodeString(token)
			if err != nil {
				t.Errorf("GenerateSecureToken() returned invalid hex: %v", err)
			}

			if len(bytes) != tt.length {
				t.Errorf("GenerateSecureToken() decoded to %d bytes, want %d", len(bytes), tt.length)
			}
		})
	}

	// Should generate different tokens each time
	token1, _ := GenerateSecureToken(16)
	token2, _ := GenerateSecureToken(16)
	if token1 == token2 {
		t.Error("GenerateSecureToken() returned same token twice")
	}
}

func TestGenerateAndValidateToken(t *testing.T) {
	// Setup keys for testing
	var err error
	privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate test RSA key: %v", err)
	}
	publicKey = &privateKey.PublicKey

	user := &models.User{
		ID:    "test-user-id",
		Email: "test@example.com",
		Roles: []string{"user", "admin"},
	}

	// Generate token
	tokenString, err := GenerateToken(user)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	if tokenString == "" {
		t.Fatal("GenerateToken() returned empty string")
	}

	// Validate token
	claims, err := ValidateToken(tokenString)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}

	// Check claims
	if claims.UserID != user.ID {
		t.Errorf("ValidateToken() UserID = %v, want %v", claims.UserID, user.ID)
	}

	if claims.Email != user.Email {
		t.Errorf("ValidateToken() Email = %v, want %v", claims.Email, user.Email)
	}

	if len(claims.Roles) != len(user.Roles) {
		t.Errorf("ValidateToken() Roles length = %v, want %v", len(claims.Roles), len(user.Roles))
	}

	for i, role := range claims.Roles {
		if role != user.Roles[i] {
			t.Errorf("ValidateToken() Role[%d] = %v, want %v", i, role, user.Roles[i])
		}
	}

	if claims.Issuer != "scenario-authenticator" {
		t.Errorf("ValidateToken() Issuer = %v, want scenario-authenticator", claims.Issuer)
	}

	// Check expiration is in the future
	if claims.ExpiresAt == nil {
		t.Error("ValidateToken() ExpiresAt is nil")
	} else {
		expiresAt := claims.ExpiresAt.Time
		if expiresAt.Before(time.Now()) {
			t.Error("ValidateToken() token already expired")
		}

		// Check expiration is roughly 1 hour from now (with tolerance)
		expectedExpiry := time.Now().Add(time.Hour)
		diff := expiresAt.Sub(expectedExpiry)
		if diff < -time.Minute || diff > time.Minute {
			t.Errorf("ValidateToken() expiration time differs by %v from expected", diff)
		}
	}
}

func TestValidateToken_Invalid(t *testing.T) {
	// Setup keys for testing
	var err error
	privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate test RSA key: %v", err)
	}
	publicKey = &privateKey.PublicKey

	tests := []struct {
		name        string
		tokenString string
		wantError   bool
	}{
		{
			name:        "empty token",
			tokenString: "",
			wantError:   true,
		},
		{
			name:        "malformed token",
			tokenString: "not.a.valid.jwt",
			wantError:   true,
		},
		{
			name:        "random string",
			tokenString: "totally_invalid_token",
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := ValidateToken(tt.tokenString)

			if tt.wantError {
				if err == nil {
					t.Error("ValidateToken() expected error but got none")
				}
				if claims != nil {
					t.Error("ValidateToken() expected nil claims but got non-nil")
				}
			}
		})
	}
}

func TestValidateToken_Expired(t *testing.T) {
	// Setup keys for testing
	var err error
	privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate test RSA key: %v", err)
	}
	publicKey = &privateKey.PublicKey

	// Create an expired token
	claims := &models.Claims{
		UserID: "test-user",
		Email:  "test@example.com",
		Roles:  []string{"user"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)), // Expired 1 hour ago
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Issuer:    "scenario-authenticator",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("Failed to sign test token: %v", err)
	}

	// Try to validate expired token
	validatedClaims, err := ValidateToken(tokenString)

	if err == nil {
		t.Error("ValidateToken() expected error for expired token but got none")
	}

	if validatedClaims != nil {
		t.Error("ValidateToken() expected nil claims for expired token")
	}

	// Error should mention token expiration
	if err != nil && !strings.Contains(err.Error(), "expired") {
		t.Logf("ValidateToken() error message: %v", err)
	}
}

func TestValidateToken_WrongSigningKey(t *testing.T) {
	// Generate first key pair
	key1, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate test RSA key: %v", err)
	}

	// Generate second key pair (different)
	key2, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate second test RSA key: %v", err)
	}

	// Sign with key1
	privateKey = key1
	user := &models.User{
		ID:    "test-user",
		Email: "test@example.com",
		Roles: []string{"user"},
	}
	tokenString, err := GenerateToken(user)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	// Try to validate with key2
	publicKey = &key2.PublicKey
	claims, err := ValidateToken(tokenString)

	if err == nil {
		t.Error("ValidateToken() expected error for wrong signing key but got none")
	}

	if claims != nil {
		t.Error("ValidateToken() expected nil claims for wrong signing key")
	}
}
