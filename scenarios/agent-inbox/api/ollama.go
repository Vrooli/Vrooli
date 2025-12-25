package main

import (
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

// OllamaRequest represents a request to the Ollama API
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Options struct {
		NumPredict  int     `json:"num_predict,omitempty"`
		Temperature float64 `json:"temperature,omitempty"`
	} `json:"options,omitempty"`
}

// OllamaResponse represents a response from the Ollama API
type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// getOllamaURL returns the Ollama API base URL
func getOllamaURL() string {
	// Try environment variable first (set by lifecycle system)
	if url := os.Getenv("OLLAMA_BASE_URL"); url != "" {
		return url
	}
	// Default to standard Ollama port
	port := os.Getenv("OLLAMA_PORT")
	if port == "" {
		port = "11434"
	}
	return fmt.Sprintf("http://localhost:%s", port)
}

// getAutoNamingModel returns the model to use for auto-naming
func getAutoNamingModel() string {
	// Use a small, fast model for naming
	if model := os.Getenv("OLLAMA_NAMING_MODEL"); model != "" {
		return model
	}
	// Default to llama3.1:8b which is fast and capable
	return "llama3.1:8b"
}

// handleAutoName generates a descriptive name for a chat using Ollama
func (s *Server) handleAutoName(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID := vars["id"]

	// Get messages for this chat
	rows, err := s.db.QueryContext(r.Context(), `
		SELECT role, content FROM messages
		WHERE chat_id = $1
		ORDER BY created_at ASC
		LIMIT 10
	`, chatID)
	if err != nil {
		s.jsonError(w, "Failed to get messages", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var conversation strings.Builder
	messageCount := 0
	for rows.Next() {
		var role, content string
		if err := rows.Scan(&role, &content); err != nil {
			continue
		}
		// Truncate long messages for the naming prompt
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		conversation.WriteString(fmt.Sprintf("%s: %s\n", role, content))
		messageCount++
	}

	if messageCount == 0 {
		s.jsonError(w, "No messages in chat to generate name from", http.StatusBadRequest)
		return
	}

	// Generate name using Ollama
	name, err := s.generateChatName(r.Context(), conversation.String())
	if err != nil {
		s.log("auto-name failed, using fallback", map[string]interface{}{"error": err.Error()})
		// Fall back to a simple default
		name = "New Conversation"
	}

	// Update the chat name in the database
	var updatedChat Chat
	err = s.db.QueryRowContext(r.Context(), `
		UPDATE chats SET name = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING id, name, preview, model, view_mode, is_read, is_archived, is_starred, created_at, updated_at
	`, name, chatID).Scan(
		&updatedChat.ID, &updatedChat.Name, &updatedChat.Preview, &updatedChat.Model, &updatedChat.ViewMode,
		&updatedChat.IsRead, &updatedChat.IsArchived, &updatedChat.IsStarred, &updatedChat.CreatedAt, &updatedChat.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		s.jsonError(w, "Chat not found", http.StatusNotFound)
		return
	}
	if err != nil {
		s.jsonError(w, "Failed to update chat name", http.StatusInternalServerError)
		return
	}

	// Get label IDs for the response
	labelRows, err := s.db.QueryContext(r.Context(), "SELECT label_id FROM chat_labels WHERE chat_id = $1", chatID)
	if err == nil {
		defer labelRows.Close()
		for labelRows.Next() {
			var labelID string
			if labelRows.Scan(&labelID) == nil {
				updatedChat.LabelIDs = append(updatedChat.LabelIDs, labelID)
			}
		}
	}
	if updatedChat.LabelIDs == nil {
		updatedChat.LabelIDs = []string{}
	}

	s.jsonResponse(w, updatedChat, http.StatusOK)
}

// generateChatName uses Ollama to generate a concise, descriptive chat name
func (s *Server) generateChatName(ctx context.Context, conversationSummary string) (string, error) {
	ollamaURL := getOllamaURL()
	model := getAutoNamingModel()

	prompt := fmt.Sprintf(`Generate a very short, descriptive title (3-6 words max) for this conversation.
Return ONLY the title, no quotes, no explanation, no punctuation at the end.

Examples of good titles:
- Code Review Discussion
- Bug Fix for Login
- API Design Questions
- Database Migration Help
- React Component Tutorial

Conversation:
%s

Title:`, conversationSummary)

	reqBody := OllamaRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
	}
	reqBody.Options.NumPredict = 20
	reqBody.Options.Temperature = 0.3

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", ollamaURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama returned %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var ollamaResp OllamaResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Clean up the response
	name := strings.TrimSpace(ollamaResp.Response)
	// Remove quotes if present
	name = strings.Trim(name, `"'`)
	// Remove trailing punctuation
	name = strings.TrimRight(name, ".!?,;:")
	// Limit length
	if len(name) > 50 {
		name = name[:50]
	}
	// Ensure we have a valid name
	if name == "" {
		name = "New Conversation"
	}

	return name, nil
}
