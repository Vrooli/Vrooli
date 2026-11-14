package operatordecision

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// SignatureHeader is the header vrooli-events signs its webhook bodies with.
const SignatureHeader = "X-Vrooli-Events-Signature"

// maxWebhookBody caps the resolution webhook body (1 MiB). A body at the cap is
// accepted; an over-cap body is rejected with 413 rather than silently truncated
// (which would fail the HMAC check and look like a permanent bad signature).
const maxWebhookBody = 1 << 20

// ValidSignature reports whether value is the HMAC-SHA256 of body under secret,
// in constant time. value may carry the "sha256=" prefix. An empty secret or an
// undecodable value is invalid. It ports validProgramEventSignature so every
// scenario's resolution webhook verifies identically.
func ValidSignature(body []byte, value, secret string) bool {
	if strings.TrimSpace(secret) == "" {
		return false
	}
	value = strings.TrimPrefix(strings.TrimSpace(value), "sha256=")
	provided, err := hex.DecodeString(value)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := mac.Sum(nil)
	return len(provided) == len(expected) && subtle.ConstantTimeCompare(provided, expected) == 1
}

// DomainEffect applies a resolved decision to one scenario domain. Handles
// reports whether the effect owns resolutions for a given producer event type
// (the resolution's SourceEventType); Apply acts on a resolution it owns. The
// Webhook fans a resolution to its effects in order and the FIRST that reports
// handled=true wins (so overlapping effects are ordered by registration, exactly
// like agent-manager's run-approval-then-authorization fall-through).
type DomainEffect interface {
	// Handles reports whether this effect owns the producer event type. The webhook
	// skips Apply for effects that do not.
	Handles(sourceEventType string) bool
	// Apply acts on a resolution. handled=true stops fan-out; handled=false lets a
	// later effect try (used when Handles is coarse). An error is surfaced as 500 so
	// vrooli-events retries, EXCEPT the effect should return a benign handled=true
	// nil for an unknown/already-decided ask to avoid a poison-message retry loop.
	Apply(ctx context.Context, resolution DecisionResolution) (handled bool, err error)
}

// Webhook builds the http.Handler a scenario mounts to receive
// notification_hub.ask.resolved.v1 through the signed vrooli-events webhook. It
// verifies the HMAC signature, decodes the standard envelope, gates on
// AskResolvedEventType, and fans the resolution to effects (first handled wins).
// It generalizes agent-manager's DecisionResolvedWebhook minus the
// effort-decision fallthrough, which an effect now carries.
//
// Responses: 405 for non-POST, 413 for an over-cap body, 400 for an unreadable or
// undecodable body, 401 for a bad/absent signature, 200 {"ignored":true} for a
// non-resolution event or an unclaimed resolution, 500 for an effect error (so
// the event is retried), and 200 {"handled":true,...} when an effect claimed it.
//
// REPLAY SAFETY is delegated to each DomainEffect: the webhook itself does not
// deduplicate, so vrooli-events MAY redeliver the same resolution (e.g. after a
// lost ack, or a re-raise inside the dedup window). An effect MUST treat a
// resolution for an already-decided / unknown ask as a benign no-op and return
// (handled=true, nil) — never re-apply the effect and never return an error that
// would trigger a retry storm. See DomainEffect.Apply.
func Webhook(secret string, effects ...DomainEffect) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		// Read one byte past the cap so a body AT the cap still verifies but an
		// OVER-cap body is detected and rejected with an explicit 413 — otherwise a
		// truncated body fails the HMAC check and masquerades as a permanent 401.
		body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBody+1))
		if err != nil {
			http.Error(w, "event webhook body could not be read", http.StatusBadRequest)
			return
		}
		if len(body) > maxWebhookBody {
			http.Error(w, "event webhook body exceeds the size limit", http.StatusRequestEntityTooLarge)
			return
		}
		if !ValidSignature(body, r.Header.Get(SignatureHeader), secret) {
			http.Error(w, "event webhook signature is invalid or unavailable", http.StatusUnauthorized)
			return
		}
		var envelope struct {
			EventID   string             `json:"event_id"`
			EventType string             `json:"event_type"`
			Source    string             `json:"source_scenario"`
			Payload   DecisionResolution `json:"payload"`
		}
		if err := json.Unmarshal(body, &envelope); err != nil {
			http.Error(w, "invalid webhook body", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if envelope.EventType != AskResolvedEventType {
			_ = json.NewEncoder(w).Encode(map[string]any{"ignored": true, "reason": "not an ask resolution"})
			return
		}
		for _, effect := range effects {
			if effect == nil || !effect.Handles(envelope.Payload.SourceEventType) {
				continue
			}
			handled, rErr := effect.Apply(r.Context(), envelope.Payload)
			if rErr != nil {
				http.Error(w, "apply decision resolution: "+rErr.Error(), http.StatusInternalServerError)
				return
			}
			if handled {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"handled": true,
					"ask_id":  envelope.Payload.AskID,
					"state":   envelope.Payload.State,
				})
				return
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ignored": true, "reason": "no matching decision"})
	})
}
