//go:build linux

package cliutil

import (
	"os"
	"testing"
)

const (
	userManager   = "0::/user.slice/user-1000.slice/user@1000.service/vrooli.slice/"
	launcherCG    = userManager + AgentSlice + "/tmux-spawn-f577a849.scope"
	sessionScope  = userManager + AgentSlice + "/" + agentScopePrefix + "claude-code-e8c6.scope"
	otherSession  = userManager + AgentSlice + "/" + agentScopePrefix + "codex-1234.scope"
	unrelatedUnit = userManager + "app.slice/some-service.service"
)

// Each chain lists cgroups from the caller (first) up to the bound root (last).
func TestCgroupWalkSessionScopeBoundary(t *testing.T) {
	cases := []struct {
		name  string
		root  string
		chain []string
		want  bool
	}{
		{"same cgroup throughout", launcherCG, []string{launcherCG, launcherCG, launcherCG}, true},
		{"agent processes inside the launcher's session scope", launcherCG, []string{sessionScope, sessionScope, sessionScope, launcherCG}, true},
		{"helper in launcher cgroup between scope and root", launcherCG, []string{sessionScope, launcherCG, launcherCG}, true},
		{"unrelated cgroup is still a boundary", launcherCG, []string{unrelatedUnit, launcherCG}, false},
		{"two different session scopes", launcherCG, []string{otherSession, sessionScope, launcherCG}, false},
		{"re-entering a scope above the root cgroup", launcherCG, []string{launcherCG, sessionScope, launcherCG}, false},
		{"root inside a session scope cannot cover another scope", sessionScope, []string{otherSession, sessionScope}, false},
		{"root inside a session scope covers its own scope", sessionScope, []string{sessionScope, sessionScope}, true},
		{"nested cgroup under a session scope is not the scope", launcherCG, []string{sessionScope + "/child", launcherCG}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			walk := cgroupWalk{root: tc.root}
			got := true
			for _, cg := range tc.chain {
				if !walk.step(cg) {
					got = false
					break
				}
			}
			if got != tc.want {
				t.Fatalf("chain %v under root %q: got %v, want %v", tc.chain, tc.root, got, tc.want)
			}
		})
	}
}

func TestIsAgentSessionScope(t *testing.T) {
	for cg, want := range map[string]bool{
		sessionScope:                          true,
		launcherCG:                            false,
		unrelatedUnit:                         false,
		"0::/" + agentScopePrefix + "x.scope": false, // not under the agent slice
		userManager + AgentSlice + "/" + agentScopePrefix + ".scope": false,
		"": false,
	} {
		if got := isAgentSessionScope(cg); got != want {
			t.Errorf("isAgentSessionScope(%q) = %v, want %v", cg, got, want)
		}
	}
}

func TestProcessDescendsFromLiveParent(t *testing.T) {
	parent, err := ObserveProcessBinding(os.Getppid())
	if err != nil {
		t.Skipf("parent binding unavailable: %v", err)
	}
	self, err := ObserveProcessBinding(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if self.Cgroup != parent.Cgroup && !isAgentSessionScope(self.Cgroup) {
		t.Skip("test process and parent are in unrelated cgroups")
	}
	if !ProcessDescendsFrom(os.Getpid(), parent) {
		t.Fatal("test process should descend from its parent")
	}
	stale := parent
	stale.Start = "0"
	if ProcessDescendsFrom(os.Getpid(), stale) {
		t.Fatal("a different root incarnation must not match")
	}
}
