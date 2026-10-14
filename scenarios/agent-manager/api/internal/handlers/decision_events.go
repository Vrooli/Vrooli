package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"agent-manager/internal/supervision"
)

// DecisionEventWebhookPath is the notification-hub ask.resolved webhook route.
const DecisionEventWebhookPath = "/api/v1/events/notification-hub"

// RunApprovalResolver applies a notification-hub ask resolution that answers a
// run-approval decision (needs_review). It reports handled=true when the
// resolution belongs to a run approval. Satisfied by *orchestration.Orchestrator.
type RunApprovalResolver interface {
	ResolveRunApprovalFromHub(ctx context.Context, resolution supervision.DecisionResolution) (bool, error)
}

// DecisionResolvedWebhook receives notification-hub's
// notification_hub.ask.resolved.v1 through the signed vrooli-events webhook
// and applies it. A run-approval resolution (needs_review) drives
// approve/reject via the resolver; otherwise it applies to the matching effort
// decision. Events about asks Agent Manager did not raise, and duplicate
// deliveries, are acknowledged without change.
func DecisionResolvedWebhook(service *supervision.Service, resolver RunApprovalResolver, secret string) http.Handler {
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
		// Run-approval resolutions (needs_review) drive approve/reject; the
		// resolver reports handled=false for any other decision, which then
		// falls through to the effort-decision resolver below.
		if resolver != nil {
			if handled, rErr := resolver.ResolveRunApprovalFromHub(r.Context(), envelope.Payload); rErr != nil {
				http.Error(w, "apply run approval resolution: "+rErr.Error(), http.StatusInternalServerError)
				return
			} else if handled {
				_ = json.NewEncoder(w).Encode(map[string]any{"run_approval": true, "ask_id": envelope.Payload.AskID, "state": envelope.Payload.State})
				return
			}
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
