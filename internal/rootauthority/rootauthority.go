// Package rootauthority verifies the operator's standing grant for a
// recoverable root control-plane transition.
//
// A transition that crosses a stored-format change is not a human-only
// decision. It is authority over a reversible operation: the candidate image
// activates, the prior image stays installed, and a state snapshot is
// retained. That is exactly the shape a standing grant can cover, so agents
// act within it instead of stalling every other agent until an operator is
// free.
//
// Irreversible or externally visible effects remain operator decisions and are
// not covered by this package.
package rootauthority

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/scopecatalog"
	"github.com/vrooli/cli-core/cliutil"
)

const (
	// Policy is the reviewed grant policy a root transition asks for.
	Policy = "root-state-transition"
	// Capability is the scope that gates the policy.
	Capability = "root-state:transition"
	// Operation is the exact operation the grant must name.
	Operation = "root-state.transition"
)

// Target returns the canonical authorization target for one host and runtime
// home. It is exact: a grant for another host or another home does not cover
// this transition, though an operator may approve a bounded wildcard.
//
// The runtime home is absolute; it is written without its leading separator so
// the target has a single stable separator between the host and the path. An
// empty host or home is not a transition target: it resolves to a sentinel that
// no per-host or per-home grant can cover.
func Target(host, home string) string {
	host = strings.TrimSpace(host)
	home = strings.TrimPrefix(strings.TrimSpace(filepath.ToSlash(home)), "/")
	if host == "" || home == "" {
		return Capability + "/_invalid/_invalid"
	}
	return Capability + "/" + host + "/" + home
}

// Refusal is returned when no standing grant covers the target. It carries the
// exact request command so the agent can ask for the narrow authority it needs
// instead of falling back to an operator-only path.
type Refusal struct {
	Target string
	Reason string
}

func (r *Refusal) Error() string {
	base := "root translation authority refused: " + r.Reason
	// The operator path needs no grant, no approval service and no rebuild.
	// It is always the first remedy, because the grant path depends on the
	// running approval service already knowing this policy.
	base += "\nA root transition is recoverable on its own: run it from an operator terminal with `make rebaseline`."
	if r.Target == "" {
		return base
	}
	if strings.TrimSpace(os.Getenv("VROOLI_AGENT_SESSION")) == "" {
		return base
	}
	// Only offer the standing-grant path when this process actually runs inside
	// an agent session; otherwise there is no session to authorize and the
	// command would be unusable in the operator's shell.
	command := authn.AuthorizationRequestCommand(Policy, r.Target)
	if command == "" {
		return base
	}
	return base + "\nTo delegate it instead, ask the operator to approve exactly this transition with: `" + command + "` (the approval service must be running a revision that includes the `" + Policy + "` policy)."
}

