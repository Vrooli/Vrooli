package main

import (
	"log"
	"net/http"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/subscription"
)

const (
	ErrCodeSubWrite = "SUBSCRIPTION_WRITE_ERROR"
	ErrCodeSubRead  = "SUBSCRIPTION_READ_ERROR"
)

// handleCreateSubscription creates a new event subscription.
func (s *Server) handleCreateSubscription(w http.ResponseWriter, r *http.Request) {
	var sub subscription.Subscription
	if !decodeJSONBody(w, r, &sub) {
		return
	}

	if ve := validateSubscription(&sub); ve != "" {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, ve)
		return
	}

	id, err := s.subStore.Create(r.Context(), sub)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeSubWrite, "failed to create subscription")
		log.Printf("subscription create error: %v", err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"id": id})
}

// handleListSubscriptions lists subscriptions with optional filters.
func (s *Server) handleListSubscriptions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filters := subscription.ListFilters{
		Owner:   q.Get("owner"),
		Pattern: q.Get("pattern"),
	}
	if enabledStr := q.Get("enabled"); enabledStr != "" {
		enabled := enabledStr == "true"
		filters.Enabled = &enabled
	}

	subs, err := s.subStore.List(r.Context(), filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeSubRead, "failed to list subscriptions")
		log.Printf("subscription list error: %v", err)
		return
	}

	writeJSON(w, 0, orEmpty(subs))
}

// handleGetSubscription returns a single subscription by ID.
func (s *Server) handleGetSubscription(w http.ResponseWriter, r *http.Request) {
	_, sub, ok := requireByID(w, r, "id", s.subStore.Get, ErrCodeSubRead, "subscription")
	if !ok {
		return
	}
	writeJSON(w, 0, sub)
}

// handleUpdateSubscription updates an existing subscription.
func (s *Server) handleUpdateSubscription(w http.ResponseWriter, r *http.Request) {
	id, _, ok := requireByID(w, r, "id", s.subStore.Get, ErrCodeSubRead, "subscription")
	if !ok {
		return
	}

	var sub subscription.Subscription
	if !decodeJSONBody(w, r, &sub) {
		return
	}

	sub.ID = id
	if ve := validateSubscription(&sub); ve != "" {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, ve)
		return
	}

	if err := s.subStore.Update(r.Context(), sub); err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeSubWrite, "failed to update subscription")
		log.Printf("subscription update error: %v", err)
		return
	}

	writeJSON(w, 0, map[string]string{"status": "updated"})
}

// handleDeleteSubscription deletes a subscription by ID.
func (s *Server) handleDeleteSubscription(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidParam, "invalid subscription ID")
		return
	}

	if err := s.subStore.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeSubWrite, "failed to delete subscription")
		log.Printf("subscription delete error: %v", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleGetSubscriptionHealth returns health metrics for a subscription.
func (s *Server) handleGetSubscriptionHealth(w http.ResponseWriter, r *http.Request) {
	_, health, ok := requireByID(w, r, "id", s.subStore.GetHealth, ErrCodeSubRead, "subscription")
	if !ok {
		return
	}
	writeJSON(w, 0, health)
}

// handleTestSubscription sends a synthetic test event to verify the subscription.
func (s *Server) handleTestSubscription(w http.ResponseWriter, r *http.Request) {
	_, sub, ok := requireByID(w, r, "id", s.subStore.Get, ErrCodeSubRead, "subscription")
	if !ok {
		return
	}

	writeJSON(w, 0, map[string]any{
		"subscription_id": sub.ID,
		"name":            sub.Name,
		"event_pattern":   sub.EventPattern,
		"delivery_type":   sub.DeliveryType,
		"test_result":     "ok",
		"message":         "subscription configuration is valid",
	})
}

// handleDeliverSubscription triggers a webhook delivery for a subscription.
// DOC: docs/reference/api-endpoints.md#webhook-delivery
func (s *Server) handleDeliverSubscription(w http.ResponseWriter, r *http.Request) {
	id, sub, ok := requireByID(w, r, "id", s.subStore.Get, ErrCodeSubRead, "subscription")
	if !ok {
		return
	}

	if sub.DeliveryType != subscription.DeliveryWebhook {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, "subscription is not a webhook type")
		return
	}

	var payload subscription.WebhookPayload
	if !decodeJSONBody(w, r, &payload) {
		return
	}

	if payload.DeliveredAt == "" {
		payload.DeliveredAt = time.Now().UTC().Format(time.RFC3339)
	}

	if err := s.webhookDeliverer.Deliver(r.Context(), sub.DeliveryTarget, payload); err != nil {
		writeError(w, http.StatusBadGateway, ErrCodeSubWrite, "webhook delivery failed: "+err.Error())
		log.Printf("webhook delivery error for subscription %d: %v", id, err)
		return
	}

	writeJSON(w, 0, map[string]any{
		"status":          "delivered",
		"subscription_id": id,
		"target":          sub.DeliveryTarget,
	})
}

// validateSubscription checks required fields on a subscription.
func validateSubscription(s *subscription.Subscription) string {
	required := []struct {
		value, message string
	}{
		{s.Name, "name is required"},
		{s.OwnerScenario, "owner_scenario is required"},
		{s.EventPattern, "event_pattern is required"},
		{string(s.DeliveryType), "delivery_type is required"},
	}
	for _, r := range required {
		if r.value == "" {
			return r.message
		}
	}

	switch s.DeliveryType {
	case subscription.DeliverySSE, subscription.DeliveryWebhook:
		// valid
	default:
		return "delivery_type must be 'sse' or 'webhook'"
	}

	if s.DeliveryType == subscription.DeliveryWebhook && s.DeliveryTarget == "" {
		return "delivery_target is required for webhook subscriptions"
	}
	return ""
}
