package channelmanager

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

// ActivityEvent is an immutable, redacted account-operation fact. Mutable
// action, release, and metric records remain useful projections, but they are
// never the audit source of truth.
type ActivityEvent struct {
	ID            string            `json:"id"`
	EventType     string            `json:"event_type"`
	OccurredAt    time.Time         `json:"occurred_at"`
	IdentityID    string            `json:"identity_id"`
	ActionID      string            `json:"action_id,omitempty"`
	ReleaseID     string            `json:"release_id,omitempty"`
	ExecutionID   string            `json:"execution_id,omitempty"`
	ActorType     string            `json:"actor_type"`
	ExecutorType  string            `json:"executor_type,omitempty"`
	CorrelationID string            `json:"correlation_id,omitempty"`
	ArtifactRefs  []string          `json:"artifact_refs,omitempty"`
	Details       map[string]string `json:"details,omitempty"`
}

func (s *Service) appendActivity(event ActivityEvent) {
	event.ID = "evt-" + formatSequence(len(s.ActivityEvents)+1)
	event.OccurredAt = event.OccurredAt.UTC()
	event.ArtifactRefs = redactArtifactRefs(event.ArtifactRefs)
	event.Details = redactActivityDetails(event.Details)
	s.ActivityEvents = append(s.ActivityEvents, event)
}

// Timeline returns a defensive chronological copy. Callers cannot mutate the
// persisted ledger through a response object.
func (s *Service) Timeline(identityID, actionID, eventType string) []ActivityEvent {
	events := make([]ActivityEvent, 0, len(s.ActivityEvents))
	for _, event := range s.ActivityEvents {
		if identityID != "" && event.IdentityID != identityID {
			continue
		}
		if actionID != "" && event.ActionID != actionID {
			continue
		}
		if eventType != "" && event.EventType != eventType {
			continue
		}
		copy := event
		copy.ArtifactRefs = redactArtifactRefs(event.ArtifactRefs)
		copy.Details = redactActivityDetails(event.Details)
		events = append(events, copy)
	}
	sort.SliceStable(events, func(i, j int) bool { return events[i].OccurredAt.Before(events[j].OccurredAt) })
	return events
}

func redactArtifactRefs(refs []string) []string {
	clean := make([]string, 0, len(refs))
	for _, ref := range refs {
		lower := strings.ToLower(ref)
		if strings.Contains(lower, "cookie") || strings.Contains(lower, "password") || strings.Contains(lower, "token") || strings.Contains(lower, "credential") {
			continue
		}
		clean = append(clean, ref)
	}
	return clean
}

func redactActivityDetails(details map[string]string) map[string]string {
	if len(details) == 0 {
		return nil
	}
	clean := make(map[string]string, len(details))
	for key, value := range details {
		lower := strings.ToLower(key + " " + value)
		if strings.Contains(lower, "cookie") || strings.Contains(lower, "credential") || strings.Contains(lower, "password") || strings.Contains(lower, "token") || strings.Contains(lower, "proxy_password") {
			continue
		}
		clean[key] = value
	}
	return clean
}

func formatSequence(n int) string {
	// A decimal sequence is sufficient because this single-operator service
	// serializes mutation at the handler boundary; it is not a security token.
	return strconv.Itoa(n)
}
