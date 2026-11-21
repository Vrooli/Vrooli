package review

import "fmt"

type RequestThreadStatus string

const (
	RequestThreadStatusPending   RequestThreadStatus = "pending"
	RequestThreadStatusFulfilled RequestThreadStatus = "fulfilled"
	RequestThreadStatusDismissed RequestThreadStatus = "dismissed"
)

var requestThreadTransitions = map[RequestThreadStatus]map[RequestThreadStatus]struct{}{
	RequestThreadStatusPending:   requestThreadEdges(RequestThreadStatusFulfilled, RequestThreadStatusDismissed),
	RequestThreadStatusFulfilled: requestThreadEdges(),
	RequestThreadStatusDismissed: requestThreadEdges(),
}

func requestThreadEdges(next ...RequestThreadStatus) map[RequestThreadStatus]struct{} {
	edges := make(map[RequestThreadStatus]struct{}, len(next))
	for _, status := range next {
		edges[status] = struct{}{}
	}
	return edges
}

// InvalidRequestThreadTransitionError reports an illegal durable child transition.
type InvalidRequestThreadTransitionError struct {
	From RequestThreadStatus
	To   RequestThreadStatus
}

func (e *InvalidRequestThreadTransitionError) Error() string {
	return fmt.Sprintf("illegal review request thread status transition %q -> %q", e.From, e.To)
}

func (t *RequestThread) ValidateStatus() error {
	if t == nil || !isKnownRequestThreadStatus(t.Status) {
		var status RequestThreadStatus
		if t != nil {
			status = t.Status
		}
		return &InvalidRequestThreadTransitionError{From: status}
	}
	return nil
}

func (t *RequestThread) IsPending() bool {
	return t != nil && t.Status == RequestThreadStatusPending
}

func (t *RequestThread) IsFulfilled() bool {
	return t != nil && t.Status == RequestThreadStatusFulfilled
}

// TransitionStatus is the single mutation owner for a persisted request thread.
func (t *RequestThread) TransitionStatus(next RequestThreadStatus) error {
	if t == nil {
		return &InvalidRequestThreadTransitionError{To: next}
	}
	from := t.Status
	if !isKnownRequestThreadStatus(from) || !isKnownRequestThreadStatus(next) {
		return &InvalidRequestThreadTransitionError{From: from, To: next}
	}
	if from == next {
		return nil
	}
	if _, ok := requestThreadTransitions[from][next]; !ok {
		return &InvalidRequestThreadTransitionError{From: from, To: next}
	}
	t.Status = next
	return nil
}

func isKnownRequestThreadStatus(status RequestThreadStatus) bool {
	_, ok := requestThreadTransitions[status]
	return ok
}

func validateRequestThreadSave(current, proposed []RequestThread) error {
	currentByID := make(map[string]RequestThread, len(current))
	for _, thread := range current {
		currentByID[thread.ID] = thread
	}
	seen := make(map[string]struct{}, len(proposed))
	for _, next := range proposed {
		if _, duplicate := seen[next.ID]; duplicate {
			return fmt.Errorf("duplicate review request thread %q", next.ID)
		}
		seen[next.ID] = struct{}{}

		previous, exists := currentByID[next.ID]
		if !exists {
			if next.Status != RequestThreadStatusPending {
				return &InvalidRequestThreadTransitionError{To: next.Status}
			}
			continue
		}
		if err := previous.TransitionStatus(next.Status); err != nil {
			return err
		}
	}
	for id := range currentByID {
		if _, exists := seen[id]; !exists {
			return fmt.Errorf("cannot remove persisted review request thread %q", id)
		}
	}
	return nil
}
