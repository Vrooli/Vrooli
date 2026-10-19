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
	DecisionWithdrawalPattern    = "agent_manager.run.decision_withdrawn.v1"
	DecisionWithdrawalEventType  = DecisionWithdrawalPattern
	RunApprovalDecisionEventType = "agent_manager.run.decision_requested.v1"
	// NotificationRequestPattern subscribes the hub to every domain-neutral
	// FYI notification any scenario raises through Vrooli Events, for example
	// personal_planner.reminder.notification.v1.
	NotificationRequestPattern   = "**.notification.v1"
	decisionRequestSuffix        = ".decision_requested.v1"
	decisionDefaultDeadline      = 24 * time.Hour
	decisionSensitivityLabel     = "private"
	maxDecisionQuestionLength    = 600
	maxDecisionSubjectLength     = 80
	maxDecisionRecommendationLen = 300
)

// DecisionWithdrawal is the exact lifecycle reference for a request that no
// longer needs a human decision. It is separate from answer payloads and has
// no answer or actor identity.
type DecisionWithdrawal struct {
	RequesterRef    string `json:"requester_ref"`
	SourceEventType string `json:"source_event_type"`
	Reason          string `json:"reason"`
}

func IsDecisionWithdrawal(eventType string) bool {
	return strings.TrimSpace(eventType) == DecisionWithdrawalEventType
}

func ParseDecisionWithdrawal(raw json.RawMessage) (DecisionWithdrawal, error) {
	var withdrawal DecisionWithdrawal
	if err := json.Unmarshal(raw, &withdrawal); err != nil {
		return DecisionWithdrawal{}, fmt.Errorf("%w: decision withdrawal payload is not valid JSON", hub.ErrInvalidArgument)
	}
	withdrawal.RequesterRef = strings.TrimSpace(withdrawal.RequesterRef)
	withdrawal.SourceEventType = strings.TrimSpace(withdrawal.SourceEventType)
	withdrawal.Reason = strings.TrimSpace(withdrawal.Reason)
	if withdrawal.RequesterRef == "" || len(withdrawal.RequesterRef) > 128 || withdrawal.SourceEventType != RunApprovalDecisionEventType || (withdrawal.Reason != "verified_empty_diff" && withdrawal.Reason != "superseded_by_new_review") {
		return DecisionWithdrawal{}, fmt.Errorf("%w: decision withdrawal identity or reason is invalid", hub.ErrInvalidArgument)
	}
	return withdrawal, nil
}

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
	BigPicture           string            `json:"big_picture"`
	TechnicalDetails     string            `json:"technical_details"`
	Note                 string            `json:"note"`
	Correlation          map[string]string `json:"correlation"`
}

// DecisionAskSpec turns a decision-request event into an ask. The event id is
// the idempotency key, so a redelivered or re-published event returns the
// same ask and sends nothing new.
func DecisionAskSpec(recipient, sourceScenario, eventType, eventID string, raw json.RawMessage, now time.Time) (hub.AskSpec, error) {
	return DecisionAskSpecWithDefaults(recipient, sourceScenario, eventType, eventID, raw, now, decisionSensitivityLabel, decisionDefaultDeadline)
}

// DecisionAskSpecWithDefaults is DecisionAskSpec with the operator-configured
// sensitivity label and default deadline threaded in. An empty sensitivity or a
// non-positive deadline falls back to the package defaults, so the configured
// behavior is a superset of the hardcoded one.
func DecisionAskSpecWithDefaults(recipient, sourceScenario, eventType, eventID string, raw json.RawMessage, now time.Time, sensitivityLabel string, defaultDeadline time.Duration) (hub.AskSpec, error) {
	if strings.TrimSpace(sensitivityLabel) == "" {
		sensitivityLabel = decisionSensitivityLabel
	}
	if defaultDeadline <= 0 {
		defaultDeadline = decisionDefaultDeadline
	}
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
	deadline := now.Add(defaultDeadline)
	deadlineSpecified := strings.TrimSpace(facts.Deadline) != ""
	if value := strings.TrimSpace(facts.Deadline); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return hub.AskSpec{}, fmt.Errorf("%w: deadline must be RFC 3339", hub.ErrInvalidArgument)
		}
		deadline = parsed
	}
	technicalDetails := facts.TechnicalDetails
	if strings.TrimSpace(technicalDetails) == "" {
		technicalDetails = facts.Note
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
		BigPicture:           facts.BigPicture,
		TechnicalDetails:     technicalDetails,
		Deadline:             deadline,
		DeadlineSpecified:    deadlineSpecified,
		SensitivityLabel:     sensitivityLabel,
		IdempotencyKey:       eventID,
		Source:               sourceScenario,
		SourceEventType:      eventType,
		Correlation:          facts.Correlation,
	}, nil
}
