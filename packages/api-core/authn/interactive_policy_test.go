package authn

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/identity"
)

func agentWithoutGrant(scopes ...string) context.Context {
	return identity.WithPrincipal(context.Background(), identity.Principal{
		Kind: identity.ActorAgent, Subject: "run", Owner: "owner", Verified: true, Scopes: scopes,
	})
}

// A refused agent has to be told the exact request that lets the operator
// approve this one action, without losing errors.Is compatibility.
func TestRequireOperationRefusalCarriesTheExactRequest(t *testing.T) {
	t.Setenv("AGENT_MANAGER_AUTHORIZATION_URL", "")
	for _, tc := range []struct {
		name, capability, operation, target, policy string
	}{
		{"task cancellation", "agent-manager:write", "agent.manage", "agent-manager:task/6041722d-aa98-4cd7-b179-b61310c2bf7f", "agent-management"},
		{"team archive", "prompt-manager:write", "team.manage", "prompt-manager:team/desktop-packaging-delivery", "team-management"},
		{"narrowest policy wins", "agent-manager:write", "agent.investigate", "agent-manager:run/abc", "agent-investigation"},
		{"swarm", "swarm-manager:write", "swarm.bookkeeping", "swarm-manager:backlog/execute/x", "swarm-bookkeeping"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := RequireOperation(agentWithoutGrant(tc.capability), tc.capability, tc.operation, tc.target)
			if !errors.Is(err, ErrOperation) {
				t.Fatalf("refusal must still satisfy errors.Is(ErrOperation): %v", err)
			}
			var refusal *OperationRefusal
			if !errors.As(err, &refusal) {
				t.Fatalf("an agent refusal must be an *OperationRefusal, got %T", err)
			}
			if refusal.Operation != tc.operation || refusal.Target != tc.target || len(refusal.Policies) == 0 || refusal.Policies[0] != tc.policy {
				t.Fatalf("refusal = %+v, want operation %s target %s policy %s", refusal, tc.operation, tc.target, tc.policy)
			}
			want := `vrooli agent authorize "$VROOLI_AGENT_SESSION" --policy ` + tc.policy + " --target " + tc.target + ` --reason "<why>"`
			if got := RequestAuthority(err); got != want {
				t.Fatalf("request command = %q, want %q", got, want)
			}
			if !strings.Contains(err.Error(), want) || !strings.HasPrefix(err.Error(), ErrOperation.Error()) {
				t.Fatalf("the error text must lead with the sentinel text and carry the command: %q", err.Error())
			}
		})
	}
}

func TestRefusalCommandNamesTheConfiguredApprovalOriginAndQuotesWildcards(t *testing.T) {
	t.Setenv("AGENT_MANAGER_AUTHORIZATION_URL", "https://agent-manager.example.test/")
	command := NewOperationRefusal("swarm.bookkeeping", "swarm-manager:*").Command()
	want := `vrooli agent authorize "$VROOLI_AGENT_SESSION" --policy swarm-bookkeeping --target 'swarm-manager:*' --approval-base https://agent-manager.example.test --reason "<why>"`
	if command != want {
		t.Fatalf("command = %q, want %q", command, want)
	}
	for _, invalid := range []string{"not a url", "ftp://agent-manager.example.test", "https://user@agent-manager.example.test", "https://agent-manager.example.test/path?x=1"} {
		t.Setenv("AGENT_MANAGER_AUTHORIZATION_URL", invalid)
		if strings.Contains(NewOperationRefusal("swarm.bookkeeping", "swarm-manager:*").Command(), "--approval-base") {
			t.Fatalf("an invalid configured origin %q leaked into the command", invalid)
		}
	}
}

func TestOperatorDecisionsAndUncoveredOperationsNeverOfferARequest(t *testing.T) {
	t.Setenv("AGENT_MANAGER_AUTHORIZATION_URL", "")
	for _, operation := range []string{"run.review", "investigation.apply", "sandbox.sync"} {
		_, err := RequireOperation(agentWithoutGrant("agent-manager:write"), "agent-manager:write", operation, "agent-manager:run/abc")
		if !errors.Is(err, ErrOperation) || RequestAuthority(err) != "" {
			t.Fatalf("%s: a human decision must not offer an agent request: %v", operation, err)
		}
		if !strings.Contains(err.Error(), "records an operator decision") || !strings.Contains(err.Error(), "no agent grant covers it") {
			t.Fatalf("%s: refusal must say it is an operator decision: %q", operation, err.Error())
		}
		if where, ok := OperatorDecision(operation); !ok || !strings.Contains(err.Error(), where) {
			t.Fatalf("%s: refusal must name where the operator answers: %q", operation, err.Error())
		}
	}
	_, err := RequireOperation(agentWithoutGrant("agent-manager:read"), "agent-manager:read", "receipt.read", "agent-manager:effort/x")
	if !errors.Is(err, ErrOperation) || RequestAuthority(err) != "" || !strings.Contains(err.Error(), "no reviewed agent grant policy") {
		t.Fatalf("an operation no policy covers must say so instead of inventing a request: %v", err)
	}
	if got := NewOperationRefusal("agent.manage", "").Command(); got != "" {
		t.Fatalf("a refusal with no target cannot carry a ready-to-run command: %q", got)
	}
}

