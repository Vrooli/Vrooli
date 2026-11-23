package wire

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
)

// OperatorAction is the refusal's instruction to an agent.
const OperatorAction = "this is the operator's action; write a needs-you item instead"

// ErrNotOperator is returned when a caller that proved it is an agent or a
// service calls a procedure that acts for the operator.
var ErrNotOperator = errors.New(OperatorAction)

// ErrNotSteward is returned when an unverified caller attempts to use the
// attributed, lower-priority steward channel.
var ErrNotSteward = errors.New("a steward finding requires a verified agent run")

// VerifiedAgent reports whether the request carries a trusted agent run.
func VerifiedAgent(ctx context.Context) bool { return provenance.FromContext(ctx).IsVerifiedAgent() }

// ResearchOperation preserves local operator access while routing delegated
// launch/control through the shared, live operation-grant evaluator. Agent
// attribution must never fall back to an ambient human availability principal.
func ResearchOperation(ctx context.Context, operation, id string) error {
	_, err := researchOperation(ctx, operation, "research-desk:commission/"+id)
	return err
}

// DecisionTarget is the grant target of one needs-you item: its owner as
// needs-you lists it (a commission, topic:<slug> or supervisor) and its ID.
// A research-decisions grant on research-desk:decision/* covers every item.
func DecisionTarget(owner, item string) string {
	return "research-desk:decision/" + owner + "/" + item
}

// ResearchDecision admits the operator, or an agent whose operator-approved
// research-decisions grant covers the item. For an agent it returns who
// answered and under which grant, which the desk writes into the entry so a
// delegated answer is never mistaken for one the operator typed.
func ResearchDecision(ctx context.Context, owner, item string) (delegation string, err error) {
	return researchOperation(ctx, "research.answer", DecisionTarget(owner, item))
}

func researchOperation(ctx context.Context, operation, target string) (string, error) {
	attribution := provenance.FromContext(ctx)
	if attribution.VerificationStatus == provenance.VerificationInvalid || attribution.VerificationStatus == provenance.VerificationUnavailable {
		return "", ErrNotOperator
	}
	if err := Caller(ctx); err == nil {
		return "", nil
	}
	p, ok := identity.PrincipalFromContext(ctx)
	if !ok || !p.IsVerified() || p.IsHuman() {
		return "", authn.NewOperationRefusal(operation, target)
	}
	admitted, grant, err := authn.AdmitOperation(ctx, "research-desk:write", operation, target)
	if err != nil {
		return "", err
	}
	return delegation(attribution, admitted, grant), nil
}

// delegation names the agent, the grant that admitted it (in its token or a
// paired operation proof) and the person who approved that grant.
func delegation(a provenance.Provenance, p identity.Principal, grant *identity.OperationGrant) string {
	who := a.RunID
	if who == "" {
		who = p.Subject
	}
	text := "agent run " + who + " under the operator's operation grant"
	if grant != nil && grant.ID != "" {
		text += " " + grant.ID
		if grant.Approver != "" {
			text += ", approved by " + grant.Approver
		}
	}
	return text
}

// Caller is nil when the request may act for the operator: it carries no
// verified agent or service identity and no agent identity that failed
// verification. A local caller with no token is indistinguishable from the
// operator's shell (SYSTEM.md §10); the desk keeps a verified caller's own
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
