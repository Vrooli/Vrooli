package httpmw

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeadersStampEveryResponse(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	for name, want := range map[string]string{
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"X-XSS-Protection":          "0",
	} {
		if got := recorder.Header().Get(name); got != want {
			t.Errorf("%s=%q want %q", name, got, want)
		}
	}
}
