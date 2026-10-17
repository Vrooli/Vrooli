// Package services contains business logic orchestration.
// This file handles preparation and building of completion requests.
package services

import (
	"context"
	"fmt"
	"log"

	"agent-inbox/domain"
	"agent-inbox/integrations"
)

// CompletionRequest contains validated data needed to make a completion.
type CompletionRequest struct {
	ChatID               string
	Model                string
	Messages             []integrations.OpenRouterMessage
	Tools                []map[string]interface{}
	ToolChoice           interface{} // nil for auto, ToolChoiceFunction for forced tool
	Plugins              []integrations.OpenRouterPlugin
	Modalities           []string // ["image", "text"] for image generation models
	Streaming            bool
	DiscoveryDiagnostics []string
}

// ShouldIncludeTools returns true if tools should be sent with the request.
func (r *CompletionRequest) ShouldIncludeTools() bool {
	return len(r.Tools) > 0
}

// ShouldIncludePlugins returns true if plugins should be sent with the request.
func (r *CompletionRequest) ShouldIncludePlugins() bool {
	return len(r.Plugins) > 0
}

// ShouldIncludeModalities returns true if modalities should be sent with the request.
func (r *CompletionRequest) ShouldIncludeModalities() bool {
	return len(r.Modalities) > 0
}

// PrepareCompletionRequest builds a validated completion request.
//
// Steps: load settings, load messages, fetch attachments, truncate to context window,
// convert to OpenRouter format, inject command/async guidance, and build plugins.
func (s *CompletionService) PrepareCompletionRequest(ctx context.Context, chatID string, streaming bool, _ string) (*CompletionRequest, error) {
	// Get chat settings
	settings, err := s.GetChatSettings(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabaseError, err)
	}
	if settings == nil {
		return nil, ErrChatNotFound
	}

	// Get messages and attachments
	messages, attachmentsByMsgID, err := s.loadMessagesAndAttachments(ctx, chatID)
	if err != nil {
		return nil, err
	}

	// Check if this is an image generation model
	isImageGen := s.contextManager.IsImageGenerationModel(ctx, settings.Model)

	// Validate and truncate messages to fit context window
	messages, err = s.contextManager.ValidateAndTruncate(ctx, settings.Model, messages)
	if err != nil {
		log.Printf("warning: context validation failed: %v", err)
	}

	// Filter attachments to only include messages that survived truncation
	filteredAttachments := filterAttachmentsByMessages(messages, attachmentsByMsgID)

	// Convert messages with multimodal support
	orMessages := s.messageConverter.ConvertToOpenRouter(ctx, messages, filteredAttachments)

	var discoveryDiagnostic string
	orMessages, discoveryDiagnostic = s.maybeInjectCommandContext(ctx, chatID, messages, orMessages)

	// Inject async guidance if operations are active
	orMessages = s.maybeInjectAsyncGuidance(chatID, orMessages)

	// Determine effective web search setting
	webSearchEnabled := s.resolveWebSearchSetting(settings.WebSearchEnabled, messages)

	// Check for PDF attachments
	hasPDF := hasPDFInAttachments(attachmentsByMsgID)

	req := &CompletionRequest{
		ChatID:    chatID,
		Model:     settings.Model,
		Messages:  orMessages,
		Plugins:   s.messageConverter.BuildPlugins(webSearchEnabled, hasPDF),
		Streaming: streaming,
	}
	if discoveryDiagnostic != "" {
		req.DiscoveryDiagnostics = append(req.DiscoveryDiagnostics, discoveryDiagnostic)
	}

	// Debug: Log messages being sent to OpenRouter
	logCompletionMessages(orMessages)

	// Enable image generation modalities if the model supports it
	if isImageGen {
		req.Modalities = []string{"image", "text"}
		log.Printf("[DEBUG] Image generation enabled for model: %s (tools disabled)", settings.Model)
	}

	return req, nil
}

