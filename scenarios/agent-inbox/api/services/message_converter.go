// Package services provides business logic orchestration.
// This file converts domain messages to OpenRouter multimodal format.
package services

import (
	"context"
	"log"

	"agent-inbox/domain"
	"agent-inbox/integrations"
)

// MessageConverter transforms domain messages into OpenRouter API format.
// This handles the complexity of multimodal content (text + images + files).
type MessageConverter struct {
	storage StorageService
}

// NewMessageConverter creates a new converter with the given storage service.
func NewMessageConverter(storage StorageService) *MessageConverter {
	return &MessageConverter{storage: storage}
}

// ConvertToOpenRouter converts domain messages to OpenRouter format.
// If a message has attachments, it builds a multimodal content array.
// Otherwise, it uses a simple string content.
func (c *MessageConverter) ConvertToOpenRouter(ctx context.Context, messages []domain.Message, attachmentsByMsgID map[string][]domain.Attachment) []integrations.OpenRouterMessage {
	result := make([]integrations.OpenRouterMessage, 0, len(messages))

	for _, msg := range messages {
		orMsg := integrations.OpenRouterMessage{
			Role:       msg.Role,
			ToolCallID: msg.ToolCallID,
			ToolCalls:  msg.ToolCalls,
		}

		// Check for attachments
		attachments := attachmentsByMsgID[msg.ID]
		if len(attachments) > 0 {
			// Build multimodal content array
			orMsg.Content = c.buildMultimodalContent(ctx, msg.Content, attachments)
		} else {
			// Simple string content
			orMsg.Content = msg.Content
		}

		result = append(result, orMsg)
	}

	return result
}

// buildMultimodalContent creates a content array with text and file parts.
func (c *MessageConverter) buildMultimodalContent(ctx context.Context, text string, attachments []domain.Attachment) []integrations.ContentPart {
	parts := make([]integrations.ContentPart, 0, len(attachments)+1)

	// Add text content first (if any)
	if text != "" {
		parts = append(parts, integrations.ContentPart{
			Type: "text",
			Text: text,
		})
	}

	// Add file parts
	for _, att := range attachments {
		fileData, err := c.storage.ReadAsBase64DataURI(ctx, att.StoragePath)
		if err != nil {
			log.Printf("[WARN] Failed to read attachment %s: %v", att.ID, err)
			continue
		}

		parts = append(parts, integrations.ContentPart{
			Type: "file",
			File: &integrations.FileContent{
				Filename: att.FileName,
				FileData: fileData,
			},
		})
	}

	return parts
}

// BuildPlugins creates the plugins array for an OpenRouter request.
// Enables web search and/or PDF parsing based on message configuration.
func (c *MessageConverter) BuildPlugins(webSearchEnabled bool, hasPDF bool) []integrations.OpenRouterPlugin {
	var plugins []integrations.OpenRouterPlugin

	if webSearchEnabled {
		plugins = append(plugins, integrations.OpenRouterPlugin{
			ID:         "web",
			MaxResults: 5, // Default, can be made configurable later
		})
	}

	if hasPDF {
		plugins = append(plugins, integrations.OpenRouterPlugin{
			ID: "file-parser",
		})
	}

	return plugins
}

// DetermineWebSearch resolves the effective web search setting for a message.
// Per-message setting overrides chat default.
func DetermineWebSearch(messageWebSearch *bool, chatWebSearchEnabled bool) bool {
	if messageWebSearch != nil {
		return *messageWebSearch
	}
	return chatWebSearchEnabled
}

// HasPDFAttachment checks if any attachment is a PDF.
func HasPDFAttachment(attachments []domain.Attachment) bool {
	for _, att := range attachments {
		if att.IsPDF() {
			return true
		}
	}
	return false
}

// ConvertMessagesLegacy converts map-based messages to OpenRouter format.
// This is kept for backward compatibility with existing code.
func ConvertMessagesLegacy(messages []map[string]interface{}) []integrations.OpenRouterMessage {
	result := make([]integrations.OpenRouterMessage, len(messages))
	for i, m := range messages {
		msg := integrations.OpenRouterMessage{
			Role: m["role"].(string),
		}
		if content, ok := m["content"].(string); ok {
			msg.Content = content
		}
		if tcid, ok := m["tool_call_id"].(string); ok {
			msg.ToolCallID = tcid
		}
		if tcs, ok := m["tool_calls"].([]domain.ToolCall); ok {
			msg.ToolCalls = tcs
		}
		result[i] = msg
	}
	return result
}
