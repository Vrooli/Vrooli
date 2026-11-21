package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInjectBYOKKeyDiscardsRawHeaderValue(t *testing.T) {
	var receivedContextValue any
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		receivedContextValue = r.Context().Value(byokContextKey{})
	})
	r := httptest.NewRequest(http.MethodPost, "/ai", nil)
	r.Header.Set(BYOKHeaderName, "raw-secret-must-not-be-retained")

	NewBYOKMiddleware().InjectBYOKKey(next).ServeHTTP(httptest.NewRecorder(), r)

	if receivedContextValue != true {
		t.Fatalf("context selection hint = %#v, want true without the raw header value", receivedContextValue)
	}
}
