package walletrecovery

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type rejectAuth struct{}

func (rejectAuth) AuthorizeWalletRecovery(*http.Request) error { return errors.New("not allowed") }

func TestHTTPHandlerFailsClosedWithoutServiceIdentity(t *testing.T) {
	h := NewHTTPHandler(Service{}, nil)
	r := httptest.NewRequest("POST", "/api/wallet-recovery/attest", strings.NewReader(`{"reservation_id":"x","target_id":"t","destination_id":"d"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 503 {
		t.Fatalf("want fail-closed 503, got %d", w.Code)
	}
}

func TestHTTPHandlerRequiresAuthorizedCaller(t *testing.T) {
	h := NewHTTPHandler(Service{}, rejectAuth{})
	r := httptest.NewRequest("POST", "/api/wallet-recovery/attest", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatalf("want 403, got %d", w.Code)
	}
}
