// Package emailevents exposes the one deliberate REST exception owned by the
// SendGrid provider. It verifies the raw request before decoding JSON.
package emailevents

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	admin "landing-page-business-suite-api/internal/administration"
	email "landing-page-business-suite-api/internal/emailevents"
)

const maxBodyBytes = 1 << 20

type Dependencies struct {
	Store     *email.Repository
	PublicKey func() string
	Throttle  interface {
		Allow(context.Context, string, admin.ThrottleRule) (bool, error)
	}
	ClientIP func(*http.Request) string
	Now      func() time.Time
	Log      func(string, map[string]any)
}

func Handler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes+1))
		if err != nil || len(body) > maxBodyBytes {
			if deps.Log != nil {
				deps.Log("sendgrid_webhook_body_rejected", map[string]any{"bytes": len(body)})
			}
			http.Error(w, "invalid webhook body", http.StatusRequestEntityTooLarge)
			return
		}
		ip := "unknown"
		if deps.ClientIP != nil {
			ip = deps.ClientIP(r)
		}
		now := time.Now().UTC()
		if deps.Now != nil {
			now = deps.Now().UTC()
		}
		publicKey := ""
		if deps.PublicKey != nil {
			publicKey = deps.PublicKey()
		}
		if err := email.Verify(publicKey, r.Header.Get("X-Twilio-Email-Event-Webhook-Signature"), r.Header.Get("X-Twilio-Email-Event-Webhook-Timestamp"), body, now); err != nil {
			if deps.Throttle != nil {
				allowed, throttleErr := deps.Throttle.Allow(r.Context(), admin.ThrottleBucket("sendgrid-webhook-bad-signature", ip), admin.ThrottleRule{Limit: 20, Window: 15 * time.Minute})
				if throttleErr != nil {
					http.Error(w, "webhook unavailable", http.StatusServiceUnavailable)
					return
				}
				if !allowed {
					http.Error(w, "too many invalid webhook attempts", http.StatusTooManyRequests)
					return
				}
			}
			if deps.Log != nil {
				deps.Log("sendgrid_webhook_signature_rejected", map[string]any{"ip": ip, "error": err.Error()})
			}
			http.Error(w, "invalid webhook signature", http.StatusUnauthorized)
			return
		}
		var events []email.Event
		if err := json.Unmarshal(body, &events); err != nil {
			http.Error(w, "invalid webhook payload", http.StatusBadRequest)
			return
		}
		if deps.Store == nil {
			http.Error(w, "webhook unavailable", http.StatusServiceUnavailable)
			return
		}
		for _, event := range events {
			if event.CustomArgs == nil {
				event.CustomArgs = map[string]string{}
			}
			if _, err := deps.Store.Record(context.WithoutCancel(r.Context()), event, now); err != nil {
				if deps.Log != nil {
					deps.Log("sendgrid_webhook_record_failed", map[string]any{"error": err.Error()})
				}
				http.Error(w, "webhook processing failed", http.StatusInternalServerError)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"accepted"}`))
	}
}
