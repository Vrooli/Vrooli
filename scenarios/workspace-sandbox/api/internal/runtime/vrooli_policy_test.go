package runtime

import "testing"

func TestEvaluateVrooliCommandPolicyBlocksDestructiveMaintenance(t *testing.T) {
	tests := []struct {
		command string
		args    []string
	}{
		{command: "vrooli", args: []string{"cleanup", "orphans"}},
		{command: "vrooli", args: []string{"cleanup", "locks"}},
		{command: "vrooli", args: []string{"orphans", "kill"}},
		{command: "/usr/bin/env", args: []string{"vrooli", "cleanup", "orphans"}},
		{command: "/usr/bin/env", args: []string{"PATH=/usr/bin", "vrooli", "cleanup", "orphans"}},
		{command: "bash", args: []string{"-lc", "vrooli cleanup orphans &> /tmp/o.txt; tail -10 /tmp/o.txt"}},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			decision := EvaluateVrooliCommandPolicy(tt.command, tt.args)
			if decision.Allowed {
				t.Fatalf("EvaluateVrooliCommandPolicy(%q, %v) allowed destructive command", tt.command, tt.args)
			}
			if decision.Code != VrooliPolicyDestructiveMaintenanceBlocked {
				t.Fatalf("code = %q, want %q", decision.Code, VrooliPolicyDestructiveMaintenanceBlocked)
			}
		})
	}
}

func TestEvaluateVrooliCommandPolicyAllowsReadOnlyCommands(t *testing.T) {
	tests := []struct {
		command string
		args    []string
	}{
		{command: "vrooli", args: []string{"help"}},
		{command: "vrooli", args: []string{"scenario", "status", "agent-manager"}},
		{command: "vrooli", args: []string{"cleanup", "orphans", "--dry-run"}},
		{command: "swarm-manager", args: []string{"sessions", "get", "--help"}},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			if decision := EvaluateVrooliCommandPolicy(tt.command, tt.args); !decision.Allowed {
				t.Fatalf("EvaluateVrooliCommandPolicy(%q, %v) denied safe command: %+v", tt.command, tt.args, decision)
			}
		})
	}
}