// Only an agent can run `vrooli agent authorize`; a service or a human gets
// the established errors unchanged.
func TestNonAgentRefusalsStayPlainSentinels(t *testing.T) {
	service := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorService, Subject: "svc", Verified: true, Scopes: []string{"agent-manager:write"}})
	if _, err := RequireOperation(service, "agent-manager:write", "agent.manage", "agent-manager:run/x"); err != ErrOperation {
		t.Fatalf("service refusal must remain the plain sentinel, got %v", err)
	}
	human := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Verified: true, ExpiresAt: time.Now().Add(time.Hour)})
	if _, err := RequireOperation(human, "agent-manager:write", "agent.manage", "agent-manager:run/x"); err != ErrCapability {
		t.Fatalf("human without the capability must stay ErrCapability, got %v", err)
	}
}

type fixedOperationAuthority struct{ err error }

func (a fixedOperationAuthority) AuthorizeOperation(context.Context, identity.Principal, string, string, string) error {
	return a.err
}

func TestDomainHeldAuthorityRefusalForAnAgentCarriesTheRequest(t *testing.T) {
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorAgent, Subject: "run", Verified: true})
	ctx = WithRequestOperationAuthority(ctx, fixedOperationAuthority{err: ErrOperation})
	_, err := RequireOperation(ctx, "agent-manager:write", "run.create", "agent-manager:*")
	if !errors.Is(err, ErrOperation) || !strings.Contains(RequestAuthority(err), "--policy run-creation") {
		t.Fatalf("a domain-held refusal for an agent must carry the request: %v", err)
	}
	other := errors.New("authority store unavailable")
	ctx = WithRequestOperationAuthority(ctx, fixedOperationAuthority{err: other})
	if _, err := RequireOperation(ctx, "agent-manager:write", "run.create", "agent-manager:*"); err != other {
		t.Fatalf("an unrelated authority error must pass through unchanged: %v", err)
	}
}

func TestCatalogueIsConsistentAndReverseLookupMatches(t *testing.T) {
	names := InteractivePolicyNames()
	if len(names) == 0 || len(slices.Compact(slices.Sorted(slices.Values(names)))) != len(names) {
		t.Fatalf("catalogue names must be unique and non-empty: %v", names)
	}
	for _, name := range names {
		policy, ok := InteractivePolicy(name)
		if !ok || len(policy.Operations) == 0 || len(policy.Scopes) == 0 || len(policy.Domains) == 0 {
			t.Fatalf("policy %s is incomplete: %+v", name, policy)
		}
		for _, operation := range policy.Operations {
			if !slices.Contains(PoliciesForOperation(operation), name) {
				t.Fatalf("operation %s is granted by %s but PoliciesForOperation omits it", operation, name)
			}
		}
	}
	// A returned policy is a copy: mutating it must not rewrite the catalogue.
	policy, _ := InteractivePolicy("team-management")
	policy.Operations[0] = "tampered"
	if again, _ := InteractivePolicy("team-management"); again.Operations[0] != "team.manage" {
		t.Fatalf("InteractivePolicy leaked its backing slice: %+v", again)
	}
	if _, ok := InteractivePolicy("no-such-policy"); ok {
		t.Fatal("unknown policy resolved")
	}
	for operation := range operatorDecisions {
		if got := PoliciesForOperation(operation); len(got) != 0 {
			t.Fatalf("operator decision %s must not be grantable, but %v grant it", operation, got)
		}
	}
}

// docs/concepts/AUTHORIZATION.md is where an operator reads what each policy
// covers. It must list exactly the catalogue's policies and operations, so
// neither a code-only policy nor a stale doc row can drift unnoticed.
func TestAuthorizationDocPolicyTableMatchesCatalogue(t *testing.T) {
	doc, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "concepts", "AUTHORIZATION.md"))
	if err != nil {
		t.Skipf("authorization doc is not reachable from this checkout: %v", err)
	}
	lines := strings.Split(string(doc), "\n")
	start := slices.IndexFunc(lines, func(line string) bool { return strings.HasPrefix(line, "| Policy | Operations |") })
	if start < 0 {
		t.Fatal("the reviewed-policy table is missing from AUTHORIZATION.md")
	}
	token := regexp.MustCompile("`([a-z-]+)`")
	operation := regexp.MustCompile("`([a-z]+\\.[a-z]+)`")
	documented := map[string][]string{}
	for _, line := range lines[start+2:] {
		if !strings.HasPrefix(line, "|") {
			break
		}
		cells := strings.Split(line, "|")
		name := token.FindStringSubmatch(cells[1])
		if name == nil {
			t.Fatalf("unreadable table row: %q", line)
		}
		for _, match := range operation.FindAllStringSubmatch(cells[2], -1) {
			documented[name[1]] = append(documented[name[1]], match[1])
		}
	}
	for _, name := range InteractivePolicyNames() {
		policy, _ := InteractivePolicy(name)
		got, ok := documented[name]
		if !ok {
			t.Errorf("policy %s is in the catalogue but not in AUTHORIZATION.md", name)
			continue
		}
		slices.Sort(got)
		want := slices.Sorted(slices.Values(policy.Operations))
		if !slices.Equal(got, want) {
			t.Errorf("policy %s: AUTHORIZATION.md lists operations %v, the catalogue grants %v", name, got, want)
		}
		delete(documented, name)
	}
	for name := range documented {
		t.Errorf("AUTHORIZATION.md documents policy %s, which the catalogue does not have", name)
	}
}
