package operatordecision

import (
	"context"
	"time"

	"github.com/vrooli/api-core/eventbus"
)

// Publisher publishes a decision event onto Vrooli Events. The api-core
// eventbus.Client satisfies it in production. It collapses the duplicate
// ApprovalEventPublisher (run_approval_events.go) and DecisionPublisher
// (supervision/effort_decisions.go) into one interface. A nil Publisher leaves a
// decision raised-but-unnotified (no regression): the durable record stands and a
// later retry can publish.
type Publisher interface {
	PublishDomainEvent(context.Context, eventbus.DomainEvent) error
}

// BuildEvent assembles the decision_requested DomainEvent a scenario publishes.
// It is the lower-level path a scenario with its OWN durable record (e.g. a run
// in needs_review) uses directly, WITHOUT a PendingStore or Raiser: build the
// facts, build the event with a deterministic occurred time, publish it.
//
// occurred MUST be deterministic for a given logical ask (e.g. the record's
// creation or approval time, or Clock.Now truncated to a window) so the EventID
// is stable across retries and a redelivery maps to the same ask rather than a
// duplicate notification. A zero occurred defers to eventbus, which stamps
// time.Now — acceptable only when the producer never retries.
func BuildEvent(source, kind string, occurred time.Time, facts Facts) eventbus.DomainEvent {
	return BuildEventTyped(source, DecisionRequestedEventType(source, kind), occurred, facts)
}

// BuildEventTyped is BuildEvent with the full EventType supplied explicitly
// instead of composed from source+kind. It exists for the PARITY case where a
// scenario's Vrooli Events source string and the source token embedded in its
// event type differ — e.g. agent-manager publishes with Source "agent-manager"
// (hyphen) but its established event types read "agent_manager.*" (underscore).
// A new scenario should prefer BuildEvent (one spelling); BuildEventTyped lets an
// existing producer adopt the substrate without changing the bytes it emits.
func BuildEventTyped(source, eventType string, occurred time.Time, facts Facts) eventbus.DomainEvent {
	return eventbus.DomainEvent{
		Source:    source,
		EventType: eventType,
		Occurred:  occurred.UTC(),
		Payload:   facts.Payload(),
	}
}

// DecisionResolution is notification-hub's ask.resolved payload. It ports
// supervision.DecisionResolution verbatim so phase 2 can alias the agent-manager
// type to this one. The fields a producer cares about are Answer (the chosen
// option key), State ("answered"/"defaulted"/"expired"), AnsweredBy (the verified
// human, where the hub carries one), SourceEventType (which producer event this
// resolves), and Correlation (the verbatim map that locates the durable record).
type DecisionResolution struct {
	AskID           string            `json:"ask_id"`
	State           string            `json:"state"`
	Answer          string            `json:"answer"`
	AnswerLabel     string            `json:"answer_label"`
	Note            string            `json:"note"`
	AnsweredBy      string            `json:"answered_by"`
	AnsweredAt      string            `json:"answered_at"`
	ByDefault       bool              `json:"by_default"`
	Late            bool              `json:"late"`
	Reason          string            `json:"reason"`
	SourceEventType string            `json:"source_event_type"`
	RequesterRef    string            `json:"requester_ref"`
	Correlation     map[string]string `json:"correlation"`
}

// Resolution states carried on a DecisionResolution.State.
const (
	StateAnswered  = "answered"
	StateDefaulted = "defaulted"
	StateExpired   = "expired"
)
