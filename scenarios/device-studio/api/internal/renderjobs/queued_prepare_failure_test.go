package renderjobs

import (
	"context"
	"errors"
	"testing"

	"device-studio/internal/jobs"
)

type preparationFailureAdapter struct {
	err    error
	before func(context.Context, *Coordinator, jobs.Job, jobs.Attempt) error
}

func (a preparationFailureAdapter) prepare(ctx context.Context, c *Coordinator, _ routePin, job jobs.Job, attempt jobs.Attempt) (preparedOperation, error) {
	if a.before != nil {
		if err := a.before(ctx, c, job, attempt); err != nil {
			return preparedOperation{}, err
		}
	}
	return preparedOperation{}, a.err
}

func (preparationFailureAdapter) validateCompletion(context.Context, *Coordinator, routePin, jobs.Job, jobs.Attempt, persistedIntent, string, string) (jobs.CompletionEvidence, error) {
	return jobs.CompletionEvidence{}, nil
}

func (preparationFailureAdapter) finalizeTerminal(context.Context, *Coordinator, jobs.SubjectFinalization) error {
	return nil
}

func TestDeterministicQueuedManifestFailureTerminalizesWithoutExecution(t *testing.T) {
	ops := &integrationOps{}
	c, ctx, _ := integrationCoordinator(t, ops, false)
	route, err := c.route(ctx)
	if err != nil {
		t.Fatal(err)
	}
	job := submitVideoDispatchFixture(t, c, route, "invalid-video-manifest")
	c.adapters[jobs.SubjectVideoRender] = preparationFailureAdapter{err: errors.Join(errDeterministicPreparation, errors.New("worker manifest invalid"))}

	failed, err := c.Advance(ctx, job.ID)
	if !errors.Is(err, errDeterministicPreparation) || failed.State != jobs.Failed {
		t.Fatalf("deterministic preparation failure remained queued: state=%s err=%v", failed.State, err)
	}
	if len(failed.Attempts) != 1 || failed.Attempts[0].State != jobs.AttemptFailed || failed.Attempts[0].Execution != nil || failed.Attempts[0].ExecutionReleasedAt != nil || failed.Attempts[0].FinishedAt == nil {
		t.Fatalf("pre-claim rejection fabricated native evidence: %+v", failed.Attempts)
	}
	if ops.starts != 0 {
		t.Fatalf("native owner started %d operations for a rejected manifest", ops.starts)
	}
	if _, err := c.jobs.PendingSubjectFinalization(ctx, job.ID); err != nil {
		t.Fatalf("terminal projection marker missing: %v", err)
	}
	settled, err := c.Advance(ctx, job.ID)
	if err != nil || settled.State != jobs.Failed {
		t.Fatalf("terminal projection retry: state=%s err=%v", settled.State, err)
	}
	if _, err := c.jobs.PendingSubjectFinalization(ctx, job.ID); !errors.Is(err, jobs.ErrNotFound) {
		t.Fatalf("terminal projection marker not acknowledged: %v", err)
	}
}

func TestTransientPrepareFailureStaysQueuedAndConcurrentClaimWins(t *testing.T) {
	t.Run("transient", func(t *testing.T) {
		ops := &integrationOps{}
		c, ctx, _ := integrationCoordinator(t, ops, false)
		route, err := c.route(ctx)
		if err != nil {
			t.Fatal(err)
		}
		job := submitVideoDispatchFixture(t, c, route, "transient-video-prepare")
		c.adapters[jobs.SubjectVideoRender] = preparationFailureAdapter{err: errors.New("temporary media store timeout")}
		got, err := c.Advance(ctx, job.ID)
		if err == nil || got.State != jobs.Queued || len(got.Attempts) != 1 || got.Attempts[0].Execution != nil {
			t.Fatalf("transient preparation error was terminalized: job=%+v err=%v", got, err)
		}
		if _, err := c.jobs.PendingSubjectFinalization(ctx, job.ID); !errors.Is(err, jobs.ErrNotFound) {
			t.Fatalf("transient error created terminal projection: %v", err)
		}
	})

	t.Run("concurrent claim", func(t *testing.T) {
		ops := &integrationOps{}
		c, ctx, _ := integrationCoordinator(t, ops, false)
		route, err := c.route(ctx)
		if err != nil {
			t.Fatal(err)
		}
		job := submitVideoDispatchFixture(t, c, route, "claim-wins-video-prepare")
		c.adapters[jobs.SubjectVideoRender] = preparationFailureAdapter{
			err: errors.Join(errDeterministicPreparation, errors.New("worker manifest invalid")),
			before: func(ctx context.Context, c *Coordinator, current jobs.Job, attempt jobs.Attempt) error {
				_, startErr := c.jobs.Start(ctx, current.ID, jobs.ExecutionClaimInput{AttemptID: attempt.ID, AttemptNumber: attempt.Number, OwnerID: OwnerID, ScopeID: "concurrent-claim", ArtifactID: "claim", SHA256: digestBytes([]byte("claim"))}, current.Version)
				return startErr
			},
		}
		got, err := c.Advance(ctx, job.ID)
		if !errors.Is(err, jobs.ErrConcurrentTransition) || got.State != jobs.Running || got.Attempts[0].Execution == nil {
			t.Fatalf("deterministic rejection overwrote concurrent execution claim: state=%s attempt=%+v err=%v", got.State, got.Attempts[0], err)
		}
		if _, err := c.jobs.PendingSubjectFinalization(ctx, job.ID); !errors.Is(err, jobs.ErrNotFound) {
			t.Fatalf("claim race created terminal projection: %v", err)
		}
	})
}

func submitVideoDispatchFixture(t *testing.T, c *Coordinator, route routePin, key string) jobs.Job {
	t.Helper()
	digest := digestBytes([]byte(key))
	job, _, err := c.jobs.Submit(context.Background(), jobs.SubmitInput{IdempotencyKey: key, Input: jobs.Input{
		ProjectID: "project", ShotRevisionID: "shot:dispatch-test:revision:1", ShotRevision: 1,
		SourceClockDigest: digest, ToolProfileDigest: digest, RendererVersion: "video-test-v1", OutputProfile: "video-sample-png-v1",
		SubjectKind: jobs.SubjectVideoRender, SubjectID: "dispatch-test-shot",
		Bindings: []jobs.ArtifactBinding{routeBinding(route), {Role: "video_closure", ArtifactID: "closure:" + digest, RevisionID: "sha256:" + digest, SHA256: digest}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	return job
}
