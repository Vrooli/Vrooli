package agentsession

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/cliutil"
)

// AgentGrantApprovalVerifier resolves the current owner-approved operation
// credential for one proof-bound target, verifies its baseline/operation pair
// with Agent Manager, and runs the canonical authn operation evaluator.
type AgentGrantApprovalVerifier struct {
	Resolve func(context.Context, string, string) (cliutil.AgentCredentials, error)
	Verify  func(context.Context, string, string) (*cliutil.VerifyPairResult, error)
	Now     func() time.Time
}

func NewAgentGrantApprovalVerifier() AgentGrantApprovalVerifier {
	return AgentGrantApprovalVerifier{
		Resolve: cliutil.ResolveAgentCredentialsForOperation,
		Verify: func(ctx context.Context, baseline, operation string) (*cliutil.VerifyPairResult, error) {
			return (cliutil.IdentityEnv{Token: baseline}).VerifyIdentityPair(ctx, operation)
		},
		Now: time.Now,
	}
}

func (v AgentGrantApprovalVerifier) VerifyExactApproval(ctx context.Context, proof ExactReapProof) (ExactReapApproval, error) {
	target := ExactReapAuthorizationTarget(proof)
	if target == "" || proof.Operation != "agent-session.reap" {
		return ExactReapApproval{}, errors.New("exact reap approval: invalid operation or proof-bound target")
	}
	resolve := v.Resolve
	if resolve == nil {
		resolve = cliutil.ResolveAgentCredentialsForOperation
	}
	credentials, err := resolve(ctx, proof.Operation, target)
	if err != nil {
		return ExactReapApproval{}, fmt.Errorf("resolve owner-approved exact-scope grant: %w", err)
	}
	if credentials.BaselineToken == "" || credentials.OperationToken == "" || credentials.LegacyToken != "" {
		return ExactReapApproval{}, errors.New("exact reap approval requires a paired baseline and operation grant")
	}
	verify := v.Verify
	if verify == nil {
		verify = func(ctx context.Context, baseline, operation string) (*cliutil.VerifyPairResult, error) {
			return (cliutil.IdentityEnv{Token: baseline}).VerifyIdentityPair(ctx, operation)
		}
	}
	verified, err := verify(ctx, credentials.BaselineToken, credentials.OperationToken)
	if err != nil {
		return ExactReapApproval{}, fmt.Errorf("verify baseline and operation credentials: %w", err)
	}
	if verified == nil || !verified.Valid || verified.Operation == nil || !verified.Operation.Valid || verified.Operation.Claims == nil {
		return ExactReapApproval{}, errors.New("Agent Manager did not verify the exact-session grant pair")
	}
	claims := verified.Operation.Claims
	nowFn := v.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	now := nowFn().UTC()
	grant := identity.OperationGrantFromMeta(claims.Meta, now)
	if strings.TrimSpace(claims.RunID) == "" {
		return ExactReapApproval{}, errors.New("verified operation grant has no run identity")
	}
	if !grantAllowsReapPolicyTarget(grant, proof.Operation, target, now) {
		return ExactReapApproval{}, errors.New("verified grant does not cover this request under the agent-session-reaping policy")
	}
	grantExpiry := grant.ExpiresAt.UTC()
	operationExpiry := time.Unix(claims.ExpiresAt, 0).UTC()
	if !now.Before(grantExpiry) || claims.ExpiresAt <= 0 || !now.Before(operationExpiry) {
		return ExactReapApproval{}, errors.New("verified operation grant is expired")
	}
	principal := identity.Principal{
		Kind: identity.ActorAgent, Subject: strings.TrimSpace(claims.RunID),
		Owner: strings.TrimSpace(grant.Approver), Verified: true,
		Scopes: append([]string(nil), claims.Scopes...), Source: identity.SourceAgentProvenance,
		ExpiresAt: operationExpiry, Grant: grant,
	}
	if _, err := authn.RequireOperation(identity.WithPrincipal(ctx, principal), "agent-session:reap", proof.Operation, target); err != nil {
		return ExactReapApproval{}, fmt.Errorf("canonical operation authorization refused exact target: %w", err)
	}
	expires := grantExpiry
	if operationExpiry.Before(expires) {
		expires = operationExpiry
	}
	return ExactReapApproval{
		DecisionID: grant.ID, ApprovedBy: grant.Approver, Operation: proof.Operation,
		Target: target, ProofDigest: proof.Digest, VerifiedAt: now, ExpiresAt: expires,
	}, nil
}

func grantAllowsReapPolicyTarget(grant *identity.OperationGrant, operation, target string, now time.Time) bool {
	if grant == nil {
		return false
	}
	if len(grant.Rules) > 0 {
		for _, rule := range grant.Rules {
			if rule.Policy == "agent-session-reaping" {
				// Keep policy pairing and target coverage scoped to this rule, while
				// delegating exact and approved domain-wildcard matching to the
				// canonical OperationGrant evaluator.
				scoped := *grant
				scoped.Rules = nil
				scoped.Policy = rule.Policy
				scoped.Operations = rule.Operations
				scoped.Targets = rule.Targets
				if scoped.Allows(operation, target, now) {
					return true
				}
			}
		}
		return false
	}
	return grant.Policy == "agent-session-reaping" && grant.Allows(operation, target, now)
}
