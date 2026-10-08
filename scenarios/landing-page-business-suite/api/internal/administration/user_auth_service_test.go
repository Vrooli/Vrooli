package administration

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/vrooli/api-core/consumeridentity"
)

func TestRequestMagicLinkRejectsMissingConfiguredOriginBeforeWritingToken(t *testing.T) {
	service := NewUserAuthService(UserAuthServiceOptions{})

	err := service.RequestMagicLink(context.Background(), "customer@example.test", "", "")
	if err == nil || !strings.Contains(err.Error(), "AUTH_MAGIC_LINK_BASE_URL") {
		t.Fatalf("RequestMagicLink() error = %v, want missing configured origin error", err)
	}
}

func TestConsumerAccessTokenUsesRS256AndKeyID(t *testing.T) {
	service := NewUserAuthService(UserAuthServiceOptions{JWTIssuer: "lpbs.test", ConsumerSigningKeyID: "current"})
	token, _, err := service.GenerateAccessToken("user-1", "user@example.test", "session-1")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token parts = %d", len(parts))
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatal(err)
	}
	var header map[string]string
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		t.Fatal(err)
	}
	if header["alg"] != consumeridentity.Algorithm || header["kid"] != "current" {
		t.Fatalf("header = %#v", header)
	}
	if _, err := service.ValidateAccessToken(token); err != nil {
		t.Fatalf("ValidateAccessToken() error = %v", err)
	}
}

func TestConsumerAccessTokenRejectsLegacyHS256(t *testing.T) {
	service := NewUserAuthService(UserAuthServiceOptions{JWTIssuer: "lpbs.test"})
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": "lpbs.test", "sub": "user-1", "uid": "user-1", "email": "user@example.test",
		"iat": time.Now().Unix(), "nbf": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte("legacy-secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ValidateAccessToken(token); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("legacy token error = %v", err)
	}
}

func TestConsumerAccessTokenAcceptsPreviousSigningKeyDuringRotation(t *testing.T) {
	oldSigner, err := consumeridentity.GenerateSigner("old", "lpbs.test", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	newSigner, err := consumeridentity.GenerateSigner("new", "lpbs.test", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	privatePEM, err := consumeridentity.MarshalPrivateKeyPEM(newSigner.Private)
	if err != nil {
		t.Fatal(err)
	}
	service := NewUserAuthService(UserAuthServiceOptions{
		JWTIssuer:             "lpbs.test",
		ConsumerSigningKeyPEM: string(privatePEM),
		ConsumerSigningKeyID:  "new",
		ConsumerPreviousKeys:  []consumeridentity.PublicKey{{ID: "old", Key: &oldSigner.Private.PublicKey}},
	})
	token, _, err := oldSigner.Sign(consumeridentity.Claims{Subject: "user-1", UserID: "user-1", Email: "user@example.test"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ValidateAccessToken(token); err != nil {
		t.Fatalf("previous-key token error = %v", err)
	}
}
