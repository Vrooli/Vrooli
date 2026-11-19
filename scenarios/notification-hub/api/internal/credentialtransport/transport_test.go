package credentialtransport

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCredentialTransportIgnoresSpoofedForwardedHeaders(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	for _, path := range []string{"/vrooli.notification_hub.v1.conversations.ConversationsService/AnswerRootWithPassword", "/api/v1/identity/root-enrollment", "/api/v1/identity/root-decision-read"} {
		for _, tc := range []struct {
			name, peer string
			tls        bool
			want       int
		}{{"remote cleartext", "192.0.2.1:42", false, 403}, {"local", "127.0.0.1:42", false, 204}, {"ipv6 local", "[::1]:42", false, 204}, {"direct tls", "192.0.2.1:42", true, 204}} {
			t.Run(tc.name, func(t *testing.T) {
				r := httptest.NewRequest("POST", "http://host"+path, nil)
				r.RemoteAddr = tc.peer
				r.Header.Set("X-Forwarded-Proto", "https")
				r.Header.Set("Origin", "https://example.test")
				if tc.tls {
					r.TLS = &tls.ConnectionState{}
				}
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				if w.Code != tc.want {
					t.Fatalf("status %d", w.Code)
				}
			})
		}
	}
}
