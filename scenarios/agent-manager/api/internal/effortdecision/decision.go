// Package effortdecision holds the parts of an operator decision (DL-5) that
// both the Agent Manager service and its CLI need: the record's states, the
// goal-home key, the goal-home view, and the decision_requested event. It
// imports no storage, so the CLI can use it.
package effortdecision

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"agent-manager/internal/goalhome"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// Decision states. pending waits for the operator; answered and defaulted
// carry an answer; expired ended without one (no default). recorded_at marks
// that `effort decision sync` wrote the answer into FEEDBACK.md.
const (
	Pending   = "pending"
	Answered  = "answered"
	Defaulted = "defaulted"
	Expired   = "expired"

	ViaPhone   = "phone"
	ViaDesktop = "desktop"
	ViaDefault = "default"

	// Source is the Vrooli Events source of decision_requested events.
	Source = "agent-manager"
)

// NormalizeGoalHome cleans a repository-relative goal-home path. Absolute
// paths and paths that leave the repository are refused.
func NormalizeGoalHome(home string) (string, error) {
	home = filepath.ToSlash(strings.TrimSpace(home))
	home = strings.TrimSuffix(strings.TrimPrefix(home, "path:"), "/"+goalhome.GoalFile)
	cleaned := path.Clean(home)
	if home == "" || !filepath.IsLocal(cleaned) {
		return "", fmt.Errorf("goal home %q must be a repository-relative directory", home)
	}
	return cleaned, nil
}

// View converts the record to the goal-home view the renderers use.
func View(d *pb.EffortDecision) goalhome.Decision {
	view := goalhome.Decision{
		ID: d.GetDecisionId(), Question: d.GetQuestion(), Recommended: d.GetRecommended(),
		RecommendationReason: d.GetRecommendationReason(), DefaultAnswer: d.GetDefaultAnswer(), Reversible: d.GetReversible(),
		Urgency: d.GetUrgency(), Note: d.GetNote(), ContextURL: d.GetContextUrl(), State: d.GetState(),
		Answer: d.GetAnswer(), AnswerLabel: d.GetAnswerLabel(), AnswerNote: d.GetAnswerNote(), AnsweredVia: d.GetAnsweredVia(),
		Late: d.GetLate(), AskID: d.GetAskId(),
	}
	for _, option := range d.GetOptions() {
		view.Options = append(view.Options, goalhome.DecisionOption{Key: option.GetKey(), Label: option.GetLabel()})
	}
	if d.GetDeadline() != nil {
		view.Deadline = d.GetDeadline().AsTime()
	}
	if d.GetCreatedAt() != nil {
		view.RaisedAt = d.GetCreatedAt().AsTime()
	}
	if d.GetAnsweredAt() != nil {
		view.AnsweredAt = d.GetAnsweredAt().AsTime()
	}
	return view
}

// RequestedEvent rebuilds the decision_requested event from the stored
// record alone, so a retry publishes byte-identical content and the same id.
func RequestedEvent(d *pb.EffortDecision) eventbus.DomainEvent {
	subject := d.GetGoalHome()
	if parts := strings.Split(subject, "/"); len(parts) > 1 && parts[0] == "scenarios" {
		subject = parts[1]
	}
	correlation := map[string]string{"goal_home": d.GetGoalHome(), "decision_id": d.GetDecisionId()}
	if d.GetEffortRef() != "" {
		correlation["effort_ref"] = d.GetEffortRef()
	}
	return eventbus.DomainEvent{
		Source:    Source,
		EventType: goalhome.DecisionRequestedEventType,
		Occurred:  d.GetCreatedAt().AsTime().UTC(),
		Payload:   View(d).RequestPayload(subject, correlation),
	}
}

// EventID is the id eventbus.Client assigns to the event: the SHA-256 of its
// JSON encoding. It delegates to the shared operatordecision substrate, which
// computes the identical hash, so the stored EventId is unchanged.
func EventID(event eventbus.DomainEvent) string {
	return operatordecision.EventID(event)
}
