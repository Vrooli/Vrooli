package teams

import (
	"prompt-manager/store"
	"prompt-manager/teamconfig"
)

func newIndependentTestTeam(id, displayName string) *store.Team {
	return &store.Team{
		ID:          id,
		DisplayName: displayName,
		Enabled:     false,
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
		DecisionMode: teamconfig.DecisionModeYolo,
	}
}

func newLeaderLedSingleProcessTestTeam(id, displayName, leadAgentID string) *store.Team {
	team := newIndependentTestTeam(id, displayName)
	team.Runtime = teamconfig.Runtime{Mode: teamconfig.RuntimeModeSingleProcess}
	team.Coordination = teamconfig.Coordination{
		Pattern:       teamconfig.CoordinationPatternLeaderLed,
		LeadAgentID:   leadAgentID,
		ReportingMode: teamconfig.ReportingModeLeader,
		MessagingMode: teamconfig.MessagingModeInSession,
		Capabilities: teamconfig.Capabilities{
			ShowOrgContext:           true,
			InjectInbox:              false,
			AllowPeerTriggers:        false,
			ShowTaskBoardGuidance:    true,
			ShowDecisionLogGuidance:  true,
			ShowKnowledgeLogGuidance: true,
			RequireHandoff:           true,
		},
	}
	team.Execution = teamconfig.Execution{
		QueuePolicy:       teamconfig.QueuePolicySerialized,
		MaxConcurrentRuns: 1,
	}
	return team
}
