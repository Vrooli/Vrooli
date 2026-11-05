package integrations

import (
	"encoding/json"
	"strings"

	"notification-hub/internal/hub"
)

const notificationRequestSuffix = ".notification.v1"

// IsNotificationRequest reports whether an event type carries a domain-neutral
// FYI notification. It mirrors IsDecisionRequest.
func IsNotificationRequest(eventType string) bool {
	return strings.HasSuffix(strings.TrimSpace(eventType), notificationRequestSuffix)
}

// notificationFacts is the domain-neutral producer contract, parallel to
// decisionFacts. The producer supplies facts; body is the only required field.
// Unlike the legacy incident path, the producer declares its own sensitivity
// (validated to one of public|private|sensitive|critical, default private),
// its own category, and its own dedupe key.
type notificationFacts struct {
	Title       string            `json:"title"`
	Body        string            `json:"body"`
	Urgency     string            `json:"urgency"`
	Sensitivity string            `json:"sensitivity"`
	Category    string            `json:"category"`
	ContextURL  string            `json:"context_url"`
	DedupeKey   string            `json:"dedupe_key"`
	ScopeKey    string            `json:"scope_key"`
	Correlation map[string]string `json:"correlation"`
}

var validNotificationSensitivities = map[string]bool{
	"public":    true,
	"private":   true,
	"sensitive": true,
	"critical":  true,
}

// parseNotificationFacts decodes the neutral facts. ok is false when the
// payload does not parse into a usable notification (no non-empty body), which
// is what lets the legacy incident renderer stay the fallback for events that
// carry no neutral facts.
func parseNotificationFacts(raw json.RawMessage) (notificationFacts, bool) {
	var facts notificationFacts
	if len(raw) == 0 {
		return notificationFacts{}, false
	}
	if err := json.Unmarshal(raw, &facts); err != nil {
		return notificationFacts{}, false
	}
	if strings.TrimSpace(facts.Body) == "" {
		return notificationFacts{}, false
	}
	return facts, true
}

// notificationSendInput turns neutral facts into a hub.SendInput. The event id
// is the idempotency key; the dedupe key defaults to the event type (never an
// incident id); urgency defaults to "normal"; sensitivity defaults to
// "private" and is clamped to the allowed set.
func notificationSendInput(recipient, sourceScenario, eventType, eventID string, facts notificationFacts) hub.SendInput {
	urgency := strings.TrimSpace(facts.Urgency)
	if urgency == "" {
		urgency = "normal"
	}
	sensitivity := strings.ToLower(strings.TrimSpace(facts.Sensitivity))
	if !validNotificationSensitivities[sensitivity] {
		sensitivity = "private"
	}
	dedupeKey := strings.TrimSpace(facts.DedupeKey)
	if dedupeKey == "" {
		dedupeKey = eventType
	}
	return hub.SendInput{
		RequestedBy:      recipient,
		Title:            strings.TrimSpace(facts.Title),
		Body:             strings.TrimSpace(facts.Body),
		Urgency:          urgency,
		SensitivityLabel: sensitivity,
		IdempotencyKey:   eventID,
		DedupeKey:        dedupeKey,
		SourceScenario:   sourceScenario,
		Category:         strings.TrimSpace(facts.Category),
		ContextURL:       strings.TrimSpace(facts.ContextURL),
		ScopeKey:         strings.TrimSpace(facts.ScopeKey),
	}
}
