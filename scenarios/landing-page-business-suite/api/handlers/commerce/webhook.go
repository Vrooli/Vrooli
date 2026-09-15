package billing

import (
	"io"
	"net/http"
)

const maxWebhookBodyBytes = 1 << 20

// WebhookDependencies keeps the Stripe-defined HTTP shape at the transport
// edge while the Stripe service owns signature verification and event effects.
type WebhookDependencies struct {
	Handle     func([]byte, string) error
	WriteError func(http.ResponseWriter, int, string, string)
	WriteJSON  func(http.ResponseWriter, any)
	Log        func(string, map[string]any)
}

// Webhook handles Stripe's signed callback. It remains REST because Stripe,
// rather than this scenario, owns the request shape.
func Webhook(deps WebhookDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBodyBytes+1))
		if err != nil {
			deps.Log("webhook_body_read_failed", map[string]any{"error": err.Error()})
			deps.WriteError(w, http.StatusBadRequest, "Failed to read request body", "validation")
			return
		}
		if len(body) > maxWebhookBodyBytes {
			deps.Log("webhook_body_too_large", map[string]any{"bytes": len(body)})
			deps.WriteError(w, http.StatusRequestEntityTooLarge, "Webhook request body is too large", "validation")
			return
		}
		signature := r.Header.Get("Stripe-Signature")
		if signature == "" {
			deps.Log("webhook_signature_missing", nil)
			deps.WriteError(w, http.StatusBadRequest, "Missing Stripe-Signature header", "validation")
			return
		}
		if err := deps.Handle(body, signature); err != nil {
			deps.Log("webhook_processing_failed", map[string]any{"error": err.Error()})
			deps.WriteError(w, http.StatusBadRequest, "Webhook processing failed", "server_error")
			return
		}
		deps.WriteJSON(w, map[string]any{"status": "success"})
	}
}
