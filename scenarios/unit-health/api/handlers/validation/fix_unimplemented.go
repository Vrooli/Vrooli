package validation

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// PreviewFix and ApplyFix satisfy the shared ScenarioValidationService Fix RPC
// added to the contract. unit-health ships no deterministic autofixer, so both
// return Unimplemented; the test-genie deterministic-fix aggregate records this
// provider as "no_fixer" and skips it cleanly.
func (*SharedHandler) PreviewFix(context.Context, *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errNoDeterministicFixer)
}

func (*SharedHandler) ApplyFix(context.Context, *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errNoDeterministicFixer)
}

var errNoDeterministicFixer = errors.New("unit-health has no deterministic fixer")
