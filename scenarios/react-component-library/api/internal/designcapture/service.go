package designcapture

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Dispatcher starts work owned by BAS and returns its execution identity.
// Any error is ambiguous unless the adapter proves dispatch did not occur.
type Dispatcher interface {
	Start(context.Context, Operation) (string, error)
}
type NotDispatchedError struct{ Err error }

func (e NotDispatchedError) Error() string { return e.Err.Error() }
func (e NotDispatchedError) Unwrap() error { return e.Err }

type Service struct {
	Repository Repository
	Dispatcher Dispatcher
}

// Start persists intent and claims dispatch exactly once. Caller cancellation
// does not cancel the bounded dispatch acknowledgement or erase its evidence.
func (s Service) Start(ctx context.Context, key string, request Request) (Operation, error) {
	if s.Repository == nil || s.Dispatcher == nil {
		return Operation{}, fmt.Errorf("capture repository and browser dispatcher are required")
	}
	op, err := s.Repository.Create(ctx, key, request)
	if err != nil {
		return op, err
	}
	if op.State != Prepared {
		return op, nil
	}
	claimed, err := s.Repository.Transition(ctx, op.ID, op.Version, Dispatching, "", nil, "")
	if errors.Is(err, ErrConflict) {
		return s.Repository.Create(ctx, key, request)
	}
	if err != nil {
		return op, err
	}
	op = claimed
	dispatchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	producer, dispatchErr := s.Dispatcher.Start(dispatchCtx, op)
	cancel()
	// Use a separate persistence budget even if dispatch timed out.
	saveCtx, saveCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer saveCancel()
	next, detail := Running, ""
	if dispatchErr != nil || producer == "" {
		next = DispatchUnknown
		detail = "Browser dispatch was not acknowledged; recover the producer identity before continuing."
		var rejected NotDispatchedError
		if errors.As(dispatchErr, &rejected) {
			next = Failed
			detail = "Browser dispatch was rejected before submission."
		}
		producer = ""
	}
	saved, saveErr := s.Repository.Transition(saveCtx, op.ID, op.Version, next, producer, nil, detail)
	if saveErr != nil {
		return op, fmt.Errorf("persist browser dispatch acknowledgement for %s: %w", op.ID, saveErr)
	}
	return saved, nil
}

// Canceller requests producer cancellation. Acknowledgement is not terminal
// evidence; Attach must still observe the producer's final outcome.
type Canceller interface {
	Cancel(context.Context, Operation) error
}

// Cancel durably records intent before contacting the producer. Retrying an
// uncertain request addresses the same producer and never starts new work.
func (s Service) Cancel(ctx context.Context, id string, canceller Canceller) (Operation, error) {
	if s.Repository == nil || canceller == nil {
		return Operation{}, fmt.Errorf("capture repository and producer canceller are required")
	}
	op, err := s.Repository.Get(ctx, id)
	if err != nil {
		return op, err
	}
	switch op.State {
	case Completed, Failed, Cancelled:
		return op, nil
	case Prepared:
		return s.Repository.Transition(ctx, op.ID, op.Version, Cancelled, "", nil, "Capture cancelled before browser dispatch.")
	case Running:
		op, err = s.Repository.Transition(ctx, op.ID, op.Version, CancelRequested, op.ProducerID, nil, "Cancellation requested; refresh evidence to confirm the producer outcome.")
		if err != nil {
			return op, err
		}
	case CancelRequested:
		// BAS stop is idempotent, so a lost acknowledgement can be retried.
	default:
		return op, fmt.Errorf("capture has no acknowledged producer to cancel; recover dispatch identity first")
	}
	cancelCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	stopErr := canceller.Cancel(cancelCtx, op)
	cancel()
	readCtx, readCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer readCancel()
	current, err := s.Repository.Get(readCtx, id)
	if err != nil {
		return op, err
	}
	if current.State == Completed || current.State == Failed || current.State == Cancelled {
		return current, nil
	}
	if stopErr != nil {
		return current, fmt.Errorf("cancellation acknowledgement unavailable for %s; intent retained, refresh evidence or retry cancellation: %w", id, stopErr)
	}
	return current, nil
}

// Retry starts an explicitly requested new attempt of the immutable render.
// It preserves the previous operation and gives the new intent its own identity.
func (s Service) Retry(ctx context.Context, id, key string) (Operation, error) {
	if s.Repository == nil {
		return Operation{}, fmt.Errorf("capture repository is required")
	}
	previous, err := s.Repository.Get(ctx, id)
	if err != nil {
		return Operation{}, err
	}
	if previous.State != Completed && previous.State != Failed && previous.State != Cancelled {
		return Operation{}, fmt.Errorf("only a terminal capture can start another attempt")
	}
	request := previous.Request
	request.PreviousID = previous.ID
	return s.Start(ctx, key, request)
}
