package agentsession

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/cliutil"
)

func TestAgentGrantApprovalVerifierUsesExactPairAndCanonicalEvaluator(t *testing.T) {
	now := time.Now().UTC()
	session, facts := testExactSession(), testExactFacts()
	proof, err := CaptureExactReapProof(session, session.Scope, facts)
	if err != nil {
		t.Fatal(err)
	}
	target := ExactReapAuthorizationTarget(proof)
	grant := &identity.OperationGrant{ID: "grant-1", Policy: "agent-session-reaping", Approver: "owner-1", Operations: []string{proof.Operation}, Targets: []string{target}, ExpiresAt: now.Add(time.Hour)}
	raw, err := json.Marshal(grant)
	if err != nil {
		t.Fatal(err)
	}
	verifier := AgentGrantApprovalVerifier{
		Now: func() time.Time { return now },
		Resolve: func(_ context.Context, operation, exactTarget string) (cliutil.AgentCredentials, error) {
			if operation != proof.Operation || exactTarget != target {
				t.Fatalf("credential lookup = %s %s", operation, exactTarget)
			}
			return cliutil.AgentCredentials{BaselineToken: "baseline", OperationToken: "operation"}, nil
		},
		Verify: func(_ context.Context, baseline, operation string) (*cliutil.VerifyPairResult, error) {
			if baseline != "baseline" || operation != "operation" {
				t.Fatalf("verification pair = %q %q", baseline, operation)
			}
			return &cliutil.VerifyPairResult{Valid: true, Operation: &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{
				RunID: "run-1", Subject: "owner-1", Scopes: []string{"agent-session:reap"}, ExpiresAt: now.Add(30 * time.Minute).Unix(), Meta: map[string]string{"operation_grant": string(raw)},
			}}}, nil
		},
	}
	approval, err := verifier.VerifyExactApproval(context.Background(), proof)
	if err != nil {
		t.Fatalf("verify exact approval: %v", err)
	}
	if approval.DecisionID != grant.ID || approval.ApprovedBy != grant.Approver || approval.Target != target || approval.ProofDigest != proof.Digest || !approval.ExpiresAt.Equal(time.Unix(now.Add(30*time.Minute).Unix(), 0).UTC()) {
		t.Fatalf("approval receipt not bound to current exact grant: %+v", approval)
	}
}

func TestAgentGrantApprovalVerifierAllowsCanonicalWildcardAndRejectsLegacyGrant(t *testing.T) {
	now := time.Now().UTC()
	session, facts := testExactSession(), testExactFacts()
	proof, err := CaptureExactReapProof(session, session.Scope, facts)
	if err != nil {
		t.Fatal(err)
	}
	wildcard := &identity.OperationGrant{ID: "grant-wild", Policy: "agent-session-reaping", Approver: "owner", Operations: []string{proof.Operation}, Targets: []string{"agent-session:*"}, ExpiresAt: now.Add(time.Hour)}
	raw, _ := json.Marshal(wildcard)
	verifier := AgentGrantApprovalVerifier{
		Now: func() time.Time { return now },
		Resolve: func(context.Context, string, string) (cliutil.AgentCredentials, error) {
			return cliutil.AgentCredentials{BaselineToken: "baseline", OperationToken: "operation"}, nil
		},
		Verify: func(context.Context, string, string) (*cliutil.VerifyPairResult, error) {
			return &cliutil.VerifyPairResult{Valid: true, Operation: &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{
				RunID: "run-1", Scopes: []string{"agent-session:reap"}, ExpiresAt: now.Add(time.Hour).Unix(), Meta: map[string]string{"operation_grant": string(raw)},
			}}}, nil
		},
	}
	if approval, err := verifier.VerifyExactApproval(context.Background(), proof); err != nil || approval.Target != ExactReapAuthorizationTarget(proof) {
		t.Fatalf("canonical policy wildcard did not cover exact proof-bound target: approval=%+v err=%v", approval, err)
	}
	wrongPolicy := *wildcard
	wrongPolicy.Policy = "agent-management"
	wrongRaw, _ := json.Marshal(&wrongPolicy)
	verifier.Verify = func(context.Context, string, string) (*cliutil.VerifyPairResult, error) {
		return &cliutil.VerifyPairResult{Valid: true, Operation: &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{
			RunID: "run-1", Scopes: []string{"agent-session:reap"}, ExpiresAt: now.Add(time.Hour).Unix(), Meta: map[string]string{"operation_grant": string(wrongRaw)},
		}}}, nil
	}
	if _, err := verifier.VerifyExactApproval(context.Background(), proof); err == nil {
		t.Fatal("accepted wildcard target under an unrelated policy")
	}
	paired := &identity.OperationGrant{ID: "grant-multipolicy", Approver: "owner", ExpiresAt: now.Add(time.Hour), Rules: []identity.OperationRule{
		{Policy: "agent-management", Operations: []string{proof.Operation}, Targets: []string{"agent-session:*"}},
		{Policy: "agent-session-reaping", Operations: []string{proof.Operation}, Targets: []string{"agent-session:*"}},
	}}
	pairedRaw, _ := json.Marshal(paired)
	verifier.Verify = func(context.Context, string, string) (*cliutil.VerifyPairResult, error) {
		return &cliutil.VerifyPairResult{Valid: true, Operation: &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{
			RunID: "run-1", Scopes: []string{"agent-session:reap"}, ExpiresAt: now.Add(time.Hour).Unix(), Meta: map[string]string{"operation_grant": string(pairedRaw)},
		}}}, nil
	}
	if _, err := verifier.VerifyExactApproval(context.Background(), proof); err != nil {
		t.Fatalf("valid policy-specific rule in a multi-policy wildcard grant refused: %v", err)
	}
	verifier.Resolve = func(context.Context, string, string) (cliutil.AgentCredentials, error) {
		return cliutil.AgentCredentials{LegacyToken: "legacy"}, nil
	}
	if _, err := verifier.VerifyExactApproval(context.Background(), proof); err == nil {
		t.Fatal("accepted a legacy unpaired credential")
	}
}

func TestExactReapGrantUsesCanonicalTargetCoverageWithinNamedPolicyRule(t *testing.T) {
	const target = "agent-session:reap/vrooli-agent-one.scope/proof/0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	now := time.Now().UTC()
	grant := &identity.OperationGrant{ID: "grant", Approver: "owner", Policy: "agent-session-reaping", Operations: []string{"agent-session.reap"}, Targets: []string{target}, ExpiresAt: now.Add(time.Hour)}
	if !grantAllowsReapPolicyTarget(grant, "agent-session.reap", target, now) {
		t.Fatal("canonical exact policy grant was not recognized")
	}
	grant.Policy = "agent-management"
	if grantAllowsReapPolicyTarget(grant, "agent-session.reap", target, now) {
		t.Fatal("accepted exact target from a different policy")
	}
	grant.Policy = "agent-session-reaping"
	grant.Rules = []identity.OperationRule{{Policy: "agent-management", Operations: []string{"agent-session.reap"}, Targets: []string{target}}}
	if grantAllowsReapPolicyTarget(grant, "agent-session.reap", target, now) {
		t.Fatal("accepted exact operation/target cross-paired from a different rule")
	}
	grant.Rules = []identity.OperationRule{{Policy: "agent-session-reaping", Operations: []string{"agent-session.reap"}, Targets: []string{"agent-session:*"}}}
	if !grantAllowsReapPolicyTarget(grant, "agent-session.reap", target, now) {
		t.Fatal("valid policy rule wildcard did not match canonical exact request target")
	}
}
