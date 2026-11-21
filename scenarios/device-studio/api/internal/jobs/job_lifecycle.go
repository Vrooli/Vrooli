package jobs

import "fmt"

type JobEvent string

const (
	JobStarted         JobEvent = "started"
	JobCheckpointed    JobEvent = "checkpointed"
	JobSucceeded       JobEvent = "succeeded"
	JobFailed          JobEvent = "failed"
	JobCancelRequested JobEvent = "cancel_requested"
	JobCancelled       JobEvent = "cancelled"
	JobRetried         JobEvent = "retried"
)

var jobStateTransitions = map[State]map[JobEvent]State{
	Queued: {
		JobStarted:         Running,
		JobFailed:          Failed,
		JobCancelRequested: Cancelled,
	},
	Running: {
		JobCheckpointed:    Checkpointed,
		JobSucceeded:       Succeeded,
		JobFailed:          Failed,
		JobCancelRequested: CancelRequested,
	},
	Checkpointed: {
		JobStarted:         Running,
		JobCancelRequested: Cancelled,
	},
	CancelRequested: {
		JobCancelled: Cancelled,
	},
	Failed: {
		JobRetried: Queued,
	},
	Cancelled: {
		JobRetried: Queued,
	},
}

func (state State) Valid() bool {
	switch state {
	case Queued, Running, Checkpointed, CancelRequested, Succeeded, Failed, Cancelled:
		return true
	default:
		return false
	}
}

// Transition applies the sole legal-edge policy for a durable render job.
// Retry is explicit because it reopens terminal records only after the
// repository has verified settlement, worker release, and quiescence evidence.
func (job *Job) Transition(event JobEvent) error {
	if job == nil || !job.State.Valid() {
		return fmt.Errorf("%w: job has unknown state", ErrInvalidTransition)
	}
	next, ok := jobStateTransitions[job.State][event]
	if !ok {
		return fmt.Errorf("%w: event %q is invalid from %s", ErrInvalidTransition, event, job.State)
	}
	job.State = next
	return nil
}
