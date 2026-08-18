// DOC: docs/guides/creating-subscriptions.md
// DOC: docs/internal/TEMPORAL-FLOWS.md
package subscription

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// WebhookDeliverer sends event payloads to webhook URLs via HTTP POST.
type WebhookDeliverer struct {
	client *http.Client
	secret []byte
}

// NewWebhookDeliverer creates a WebhookDeliverer with a 10-second timeout.
func NewWebhookDeliverer() *WebhookDeliverer {
	return &WebhookDeliverer{
		client: &http.Client{Timeout: 10 * time.Second},
		secret: []byte(os.Getenv("VROOLI_EVENTS_WEBHOOK_SECRET")),
	}
}

func NewWebhookDelivererWithSecret(secret string) *WebhookDeliverer {
	d := NewWebhookDeliverer()
	d.secret = []byte(secret)
	return d
}

// WebhookPayload is the JSON body sent to webhook endpoints.
type WebhookPayload struct {
	EventID        string      `json:"event_id"`
	EventType      string      `json:"event_type"`
	SourceScenario string      `json:"source_scenario"`
	Payload        interface{} `json:"payload,omitempty"`
	DeliveredAt    string      `json:"delivered_at"`
}

// Deliver sends the payload to the webhook URL and returns nil on success.
func (w *WebhookDeliverer) Deliver(ctx context.Context, target string, payload WebhookPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "vrooli-events/1.0")
	req.Header.Set("X-Vrooli-Events-Event-ID", payload.EventID)
	// Redeliveries carry the stable event identity separately from the body so
	// receivers can deduplicate before parsing or persisting the payload.
	req.Header.Set("X-Vrooli-Events-Idempotency-Key", payload.EventID)
	if len(w.secret) > 0 {
		mac := hmac.New(sha256.New, w.secret)
		_, _ = mac.Write(body)
		req.Header.Set("X-Vrooli-Events-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook delivery failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}
