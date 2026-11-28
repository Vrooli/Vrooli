package enginemocks

import (
	"context"
	"encoding/json"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

// Session is a function-backed fake for engine session behavior tests.
// Unconfigured lifecycle methods succeed and an unconfigured Run succeeds with
// an empty outcome, so each test only arranges behavior relevant to its claim.
type Session struct {
	RunFunc             func(context.Context, contracts.CompiledInstruction) (contracts.StepOutcome, error)
	ResetFunc           func(context.Context) error
	CloseFunc           func(context.Context) error
	GetStorageStateFunc func(context.Context) (json.RawMessage, error)
}

func (s *Session) Run(ctx context.Context, instruction contracts.CompiledInstruction) (contracts.StepOutcome, error) {
	if s.RunFunc != nil {
		return s.RunFunc(ctx, instruction)
	}
	return contracts.StepOutcome{Success: true}, nil
}

func (s *Session) Reset(ctx context.Context) error {
	if s.ResetFunc != nil {
		return s.ResetFunc(ctx)
	}
	return nil
}

func (s *Session) Close(ctx context.Context) error {
	if s.CloseFunc != nil {
		return s.CloseFunc(ctx)
	}
	return nil
}

func (s *Session) GetStorageState(ctx context.Context) (json.RawMessage, error) {
	if s.GetStorageStateFunc != nil {
		return s.GetStorageStateFunc(ctx)
	}
	return nil, nil
}
