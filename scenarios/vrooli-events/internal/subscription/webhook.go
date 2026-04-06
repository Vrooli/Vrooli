package subscription

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// WebhookDeliverer sends event payloads to webhook URLs via HTTP POST.
type WebhookDeliverer struct {
	client *http.Client
}

// NewWebhookDeliverer creates a WebhookDeliverer with a 10-second timeout.
func NewWebhookDeliverer() *WebhookDeliverer {
	return &WebhookDeliverer{
		client: &http.Client{Timeout: 10 * time.Second},
	}
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
