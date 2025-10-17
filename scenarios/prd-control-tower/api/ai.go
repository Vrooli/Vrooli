package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"

	"github.com/gorilla/mux"
)

// AIGenerateRequest represents an AI generation request
type AIGenerateRequest struct {
	Section string `json:"section"` // Section to generate (e.g., "Executive Summary", "Technical Architecture")
	Context string `json:"context"` // Additional context for generation
}

// AIGenerateResponse represents the AI generation result
type AIGenerateResponse struct {
	DraftID       string `json:"draft_id"`
	Section       string `json:"section"`
	GeneratedText string `json:"generated_text"`
	Model         string `json:"model"`
	Success       bool   `json:"success"`
	Message       string `json:"message,omitempty"`
}

func handleAIGenerateSection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if db == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	draftID := vars["id"]

	// Parse request body
	var req AIGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.Section == "" {
		http.Error(w, "Section is required", http.StatusBadRequest)
		return
	}

	// Get draft from database
	var draft Draft
	var owner sql.NullString
	err := db.QueryRow(`
		SELECT id, entity_type, entity_name, content, owner, created_at, updated_at, status
		FROM drafts
		WHERE id = $1
	`, draftID).Scan(
		&draft.ID,
		&draft.EntityType,
		&draft.EntityName,
		&draft.Content,
		&owner,
		&draft.CreatedAt,
		&draft.UpdatedAt,
		&draft.Status,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Draft not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get draft: %v", err), http.StatusInternalServerError)
		return
	}

	// Generate AI content
	generatedText, model, err := generateAIContent(draft, req.Section, req.Context)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		response := AIGenerateResponse{
			DraftID: draftID,
			Section: req.Section,
			Success: false,
			Message: fmt.Sprintf("AI generation failed: %v", err),
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	response := AIGenerateResponse{
		DraftID:       draftID,
		Section:       req.Section,
		GeneratedText: generatedText,
		Model:         model,
		Success:       true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func generateAIContent(draft Draft, section string, context string) (string, string, error) {
	// Check if resource-openrouter HTTP API is configured
	openrouterURL := os.Getenv("RESOURCE_OPENROUTER_URL")
	if openrouterURL != "" {
		// HTTP API is configured - use it
		return generateAIContentHTTP(openrouterURL, draft, section, context)
	}

	// Fallback to CLI if HTTP API not configured
	return generateAIContentCLI(draft, section, context)
}

func generateAIContentHTTP(baseURL string, draft Draft, section string, context string) (string, string, error) {
	// Construct prompt
	prompt := buildPrompt(draft, section, context)

	// Call OpenRouter API
	payload := map[string]interface{}{
		"model": "anthropic/claude-3.5-sonnet",
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_tokens": 2000,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", baseURL+"/api/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to call OpenRouter API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("OpenRouter API returned error: %s, body: %s", resp.Status, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract generated text
	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", "", fmt.Errorf("no choices in response")
	}

	choice := choices[0].(map[string]interface{})
	message := choice["message"].(map[string]interface{})
	content := message["content"].(string)

	model := "anthropic/claude-3.5-sonnet"
	if m, ok := result["model"].(string); ok {
		model = m
	}

	return content, model, nil
}

func generateAIContentCLI(draft Draft, section string, context string) (string, string, error) {
	// Check if resource-openrouter CLI is available
	_, err := exec.LookPath("resource-openrouter")
	if err != nil {
		return "", "", fmt.Errorf("resource-openrouter not available - install it to enable AI assistance")
	}

	// Construct prompt
	prompt := buildPrompt(draft, section, context)

	// Run resource-openrouter
	cmd := exec.Command("resource-openrouter", "chat", "--model", "anthropic/claude-3.5-sonnet", "--message", prompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return "", "", fmt.Errorf("resource-openrouter failed: %v, stderr: %s", err, stderr.String())
	}

	return stdout.String(), "anthropic/claude-3.5-sonnet", nil
}

func buildPrompt(draft Draft, section string, context string) string {
	prompt := fmt.Sprintf(`You are an expert technical writer helping to create a Product Requirements Document (PRD).

Entity Type: %s
Entity Name: %s

Current PRD Content:
%s

Task: Generate the "%s" section for this PRD.

`, draft.EntityType, draft.EntityName, draft.Content, section)

	if context != "" {
		prompt += fmt.Sprintf(`Additional Context:
%s

`, context)
	}

	prompt += `Requirements:
- Follow Vrooli PRD standards and structure
- Be specific and actionable
- Include measurable criteria where applicable
- Use markdown formatting
- Focus on business value and technical clarity

Generate only the content for the "%s" section. Do not include the section header itself.`

	return fmt.Sprintf(prompt, section)
}
