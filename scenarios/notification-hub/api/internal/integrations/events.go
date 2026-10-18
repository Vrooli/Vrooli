package integrations

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"notification-hub/internal/hub"
)

type eventPayload struct {
	EventID          string          `json:"event_id"`
	EventType        string          `json:"event_type"`
	SourceScenario   string          `json:"source_scenario"`
	Title            string          `json:"title"`
	Body             string          `json:"body"`
	Urgency          string          `json:"urgency"`
	SensitivityLabel string          `json:"sensitivity_label"`
	Payload          json.RawMessage `json:"payload"`
}

type heartbeatAttemptNotification struct {
	TeamID              string `json:"teamId"`
	AgentID             string `json:"agentId"`
	RunID               string `json:"runId"`
	OwnerObligationID   string `json:"ownerObligationId"`
	RetainedOwnerState  string `json:"retainedOwnerState"`
	ObservedAt          string `json:"observedAt"`
	Status              string `json:"status"`
	ConsecutiveFailures int    `json:"consecutiveFailures"`
	FailureEpisodeID    string `json:"failureEpisodeId"`
	RecoveryOutcome     string `json:"recoveryOutcome"`
	RecoveryRequestID   string `json:"recoveryRequestId"`
	ReviewURL           string `json:"reviewUrl"`
	AttentionEligible   bool   `json:"attentionEligible"`
}

// heartbeatAttemptFYI maps a definitive threshold-or-later failed attempt or
// a retained owner-state fact to one scoped operator FYI. Later events can
// recover a missed webhook while the stable identity prevents repeats.
func heartbeatAttemptFYI(eventType, sourceScenario, eventID string, raw json.RawMessage) (hub.SendInput, bool, error) {
	if sourceScenario != "prompt-manager" {
		return hub.SendInput{}, false, nil
	}
	var attempt heartbeatAttemptNotification
	if err := json.Unmarshal(raw, &attempt); err != nil {
		return hub.SendInput{}, false, fmt.Errorf("decode prompt-manager heartbeat attempt: %w", err)
	}
	if eventType == HeartbeatOwnerStateEventType {
		return heartbeatRetainedOwnerStateFYI(sourceScenario, eventID, attempt)
	}
	if eventType != HeartbeatAttemptEventType {
		return hub.SendInput{}, false, nil
	}
	if attempt.Status != "failed" || !attempt.AttentionEligible || attempt.ConsecutiveFailures < 3 {
		return hub.SendInput{}, false, nil
	}
	teamID := strings.TrimSpace(attempt.TeamID)
	agentID := strings.TrimSpace(attempt.AgentID)
	episodeID := strings.TrimSpace(attempt.FailureEpisodeID)
	if eventID == "" || teamID == "" || agentID == "" || episodeID == "" {
		return hub.SendInput{}, false, errors.New("threshold heartbeat attempt requires event, team, agent, and failure episode identity")
	}
	body := fmt.Sprintf("Prompt Manager recorded %d consecutive failed heartbeats for team %s and agent %s (the attention threshold is three). ", attempt.ConsecutiveFailures, teamID, agentID)
	switch attempt.RecoveryOutcome {
	case "request_submitted":
		if strings.TrimSpace(attempt.RecoveryRequestID) == "" {
			return hub.SendInput{}, false, errors.New("submitted heartbeat recovery request has no request id")
		}
		body += fmt.Sprintf("A recovery request (%s) was submitted. This does not mean it was approved or completed.", strings.TrimSpace(attempt.RecoveryRequestID))
	case "request_unconfirmed":
		body += "The recovery request outcome could not be confirmed; this does not establish whether it was created."
	case "request_suppressed":
		body += "No new recovery request was sent because an earlier request remains latched."
	case "requester_unavailable":
		body += "No recovery request was submitted because the recovery requester was unavailable."
	case "not_requested", "below_threshold":
		body += "No recovery request was submitted for this failure."
	default:
		return hub.SendInput{}, false, fmt.Errorf("unsupported heartbeat recovery outcome %q", attempt.RecoveryOutcome)
	}
	return hub.SendInput{
		RequestedBy:      sourceScenario,
		Title:            "Prompt Manager heartbeat needs review",
		Body:             body,
		Urgency:          "normal",
		SensitivityLabel: "private",
		IdempotencyKey:   episodeID,
		DedupeKey:        HeartbeatAttemptEventType + ":" + episodeID,
		SourceScenario:   sourceScenario,
		Category:         "prompt_manager_heartbeat",
		ContextURL:       strings.TrimSpace(attempt.ReviewURL),
		ScopeKey:         "prompt-manager:" + teamID + ":" + agentID,
	}, true, nil
}

