package main

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// OpenRouter API types

type OpenRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenRouterRequest struct {
	Model    string              `json:"model"`
	Messages []OpenRouterMessage `json:"messages"`
	Stream   bool                `json:"stream"`
}

type OpenRouterChoice struct {
	Index        int               `json:"index"`
	Message      OpenRouterMessage `json:"message,omitempty"`
	Delta        OpenRouterMessage `json:"delta,omitempty"`
	FinishReason string            `json:"finish_reason,omitempty"`
}

type OpenRouterUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type OpenRouterResponse struct {
	ID      string             `json:"id"`
	Model   string             `json:"model"`
	Choices []OpenRouterChoice `json:"choices"`
	Usage   OpenRouterUsage    `json:"usage,omitempty"`
}

type ModelInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Pricing     struct {
		Prompt     string `json:"prompt"`
		Completion string `json:"completion"`
	} `json:"pricing,omitempty"`
}

// Available models - curated list of popular models
var availableModels = []ModelInfo{
	{ID: "anthropic/claude-3.5-sonnet", Name: "Claude 3.5 Sonnet", Description: "Anthropic's most intelligent model"},
	{ID: "anthropic/claude-3-haiku", Name: "Claude 3 Haiku", Description: "Fast and cost-effective"},
	{ID: "openai/gpt-4o", Name: "GPT-4o", Description: "OpenAI's flagship model"},
	{ID: "openai/gpt-4o-mini", Name: "GPT-4o Mini", Description: "Fast and affordable GPT-4"},
	{ID: "google/gemini-pro-1.5", Name: "Gemini Pro 1.5", Description: "Google's latest model"},
	{ID: "meta-llama/llama-3.1-70b-instruct", Name: "Llama 3.1 70B", Description: "Meta's open-source model"},
	{ID: "mistralai/mistral-large", Name: "Mistral Large", Description: "Mistral's flagship model"},
}

// handleListModels returns available OpenRouter models
func (s *Server) handleListModels(w http.ResponseWriter, r *http.Request) {
	s.jsonResponse(w, availableModels, http.StatusOK)
}

// handleChatComplete sends conversation to OpenRouter and streams the response
func (s *Server) handleChatComplete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID := vars["id"]

	// Validate chat ID
	var chatModel string
	err := s.db.QueryRowContext(r.Context(), "SELECT model FROM chats WHERE id = $1", chatID).Scan(&chatModel)
	if err == sql.ErrNoRows {
		s.jsonError(w, "Chat not found", http.StatusNotFound)
		return
	}
	if err != nil {
		s.jsonError(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Get OpenRouter API key
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		s.jsonError(w, "OpenRouter API key not configured", http.StatusServiceUnavailable)
		return
	}

	// Get all messages for this chat
	rows, err := s.db.QueryContext(r.Context(), `
		SELECT role, content FROM messages WHERE chat_id = $1 ORDER BY created_at ASC
	`, chatID)
	if err != nil {
		s.jsonError(w, "Failed to get messages", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var messages []OpenRouterMessage
	for rows.Next() {
		var msg OpenRouterMessage
		if err := rows.Scan(&msg.Role, &msg.Content); err != nil {
			continue
		}
		messages = append(messages, msg)
	}

	if len(messages) == 0 {
		s.jsonError(w, "No messages in chat", http.StatusBadRequest)
		return
	}

	// Check if streaming is requested
	streaming := r.URL.Query().Get("stream") == "true"

	// Build OpenRouter request
	orReq := OpenRouterRequest{
		Model:    chatModel,
		Messages: messages,
		Stream:   streaming,
	}

	reqBody, err := json.Marshal(orReq)
	if err != nil {
		s.jsonError(w, "Failed to build request", http.StatusInternalServerError)
		return
	}

	// Make request to OpenRouter
	httpReq, err := http.NewRequestWithContext(r.Context(), "POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		s.jsonError(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("HTTP-Referer", "https://vrooli.com")
	httpReq.Header.Set("X-Title", "Agent Inbox")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		s.jsonError(w, "OpenRouter request failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		s.jsonError(w, fmt.Sprintf("OpenRouter error (%d): %s", resp.StatusCode, string(body)), resp.StatusCode)
		return
	}

	if streaming {
		s.handleStreamingResponse(w, r, resp.Body, chatID, chatModel)
	} else {
		s.handleNonStreamingResponse(w, r, resp.Body, chatID, chatModel)
	}
}

func (s *Server) handleStreamingResponse(w http.ResponseWriter, r *http.Request, body io.Reader, chatID, model string) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))

	flusher, ok := w.(http.Flusher)
	if !ok {
		s.jsonError(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	scanner := bufio.NewScanner(body)
	var fullContent strings.Builder
	var tokenCount int

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk OpenRouterResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			content := chunk.Choices[0].Delta.Content
			fullContent.WriteString(content)
			tokenCount++

			// Send SSE event
			eventData, _ := json.Marshal(map[string]string{"content": content})
			fmt.Fprintf(w, "data: %s\n\n", eventData)
			flusher.Flush()
		}
	}

	// Send completion event
	fmt.Fprintf(w, "data: {\"done\": true}\n\n")
	flusher.Flush()

	// Save the complete response as a message
	if fullContent.Len() > 0 {
		s.saveAssistantMessage(r.Context(), chatID, model, fullContent.String(), tokenCount)
	}
}

func (s *Server) handleNonStreamingResponse(w http.ResponseWriter, r *http.Request, body io.Reader, chatID, model string) {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		s.jsonError(w, "Failed to read response", http.StatusInternalServerError)
		return
	}

	var orResp OpenRouterResponse
	if err := json.Unmarshal(bodyBytes, &orResp); err != nil {
		s.jsonError(w, "Failed to parse response", http.StatusInternalServerError)
		return
	}

	if len(orResp.Choices) == 0 {
		s.jsonError(w, "No response from model", http.StatusInternalServerError)
		return
	}

	content := orResp.Choices[0].Message.Content
	tokenCount := orResp.Usage.CompletionTokens

	// Save the response as a message
	msg, err := s.saveAssistantMessage(r.Context(), chatID, model, content, tokenCount)
	if err != nil {
		s.jsonError(w, "Failed to save message", http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, msg, http.StatusOK)
}

func (s *Server) saveAssistantMessage(ctx context.Context, chatID, model, content string, tokenCount int) (*Message, error) {
	var msg Message
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO messages (chat_id, role, content, model, token_count)
		VALUES ($1, 'assistant', $2, $3, $4)
		RETURNING id, chat_id, role, content, model, token_count, created_at
	`, chatID, content, sql.NullString{String: model, Valid: model != ""}, tokenCount).Scan(
		&msg.ID, &msg.ChatID, &msg.Role, &msg.Content, &sql.NullString{}, &msg.TokenCount, &msg.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	msg.Model = model

	// Update chat preview
	preview := content
	if len(preview) > 100 {
		preview = preview[:100] + "..."
	}
	s.db.ExecContext(ctx, "UPDATE chats SET preview = $1, is_read = false, updated_at = NOW() WHERE id = $2", preview, chatID)

	return &msg, nil
}
