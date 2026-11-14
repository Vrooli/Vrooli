package jobs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJobTransitionEventMatrix(t *testing.T) {
	cases := []struct {
		from  State
		event JobEvent
		to    State
	}{
		{Queued, JobStarted, Running},
		{Queued, JobFailed, Failed},
		{Queued, JobCancelRequested, Cancelled},
		{Running, JobCheckpointed, Checkpointed},
		{Running, JobSucceeded, Succeeded},
		{Running, JobFailed, Failed},
		{Running, JobCancelRequested, CancelRequested},
		{Checkpointed, JobStarted, Running},
		{Checkpointed, JobCancelRequested, Cancelled},
		{CancelRequested, JobCancelled, Cancelled},
		{Failed, JobRetried, Queued},
		{Cancelled, JobRetried, Queued},
	}
	for _, tc := range cases {
		t.Run(string(tc.from)+"_"+string(tc.event), func(t *testing.T) {
			job := Job{State: tc.from}
			require.NoError(t, job.Transition(tc.event))
			require.Equal(t, tc.to, job.State)
		})
	}
}

func TestJobTransitionRejectsIllegalAndUnknownStateWithoutMutation(t *testing.T) {
	for _, tc := range []struct {
		state State
		event JobEvent
	}{
		{Queued, JobSucceeded},
		{Succeeded, JobStarted},
		{CancelRequested, JobRetried},
		{State("unrecognized"), JobStarted},
	} {
		job := Job{State: tc.state}
		require.ErrorIs(t, job.Transition(tc.event), ErrInvalidTransition)
		require.Equal(t, tc.state, job.State)
	}
	var nilJob *Job
	require.ErrorIs(t, nilJob.Transition(JobStarted), ErrInvalidTransition)
}
