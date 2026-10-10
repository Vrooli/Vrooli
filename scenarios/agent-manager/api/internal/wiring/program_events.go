package wiring

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
	programRuntimeEventPattern = "vrooli.program_runtime.program_event.v1"
	// askResolvedEventPattern carries operator answers to effort decisions.
	askResolvedEventPattern = "notification_hub.ask.resolved.v1"
)

// ReconcileProgramRuntimeSubscription creates or updates the program-runtime
// subscription owned by agent-manager. vrooli-events remains the owner of the
// durable row.
func ReconcileProgramRuntimeSubscription(ctx context.Context, target string) error {
	return reconcileEventSubscription(ctx, "agent-manager-program-events", programRuntimeEventPattern, target)
}

// ReconcileDecisionEventSubscription creates or updates the subscription that
// delivers notification-hub ask resolutions to the effort decision webhook.
func ReconcileDecisionEventSubscription(ctx context.Context, target string) error {
	return reconcileEventSubscription(ctx, "agent-manager-decision-answers", askResolvedEventPattern, target)
}

// reconcileEventSubscription keeps one enabled subscription per delivery
// target.
func reconcileEventSubscription(ctx context.Context, name, pattern, target string) error {
	base, err := discovery.ResolveScenarioURLDefault(ctx, "vrooli-events")
	if err != nil {
		return err
	}
	base = strings.TrimRight(base, "/")
	client := &http.Client{Timeout: 10 * time.Second}
	var existing []struct {
		ID             int64  `json:"id"`
		EventPattern   string `json:"event_pattern"`
		DeliveryTarget string `json:"delivery_target"`
		Enabled        bool   `json:"enabled"`
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/v1/subscriptions?owner=agent-manager", nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("list event subscriptions returned %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(&existing); err != nil {
		return err
	}
	payload := map[string]any{"name": name, "owner_scenario": "agent-manager", "event_pattern": pattern, "delivery_type": "webhook", "delivery_target": target, "enabled": true}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	for _, item := range existing {
		if item.Enabled && item.DeliveryTarget == target && item.EventPattern == pattern {
			return nil
		}
		if item.Enabled && item.DeliveryTarget == target {
			return putSubscription(ctx, client, base, item.ID, body)
		}
	}
	req, err = http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/subscriptions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("create event subscription returned %s", resp.Status)
	}
	return nil
}

func putSubscription(ctx context.Context, client *http.Client, base string, id int64, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, fmt.Sprintf("%s/api/v1/subscriptions/%d", base, id), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("update event subscription returned %s", resp.Status)
	}
	return nil
}
