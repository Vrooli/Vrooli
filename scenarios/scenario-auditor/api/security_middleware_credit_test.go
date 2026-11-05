package main

import "testing"

func TestLooksLikeSecurityHeadersMiddleware(t *testing.T) {
	middleware := `package middleware
import "net/http"
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		next.ServeHTTP(w, r)
	})
}`
	if !looksLikeSecurityHeadersMiddleware(middleware) {
		t.Error("a file setting all four hardening headers must be recognized as a security-headers middleware")
	}

	partial := `package x
import "net/http"
func H(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Frame-Options", "DENY")
	w.Write([]byte("hi"))
}`
	if looksLikeSecurityHeadersMiddleware(partial) {
		t.Error("a file setting only one header must not be treated as a security-headers middleware")
	}
}
