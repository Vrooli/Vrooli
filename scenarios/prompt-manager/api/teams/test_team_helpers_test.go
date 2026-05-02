package teams

import (
	"prompt-manager/internal/testutil/fixtures"
	"prompt-manager/store"
)

func newIndependentTestTeam(id, displayName string) *store.Team {
	return fixtures.IndependentTeam(
		id,
		displayName,
		fixtures.WithEnabled(false),
		fixtures.WithContractAgents("agent-1", "agent-2", "lead", "dev-1", "quality-auditor"),
	)
}

func newLeaderLedSingleProcessTestTeam(id, displayName, leadAgentID string) *store.Team {
	return fixtures.LeaderLedSingleProcessTeam(
		id,
		displayName,
		leadAgentID,
		fixtures.WithEnabled(false),
		fixtures.WithContractAgents("agent-1", "agent-2", "lead", "dev-1", "quality-auditor"),
	)
}
