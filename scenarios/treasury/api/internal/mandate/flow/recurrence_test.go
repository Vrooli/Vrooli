package flow

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRecurrenceTransitionMatrix(t *testing.T) {
	tests := []struct {
		status RecurrenceStatus
		event  RecurrenceEvent
		want   RecurrenceStatus
		fails  bool
	}{
		{RecurrenceActive, RecurrenceBoundary, RecurrenceActive, false},
		{RecurrenceActive, RecurrenceCancel, RecurrenceCancelled, false},
		{RecurrenceCancelled, RecurrenceBoundary, RecurrenceCancelled, true},
		{RecurrenceCancelled, RecurrenceCancel, RecurrenceCancelled, true},
	}
	for _, test := range tests {
		got, err := TransitionRecurrence(test.status, test.event)
		require.Equal(t, test.want, got)
		require.Equal(t, test.fails, err != nil)
		require.NoError(t, CheckRecurrenceInvariants(got))
	}
}

func TestRecurrenceNamedTraces(t *testing.T) {
	traces := []struct {
		name   string
		events []RecurrenceEvent
		want   RecurrenceStatus
	}{
		{name: "two due periods remain active", events: []RecurrenceEvent{RecurrenceBoundary, RecurrenceBoundary}, want: RecurrenceActive},
		{name: "operator cancellation is terminal", events: []RecurrenceEvent{RecurrenceBoundary, RecurrenceCancel}, want: RecurrenceCancelled},
	}
	for _, trace := range traces {
		t.Run(trace.name, func(t *testing.T) {
			status := RecurrenceActive
			for _, event := range trace.events {
				var err error
				status, err = TransitionRecurrence(status, event)
				require.NoError(t, err)
			}
			require.Equal(t, trace.want, status)
		})
	}
}
