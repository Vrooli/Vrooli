package handlers

import (
	"context"
	"log"
	"strings"
	"time"
)

const defaultAutoNameSeed = "New Chat"

// maybeAutoNameChat auto-generates a chat name only when the current name
// is still the default seed name. This protects user-provided names.
func (h *Handlers) maybeAutoNameChat(ctx context.Context, chatID string) {
	if h == nil || h.Repo == nil || h.OllamaClient == nil || chatID == "" {
		return
	}

	chat, err := h.Repo.GetChat(ctx, chatID)
	if err != nil || chat == nil {
		return
	}
	if strings.TrimSpace(chat.Name) != defaultAutoNameSeed {
		return
	}

	messages, err := h.Repo.GetMessages(ctx, chatID)
	if err != nil || len(messages) == 0 {
		return
	}

	maxMessages, maxContentLen := h.OllamaClient.SummaryLimits()
	summary := buildConversationSummary(messages, maxMessages, maxContentLen)
	if strings.TrimSpace(summary) == "" {
		return
	}

	// Keep automatic naming opportunistic and fast; do not block request flow
	// for long when local Ollama is unavailable or slow.
	nameCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	name, err := h.OllamaClient.GenerateChatName(nameCtx, summary)
	if err != nil {
		log.Printf("[DEBUG] maybeAutoNameChat skipped for chat %s: %v", chatID, err)
		return
	}

	name = strings.TrimSpace(name)
	if name == "" || name == defaultAutoNameSeed {
		return
	}

	if _, err := h.Repo.UpdateChat(ctx, chatID, &name, nil); err != nil {
		log.Printf("[DEBUG] maybeAutoNameChat update failed for chat %s: %v", chatID, err)
	}
}
