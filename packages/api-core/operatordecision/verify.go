package operatordecision

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/vrooli/api-core/askreader"
)

// The reasons a claim is refused. They are the keys of WebhookStats.Refused.
const (
	// ReasonNoSuchAsk: the Hub holds no such ask for this scenario.
	ReasonNoSuchAsk = "no_such_ask"
	// ReasonOtherSource: the Hub says the ask belongs to another scenario.
	ReasonOtherSource = "other_source"
	// ReasonAskOpen: the claim says the ask is closed and the Hub still has it open.
	ReasonAskOpen = "ask_open"
	// ReasonMismatch: the body differs from the Hub's record.
	ReasonMismatch = "mismatch"
	// ReasonNoSuchRequest: the Hub holds no such context request on the ask.
	ReasonNoSuchRequest = "no_such_request"
	// ReasonUnconfirmed: the claim failed for a reason that has no name of its own.
	ReasonUnconfirmed = "unconfirmed"
)

// RefusedError is a claim Notification Hub does not confirm. Reason is the stable
// short name the webhook counts it under; Detail says what differed.
type RefusedError struct {
	Reason string
	Detail string
}

func (e *RefusedError) Error() string { return e.Detail }

// closedAskStates are the states an ask never leaves. A resolution is a claim
// that the ask reached one.
var closedAskStates = map[string]bool{StateAnswered: true, StateDefaulted: true, StateExpired: true, StateDismissed: true, StateWithdrawn: true}

// VerifiedResolution is the resolution Notification Hub itself stands behind: the
// claimed resolution checked against the Hub's record of the ask, with every field
// taken from the Hub. It is refused unless the ask is closed and the Hub says
// exactly what the claim does about the state, the answer, who answered, the
// producer event, its requester reference, the subject key and the correlation
// that locates the scenario's own record. The checked fields are the ones an
// effect decides or locates by, so a body that differs in any of them is forged or
// stale and applies nothing.
func VerifiedResolution(claimed DecisionResolution, hub askreader.Ask) (DecisionResolution, error) {
	if !closedAskStates[hub.State] {
		return DecisionResolution{}, &RefusedError{Reason: ReasonAskOpen, Detail: fmt.Sprintf("the Hub still has ask %q in state %q", hub.ID, hub.State)}
	}
	var differs []string
	check := func(name, claim, held string) {
		if strings.TrimSpace(claim) != held {
			differs = append(differs, name)
		}
	}
	check("state", claimed.State, hub.State)
	check("answer", claimed.Answer, hub.Answer)
	check("answered_by", claimed.AnsweredBy, hub.AnsweredBy)
	check("source_event_type", claimed.SourceEventType, hub.SourceEventType)
	check("requester_ref", claimed.RequesterRef, hub.RequesterRef)
	check("subject_key", claimed.SubjectKey, hub.SubjectKey)
	check("superseded_by_ask_id", claimed.SupersededByAskID, hub.SupersededByAskID)
	if !sameMap(claimed.Correlation, hub.Correlation) {
		differs = append(differs, "correlation")
	}
	if len(differs) > 0 {
		return DecisionResolution{}, &RefusedError{Reason: ReasonMismatch, Detail: fmt.Sprintf("ask %q: the event body differs from the Hub's record in %s", hub.ID, strings.Join(differs, ", "))}
	}
	return DecisionResolution{
		AskID: hub.ID, State: hub.State, Answer: hub.Answer, AnswerLabel: hub.AnswerLabel, Note: hub.Note,
		AnsweredBy: hub.AnsweredBy, AnsweredByIdentityKind: hub.AnsweredByIdentityKind, AnsweredByIdentitySource: hub.AnsweredByIdentitySource,
		AnsweredAt: hub.AnsweredAt, ResolvedAt: hub.ResolvedAt, ByDefault: hub.State == StateDefaulted, Late: hub.Late, Reason: hub.Reason,
		Source: hub.Source, SourceEventType: hub.SourceEventType, RequesterRef: hub.RequesterRef,
		Correlation: maps.Clone(hub.Correlation), SubjectKey: hub.SubjectKey, SupersededByAskID: hub.SupersededByAskID,
		// The form answers a consumer applies come from the Hub's record too.
		Responses: slices.Clone(hub.Responses), AnswerSummary: hub.AnswerSummary,
	}, nil
}

// VerifiedContextRequest is the context request the Hub itself stands behind: the
// ask must hold that exact question under that request id, for the same producer
// event and subject key. The correlation, requester reference, event type,
// subject key and question the producer acts on come from the Hub, so a forged
// correlation cannot redirect the answer. The correlation and requester reference
// are not compared: an open ask the producer revised since the event was published
// legitimately carries newer ones, and the Hub's are the ones to act on.
func VerifiedContextRequest(claimed ContextRequest, hub askreader.Ask) (ContextRequest, error) {
	var held *askreader.ContextRequest
	for i := range hub.ContextRequests {
		if hub.ContextRequests[i].ID == strings.TrimSpace(claimed.RequestID) {
			held = &hub.ContextRequests[i]
		}
	}
	if held == nil {
		return ContextRequest{}, &RefusedError{Reason: ReasonNoSuchRequest, Detail: fmt.Sprintf("the Hub holds no context request %q on ask %q", claimed.RequestID, hub.ID)}
	}
	var differs []string
	check := func(name, claim, held string) {
		if strings.TrimSpace(claim) != held {
			differs = append(differs, name)
		}
	}
	check("question", claimed.Question, held.Question)
	check("source_event_type", claimed.SourceEventType, hub.SourceEventType)
	check("subject_key", claimed.SubjectKey, hub.SubjectKey)
	if len(differs) > 0 {
		return ContextRequest{}, &RefusedError{Reason: ReasonMismatch, Detail: fmt.Sprintf("ask %q: the context request differs from the Hub's record in %s", hub.ID, strings.Join(differs, ", "))}
	}
	return ContextRequest{
		AskID: hub.ID, RequestID: held.ID, Question: held.Question, RequestedAt: held.RequestedAt,
		Source: hub.Source, SourceEventType: hub.SourceEventType, RequesterRef: hub.RequesterRef,
		SubjectKey: hub.SubjectKey, Correlation: maps.Clone(hub.Correlation),
	}, nil
}

func sameMap(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	return maps.Equal(a, b)
}
