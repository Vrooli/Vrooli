// Package heartbeat exposes the heartbeat transport boundary.
package heartbeat

import domain "prompt-manager/internal/heartbeat"

type (
	Executor                   = domain.Executor
	HandlersDeps               = domain.HandlersDeps
	MemberflowContractFindings = domain.MemberflowContractFindings
	PromptBuildRequest         = domain.PromptBuildRequest
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
)
