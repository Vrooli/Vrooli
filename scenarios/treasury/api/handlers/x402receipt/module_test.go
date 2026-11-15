package x402receipt

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"treasury/internal/ledger"

	"github.com/vrooli/api-core/identity"
)

type receiptReaderStub struct {
	value ledger.AcceptedReceipt
	err   error
	gotID string
}

func TestModuleUsesConfiguredScenarioAuthenticatorJWKS(t *testing.T) {
	externalID := "treasury-x402-inflow:0x" + strings.Repeat("c", 64)
	reader := &receiptReaderStub{value: ledger.AcceptedReceipt{
		ExternalID: externalID, PostingID: "posting-configured-jwks", ReceiptID: "receipt-configured-jwks",
		AcceptedAt: time.Now().UTC(), AmountMinor: 4900, Currency: "USD", Basis: "authoritative",
	}}
	jwksURL, token := x402ReceiptServiceJWT(t, requiredCapability)
	t.Setenv("VROOLI_AUTH_SCENARIO_ISSUER", "scenario-authenticator")
	t.Setenv("VROOLI_AUTH_SCENARIO_JWKS_URL", jwksURL)
	router := mux.NewRouter()
	Module(reader).Mount(router)
	request := httptest.NewRequest(http.MethodGet, path+"?external_id="+externalID, nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("configured run-local receipt reader JWT should pass auth: status=%d body=%s", response.Code, response.Body.String())
	}
}

func x402ReceiptServiceJWT(t *testing.T, scopes ...string) (string, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	encode := func(value any) string {
		raw, marshalErr := json.Marshal(value)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	unsigned := encode(map[string]string{"alg": "RS256", "typ": "JWT", "kid": "configured-receipt-jwks"}) + "." + encode(map[string]any{
		"user_id": "landing-page-business-suite-treasury-x402-receipt-reader", "sub": "landing-page-business-suite-treasury-x402-receipt-reader", "actor_kind": "service", "scope": scopes,
		"aud": audience, "iss": "scenario-authenticator", "iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
	})
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	token := unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)
	public, err := json.Marshal(map[string]any{"keys": []any{map[string]any{
		"kty": "RSA", "kid": "configured-receipt-jwks", "alg": "RS256", "use": "sig",
		"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}}})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/jwks" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(public)
	}))
	t.Cleanup(server.Close)
	return server.URL + "/jwks", token
}

func (r *receiptReaderStub) GetAcceptedReceiptByExternalID(_ context.Context, externalID string) (ledger.AcceptedReceipt, error) {
	r.gotID = externalID
	return r.value, r.err
}

func callReceiptHandler(h *handler, externalID string, principal *identity.Principal) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, path+"?external_id="+externalID, nil)
	if principal != nil {
		request = request.WithContext(identity.WithPrincipal(request.Context(), *principal))
	}
	response := httptest.NewRecorder()
	h.get(response, request)
	return response
}

func TestReadbackRequiresServiceScopeAndExactAcceptedReceipt(t *testing.T) {
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	reader := &receiptReaderStub{value: ledger.AcceptedReceipt{
		ExternalID: "treasury-x402-inflow:0x" + strings.Repeat("a", 64), PostingID: "posting-1", ReceiptID: "receipt-1",
		AcceptedAt: now, AmountMinor: 150, Currency: "USD", Basis: "authoritative",
	}}
	h := &handler{reader: reader}
	externalID := reader.value.ExternalID
	if got := callReceiptHandler(h, externalID, nil).Code; got != http.StatusUnauthorized {
		t.Fatalf("anonymous request status=%d, want 401", got)
	}
	noScope := identity.Principal{Kind: identity.ActorService, Subject: "service:test", Verified: true, Scopes: []string{"treasury:other"}}
	if got := callReceiptHandler(h, externalID, &noScope).Code; got != http.StatusForbidden {
		t.Fatalf("service without receipt capability status=%d, want 403", got)
	}
	operator := identity.Principal{Kind: identity.ActorHuman, Subject: "human:test", Verified: true, Scopes: []string{requiredCapability}}
	if got := callReceiptHandler(h, externalID, &operator).Code; got != http.StatusUnauthorized {
		t.Fatalf("human request status=%d, want 401", got)
	}
	service := identity.Principal{Kind: identity.ActorService, Subject: "service:test", Verified: true, Scopes: []string{requiredCapability}}
	if got := callReceiptHandler(h, "treasury-x402-inflow:0xabc", &service).Code; got != http.StatusBadRequest {
		t.Fatalf("malformed transaction status=%d, want 400", got)
	}
	response := callReceiptHandler(h, externalID, &service)
	if response.Code != http.StatusOK {
		t.Fatalf("valid scoped readback status=%d body=%s", response.Code, response.Body.String())
	}
	var got receiptResponse
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.ExternalID != externalID || got.PostingID != "posting-1" || got.ReceiptID != "receipt-1" || got.AmountMinor != 150 || reader.gotID != externalID {
		t.Fatalf("unexpected receipt readback: got=%+v lookup=%q", got, reader.gotID)
	}
}

func TestReadbackDoesNotTurnMissingOrAmbiguousRowsIntoSuccess(t *testing.T) {
	service := identity.Principal{Kind: identity.ActorService, Subject: "service:test", Verified: true, Scopes: []string{requiredCapability}}
	externalID := "treasury-x402-inflow:0x" + strings.Repeat("b", 64)
	for _, tc := range []struct {
		name string
		err  error
		want int
	}{{"missing", ledger.ErrNotFound, http.StatusNotFound}, {"ambiguous", ledger.ErrAmbiguousAcceptedReceipt, http.StatusConflict}, {"storage", errors.New("database unavailable"), http.StatusServiceUnavailable}} {
		t.Run(tc.name, func(t *testing.T) {
			h := &handler{reader: &receiptReaderStub{err: tc.err}}
			if got := callReceiptHandler(h, externalID, &service).Code; got != tc.want {
				t.Fatalf("status=%d, want %d", got, tc.want)
			}
		})
	}
}
