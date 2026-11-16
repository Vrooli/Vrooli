package operatordecision

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// AskContextRequestedEventType is notification-hub's context-request event type.
// The hub publishes it when the operator asks the producer a question about an
// open decision ("I need more context"). Only a decision whose producer declared
// Facts.AcceptsContextRequests offers that button, so the event always reaches a
// producer that said it would answer. A producer answers by re-sending the
// decision under the same SubjectKey with the answer in its text; the hub updates
// the open decision in place and marks the request addressed.
const AskContextRequestedEventType = "notification_hub.ask.context_requested.v1"

// ContextRequest is notification-hub's ask.context_requested payload: facts
// only. Match it to the producer's own record by Correlation (returned verbatim)
// or by Source, SourceEventType and SubjectKey.
type ContextRequest struct {
	AskID string `json:"ask_id"`
	// RequestID identifies this question; a redelivery carries the same one.
	RequestID string `json:"request_id"`
	// Question is the operator's question, in their words.
	Question    string `json:"question"`
	RequestedAt string `json:"requested_at"`
	// Source and SourceEventType name the producer event that raised the decision.
	Source          string `json:"source"`
	SourceEventType string `json:"source_event_type"`
	// RequesterRef is the event id of the revision the operator was looking at.
	RequesterRef string `json:"requester_ref"`
	// SubjectKey is the key to re-send the decision under to answer. It is always
	// set: a decision without one cannot declare AcceptsContextRequests.
	SubjectKey  string            `json:"subject_key"`
	Correlation map[string]string `json:"correlation"`
}

// Payload is the event payload the hub publishes, in the shape structpb accepts.
func (c ContextRequest) Payload() map[string]any {
	correlation := make(map[string]any, len(c.Correlation))
	for key, value := range c.Correlation {
		correlation[key] = value
	}
	return map[string]any{
		"ask_id":            c.AskID,
		"request_id":        c.RequestID,
		"question":          c.Question,
		"requested_at":      c.RequestedAt,
		"source":            c.Source,
		"source_event_type": c.SourceEventType,
		"requester_ref":     c.RequesterRef,
		"subject_key":       c.SubjectKey,
		"correlation":       correlation,
	}
}

// ErrContextRequestMalformed is returned for a context-request payload that names
// no ask or carries no question.
var ErrContextRequestMalformed = errors.New("context request payload needs an ask_id and a question")

// DecodeContextRequest reads a context-request event payload.
func DecodeContextRequest(payload []byte) (ContextRequest, error) {
	var request ContextRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		return ContextRequest{}, err
	}
	if strings.TrimSpace(request.AskID) == "" || strings.TrimSpace(request.Question) == "" {
		return ContextRequest{}, ErrContextRequestMalformed
	}
	return request, nil
}

// ContextResponder is the optional half of a DomainEffect for a producer that
// answers context requests. An effect that implements it receives the operator's
// question for the producer events it Handles; one that does not is never asked,
// and its decisions do not offer the button (see Facts.AcceptsContextRequests).
//
// The webhook delivers a context request only when the scenario subscribes to
// AskContextRequestedEventType in addition to AskResolvedEventType.
type ContextResponder interface {
	// RespondToContext answers the operator, normally by re-sending the decision
	// under the same SubjectKey with the answer in its text. handled=true stops
	// fan-out. Redelivery is possible: a request already answered is a benign
	// (handled=true, nil), never an error that triggers a retry storm.
	RespondToContext(ctx context.Context, request ContextRequest) (handled bool, err error)
}
