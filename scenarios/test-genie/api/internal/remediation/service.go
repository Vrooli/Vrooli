package remediation

import (
	"context"
	"fmt"
	"time"
)

type Clock interface{ Now() time.Time }
type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

// Service owns job lifecycle transitions. Agent policy and execution are
// delegated through Launcher; verification execution is deliberately injected.
type Service struct {
	repo  Repository
	clock Clock
}

func NewService(repo Repository, clock Clock) *Service {
	if clock == nil {
		clock = systemClock{}
	}
	return &Service{repo: repo, clock: clock}
}

func (s *Service) Create(ctx context.Context, plan Plan, selected, requirements []string, additionalContext string) (Job, error) {
	selected, err := ValidateSelection(plan, selected)
	if err != nil {
		return Job{}, err
	}
	requirements, err = ValidateRequirementSelection(plan, requirements)
	if err != nil {
		return Job{}, err
	}
	if len(selected) == 0 && len(requirements) == 0 {
		return Job{}, fmt.Errorf("%w: select at least one finding or requirement", ErrInvalidSelector)
	}
	if active, err := s.repo.ActiveForScenario(ctx, plan.Scenario); err == nil && active.ID != "" {
		return Job{}, ErrActiveJob
	} else if err != nil && err != ErrNotFound {
		return Job{}, err
	}
	job := NewJob(plan, selected, requirements, additionalContext, s.clock.Now())
	if err := s.repo.Create(ctx, job); err != nil {
		return Job{}, err
	}
	return job, nil
}

func (s *Service) Get(ctx context.Context, id string) (Job, error) { return s.repo.Get(ctx, id) }
func (s *Service) List(ctx context.Context, scenario string, limit int) ([]Job, error) {
	return s.repo.ListByScenario(ctx, scenario, limit)
}
func (s *Service) Active(ctx context.Context, scenario string) (Job, error) {
	return s.repo.ActiveForScenario(ctx, scenario)
}

func (s *Service) MarkRunning(ctx context.Context, id string, attribution Attribution) (Job, error) {
	return s.transition(ctx, id, JobStatusCreated, JobStatusRunning, func(j *Job) { j.Attribution = attribution })
}
func (s *Service) MarkAgentCompleted(ctx context.Context, id, outputRef string) (Job, error) {
	return s.transition(ctx, id, JobStatusRunning, JobStatusAgentCompleted, func(j *Job) { j.Attribution.OutputReference = outputRef })
}
func (s *Service) StartVerification(ctx context.Context, id string, verification Verification) (Job, error) {
	return s.transition(ctx, id, JobStatusAgentCompleted, JobStatusVerificationRunning, func(j *Job) { j.Verification = verification })
}

// ReserveVerification atomically claims the verification slot before a
// server-owned run is started. This prevents duplicate UI requests from
// launching competing reruns for the same remediation job.
func (s *Service) ReserveVerification(ctx context.Context, id string) (Job, error) {
	return s.StartVerification(ctx, id, Verification{})
}

// SetVerificationRun attaches the durable run identity after admission. It is
// deliberately limited to a reserved verification lifecycle state.
func (s *Service) SetVerificationRun(ctx context.Context, id string, verification Verification) (Job, error) {
	job, err := s.repo.Get(ctx, id)
	if err != nil {
		return Job{}, err
	}
	if job.Status != JobStatusVerificationRunning {
		return Job{}, ErrInvalidState
	}
	job.Verification = verification
	job.UpdatedAt = s.clock.Now()
	if err := s.repo.Update(ctx, job); err != nil {
		return Job{}, err
	}
	return job, nil
}

// ReleaseVerificationReservation returns a job to the provisional agent state
// when the server cannot admit a rerun. Its immutable source and attribution
// remain available for a later explicit retry.
func (s *Service) ReleaseVerificationReservation(ctx context.Context, id string) (Job, error) {
	return s.transition(ctx, id, JobStatusVerificationRunning, JobStatusAgentCompleted, func(j *Job) { j.Verification = Verification{} })
}
func (s *Service) CompleteVerification(ctx context.Context, id string, verification Verification, delta FindingDelta, degraded string) (Job, error) {
	return s.transition(ctx, id, JobStatusVerificationRunning, JobStatusVerified, func(j *Job) {
		if verification.ExecutionID != "" {
			j.Verification.ExecutionID = verification.ExecutionID
		}
		if verification.RunID != "" {
			j.Verification.RunID = verification.RunID
		}
		j.Verification.CompletedAt = s.clock.Now()
		j.Verification.Delta = delta
		j.Verification.Degraded = degraded
		if degraded != "" {
			j.Status = JobStatusDegraded
		}
	})
}
func (s *Service) Cancel(ctx context.Context, id string) (Job, error) {
	job, err := s.repo.Get(ctx, id)
	if err != nil {
		return Job{}, err
	}
	if !IsActiveStatus(job.Status) {
		return Job{}, ErrInvalidState
	}
	job.Status = JobStatusCancelled
	job.CancelledAt = s.clock.Now()
	job.UpdatedAt = job.CancelledAt
	if err := s.repo.Update(ctx, job); err != nil {
		return Job{}, err
	}
	return job, nil
}
func (s *Service) Fail(ctx context.Context, id, failure string) (Job, error) {
	job, err := s.repo.Get(ctx, id)
	if err != nil {
		return Job{}, err
	}
	if !IsActiveStatus(job.Status) {
		return Job{}, ErrInvalidState
	}
	job.Status = JobStatusFailed
	job.Failure = failure
	job.UpdatedAt = s.clock.Now()
	if err := s.repo.Update(ctx, job); err != nil {
		return Job{}, err
	}
	return job, nil
}
func (s *Service) transition(ctx context.Context, id, from, to string, mutate func(*Job)) (Job, error) {
	job, err := s.repo.Get(ctx, id)
	if err != nil {
		return Job{}, err
	}
	if job.Status != from {
		return Job{}, fmt.Errorf("%w: %s -> %s", ErrInvalidState, job.Status, to)
	}
	job.Status = to
	mutate(&job)
	job.UpdatedAt = s.clock.Now()
	if err := s.repo.Update(ctx, job); err != nil {
		return Job{}, err
	}
	return job, nil
}
