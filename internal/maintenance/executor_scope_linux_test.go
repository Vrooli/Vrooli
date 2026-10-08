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
