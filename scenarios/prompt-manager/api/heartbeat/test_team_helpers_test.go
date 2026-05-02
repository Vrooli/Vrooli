package heartbeat

import (
	"prompt-manager/internal/testutil/fixtures"
	"prompt-manager/store"
)

func newIndependentTestTeam(id, displayName string) *store.Team {
	return fixtures.IndependentTeam(id, displayName, fixtures.WithContractAgents("agent-1", "agent-2", "lead", "dev-1", "director", "strategist"))
}

func newLeaderLedSingleProcessTestTeam(id, displayName, leadAgentID string) *store.Team {
	return fixtures.LeaderLedSingleProcessTeam(id, displayName, leadAgentID, fixtures.WithContractAgents("agent-1", "agent-2", "lead", "dev-1", "director", "strategist"))
}
