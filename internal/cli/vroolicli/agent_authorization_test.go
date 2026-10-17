package vroolicli

import (
	"testing"

	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func TestAuthorizationSelectsLiveLeaseWithoutAcceptingHarnessAssertions(t *testing.T) {
	leases := []scenarioruntime.EditorLease{{SessionID: "current", PID: 10, Status: scenarioruntime.EditorLeaseActive, Owner: scenarioruntime.EditorLeaseOwner{Ref: "run"}}, {SessionID: "old", PID: 11, Status: scenarioruntime.EditorLeaseStopped}}
	lease, err := selectAuthorizationLease(leases, "run")
	if err != nil || lease.SessionID != "current" {
		t.Fatal("run did not resolve through its live host lease")
	}
	for _, selector := range []string{"old", "made-up-harness"} {
		if _, err := selectAuthorizationLease(leases, selector); err == nil {
			t.Fatal("unverified session selector accepted")
		}
	}
	leases = append(leases, scenarioruntime.EditorLease{SessionID: "second", Status: scenarioruntime.EditorLeaseActive, Owner: scenarioruntime.EditorLeaseOwner{Ref: "run"}})
	if _, err := selectAuthorizationLease(leases, "run"); err == nil {
		t.Fatal("ambiguous run selector accepted")
	}
}

func TestAuthorizationURLsNeverCarryCredentials(t *testing.T) {
	url, err := authorizationDecisionURL("https://agent-manager.example.test", "request", "approve")
	if err != nil || url != "https://agent-manager.example.test/api/v1/authorizations/request/approve" {
		t.Fatalf("approval URL=%q err=%v", url, err)
	}
	for _, base := range []string{"https://user:secret@example.test", "https://example.test?token=secret", "file:///tmp/approval", "https://example.test#token"} {
		if _, err := authorizationDecisionURL(base, "request", "approve"); err == nil {
			t.Fatal("credential-bearing or invalid approval base accepted")
		}
	}
}

func TestAuthorizationSelectorAcceptsScopeAndPID(t *testing.T) {
	leases := []scenarioruntime.EditorLease{{SessionID: "e03eeb2e", PID: 1154310, Scope: "vrooli-agent-claude-code-954a", Status: scenarioruntime.EditorLeaseActive}}
	for _, selector := range []string{"e03eeb2e", "vrooli-agent-claude-code-954a", "1154310"} {
		if lease, err := selectAuthorizationLease(leases, selector); err != nil || lease.SessionID != "e03eeb2e" {
			t.Fatalf("selector %q did not resolve the live lease: %v", selector, err)
		}
	}
}

func TestAuthorizationRulesPairTargetsWithTheirPolicy(t *testing.T) {
	parse := func(args ...string) ([]authorizationRule, error) {
		r := &authorizationRules{}
		p, tg := authorizationPolicyFlag{r}, authorizationTargetFlag{r}
		for i := 0; i < len(args); i += 2 {
			flag := map[string]interface{ Set(string) error }{"--policy": p, "--target": tg}[args[i]]
			if err := flag.Set(args[i+1]); err != nil {
				return nil, err
			}
		}
		return r.resolve()
	}
	rules, err := parse("--policy", "agent-management", "--target", "agent-manager:run/r1", "--policy", "effort-metadata", "--target", "agent-manager:effort/bas")
	if err != nil || len(rules) != 2 || rules[0].Targets[0] != "agent-manager:run/r1" || rules[1].Targets[0] != "agent-manager:effort/bas" {
		t.Fatalf("rules=%+v err=%v", rules, err)
	}
	if rules, err := parse("--target", "agent-manager:run/r1", "--policy", "agent-management"); err != nil || len(rules) != 1 || len(rules[0].Targets) != 1 {
		t.Fatalf("single-policy target order changed meaning: %+v %v", rules, err)
	}
	if rules, err := parse("--target", "swarm-manager:*"); err != nil || rules[0].Policy != "swarm-bookkeeping" {
		t.Fatalf("default policy lost: %+v %v", rules, err)
	}
	if _, err := parse("--target", "x:y", "--policy", "a", "--policy", "b", "--target", "x:z"); err == nil {
		t.Fatal("an unplaced target was guessed into a policy")
	}
	if _, err := parse("--policy", "a", "--target", "x:y", "--policy", "b"); err == nil {
		t.Fatal("a policy without targets was accepted")
	}
}
