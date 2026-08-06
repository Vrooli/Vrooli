package store

import (
	"prompt-manager/teamconfig"
	"prompt-manager/teamcontract"
)

// newIndependentTestTeam duplicates the builder in
// internal/testutil/fixtures on purpose. All 16 test files in this package are
// internal `package store`, so importing fixtures would be an import cycle:
// fixtures returns *store.Team and therefore imports store. A shared builder
// cannot live in a leaf package that must not import the package it builds for.
//
// This is the one deliberate exception to the shared-substrate rule in
// api/TESTING_GUIDE.md. Do not "consolidate" it without first moving these
// tests to `package store_test`, which is a larger change with its own tradeoff:
// it would lose access to unexported store internals these tests assert on.
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
