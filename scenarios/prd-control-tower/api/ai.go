package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// ReferencePRD represents a reference PRD for AI generation
type ReferencePRD struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// AIGenerateRequest represents an AI generation request
type AIGenerateRequest struct {
	Section                string         `json:"section"`                   // Section to generate (e.g., "Executive Summary", "Technical Architecture", "Full PRD")
	Context                string         `json:"context"`                   // Additional context for generation
	Action                 string         `json:"action"`                    // Quick action type (improve, expand, simplify, grammar)
	IncludeExistingContent *bool          `json:"include_existing_content"`  // Whether to include existing PRD content in the prompt (default: true)
	ReferencePRDs          []ReferencePRD `json:"reference_prds,omitempty"`  // Reference PRD examples
	Model                  string         `json:"model,omitempty"`           // Override model (e.g., "openrouter/x-ai/grok-code-fast-1")
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

	if req.Section == "" && req.Action == "" {
		respondBadRequest(w, "Section or Action is required")
		return
	}

	// Get draft from database
	draft, err := getDraftByID(draftID)
	if handleDraftError(w, err, "Failed to get draft") {
		return
	}

	// Default include_existing_content to true if not specified
	includeExisting := true
	if req.IncludeExistingContent != nil {
		includeExisting = *req.IncludeExistingContent
	}

	// Generate AI content
	generatedText, model, err := generateAIContent(draft, req.Section, req.Context, req.Action, includeExisting, req.ReferencePRDs, req.Model)
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

func generateAIContent(draft Draft, section string, context string, action string, includeExisting bool, referencePRDs []ReferencePRD, modelOverride string) (string, string, error) {
	// Use OpenRouter API directly (resource-openrouter is just a thin wrapper)
	openrouterURL := os.Getenv("RESOURCE_OPENROUTER_URL")
	if openrouterURL == "" {
		// Default to OpenRouter's public API endpoint
		openrouterURL = "https://openrouter.ai/api/v1"
	}

	return generateAIContentHTTP(openrouterURL, draft, section, context, action, includeExisting, referencePRDs, modelOverride)
}

func generateAIContentHTTP(baseURL string, draft Draft, section string, context string, action string, includeExisting bool, referencePRDs []ReferencePRD, modelOverride string) (string, string, error) {
	// Construct prompt
	prompt := buildPrompt(draft, section, context, action, includeExisting, referencePRDs)

	// Determine which model to use
	model := "anthropic/claude-3.5-sonnet"
	if modelOverride != "" && modelOverride != "default" {
		// Remove "openrouter/" prefix if present
		if strings.HasPrefix(modelOverride, "openrouter/") {
			model = strings.TrimPrefix(modelOverride, "openrouter/")
		} else {
			model = modelOverride
		}
	}

	// Call OpenRouter API
	payload := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_tokens": 4000,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Construct API endpoint URL
	apiURL := baseURL
	if !strings.HasSuffix(apiURL, "/chat/completions") {
		if strings.HasSuffix(apiURL, "/api/v1") {
			apiURL += "/chat/completions"
		} else {
			apiURL += "/api/v1/chat/completions"
		}
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	// Get API key from environment
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return "", "", fmt.Errorf("OPENROUTER_API_KEY environment variable not set")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

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

	usedModel := model
	if m, ok := result["model"].(string); ok {
		usedModel = m
	}

	return content, usedModel, nil
}

func generateAIContentCLI(draft Draft, section string, context string, action string, includeExisting bool, referencePRDs []ReferencePRD) (string, string, error) {
	// Check if resource-openrouter CLI is available
	_, err := exec.LookPath("resource-openrouter")
	if err != nil {
		return "", "", fmt.Errorf("resource-openrouter not available - install it to enable AI assistance")
	}

	// Construct prompt
	prompt := buildPrompt(draft, section, context, action, includeExisting, referencePRDs)

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

func buildPrompt(draft Draft, section string, context string, action string, includeExisting bool, referencePRDs []ReferencePRD) string {
	// If action is specified, use action-based prompt
	if action != "" {
		return buildActionPrompt(action, context)
	}

	// Otherwise, use section-based prompt
	var prompt strings.Builder

	prompt.WriteString("You are an expert technical writer helping to create a Product Requirements Document (PRD).\n\n")
	prompt.WriteString(fmt.Sprintf("Entity Type: %s\n", draft.EntityType))
	prompt.WriteString(fmt.Sprintf("Entity Name: %s\n\n", draft.EntityName))

	// Include reference PRDs if provided
	if len(referencePRDs) > 0 {
		prompt.WriteString("Reference PRD Examples (for style and structure guidance):\n")
		prompt.WriteString("=" + strings.Repeat("=", 70) + "\n\n")
		for i, ref := range referencePRDs {
			prompt.WriteString(fmt.Sprintf("Reference PRD %d: %s\n", i+1, ref.Name))
			prompt.WriteString(strings.Repeat("-", 70) + "\n")
			prompt.WriteString(ref.Content)
			prompt.WriteString("\n")
			prompt.WriteString(strings.Repeat("-", 70) + "\n\n")
		}
		prompt.WriteString("=" + strings.Repeat("=", 70) + "\n\n")
	}

	// Include existing PRD content if requested
	if includeExisting && draft.Content != "" {
		prompt.WriteString("Current PRD Content:\n")
		prompt.WriteString("=" + strings.Repeat("=", 70) + "\n")
		prompt.WriteString(draft.Content)
		prompt.WriteString("\n")
		prompt.WriteString("=" + strings.Repeat("=", 70) + "\n\n")
	}

	// Determine task description based on section
	isFullPRD := section == "Full PRD" || section == "🎯 Full PRD"
	if isFullPRD {
		prompt.WriteString("Task: Generate a complete, comprehensive PRD for this entity.\n\n")
	} else {
		prompt.WriteString(fmt.Sprintf("Task: Generate the \"%s\" section for this PRD.\n\n", section))
	}

	// Include additional context if provided
	if context != "" {
		prompt.WriteString("Additional Context:\n")
		prompt.WriteString("=" + strings.Repeat("=", 70) + "\n")
		prompt.WriteString(context)
		prompt.WriteString("\n")
		prompt.WriteString("=" + strings.Repeat("=", 70) + "\n\n")
	}

	// Add requirements
	prompt.WriteString("Requirements:\n")
	prompt.WriteString("- Follow Vrooli PRD standards and structure\n")
	prompt.WriteString("- Be specific and actionable\n")
	prompt.WriteString("- Include measurable criteria where applicable\n")
	prompt.WriteString("- Use markdown formatting\n")
	prompt.WriteString("- Focus on business value and technical clarity\n")
	if len(referencePRDs) > 0 {
		prompt.WriteString("- Use the reference PRD examples above as inspiration for style, structure, and level of detail\n")
	}
	prompt.WriteString("\n")

	if isFullPRD {
		prompt.WriteString("Generate a complete PRD including all standard sections (Overview, Operational Targets, Tech Direction, Dependencies, UX & Branding, etc.). Do not include markdown headers for the document title.")
	} else {
		prompt.WriteString(fmt.Sprintf("Generate only the content for the \"%s\" section. Do not include the section header itself.", section))
	}

	return prompt.String()
}

func buildActionPrompt(action string, selectedText string) string {
	actionPrompts := map[string]string{
		"improve": `Improve the following text to be more professional, clear, and actionable:

%s

Requirements:
- Maintain the original meaning and key points
- Use stronger, more precise language
- Add clarity and remove ambiguity
- Keep markdown formatting
- Return only the improved text without explanations`,
		"expand": `Expand the following text with more detail, examples, and context:

%s

Requirements:
- Add relevant details and examples
- Explain technical concepts more thoroughly
- Include practical implications
- Maintain markdown formatting
- Return only the expanded text without explanations`,
		"simplify": `Simplify the following text to be more concise and easier to understand:

%s

Requirements:
- Remove unnecessary jargon and complexity
- Use simpler language where possible
- Keep only essential information
- Maintain markdown formatting
- Return only the simplified text without explanations`,
		"grammar": `Fix grammar, spelling, and formatting issues in the following text:

%s

Requirements:
- Correct all grammatical errors
- Fix spelling mistakes
- Improve sentence structure
- Ensure consistent markdown formatting
- Return only the corrected text without explanations`,
		"technical": `Make the following text more technical and precise:

%s

Requirements:
- Add technical accuracy and specificity
- Include relevant technical terms
- Remove vague language
- Add measurable criteria where applicable
- Maintain markdown formatting
- Return only the enhanced text without explanations`,
		"clarify": `Clarify and restructure the following text to be more understandable:

%s

Requirements:
- Break down complex ideas into clear points
- Add structure with headings or lists if helpful
- Explain ambiguous terms
- Improve logical flow
- Maintain markdown formatting
- Return only the clarified text without explanations`,
	}

	template, exists := actionPrompts[action]
	if !exists {
		template = actionPrompts["improve"] // Default to improve
	}

	return fmt.Sprintf(template, selectedText)
}