// loadMessagesAndAttachments loads messages and their attachments for a chat.
func (s *CompletionService) loadMessagesAndAttachments(ctx context.Context, chatID string) ([]domain.Message, map[string][]domain.Attachment, error) {
	messages, err := s.repo.GetMessagesForCompletion(ctx, chatID)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrMessagesFailed, err)
	}
	if len(messages) == 0 {
		return nil, nil, ErrNoMessages
	}

	messageIDs := make([]string, len(messages))
	for i, msg := range messages {
		messageIDs[i] = msg.ID
	}

	attachmentsByMsgID, err := s.repo.GetAttachmentsForMessages(ctx, messageIDs)
	if err != nil {
		log.Printf("warning: failed to fetch attachments: %v", err)
		attachmentsByMsgID = make(map[string][]domain.Attachment)
	}

	return messages, attachmentsByMsgID, nil
}

// filterAttachmentsByMessages returns only attachments for messages in the list.
func filterAttachmentsByMessages(messages []domain.Message, attachmentsByMsgID map[string][]domain.Attachment) map[string][]domain.Attachment {
	messageIDSet := make(map[string]bool, len(messages))
	for _, msg := range messages {
		messageIDSet[msg.ID] = true
	}
	filtered := make(map[string][]domain.Attachment)
	for msgID, atts := range attachmentsByMsgID {
		if messageIDSet[msgID] {
			filtered[msgID] = atts
		}
	}
	return filtered
}

// maybeInjectAsyncGuidance injects async guidance system message if operations are active.
func (s *CompletionService) maybeInjectAsyncGuidance(chatID string, orMessages []integrations.OpenRouterMessage) []integrations.OpenRouterMessage {
	if s.asyncTracker == nil {
		return orMessages
	}
	activeOps := s.asyncTracker.GetActiveOperations(chatID)
	if len(activeOps) == 0 {
		return orMessages
	}
	guidance := s.buildAsyncGuidanceMessage(activeOps)
	orMessages = append([]integrations.OpenRouterMessage{
		{Role: "system", Content: guidance},
	}, orMessages...)
	log.Printf("[DEBUG] Injected async guidance for %d active operations", len(activeOps))
	return orMessages
}

// resolveWebSearchSetting determines if web search should be enabled.
func (s *CompletionService) resolveWebSearchSetting(chatDefault bool, messages []domain.Message) bool {
	webSearchEnabled := chatDefault
	log.Printf("[DEBUG] web search: chat default=%v, checking %d messages", webSearchEnabled, len(messages))
	for _, msg := range messages {
		if msg.Role == "user" {
			log.Printf("[DEBUG] user message %s: web_search=%v", msg.ID, msg.WebSearch)
			if msg.WebSearch != nil && *msg.WebSearch {
				webSearchEnabled = true
				break
			}
		}
	}
	log.Printf("[DEBUG] effective web search enabled=%v", webSearchEnabled)
	return webSearchEnabled
}

// hasPDFInAttachments checks if any message has a PDF attachment.
func hasPDFInAttachments(attachmentsByMsgID map[string][]domain.Attachment) bool {
	for _, attachments := range attachmentsByMsgID {
		if HasPDFAttachment(attachments) {
			return true
		}
	}
	return false
}

// logCompletionMessages logs debug info about messages being sent.
func logCompletionMessages(orMessages []integrations.OpenRouterMessage) {
	for i, msg := range orMessages {
		toolCallsStr := ""
		if len(msg.ToolCalls) > 0 {
			ids := make([]string, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				ids[j] = tc.ID
			}
			toolCallsStr = fmt.Sprintf(", tool_calls=%v", ids)
		}
		toolCallIDStr := ""
		if msg.ToolCallID != "" {
			toolCallIDStr = fmt.Sprintf(", tool_call_id=%s", msg.ToolCallID)
		}
		contentPreview := formatContentPreview(msg.Content)
		log.Printf("[DEBUG] PrepareCompletionRequest msg[%d]: role=%s, content=%q%s%s",
			i, msg.Role, contentPreview, toolCallsStr, toolCallIDStr)
	}
}

// formatContentPreview returns a truncated preview of message content.
func formatContentPreview(content interface{}) string {
	if s, ok := content.(string); ok && len(s) > 50 {
		return s[:50] + "..."
	} else if s, ok := content.(string); ok {
		return s
	}
	return "[multimodal]"
}
