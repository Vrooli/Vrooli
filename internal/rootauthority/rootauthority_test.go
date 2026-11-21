package rootauthority

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/cliutil"
)

func grantToken(t *testing.T, grant *identity.OperationGrant) string {
	t.Helper()
	raw, err := json.Marshal(grant)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func verifierFor(t *testing.T, now time.Time, scopes []string, token string) Verifier {
	t.Helper()
	return Verifier{
		Now: func() time.Time { return now },
		Resolve: func(_ context.Context, operation, target string) (cliutil.AgentCredentials, error) {
			if operation != Operation || !strings.HasPrefix(target, Capability+"/") {
				t.Fatalf("credential lookup = %q %q", operation, target)
			}
			return cliutil.AgentCredentials{BaselineToken: "baseline", OperationToken: "operation"}, nil
		},
		VerifyPair: func(_ context.Context, baseline, operation string) (*cliutil.VerifyPairResult, error) {
			if baseline != "baseline" || operation != "operation" {
				t.Fatalf("verification pair = %q %q", baseline, operation)
			}
			return &cliutil.VerifyPairResult{Valid: true, Operation: &cliutil.VerifyResult{
				Valid: true,
				Claims: &cliutil.VerifiedClaims{
					RunID: "run-1", Scopes: scopes, ExpiresAt: now.Add(time.Hour).Unix(),
					Meta: map[string]string{"operation_grant": token},
				},
			}}, nil
		},
	}
}

func TestVerifyAdmitsExactGrant(t *testing.T) {
	now := time.Now().UTC()
	target := Target("host-a", "/home/owner/.vrooli")
	grant := &identity.OperationGrant{
		ID: "grant-1", Policy: Policy, Approver: "owner-1",
		Operations: []string{Operation}, Targets: []string{target}, ExpiresAt: now.Add(time.Hour),
	}
	decision, err := verifierFor(t, now, []string{Capability}, grantToken(t, grant)).Verify(context.Background(), target)
	if err != nil {
		t.Fatalf("exact grant refused: %v", err)
	}
	if decision.GrantID != grant.ID || decision.ApprovedBy != "owner-1" || decision.RunID != "run-1" || decision.Target != target {
		t.Fatalf("decision is not bound to the verified grant: %+v", decision)
	}
	if !decision.ExpiresAt.Equal(time.Unix(now.Add(time.Hour).Unix(), 0).UTC()) {
		t.Fatalf("expires_at = %s, want the grant expiry", decision.ExpiresAt)
	}
}

func TestVerifyAdmitsBoundedWildcardWithinThePolicyRule(t *testing.T) {
	now := time.Now().UTC()
	grant := &identity.OperationGrant{
		ID: "grant-wild", Approver: "owner", ExpiresAt: now.Add(time.Hour),
		Rules: []identity.OperationRule{
			{Policy: Policy, Operations: []string{Operation}, Targets: []string{Capability + "/*"}},
		},
	}
	if _, err := verifierFor(t, now, []string{Capability}, grantToken(t, grant)).Verify(context.Background(), Target("host-a", "/home/owner/.vrooli")); err != nil {
		t.Fatalf("approved wildcard did not cover the request target: %v", err)
	}
}

func TestVerifyRefusesUnrelatedPolicyEvenWithMatchingTarget(t *testing.T) {
	now := time.Now().UTC()
	target := Target("host-a", "/home/owner/.vrooli")
	grant := &identity.OperationGrant{
		ID: "grant-other", Policy: "agent-management", Approver: "owner",
		Operations: []string{Operation}, Targets: []string{target}, ExpiresAt: now.Add(time.Hour),
	}
	_, err := verifierFor(t, now, []string{Capability}, grantToken(t, grant)).Verify(context.Background(), target)
	if err == nil {
		t.Fatal("accepted a grant issued under an unrelated policy")
	}
	if !IsRefusal(err) {
		t.Fatalf("expected an authority refusal, got %v", err)
	}
}

func TestVerifyRefusesMissingCapabilityScope(t *testing.T) {
	now := time.Now().UTC()
	target := Target("host-a", "/home/owner/.vrooli")
	grant := &identity.OperationGrant{
		ID: "grant-1", Policy: Policy, Approver: "owner",
		Operations: []string{Operation}, Targets: []string{target}, ExpiresAt: now.Add(time.Hour),
	}
	if _, err := verifierFor(t, now, nil, grantToken(t, grant)).Verify(context.Background(), target); err == nil {
		t.Fatal("accepted a grant without the capability scope")
	}
}

func TestVerifyRefusesExpiredAndUnpairedCredentials(t *testing.T) {
	now := time.Now().UTC()
	target := Target("host-a", "/home/owner/.vrooli")
	expired := &identity.OperationGrant{
		ID: "grant-old", Policy: Policy, Approver: "owner",
		Operations: []string{Operation}, Targets: []string{target}, ExpiresAt: now.Add(-time.Minute),
	}
	if _, err := verifierFor(t, now, []string{Capability}, grantToken(t, expired)).Verify(context.Background(), target); err == nil {
		t.Fatal("accepted an expired grant")
	}
	unpaired := verifierFor(t, now, []string{Capability}, "")
	unpaired.Resolve = func(context.Context, string, string) (cliutil.AgentCredentials, error) {
		return cliutil.AgentCredentials{LegacyToken: "legacy"}, nil
	}
	if _, err := unpaired.Verify(context.Background(), target); err == nil {
		t.Fatal("accepted a legacy unpaired credential")
	}
}

func TestVerifyRefusesAnotherHostOrHome(t *testing.T) {
	now := time.Now().UTC()
	target := Target("host-b", "/home/owner/.vrooli")
	grant := &identity.OperationGrant{
		ID: "grant-1", Policy: Policy, Approver: "owner",
		Operations: []string{Operation}, Targets: []string{Target("host-a", "/home/owner/.vrooli")}, ExpiresAt: now.Add(time.Hour),
	}
	if _, err := verifierFor(t, now, []string{Capability}, grantToken(t, grant)).Verify(context.Background(), target); err == nil {
		t.Fatal("accepted a grant for a different host")
	}
}

func TestRefusalCarriesTheRequestCommand(t *testing.T) {
	refusal := &Refusal{Target: Target("host-a", "/home/owner/.vrooli"), Reason: "no grant"}
	if !errors.Is(refusal, refusal) || !IsRefusal(refusal) {
		t.Fatal("refusal is not recognizable")
	}
	if !strings.Contains(refusal.Error(), "vrooli agent authorize") || !strings.Contains(refusal.Error(), Policy) {
		t.Fatalf("refusal does not name the request command: %s", refusal.Error())
	}
}

func TestTargetIsExactAndCanonical(t *testing.T) {
	if got, want := Target(" host-a ", " /home/owner/.vrooli "), "root-state:transition/host-a/home/owner/.vrooli"; got != want {
		t.Fatalf("target = %q, want %q", got, want)
	}
	if ScopeSatisfied(nil) || !ScopeSatisfied([]string{Capability}) || !ScopeSatisfied([]string{"root-state:*"}) {
		t.Fatal("capability scope resolution is wrong")
	}
}
