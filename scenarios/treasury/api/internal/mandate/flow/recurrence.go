package flow

import "fmt"

type RecurrenceStatus string
type RecurrenceEvent string

const (
	RecurrenceActive    RecurrenceStatus = "active"
	RecurrenceCancelled RecurrenceStatus = "cancelled"
	RecurrenceBoundary  RecurrenceEvent  = "boundary"
	RecurrenceCancel    RecurrenceEvent  = "cancel"
)

// TransitionRecurrence is the level-3 explicit state/event model for standing
// obligations. A boundary keeps an active obligation active; cancellation is
// terminal, so no later boundary can raise another occurrence.
func TransitionRecurrence(status RecurrenceStatus, event RecurrenceEvent) (RecurrenceStatus, error) {
	switch {
	case status == RecurrenceActive && event == RecurrenceBoundary:
		return RecurrenceActive, nil
	case status == RecurrenceActive && event == RecurrenceCancel:
		return RecurrenceCancelled, nil
	case status == RecurrenceCancelled:
		return status, fmt.Errorf("cancelled recurrence is terminal")
	default:
		return status, fmt.Errorf("invalid recurrence transition %s + %s", status, event)
	}
}
