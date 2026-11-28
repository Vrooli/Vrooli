// Package credentialtransport refuses direct remote cleartext credential input.
// Loopback peers are trusted local ingress, including the scenario UI proxy.
package credentialtransport

import (
	"net"
	"net/http"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/identity/root-decision-read" || r.URL.Path == "/api/v1/identity/root-enrollment" || r.URL.Path == "/vrooli.notification_hub.v1.identity.IdentityService/Login" || r.URL.Path == "/vrooli.notification_hub.v1.conversations.ConversationsService/AnswerRootWithPassword" {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			ip := net.ParseIP(host)
			if r.TLS == nil && (err != nil || ip == nil || !ip.IsLoopback()) {
				http.Error(w, "secure connection required", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
