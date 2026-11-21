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
	"time"

	"agent-manager/internal/goalhome"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Decision states. pending waits for the operator; answered and defaulted
// carry an answer; expired ended without one (no default). recorded_at marks
// that `effort decision sync` wrote the answer into FEEDBACK.md.
const (
	Pending   = "pending"
	Answered  = "answered"
	Defaulted = "defaulted"
	Expired   = "expired"
	Dismissed = "dismissed"

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
	home = strings.TrimSuffix(home, "/"+goalhome.CharterFile)
	cleaned := path.Clean(home)
	if home == "" || !filepath.IsLocal(cleaned) {
		return "", fmt.Errorf("goal home %q must be a repository-relative directory", home)
	}
	return cleaned, nil
}

// NormalizeDecisionHome cleans the key a decision is stored under. It is a
// repository-relative goal home, or the absolute directory of a home kept outside
// the repository (a research cohort under ~/.vrooli): a decision never reads the
// home, it only names it, so its key may live anywhere. An absolute path with a
// "." or ".." element is refused.
func NormalizeDecisionHome(home string) (string, error) {
	trimmed := filepath.ToSlash(strings.TrimSpace(home))
	trimmed = strings.TrimSuffix(strings.TrimPrefix(trimmed, "path:"), "/"+goalhome.GoalFile)
	trimmed = strings.TrimSuffix(trimmed, "/"+goalhome.CharterFile)
	if !path.IsAbs(trimmed) {
		return NormalizeGoalHome(home)
	}
	if cleaned := path.Clean(trimmed); cleaned != "/" && strings.TrimSuffix(trimmed, "/") == cleaned {
		return cleaned, nil
	}
	return "", fmt.Errorf("goal home %q must be an absolute directory without . or .. elements", trimmed)
}

// View converts the record to the goal-home view the renderers use.
func View(d *pb.EffortDecision) goalhome.Decision {
	view := goalhome.Decision{
		ID: d.GetDecisionId(), Question: d.GetQuestion(), Recommended: d.GetRecommended(),
		RecommendationReason: d.GetRecommendationReason(), DefaultAnswer: d.GetDefaultAnswer(), Reversible: d.GetReversible(),
		Urgency: d.GetUrgency(), Note: d.GetNote(), BigPicture: d.GetBigPicture(), TechnicalDetails: d.GetTechnicalDetails(),
		ContextURL: d.GetContextUrl(), Explainer: d.GetExplainerHtml(), State: d.GetState(),
		Answer: d.GetAnswer(), AnswerLabel: d.GetAnswerLabel(), AnswerNote: d.GetAnswerNote(), AnsweredVia: d.GetAnsweredVia(),
		ResolutionReason: d.GetResolutionReason(),
		ResolvedAt:       timestampTime(d.GetResolvedAt()),
		Late:             d.GetLate(), AskID: d.GetAskId(),
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

func timestampTime(value *timestamppb.Timestamp) time.Time {
	if value == nil {
		return time.Time{}
	}
	return value.AsTime()
}

// SubjectKey is the Hub subject key of a decision: one open Hub decision per
// (goal home, decision id), so a revised decision updates its ask in place and a
// withdrawal or an operator question finds it by the same name.
func SubjectKey(goalHome, decisionID string) string {
	return operatordecision.SubjectKeyFrom("effort-decision:" + goalHome + ":" + decisionID)
}

// RequestedEvent rebuilds the decision_requested event from the stored
// record alone, so a retry publishes byte-identical content and the same id.
// The page is the one the raiser sent, or one generated from the record.
//
// The decision declares it answers the operator's "I need more context" question
// only when a verified run raised it: that run is who Agent Manager hands the
// question to, and the answer is a revised raise under the same decision id.
func RequestedEvent(d *pb.EffortDecision) eventbus.DomainEvent {
	subject := HomeName(d.GetGoalHome())
	correlation := map[string]string{"goal_home": d.GetGoalHome(), "decision_id": d.GetDecisionId()}
	if d.GetEffortRef() != "" {
		correlation["effort_ref"] = d.GetEffortRef()
	}
	view := View(d)
	facts := view.Facts(subject, correlation)
	if facts.Explainer == "" {
		facts.Explainer = Page(view, facts, d.GetGoalHome())
	}
	facts.SubjectKey = SubjectKey(d.GetGoalHome(), d.GetDecisionId())
	facts.AcceptsContextRequests = d.GetRaisedByRunId() != ""
	// A revision is a new assertion, so its event is stamped with the revision
	// time: it differs from every earlier event even when the text returns to an
	// earlier revision, and the Hub never mistakes it for a redelivery.
	occurred := d.GetCreatedAt()
	if d.GetRevisedAt() != nil {
		occurred = d.GetRevisedAt()
	}
	return eventbus.DomainEvent{
		Source:    Source,
		EventType: goalhome.DecisionRequestedEventType,
		Occurred:  occurred.AsTime().UTC(),
		Payload:   facts.Payload(),
	}
}

// EventID is the id eventbus.Client assigns to the event: the SHA-256 of its
// JSON encoding. It delegates to the shared operatordecision substrate, which
// computes the identical hash, so the stored EventId is unchanged.
func EventID(event eventbus.DomainEvent) string {
	return operatordecision.EventID(event)
}
