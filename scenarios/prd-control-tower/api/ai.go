package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"

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
	vars := mux.Vars(r)
	draftID := vars["id"]

	// Parse request body
	var req AIGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondInvalidJSON(w, err)
		return
	}

	if req.Section == "" {
		respondBadRequest(w, "Section is required")
		return
	}

	// Get draft from database
	draft, err := getDraftByID(draftID)
	if handleDraftError(w, err, "Failed to get draft") {
		return
	}

	// Generate AI content
	generatedText, model, err := generateAIContent(draft, req.Section, req.Context)
	if err != nil {
		response := AIGenerateResponse{
			DraftID: draftID,
			Section: req.Section,
			Success: false,
			Message: fmt.Sprintf("AI generation failed: %v", err),
		}
		respondJSON(w, http.StatusInternalServerError, response)
		return
	}

	response := AIGenerateResponse{
		DraftID:       draftID,
		Section:       req.Section,
		GeneratedText: generatedText,
		Model:         model,
		Success:       true,
	}

	respondJSON(w, http.StatusOK, response)
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
	payload := map[string]any{
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

	// Set timeout for AI generation requests (60 seconds)
	client := &http.Client{
		Timeout: 60 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to call OpenRouter API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("OpenRouter API returned error: %s, body: %s", resp.Status, string(body))
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract generated text with safe type assertions
	choices, ok := result["choices"].([]any)
	if !ok || len(choices) == 0 {
		return "", "", fmt.Errorf("no choices in response")
	}

	choice, ok := choices[0].(map[string]any)
	if !ok {
		return "", "", fmt.Errorf("invalid choice format in response")
	}

	message, ok := choice["message"].(map[string]any)
	if !ok {
		return "", "", fmt.Errorf("invalid message format in response")
	}

	content, ok := message["content"].(string)
	if !ok {
		return "", "", fmt.Errorf("invalid content format in response")
	}

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

	prompt += fmt.Sprintf(`Requirements:
- Follow Vrooli PRD standards and structure
- Be specific and actionable
- Include measurable criteria where applicable
- Use markdown formatting
- Focus on business value and technical clarity

Generate only the content for the "%s" section. Do not include the section header itself.`, section)

	return prompt
}
