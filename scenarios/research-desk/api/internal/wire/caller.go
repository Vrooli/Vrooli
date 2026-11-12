package wire

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
)

// OperatorAction is the refusal's instruction to an agent (rulings F13).
const OperatorAction = "this is the operator's action; write a needs-you item instead"

// ErrNotOperator is returned when a caller that proved it is an agent or a
// service calls a procedure that acts for the operator.
var ErrNotOperator = errors.New(OperatorAction)

// Caller is nil when the request may act for the operator: it carries no
// verified agent or service identity and no agent identity that failed
// verification. A local caller with no token is indistinguishable from the
// operator's shell (SYSTEM.md §2); the desk keeps a verified caller's own
// identity and never replaces it with a broader one.
func Caller(ctx context.Context) error {
	agent := provenance.FromContext(ctx)
	switch {
	case agent.IsVerifiedAgent():
		return fmt.Errorf("%w (caller: agent run %s)", ErrNotOperator, agent.RunID)
	case agent.VerificationStatus == provenance.VerificationInvalid, agent.VerificationStatus == provenance.VerificationUnavailable:
		return fmt.Errorf("%w (caller offered an agent identity that is %s)", ErrNotOperator, agent.VerificationStatus)
	}
	if principal, ok := identity.PrincipalFromContext(ctx); ok && principal.IsVerified() {
		switch principal.Kind {
		case identity.ActorAgent, identity.ActorService:
			return fmt.Errorf("%w (caller: %s %s)", ErrNotOperator, principal.Kind, principal.Subject)
		}
	}
	return nil
}

// OperatorOnly refuses the named procedures to a caller Caller refuses, with
// PermissionDenied and the reason; every other procedure stays open.
func OperatorOnly(procedures ...string) connect.HandlerOption {
	only := make(map[string]bool, len(procedures))
	for _, p := range procedures {
		only[p] = true
	}
	return connect.WithInterceptors(connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if only[req.Spec().Procedure] {
				if err := Caller(ctx); err != nil {
					return nil, connect.NewError(connect.CodePermissionDenied, err)
				}
			}
			return next(ctx, req)
		}
	}))
}
