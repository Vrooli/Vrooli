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
	job, err := s.transition(ctx, id, JobStatusLaunchPending, JobStatusRunning, func(j *Job) { j.Attribution = attribution; j.Failure = "" })
	if err != nil {
		return Job{}, err
	}
	attempt := newAttempt("launch", "accepted", launchIdempotencyKey(job), attribution.RoleRef, "Agent Manager accepted the launch", job.UpdatedAt)
	attempt.TaskID, attempt.RunID = attribution.TaskID, attribution.RunID
	if err := s.repo.AppendAttempt(ctx, attempt, job.ID); err != nil {
		return Job{}, err
	}
	return s.repo.Get(ctx, job.ID)
}

// PrepareLaunch records a stable remote idempotency key before a caller can
// contact Agent Manager. A replay returns the pending job instead of creating
// a second launch intent; Agent Manager uses the same key to deduplicate the
// remote side of the crash window.
func (s *Service) PrepareLaunch(ctx context.Context, id, roleRef string) (Job, error) {
	job, err := s.repo.Get(ctx, id)
	if err != nil {
		return Job{}, err
	}
	if job.Status == JobStatusLaunchPending || job.Status == JobStatusRunning {
		return job, nil
	}
	if job.Status != JobStatusCreated {
		return Job{}, ErrInvalidState
	}
	job.LaunchAttempt++
	job.Status = JobStatusLaunchPending
	job.Attribution.RoleRef = roleRef
	job.UpdatedAt = s.clock.Now()
	if err := s.repo.UpdateIfStatus(ctx, job, JobStatusCreated); err != nil {
		return Job{}, err
	}
	attempt := newAttempt("launch", "prepared", launchIdempotencyKey(job), roleRef, "launch intent persisted before Agent Manager call", job.UpdatedAt)
	if err := s.repo.AppendAttempt(ctx, attempt, job.ID); err != nil {
		return Job{}, err
	}
	return s.repo.Get(ctx, job.ID)
}

// RetryLaunch creates a distinct remote attempt only after a terminal launch
// outcome. It never overwrites the old attempt or reuses its idempotency key.
func (s *Service) RetryLaunch(ctx context.Context, id string) (Job, error) {
	job, err := s.repo.Get(ctx, id)
	if err != nil {
		return Job{}, err
	}
	if job.Status != JobStatusFailed && job.Status != JobStatusCancelled {
		return Job{}, ErrInvalidState
	}
	if job.Attribution.RoleRef == "" {
		return Job{}, ErrInvalidState
	}
	previous := job.Status
	job.LaunchAttempt++
	job.Status, job.Failure, job.CancelledAt, job.UpdatedAt = JobStatusLaunchPending, "", time.Time{}, s.clock.Now()
	if err := s.repo.UpdateIfStatus(ctx, job, previous); err != nil {
		return Job{}, err
	}
	attempt := newAttempt("launch", "retry_prepared", launchIdempotencyKey(job), job.Attribution.RoleRef, "operator started a new retry attempt", job.UpdatedAt)
	if err := s.repo.AppendAttempt(ctx, attempt, job.ID); err != nil {
		return Job{}, err
	}
	return s.repo.Get(ctx, job.ID)
}

func (s *Service) RecordLaunchFailure(ctx context.Context, id, detail string) (Job, error) {
	job, err := s.repo.Get(ctx, id)
	if err != nil {
		return Job{}, err
	}
	if job.Status != JobStatusLaunchPending {
		return Job{}, ErrInvalidState
	}
	job.Failure = detail
	job.UpdatedAt = s.clock.Now()
	if err := s.repo.UpdateIfStatus(ctx, job, JobStatusLaunchPending); err != nil {
		return Job{}, err
	}
	attempt := newAttempt("launch", "failed", launchIdempotencyKey(job), job.Attribution.RoleRef, detail, job.UpdatedAt)
	if err := s.repo.AppendAttempt(ctx, attempt, job.ID); err != nil {
		return Job{}, err
	}
	return s.repo.Get(ctx, job.ID)
}
func (s *Service) MarkAgentCompleted(ctx context.Context, id, outputRef string) (Job, error) {
	return s.transition(ctx, id, JobStatusRunning, JobStatusAgentCompleted, func(j *Job) { j.Attribution.OutputReference = outputRef })
}
func (s *Service) StartVerification(ctx context.Context, id string, verification Verification) (Job, error) {
	job, err := s.transition(ctx, id, JobStatusAgentCompleted, JobStatusVerificationRunning, func(j *Job) { j.Verification = verification })
	if err != nil {
		return Job{}, err
	}
	attempt := newAttempt("verification", "reserved", "test-genie/remediation/"+job.ID+"/verification", "", "server-owned verification reserved", job.UpdatedAt)
	if err := s.repo.AppendAttempt(ctx, attempt, job.ID); err != nil {
		return Job{}, err
	}
	return s.repo.Get(ctx, job.ID)
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
	if err := s.repo.UpdateIfStatus(ctx, job, JobStatusVerificationRunning); err != nil {
		return Job{}, err
	}
	attempt := newAttempt("verification", "launched", "test-genie/remediation/"+job.ID+"/verification", "", "server-owned verification launched", job.UpdatedAt)
	attempt.RunID = verification.RunID
	if err := s.repo.AppendAttempt(ctx, attempt, job.ID); err != nil {
		return Job{}, err
	}
	return s.repo.Get(ctx, job.ID)
}

