package destinations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
)

func (f *RecoveryDecisionFlow) finishWithoutRepair(ctx context.Context, ask *operatordecision.PendingAsk, detail, category string) (bool, error) {
	return true, f.queueResult(ctx, ask, operatordecision.PendingStateRejected, "",
		"Backup drive recovery needs attention", detail, category, "high")
}

func (f *RecoveryDecisionFlow) queueResult(ctx context.Context, ask *operatordecision.PendingAsk, state, by, title, body, category, urgency string) error {
	if ask.Extra == nil {
		ask.Extra = map[string]string{}
	}
	ask.State = state
	ask.ResolvedAt = ptrTime(f.now().UTC())
	ask.ResolvedBy = by
	ask.Extra["result_title"] = title
	ask.Extra["result_body"] = body
	ask.Extra["result_category"] = category
	ask.Extra["result_urgency"] = urgency
	ask.Extra["result_notification_pending"] = "true"
	if err := f.asks.Put(ctx, ask); err != nil {
		return fmt.Errorf("persist volume recovery result notice: %w", err)
	}
	return f.publishPendingResult(ctx, ask)
}

func (f *RecoveryDecisionFlow) publishPendingResult(ctx context.Context, ask *operatordecision.PendingAsk) error {
	if ask == nil || ask.Extra["result_notification_pending"] != "true" {
		return nil
	}
	if f.publisher == nil {
		return fmt.Errorf("notification publisher is unavailable")
	}
	event := recoveryNotification(ask.Extra["result_title"], ask.Extra["result_body"], ask.Extra["result_category"], ask.Extra["result_urgency"], ask.Subject, ask.ID)
	if ask.ResolvedAt != nil {
		event.Occurred = ask.ResolvedAt.UTC()
	}
	if err := f.publisher.PublishDomainEvent(ctx, event); err != nil {
		return fmt.Errorf("publish volume recovery result notice: %w", err)
	}
	ask.Extra["result_notification_pending"] = "false"
	if err := f.asks.Put(ctx, ask); err != nil {
		return fmt.Errorf("record volume recovery result notice: %w", err)
	}
	return nil
}

func ptrTime(value time.Time) *time.Time { return &value }

func recoveryNotification(title, body, category, urgency, recoveryID, askID string) eventbus.DomainEvent {
	body = fmt.Sprintf("%s\nOther currently actionable operator items: %s. Full supervisor action file: %s", body, currentOperatorActionsDigest(), operatorActionsFileFullPath)
	return eventbus.DomainEvent{
		Source:    recoveryDecisionSource,
		EventType: recoveryNotificationType,
		Occurred:  time.Now().UTC(),
		Payload: map[string]any{
			"title": title, "body": body, "category": category, "urgency": urgency,
			"sensitivity": "private", "dedupe_key": filepath.Clean(recoveryID) + ":" + askID + ":" + category,
			"correlation": map[string]any{"recovery_id": recoveryID, "ask_id": askID},
		},
	}
}

// reconcileRecoveryDecisionSubscription keeps a single resolution delivery
// route pointing to DBM's signed callback. The route is durable in
// vrooli-events; DBM retries reconciliation on startup without blocking its
// API listener.
func reconcileRecoveryDecisionSubscription(ctx context.Context, target string) error {
	base, err := discovery.ResolveScenarioURLDefault(ctx, "vrooli-events")
	if err != nil {
		return err
	}
	base = strings.TrimRight(base, "/")
	client := &http.Client{Timeout: 10 * time.Second}
	const owner = "data-backup-manager"
	pattern := operatordecision.AskResolvedEventType
	var existing []struct {
		ID             int64  `json:"id"`
		EventPattern   string `json:"event_pattern"`
		DeliveryTarget string `json:"delivery_target"`
		Enabled        bool   `json:"enabled"`
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/v1/subscriptions?owner="+owner, nil)
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
	payload, err := json.Marshal(map[string]any{
		"name": "data-backup-manager-volume-recovery-decisions", "owner_scenario": owner,
		"event_pattern": pattern, "delivery_type": "webhook", "delivery_target": target, "enabled": true,
	})
	if err != nil {
		return err
	}
	for _, item := range existing {
		if item.Enabled && item.DeliveryTarget == target && item.EventPattern == pattern {
			return nil
		}
		if item.Enabled && item.DeliveryTarget == target {
			req, err = http.NewRequestWithContext(ctx, http.MethodPut, fmt.Sprintf("%s/api/v1/subscriptions/%d", base, item.ID), bytes.NewReader(payload))
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
				return fmt.Errorf("update event subscription returned %s", resp.Status)
			}
			return nil
		}
	}
	req, err = http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/subscriptions", bytes.NewReader(payload))
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

var _ operatordecision.DomainEffect = (*RecoveryDecisionFlow)(nil)
