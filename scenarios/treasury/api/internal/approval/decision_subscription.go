package approval

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

const (
	DecisionAnswerSubscriptionName = "treasury-approval-decision-answers"
	DecisionAnswerEventPattern     = "notification_hub.ask.resolved.v1"
)

// ReconcileDecisionEventSubscription ensures Events has one Treasury-owned
// delivery route for Notification Hub answers. Events owns the durable row;
// this owner keeps the exact target and event pattern current across restarts.
func ReconcileDecisionEventSubscription(ctx context.Context, eventsBase, target string) error {
	base := strings.TrimRight(strings.TrimSpace(eventsBase), "/")
	if base == "" {
		resolved, err := discovery.ResolveScenarioURLDefault(ctx, "vrooli-events")
		if err != nil {
			return err
		}
		base = strings.TrimRight(resolved, "/")
	}
	if base == "" || strings.TrimSpace(target) == "" {
		return fmt.Errorf("Vrooli Events base URL and Treasury answer webhook target are required")
	}
	client := &http.Client{Timeout: 10 * time.Second}
	var existing []struct {
		ID             int64  `json:"id"`
		Name           string `json:"name"`
		EventPattern   string `json:"event_pattern"`
		DeliveryTarget string `json:"delivery_target"`
		Enabled        bool   `json:"enabled"`
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/v1/subscriptions?owner=treasury", nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("list Treasury event subscriptions: %s", response.Status)
	}
	if err := json.NewDecoder(response.Body).Decode(&existing); err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]any{
		"name": DecisionAnswerSubscriptionName, "owner_scenario": "treasury",
		"event_pattern": DecisionAnswerEventPattern, "delivery_type": "webhook",
		"delivery_target": strings.TrimSpace(target), "enabled": true,
	})
	if err != nil {
		return err
	}
	var named *int64
	for _, item := range existing {
		if item.Name != DecisionAnswerSubscriptionName {
			continue
		}
		if named != nil {
			return fmt.Errorf("multiple Treasury answer subscriptions are registered")
		}
		id := item.ID
		named = &id
		if item.Enabled && item.DeliveryTarget == strings.TrimSpace(target) && item.EventPattern == DecisionAnswerEventPattern {
			return nil
		}
	}
	method, endpoint := http.MethodPost, base+"/api/v1/subscriptions"
	if named != nil {
		method, endpoint = http.MethodPut, fmt.Sprintf("%s/api/v1/subscriptions/%d", base, *named)
	}
	request, err = http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err = client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		return fmt.Errorf("reconcile Treasury answer subscription: %s", response.Status)
	}
	return nil
}
