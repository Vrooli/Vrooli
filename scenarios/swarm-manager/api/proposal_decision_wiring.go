package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"swarm-manager/internal/agentsessions"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

const (
	// eventsWebhookIdentity and eventsWebhookField are the shared credential the
	// vrooli-events webhook is signed with. They match agent-manager's
	// events_secret.go byte-for-byte: the SAME value vrooli-events uses to sign the
	// resolution webhook body, so a secret minted here verifies signatures
	// identically whether agent-manager or swarm-manager resolved it first.
	eventsWebhookIdentity = "vrooli/vrooli-events"
	eventsWebhookField    = "agent-manager-webhook-secret"
)

// resolveEventsWebhookSecret resolves the shared vrooli-events webhook secret:
// the VROOLI_EVENTS_WEBHOOK_SECRET env override first, then the credential
// authority (resolve-or-mint under the shared identity/field, mirroring
// agent-manager's resolver). It returns "" only when the env is unset AND the
// credential authority is unavailable, which is the one case the caller still
// refuses to mount the webhook for.
func resolveEventsWebhookSecret() (string, error) {
	if env := strings.TrimSpace(os.Getenv("VROOLI_EVENTS_WEBHOOK_SECRET")); env != "" {
		return env, nil
	}
	identity, err := credentialauthority.ParseIdentity(eventsWebhookIdentity)
	if err != nil {
		return "", fmt.Errorf("parse events webhook credential identity: %w", err)
	}
	authority, err := credentialauthority.Default()
	if err != nil {
		return "", fmt.Errorf("credential authority unavailable: %w", err)
	}
	return authority.ResolveOrMint(identity, eventsWebhookField, nil, func() (string, error) {
		return credentialauthority.RandomBase64(32)
	})
}

// decisionEventWebhookPath is the notification-hub ask.resolved webhook route
// swarm-manager mounts to receive operator answers for proposal decisions. It
// mirrors agent-manager's DecisionEventWebhookPath.
const decisionEventWebhookPath = "/api/v1/events/notification-hub"

// askResolvedEventPattern is the event vrooli-events delivers to the webhook.
const askResolvedEventPattern = operatordecision.AskResolvedEventType

// wireProposalDecisionRail seats swarm-manager on the shared operatordecision
// rail: it installs the Vrooli Events publisher that raises a ready proposal's
// decision event, mounts the resolution webhook that applies the operator's
// answer through the SAME attempt decider (single mutation authority), and
// reconciles the vrooli-events subscription so answers are delivered back.
//
// Degradations are best-effort and non-fatal, matching agent-manager:
//   - No events base yet → the publisher still constructs (it resolves lazily);
//     raises retry on the next proposal.
//   - No webhook secret AND no credential authority → the webhook is NOT mounted
//     (an unsigned resolution endpoint would be unauthenticated); proposals still
//     raise and the operator can still decide from the inbox. The secret is the
//     same shared credential agent-manager uses, resolved from the env override or
//     minted through the credential authority, so no env config is required in the
//     common case.
func (s *Server) wireProposalDecisionRail() {
	if s.agentSessionSvc == nil || s.attemptDecisions == nil {
		return
	}
	// The publisher resolves vrooli-events lazily, so it is safe to construct even
	// before that scenario is ready.
	publisher := eventbus.NewDiscoveredClient(context.Background())
	s.agentSessionSvc.SetDecisionEventPublisher(publisher)

	secret, err := resolveEventsWebhookSecret()
	if err != nil || strings.TrimSpace(secret) == "" {
		slog.Warn("proposal decision webhook not mounted: events webhook secret unavailable; proposals still raise and are decidable from the inbox", "err", err)
		return
	}
	effect := agentsessions.NewProposalDecisionEffect(s.attemptDecisions, s.agentSessionSvc)
	s.router.Handle(decisionEventWebhookPath, operatordecision.Webhook(secret, effect)).Methods(http.MethodPost)

	if port := strings.TrimSpace(os.Getenv("API_PORT")); port != "" {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			target := "http://127.0.0.1:" + port + decisionEventWebhookPath
			if err := reconcileDecisionEventSubscription(ctx, target); err != nil {
				slog.Warn("proposal decision answer subscription reconciliation failed", "err", err)
			}
		}()
	}
}

// reconcileDecisionEventSubscription keeps one enabled vrooli-events subscription
// that delivers notification-hub ask resolutions to the swarm-manager webhook.
// vrooli-events owns the durable row; this only ensures it exists and targets the
// live webhook. Ported from agent-manager's reconcileEventSubscription.
func reconcileDecisionEventSubscription(ctx context.Context, target string) error {
	base, err := discovery.ResolveScenarioURLDefault(ctx, "vrooli-events")
	if err != nil {
		return err
	}
	base = strings.TrimRight(base, "/")
	client := &http.Client{Timeout: 10 * time.Second}
	const owner = "swarm-manager"
	const name = "swarm-manager-decision-answers"
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
	payload := map[string]any{"name": name, "owner_scenario": owner, "event_pattern": askResolvedEventPattern, "delivery_type": "webhook", "delivery_target": target, "enabled": true}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	for _, item := range existing {
		if item.Enabled && item.DeliveryTarget == target && item.EventPattern == askResolvedEventPattern {
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
