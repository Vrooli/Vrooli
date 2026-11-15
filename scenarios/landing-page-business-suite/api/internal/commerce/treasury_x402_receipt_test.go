package commerce

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrooli/api-core/servicetoken"
)

type fixedTreasuryReceiptToken string

func (t fixedTreasuryReceiptToken) Token(context.Context) (string, error) { return string(t), nil }

func TestTreasuryX402ReceiptReaderUsesScopedBearerAndReturnsAcceptedLink(t *testing.T) {
	wantExternal := "treasury-x402-inflow:0xabc/def"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != treasuryX402ReceiptPath {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("external_id"); got != wantExternal {
			t.Fatalf("external_id = %q, want %q", got, wantExternal)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer short-lived-scope-token" {
			t.Fatalf("authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"external_id":"treasury-x402-inflow:0xabc/def","posting_id":"posting-1","receipt_id":"receipt-1","accepted_at":"2026-11-14T11:00:00Z","amount_minor":4900,"currency":"USD","basis":"derived"}`))
	}))
	defer server.Close()
	reader, err := NewTreasuryX402ReceiptReader(server.URL, fixedTreasuryReceiptToken("short-lived-scope-token"), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := reader.ReadAccepted(context.Background(), wantExternal)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.PostingID != "posting-1" || receipt.ReceiptID != "receipt-1" || receipt.AmountMinor != 4900 || receipt.Currency != "USD" || !receipt.AcceptedAt.Equal(time.Date(2026, 11, 14, 11, 0, 0, 0, time.UTC)) {
		t.Fatalf("receipt = %+v", receipt)
	}
}

func TestTreasuryX402ReceiptReaderTreatsMissingAsNoReceipt(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	reader, err := NewTreasuryX402ReceiptReader(server.URL, fixedTreasuryReceiptToken("token"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reader.ReadAccepted(context.Background(), "treasury-x402-inflow:tx"); !errors.Is(err, ErrTreasuryX402ReceiptNotFound) {
		t.Fatalf("missing receipt error = %v", err)
	}
}

func TestTreasuryX402ReceiptReaderDoesNotFollowRedirects(t *testing.T) {
	var redirected bool
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		redirected = true
		w.WriteHeader(http.StatusOK)
	}))
	defer destination.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", destination.URL)
		w.WriteHeader(http.StatusFound)
	}))
	defer source.Close()
	reader, err := NewTreasuryX402ReceiptReader(source.URL, fixedTreasuryReceiptToken("token"), source.Client())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reader.ReadAccepted(context.Background(), "treasury-x402-inflow:tx"); err == nil {
		t.Fatal("redirect was accepted as a receipt")
	}
	if redirected {
		t.Fatal("service token was sent through an HTTP redirect")
	}
}

func TestTreasuryX402ReceiptReaderFailsClosedOnMismatchedReceipt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"external_id":"other","posting_id":"posting-1","receipt_id":"receipt-1","accepted_at":"2026-11-14T11:00:00Z","amount_minor":4900,"currency":"USD","basis":"derived"}`))
	}))
	defer server.Close()
	reader, err := NewTreasuryX402ReceiptReader(server.URL, fixedTreasuryReceiptToken("token"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reader.ReadAccepted(context.Background(), "treasury-x402-inflow:tx"); err == nil {
		t.Fatal("receipt for another external ID was accepted")
	}
}

func TestTreasuryX402ReceiptReaderRequiresSafeOriginAndServiceToken(t *testing.T) {
	if _, err := NewTreasuryX402ReceiptReader("http://example.com", fixedTreasuryReceiptToken("token"), nil); err == nil {
		t.Fatal("non-loopback plain HTTP was accepted")
	}
	if _, err := NewTreasuryX402ReceiptReader("http://127.0.0.1:1234", nil, nil); err == nil {
		t.Fatal("missing service identity was accepted")
	}
}

func TestTreasuryX402ReceiptTokensRequireExactPrincipalAudienceAndScope(t *testing.T) {
	var credentialSeen string
	tokens := NewTreasuryX402ReceiptTokens(func(context.Context) (string, error) { return "operator-issued-credential", nil }, "http://127.0.0.1:15000", nil,
		servicetoken.WithExchanger(func(_ context.Context, credential string) (servicetoken.Exchanged, error) {
			credentialSeen = credential
			return servicetoken.Exchanged{
				AccessToken: "short-lived-token", ExpiresAt: time.Now().Add(time.Minute),
				Subject: TreasuryX402ReceiptPrincipal, Audience: TreasuryX402ReceiptAudience,
				Scopes: []string{TreasuryX402ReceiptScope},
			}, nil
		}),
	)
	token, err := tokens.Token(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if token != "short-lived-token" || credentialSeen != "operator-issued-credential" {
		t.Fatalf("token=%q credential forwarded=%q", token, credentialSeen)
	}

	wrong := NewTreasuryX402ReceiptTokens(func(context.Context) (string, error) { return "credential", nil }, "http://127.0.0.1:15000", nil,
		servicetoken.WithExchanger(func(context.Context, string) (servicetoken.Exchanged, error) {
			return servicetoken.Exchanged{AccessToken: "wrong-token", ExpiresAt: time.Now().Add(time.Minute), Subject: "other", Audience: TreasuryX402ReceiptAudience, Scopes: []string{TreasuryX402ReceiptScope}}, nil
		}),
	)
	if token, err := wrong.Token(context.Background()); token != "" || !errors.Is(err, servicetoken.ErrIdentityMismatch) {
		t.Fatalf("mismatched identity returned token=%q err=%v", token, err)
	}
}
