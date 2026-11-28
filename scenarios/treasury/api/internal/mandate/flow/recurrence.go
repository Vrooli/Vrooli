package flow

import "fmt"

type (
	RecurrenceStatus string
	RecurrenceEvent  string
)

const (
	RecurrenceActive    RecurrenceStatus = "active"
	RecurrenceCancelled RecurrenceStatus = "cancelled"
	RecurrenceBoundary  RecurrenceEvent  = "boundary"
	RecurrenceCancel    RecurrenceEvent  = "cancel"
)

func CheckRecurrenceInvariants(status RecurrenceStatus) error {
	if status != RecurrenceActive && status != RecurrenceCancelled {
		return fmt.Errorf("unknown recurrence status %q", status)
	}
	return nil
}

// TransitionRecurrence is the level-3 explicit state/event model for standing
// obligations. A boundary keeps an active obligation active; cancellation is
// terminal, so no later boundary can raise another occurrence.
func TransitionRecurrence(status RecurrenceStatus, event RecurrenceEvent) (RecurrenceStatus, error) {
	if err := CheckRecurrenceInvariants(status); err != nil {
		return status, err
	}
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