func heartbeatRetainedOwnerStateFYI(sourceScenario, eventID string, fact heartbeatAttemptNotification) (hub.SendInput, bool, error) {
	teamID := strings.TrimSpace(fact.TeamID)
	agentID := strings.TrimSpace(fact.AgentID)
	ownerID := strings.TrimSpace(fact.OwnerObligationID)
	state := strings.TrimSpace(fact.RetainedOwnerState)
	if eventID == "" || teamID == "" || agentID == "" || ownerID == "" {
		return hub.SendInput{}, false, errors.New("retained heartbeat owner state requires event, team, agent, and obligation identity")
	}
	if state == "preparation_interrupted" && strings.TrimSpace(fact.RunID) != "" {
		return hub.SendInput{}, false, errors.New("preparation-interrupted heartbeat owner state cannot include a run id")
	}
	var stateDescription string
	switch state {
	case "preparation_interrupted":
		stateDescription = "Prompt Manager was interrupted before sending the first Agent Manager task or run request for this accepted heartbeat obligation. The obligation remains retained. This does not establish that a task or run exists. It needs a later ordinary enabled scheduled admission or a fresh authorized manual admission."
	case "dispatch_uncertain":
		stateDescription = "Prompt Manager retained this heartbeat's run-start obligation because Agent Manager's response did not establish whether it accepted the run. Review the current team and member state; PM has not released ownership or started a replacement heartbeat."
	case "owner_unreachable":
		stateDescription = "Prompt Manager could not confirm the owner run state and retained the execution slot. Review the owner status; the missing observation does not prove that the run is absent."
	case "needs_review":
		stateDescription = "Agent Manager reports that this heartbeat's run needs review. Prompt Manager retained the execution slot pending owner-reported terminal state."
	case "watcher_unavailable":
		stateDescription = "Prompt Manager's local completion observer is unavailable. The owner run has not been proven terminal, so the execution slot remains retained for reconciliation."
	default:
		return hub.SendInput{}, false, fmt.Errorf("unsupported retained heartbeat owner state %q", state)
	}
	if runID := strings.TrimSpace(fact.RunID); runID != "" {
		stateDescription += fmt.Sprintf(" Owner run: %s.", runID)
	}
	obligationKey := "prompt-manager:" + teamID + ":" + agentID + ":" + ownerID + ":" + state
	return hub.SendInput{
		RequestedBy:      sourceScenario,
		Title:            "Prompt Manager heartbeat owner state needs review",
		Body:             fmt.Sprintf("Team %s, agent %s: %s", teamID, agentID, stateDescription),
		Urgency:          "normal",
		SensitivityLabel: "private",
		IdempotencyKey:   obligationKey,
		DedupeKey:        HeartbeatAttemptEventType + ":owner-state:" + obligationKey,
		SourceScenario:   sourceScenario,
		Category:         "prompt_manager_heartbeat",
		ContextURL:       strings.TrimSpace(fact.ReviewURL),
		ScopeKey:         "prompt-manager:" + teamID + ":" + agentID,
	}, true, nil
}

