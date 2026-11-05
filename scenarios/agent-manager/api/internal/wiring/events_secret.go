package wiring

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// recoveringEventsSecret caches a successfully resolved signing secret while
// allowing later requests to recover from a startup credential outage.
func recoveringEventsSecret(initial string, resolve func() (string, error)) func() string {
	var mu sync.Mutex
	secret := strings.TrimSpace(initial)
	return func() string {
		mu.Lock()
		defer mu.Unlock()
		if secret == "" {
			if value, err := resolve(); err == nil {
				secret = strings.TrimSpace(value)
			}
		}
		return secret
	}
}

func recoveringEventsWebhook(secret func() string, handler func(string) http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		value := secret()
		if value == "" {
			http.Error(w, "event verification temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		handler(value).ServeHTTP(w, r)
	})
}

const (
	eventsWebhookIdentity = "vrooli/vrooli-events"
	eventsWebhookField    = "agent-manager-webhook-secret"
)

func resolveEventsWebhookSecret() (string, error) {
	identity, err := credentialauthority.ParseIdentity(eventsWebhookIdentity)
	if err != nil {
		return "", fmt.Errorf("parse events webhook credential identity: %w", err)
	}
	authority, err := credentialauthority.Default()
	if err != nil {
		return "", fmt.Errorf("credential authority unavailable: %w", err)
	}
	return authority.ResolveOrMint(identity, eventsWebhookField, nil, func() (string, error) {
		return credentialauthority.RandomBase64(32)
	})
}
