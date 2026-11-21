//go:build linux

package maintenance

import "testing"

func TestExecutorScopeForeignServiceProofIsNotNameBased(t *testing.T) {
	for _, tc := range []struct {
		scope   string
		outside bool
	}{
		{"/user.slice/user-1000.slice/user@1000.service/app.slice/gcr-ssh-agent.service", true},
		{"/user.slice/user-1000.slice/user@1000.service/vrooli.slice/vrooli-services.slice/vrooli-service-agent-manager.scope", false},
		{"/user.slice/user-1000.slice/user@1000.service/vrooli-agents.slice/agent.scope", false},
		{"/user.slice/user-1000.slice/user@1000.service/app.slice/terminal.scope", false},
		{"/user.slice/user-1000.slice/user@1000.service", false},
		{"/user.slice/user-1000.slice/user@1000.service/init.scope", true},
		{"", false},
	} {
		if got := outsideManagedExecutionScope(processTableEntry{Cgroup: tc.scope, Command: "ssh-agent"}); got != tc.outside {
			t.Fatalf("scope=%q outside=%t", tc.scope, got)
		}
	}
}

func TestExecutorHomeIdentityScopeExcludesOnlyExactDockerWorkloadCgroup(t *testing.T) {
	docker := "/workload.slice/docker-5bb775c5715aebbeeea2840da2d92c68dd9f58bd2e1a61b238d110d63c75a924.scope"
	for _, tc := range []struct {
		name    string
		cgroup  string
		inScope bool
	}{
		{name: "exact docker workload is non-native", cgroup: docker, inScope: false},
		{name: "nested docker cgroup is not an exact owner proof", cgroup: docker + "/child.scope", inScope: true},
		{name: "malformed docker id is not an owner proof", cgroup: "/workload.slice/docker-short.scope", inScope: true},
		{name: "managed agent scope remains in scope", cgroup: "/user.slice/user-1000.slice/user@1000.service/vrooli.slice/vrooli-agents.slice/agent.scope", inScope: true},
		{name: "missing cgroup remains in scope", cgroup: "", inScope: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := executorHomeIdentityInScope(processTableEntry{PID: 77, Cgroup: tc.cgroup}); got != tc.inScope {
				t.Fatalf("home identity in-scope=%t, want %t for %q", got, tc.inScope, tc.cgroup)
			}
		})
	}
}
