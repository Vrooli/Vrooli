package fixtures

import (
	"encoding/json"
	"os"
	"path/filepath"

	"prompt-manager/store"
	"prompt-manager/teamconfig"
	"prompt-manager/teamcontract"
)

// WriteTeam writes team.json from the real store.Team type. Tests using this
// helper fail at compile time when the persisted contract changes instead of
// silently drifting through hand-written JSON fixtures.
func WriteTeam(root string, team *store.Team) (string, error) {
	if team == nil {
		return "", os.ErrInvalid
	}
	return WriteJSON(root, filepath.Join("teams", team.ID, "team.json"), team)
}

// WriteJSON persists a typed fixture at a root-relative path. It keeps the
// common fixture substrate independent of domain packages while preserving
// compile-time checking at every typed call site.
func WriteJSON(root, relativePath string, value any) (string, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}
	path := filepath.Join(root, filepath.FromSlash(relativePath))
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// TeamOption customizes a team fixture while preserving a shared base
// contract for prompt-manager team behavior tests.
type TeamOption func(*store.Team)

func WithEnabled(enabled bool) TeamOption {
	return func(team *store.Team) {
		team.Enabled = enabled
		team.EnabledSet = true
	}
}

func WithContractAgents(agentIDs ...string) TeamOption {
	return func(team *store.Team) {
		team.OperatingContract = teamcontract.Minimal(team.DecisionMode, agentIDs...)
	}
}

func WithDecisionMode(mode string) TeamOption {
	return func(team *store.Team) {
		team.DecisionMode = mode
		team.OperatingContract = teamcontract.Minimal(mode, "agent-1", "agent-2", "lead", "dev-1")
	}
}

func WithLeadAgent(leadAgentID string) TeamOption {
	return func(team *store.Team) {
		team.Coordination.LeadAgentID = leadAgentID
	}
}

// IndependentTeam returns a multi-process independent team with yolo
// decision mode and a minimal operating contract.
func IndependentTeam(id, displayName string, opts ...TeamOption) *store.Team {
	team := &store.Team{
		ID:          id,
		DisplayName: displayName,
		Enabled:     true,
		EnabledSet:  true,
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
	for _, opt := range opts {
		opt(team)
	}
	return team
}

// LeaderLedSingleProcessTeam returns a serialized single-process team
// where one lead agent coordinates in-session work.
func LeaderLedSingleProcessTeam(id, displayName, leadAgentID string, opts ...TeamOption) *store.Team {
	team := IndependentTeam(id, displayName, opts...)
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
	for _, opt := range opts {
		opt(team)
	}
	return team
}
