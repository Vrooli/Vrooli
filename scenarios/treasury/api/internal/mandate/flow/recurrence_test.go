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
	}
}