// Decision records the verified grant behind one forced transition.
type Decision struct {
	GrantID    string    `json:"grant_id"`
	ApprovedBy string    `json:"approved_by"`
	RunID      string    `json:"run_id"`
	Operation  string    `json:"operation"`
	Target     string    `json:"target"`
	VerifiedAt time.Time `json:"verified_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// Verifier resolves and verifies the paired baseline/operation credential that
// carries a standing grant, then runs the canonical operation evaluator.
type Verifier struct {
	Resolve    func(context.Context, string, string) (cliutil.AgentCredentials, error)
	VerifyPair func(context.Context, string, string) (*cliutil.VerifyPairResult, error)
	Now        func() time.Time
}

// NewVerifier returns a verifier bound to the live agent credential broker.
func NewVerifier() Verifier {
	return Verifier{
		Resolve: cliutil.ResolveAgentCredentialsForOperation,
		VerifyPair: func(ctx context.Context, baseline, operation string) (*cliutil.VerifyPairResult, error) {
			return (cliutil.IdentityEnv{Token: baseline}).VerifyIdentityPair(ctx, operation)
		},
		Now: time.Now,
	}
}

// Verify admits the transition only when an operator-approved grant covers the
// exact target under the reviewed policy.
func (v Verifier) Verify(ctx context.Context, target string) (Decision, error) {
	if strings.TrimSpace(target) == "" {
		return Decision{}, &Refusal{Reason: "no target was named"}
	}
	resolve := v.Resolve
	if resolve == nil {
		resolve = cliutil.ResolveAgentCredentialsForOperation
	}
	credentials, err := resolve(ctx, Operation, target)
	if err != nil {
		return Decision{}, &Refusal{Target: target, Reason: fmt.Sprintf("no approved grant was found (%v)", err)}
	}
	if credentials.BaselineToken == "" || credentials.OperationToken == "" || credentials.LegacyToken != "" {
		return Decision{}, &Refusal{Target: target, Reason: "a paired baseline and operation grant is required; an unpaired session token is not authority"}
	}
	verify := v.VerifyPair
	if verify == nil {
		verify = func(ctx context.Context, baseline, operation string) (*cliutil.VerifyPairResult, error) {
			return (cliutil.IdentityEnv{Token: baseline}).VerifyIdentityPair(ctx, operation)
		}
	}
	verified, err := verify(ctx, credentials.BaselineToken, credentials.OperationToken)
	if err != nil {
		return Decision{}, &Refusal{Target: target, Reason: fmt.Sprintf("the credential broker did not verify the grant pair (%v)", err)}
	}
	if verified == nil || !verified.Valid || verified.Operation == nil || !verified.Operation.Valid || verified.Operation.Claims == nil {
		return Decision{}, &Refusal{Target: target, Reason: "the broker did not verify a valid operation grant"}
	}
	claims := verified.Operation.Claims
	nowFn := v.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	now := nowFn().UTC()
	if strings.TrimSpace(claims.RunID) == "" {
		return Decision{}, &Refusal{Target: target, Reason: "the verified grant names no accountable run"}
	}
	grant := identity.OperationGrantFromMeta(claims.Meta, now)
	if grant == nil {
		return Decision{}, &Refusal{Target: target, Reason: "the verified credential carries no operation grant"}
	}
	if !grantAllowsPolicyTarget(grant, Operation, target, now) {
		return Decision{}, &Refusal{Target: target, Reason: "the verified grant does not cover this operation under the root-state-transition policy"}
	}
	grantExpiry := grant.ExpiresAt.UTC()
	operationExpiry := time.Unix(claims.ExpiresAt, 0).UTC()
	if claims.ExpiresAt <= 0 || !now.Before(grantExpiry) || !now.Before(operationExpiry) {
		return Decision{}, &Refusal{Target: target, Reason: "the verified grant is expired"}
	}
	principal := identity.Principal{
		Kind: identity.ActorAgent, Subject: strings.TrimSpace(claims.RunID),
		Owner: strings.TrimSpace(grant.Approver), Verified: true,
		Scopes: append([]string(nil), claims.Scopes...), Source: identity.SourceAgentProvenance,
		ExpiresAt: operationExpiry, Grant: grant,
	}
	if _, err := authn.RequireOperation(identity.WithPrincipal(ctx, principal), Capability, Operation, target); err != nil {
		return Decision{}, &Refusal{Target: target, Reason: "the canonical operation evaluator refused the target: " + err.Error()}
	}
	expires := grantExpiry
	if operationExpiry.Before(expires) {
		expires = operationExpiry
	}
	return Decision{
		GrantID: grant.ID, ApprovedBy: strings.TrimSpace(grant.Approver), RunID: strings.TrimSpace(claims.RunID),
		Operation: Operation, Target: target, VerifiedAt: now, ExpiresAt: expires,
	}, nil
}

// grantAllowsPolicyTarget keeps policy pairing and target coverage scoped to
// this policy's rule while delegating exact and approved domain-wildcard
// matching to the canonical OperationGrant evaluator.
func grantAllowsPolicyTarget(grant *identity.OperationGrant, operation, target string, now time.Time) bool {
	if grant == nil {
		return false
	}
	if len(grant.Rules) > 0 {
		for _, rule := range grant.Rules {
			if rule.Policy != Policy {
				continue
			}
			scoped := *grant
			scoped.Rules = nil
			scoped.Policy = rule.Policy
			scoped.Operations = rule.Operations
			scoped.Targets = rule.Targets
			if scoped.Allows(operation, target, now) {
				return true
			}
		}
		return false
	}
	return grant.Policy == Policy && grant.Allows(operation, target, now)
}

// ScopeSatisfied reports whether the grant's scopes cover the capability, so a
// caller can explain a refusal that came only from the scope check.
func ScopeSatisfied(scopes []string) bool { return scopecatalog.Resolve(scopes, Capability) }

// IsRefusal reports whether err is an authority refusal rather than an
// infrastructure failure.
func IsRefusal(err error) bool {
	var refusal *Refusal
	return errors.As(err, &refusal)
}
