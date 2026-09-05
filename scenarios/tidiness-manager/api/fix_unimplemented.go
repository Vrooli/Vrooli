package main

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// PreviewFix and ApplyFix satisfy the shared ScenarioValidationService Fix RPC
// added to the contract. tidiness-manager ships no deterministic autofixer, so both
// return Unimplemented; the test-genie deterministic-fix aggregate records this
// provider as "no_fixer" and skips it cleanly.
func (*scenarioValidationHandler) PreviewFix(context.Context, *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errNoDeterministicFixer)
}

func (*scenarioValidationHandler) ApplyFix(context.Context, *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errNoDeterministicFixer)
}

var errNoDeterministicFixer = errors.New("tidiness-manager has no deterministic fixer")