var (
	ErrEventIntegrationUnconfigured = errors.New("event integration is not configured")
	ErrProducerOwnedCopy            = errors.New("event renderer owns notification copy")
)

// EventWebhook is the single sanctioned REST exception for event ingress.
// The signature covers the exact bytes that are decoded, and the event id is
// used as the notification idempotency key.
func EventWebhook(service *hub.Service, secret string) http.Handler {
	return EventWebhookWithTemplates(service, secret, nil)
}

func EventWebhookWithTemplates(service *hub.Service, secret string, templates func() map[string]EventTemplate) http.Handler {
	return EventWebhookWithTemplatesAndSensitivity(service, secret, templates, nil)
}

func EventWebhookWithTemplatesAndSensitivity(service *hub.Service, secret string, templates func() map[string]EventTemplate, sensitivity func() map[string]string) http.Handler {
	return EventWebhookFromSecretSource(service, StaticSecret(secret), templates, sensitivity)
}

// EventWebhookFromSecretSource reads the signing secret per request, so a
// secret that becomes resolvable after boot is used without a restart. An
// unavailable secret fails closed with 401.
func EventWebhookFromSecretSource(service *hub.Service, secretSource SecretSource, templates func() map[string]EventTemplate, sensitivity func() map[string]string) http.Handler {
	return EventWebhookFromSecretSourceWithDecisionDefaults(service, secretSource, templates, sensitivity, nil)
}

// DecisionDefaults carries the operator-configured decision sensitivity label
// and default deadline consulted when a *.decision_requested.v1 event omits its
// own deadline. A nil accessor keeps the package defaults.
type DecisionDefaults struct {
	Sensitivity     string
	DefaultDeadline time.Duration
}

