// Package heartbeat exposes the heartbeat transport boundary.
package heartbeat

import domain "prompt-manager/internal/heartbeat"

type (
	AgentManagerClient         = domain.AgentManagerClient
	Executor                   = domain.Executor
	HandlersDeps               = domain.HandlersDeps
	MemberflowContractFindings = domain.MemberflowContractFindings
	PromptBuildRequest         = domain.PromptBuildRequest
	TeamObjective              = domain.TeamObjective
	TeamObjectiveContext       = domain.TeamObjectiveContext
	TeamObjectiveProvider      = domain.TeamObjectiveProvider
)

var (
	NewAgentManagerClient    = domain.NewAgentManagerClient
	NewExecutor              = domain.NewExecutor
	NewHandlers              = domain.NewHandlers
	NewHeartbeatControlStore = domain.NewHeartbeatControlStore
	NewPromptBuilder         = domain.NewPromptBuilder
	NewRunRegistry           = domain.NewRunRegistry
	NewScheduler             = domain.NewScheduler
	NewTeamExecutionStore    = domain.NewTeamExecutionStore
	WireStandingSupervisor   = domain.WireStandingSupervisor
)
