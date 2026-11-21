package main

import (
	"context"
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
	"github.com/vrooli/vrooli/packages/credential-authority-go/hubidentity"
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
	s.agentSessionSvc.SetProposalNoticeBaseResolver(discovery.ResolveExternalURLForNotice)
	publisher := eventbus.NewDiscoveredClient(context.Background())
	s.agentSessionSvc.SetDecisionEventPublisher(publisher)

	secret, err := resolveEventsWebhookSecret()
	if err != nil || strings.TrimSpace(secret) == "" {
		slog.Warn("proposal decision webhook not mounted: events webhook secret unavailable; proposals still raise and are decidable from the inbox", "err", err)
		return
	}
	effect := agentsessions.NewProposalDecisionEffect(s.attemptDecisions, s.agentSessionSvc)
	s.router.Handle(decisionEventWebhookPath, operatordecision.NewWebhook(hubidentity.WebhookConfig(operatordecision.WebhookConfig{
		Secret: operatordecision.StaticSecret(secret), Source: agentsessions.ProposalDecisionSource,
		Effects: []operatordecision.DomainEffect{effect},
	}))).Methods(http.MethodPost)

	if port := strings.TrimSpace(os.Getenv("API_PORT")); port != "" {
		go func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go func() {
				select {
				case <-s.proposalDecisionStop:
					cancel()
				case <-ctx.Done():
				}
			}()
			target := "http://127.0.0.1:" + port + decisionEventWebhookPath
			eventbus.RunSubscriptionReconciler(ctx, eventbus.SubscriptionReconcilerConfig{
				Interval: 30 * time.Second, AttemptTimeout: 15 * time.Second, Attempts: 1,
			}, func(ctx context.Context) error { return reconcileDecisionEventSubscription(ctx, target) }, func(err error) {
				slog.Warn("proposal decision answer subscription reconciliation failed; retry scheduled", "err", err)
			})
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
	return reconcileDecisionEventSubscriptionAt(ctx, base, target)
}

func reconcileDecisionEventSubscriptionAt(ctx context.Context, base, target string) error {
	client, err := eventbus.NewSubscriptionClient(base, nil)
	if err != nil {
		return err
	}
	const owner = "swarm-manager"
	const name = "swarm-manager-decision-answers"
	payload := eventbus.Subscription{
		Name: name, OwnerScenario: owner, EventPattern: askResolvedEventPattern,
		DeliveryType: "webhook", DeliveryTarget: target, Enabled: true,
	}
	return client.EnsureNamedSubscription(ctx, payload)
}