// EventWebhookFromSecretSourceWithDecisionDefaults is EventWebhookFromSecretSource
// with the operator-owned decision defaults threaded in.
func EventWebhookFromSecretSourceWithDecisionDefaults(service *hub.Service, secretSource SecretSource, templates func() map[string]EventTemplate, sensitivity func() map[string]string, decisionDefaults func() DecisionDefaults) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		secret := ""
		if secretSource != nil {
			secret = secretSource()
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil || strings.TrimSpace(secret) == "" || !validSignature(body, r.Header.Get("X-Vrooli-Events-Signature"), secret) {
			writeJSONError(w, http.StatusUnauthorized, "event webhook signature is invalid or unavailable")
			return
		}
		var payload eventPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			writeJSONError(w, http.StatusBadRequest, "event webhook body is invalid")
			return
		}
		eventID := strings.TrimSpace(r.Header.Get("X-Vrooli-Events-Event-ID"))
		if eventID == "" {
			eventID = strings.TrimSpace(payload.EventID)
		}
		if payload.Title != "" || payload.Body != "" || payload.SensitivityLabel != "" {
			writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("%v: producers cannot provide title, body, or sensitivity_label", ErrProducerOwnedCopy))
			return
		}
		if IsDecisionRequest(payload.EventType) {
			if eventID == "" {
				writeJSONError(w, http.StatusBadRequest, "event_id is required")
				return
			}
			recipient := service.ResolveRecipient(r.Context())
			if recipient == "" {
				recipient = payload.SourceScenario
			}
			sensitivityLabel, defaultDeadline := decisionSensitivityLabel, decisionDefaultDeadline
			if decisionDefaults != nil {
				defaults := decisionDefaults()
				sensitivityLabel, defaultDeadline = defaults.Sensitivity, defaults.DefaultDeadline
			}
			spec, specErr := DecisionAskSpecWithDefaults(recipient, payload.SourceScenario, payload.EventType, eventID, payload.Payload, time.Now().UTC(), sensitivityLabel, defaultDeadline)
			if specErr == nil {
				var ask hub.AskRecord
				ask, _, specErr = service.AskWithSpec(r.Context(), spec)
				if specErr == nil {
					writeJSON(w, http.StatusAccepted, map[string]string{"ask_id": ask.ID, "event_id": eventID})
					return
				}
			}
			status := http.StatusInternalServerError
			if errors.Is(specErr, hub.ErrInvalidArgument) {
				status = http.StatusBadRequest
			} else if errors.Is(specErr, hub.ErrAskIdentityConflict) {
				status = http.StatusConflict
			}
			writeJSONError(w, status, specErr.Error())
			return
		}
		// The recipient is the operator (override, then operator state). With
		// neither set the notification is still recorded, owned by the source
		// scenario, and its delivery attempt is 'unroutable' with the setting
		// that fixes it; it is never silently dropped.
		recipient := service.ResolveRecipient(r.Context())
		if recipient == "" {
			recipient = payload.SourceScenario
		}
		if payload.EventType == HeartbeatAttemptEventType || payload.EventType == HeartbeatOwnerStateEventType {
			input, notify, mapErr := heartbeatAttemptFYI(payload.EventType, payload.SourceScenario, eventID, payload.Payload)
			if mapErr != nil {
				writeJSONError(w, http.StatusBadRequest, mapErr.Error())
				return
			}
			if !notify {
				writeJSON(w, http.StatusAccepted, map[string]string{"event_id": eventID, "ignored": "not a threshold-or-later failed heartbeat"})
				return
			}
			input.RequestedBy = recipient
			notification, sendErr := service.Send(r.Context(), input)
			if sendErr != nil {
				status := http.StatusInternalServerError
				if errors.Is(sendErr, hub.ErrInvalidArgument) {
					status = http.StatusBadRequest
				}
				writeJSONError(w, status, sendErr.Error())
				return
			}
			writeJSON(w, http.StatusAccepted, map[string]string{"notification_id": notification.ID, "event_id": eventID})
			return
		}
		// Primary path: a domain-neutral FYI notification. It fires when the
		// event type carries the notification suffix OR the payload parses into
		// usable neutral facts (a non-empty body). The legacy incident renderer
		// below stays the fallback for events that carry no neutral facts.
		if IsNotificationRequest(payload.EventType) || func() bool { _, ok := parseNotificationFacts(payload.Payload); return ok }() {
			facts, ok := parseNotificationFacts(payload.Payload)
			if !ok || eventID == "" {
				writeJSONError(w, http.StatusBadRequest, "event_id and a non-empty body are required")
				return
			}
			notification, err := service.Send(r.Context(), notificationSendInput(recipient, payload.SourceScenario, payload.EventType, eventID, facts))
			if err != nil {
				status := http.StatusInternalServerError
				if errors.Is(err, hub.ErrInvalidArgument) {
					status = http.StatusBadRequest
				}
				writeJSONError(w, status, err.Error())
				return
			}
			writeJSON(w, http.StatusAccepted, map[string]string{"notification_id": notification.ID, "event_id": eventID})
			return
		}
		var configured map[string]EventTemplate
		if templates != nil {
			configured = templates()
		}
		var configuredSensitivity map[string]string
		if sensitivity != nil {
			configuredSensitivity = sensitivity()
		}
		rendered := RenderEventWithTemplatesAndSensitivity(payload.EventType, payload.Payload, configured, configuredSensitivity)
		if eventID == "" || rendered.Body == "" {
			writeJSONError(w, http.StatusBadRequest, "event_id, body, and sensitivity_label are required")
			return
		}
		if payload.EventType == "incident.remediation_approval_requested.v1" {
			ask, _, askErr := service.AskWithSpec(r.Context(), hub.AskSpec{
				Recipient:        recipient,
				Title:            rendered.Title,
				Question:         rendered.Body,
				Options:          []hub.AskOption{{Key: "approve", Label: "Approve"}, {Key: "reject", Label: "Reject"}},
				Deadline:         time.Now().UTC().Add(24 * time.Hour),
				SensitivityLabel: rendered.SensitivityLabel,
				IdempotencyKey:   eventID,
				Source:           payload.SourceScenario,
				SourceEventType:  payload.EventType,
			})
			askID := ask.ID
			if askErr != nil {
				status := http.StatusInternalServerError
				if errors.Is(askErr, hub.ErrInvalidArgument) {
					status = http.StatusBadRequest
				}
				writeJSONError(w, status, askErr.Error())
				return
			}
			writeJSON(w, http.StatusAccepted, map[string]string{"ask_id": askID, "event_id": eventID})
			return
		}
		notification, err := service.Send(r.Context(), hub.SendInput{RequestedBy: recipient, Title: rendered.Title, Body: rendered.Body, SensitivityLabel: rendered.SensitivityLabel, IdempotencyKey: eventID, DedupeKey: eventDedupeKey(payload.EventType, payload.Payload)})
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, hub.ErrInvalidArgument) {
				status = http.StatusBadRequest
			}
			writeJSONError(w, status, err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"notification_id": notification.ID, "event_id": eventID})
	})
}

