package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
)

const (
	onboardingTokenIdentity = "vrooli/onboarding" // #nosec G101 -- this is a credential-store identity, never a secret.
	onboardingTokenField = "api-token" // #nosec G101 -- this is a credential-store field identifier, never a secret.
)

// onboardingExpectedToken is deliberately resolved through the credential
// authority. It is not configurable through a flag or environment variable,
// which prevents a remote caller from changing the secret it is expected to
// present by changing process configuration.
var onboardingExpectedToken = func(ctx context.Context) (string, error) {
	client, err := onboardingCredentialClient()
	if err != nil {
		return "", err
	}
	return client.Resolve(ctx, onboardingTokenIdentity, onboardingTokenField)
}

func onboardingMutationAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requestIsLoopback(r) {
			next.ServeHTTP(w, r)
			return
		}

		presented, parseErr := decodeBearerToken(r.Header.Get("Authorization"))
		if parseErr != nil {
			writeAuthError(w, "missing bearer token")
			return
		}
		expected, err := onboardingExpectedToken(r.Context())
		if err != nil || strings.TrimSpace(expected) == "" || !secureTokenEqual(presented, expected) {
			writeAuthError(w, "invalid bearer token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requestIsLoopback(r *http.Request) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		host = strings.TrimSpace(r.RemoteAddr)
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func secureTokenEqual(presented, expected string) bool {
	presentedBytes := []byte(presented)
	expectedBytes := []byte(strings.TrimSpace(expected))
	if len(presentedBytes) != len(expectedBytes) {
		return false
	}
	return subtle.ConstantTimeCompare(presentedBytes, expectedBytes) == 1
}

func writeAuthError(w http.ResponseWriter, reason string) {
	writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "onboarding operator authorization required", "reason": reason})
}

var errInvalidRemoteAuthorization = errors.New("invalid remote authorization")

func decodeBearerToken(header string) (string, error) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") || parts[1] == "" {
		return "", errInvalidRemoteAuthorization
	}
	return parts[1], nil
}

func authErrorJSON(reason string) []byte {
	body, _ := json.Marshal(map[string]string{"error": "onboarding operator authorization required", "reason": reason})
	return body
}
