package integrations

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// The vrooli-events publisher signs every webhook with this credential
// (scenarios/vrooli-events/api/internal/subscription/secret.go). The hub
// verifies with the same value, resolved the same way: the explicit
// environment override first, then the credential authority. The hub never
// mints it; the publisher side owns the value.
const (
	eventsWebhookCredentialIdentity = "vrooli/vrooli-events"
	eventsWebhookCredentialField    = "agent-manager-webhook-secret"
	eventsWebhookSecretEnv          = "VROOLI_EVENTS_WEBHOOK_SECRET"
	webhookSecretRetryInterval      = 30 * time.Second
)

// SecretSource returns the current webhook secret, or "" when it is not
// available. The handler treats "" as "fail closed" (401).
type SecretSource func() string

// StaticSecret adapts a fixed value, for tests and explicit overrides.
func StaticSecret(secret string) SecretSource {
	return func() string { return secret }
}

// ResolveEventsWebhookSecret resolves the signing secret exactly as the
// vrooli-events publisher does. It never returns the value in an error.
func ResolveEventsWebhookSecret(getenv func(string) string) (string, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	if secret := strings.TrimSpace(getenv(eventsWebhookSecretEnv)); secret != "" {
		return secret, nil
	}
	identity, err := credentialauthority.ParseIdentity(eventsWebhookCredentialIdentity)
	if err != nil {
		return "", fmt.Errorf("parse events webhook credential identity: %w", err)
	}
	authority, err := credentialauthority.Default()
	if err != nil {
		return "", fmt.Errorf("credential authority unavailable: %w", err)
	}
	return authority.Require(identity, eventsWebhookCredentialField)
}

// CachedSecret resolves lazily and keeps the first successful value for the
// life of the process. A failed resolution (for example a locked keyring at
// boot) is retried at most every webhookSecretRetryInterval, so a store that
// unlocks later is observed without a restart and a burst of deliveries does
// not hammer the credential store.
func CachedSecret(resolve func() (string, error), onError func(error)) SecretSource {
	var (
		mu        sync.Mutex
		value     string
		lastTried time.Time
	)
	return func() string {
		mu.Lock()
		defer mu.Unlock()
		if value != "" {
			return value
		}
		if !lastTried.IsZero() && time.Since(lastTried) < webhookSecretRetryInterval {
			return ""
		}
		lastTried = time.Now()
		resolved, err := resolve()
		if err != nil {
			if onError != nil {
				onError(err)
			}
			return ""
		}
		value = strings.TrimSpace(resolved)
		return value
	}
}
