// Package capabilities owns the transport-free capability lifecycle exposed by
// onboarding. Provider policy stays in the control plane; this package only
// validates the action boundary and projects provider results.
package capabilities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/operatorcapability"
)

type Executor interface {
	DiscoverCapabilities(context.Context) ([]operatorcapability.Status, error)
	PreviewCapability(context.Context, operatorcapability.ActionRequest) (operatorcapability.Preview, error)
	ApplyCapability(context.Context, operatorcapability.ActionRequest) (operatorcapability.Result, error)
	RemoveCapability(string) error
}

type Service struct{ Executor Executor }

func (s Service) List(ctx context.Context) ([]operatorcapability.Status, error) {
	if s.Executor == nil {
		return nil, errors.New("capability executor is unavailable")
	}
	return s.Executor.DiscoverCapabilities(ctx)
}

func (s Service) Status(ctx context.Context) ([]operatorcapability.Status, error) {
	return s.List(ctx)
}

func (s Service) Preview(ctx context.Context, request operatorcapability.ActionRequest) (operatorcapability.Preview, error) {
	if err := NormalizeActionRequest(&request); err != nil {
		return operatorcapability.Preview{}, err
	}
	if s.Executor == nil {
		return operatorcapability.Preview{}, errors.New("capability executor is unavailable")
	}
	return s.Executor.PreviewCapability(ctx, request)
}

func (s Service) Apply(ctx context.Context, request operatorcapability.ActionRequest) (operatorcapability.Result, error) {
	if err := NormalizeActionRequest(&request); err != nil {
		return operatorcapability.Result{}, err
	}
	if !request.Confirm {
		return operatorcapability.Result{
			CapabilityID: request.CapabilityID,
			State:        operatorcapability.StateReadyToPreview,
			Outcome:      "confirmation_required",
			ErrorCode:    "confirmation_required",
			Retryable:    true,
			Remediation:  "review the preview and confirm the exact planned mutations",
		}, nil
	}
	if s.Executor == nil {
		return operatorcapability.Result{}, errors.New("capability executor is unavailable")
	}
	result, err := s.Executor.ApplyCapability(ctx, request)
	if err != nil {
		return result, err
	}
	if result.State == operatorcapability.StateReady && s.Executor != nil {
		if err := s.Executor.RemoveCapability(request.CapabilityID); err != nil {
			return result, fmt.Errorf("capability applied but pending operator metadata could not be reconciled: %w", err)
		}
	}
	return result, nil
}

func NormalizeActionRequest(request *operatorcapability.ActionRequest) error {
	if request == nil {
		return errors.New("capability action is required")
	}
	request.CapabilityID = strings.TrimSpace(request.CapabilityID)
	if request.CapabilityID == "" {
		return errors.New("capability_id is required")
	}
	if request.IdempotencyKey == "" {
		request.IdempotencyKey = operatorcapability.StableIdempotencyKey(request.CapabilityID, request.Inputs)
	}
	return request.Validate()
}

func RawInputs(values map[string]any) (map[string]json.RawMessage, error) {
	if len(values) == 0 {
		return nil, nil
	}
	result := make(map[string]json.RawMessage, len(values))
	for key, value := range values {
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("encode capability input %q: %w", key, err)
		}
		result[key] = encoded
	}
	return result, nil
}
