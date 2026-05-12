package clipolicy

import "testing"

func TestClassifyAgentCommandAllowsOperatorContext(t *testing.T) {
	decision := ClassifyAgentCommand([]string{"vrooli", "cleanup", "orphans"}, nil)
	if !decision.Allowed {
		t.Fatalf("operator command denied: %+v", decision)
	}
}

func TestClassifyAgentCommandBlocksDestructiveMaintenance(t *testing.T) {
	env := []string{"VROOLI_SANDBOX_ID=sbx-1"}
	tests := [][]string{
		{"vrooli", "cleanup", "orphans"},
		{"vrooli", "cleanup", "locks"},
		{"vrooli", "orphans", "kill"},
		{"vrooli", "locks", "clean"},
		{"/usr/bin/env", "vrooli", "cleanup", "orphans"},
		{"/usr/bin/env", "PATH=/usr/bin", "vrooli", "cleanup", "orphans"},
		{"bash", "-lc", "vrooli cleanup orphans &> /tmp/o.txt; tail -10 /tmp/o.txt"},
		{"sh", "-c", "vrooli locks clean"},
		{"vrooli", "stop"},
		{"vrooli", "scenario", "stop-all"},
		{"vrooli", "resource", "stop-all"},
	}
	for _, argv := range tests {
		t.Run(argv[0], func(t *testing.T) {
			decision := ClassifyAgentCommand(argv, env)
			if decision.Allowed {
				t.Fatalf("ClassifyAgentCommand(%v) allowed destructive command", argv)
			}
			if decision.Code != CodeDestructiveVrooliMaintenanceBlocked {
				t.Fatalf("code = %q, want %q", decision.Code, CodeDestructiveVrooliMaintenanceBlocked)
			}
		})
	}
}

func TestClassifyAgentCommandAllowsReadOnlyCommands(t *testing.T) {
	env := []string{"VROOLI_SANDBOX_MERGED=/workspace"}
	tests := [][]string{
		{"vrooli", "help"},
		{"vrooli", "scenario", "status", "agent-manager"},
		{"vrooli", "scenario", "port", "agent-manager", "API_PORT"},
		{"vrooli", "scenario", "stop", "prompt-manager"},
		{"vrooli", "cleanup", "orphans", "--dry-run"},
		{"vrooli", "orphans", "kill", "--dry-run"},
		{"swarm-manager", "sessions", "get", "--help"},
	}
	for _, argv := range tests {
		t.Run(argv[0], func(t *testing.T) {
			if decision := ClassifyAgentCommand(argv, env); !decision.Allowed {
				t.Fatalf("ClassifyAgentCommand(%v) denied safe command: %+v", argv, decision)
			}
		})
	}
}
