package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"agent-manager/internal/supervision"
)

// DecisionEventWebhookPath is the notification-hub ask.resolved webhook route.
const DecisionEventWebhookPath = "/api/v1/events/notification-hub"

// DecisionResolvedWebhook receives notification-hub's
// notification_hub.ask.resolved.v1 through the signed vrooli-events webhook
// and applies it to the matching effort decision. Events about asks Agent
// Manager did not raise, and duplicate deliveries, are acknowledged without
// change.
func DecisionResolvedWebhook(service *supervision.Service, secret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil || strings.TrimSpace(secret) == "" || !validProgramEventSignature(body, r.Header.Get("X-Vrooli-Events-Signature"), secret) {
			http.Error(w, "event webhook signature is invalid or unavailable", http.StatusUnauthorized)
			return
		}
		var envelope struct {
			EventID   string                         `json:"event_id"`
			EventType string                         `json:"event_type"`
			Source    string                         `json:"source_scenario"`
			Payload   supervision.DecisionResolution `json:"payload"`
		}
		if err := json.Unmarshal(body, &envelope); err != nil {
			http.Error(w, "invalid webhook body", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if envelope.EventType != supervision.AskResolvedEventType {
			_ = json.NewEncoder(w).Encode(map[string]any{"ignored": true, "reason": "not an ask resolution"})
			return
		}
		if service == nil || service.Efforts == nil {
			http.Error(w, "effort supervision unavailable", http.StatusServiceUnavailable)
			return
		}
		decision, err := service.Efforts.ResolveDecisionFromHub(r.Context(), envelope.Payload)
		if err != nil {
			http.Error(w, "apply ask resolution: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if decision == nil {
			_ = json.NewEncoder(w).Encode(map[string]any{"ignored": true, "reason": "no matching effort decision"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"goal_home": decision.GoalHome, "decision_id": decision.DecisionId, "state": decision.State})
	})
}
