package server

import (
	"strings"
	"time"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

type AgentRunResult struct {
	Success          bool
	Output           string
	Error            string
	RunID            string
	SessionID        string
	RunnerType       string
	ProfileKey       string
	ExecutionTime    time.Duration
	TokensUsed       int32
	CostEstimate     float64
	RateLimited      bool
	Timeout          bool
	MaxTurnsExceeded bool
}

func buildAgentRunResult(run *domainpb.Run, startTime time.Time, runnerType, profileKey string) *AgentRunResult {
	result := &AgentRunResult{
		RunID:      run.GetId(),
		SessionID:  run.GetSessionId(),
		RunnerType: runnerType,
		ProfileKey: profileKey,
		Success:    run.GetStatus() == domainpb.RunStatus_RUN_STATUS_COMPLETE,
	}

	if run.Summary != nil {
		result.Output = run.Summary.GetDescription()
		result.TokensUsed = run.Summary.GetTokensUsed()
		result.CostEstimate = run.Summary.GetCostEstimate()
	}

	if run.ErrorMsg != "" {
		result.Error = run.ErrorMsg
		lower := strings.ToLower(run.ErrorMsg)
		result.RateLimited = strings.Contains(lower, "rate limit") || strings.Contains(lower, "429")
		result.Timeout = strings.Contains(lower, "timeout")
		result.MaxTurnsExceeded = strings.Contains(lower, "max turns") || strings.Contains(lower, "max_turns")
	}

	if run.StartedAt != nil && run.EndedAt != nil {
		result.ExecutionTime = run.EndedAt.AsTime().Sub(run.StartedAt.AsTime())
	} else {
		result.ExecutionTime = time.Since(startTime)
	}

	return result
}
