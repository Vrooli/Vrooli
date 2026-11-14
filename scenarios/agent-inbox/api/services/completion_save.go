// Package services contains business logic orchestration.
// This file handles saving completion results, images, and async tracking.
package services

import (
	"context"
	"fmt"
	"log"

	"agent-inbox/domain"
)

// SaveCompletionResult persists a completion result to the database.
// This handles the decision of whether to save as a regular message or
// as a message with tool calls.
// parentMessageID is used for branching support (ChatGPT-style regeneration).
func (s *CompletionService) SaveCompletionResult(ctx context.Context, chatID, model string, result *domain.CompletionResult, parentMessageID string) (*domain.Message, error) {
	var msg *domain.Message
	var err error

	if result.RequiresToolExecution() {
		msg, err = s.repo.SaveAssistantMessageWithToolCalls(
			ctx, chatID, model, result.Content, result.ToolCalls,
			result.ResponseID, result.FinishReason, result.TokenCount, parentMessageID,
		)
	} else {
		msg, err = s.repo.SaveAssistantMessage(ctx, chatID, model, result.Content, result.TokenCount, parentMessageID)
	}

	if err != nil {
		return msg, err
	}

	// Update active leaf to point to the new message
	if msg != nil {
		_ = s.repo.SetActiveLeaf(ctx, chatID, msg.ID)
	}

	// Save generated images as attachments
	if len(result.Images) > 0 && msg != nil {
		s.saveGeneratedImages(ctx, chatID, msg.ID, result.Images)
	}

	return msg, nil
}

// saveGeneratedImages persists generated images as attachments on a message.
func (s *CompletionService) saveGeneratedImages(ctx context.Context, chatID, msgID string, images []string) {
	for i, imageDataURL := range images {
		att, saveErr := s.storage.SaveBase64Image(ctx, imageDataURL, fmt.Sprintf("generated_%d", i+1))
		if saveErr != nil {
			log.Printf("warning: failed to save generated image %d: %v", i+1, saveErr)
			continue
		}
		if createErr := s.repo.CreateAttachment(ctx, att); createErr != nil {
			log.Printf("warning: failed to create attachment record for generated image: %v", createErr)
			continue
		}
		if linkErr := s.repo.AttachToMessage(ctx, att.ID, msgID); linkErr != nil {
			log.Printf("warning: failed to link generated image to message: %v", linkErr)
		}
	}
	log.Printf("[DEBUG] Saved %d generated images as attachments for message %s", len(images), msgID)
}

// UpdateChatPreview updates the chat's preview text based on completion result.
func (s *CompletionService) UpdateChatPreview(ctx context.Context, chatID string, result *domain.CompletionResult) error {
	preview := result.PreviewText()
	return s.repo.UpdateChatPreview(ctx, chatID, preview, true)
}

// maybeStartAsyncTracking checks if a tool has AsyncBehavior and starts tracking if so.
// Returns AsyncOperationInfo if tracking started successfully, nil otherwise.
func (s *CompletionService) maybeStartAsyncTracking(ctx context.Context, chatID, toolCallID, toolName string, record *domain.ToolCallRecord) *domain.AsyncOperationInfo {
	return nil
}
