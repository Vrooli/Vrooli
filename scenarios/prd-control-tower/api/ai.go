package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// ReferencePRD represents a reference PRD for AI generation
type ReferencePRD struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// AIGenerateRequest represents an AI generation request
type AIGenerateRequest struct {
	Section                string         `json:"section"`                  // Section to generate (e.g., "Executive Summary", "Technical Architecture", "Full PRD")
	Context                string         `json:"context"`                  // Additional context for generation
	Action                 string         `json:"action"`                   // Quick action type (improve, expand, simplify, grammar)
	IncludeExistingContent *bool          `json:"include_existing_content"` // Whether to include existing PRD content in the prompt (default: true)
	ReferencePRDs          []ReferencePRD `json:"reference_prds,omitempty"` // Reference PRD examples
	Model                  string         `json:"model,omitempty"`          // Override model (e.g., "openrouter/x-ai/grok-code-fast-1")
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

// AIGenerateDraftRequest represents an AI request that can create/seed a draft without needing an existing draft ID.
// If save_generated_to_draft is true (default), the generated content is persisted into the draft and draft file.
type AIGenerateDraftRequest struct {
	EntityType           string         `json:"entity_type"`
	EntityName           string         `json:"entity_name"`
	Content              string         `json:"content,omitempty"`                  // Optional seed content for the draft (e.g., existing PRD text)
	Owner                string         `json:"owner,omitempty"`                    // Optional owner for metadata
	Section              string         `json:"section"`                            // Section to generate (e.g., "Executive Summary", "🎯 Full PRD")
	Context              string         `json:"context"`                            // Additional context for generation (can be requirements, notes, etc.)
	Action               string         `json:"action"`                             // Quick action type (improve, expand, simplify, grammar, technical, clarify)
	IncludeExisting      *bool          `json:"include_existing_content,omitempty"` // Default true
	ReferencePRDs        []ReferencePRD `json:"reference_prds,omitempty"`           // Reference PRD examples
	Model                string         `json:"model,omitempty"`                    // Override model (e.g., "openrouter/x-ai/grok-code-fast-1")
	SaveGeneratedToDraft *bool          `json:"save_generated_to_draft,omitempty"`  // Default true
}

type AIGenerateDraftResponse struct {
	DraftID        string `json:"draft_id"`
	EntityType     string `json:"entity_type"`
	EntityName     string `json:"entity_name"`
	Section        string `json:"section"`
	GeneratedText  string `json:"generated_text"`
	Model          string `json:"model"`
	SavedToDraft   bool   `json:"saved_to_draft"`
	Success        bool   `json:"success"`
	Message        string `json:"message,omitempty"`
	DraftFilePath  string `json:"draft_file_path,omitempty"`
	UpdatedAt      string `json:"updated_at,omitempty"`
	IncludeContent bool   `json:"include_existing_content"`
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

// handleAIGenerateDraft creates/updates a draft by entity type/name (if needed), runs AI generation, and returns the created draft_id.
// This is intended for CLI/agent flows where the caller wants a single API call to obtain a draft ID plus generated content.
func handleAIGenerateDraft(w http.ResponseWriter, r *http.Request) {
	// Defensive check for unit tests (middleware protects in production)
	if db == nil {
		respondServiceUnavailable(w, "Database not available")
		return
	}

	var req AIGenerateDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondInvalidJSON(w, err)
		return
	}

	response, status := executeAIGenerateDraft(sqlDB{db: db}, req)
	respondJSON(w, status, response)
}

func executeAIGenerateDraft(store dbQueryExecutor, req AIGenerateDraftRequest) (AIGenerateDraftResponse, int) {
	req.EntityType = strings.TrimSpace(req.EntityType)
	req.EntityName = strings.TrimSpace(req.EntityName)

	if !isValidEntityType(req.EntityType) {
		return AIGenerateDraftResponse{
			Success: false,
			Message: "Invalid entity type. Must be 'scenario' or 'resource'",
		}, http.StatusBadRequest
	}
	if req.EntityName == "" {
		return AIGenerateDraftResponse{
			Success: false,
			Message: "entity_name is required",
		}, http.StatusBadRequest
	}
	if req.Section == "" && req.Action == "" {
		return AIGenerateDraftResponse{
			Success: false,
			Message: "Section or Action is required",
		}, http.StatusBadRequest
	}

	// Defaults
	includeExisting := true
	if req.IncludeExisting != nil {
		includeExisting = *req.IncludeExisting
	}
	saveGenerated := true
	if req.SaveGeneratedToDraft != nil {
		saveGenerated = *req.SaveGeneratedToDraft
	}

	draft, err := ensureDraftForAI(store, req.EntityType, req.EntityName, req.Content, req.Owner)
	if err != nil {
		return AIGenerateDraftResponse{
			EntityType: req.EntityType,
			EntityName: req.EntityName,
			Section:    req.Section,
			Success:    false,
			Message:    fmt.Sprintf("Failed to ensure draft for AI generation: %v", err),
		}, http.StatusInternalServerError
	}

	generatedText, model, err := generateAIContent(draft, req.Section, req.Context, req.Action, includeExisting, req.ReferencePRDs, req.Model)
	if err != nil {
		return AIGenerateDraftResponse{
			DraftID:        draft.ID,
			EntityType:     draft.EntityType,
			EntityName:     draft.EntityName,
			Section:        req.Section,
			Success:        false,
			Message:        fmt.Sprintf("AI generation failed: %v", err),
			IncludeContent: includeExisting,
		}, http.StatusInternalServerError
	}

	var draftFilePath string
	var updatedAt string
	if saveGenerated {
		now := time.Now()
		ownerValue := nullString(req.Owner)
		if _, err := store.Exec(`
			UPDATE drafts
			SET content = $1, owner = COALESCE($2, owner), status = $3, updated_at = $4
			WHERE id = $5
		`, generatedText, ownerValue, DraftStatusDraft, now, draft.ID); err != nil {
			return AIGenerateDraftResponse{
				DraftID:    draft.ID,
				EntityType: draft.EntityType,
				EntityName: draft.EntityName,
				Section:    req.Section,
				Success:    false,
				Message:    fmt.Sprintf("Failed to persist AI-generated content to draft: %v", err),
			}, http.StatusInternalServerError
		}

		if err := saveDraftToFile(draft.EntityType, draft.EntityName, generatedText); err != nil {
			return AIGenerateDraftResponse{
				DraftID:    draft.ID,
				EntityType: draft.EntityType,
				EntityName: draft.EntityName,
				Section:    req.Section,
				Success:    false,
				Message:    fmt.Sprintf("Failed to save draft file: %v", err),
			}, http.StatusInternalServerError
		}

		draftFilePath = getDraftPath(draft.EntityType, draft.EntityName)
		if abs, err := filepath.Abs(draftFilePath); err == nil {
			draftFilePath = abs
		}
		updatedAt = now.Format(time.RFC3339)
	}

	return AIGenerateDraftResponse{
		DraftID:        draft.ID,
		EntityType:     draft.EntityType,
		EntityName:     draft.EntityName,
		Section:        req.Section,
		GeneratedText:  strings.TrimSpace(generatedText),
		Model:          model,
		SavedToDraft:   saveGenerated,
		Success:        true,
		DraftFilePath:  draftFilePath,
		UpdatedAt:      updatedAt,
		IncludeContent: includeExisting,
	}, http.StatusOK
}

func ensureDraftForAI(store dbQueryExecutor, entityType, entityName, content, owner string) (Draft, error) {
	var draft Draft
	var ownerValue sql.NullString
	var sourceBacklogID sql.NullString

	row := store.QueryRow(`
		SELECT id, entity_type, entity_name, content, owner, source_backlog_id, created_at, updated_at, status
		FROM drafts
		WHERE entity_type = $1 AND entity_name = $2
	`, entityType, entityName)

	scanErr := row.Scan(
		&draft.ID,
		&draft.EntityType,
		&draft.EntityName,
		&draft.Content,
		&ownerValue,
		&sourceBacklogID,
		&draft.CreatedAt,
		&draft.UpdatedAt,
		&draft.Status,
	)

	switch {
	case errors.Is(scanErr, sql.ErrNoRows):
		now := time.Now()
		draft = Draft{
			ID:         uuid.New().String(),
			EntityType: entityType,
			EntityName: entityName,
			Content:    content,
			Owner:      strings.TrimSpace(owner),
			CreatedAt:  now,
			UpdatedAt:  now,
			Status:     DraftStatusDraft,
		}

		if _, err := store.Exec(`
			INSERT INTO drafts (id, entity_type, entity_name, content, owner, source_backlog_id, created_at, updated_at, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $7, $8)
		`, draft.ID, entityType, entityName, content, nullString(owner), sql.NullString{}, now, DraftStatusDraft); err != nil {
			return Draft{}, fmt.Errorf("failed to create draft: %w", err)
		}

		return draft, nil
	case scanErr != nil:
		return Draft{}, fmt.Errorf("failed to query draft: %w", scanErr)
	default:
		if ownerValue.Valid {
			draft.Owner = ownerValue.String
		}
		if sourceBacklogID.Valid {
			draft.SourceBacklogID = &sourceBacklogID.String
		}

		// Apply seed updates if caller provided them.
		desiredContent := draft.Content
		if strings.TrimSpace(content) != "" {
			desiredContent = content
		}
		desiredOwner := draft.Owner
		if strings.TrimSpace(owner) != "" {
			desiredOwner = strings.TrimSpace(owner)
		}
		desiredStatus := DraftStatusDraft

		if desiredContent != draft.Content || desiredOwner != draft.Owner || desiredStatus != draft.Status {
			now := time.Now()
			if _, err := store.Exec(`
				UPDATE drafts
				SET content = $1, owner = $2, status = $3, updated_at = $4
				WHERE id = $5
			`, desiredContent, nullString(desiredOwner), desiredStatus, now, draft.ID); err != nil {
				return Draft{}, fmt.Errorf("failed to update draft seed: %w", err)
			}
			draft.Content = desiredContent
			draft.Owner = desiredOwner
			draft.Status = desiredStatus
			draft.UpdatedAt = now
		}

		return draft, nil
	}
}

func generateAIContent(draft Draft, section string, context string, action string, includeExisting bool, referencePRDs []ReferencePRD, modelOverride string) (string, string, error) {
	// Use OpenRouter API directly (resource-openrouter is just a thin wrapper)
	openrouterURL := os.Getenv("RESOURCE_OPENROUTER_URL")
	if openrouterURL == "" {
		// Default to OpenRouter's public API endpoint
		openrouterURL = "https://openrouter.ai/api/v1"
	}

	if isFullPRDSection(section) && action == "" {
		return generateCompliantFullPRDHTTP(openrouterURL, draft, context, includeExisting, referencePRDs, modelOverride)
	}
	return generateAIContentHTTP(openrouterURL, draft, section, context, action, includeExisting, referencePRDs, modelOverride)
}

func generateAIContentHTTP(baseURL string, draft Draft, section string, context string, action string, includeExisting bool, referencePRDs []ReferencePRD, modelOverride string) (string, string, error) {
	// Construct prompt
	prompt := buildPrompt(draft, section, context, action, includeExisting, referencePRDs)
	maxTokens := 4000
	if isFullPRDSection(section) && action == "" {
		maxTokens = 6500
	}
	return openRouterChatCompletion(baseURL, prompt, modelOverride, maxTokens)
}

func openRouterChatCompletion(baseURL string, prompt string, modelOverride string, maxTokens int) (string, string, error) {
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
		"max_tokens": maxTokens,
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

	// Set timeout for AI generation requests (3 minutes for large PRD generation)
	client := &http.Client{
		Timeout: 180 * time.Second,
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

func generateCompliantFullPRDHTTP(baseURL string, draft Draft, context string, includeExisting bool, referencePRDs []ReferencePRD, modelOverride string) (string, string, error) {
	const maxAttempts = 3
	maxTokens := 6500

	prompt := buildPrompt(draft, "🎯 Full PRD", context, "", includeExisting, referencePRDs)
	text, usedModel, err := openRouterChatCompletion(baseURL, prompt, modelOverride, maxTokens)
	if err != nil {
		return "", "", err
	}
	text = sanitizeGeneratedMarkdown(text)
	validation := ValidatePRDTemplateV2(text)
	if isPRDTemplateCompliant(validation) {
		return text, usedModel, nil
	}

	for attempt := 2; attempt <= maxAttempts; attempt++ {
		repairPrompt := buildFullPRDRepairPrompt(draft, context, includeExisting, referencePRDs, text, validation)
		next, model2, err := openRouterChatCompletion(baseURL, repairPrompt, modelOverride, maxTokens)
		if err != nil {
			return "", "", err
		}
		if strings.TrimSpace(model2) != "" {
			usedModel = model2
		}
		text = sanitizeGeneratedMarkdown(next)
		validation = ValidatePRDTemplateV2(text)
		if isPRDTemplateCompliant(validation) {
			return text, usedModel, nil
		}
	}

	return text, usedModel, fmt.Errorf("generated PRD did not match required template after %d attempts: %s", maxAttempts, summarizePRDTemplateIssues(validation))
}

func isFullPRDSection(section string) bool {
	switch strings.TrimSpace(section) {
	case "Full PRD", "🎯 Full PRD":
		return true
	default:
		return false
	}
}

func sanitizeGeneratedMarkdown(text string) string {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```") {
		lines := strings.Split(text, "\n")
		if len(lines) >= 3 && strings.HasPrefix(strings.TrimSpace(lines[0]), "```") && strings.TrimSpace(lines[len(lines)-1]) == "```" {
			text = strings.TrimSpace(strings.Join(lines[1:len(lines)-1], "\n"))
		}
	}
	return strings.TrimSpace(text)
}

func isPRDTemplateCompliant(result PRDValidationResultV2) bool {
	if len(result.Violations) > 0 {
		return false
	}
	if len(result.UnexpectedSections) > 0 {
		return false
	}
	for _, issue := range result.ContentIssues {
		if issue.Severity == "error" {
			return false
		}
	}
	return true
}

func summarizePRDTemplateIssues(result PRDValidationResultV2) string {
	parts := []string{}
	if len(result.MissingSections) > 0 {
		parts = append(parts, fmt.Sprintf("missing sections: %s", strings.Join(result.MissingSections, ", ")))
	}
	if len(result.MissingSubsections) > 0 {
		keys := make([]string, 0, len(result.MissingSubsections))
		for k := range result.MissingSubsections {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var missing []string
		for _, key := range keys {
			missing = append(missing, fmt.Sprintf("%s -> %s", key, strings.Join(result.MissingSubsections[key], ", ")))
		}
		parts = append(parts, fmt.Sprintf("missing subsections: %s", strings.Join(missing, "; ")))
	}
	if len(result.UnexpectedSections) > 0 {
		parts = append(parts, fmt.Sprintf("unexpected sections: %s", strings.Join(result.UnexpectedSections, ", ")))
	}
	var contentErrs []string
	for _, issue := range result.ContentIssues {
		if issue.Severity == "error" {
			contentErrs = append(contentErrs, fmt.Sprintf("%s: %s", issue.Section, issue.Message))
		}
	}
	if len(contentErrs) > 0 {
		parts = append(parts, fmt.Sprintf("content errors: %s", strings.Join(contentErrs, "; ")))
	}
	if len(parts) == 0 {
		return "unknown validation mismatch"
	}
	return strings.Join(parts, " | ")
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
	isFullPRD := isFullPRDSection(section)
	if isFullPRD {
		prompt.WriteString("Task: Generate a complete PRD that matches the Vrooli PRD template exactly.\n\n")
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
		prompt.WriteString("Template (MUST match headings exactly; do not add extra ##/### sections; fill in content under each heading):\n\n")
		prompt.WriteString("# Product Requirements Document (PRD)\n\n")
		prompt.WriteString("## 🎯 Overview\n")
		prompt.WriteString("Purpose: ...\n")
		prompt.WriteString("Target users: ...\n")
		prompt.WriteString("Deployment surfaces: ...\n")
		prompt.WriteString("Value proposition: ...\n\n")
		prompt.WriteString("## 🎯 Operational Targets\n\n")
		prompt.WriteString("### 🔴 P0 – Must ship for viability\n")
		prompt.WriteString("- [ ] OT-P0-001 | Title | One-line measurable outcome\n\n")
		prompt.WriteString("### 🟠 P1 – Should have post-launch\n")
		prompt.WriteString("- [ ] OT-P1-001 | Title | One-line measurable outcome\n\n")
		prompt.WriteString("### 🟢 P2 – Future / expansion\n")
		prompt.WriteString("- [ ] OT-P2-001 | Title | One-line measurable outcome\n\n")
		prompt.WriteString("## 🧱 Tech Direction Snapshot\n")
		prompt.WriteString("Preferred stacks: ...\n")
		prompt.WriteString("Preferred storage: ...\n")
		prompt.WriteString("Integration strategy: ...\n")
		prompt.WriteString("Non-goals: ...\n\n")
		prompt.WriteString("## 🤝 Dependencies & Launch Plan\n")
		prompt.WriteString("Required resources: ...\n")
		prompt.WriteString("Scenario dependencies: ...\n")
		prompt.WriteString("Operational risks: ...\n")
		prompt.WriteString("Launch sequencing: ...\n\n")
		prompt.WriteString("## 🎨 UX & Branding\n")
		prompt.WriteString("User experience: ...\n")
		prompt.WriteString("Visual design: ...\n")
		prompt.WriteString("Accessibility: ...\n\n")
		prompt.WriteString("IMPORTANT:\n")
		prompt.WriteString("- Return ONLY the PRD content (no preamble)\n")
		prompt.WriteString("- Use the exact headings shown above (including emojis)\n")
		prompt.WriteString("- Do not wrap the response in code fences\n")
	} else {
		prompt.WriteString(fmt.Sprintf("Generate only the content for the \"%s\" section. Do not include the section header itself.\n\nIMPORTANT: Return ONLY the section content. Do not include any preamble, introduction, explanations, or phrases like \"Here's the section:\" or \"Here is the content:\". Start directly with the content.", section))
	}

	return prompt.String()
}

func buildFullPRDRepairPrompt(draft Draft, context string, includeExisting bool, referencePRDs []ReferencePRD, previous string, validation PRDValidationResultV2) string {
	var prompt strings.Builder
	prompt.WriteString("You are fixing a Vrooli PRD so it matches the required template exactly.\n\n")
	prompt.WriteString(fmt.Sprintf("Entity Type: %s\n", draft.EntityType))
	prompt.WriteString(fmt.Sprintf("Entity Name: %s\n\n", draft.EntityName))

	if includeExisting && draft.Content != "" {
		prompt.WriteString("Existing PRD content (optional context):\n")
		prompt.WriteString("=" + strings.Repeat("=", 70) + "\n")
		prompt.WriteString(draft.Content)
		prompt.WriteString("\n")
		prompt.WriteString("=" + strings.Repeat("=", 70) + "\n\n")
	}

	if strings.TrimSpace(context) != "" {
		prompt.WriteString("Additional Context:\n")
		prompt.WriteString("=" + strings.Repeat("=", 70) + "\n")
		prompt.WriteString(context)
		prompt.WriteString("\n")
		prompt.WriteString("=" + strings.Repeat("=", 70) + "\n\n")
	}

	prompt.WriteString("Validation issues to fix:\n")
	prompt.WriteString(summarizePRDTemplateIssues(validation))
	prompt.WriteString("\n\n")

	prompt.WriteString("Template (MUST match headings exactly; do not add extra ##/### sections):\n\n")
	prompt.WriteString("# Product Requirements Document (PRD)\n\n")
	prompt.WriteString("## 🎯 Overview\n")
	prompt.WriteString("Purpose: ...\n")
	prompt.WriteString("Target users: ...\n")
	prompt.WriteString("Deployment surfaces: ...\n")
	prompt.WriteString("Value proposition: ...\n\n")
	prompt.WriteString("## 🎯 Operational Targets\n\n")
	prompt.WriteString("### 🔴 P0 – Must ship for viability\n")
	prompt.WriteString("- [ ] OT-P0-001 | Title | One-line measurable outcome\n\n")
	prompt.WriteString("### 🟠 P1 – Should have post-launch\n")
	prompt.WriteString("- [ ] OT-P1-001 | Title | One-line measurable outcome\n\n")
	prompt.WriteString("### 🟢 P2 – Future / expansion\n")
	prompt.WriteString("- [ ] OT-P2-001 | Title | One-line measurable outcome\n\n")
	prompt.WriteString("## 🧱 Tech Direction Snapshot\n")
	prompt.WriteString("Preferred stacks: ...\n")
	prompt.WriteString("Preferred storage: ...\n")
	prompt.WriteString("Integration strategy: ...\n")
	prompt.WriteString("Non-goals: ...\n\n")
	prompt.WriteString("## 🤝 Dependencies & Launch Plan\n")
	prompt.WriteString("Required resources: ...\n")
	prompt.WriteString("Scenario dependencies: ...\n")
	prompt.WriteString("Operational risks: ...\n")
	prompt.WriteString("Launch sequencing: ...\n\n")
	prompt.WriteString("## 🎨 UX & Branding\n")
	prompt.WriteString("User experience: ...\n")
	prompt.WriteString("Visual design: ...\n")
	prompt.WriteString("Accessibility: ...\n\n")

	prompt.WriteString("Draft that failed validation:\n")
	prompt.WriteString("=" + strings.Repeat("=", 70) + "\n")
	prompt.WriteString(previous)
	prompt.WriteString("\n")
	prompt.WriteString("=" + strings.Repeat("=", 70) + "\n\n")

	prompt.WriteString("Rewrite the PRD so it complies. Preserve good content, but ensure required headings are present and there are no unexpected sections.\n")
	prompt.WriteString("Return ONLY the PRD content. Do not wrap in code fences.\n")
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
- Return only the improved text without explanations

IMPORTANT: Return ONLY the improved text. Do not include any preamble, introduction, or explanations. Start directly with the improved content.`,
		"expand": `Expand the following text with more detail, examples, and context:

%s

Requirements:
- Add relevant details and examples
- Explain technical concepts more thoroughly
- Include practical implications
- Maintain markdown formatting
- Return only the expanded text without explanations

IMPORTANT: Return ONLY the expanded text. Do not include any preamble, introduction, or explanations. Start directly with the expanded content.`,
		"simplify": `Simplify the following text to be more concise and easier to understand:

%s

Requirements:
- Remove unnecessary jargon and complexity
- Use simpler language where possible
- Keep only essential information
- Maintain markdown formatting
- Return only the simplified text without explanations

IMPORTANT: Return ONLY the simplified text. Do not include any preamble, introduction, or explanations. Start directly with the simplified content.`,
		"grammar": `Fix grammar, spelling, and formatting issues in the following text:

%s

Requirements:
- Correct all grammatical errors
- Fix spelling mistakes
- Improve sentence structure
- Ensure consistent markdown formatting
- Return only the corrected text without explanations

IMPORTANT: Return ONLY the corrected text. Do not include any preamble, introduction, or explanations. Start directly with the corrected content.`,
		"technical": `Make the following text more technical and precise:

%s

Requirements:
- Add technical accuracy and specificity
- Include relevant technical terms
- Remove vague language
- Add measurable criteria where applicable
- Maintain markdown formatting
- Return only the enhanced text without explanations

IMPORTANT: Return ONLY the enhanced text. Do not include any preamble, introduction, or explanations. Start directly with the enhanced content.`,
		"clarify": `Clarify and restructure the following text to be more understandable:

%s

Requirements:
- Break down complex ideas into clear points
- Add structure with headings or lists if helpful
- Explain ambiguous terms
- Improve logical flow
- Maintain markdown formatting
- Return only the clarified text without explanations

IMPORTANT: Return ONLY the clarified text. Do not include any preamble, introduction, or explanations. Start directly with the clarified content.`,
	}

	template, exists := actionPrompts[action]
	if !exists {
		template = actionPrompts["improve"] // Default to improve
	}

	return fmt.Sprintf(template, selectedText)
}
