package renderjobs

import (
	"context"
	"errors"

	"device-studio/internal/jobs"
)

// errDeterministicPreparation marks a successfully read immutable closure that
// is invalid by contract. I/O, provider, and route failures must not use it.
var errDeterministicPreparation = errors.New("renderjobs: deterministic preparation rejection")

// subjectAdapter owns only typed manifest/input preparation, completion verification,
// and projection of a known terminal result. Coordinator retains the single
// attempt, native operation, cancellation, recovery, and release lifecycle.
type subjectAdapter interface {
	prepare(context.Context, *Coordinator, routePin, jobs.Job, jobs.Attempt) (preparedOperation, error)
	validateCompletion(context.Context, *Coordinator, routePin, jobs.Job, jobs.Attempt, persistedIntent, string, string) (jobs.CompletionEvidence, error)
	finalizeTerminal(context.Context, *Coordinator, jobs.SubjectFinalization) error
}

type stillSubjectAdapter struct{}

func (stillSubjectAdapter) prepare(ctx context.Context, c *Coordinator, route routePin, job jobs.Job, attempt jobs.Attempt) (preparedOperation, error) {
	closure, err := c.loadClosure(ctx, route, job)
	if err != nil {
		return preparedOperation{}, err
	}
	return c.prepareAttemptFromClosure(ctx, route, job, attempt, closure)
}

func (stillSubjectAdapter) validateCompletion(ctx context.Context, c *Coordinator, route routePin, job jobs.Job, attempt jobs.Attempt, intent persistedIntent, nativeKey, nativeSHA string) (jobs.CompletionEvidence, error) {
	return c.validateAndPublish(ctx, route, job, attempt, intent, nativeKey, nativeSHA)
}

func (stillSubjectAdapter) finalizeTerminal(context.Context, *Coordinator, jobs.SubjectFinalization) error {
	return nil
}

func (c *Coordinator) adapterFor(job jobs.Job) (subjectAdapter, error) {
	kind := job.Input.SubjectKind
	if kind == "" {
		kind = jobs.SubjectStillRender
	}
	adapter, ok := c.adapters[kind]
	if !ok || adapter == nil {
		return nil, errors.New("renderjobs: unsupported job subject")
	}
	return adapter, nil
}

func operationModeForSubject(kind jobs.SubjectKind) (operationMode, error) {
	switch kind {
	case "", jobs.SubjectStillRender:
		return operationStillRender, nil
	case jobs.SubjectSourceInspection:
		return operationSourceInspect, nil
	case jobs.SubjectVideoRender:
		return operationVideoRender, nil
	case jobs.SubjectPhysicalRender:
		return operationPhysicalRender, nil
	case jobs.SubjectGeometryCreate:
		return operationGeometryCreate, nil
	default:
		return "", errors.New("renderjobs: unsupported job subject")
	}
}

func operationModeForJob(input jobs.Input) (operationMode, error) {
	if input.SubjectKind == jobs.SubjectPhysicalRender {
		switch input.OutputProfile {
		case OutputProfilePNG:
			return operationPhysicalRender, nil
		case OutputProfilePhysicalPreviewPNG:
			return operationPhysicalPreview, nil
		case OutputProfilePhysicalDiagnosticPNG:
			return operationPhysicalPreview, nil
		case OutputProfilePhysicalMotionMP4:
			return operationPhysicalMotion, nil
		case OutputProfilePhysicalMotionMP4V2:
			return operationPhysicalMotionV2, nil
		case OutputProfilePhysicalRecordingMP4:
			return operationPhysicalRecording, nil
		case OutputProfilePhysicalDiagnosticMotionMP4:
			return operationPhysicalRecording, nil
		default:
			return "", errors.New("renderjobs: physical job output profile is unsupported")
		}
	}
	if input.OutputProfile == OutputProfilePhysicalMotionMP4 || input.OutputProfile == OutputProfilePhysicalMotionMP4V2 || input.OutputProfile == OutputProfilePhysicalRecordingMP4 || input.OutputProfile == OutputProfilePhysicalDiagnosticMotionMP4 {
		return "", errors.New("renderjobs: physical motion output profile requires a physical render subject")
	}
	return operationModeForSubject(input.SubjectKind)
}

func isPhysicalOperation(mode operationMode) bool {
	return mode == operationPhysicalRender || mode == operationPhysicalPreview || mode == operationPhysicalMotion || mode == operationPhysicalMotionV2 || mode == operationPhysicalRecording
}

func (c *Coordinator) finalizeTerminal(ctx context.Context, route routePin, job jobs.Job) error {
	if err := c.requireRoute(ctx, route); err != nil {
		return err
	}
	pending, err := c.jobs.PendingSubjectFinalization(ctx, job.ID)
	if errors.Is(err, jobs.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	adapter, ok := c.adapters[pending.Kind]
	if !ok || adapter == nil {
		return errors.New("renderjobs: no adapter can finalize terminal subject")
	}
	attempt, ok := currentAttempt(job)
	if !ok || pending.AttemptID != attempt.ID || pending.Kind != job.Input.SubjectKind || pending.State != job.State {
		return errors.New("renderjobs: terminal projection does not match the routed job attempt")
	}
	if err := adapter.finalizeTerminal(ctx, c, pending); err != nil {
		return err
	}
	if err := c.requireRoute(ctx, route); err != nil {
		return err
	}
	return c.jobs.CompleteSubjectFinalization(ctx, pending.JobID, pending.AttemptID, pending.Kind)
}