// ReleaseVerificationReservation returns a job to the provisional agent state
// when the server cannot admit a rerun. Its immutable source and attribution
// remain available for a later explicit retry.
func (s *Service) ReleaseVerificationReservation(ctx context.Context, id string) (Job, error) {
	return s.transition(ctx, id, JobStatusVerificationRunning, JobStatusAgentCompleted, func(j *Job) { j.Verification = Verification{} })
}
func (s *Service) CompleteVerification(ctx context.Context, id string, verification Verification, delta FindingDelta, requirementDelta RequirementDelta, degraded string) (Job, error) {
	job, err := s.transition(ctx, id, JobStatusVerificationRunning, JobStatusVerified, func(j *Job) {
		if verification.ExecutionID != "" {
			j.Verification.ExecutionID = verification.ExecutionID
		}
		if verification.RunID != "" {
			j.Verification.RunID = verification.RunID
		}
		j.Verification.CompletedAt = s.clock.Now()
		j.Verification.Delta = delta
		j.Verification.RequirementDelta = requirementDelta
		j.Verification.Degraded = degraded
		if degraded != "" {
			j.Status = JobStatusDegraded
		}
	})
	if err != nil {
		return Job{}, err
	}
	state, detail := "verified", "verification completed"
	if degraded != "" {
		state, detail = "degraded", degraded
	}
	attempt := newAttempt("verification", state, "test-genie/remediation/"+job.ID+"/verification", "", detail, job.UpdatedAt)
	attempt.RunID = job.Verification.RunID
	if err := s.repo.AppendAttempt(ctx, attempt, job.ID); err != nil {
		return Job{}, err
	}
	return s.repo.Get(ctx, job.ID)
}
func (s *Service) Cancel(ctx context.Context, id string) (Job, error) {
	job, err := s.repo.Get(ctx, id)
	if err != nil {
		return Job{}, err
	}
	if !IsActiveStatus(job.Status) {
		return Job{}, ErrInvalidState
	}
	previous := job.Status
	job.Status = JobStatusCancelled
	job.CancelledAt = s.clock.Now()
	job.UpdatedAt = job.CancelledAt
	if err := s.repo.UpdateIfStatus(ctx, job, previous); err != nil {
		return Job{}, err
	}
	attempt := newAttempt("operator", "cancelled", "test-genie/remediation/"+job.ID+"/cancel", "", "operator cancelled remediation", job.UpdatedAt)
	if err := s.repo.AppendAttempt(ctx, attempt, job.ID); err != nil {
		return Job{}, err
	}
	return s.repo.Get(ctx, job.ID)
}
func (s *Service) Fail(ctx context.Context, id, failure string) (Job, error) {
	job, err := s.repo.Get(ctx, id)
	if err != nil {
		return Job{}, err
	}
	if !IsActiveStatus(job.Status) {
		return Job{}, ErrInvalidState
	}
	previous := job.Status
	job.Status = JobStatusFailed
	job.Failure = failure
	job.UpdatedAt = s.clock.Now()
	if err := s.repo.UpdateIfStatus(ctx, job, previous); err != nil {
		return Job{}, err
	}
	kind := "lifecycle"
	if job.Attribution.RunID != "" {
		kind = "agent"
	}
	attempt := newAttempt(kind, "failed", "test-genie/remediation/"+job.ID+"/failure", job.Attribution.RoleRef, failure, job.UpdatedAt)
	attempt.TaskID, attempt.RunID = job.Attribution.TaskID, job.Attribution.RunID
	if err := s.repo.AppendAttempt(ctx, attempt, job.ID); err != nil {
		return Job{}, err
	}
	return s.repo.Get(ctx, job.ID)
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
	if err := s.repo.UpdateIfStatus(ctx, job, from); err != nil {
		return Job{}, err
	}
	return job, nil
}
