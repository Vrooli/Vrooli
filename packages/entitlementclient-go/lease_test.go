package entitlementclient

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/vrooli/api-core/consumeridentity"
)

func TestLeaseRoundTripAndTamperRejection(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keys := consumeridentity.NewKeySet(consumeridentity.PublicKey{ID: "lease-key", Key: &key.PublicKey})
	payload := Payload{UserIdentity: "user@example.com", Status: "active", PlanTier: "pro", PlanRank: 3, Features: []string{"ai"}, NotAfter: time.Now().Add(time.Hour)}
	token, err := Sign(payload, "lease-key", key)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Verify(token, keys, time.Now())
	if err != nil || got.PlanTier != "pro" || got.PlanRank != 3 {
		t.Fatalf("verify = %#v, %v", got, err)
	}
	parts := split(t, token)
	parts[1] += "x"
	if _, err := Verify(parts[0]+"."+parts[1]+"."+parts[2], keys, time.Now()); !errors.Is(err, ErrLeaseSignature) && !errors.Is(err, ErrLeaseMalformed) {
		t.Fatalf("tampered lease error = %v", err)
	}
}

func TestLeaseExpiryIsRejectedEvenWithValidSignature(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keys := consumeridentity.NewKeySet(consumeridentity.PublicKey{ID: "lease-key", Key: &key.PublicKey})
	issued := time.Now()
	token, err := Sign(Payload{UserIdentity: "user@example.com", Status: "active", NotAfter: issued.Add(time.Minute)}, "lease-key", key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(token, keys, issued.Add(2*time.Minute)); !errors.Is(err, ErrLeaseExpired) {
		t.Fatalf("expiry error = %v", err)
	}
}

func TestCachedLeaseHonorsSignedExpiryWithoutRefreshing(t *testing.T) {
	issued := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	client := NewClient("http://127.0.0.1:1", nil, nil)
	client.cache["user@example.com"] = Payload{UserIdentity: "user@example.com", Status: "active", NotAfter: issued.Add(time.Hour)}

	if _, err := client.CachedAt("USER@example.com", issued.Add(30*time.Minute)); err != nil {
		t.Fatalf("valid cached lease = %v", err)
	}
	if _, err := client.CachedAt("user@example.com", issued.Add(time.Hour)); !errors.Is(err, ErrLeaseExpired) {
		t.Fatalf("expired cached lease error = %v", err)
	}
}

func split(t *testing.T, token string) []string {
	t.Helper()
	parts := make([]string, 0, 3)
	start := 0
	for i, char := range token {
		if char == '.' {
			parts = append(parts, token[start:i])
			start = i + 1
		}
	}
	parts = append(parts, token[start:])
	if len(parts) != 3 {
		t.Fatal("invalid token")
	}
	return parts
}
