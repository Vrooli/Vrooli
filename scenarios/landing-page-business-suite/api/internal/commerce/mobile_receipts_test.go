package commerce

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGooglePlayDeveloperValidatorUsesServerVerificationAndBindsAccount(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet || !strings.Contains(r.URL.Path, "/purchases/subscriptions/pro/tokens/purchase-token") {
			t.Fatalf("unexpected Play lookup: %s %s", r.Method, r.URL)
		}
		if r.Header.Get("Authorization") != "Bearer server-oauth-token" {
			t.Fatalf("missing server authorization")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"purchaseState":0,"orderId":"GPA.order","productId":"pro","purchaseToken":"purchase-token","obfuscatedExternalAccountId":"account-token","expiryTimeMillis":"4102444800000"}`)),
			Header:     make(http.Header),
		}, nil
	})}
	validator := GooglePlayDeveloperValidator{
		PackageName: "com.vrooli.app",
		ProductID:   "pro",
		OAuthToken: func(context.Context) (string, error) {
			return "server-oauth-token", nil
		},
		ResolveIdentity: func(context.Context, string) (string, error) {
			return "buyer@example.com", nil
		},
		Client: client,
		Now:    func() time.Time { return time.Unix(1700000000, 0) },
	}

	got, err := validator.Validate(context.Background(), Receipt{Source: "google", Token: "purchase-token", UserIdentity: "buyer@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ExternalSubscription != "purchase-token" || got.PlanTier != "pro" {
		t.Fatalf("normalized purchase = %+v", got)
	}
}

func TestGooglePlayDeveloperValidatorRejectsUnboundPurchase(t *testing.T) {
	validator := GooglePlayDeveloperValidator{
		PackageName: "com.vrooli.app",
		ProductID:   "pro",
		OAuthToken:  func(context.Context) (string, error) { return "token", nil },
		ResolveIdentity: func(context.Context, string) (string, error) {
			return "different@example.com", nil
		},
		Client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"purchaseState":0,"orderId":"GPA.order","productId":"pro","purchaseToken":"purchase-token","obfuscatedExternalAccountId":"account-token","expiryTimeMillis":"4102444800000"}`)), Header: make(http.Header)}, nil
		})},
	}
	if _, err := validator.Validate(context.Background(), Receipt{Source: "google", Token: "purchase-token", UserIdentity: "buyer@example.com"}); err != ErrReceiptBound {
		t.Fatalf("unbound purchase error = %v, want ErrReceiptBound", err)
	}
}

type appleFixture struct {
	validator AppleSignedTransactionValidator
	token     string
}

func newAppleFixture(t *testing.T, identity string) appleFixture {
	t.Helper()
	now := time.Unix(1_700_000_000, 0).UTC()
	certificateNow := time.Now().UTC()
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	rootTemplate := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Recorded Apple Root"}, NotBefore: certificateNow.Add(-time.Hour), NotAfter: certificateNow.Add(24 * time.Hour), IsCA: true, BasicConstraintsValid: true}
	rootDER, err := x509.CreateCertificate(rand.Reader, rootTemplate, rootTemplate, &rootKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	root, err := x509.ParseCertificate(rootDER)
	if err != nil {
		t.Fatal(err)
	}
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	leafTemplate := &x509.Certificate{SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: "Recorded Apple Signing"}, NotBefore: certificateNow.Add(-time.Hour), NotAfter: certificateNow.Add(24 * time.Hour)}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, root, &leafKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"ES256","x5c":["` + base64.StdEncoding.EncodeToString(leafDER) + `"]}`))
	payload, err := json.Marshal(map[string]any{
		"transactionId": "txn-recorded-1", "originalTransactionId": "orig-recorded-1",
		"bundleId": "com.vrooli.app", "productId": "pro", "appAccountToken": "account-token",
		"expiresDate": now.Add(time.Hour).UnixMilli(),
	})
	if err != nil {
		t.Fatal(err)
	}
	payloadPart := base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(header + "." + payloadPart))
	r, s, err := ecdsa.Sign(rand.Reader, leafKey, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	signature := make([]byte, 64)
	r.FillBytes(signature[:32])
	s.FillBytes(signature[32:])
	token := header + "." + payloadPart + "." + base64.RawURLEncoding.EncodeToString(signature)
	return appleFixture{validator: AppleSignedTransactionValidator{
		RootCertificate: root, ExpectedBundleID: "com.vrooli.app", ProductPlans: map[string]string{"pro": "pro"},
		ResolveIdentity: func(context.Context, string) (string, error) { return identity, nil }, Now: func() time.Time { return now },
	}, token: token}
}

func TestAppleSignedTransactionValidatorAcceptsRecordedPayload(t *testing.T) {
	fixture := newAppleFixture(t, "buyer@example.com")
	got, err := fixture.validator.Validate(context.Background(), Receipt{Source: "apple", Token: fixture.token, UserIdentity: "buyer@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if got.SubscriptionID != "orig-recorded-1" || got.PlanTier != "pro" || got.Status != "active" {
		t.Fatalf("normalized Apple transaction = %+v", got)
	}
}

func TestAppleSignedTransactionValidatorRejectsTamperedPayload(t *testing.T) {
	fixture := newAppleFixture(t, "buyer@example.com")
	parts := strings.Split(fixture.token, ".")
	parts[1] = base64.RawURLEncoding.EncodeToString([]byte(`{"transactionId":"tampered"}`))
	if _, err := fixture.validator.Validate(context.Background(), Receipt{Source: "apple", Token: strings.Join(parts, "."), UserIdentity: "buyer@example.com"}); err != ErrReceiptInvalid {
		t.Fatalf("tampered Apple payload error = %v, want ErrReceiptInvalid", err)
	}
}

func TestAppleSignedTransactionValidatorRejectsBoundReceipt(t *testing.T) {
	fixture := newAppleFixture(t, "different@example.com")
	if _, err := fixture.validator.Validate(context.Background(), Receipt{Source: "apple", Token: fixture.token, UserIdentity: "buyer@example.com"}); err != ErrReceiptBound {
		t.Fatalf("bound Apple receipt error = %v, want ErrReceiptBound", err)
	}
}