func eventDedupeKey(eventType string, raw json.RawMessage) string {
	var facts struct {
		IncidentID string `json:"incident_id"`
	}
	_ = json.Unmarshal(raw, &facts)
	if strings.TrimSpace(facts.IncidentID) == "" {
		return eventType
	}
	return eventType + ":" + facts.IncidentID
}

type RenderedEvent struct {
	Title            string
	Body             string
	SensitivityLabel string
}

// RenderEvent owns notification copy. Event producers supply facts only.
func RenderEvent(eventType string, raw json.RawMessage) RenderedEvent {
	return RenderEventWithTemplates(eventType, raw, nil)
}

// RenderEventWithTemplates applies operator-owned templates to event facts.
// The producer still supplies facts only; templates can select facts but
// cannot override the server-owned sensitivity policy.
func RenderEventWithTemplates(eventType string, raw json.RawMessage, templates map[string]EventTemplate) RenderedEvent {
	return RenderEventWithTemplatesAndSensitivity(eventType, raw, templates, nil)
}

func RenderEventWithTemplatesAndSensitivity(eventType string, raw json.RawMessage, templates map[string]EventTemplate, sensitivityBySeverity map[string]string) RenderedEvent {
	if len(sensitivityBySeverity) == 0 {
		sensitivityBySeverity = map[string]string{"critical": "critical", "warning": "sensitive", "informational": "public"}
	}
	var facts struct {
		CheckID         string `json:"check_id"`
		SourceCheckID   string `json:"source_check_id"`
		Severity        string `json:"severity"`
		Status          string `json:"status"`
		Message         string `json:"message"`
		Reason          string `json:"reason"`
		IncidentTitle   string `json:"incident_title"`
		IncidentSummary string `json:"incident_summary"`
		CandidateID     string `json:"candidate_id"`
		CandidateTitle  string `json:"candidate_title"`
	}
	_ = json.Unmarshal(raw, &facts)
	if facts.CheckID == "" {
		facts.CheckID = facts.SourceCheckID
	}
	severity := strings.ToLower(strings.TrimSpace(facts.Severity))
	sensitivity := sensitivityBySeverity[severity]
	if sensitivity == "" {
		sensitivity = sensitivityBySeverity["informational"]
	}
	if sensitivity == "" {
		sensitivity = "public"
	}
	label := facts.CheckID
	if label == "" {
		label = "host incident"
	}
	body := facts.Message
	if body == "" {
		body = facts.Reason
	}
	if body == "" {
		body = facts.IncidentSummary
	}
	if body == "" {
		body = "Incident status: " + nonEmpty(facts.Status, "unknown")
	}
	if eventType == "incident.remediation_approval_requested.v1" {
		candidate := nonEmpty(facts.CandidateTitle, facts.CandidateID)
		if candidate == "" {
			candidate = "the proposed remediation"
		}
		return RenderedEvent{"Remediation approval required", "Approve remediation \"" + candidate + "\" for " + nonEmpty(facts.IncidentTitle, label) + "? Choose approve or reject.\nEvidence: " + body, sensitivity}
	}
	if template, ok := templates[eventType]; ok {
		title := renderTemplate(template.Title, eventType, label, severity, body, facts.Reason)
		templatedBody := renderTemplate(template.Body, eventType, label, severity, body, facts.Reason)
		if title != "" {
			return RenderedEvent{title, nonEmpty(templatedBody, body), sensitivity}
		}
		if templatedBody != "" {
			return RenderedEvent{"Vrooli event: " + eventType, templatedBody, sensitivity}
		}
	}
	switch eventType {
	case "incident.opened.v1":
		return RenderedEvent{"Incident opened: " + label, body, sensitivity}
	case "incident.severity_changed.v1":
		return RenderedEvent{"Incident severity changed: " + label, body, sensitivity}
	case "incident.resolved.v1":
		return RenderedEvent{"Incident resolved: " + label, body, sensitivity}
	default:
		return RenderedEvent{"Vrooli event: " + eventType, body, sensitivity}
	}
}

