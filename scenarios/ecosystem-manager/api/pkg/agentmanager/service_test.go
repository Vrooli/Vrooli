package agentmanager

import (
	"testing"
	"time"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestBuildExecuteResult_TerminalStatus(t *testing.T) {
	svc := &AgentService{}

	tests := []struct {
		name        string
		status      domainpb.RunStatus
		wantSuccess bool
	}{
		{
			name:        "COMPLETE maps to success=true",
			status:      domainpb.RunStatus_RUN_STATUS_COMPLETE,
			wantSuccess: true,
		},
		{
			name:        "FAILED maps to success=false",
			status:      domainpb.RunStatus_RUN_STATUS_FAILED,
			wantSuccess: false,
		},
		{
			name:        "CANCELLED maps to success=false",
			status:      domainpb.RunStatus_RUN_STATUS_CANCELLED,
			wantSuccess: false,
		},
		{
			name:        "RUNNING maps to success=false",
			status:      domainpb.RunStatus_RUN_STATUS_RUNNING,
			wantSuccess: false,
		},
		{
			name:        "PENDING maps to success=false",
			status:      domainpb.RunStatus_RUN_STATUS_PENDING,
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := &domainpb.Run{
				Id:     "run-123",
				Status: tt.status,
			}
			result := svc.buildExecuteResult(run)
			if result.Success != tt.wantSuccess {
				t.Errorf("buildExecuteResult() Success = %v, want %v", result.Success, tt.wantSuccess)
			}
			if result.RunID != "run-123" {
				t.Errorf("buildExecuteResult() RunID = %q, want %q", result.RunID, "run-123")
			}
		})
	}
}

func TestBuildExecuteResult_ErrorHeuristics(t *testing.T) {
	svc := &AgentService{}

	tests := []struct {
		name            string
		errorMsg        string
		wantRateLimited bool
		wantTimeout     bool
		wantMaxTurns    bool
	}{
		{
			name:            "rate limit detected from message",
			errorMsg:        "rate limit exceeded, try again later",
			wantRateLimited: true,
		},
		{
			name:            "429 status detected",
			errorMsg:        "HTTP 429: too many requests",
			wantRateLimited: true,
		},
		{
			name:        "timeout detected",
			errorMsg:    "execution timeout after 30 minutes",
			wantTimeout: true,
		},
		{
			name:         "max_turns detected (underscore)",
			errorMsg:     "hit max_turns limit of 60",
			wantMaxTurns: true,
		},
		{
			name:         "max turns detected (space)",
			errorMsg:     "exceeded max turns",
			wantMaxTurns: true,
		},
		{
			name:     "generic error triggers none",
			errorMsg: "task failed with unknown error",
		},
		{
			name:     "empty error triggers none",
			errorMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := &domainpb.Run{
				Id:       "run-456",
				Status:   domainpb.RunStatus_RUN_STATUS_FAILED,
				ErrorMsg: tt.errorMsg,
			}
			result := svc.buildExecuteResult(run)

			if result.RateLimited != tt.wantRateLimited {
				t.Errorf("RateLimited = %v, want %v", result.RateLimited, tt.wantRateLimited)
			}
			if result.Timeout != tt.wantTimeout {
				t.Errorf("Timeout = %v, want %v", result.Timeout, tt.wantTimeout)
			}
			if result.MaxTurnsExceeded != tt.wantMaxTurns {
				t.Errorf("MaxTurnsExceeded = %v, want %v", result.MaxTurnsExceeded, tt.wantMaxTurns)
			}
			if tt.errorMsg != "" && result.ErrorMessage != tt.errorMsg {
				t.Errorf("ErrorMessage = %q, want %q", result.ErrorMessage, tt.errorMsg)
			}
		})
	}
}

func TestBuildExecuteResult_SummaryExtraction(t *testing.T) {
	svc := &AgentService{}

	t.Run("extracts summary fields", func(t *testing.T) {
		run := &domainpb.Run{
			Id:     "run-789",
			Status: domainpb.RunStatus_RUN_STATUS_COMPLETE,
			Summary: &domainpb.RunSummary{
				Description:  "Task completed successfully",
				TokensUsed:   1500,
				CostEstimate: 0.025,
			},
		}
		result := svc.buildExecuteResult(run)

		if result.Output != "Task completed successfully" {
			t.Errorf("Output = %q, want %q", result.Output, "Task completed successfully")
		}
		if result.TokensUsed != 1500 {
			t.Errorf("TokensUsed = %d, want %d", result.TokensUsed, 1500)
		}
		if result.CostEstimate != 0.025 {
			t.Errorf("CostEstimate = %f, want %f", result.CostEstimate, 0.025)
		}
	})

	t.Run("nil summary yields zero values", func(t *testing.T) {
		run := &domainpb.Run{
			Id:     "run-000",
			Status: domainpb.RunStatus_RUN_STATUS_FAILED,
		}
		result := svc.buildExecuteResult(run)

		if result.Output != "" {
			t.Errorf("Output = %q, want empty", result.Output)
		}
		if result.TokensUsed != 0 {
			t.Errorf("TokensUsed = %d, want 0", result.TokensUsed)
		}
	})
}

func TestBuildExecuteResult_DurationCalculation(t *testing.T) {
	svc := &AgentService{}

	t.Run("calculates duration from timestamps", func(t *testing.T) {
		start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
		end := start.Add(90 * time.Second)

		run := &domainpb.Run{
			Id:        "run-dur",
			Status:    domainpb.RunStatus_RUN_STATUS_COMPLETE,
			StartedAt: timestamppb.New(start),
			EndedAt:   timestamppb.New(end),
		}
		result := svc.buildExecuteResult(run)

		if result.DurationSeconds != 90 {
			t.Errorf("DurationSeconds = %d, want 90", result.DurationSeconds)
		}
	})

	t.Run("nil timestamps yield zero duration", func(t *testing.T) {
		run := &domainpb.Run{
			Id:     "run-nodur",
			Status: domainpb.RunStatus_RUN_STATUS_RUNNING,
		}
		result := svc.buildExecuteResult(run)

		if result.DurationSeconds != 0 {
			t.Errorf("DurationSeconds = %d, want 0", result.DurationSeconds)
		}
	})
}
