package store

import (
	"prompt-manager/teamconfig"
	"prompt-manager/teamcontract"
)

func newIndependentTestTeam(id, displayName string) *Team {
	return &Team{
		ID:          id,
		DisplayName: displayName,
		Enabled:     true,
		Runtime: teamconfig.Runtime{
			Mode: teamconfig.RuntimeModeMultiProcess,
		},
		Coordination: teamconfig.Coordination{
			Pattern:       teamconfig.CoordinationPatternIndependent,
			ReportingMode: teamconfig.ReportingModeNone,
			MessagingMode: teamconfig.MessagingModeDisabled,
			Capabilities: teamconfig.Capabilities{
				ShowOrgContext:           false,
				InjectInbox:              false,
				AllowPeerTriggers:        false,
				ShowTaskBoardGuidance:    true,
				ShowDecisionLogGuidance:  true,
				ShowKnowledgeLogGuidance: true,
				RequireHandoff:           true,
			},
		},
		Execution: teamconfig.Execution{
			QueuePolicy:       teamconfig.QueuePolicyBoundedParallel,
			MaxConcurrentRuns: 2,
		},
		DecisionMode:      teamconfig.DecisionModeYolo,
		OperatingContract: teamcontract.Minimal(teamconfig.DecisionModeYolo, "agent-1", "agent-2", "lead", "dev-1"),
	}
}
