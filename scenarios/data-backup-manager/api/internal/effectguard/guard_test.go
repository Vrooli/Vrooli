package effectguard

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// [REQ:DBM-TEST-ISOLATION] Deny before catalog, filesystem, or job admission.
func TestProductionRejectsTestRPCBeforeHandler(t *testing.T) {
	called := false
	h := ProtectProduction(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	r := httptest.NewRequest("POST", "/RestoreTarget", nil)
	r.Header.Set("X-Vrooli-Test-Mode", "1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if called || w.Code != 403 {
		t.Fatalf("test reached production handler: %v %d", called, w.Code)
	}
}
