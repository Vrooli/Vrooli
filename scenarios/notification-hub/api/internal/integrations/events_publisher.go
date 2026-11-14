package integrations

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/eventbus"
)

// ErrEventsUnavailable means the vrooli-events base URL is unknown, so an
// event cannot be published yet. Callers keep the event in their outbox.
var ErrEventsUnavailable = errors.New("vrooli-events base URL is not resolved")

// EventsPublisher publishes domain events to the vrooli-events base URL the
// integration config currently holds. Unlike a bare eventbus.Client, an
// unknown base URL is an error rather than a silent no-op, so the hub's
// ask-resolved outbox retries instead of losing the event.
type EventsPublisher struct {
	BaseURL func() string
}

func (p EventsPublisher) PublishDomainEvent(ctx context.Context, event eventbus.DomainEvent) error {
	base := ""
	if p.BaseURL != nil {
		base = strings.TrimSpace(p.BaseURL())
	}
	if base == "" {
		return ErrEventsUnavailable
	}
	if err := (eventbus.Client{BaseURL: base}).PublishDomainEvent(ctx, event); err != nil {
		return err
	}
	// Only Agent Manager's decision route has an exact durable subscription
	// contract here. Keep its resolution in the outbox until Events confirms
	// the corresponding queue row; other producers retain their own routing.
	source, _ := event.Payload["source"].(string)
	if event.EventType != "notification_hub.ask.resolved.v1" || source != "agent-manager" {
		return nil
	}
	return reconcileAgentManagerResolution(ctx, base, event)
}

func reconcileAgentManagerResolution(ctx context.Context, base string, event eventbus.DomainEvent) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	base = strings.TrimRight(base, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/v1/subscriptions?owner=agent-manager", nil)
	if err != nil {
		return err
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("list Agent Manager subscriptions: %s", response.Status)
	}
	var subscriptions []struct {
		ID      int64  `json:"id"`
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
		Pattern string `json:"event_pattern"`
	}
	if err := json.NewDecoder(response.Body).Decode(&subscriptions); err != nil {
		return err
	}
	var id int64
	for _, sub := range subscriptions {
		if sub.Name == "agent-manager-decision-answers" && sub.Enabled && sub.Pattern == "notification_hub.ask.resolved.v1" {
			if id != 0 {
				return errors.New("multiple Agent Manager answer subscriptions")
			}
			id = sub.ID
		}
	}
	if id == 0 {
		return errors.New("Agent Manager answer subscription unavailable")
	}
	// This is the same deterministic identity used by eventbus.Client.
	seed, err := json.Marshal(event)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(seed)
	body, err := json.Marshal(map[string]string{"event_id": hex.EncodeToString(hash[:])})
	if err != nil {
		return err
	}
	req, err = http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/api/v1/subscriptions/%d/deliveries/reconcile", base, id), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	response, err = http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		return fmt.Errorf("reconcile Agent Manager answer delivery: %s", response.Status)
	}
	return nil
}
