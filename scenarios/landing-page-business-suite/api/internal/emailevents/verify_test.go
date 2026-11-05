package emailevents

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"errors"
	"testing"
	"time"
)

func TestVerifySignedEventWebhook(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	timestamp := "1700000000"
	payload := []byte(`[ {"event":"delivered"} ]`)
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256Digest([]byte(timestamp), payload)
	r, s, err := ecdsa.Sign(rand.Reader, key, digest)
	if err != nil {
		t.Fatal(err)
	}
	sig, err := asn1.Marshal(ecdsaSignature{r, s})
	if err != nil {
		t.Fatal(err)
	}
	publicKey := base64.StdEncoding.EncodeToString(der)
	signature := base64.StdEncoding.EncodeToString(sig)
	if err := Verify(publicKey, signature, timestamp, payload, now); err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}

	cases := []struct {
		name         string
		key, sig, ts string
		body         []byte
		at           time.Time
		want         error
	}{
		{"modified body", publicKey, signature, timestamp, []byte("tampered"), now, ErrInvalidSignature},
		{"modified timestamp", publicKey, signature, "1700000001", payload, now, ErrInvalidSignature},
		{"old", publicKey, signature, timestamp, payload, now.Add(11 * time.Minute), ErrStaleSignature},
		{"missing signature", publicKey, "", timestamp, payload, now, ErrMissingSignature},
		{"wrong key", base64.StdEncoding.EncodeToString(mustPublicKey(t)), signature, timestamp, payload, now, ErrInvalidSignature},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Verify(tc.key, tc.sig, tc.ts, tc.body, tc.at); !errors.Is(got, tc.want) {
				t.Fatalf("error = %v, want %v", got, tc.want)
			}
		})
	}
}

func sha256Digest(parts ...[]byte) []byte {
	h := sha256.New()
	for _, part := range parts {
		_, _ = h.Write(part)
	}
	return h.Sum(nil)
}

func mustPublicKey(t *testing.T) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return der
}
