package integrations

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"notification-hub/internal/hub"
)

// DecisionRequestPattern subscribes the hub to every operator decision any
// scenario raises through Vrooli Events, for example
// agent_manager.effort.decision_requested.v1.
const (
	DecisionRequestPattern       = "**.decision_requested.v1"
	decisionRequestSuffix        = ".decision_requested.v1"
	decisionDefaultDeadline      = 24 * time.Hour
	decisionSensitivityLabel     = "private"
	maxDecisionQuestionLength    = 600
	maxDecisionSubjectLength     = 80
	maxDecisionRecommendationLen = 300
)

// IsDecisionRequest reports whether an event type asks the operator for a
// structured decision.
func IsDecisionRequest(eventType string) bool {
	return strings.HasSuffix(strings.TrimSpace(eventType), decisionRequestSuffix)
}

// decisionFacts is the producer contract for *.decision_requested.v1. The
// producer supplies facts only; the hub owns the notification copy.
type decisionFacts struct {
	Subject              string            `json:"subject"`
	Question             string            `json:"question"`
	Options              []hub.AskOption   `json:"options"`
	Recommended          string            `json:"recommended"`
	RecommendationReason string            `json:"recommendation_reason"`
	DefaultAnswer        string            `json:"default_answer"`
	Reversible           bool              `json:"reversible"`
	Deadline             string            `json:"deadline"`
	Urgency              string            `json:"urgency"`
	ContextURL           string            `json:"context_url"`
	Correlation          map[string]string `json:"correlation"`
}

// DecisionAskSpec turns a decision-request event into an ask. The event id is
// the idempotency key, so a redelivered or re-published event returns the
// same ask and sends nothing new.
func DecisionAskSpec(recipient, sourceScenario, eventType, eventID string, raw json.RawMessage, now time.Time) (hub.AskSpec, error) {
	var facts decisionFacts
	if err := json.Unmarshal(raw, &facts); err != nil {
		return hub.AskSpec{}, fmt.Errorf("%w: decision payload is not valid JSON", hub.ErrInvalidArgument)
	}
	question := strings.TrimSpace(facts.Question)
	if question == "" || len(question) > maxDecisionQuestionLength {
		return hub.AskSpec{}, fmt.Errorf("%w: question is required and at most %d characters", hub.ErrInvalidArgument, maxDecisionQuestionLength)
	}
	if len(facts.Options) < 2 {
		return hub.AskSpec{}, fmt.Errorf("%w: a decision needs at least two options", hub.ErrInvalidArgument)
	}
	if facts.Recommended == "" {
		return hub.AskSpec{}, fmt.Errorf("%w: a decision needs a recommended option", hub.ErrInvalidArgument)
	}
	if len(facts.RecommendationReason) > maxDecisionRecommendationLen {
		return hub.AskSpec{}, fmt.Errorf("%w: recommendation_reason exceeds %d characters", hub.ErrInvalidArgument, maxDecisionRecommendationLen)
	}
	deadline := now.Add(decisionDefaultDeadline)
	if value := strings.TrimSpace(facts.Deadline); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return hub.AskSpec{}, fmt.Errorf("%w: deadline must be RFC 3339", hub.ErrInvalidArgument)
		}
		deadline = parsed
	}
	subject := strings.TrimSpace(facts.Subject)
	if len(subject) > maxDecisionSubjectLength {
		subject = subject[:maxDecisionSubjectLength]
	}
	title := "Decision needed"
	if subject != "" {
		title += ": " + subject
	}
	return hub.AskSpec{
		Recipient:            recipient,
		Title:                title,
		Question:             question,
		Options:              facts.Options,
		Recommended:          strings.TrimSpace(facts.Recommended),
		RecommendationReason: strings.TrimSpace(facts.RecommendationReason),
		DefaultAnswer:        strings.TrimSpace(facts.DefaultAnswer),
		Reversible:           facts.Reversible,
		Urgency:              facts.Urgency,
		ContextURL:           strings.TrimSpace(facts.ContextURL),
		Deadline:             deadline,
		SensitivityLabel:     decisionSensitivityLabel,
		IdempotencyKey:       eventID,
		Source:               sourceScenario,
		SourceEventType:      eventType,
		Correlation:          facts.Correlation,
	}, nil
}
