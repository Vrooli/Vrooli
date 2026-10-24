package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/operatordecision"
)

const (
	decisionSubscriptionOwner = "deployment-manager"
	decisionSubscriptionName  = "deployment-manager-decision-answers"
)

// reconcileDecisionEventSubscription keeps exactly one enabled vrooli-events
// subscription that delivers notification-hub ask resolutions to the
// deployment-manager release-decision webhook. Without it an answered release
// decision is never delivered back, so the ask-don't-block round-trip is
// dead-on-arrival. vrooli-events owns the durable row; this only ensures it
// exists and targets the live webhook. Ported from swarm-manager's
// reconcileDecisionEventSubscription.
func reconcileDecisionEventSubscription(ctx context.Context, target string) error {
	base, err := discovery.ResolveScenarioURLDefault(ctx, "vrooli-events")
	if err != nil {
		return err
	}
	base = strings.TrimRight(base, "/")
	client := &http.Client{Timeout: 10 * time.Second}
	const pattern = operatordecision.AskResolvedEventType
	var existing []struct {
		ID             int64  `json:"id"`
		EventPattern   string `json:"event_pattern"`
		DeliveryTarget string `json:"delivery_target"`
		Enabled        bool   `json:"enabled"`
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/v1/subscriptions?owner="+decisionSubscriptionOwner, nil)
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
	payload := map[string]any{
		"name":            decisionSubscriptionName,
		"owner_scenario":  decisionSubscriptionOwner,
		"event_pattern":   pattern,
		"delivery_type":   "webhook",
		"delivery_target": target,
		"enabled":         true,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	for _, item := range existing {
		if item.Enabled && item.DeliveryTarget == target && item.EventPattern == pattern {
			return nil
		}
		if item.Enabled && item.DeliveryTarget == target {
			return putDecisionSubscription(ctx, client, base, item.ID, body)
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

func putDecisionSubscription(ctx context.Context, client *http.Client, base string, id int64, body []byte) error {
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
