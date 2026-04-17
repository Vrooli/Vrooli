// Package services contains business logic orchestration.
// This file handles tool call approval and rejection flows.
package services

import (
	"agent-inbox/domain"
	"context"
	"fmt"
	"log"
	"time"
)

// ApprovalResult contains the result of approving a tool call.
type ApprovalResult struct {
	// ToolResult is the execution result after approval.
	ToolResult *domain.ToolCallRecord
	// PendingApprovals remaining after this approval.
	PendingApprovals []*domain.ToolCallRecord
	// AutoContinued indicates if all approvals are resolved and continuation was triggered.
	AutoContinued bool
}

// ApproveToolCall approves and executes a pending tool call.
// Returns the execution result and whether auto-continuation should occur.
func (s *CompletionService) ApproveToolCall(ctx context.Context, chatID, toolCallID string) (*ApprovalResult, error) {
	record, err := s.validatePendingToolCall(ctx, chatID, toolCallID)
	if err != nil {
		return nil, err
	}

	// Update status to approved
	if err := s.repo.UpdateToolCallStatus(ctx, toolCallID, domain.StatusApproved, ""); err != nil {
		return nil, fmt.Errorf("failed to update status: %w", err)
	}

	// Execute the tool
	executedRecord, err := s.executor.ExecuteTool(ctx, chatID, toolCallID, record.ToolName, record.Arguments)
	if err != nil {
		log.Printf("warning: tool execution failed after approval: %v", err)
	}

	// Save tool call record, response message, and update active leaf
	s.saveApprovalResult(ctx, chatID, record.MessageID, toolCallID, executedRecord)

	// Check for remaining pending approvals
	pending, _ := s.repo.GetPendingApprovals(ctx, chatID)

	return &ApprovalResult{
		ToolResult:       executedRecord,
		PendingApprovals: pending,
		AutoContinued:    len(pending) == 0,
	}, nil
}

// RejectToolCall rejects a pending tool call.
func (s *CompletionService) RejectToolCall(ctx context.Context, chatID, toolCallID, reason string) error {
	record, err := s.validatePendingToolCall(ctx, chatID, toolCallID)
	if err != nil {
		return err
	}

	// Update status to rejected
	errorMsg := "Rejected by user"
	if reason != "" {
		errorMsg = reason
	}
	if err := s.repo.UpdateToolCallStatus(ctx, toolCallID, domain.StatusRejected, errorMsg); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Save tool response message with rejection info
	rejectionResult := fmt.Sprintf(`{"rejected": true, "reason": %q}`, reason)
	s.saveRejectionResult(ctx, chatID, record, toolCallID, rejectionResult, errorMsg)

	return nil
}

// GetPendingApprovals returns all pending tool call approvals for a chat.
func (s *CompletionService) GetPendingApprovals(ctx context.Context, chatID string) ([]*domain.ToolCallRecord, error) {
	return s.repo.GetPendingApprovals(ctx, chatID)
}

// validatePendingToolCall retrieves and validates a tool call is pending approval.
func (s *CompletionService) validatePendingToolCall(ctx context.Context, chatID, toolCallID string) (*domain.ToolCallRecord, error) {
	record, err := s.repo.GetToolCallByID(ctx, toolCallID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tool call: %w", err)
	}
	if record == nil {
		return nil, fmt.Errorf("tool call not found: %s", toolCallID)
	}
	if record.Status != domain.StatusPendingApproval {
		return nil, fmt.Errorf("tool call is not pending approval: status=%s", record.Status)
	}
	if record.ChatID != chatID {
		return nil, fmt.Errorf("tool call does not belong to chat")
	}
	return record, nil
}

// createPendingApprovalRecord creates a ToolCallRecord for a tool awaiting approval.
func (s *CompletionService) createPendingApprovalRecord(chatID, messageID string, tc domain.ToolCall) *domain.ToolCallRecord {
	return &domain.ToolCallRecord{
		ID:        tc.ID,
		MessageID: messageID,
		ChatID:    chatID,
		ToolName:  tc.Function.Name,
		Arguments: tc.Function.Arguments,
		Status:    domain.StatusPendingApproval,
		StartedAt: time.Now(),
	}
}

// saveApprovalResult persists the result of an approved tool call.
func (s *CompletionService) saveApprovalResult(ctx context.Context, chatID, messageID, toolCallID string, record *domain.ToolCallRecord) {
	if s.toolPersistence != nil {
		toolMsg, saveErr := s.toolPersistence.SaveToolResult(ctx, SaveToolResultParams{
			ChatID:          chatID,
			MessageID:       messageID,
			ToolCallID:      toolCallID,
			Record:          record,
			Result:          record.Result,
			ParentMessageID: messageID,
		})
		if saveErr != nil {
			log.Printf("[ERROR] Failed to save approval result: %v", saveErr)
			return
		}
		log.Printf("[DEBUG] Atomically saved approval result: msg=%s, tool_call_id=%s", toolMsg.ID, toolCallID)
	} else {
		_ = s.repo.SaveToolCallRecord(ctx, messageID, record)
		toolMsg, _ := s.repo.SaveToolResponseMessage(ctx, chatID, toolCallID, record.Result, messageID)
		if toolMsg != nil {
			_ = s.repo.SetActiveLeaf(ctx, chatID, toolMsg.ID)
		}
	}
}

// saveRejectionResult persists the result of a rejected tool call.
func (s *CompletionService) saveRejectionResult(ctx context.Context, chatID string, record *domain.ToolCallRecord, toolCallID, rejectionResult, errorMsg string) {
	if s.toolPersistence != nil {
		rejectionRecord := &domain.ToolCallRecord{
			ID:           toolCallID,
			MessageID:    record.MessageID,
			ChatID:       chatID,
			ToolName:     record.ToolName,
			Arguments:    record.Arguments,
			Result:       rejectionResult,
			Status:       domain.StatusRejected,
			ErrorMessage: errorMsg,
			StartedAt:    record.StartedAt,
		}
		toolMsg, saveErr := s.toolPersistence.SaveToolResult(ctx, SaveToolResultParams{
			ChatID:          chatID,
			MessageID:       record.MessageID,
			ToolCallID:      toolCallID,
			Record:          rejectionRecord,
			Result:          rejectionResult,
			ParentMessageID: record.MessageID,
		})
		if saveErr != nil {
			log.Printf("[ERROR] Failed to save rejection result: %v", saveErr)
			return
		}
		log.Printf("[DEBUG] Atomically saved rejection result: msg=%s, tool_call_id=%s", toolMsg.ID, toolCallID)
	} else {
		toolMsg, _ := s.repo.SaveToolResponseMessage(ctx, chatID, toolCallID, rejectionResult, record.MessageID)
		if toolMsg != nil {
			_ = s.repo.SetActiveLeaf(ctx, chatID, toolMsg.ID)
		}
	}
}
