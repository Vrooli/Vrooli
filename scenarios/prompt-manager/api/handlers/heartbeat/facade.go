// Package heartbeat exposes the heartbeat transport boundary.
package heartbeat

import domain "prompt-manager/internal/heartbeat"

type (
	FiniteOwnerStartup         = domain.FiniteOwnerStartup
	PreparedFiniteInstallation = domain.PreparedFiniteInstallation
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
	PrepareFiniteOwnerStartup = domain.PrepareFiniteOwnerStartup
	NewSchedulerWithFinite    = domain.NewSchedulerWithFinite
	NewAgentManagerClient     = domain.NewAgentManagerClient
	NewExecutor               = domain.NewExecutor
	NewHandlers               = domain.NewHandlers
	NewHeartbeatControlStore  = domain.NewHeartbeatControlStore
	NewPromptBuilder          = domain.NewPromptBuilder
	NewRunRegistry            = domain.NewRunRegistry
	NewScheduler              = domain.NewScheduler
	NewTeamExecutionStore     = domain.NewTeamExecutionStore
)