func renderTemplate(template, eventType, label, severity, message, reason string) string {
	return strings.NewReplacer(
		"{{event_type}}", eventType,
		"{{check_id}}", label,
		"{{severity}}", severity,
		"{{message}}", message,
		"{{reason}}", reason,
	).Replace(strings.TrimSpace(template))
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

// Subscription names owned by the hub. The operator configures the pattern
// of the first; the decision subscription is part of the ask contract.
const (
	eventSubscriptionName               = "notification-hub-events"
	decisionSubscriptionName            = "notification-hub-decisions"
	notificationSubscriptionName        = "notification-hub-notifications"
	heartbeatAttemptSubscriptionName    = "notification-hub-prompt-manager-heartbeat-attempts"
	heartbeatOwnerStateSubscriptionName = "notification-hub-prompt-manager-heartbeat-owner-states"
	HeartbeatAttemptEventType           = "prompt-manager.heartbeat.attempt.v1"
	HeartbeatAttemptEventPattern        = "prompt-manager.heartbeat.attempt.v1"
	HeartbeatOwnerStateEventType        = "prompt-manager.heartbeat.owner-state.v1"
	HeartbeatOwnerStateEventPattern     = "prompt-manager.heartbeat.owner-state.v1"
)

// SubscriptionSpec is one named vrooli-events subscription delivered to the
// hub's webhook.
type SubscriptionSpec struct {
	Name    string
	Pattern string
}

// EnsureEventSubscription makes the optional vrooli-events integration
// durable and idempotent. It is startup reconciliation rather than a new
// control plane: the events scenario remains the owner of subscription state
// and retry delivery.
func EnsureEventSubscription(ctx context.Context, baseURL, target, pattern string) error {
	return EnsureEventSubscriptions(ctx, baseURL, target, []SubscriptionSpec{{Name: eventSubscriptionName, Pattern: pattern}})
}

// EnsureDecisionSubscription reconciles the subscription that turns every
// *.decision_requested.v1 event into an ask.
func EnsureDecisionSubscription(ctx context.Context, baseURL, target string) error {
	return EnsureEventSubscriptions(ctx, baseURL, target, []SubscriptionSpec{{Name: decisionSubscriptionName, Pattern: DecisionRequestPattern}})
}

// EnsureNotificationSubscription reconciles the subscription that turns every
// *.notification.v1 event into a domain-neutral FYI notification.
func EnsureNotificationSubscription(ctx context.Context, baseURL, target string) error {
	return EnsureEventSubscriptions(ctx, baseURL, target, []SubscriptionSpec{{Name: notificationSubscriptionName, Pattern: NotificationRequestPattern}})
}

// EnsureHeartbeatAttemptSubscription reconciles exact, separate subscriptions
// for immutable PM attempt and retained-owner-state facts. Both use the same
// signed Events webhook; owner-state facts never enter the bounded attempt feed.
func EnsureHeartbeatAttemptSubscription(ctx context.Context, baseURL, target string) error {
	return EnsureEventSubscriptions(ctx, baseURL, target, []SubscriptionSpec{
		{Name: heartbeatAttemptSubscriptionName, Pattern: HeartbeatAttemptEventPattern},
		{Name: heartbeatOwnerStateSubscriptionName, Pattern: HeartbeatOwnerStateEventPattern},
	})
}

// EnsureEventSubscriptions reconciles each named subscription to the hub's
// webhook. An enabled subscription with the same target and pattern is kept;
// one with the same name is updated in place; the unnamed legacy row of the
// primary subscription is adopted; anything else is created.
func EnsureEventSubscriptions(ctx context.Context, baseURL, target string, specs []SubscriptionSpec) error {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	target = strings.TrimSpace(target)
	if baseURL == "" || target == "" || len(specs) == 0 {
		return ErrEventIntegrationUnconfigured
	}
	for _, spec := range specs {
		if strings.TrimSpace(spec.Pattern) == "" || strings.TrimSpace(spec.Name) == "" {
			return ErrEventIntegrationUnconfigured
		}
	}
	client := &http.Client{Timeout: 10 * time.Second}
	listURL := baseURL + "/api/v1/subscriptions?owner=notification-hub"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("list event subscriptions: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		return fmt.Errorf("list event subscriptions returned %s", response.Status)
	}
	var existing []struct {
		ID             int64  `json:"id"`
		Name           string `json:"name"`
		EventPattern   string `json:"event_pattern"`
		DeliveryTarget string `json:"delivery_target"`
		Enabled        bool   `json:"enabled"`
	}
	_ = json.NewDecoder(response.Body).Decode(&existing)
	for _, spec := range specs {
		pattern := strings.TrimSpace(spec.Pattern)
		var update int64
		satisfied := false
		for _, item := range existing {
			if !item.Enabled || item.DeliveryTarget != target {
				continue
			}
			if item.EventPattern == pattern {
				satisfied = true
				break
			}
			if update == 0 && (item.Name == spec.Name || (item.Name == "" && spec.Name == eventSubscriptionName)) {
				update = item.ID
			}
		}
		if satisfied {
			continue
		}
		// vrooli-events treats PUT as a full resource replacement. Keep the
		// owner-controlled identity and delivery type in the update; sending
		// only the changed fields would fail validation and leave a legacy
		// pattern active.
		body, err := json.Marshal(map[string]any{
			"name":            spec.Name,
			"owner_scenario":  "notification-hub",
			"event_pattern":   pattern,
			"delivery_type":   "webhook",
			"delivery_target": target,
			"enabled":         true,
		})
		if err != nil {
			return err
		}
		method, url, verb := http.MethodPost, baseURL+"/api/v1/subscriptions", "create"
		if update != 0 {
			method, url, verb = http.MethodPut, fmt.Sprintf("%s/api/v1/subscriptions/%d", baseURL, update), "update"
		}
		if err := sendSubscription(ctx, client, method, url, body); err != nil {
			return fmt.Errorf("%s event subscription %s: %w", verb, spec.Name, err)
		}
	}
	return nil
}

func sendSubscription(ctx context.Context, client *http.Client, method, url string, body []byte) error {
	request, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		return fmt.Errorf("vrooli-events returned %s", response.Status)
	}
	return nil
}

func validSignature(body []byte, value, secret string) bool {
	value = strings.TrimSpace(strings.TrimPrefix(value, "sha256="))
	provided, err := hex.DecodeString(value)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := mac.Sum(nil)
	return len(provided) == len(expected) && subtle.ConstantTimeCompare(provided, expected) == 1
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
