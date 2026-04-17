// Package services contains business logic orchestration.
// This file contains startup reconciliation for orphaned tool calls.
//
// TEMPORAL FLOW DESIGN:
// When the server restarts, tool calls that were in 'pending' or 'running' status
// may be orphaned. This service reconciles their state by:
//  1. Checking if the external agent (agent-manager) is still running
//  2. Updating the status to 'cancelled' if the agent is not found
//  3. Updating to 'completed' or 'failed' if the agent finished while we were down
//
// This ensures progress continuity across server restarts.
package services

import (
	"agent-inbox/domain"
	"agent-inbox/integrations"
	"agent-inbox/persistence"
	"context"
	"log"
	"time"
)

// ReconciliationService handles startup reconciliation of orphaned tool calls.
type ReconciliationService struct {
	repo *persistence.Repository
}

// NewReconciliationService creates a new reconciliation service.
func NewReconciliationService(repo *persistence.Repository) *ReconciliationService {
	return &ReconciliationService{repo: repo}
}

// ReconcileOrphanedToolCalls finds and reconciles tool calls that were interrupted
// by a server restart. This should be called during startup after database connection.
//
// Returns the number of tool calls reconciled and any error encountered.
func (s *ReconciliationService) ReconcileOrphanedToolCalls(ctx context.Context) (int, error) {
	orphaned, err := s.repo.ListOrphanedToolCalls(ctx)
	if err != nil {
		return 0, err
	}

	if len(orphaned) == 0 {
		return 0, nil
	}

	log.Printf("reconciliation: found %d orphaned tool calls", len(orphaned))

	reconciled := 0
	for _, tc := range orphaned {
		if err := s.reconcileToolCall(ctx, tc); err != nil {
			log.Printf("reconciliation: failed to reconcile tool call %s: %v", tc.ID, err)
			continue
		}
		reconciled++
	}

	return reconciled, nil
}

// reconcileToolCall reconciles a single orphaned tool call.
func (s *ReconciliationService) reconcileToolCall(ctx context.Context, tc domain.ToolCallRecord) error {
	// If this tool call has an external run ID (agent-manager), try to check its status
	if tc.ExternalRunID != "" && tc.ScenarioName == "agent-manager" {
		return s.reconcileAgentManagerToolCall(ctx, tc)
	}

	// For tool calls without external tracking, mark as cancelled
	return s.markAsCancelled(ctx, tc, "Server restarted while tool was executing")
}

// reconcileAgentManagerToolCall checks agent-manager for the current status of an agent run.
func (s *ReconciliationService) reconcileAgentManagerToolCall(ctx context.Context, tc domain.ToolCallRecord) error {
	client, err := integrations.NewAgentManagerClient()
	if err != nil {
		// Agent manager not available, mark as cancelled
		return s.markAsCancelled(ctx, tc, "Agent manager unavailable during reconciliation")
	}

	// Check the agent status - returns a proto Run
	run, err := client.CheckAgentStatus(ctx, tc.ExternalRunID)
	if err != nil {
		// Could not reach agent-manager or run not found
		return s.markAsCancelled(ctx, tc, "Could not verify agent status: "+err.Error())
	}

	// Convert proto status to local status string
	localStatus := integrations.ProtoRunStatusToLocal(run.GetStatus())

	if integrations.IsActiveRunStatus(string(localStatus)) {
		// Agent is still running - leave as-is but log it
		log.Printf("reconciliation: agent run %s still active (status=%s) for tool call %s",
			tc.ExternalRunID, localStatus, tc.ID)
		return nil
	}

	// Terminal status - map to local tool call status
	switch localStatus {
	case integrations.RunStatusComplete:
		return s.repo.UpdateToolCallStatus(ctx, tc.ID, domain.StatusCompleted, "")

	case integrations.RunStatusFailed:
		errorMsg := run.GetErrorMsg()
		if errorMsg == "" {
			errorMsg = "Agent run failed"
		}
		return s.repo.UpdateToolCallStatus(ctx, tc.ID, domain.StatusFailed, errorMsg)

	case integrations.RunStatusCancelled:
		return s.repo.UpdateToolCallStatus(ctx, tc.ID, domain.StatusCancelled, "Agent run was cancelled")

	default:
		return s.markAsCancelled(ctx, tc, "Unknown agent status: "+string(localStatus))
	}
}

// markAsCancelled updates a tool call to cancelled status with an error message.
func (s *ReconciliationService) markAsCancelled(ctx context.Context, tc domain.ToolCallRecord, reason string) error {
	log.Printf("reconciliation: marking tool call %s (%s) as cancelled: %s", tc.ID, tc.ToolName, reason)
	return s.repo.UpdateToolCallStatus(ctx, tc.ID, domain.StatusCancelled, reason)
}

// ReconciliationStats contains summary statistics from reconciliation.
type ReconciliationStats struct {
	TotalOrphaned int           `json:"total_orphaned"`
	Reconciled    int           `json:"reconciled"`
	Duration      time.Duration `json:"duration"`
}
