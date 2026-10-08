// Package operatorinputs owns the queue lifecycle behind the onboarding
// operator-inputs service. It deliberately has no transport dependencies.
package operatorinputs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/operatorcapability"
)

var (
	ErrInvalidAnswer      = errors.New("invalid operator input answer")
	ErrFailedPrecondition = errors.New("operator capability is not ready")
	ErrRevisionConflict   = errors.New("operator input configuration revision conflict")
)

// CapabilityApplier is the narrow seam to the control plane. Queue handling
// remains testable without importing the onboarding server or a transport.
type CapabilityApplier interface {
	ApplyCapability(context.Context, operatorcapability.ActionRequest) (operatorcapability.Result, error)
}

type Service struct {
	Applier         CapabilityApplier
	CurrentRevision func(context.Context) (string, error)
}

func (s Service) List(context.Context) (operatorcapability.Pending, error) {
	return operatorcapability.Load()
}

func (s Service) Resolve(ctx context.Context, expectedRevision string, answers []operatorcapability.Answer) error {
	if len(answers) == 0 {
		return fmt.Errorf("%w: at least one answer is required", ErrInvalidAnswer)
	}
	if expectedRevision = strings.TrimSpace(expectedRevision); expectedRevision != "" {
		if s.CurrentRevision == nil {
			return fmt.Errorf("%w: current configuration revision is unavailable", ErrRevisionConflict)
		}
		current, err := s.CurrentRevision(ctx)
		if err != nil {
			return fmt.Errorf("read current configuration revision: %w", err)
		}
		if current = strings.TrimSpace(current); current != expectedRevision {
			return fmt.Errorf("%w: expected %q, found %q; reload the target questions and retry", ErrRevisionConflict, expectedRevision, current)
		}
	}
	if _, err := operatorcapability.ResolveWith(answers, func(values map[string]string) error {
		queue, err := operatorcapability.Load()
		if err != nil {
			return fmt.Errorf("load operator input queue for apply: %w", err)
		}
		requests, err := BuildCapabilityRequests(queue, values)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidAnswer, err)
		}
		defer ClearCapabilityRequests(requests)
		if len(requests) == 0 {
			return nil
		}
		if s.Applier == nil {
			return errors.New("operator capability applier is unavailable")
		}
		for _, request := range requests {
			result, applyErr := s.Applier.ApplyCapability(ctx, request)
			if applyErr != nil {
				return fmt.Errorf("apply capability %q: %w", request.CapabilityID, applyErr)
			}
			if result.State != operatorcapability.StateReady {
				remediation := strings.TrimSpace(result.Remediation)
				if remediation == "" {
					remediation = "review the capability status and retry"
				}
				return fmt.Errorf("%w: capability %q did not become ready: %s", ErrFailedPrecondition, request.CapabilityID, remediation)
			}
		}
		return nil
	}); err != nil {
		if errors.Is(err, ErrFailedPrecondition) || errors.Is(err, ErrInvalidAnswer) {
			return err
		}
		return err
	}
	return nil
}

func BuildCapabilityRequests(queue operatorcapability.Pending, values map[string]string) ([]operatorcapability.ActionRequest, error) {
	groups := map[string]*operatorcapability.ActionRequest{}
	for _, request := range queue.Requests {
		if request.Decision == operatorcapability.DecisionDeclined {
			continue
		}
		capabilityID := strings.TrimSpace(request.CapabilityID)
		inputID := strings.TrimSpace(request.InputID)
		if capabilityID == "" || inputID == "" {
			return nil, fmt.Errorf("operator input %q is missing generic capability metadata", request.ID)
		}
		value, ok := values[request.ID]
		if !ok {
			continue
		}
		group := groups[capabilityID]
		if group == nil {
			group = &operatorcapability.ActionRequest{CapabilityID: capabilityID, Inputs: map[string]json.RawMessage{}}
			groups[capabilityID] = group
		}
		if inputID == "confirm" {
			group.Confirm = strings.EqualFold(strings.TrimSpace(value), "true")
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("encode operator input %q: %w", request.ID, err)
		}
		if request.Kind == operatorcapability.KindBoolean || request.Kind == operatorcapability.KindConfirmation {
			if value != "true" && value != "false" {
				return nil, fmt.Errorf("operator input %q must be boolean", request.ID)
			}
			encoded = json.RawMessage(value)
		}
		group.Inputs[inputID] = encoded
	}
	requests := make([]operatorcapability.ActionRequest, 0, len(groups))
	for _, request := range groups {
		request.IdempotencyKey = operatorcapability.StableIdempotencyKey(request.CapabilityID, request.Inputs)
		requests = append(requests, *request)
	}
	return requests, nil
}

func ClearCapabilityRequests(requests []operatorcapability.ActionRequest) {
	for i := range requests {
		for key, value := range requests[i].Inputs {
			for index := range value {
				value[index] = 0
			}
			delete(requests[i].Inputs, key)
		}
	}
}
