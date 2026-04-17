package services

import (
	"agent-inbox/domain"
	"context"
)

// =============================================================================
// Mock Repository - Message and Tool Operations
// =============================================================================

func (m *mockCompletionRepository) SaveAssistantMessage(ctx context.Context, chatID, model, content string, tokenCount int, parentMessageID string) (*domain.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.saveAssistantMessageError != nil {
		return nil, m.saveAssistantMessageError
	}

	m.saveAssistantMessageCalls = append(m.saveAssistantMessageCalls, saveAssistantMessageCall{
		ChatID:  chatID,
		Model:   model,
		Content: content,
	})

	msg := &domain.Message{
		ID:              m.generateMessageID(),
		ChatID:          chatID,
		Role:            "assistant",
		Content:         content,
		Model:           model,
		ParentMessageID: parentMessageID,
	}
	m.messages[chatID] = append(m.messages[chatID], msg)
	return msg, nil
}

func (m *mockCompletionRepository) SaveAssistantMessageWithToolCalls(ctx context.Context, chatID, model, content string, toolCalls []domain.ToolCall, responseID, finishReason string, tokenCount int, parentMessageID string) (*domain.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.saveAssistantMessageError != nil {
		return nil, m.saveAssistantMessageError
	}

	m.saveAssistantMessageCalls = append(m.saveAssistantMessageCalls, saveAssistantMessageCall{
		ChatID:    chatID,
		Model:     model,
		Content:   content,
		ToolCalls: toolCalls,
	})

	msg := &domain.Message{
		ID:              m.generateMessageID(),
		ChatID:          chatID,
		Role:            "assistant",
		Content:         content,
		Model:           model,
		ParentMessageID: parentMessageID,
		ToolCalls:       toolCalls,
		FinishReason:    finishReason,
	}
	m.messages[chatID] = append(m.messages[chatID], msg)
	return msg, nil
}

func (m *mockCompletionRepository) SaveToolResponseMessage(ctx context.Context, chatID, toolCallID, result string, parentMessageID string) (*domain.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.saveToolResponseMessageError != nil {
		return nil, m.saveToolResponseMessageError
	}

	m.saveToolResponseMessageCalls = append(m.saveToolResponseMessageCalls, saveToolResponseMessageCall{
		ChatID:     chatID,
		ToolCallID: toolCallID,
		Result:     result,
		ParentID:   parentMessageID,
	})

	msg := &domain.Message{
		ID:              m.generateMessageID(),
		ChatID:          chatID,
		Role:            "tool",
		Content:         result,
		ToolCallID:      toolCallID,
		ParentMessageID: parentMessageID,
	}
	m.messages[chatID] = append(m.messages[chatID], msg)
	return msg, nil
}

func (m *mockCompletionRepository) SetActiveLeaf(ctx context.Context, chatID, messageID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.setActiveLeafCalls = append(m.setActiveLeafCalls, setActiveLeafCall{
		ChatID:    chatID,
		MessageID: messageID,
	})
	m.activeLeafs[chatID] = messageID
	return nil
}

func (m *mockCompletionRepository) GetAttachmentsForMessages(ctx context.Context, messageIDs []string) (map[string][]domain.Attachment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make(map[string][]domain.Attachment)
	for _, msgID := range messageIDs {
		attIDs := m.messageAttachments[msgID]
		for _, attID := range attIDs {
			if att := m.attachments[attID]; att != nil {
				result[msgID] = append(result[msgID], *att)
			}
		}
	}
	return result, nil
}

func (m *mockCompletionRepository) CreateAttachment(ctx context.Context, att *domain.Attachment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attachments[att.ID] = att
	return nil
}

func (m *mockCompletionRepository) AttachToMessage(ctx context.Context, attachmentID, messageID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messageAttachments[messageID] = append(m.messageAttachments[messageID], attachmentID)
	return nil
}

func (m *mockCompletionRepository) SaveToolCallRecord(ctx context.Context, messageID string, record *domain.ToolCallRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.saveToolCallRecordError != nil {
		return m.saveToolCallRecordError
	}

	m.saveToolCallRecordCalls = append(m.saveToolCallRecordCalls, saveToolCallRecordCall{
		MessageID: messageID,
		Record:    record,
	})
	m.toolCallRecords[record.ID] = record
	return nil
}

func (m *mockCompletionRepository) GetToolCallByID(ctx context.Context, toolCallID string) (*domain.ToolCallRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.getToolCallByIDError != nil {
		return nil, m.getToolCallByIDError
	}
	return m.toolCallRecords[toolCallID], nil
}

func (m *mockCompletionRepository) UpdateToolCallStatus(ctx context.Context, id, status, errorMessage string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.updateToolCallStatusError != nil {
		return m.updateToolCallStatusError
	}

	m.updateToolCallStatusCalls = append(m.updateToolCallStatusCalls, updateToolCallStatusCall{
		ID:           id,
		Status:       status,
		ErrorMessage: errorMessage,
	})

	if record := m.toolCallRecords[id]; record != nil {
		record.Status = status
		record.ErrorMessage = errorMessage
	}
	return nil
}

func (m *mockCompletionRepository) GetPendingApprovals(ctx context.Context, chatID string) ([]*domain.ToolCallRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.getPendingApprovalsError != nil {
		return nil, m.getPendingApprovalsError
	}

	var pending []*domain.ToolCallRecord
	for _, record := range m.toolCallRecords {
		if record.ChatID == chatID && record.Status == domain.StatusPendingApproval {
			pending = append(pending, record)
		}
	}
	return pending, nil
}
